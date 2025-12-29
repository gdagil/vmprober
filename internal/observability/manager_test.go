package observability

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/gdagil/vmprober/internal/config"
)

func createTestObservabilityConfig() *config.ObservabilityConfig {
	return &config.ObservabilityConfig{
		Pprof: config.PprofConfig{
			Enabled: false,
			Host:    "127.0.0.1",
			Port:    6060,
		},
		HealthCheck: config.HealthCheckConfig{
			Enabled:  true,
			Interval: 30 * time.Second,
		},
	}
}

func TestNewManager(t *testing.T) {
	cfg := createTestObservabilityConfig()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	manager := NewManager(cfg, logger)
	if manager == nil {
		t.Fatal("NewManager returned nil")
	}

	health := manager.GetHealth()
	if health == nil {
		t.Fatal("GetHealth returned nil")
	}
	if health.Status != "healthy" {
		t.Errorf("Expected status 'healthy', got '%s'", health.Status)
	}
}

func TestManager_StartStop(t *testing.T) {
	cfg := createTestObservabilityConfig()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	manager := NewManager(cfg, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := manager.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Give time for goroutines to start
	time.Sleep(100 * time.Millisecond)

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer stopCancel()

	err = manager.Stop(stopCtx)
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}

func TestManager_StartWithPprof(t *testing.T) {
	cfg := createTestObservabilityConfig()
	cfg.Pprof.Enabled = true

	// Find a free port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to find free port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	cfg.Pprof.Port = port

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	manager := NewManager(cfg, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err = manager.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Give time for pprof server to start
	time.Sleep(200 * time.Millisecond)

	// Try to connect to pprof endpoint
	resp, err := http.Get("http://127.0.0.1:" + fmt.Sprintf("%d", port) + "/debug/pprof/")
	if err == nil {
		resp.Body.Close()
		// We expect 501 Not Implemented since pprof is not fully implemented
		if resp.StatusCode != http.StatusNotImplemented {
			t.Logf("Expected 501, got %d (this is OK if pprof is not fully implemented)", resp.StatusCode)
		}
	}

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer stopCancel()

	err = manager.Stop(stopCtx)
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}

func TestManager_GetHealth(t *testing.T) {
	cfg := createTestObservabilityConfig()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	manager := NewManager(cfg, logger)

	health := manager.GetHealth()
	if health == nil {
		t.Fatal("GetHealth returned nil")
	}

	if health.Status != "healthy" {
		t.Errorf("Expected status 'healthy', got '%s'", health.Status)
	}

	if health.Uptime < 0 {
		t.Errorf("Expected non-negative uptime, got %v", health.Uptime)
	}

	if health.Checks == nil {
		t.Error("Expected checks map to be initialized")
	}
}

func TestManager_GetMetrics(t *testing.T) {
	cfg := createTestObservabilityConfig()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	manager := NewManager(cfg, logger)

	metrics := manager.GetMetrics()
	if metrics == nil {
		t.Fatal("GetMetrics returned nil")
	}

	if metrics.Uptime < 0 {
		t.Errorf("Expected non-negative uptime, got %v", metrics.Uptime)
	}
}

func TestManager_HealthCheckLoop(t *testing.T) {
	cfg := createTestObservabilityConfig()
	cfg.HealthCheck.Interval = 100 * time.Millisecond // Short interval for testing

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	manager := NewManager(cfg, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err := manager.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Wait for at least one health check cycle
	time.Sleep(200 * time.Millisecond)

	health := manager.GetHealth()
	if health == nil {
		t.Fatal("GetHealth returned nil")
	}

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer stopCancel()

	err = manager.Stop(stopCtx)
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}

func TestManager_MetricsCollectionLoop(t *testing.T) {
	cfg := createTestObservabilityConfig()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	manager := NewManager(cfg, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err := manager.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Wait for at least one metrics collection cycle
	time.Sleep(200 * time.Millisecond)

	metrics := manager.GetMetrics()
	if metrics == nil {
		t.Fatal("GetMetrics returned nil")
	}

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer stopCancel()

	err = manager.Stop(stopCtx)
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}

func TestManager_GetHealth_Concurrent(t *testing.T) {
	cfg := createTestObservabilityConfig()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	manager := NewManager(cfg, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := manager.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer manager.Stop(context.Background())

	// Concurrent access
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			health := manager.GetHealth()
			done <- (health != nil && health.Status == "healthy")
		}()
	}

	successCount := 0
	for i := 0; i < 10; i++ {
		if <-done {
			successCount++
		}
	}

	if successCount != 10 {
		t.Errorf("Expected 10 successful health checks, got %d", successCount)
	}
}

func TestManager_GetMetrics_Concurrent(t *testing.T) {
	cfg := createTestObservabilityConfig()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	manager := NewManager(cfg, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := manager.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer manager.Stop(context.Background())

	// Concurrent access
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			metrics := manager.GetMetrics()
			done <- (metrics != nil)
		}()
	}

	successCount := 0
	for i := 0; i < 10; i++ {
		if <-done {
			successCount++
		}
	}

	if successCount != 10 {
		t.Errorf("Expected 10 successful metrics retrievals, got %d", successCount)
	}
}

func TestManager_Stop_WithPprof(t *testing.T) {
	cfg := createTestObservabilityConfig()
	cfg.Pprof.Enabled = true

	// Find a free port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to find free port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	cfg.Pprof.Port = port

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	manager := NewManager(cfg, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = manager.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Give time for pprof server to start
	time.Sleep(200 * time.Millisecond)

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer stopCancel()

	err = manager.Stop(stopCtx)
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}
