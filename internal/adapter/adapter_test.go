package adapter

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/gdagil/vmprober/internal/config"
	"github.com/gdagil/vmprober/internal/types"
)

func createTestPushConfig() *config.PushConfig {
	return &config.PushConfig{
		Enabled: true,
		Endpoints: []config.EndpointConfig{
			{
				URL: "http://localhost:8428/api/v1/import/prometheus",
				Headers: map[string]string{
					"Content-Type": "text/plain",
				},
			},
		},
		Retry: config.RetryConfig{
			MaxAttempts:  3,
			Backoff:     "exponential",
			InitialDelay: 1 * time.Second,
			MaxDelay:     30 * time.Second,
			Multiplier:   2.0,
		},
		Batch: config.BatchConfig{
			Size:    100,
			Timeout: 5 * time.Second,
		},
	}
}

func TestNewVictoriaMetricsAdapter(t *testing.T) {
	cfg := createTestPushConfig()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	adapter, err := NewVictoriaMetricsAdapter(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create adapter: %v", err)
	}

	if adapter == nil {
		t.Fatal("Adapter is nil")
	}

	ctx := context.Background()
	defer adapter.Stop(ctx)
}

func TestVictoriaMetricsAdapter_StartStop(t *testing.T) {
	cfg := createTestPushConfig()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	adapter, err := NewVictoriaMetricsAdapter(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create adapter: %v", err)
	}

	ctx := context.Background()
	if err := adapter.Start(ctx); err != nil {
		t.Fatalf("Failed to start adapter: %v", err)
	}

	// Give time to start
	time.Sleep(100 * time.Millisecond)

	if err := adapter.Stop(ctx); err != nil {
		t.Fatalf("Failed to stop adapter: %v", err)
	}
}

func TestVictoriaMetricsAdapter_Push(t *testing.T) {
	cfg := createTestPushConfig()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	adapter, err := NewVictoriaMetricsAdapter(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create adapter: %v", err)
	}

	ctx := context.Background()
	if err := adapter.Start(ctx); err != nil {
		t.Fatalf("Failed to start adapter: %v", err)
	}
	defer adapter.Stop(ctx)

	metrics := []types.Metric{
		{
			Name:      "test_metric",
			Value:     42.0,
			Timestamp: time.Now(),
			Labels: map[string]string{
				"test": "true",
			},
			Type: types.MetricTypeGauge,
		},
	}

	// Push may fail if endpoint is unavailable, this is normal for testing
	err = adapter.Push(ctx, metrics)
	if err != nil {
		t.Logf("Push failed (expected if endpoint unavailable): %v", err)
	}

	// Give time to process
	time.Sleep(200 * time.Millisecond)

	stats := adapter.GetStats()
	if stats == nil {
		t.Fatal("Stats is nil")
	}
}

func TestVictoriaMetricsAdapter_Flush(t *testing.T) {
	cfg := createTestPushConfig()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	adapter, err := NewVictoriaMetricsAdapter(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create adapter: %v", err)
	}

	ctx := context.Background()
	if err := adapter.Start(ctx); err != nil {
		t.Fatalf("Failed to start adapter: %v", err)
	}
	defer adapter.Stop(ctx)

	// Add metrics to queue
	metrics := []types.Metric{
		{
			Name:      "test_metric",
			Value:     1.0,
			Timestamp: time.Now(),
			Type:      types.MetricTypeGauge,
		},
	}

	adapter.Push(ctx, metrics)

	// Flush should send all metrics
	if err := adapter.Flush(ctx); err != nil {
		t.Logf("Flush failed (expected if endpoint unavailable): %v", err)
	}
}

func TestVictoriaMetricsAdapter_GetStats(t *testing.T) {
	cfg := createTestPushConfig()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	adapter, err := NewVictoriaMetricsAdapter(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create adapter: %v", err)
	}

	stats := adapter.GetStats()
	if stats == nil {
		t.Fatal("Stats is nil")
	}

	if stats.TotalPushed != 0 {
		t.Errorf("Expected 0 pushed initially, got %d", stats.TotalPushed)
	}
}

func TestNewVictoriaMetricsAdapter_NoEndpoints(t *testing.T) {
	cfg := &config.PushConfig{
		Enabled:   true,
		Endpoints: []config.EndpointConfig{},
		Batch: config.BatchConfig{
			Size:    100,
			Timeout: 5 * time.Second,
		},
	}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	_, err := NewVictoriaMetricsAdapter(cfg, logger)
	if err == nil {
		t.Error("Expected error when no endpoints configured")
	}
}


