package worker

import (
	"context"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/1cbyc/go-task-processing-system/config"
	"github.com/1cbyc/go-task-processing-system/job"
)

type Worker struct {
	ID        string
	Status    WorkerStatus
	Jobs      <-chan *job.Job
	Results   chan<- *job.Job
	Config    *config.WorkerConfig
	Logger    *logrus.Logger
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	mu        sync.RWMutex
	stats     *WorkerStats
	health    *WorkerHealth
	processor JobProcessor
}

type WorkerStatus string

const (
	WorkerStatusIdle     WorkerStatus = "idle"
	WorkerStatusRunning  WorkerStatus = "running"
	WorkerStatusStopping WorkerStatus = "stopping"
	WorkerStatusStopped  WorkerStatus = "stopped"
)

type WorkerStats struct {
	JobsProcessed         int64
	JobsSucceeded         int64
	JobsFailed            int64
	TotalProcessingTime   time.Duration
	AverageProcessingTime time.Duration
	LastJobTime           time.Time
	StartTime             time.Time
	mu                    sync.RWMutex
}

type WorkerHealth struct {
	IsHealthy    bool
	LastCheck    time.Time
	ErrorCount   int64
	SuccessCount int64
	mu           sync.RWMutex
}

type JobProcessor interface {
	Process(ctx context.Context, j *job.Job) (*job.Result, error)
}

type DefaultProcessor struct{}

func (p *DefaultProcessor) Process(ctx context.Context, j *job.Job) (*job.Result, error) {
	time.Sleep(time.Duration(100+time.Now().UnixNano()%1000) * time.Millisecond)

	result := &job.Result{
		Data: map[string]interface{}{
			"processed_at": time.Now(),
			"job_type":     j.Type,
			"worker_id":    j.WorkerID,
		},
		Metrics: map[string]interface{}{
			"processing_time_ms": j.Duration().Milliseconds(),
		},
	}

	return result, nil
}

func New(id string, jobs <-chan *job.Job, results chan<- *job.Job, cfg *config.WorkerConfig, logger *logrus.Logger) *Worker {
	ctx, cancel := context.WithCancel(context.Background())

	return &Worker{
		ID:        id,
		Status:    WorkerStatusIdle,
		Jobs:      jobs,
		Results:   results,
		Config:    cfg,
		Logger:    logger,
		ctx:       ctx,
		cancel:    cancel,
		stats:     &WorkerStats{StartTime: time.Now()},
		health:    &WorkerHealth{IsHealthy: true, LastCheck: time.Now()},
		processor: &DefaultProcessor{},
	}
}

func (w *Worker) Start() {
	w.mu.Lock()
	if w.Status != WorkerStatusIdle {
		w.mu.Unlock()
		return
	}
	w.Status = WorkerStatusRunning
	w.mu.Unlock()

	w.wg.Add(1)
	go w.run()

	w.Logger.WithFields(logrus.Fields{
		"worker_id": w.ID,
		"status":    w.Status,
	}).Info("Worker started")
}

func (w *Worker) Stop() {
	w.mu.Lock()
	if w.Status == WorkerStatusStopped {
		w.mu.Unlock()
		return
	}
	w.Status = WorkerStatusStopping
	w.mu.Unlock()

	w.cancel()
	w.wg.Wait()

	w.mu.Lock()
	w.Status = WorkerStatusStopped
	w.mu.Unlock()

	w.Logger.WithFields(logrus.Fields{
		"worker_id": w.ID,
		"status":    w.Status,
	}).Info("Worker stopped")
}

func (w *Worker) run() {
	defer w.wg.Done()

	for {
		select {
		case j, ok := <-w.Jobs:
			if !ok {
				return
			}
			w.processJob(j)
		case <-w.ctx.Done():
			return
		}
	}
}

func (w *Worker) processJob(j *job.Job) {
	startTime := time.Now()

	if err := j.Start(w.ID); err != nil {
		w.Logger.WithFields(logrus.Fields{
			"worker_id": w.ID,
			"job_id":    j.ID,
			"error":     err.Error(),
		}).Error("Failed to start job")
		return
	}

	w.Logger.WithFields(logrus.Fields{
		"worker_id": w.ID,
		"job_id":    j.ID,
		"job_type":  j.Type,
	}).Debug("Processing job")

	ctx, cancel := context.WithTimeout(w.ctx, w.Config.JobTimeout)
	defer cancel()

	result, err := w.processor.Process(ctx, j)

	if err != nil {
		w.handleJobFailure(j, err)
	} else {
		w.handleJobSuccess(j, result)
	}

	w.updateStats(startTime, err == nil)
}

func (w *Worker) handleJobSuccess(j *job.Job, result *job.Result) {
	if err := j.Complete(result); err != nil {
		w.Logger.WithFields(logrus.Fields{
			"worker_id": w.ID,
			"job_id":    j.ID,
			"error":     err.Error(),
		}).Error("Failed to complete job")
		return
	}

	w.Results <- j

	w.Logger.WithFields(logrus.Fields{
		"worker_id": w.ID,
		"job_id":    j.ID,
		"duration":  j.Duration(),
	}).Info("Job completed successfully")
}

func (w *Worker) handleJobFailure(j *job.Job, err error) {
	if processErr := j.Fail(err); processErr != nil {
		w.Logger.WithFields(logrus.Fields{
			"worker_id": w.ID,
			"job_id":    j.ID,
			"error":     processErr.Error(),
		}).Error("Failed to mark job as failed")
		return
	}

	if j.CanRetry() {
		w.Results <- j
		w.Logger.WithFields(logrus.Fields{
			"worker_id":    w.ID,
			"job_id":       j.ID,
			"attempts":     j.Attempts,
			"max_attempts": j.MaxAttempts,
		}).Warn("Job failed, will retry")
	} else {
		w.Logger.WithFields(logrus.Fields{
			"worker_id": w.ID,
			"job_id":    j.ID,
			"attempts":  j.Attempts,
			"error":     err.Error(),
		}).Error("Job failed permanently")
	}
}

func (w *Worker) updateStats(startTime time.Time, success bool) {
	w.stats.mu.Lock()
	defer w.stats.mu.Unlock()

	w.stats.JobsProcessed++
	processingTime := time.Since(startTime)
	w.stats.TotalProcessingTime += processingTime
	w.stats.AverageProcessingTime = w.stats.TotalProcessingTime / time.Duration(w.stats.JobsProcessed)
	w.stats.LastJobTime = time.Now()

	if success {
		w.stats.JobsSucceeded++
	} else {
		w.stats.JobsFailed++
	}
}

func (w *Worker) GetStatus() WorkerStatus {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.Status
}

func (w *Worker) GetStats() WorkerStats {
	w.stats.mu.RLock()
	defer w.stats.mu.RUnlock()
	return *w.stats
}

func (w *Worker) GetHealth() WorkerHealth {
	w.health.mu.RLock()
	defer w.health.mu.RUnlock()
	return *w.health
}

func (w *Worker) SetProcessor(processor JobProcessor) {
	w.processor = processor
}

func (w *Worker) IsHealthy() bool {
	w.health.mu.RLock()
	defer w.health.mu.RUnlock()
	return w.health.IsHealthy
}
