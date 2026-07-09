package shutdown

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/gdagil/vmprober/internal/adapter"
	"github.com/gdagil/vmprober/internal/config"
	"github.com/gdagil/vmprober/internal/normalizer"
	"github.com/gdagil/vmprober/internal/observability"
	"github.com/gdagil/vmprober/internal/scheduler"
	"github.com/gdagil/vmprober/internal/server"
	"github.com/gdagil/vmprober/internal/types"
	"github.com/gdagil/vmprober/internal/wal"
)

// mockWALManager is a wal.WALManager stub that lets tests control the errors
// returned by Flush and Close so the WALComponent.Shutdown error paths can be
// exercised deterministically.
type mockWALManager struct {
	flushErr    error
	closeErr    error
	flushCalled bool
	closeCalled bool
}

func (m *mockWALManager) Write(_ context.Context, _ *types.Record) error { return nil }
func (m *mockWALManager) Read(_ context.Context, _ wal.WALFilter) ([]*types.Record, error) {
	return nil, nil
}
func (m *mockWALManager) Flush(_ context.Context) error {
	m.flushCalled = true
	return m.flushErr
}
func (m *mockWALManager) Rotate(_ context.Context) error { return nil }
func (m *mockWALManager) Close(_ context.Context) error {
	m.closeCalled = true
	return m.closeErr
}
func (m *mockWALManager) GetStats() *wal.WALStats                        { return &wal.WALStats{} }
func (m *mockWALManager) MarkSent(_ context.Context, _ string) error     { return nil }
func (m *mockWALManager) GetUnsentRecords(_ context.Context) ([]*types.Record, error) {
	return nil, nil
}
func (m *mockWALManager) DeleteSentRecords(_ context.Context, _ time.Duration) error { return nil }

// mockVMAdapter is an adapter.VictoriaMetricsAdapter stub that lets tests
// control the errors returned by Flush and Stop so the AdapterComponent.Shutdown
// error paths can be exercised deterministically.
type mockVMAdapter struct {
	flushErr    error
	stopErr     error
	flushCalled bool
	stopCalled  bool
}

func (m *mockVMAdapter) Start(_ context.Context) error            { return nil }
func (m *mockVMAdapter) Push(_ context.Context, _ []types.Metric) error { return nil }
func (m *mockVMAdapter) GetStats() *adapter.Stats                 { return &adapter.Stats{} }
func (m *mockVMAdapter) Flush(_ context.Context) error {
	m.flushCalled = true
	return m.flushErr
}
func (m *mockVMAdapter) Stop(_ context.Context) error {
	m.stopCalled = true
	return m.stopErr
}

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

func TestWALComponent_Shutdown_FlushError(t *testing.T) {
	flushErr := errors.New("flush boom")
	mock := &mockWALManager{flushErr: flushErr}
	component := NewWALComponent(mock)

	err := component.Shutdown(context.Background())
	if !errors.Is(err, flushErr) {
		t.Fatalf("Expected flush error to be returned, got %v", err)
	}
	if !mock.flushCalled {
		t.Error("Flush should have been called")
	}
	if mock.closeCalled {
		t.Error("Close should not be called when Flush fails")
	}
}

func TestWALComponent_Shutdown_CloseError(t *testing.T) {
	closeErr := errors.New("close boom")
	mock := &mockWALManager{closeErr: closeErr}
	component := NewWALComponent(mock)

	err := component.Shutdown(context.Background())
	if !errors.Is(err, closeErr) {
		t.Fatalf("Expected close error to be returned, got %v", err)
	}
	if !mock.flushCalled {
		t.Error("Flush should have been called before Close")
	}
	if !mock.closeCalled {
		t.Error("Close should have been called after a successful Flush")
	}
}

func TestWALComponent_Shutdown_Success(t *testing.T) {
	mock := &mockWALManager{}
	component := NewWALComponent(mock)

	if err := component.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown should succeed, got %v", err)
	}
	if !mock.flushCalled || !mock.closeCalled {
		t.Errorf("Both Flush and Close should be called (flush=%v close=%v)", mock.flushCalled, mock.closeCalled)
	}
}

func TestAdapterComponent_Shutdown_FlushError(t *testing.T) {
	flushErr := errors.New("flush boom")
	mock := &mockVMAdapter{flushErr: flushErr}
	component := NewAdapterComponent(mock)

	err := component.Shutdown(context.Background())
	if !errors.Is(err, flushErr) {
		t.Fatalf("Expected flush error to be returned, got %v", err)
	}
	if !mock.flushCalled {
		t.Error("Flush should have been called")
	}
	if mock.stopCalled {
		t.Error("Stop should not be called when Flush fails")
	}
}

func TestAdapterComponent_Shutdown_StopError(t *testing.T) {
	stopErr := errors.New("stop boom")
	mock := &mockVMAdapter{stopErr: stopErr}
	component := NewAdapterComponent(mock)

	err := component.Shutdown(context.Background())
	if !errors.Is(err, stopErr) {
		t.Fatalf("Expected stop error to be returned, got %v", err)
	}
	if !mock.flushCalled {
		t.Error("Flush should have been called before Stop")
	}
	if !mock.stopCalled {
		t.Error("Stop should have been called after a successful Flush")
	}
}

func TestAdapterComponent_Shutdown_Success(t *testing.T) {
	mock := &mockVMAdapter{}
	component := NewAdapterComponent(mock)

	if err := component.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown should succeed, got %v", err)
	}
	if !mock.flushCalled || !mock.stopCalled {
		t.Errorf("Both Flush and Stop should be called (flush=%v stop=%v)", mock.flushCalled, mock.stopCalled)
	}
}
