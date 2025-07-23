package dispatcher

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/1cbyc/go-task-processing-system/config"
	"github.com/1cbyc/go-task-processing-system/job"
	"github.com/1cbyc/go-task-processing-system/worker"
)

var ErrQueueFull = errors.New("job queue is full")

type Dispatcher struct {
	ID          string
	Workers     map[string]*worker.Worker
	JobQueue    chan *job.Job
	ResultQueue chan *job.Job
	Config      *config.WorkerConfig
	Logger      *logrus.Logger
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	mu          sync.RWMutex
	stats       *DispatcherStats
	health      *DispatcherHealth
	workerPool  *WorkerPool
}

type DispatcherStats struct {
	JobsSubmitted   int64
	JobsCompleted   int64
	JobsFailed      int64
	WorkersActive   int64
	QueueSize       int64
	AverageWaitTime time.Duration
	StartTime       time.Time
	mu              sync.RWMutex
}

type DispatcherHealth struct {
	IsHealthy    bool
	LastCheck    time.Time
	ErrorCount   int64
	SuccessCount int64
	mu           sync.RWMutex
}

type WorkerPool struct {
	workers []*worker.Worker
	current int
	mu      sync.Mutex
}

func NewWorkerPool(workers []*worker.Worker) *WorkerPool {
	return &WorkerPool{
		workers: workers,
		current: 0,
	}
}

func (wp *WorkerPool) GetNextWorker() *worker.Worker {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if len(wp.workers) == 0 {
		return nil
	}

	worker := wp.workers[wp.current]
	wp.current = (wp.current + 1) % len(wp.workers)
	return worker
}

func (wp *WorkerPool) GetWorkers() []*worker.Worker {
	wp.mu.Lock()
	defer wp.mu.Unlock()
	return append([]*worker.Worker{}, wp.workers...)
}

func New(cfg *config.WorkerConfig, logger *logrus.Logger) *Dispatcher {
	ctx, cancel := context.WithCancel(context.Background())

	d := &Dispatcher{
		ID:          uuid.New().String(),
		Workers:     make(map[string]*worker.Worker),
		JobQueue:    make(chan *job.Job, cfg.MaxConcurrency),
		ResultQueue: make(chan *job.Job, cfg.MaxConcurrency),
		Config:      cfg,
		Logger:      logger,
		ctx:         ctx,
		cancel:      cancel,
		stats:       &DispatcherStats{StartTime: time.Now()},
		health:      &DispatcherHealth{IsHealthy: true, LastCheck: time.Now()},
	}

	d.workerPool = NewWorkerPool([]*worker.Worker{})

	return d
}

func (d *Dispatcher) Start() {
	d.Logger.WithFields(logrus.Fields{
		"dispatcher_id": d.ID,
		"num_workers":   d.Config.NumWorkers,
	}).Info("Starting dispatcher")

	d.startWorkers()
	d.wg.Add(2)
	go d.dispatchJobs()
	go d.processResults()
}

func (d *Dispatcher) Stop() {
	d.Logger.WithFields(logrus.Fields{
		"dispatcher_id": d.ID,
	}).Info("Stopping dispatcher")

	d.cancel()

	d.mu.Lock()
	for _, w := range d.Workers {
		w.Stop()
	}
	d.mu.Unlock()

	d.wg.Wait()
	close(d.JobQueue)
	close(d.ResultQueue)

	d.Logger.WithFields(logrus.Fields{
		"dispatcher_id": d.ID,
	}).Info("Dispatcher stopped")
}

func (d *Dispatcher) SubmitJob(j *job.Job) error {
	select {
	case d.JobQueue <- j:
		d.updateStats(1, 0, 0)
		d.Logger.WithFields(logrus.Fields{
			"dispatcher_id": d.ID,
			"job_id":        j.ID,
			"job_type":      j.Type,
			"priority":      j.Priority,
		}).Debug("Job submitted to queue")
		return nil
	case <-d.ctx.Done():
		return d.ctx.Err()
	default:
		return ErrQueueFull
	}
}

func (d *Dispatcher) startWorkers() {
	d.mu.Lock()
	defer d.mu.Unlock()

	for i := 0; i < d.Config.NumWorkers; i++ {
		workerID := uuid.New().String()
		w := worker.New(workerID, d.JobQueue, d.ResultQueue, d.Config, d.Logger)
		d.Workers[workerID] = w
		w.Start()
	}

	workers := make([]*worker.Worker, 0, len(d.Workers))
	for _, w := range d.Workers {
		workers = append(workers, w)
	}
	d.workerPool = NewWorkerPool(workers)
}

func (d *Dispatcher) dispatchJobs() {
	defer d.wg.Done()

	for {
		select {
		case j, ok := <-d.JobQueue:
			if !ok {
				return
			}
			d.processJob(j)
		case <-d.ctx.Done():
			return
		}
	}
}

func (d *Dispatcher) processJob(j *job.Job) {
	worker := d.workerPool.GetNextWorker()
	if worker == nil {
		d.Logger.WithFields(logrus.Fields{
			"dispatcher_id": d.ID,
			"job_id":        j.ID,
		}).Error("No available workers")
		return
	}

	select {
	case d.JobQueue <- j:
		d.Logger.WithFields(logrus.Fields{
			"dispatcher_id": d.ID,
			"job_id":        j.ID,
			"worker_id":     worker.ID,
		}).Debug("Job dispatched to worker")
	default:
		d.Logger.WithFields(logrus.Fields{
			"dispatcher_id": d.ID,
			"job_id":        j.ID,
		}).Warn("Worker queue full, job will be retried")
	}
}

func (d *Dispatcher) processResults() {
	defer d.wg.Done()

	for {
		select {
		case result, ok := <-d.ResultQueue:
			if !ok {
				return
			}
			d.handleResult(result)
		case <-d.ctx.Done():
			return
		}
	}
}

func (d *Dispatcher) handleResult(result *job.Job) {
	switch result.Status {
	case job.StatusCompleted:
		d.updateStats(0, 1, 0)
		d.Logger.WithFields(logrus.Fields{
			"dispatcher_id": d.ID,
			"job_id":        result.ID,
			"duration":      result.Duration(),
		}).Info("Job completed successfully")
	case job.StatusFailed:
		d.updateStats(0, 0, 1)
		d.Logger.WithFields(logrus.Fields{
			"dispatcher_id": d.ID,
			"job_id":        result.ID,
			"error":         result.Error,
		}).Error("Job failed permanently")
	}
}

func (d *Dispatcher) updateStats(submitted, completed, failed int64) {
	d.stats.mu.Lock()
	defer d.stats.mu.Unlock()

	d.stats.JobsSubmitted += submitted
	d.stats.JobsCompleted += completed
	d.stats.JobsFailed += failed
	d.stats.WorkersActive = int64(len(d.Workers))
	d.stats.QueueSize = int64(len(d.JobQueue))
}

func (d *Dispatcher) GetStats() DispatcherStats {
	d.stats.mu.RLock()
	defer d.stats.mu.RUnlock()
	return *d.stats
}

func (d *Dispatcher) GetHealth() DispatcherHealth {
	d.health.mu.RLock()
	defer d.health.mu.RUnlock()
	return *d.health
}

func (d *Dispatcher) GetWorkers() map[string]*worker.Worker {
	d.mu.RLock()
	defer d.mu.RUnlock()

	workers := make(map[string]*worker.Worker)
	for id, w := range d.Workers {
		workers[id] = w
	}
	return workers
}

func (d *Dispatcher) IsHealthy() bool {
	d.health.mu.RLock()
	defer d.health.mu.RUnlock()
	return d.health.IsHealthy
}

func (d *Dispatcher) Wait() {
	d.wg.Wait()
}
