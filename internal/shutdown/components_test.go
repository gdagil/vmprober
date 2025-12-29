package shutdown

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/gdagil/vmprober/internal/adapter"
	"github.com/gdagil/vmprober/internal/config"
	"github.com/gdagil/vmprober/internal/normalizer"
	"github.com/gdagil/vmprober/internal/observability"
	"github.com/gdagil/vmprober/internal/scheduler"
	"github.com/gdagil/vmprober/internal/server"
	"github.com/gdagil/vmprober/internal/wal"
)

func TestNewServerComponent(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	srv := server.NewServer("127.0.0.1", 0, logger)
	component := NewServerComponent(srv)

	if component == nil {
		t.Fatal("NewServerComponent returned nil")
	}

	if component.Name() != "http_server" {
		t.Errorf("Expected name 'http_server', got '%s'", component.Name())
	}

	if component.Priority() != 1 {
		t.Errorf("Expected priority 1, got %d", component.Priority())
	}

	ctx := context.Background()
	if err := srv.Start(ctx); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	// Shutdown should work
	if err := component.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}
}

func TestNewSchedulerComponent(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	sched := scheduler.NewScheduler(logger)
	component := NewSchedulerComponent(sched)

	if component == nil {
		t.Fatal("NewSchedulerComponent returned nil")
	}

	if component.Name() != "scheduler" {
		t.Errorf("Expected name 'scheduler', got '%s'", component.Name())
	}

	if component.Priority() != 2 {
		t.Errorf("Expected priority 2, got %d", component.Priority())
	}

	ctx := context.Background()
	if err := sched.Start(ctx); err != nil {
		t.Fatalf("Failed to start scheduler: %v", err)
	}

	// Shutdown should work
	if err := component.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}
}

func TestNewWALComponent(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	cfg := &config.WALConfig{
		Dir:         t.TempDir(),
		SegmentSize: "1MB",
		MaxAge:      1 * time.Hour,
	}

	walManager, err := wal.NewWALManager(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create WAL manager: %v", err)
	}

	component := NewWALComponent(walManager)

	if component == nil {
		t.Fatal("NewWALComponent returned nil")
	}

	if component.Name() != "wal_system" {
		t.Errorf("Expected name 'wal_system', got '%s'", component.Name())
	}

	if component.Priority() != 3 {
		t.Errorf("Expected priority 3, got %d", component.Priority())
	}

	ctx := context.Background()

	// Shutdown should work (includes Flush)
	if err := component.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}
}

func TestNewAdapterComponent(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	cfg := &config.PushConfig{
		Enabled: true,
		Endpoints: []config.EndpointConfig{
			{
				URL: "http://localhost:8428/api/v1/import/prometheus",
			},
		},
		Batch: config.BatchConfig{
			Size:    100,
			Timeout: 5 * time.Second,
		},
	}

	vmAdapter, err := adapter.NewVictoriaMetricsAdapter(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create adapter: %v", err)
	}

	component := NewAdapterComponent(vmAdapter)

	if component == nil {
		t.Fatal("NewAdapterComponent returned nil")
	}

	if component.Name() != "victoriametrics_adapter" {
		t.Errorf("Expected name 'victoriametrics_adapter', got '%s'", component.Name())
	}

	if component.Priority() != 4 {
		t.Errorf("Expected priority 4, got %d", component.Priority())
	}

	ctx := context.Background()
	if err := vmAdapter.Start(ctx); err != nil {
		t.Fatalf("Failed to start adapter: %v", err)
	}

	// Shutdown should work (includes Flush)
	if err := component.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}
}

func TestNewNormalizerComponent(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	norm := normalizer.NewNormalizer(logger)
	component := NewNormalizerComponent(norm)

	if component == nil {
		t.Fatal("NewNormalizerComponent returned nil")
	}

	if component.Name() != "normalizer" {
		t.Errorf("Expected name 'normalizer', got '%s'", component.Name())
	}

	if component.Priority() != 5 {
		t.Errorf("Expected priority 5, got %d", component.Priority())
	}

	ctx := context.Background()

	// Shutdown should work
	if err := component.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}
}

func TestNewObservabilityComponent(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	cfg := &config.ObservabilityConfig{
		Pprof: config.PprofConfig{
			Enabled: false,
		},
		HealthCheck: config.HealthCheckConfig{
			Enabled:  true,
			Interval: 30 * time.Second,
		},
	}

	obsManager := observability.NewManager(cfg, logger)
	component := NewObservabilityComponent(obsManager)

	if component == nil {
		t.Fatal("NewObservabilityComponent returned nil")
	}

	if component.Name() != "observability" {
		t.Errorf("Expected name 'observability', got '%s'", component.Name())
	}

	if component.Priority() != 6 {
		t.Errorf("Expected priority 6, got %d", component.Priority())
	}

	ctx := context.Background()
	if err := obsManager.Start(ctx); err != nil {
		t.Fatalf("Failed to start observability: %v", err)
	}

	// Shutdown should work
	if err := component.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}
}
