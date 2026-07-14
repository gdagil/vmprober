package observability

import (
	"context"
	"fmt"
	"net/http"
	"net/http/pprof"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/gdagil/vmprober/internal/config"
)

// Manager manages the observability system.
type Manager interface {
	// Start starts observability components
	Start(ctx context.Context) error

	// Stop stops observability components
	Stop(ctx context.Context) error

	// GetHealth returns system health status
	GetHealth() *HealthStatus

	// GetMetrics returns system metrics
	GetMetrics() *SystemMetrics
}

// HealthStatus system health status
type HealthStatus struct {
	Status    string                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Uptime    time.Duration          `json:"uptime"`
	Checks    map[string]HealthCheck `json:"checks"`
}

// HealthCheck component health check
type HealthCheck struct {
	Status    string        `json:"status"`
	Message   string        `json:"message,omitempty"`
	Timestamp time.Time     `json:"timestamp"`
	Duration  time.Duration `json:"duration,omitempty"`
}

// SystemMetrics system metrics
type SystemMetrics struct {
	Timestamp  time.Time     `json:"timestamp"`
	Uptime     time.Duration `json:"uptime"`
	Goroutines int           `json:"goroutines"`
	Memory     MemoryMetrics `json:"memory"`
}

// MemoryMetrics memory metrics
type MemoryMetrics struct {
	Alloc      uint64 `json:"alloc"`
	TotalAlloc uint64 `json:"total_alloc"`
	Sys        uint64 `json:"sys"`
	HeapAlloc  uint64 `json:"heap_alloc"`
	HeapSys    uint64 `json:"heap_sys"`
}

// DefaultObservabilityManager observability manager implementation
type DefaultObservabilityManager struct {
	config      *config.ObservabilityConfig
	logger      *logrus.Logger
	pprofServer *http.Server
	startTime   time.Time
	mu          sync.RWMutex
	health      *HealthStatus
	metrics     *SystemMetrics
}

// NewManager creates a new observability manager.
func NewManager(cfg *config.ObservabilityConfig, logger *logrus.Logger) Manager {
	return &DefaultObservabilityManager{
		config:    cfg,
		logger:    logger,
		startTime: time.Now(),
		health: &HealthStatus{
			Status:    "healthy",
			Timestamp: time.Now(),
			Checks:    make(map[string]HealthCheck),
		},
		metrics: &SystemMetrics{
			Timestamp: time.Now(),
		},
	}
}

// Start starts observability components
func (o *DefaultObservabilityManager) Start(ctx context.Context) error {
	// Start pprof if enabled
	if o.config.Pprof.Enabled {
		if err := o.startPprof(ctx); err != nil {
			return fmt.Errorf("failed to start pprof: %w", err)
		}
	}

	// Start health check loop
	go o.healthCheckLoop(ctx)

	// Start metrics collection loop
	go o.metricsCollectionLoop(ctx)

	o.logger.Info("Observability manager started")
	return nil
}

// Stop stops observability components
func (o *DefaultObservabilityManager) Stop(ctx context.Context) error {
	if o.pprofServer != nil {
		shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := o.pprofServer.Shutdown(shutdownCtx); err != nil {
			o.logger.WithError(err).Error("Failed to shutdown pprof server")
		}
	}

	o.logger.Info("Observability manager stopped")
	return nil
}

// GetHealth returns system health status
func (o *DefaultObservabilityManager) GetHealth() *HealthStatus {
	o.mu.RLock()
	defer o.mu.RUnlock()

	health := *o.health
	health.Uptime = time.Since(o.startTime)
	health.Timestamp = time.Now()
	health.Checks = make(map[string]HealthCheck)
	for k, v := range o.health.Checks {
		health.Checks[k] = v
	}

	return &health
}

// GetMetrics returns system metrics
func (o *DefaultObservabilityManager) GetMetrics() *SystemMetrics {
	o.mu.RLock()
	defer o.mu.RUnlock()

	metrics := *o.metrics
	metrics.Timestamp = time.Now()
	metrics.Uptime = time.Since(o.startTime)

	return &metrics
}

// startPprof starts pprof server.
func (o *DefaultObservabilityManager) startPprof(_ context.Context) error {
	addr := fmt.Sprintf("%s:%d", o.config.Pprof.Host, o.config.Pprof.Port)

	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	o.pprofServer = &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		if err := o.pprofServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			o.logger.WithError(err).Error("pprof server error")
		}
	}()

	o.logger.WithField("addr", addr).Info("pprof server started")
	return nil
}

// healthCheckLoop health check loop
func (o *DefaultObservabilityManager) healthCheckLoop(ctx context.Context) {
	interval := o.config.HealthCheck.Interval
	if interval <= 0 {
		interval = 30 * time.Second // Default interval
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			o.updateHealth()
		}
	}
}

// metricsCollectionLoop metrics collection loop
func (o *DefaultObservabilityManager) metricsCollectionLoop(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			o.updateMetrics()
		}
	}
}

// updateHealth updates health status
func (o *DefaultObservabilityManager) updateHealth() {
	o.mu.Lock()
	defer o.mu.Unlock()

	// Basic health check
	o.health.Status = "healthy"
	o.health.Timestamp = time.Now()

	// In a real implementation, various component checks would be here
}

// updateMetrics updates system metrics
func (o *DefaultObservabilityManager) updateMetrics() {
	o.mu.Lock()
	defer o.mu.Unlock()

	// In a real implementation, runtime would be imported here to get metrics
	o.metrics.Timestamp = time.Now()
	o.metrics.Uptime = time.Since(o.startTime)
	// o.metrics.Goroutines = runtime.NumGoroutine()
	// var m runtime.MemStats
	// runtime.ReadMemStats(&m)
	// o.metrics.Memory.Alloc = m.Alloc
	// etc.
}
