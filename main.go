package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/1cbyc/go-task-processing-system/config"
	"github.com/1cbyc/go-task-processing-system/dispatcher"
	"github.com/1cbyc/go-task-processing-system/job"
)

var logger *logrus.Logger

func main() {
	setupLogger()

	cfg, err := config.Load()
	if err != nil {
		logger.Fatalf("Failed to load configuration: %v", err)
	}

	dispatcher := dispatcher.New(&cfg.Worker, logger)
	dispatcher.Start()

	server := setupHTTPServer(cfg, dispatcher)

	go func() {
		logger.WithFields(logrus.Fields{
			"host": cfg.Server.Host,
			"port": cfg.Server.Port,
		}).Info("Starting HTTP server")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("HTTP server error: %v", err)
		}
	}()

	go submitSampleJobs(dispatcher)

	waitForShutdown(server, dispatcher)
}

func setupLogger() {
	logger = logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetLevel(logrus.InfoLevel)
}

func setupHTTPServer(cfg *config.Config, d *dispatcher.Dispatcher) *http.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", healthHandler(d))
	mux.HandleFunc("/stats", statsHandler(d))
	mux.HandleFunc("/jobs", jobsHandler(d))
	mux.HandleFunc("/workers", workersHandler(d))

	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      mux,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	return server
}

func healthHandler(d *dispatcher.Dispatcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		health := d.GetHealth()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"healthy","timestamp":"%s","dispatcher_healthy":%t}`,
			time.Now().Format(time.RFC3339), health.IsHealthy)
	}
}

func statsHandler(d *dispatcher.Dispatcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stats := d.GetStats()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"jobs_submitted":%d,"jobs_completed":%d,"jobs_failed":%d,"workers_active":%d,"queue_size":%d,"uptime":"%s"}`,
			stats.JobsSubmitted, stats.JobsCompleted, stats.JobsFailed, stats.WorkersActive, stats.QueueSize, time.Since(stats.StartTime).String())
	}
}

func jobsHandler(d *dispatcher.Dispatcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			submitJobHandler(d)(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprintf(w, `{"error":"method not allowed"}`)
	}
}

func submitJobHandler(d *dispatcher.Dispatcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jobType := r.URL.Query().Get("type")
		if jobType == "" {
			jobType = "default"
		}

		priority := job.PriorityNormal
		if r.URL.Query().Get("priority") == "high" {
			priority = job.PriorityHigh
		}

		j := job.New(jobType, map[string]interface{}{
			"source":    "http",
			"timestamp": time.Now(),
		}, priority)

		if err := d.SubmitJob(j); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"error":"%s"}`, err.Error())
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		fmt.Fprintf(w, `{"job_id":"%s","status":"submitted"}`, j.ID)
	}
}

func workersHandler(d *dispatcher.Dispatcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		workers := d.GetWorkers()
		workerStats := make(map[string]interface{})

		for id, worker := range workers {
			stats := worker.GetStats()
			workerStats[id] = map[string]interface{}{
				"status":         worker.GetStatus(),
				"jobs_processed": stats.JobsProcessed,
				"jobs_succeeded": stats.JobsSucceeded,
				"jobs_failed":    stats.JobsFailed,
				"uptime":         time.Since(stats.StartTime).String(),
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"workers":%d}`, len(workers))
	}
}

func submitSampleJobs(d *dispatcher.Dispatcher) {
	time.Sleep(2 * time.Second)

	jobTypes := []string{"email", "report", "backup", "sync", "cleanup"}
	priorities := []job.Priority{job.PriorityLow, job.PriorityNormal, job.PriorityHigh, job.PriorityUrgent}

	for i := 0; i < 50; i++ {
		jobType := jobTypes[i%len(jobTypes)]
		priority := priorities[i%len(priorities)]

		j := job.New(jobType, map[string]interface{}{
			"batch_id":  i / 10,
			"sequence":  i,
			"timestamp": time.Now(),
		}, priority)

		if err := d.SubmitJob(j); err != nil {
			logger.WithError(err).Error("Failed to submit sample job")
		}

		time.Sleep(100 * time.Millisecond)
	}

	logger.Info("Sample jobs submitted")
}

func waitForShutdown(server *http.Server, d *dispatcher.Dispatcher) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	logger.Info("Shutdown signal received")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Errorf("Server shutdown error: %v", err)
	}

	d.Stop()
	logger.Info("Application shutdown complete")
}
