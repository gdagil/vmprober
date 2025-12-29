package server

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"time"

	"github.com/VictoriaMetrics/metrics"
	"github.com/sirupsen/logrus"

	"github.com/gdagil/vmprober/internal/scheduler"
)

//go:embed static
var staticFiles embed.FS

// Server HTTP server for metrics and health checks
type Server struct {
	httpServer *http.Server
	logger     *logrus.Logger
	startTime  time.Time
	scheduler  *scheduler.Scheduler
}

// NewServer creates a new HTTP server
func NewServer(host string, port int, logger *logrus.Logger) *Server {
	mux := http.NewServeMux()
	server := &Server{
		logger:    logger,
		startTime: time.Now(),
		httpServer: &http.Server{
			Addr:         fmt.Sprintf("%s:%d", host, port),
			Handler:      mux,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
		},
	}

	// Create sub-filesystem for static files
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		logger.WithError(err).Fatal("Failed to create static sub-filesystem")
	}

	// Register handlers
	mux.HandleFunc("/health", server.healthHandler)
	mux.HandleFunc("/ready", server.readyHandler)
	mux.HandleFunc("/metrics", server.metricsHandler)
	mux.HandleFunc("/api/v1/jobs", server.jobsHandler)
	mux.HandleFunc("/api/v1/stats", server.statsHandler)
	mux.HandleFunc("/api/v1/targets", server.targetsHandler)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))
	mux.HandleFunc("/", server.uiHandler)

	return server
}

// SetScheduler sets the scheduler
func (s *Server) SetScheduler(sched *scheduler.Scheduler) {
	s.scheduler = sched
}

// Start starts the HTTP server
func (s *Server) Start(ctx context.Context) error {
	s.logger.WithFields(logrus.Fields{
		"addr": s.httpServer.Addr,
	}).Info("Starting HTTP server")

	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.WithError(err).Error("HTTP server error")
		}
	}()

	return nil
}

// Stop stops the HTTP server
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("Stopping HTTP server")
	return s.httpServer.Shutdown(ctx)
}

// healthHandler handles /health endpoint
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"healthy","uptime":"%s"}`, time.Since(s.startTime))
}

// readyHandler handles /ready endpoint
func (s *Server) readyHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"ready":true}`)
}

// metricsHandler handles /metrics endpoint
func (s *Server) metricsHandler(w http.ResponseWriter, r *http.Request) {
	metrics.WritePrometheus(w, true)
}

// jobsHandler handles /api/v1/jobs endpoint
func (s *Server) jobsHandler(w http.ResponseWriter, r *http.Request) {
	if s.scheduler == nil {
		http.Error(w, "Scheduler not initialized", http.StatusInternalServerError)
		return
	}

	jobs := s.scheduler.GetAllJobs()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(jobs); err != nil {
		s.logger.WithError(err).Error("Failed to encode jobs response")
	}
}

// statsHandler handles /api/v1/stats endpoint
func (s *Server) statsHandler(w http.ResponseWriter, r *http.Request) {
	if s.scheduler == nil {
		http.Error(w, "Scheduler not initialized", http.StatusInternalServerError)
		return
	}

	stats := s.scheduler.GetStats()
	response := map[string]interface{}{
		"scheduler":  stats,
		"uptime":     time.Since(s.startTime).String(),
		"start_time": s.startTime.Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.logger.WithError(err).Error("Failed to encode stats response")
	}
}

// targetsHandler handles /api/v1/targets endpoint.
func (s *Server) targetsHandler(w http.ResponseWriter, r *http.Request) {
	if s.scheduler == nil {
		http.Error(w, "Scheduler not initialized", http.StatusInternalServerError)
		return
	}

	jobs := s.scheduler.GetAllJobs()
	targets := make([]map[string]interface{}, 0, len(jobs))

	for _, job := range jobs {
		status := "down"
		if job.LastStatus == "up" {
			status = "up"
		} else if job.LastStatus == "" {
			status = "unknown"
		}

		targets = append(targets, map[string]interface{}{
			"id":              job.ID,
			"host":            job.Target.Host,
			"port":            job.Target.Port,
			"protocol":        job.Target.Protocol,
			"interval":        job.Interval.String(),
			"timeout":         job.Target.Timeout.String(),
			"labels":          job.Target.Labels,
			"enabled":         job.Target.Enabled,
			"next_run":        job.NextRun.Format(time.RFC3339),
			"status":          status,
			"success_count":   job.SuccessCount,
			"failed_count":    job.FailedCount,
			"last_probe_time": job.LastProbeTime.Format(time.RFC3339),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(targets); err != nil {
		s.logger.WithError(err).Error("Failed to encode targets response")
	}
}

// uiHandler handles the main UI page.
func (s *Server) uiHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	// Read HTML from embedded file
	htmlContent, err := staticFiles.ReadFile("static/index.html")
	if err != nil {
		s.logger.WithError(err).Error("Failed to read index.html")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err = w.Write(htmlContent); err != nil {
		s.logger.WithError(err).Error("Failed to write HTML content")
	}
}
