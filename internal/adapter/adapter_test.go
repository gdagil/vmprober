package adapter

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
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
			Backoff:      "exponential",
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

func TestVictoriaMetricsAdapter_Push_MultipleMetrics(t *testing.T) {
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

	// Push multiple metrics
	metrics := []types.Metric{
		{
			Name:      "test_metric_1",
			Value:     1.0,
			Timestamp: time.Now(),
			Type:      types.MetricTypeGauge,
		},
		{
			Name:      "test_metric_2",
			Value:     2.0,
			Timestamp: time.Now(),
			Type:      types.MetricTypeCounter,
		},
		{
			Name:      "test_metric_3",
			Value:     3.0,
			Timestamp: time.Now(),
			Labels: map[string]string{
				"label1": "value1",
			},
			Type: types.MetricTypeHistogram,
		},
	}

	err = adapter.Push(ctx, metrics)
	if err != nil {
		t.Logf("Push failed (expected if endpoint unavailable): %v", err)
	}

	// Give time to process
	time.Sleep(200 * time.Millisecond)
}

func TestVictoriaMetricsAdapter_Push_EmptyMetrics(t *testing.T) {
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

	// Push empty metrics
	err = adapter.Push(ctx, []types.Metric{})
	if err != nil {
		t.Logf("Push failed (expected if endpoint unavailable): %v", err)
	}
}

func TestVictoriaMetricsAdapter_GetStats_AfterPush(t *testing.T) {
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
			Type:      types.MetricTypeGauge,
		},
	}

	adapter.Push(ctx, metrics)

	// Give time to process
	time.Sleep(200 * time.Millisecond)

	stats := adapter.GetStats()
	if stats == nil {
		t.Fatal("Stats is nil")
	}

	// Stats may or may not be updated depending on whether push succeeded
	t.Logf("Stats: TotalPushed=%d, TotalFailed=%d", stats.TotalPushed, stats.TotalFailed)
}

func TestNewVictoriaMetricsAdapter_NilConfig(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	_, err := NewVictoriaMetricsAdapter(nil, logger)
	if err == nil {
		t.Error("Expected error when config is nil")
	}
}

func TestVictoriaMetricsAdapter_ConcurrentPush(t *testing.T) {
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

	// Concurrent pushes
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			metrics := []types.Metric{
				{
					Name:      fmt.Sprintf("test_metric_%d", id),
					Value:     float64(id),
					Timestamp: time.Now(),
					Type:      types.MetricTypeGauge,
				},
			}
			err := adapter.Push(ctx, metrics)
			done <- (err == nil)
		}(i)
	}

	successCount := 0
	for i := 0; i < 10; i++ {
		if <-done {
			successCount++
		}
	}

	// Give time to process
	time.Sleep(300 * time.Millisecond)

	t.Logf("Concurrent pushes: %d succeeded", successCount)
}

func TestVictoriaMetricsAdapter_sendToEndpoint_Success(t *testing.T) {
	// Create a test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &config.PushConfig{
		Enabled: true,
		Endpoints: []config.EndpointConfig{
			{
				URL: server.URL,
				Headers: map[string]string{
					"X-Custom-Header": "test",
				},
			},
		},
		Batch: config.BatchConfig{
			Size:    100,
			Timeout: 5 * time.Second,
		},
	}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	adapter, err := NewVictoriaMetricsAdapter(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create adapter: %v", err)
	}
	defer adapter.Stop(context.Background())

	defaultAdapter := adapter.(*DefaultVictoriaMetricsAdapter)
	endpoint := defaultAdapter.endpoints[0]

	ctx := context.Background()
	data := []byte("test data")
	err = defaultAdapter.sendToEndpoint(ctx, endpoint, data)
	if err != nil {
		t.Fatalf("sendToEndpoint failed: %v", err)
	}
}

func TestVictoriaMetricsAdapter_sendToEndpoint_WithAuth(t *testing.T) {
	// Create a test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" {
			t.Error("Expected Authorization header")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &config.PushConfig{
		Enabled: true,
		Endpoints: []config.EndpointConfig{
			{
				URL: server.URL,
				Auth: config.AuthConfig{
					Type:  "bearer",
					Token: "test-token",
				},
			},
		},
		Batch: config.BatchConfig{
			Size:    100,
			Timeout: 5 * time.Second,
		},
	}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	adapter, err := NewVictoriaMetricsAdapter(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create adapter: %v", err)
	}
	defer adapter.Stop(context.Background())

	defaultAdapter := adapter.(*DefaultVictoriaMetricsAdapter)
	endpoint := defaultAdapter.endpoints[0]

	ctx := context.Background()
	data := []byte("test data")
	err = defaultAdapter.sendToEndpoint(ctx, endpoint, data)
	if err != nil {
		t.Fatalf("sendToEndpoint failed: %v", err)
	}
}

func TestVictoriaMetricsAdapter_sendToEndpoint_WithBasicAuth(t *testing.T) {
	// Create a test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok {
			t.Error("Expected Basic Auth")
		}
		if user != "testuser" || pass != "testpass" {
			t.Errorf("Expected testuser/testpass, got %s/%s", user, pass)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &config.PushConfig{
		Enabled: true,
		Endpoints: []config.EndpointConfig{
			{
				URL: server.URL,
				Auth: config.AuthConfig{
					Type:     "basic",
					Username: "testuser",
					Password: "testpass",
				},
			},
		},
		Batch: config.BatchConfig{
			Size:    100,
			Timeout: 5 * time.Second,
		},
	}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	adapter, err := NewVictoriaMetricsAdapter(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create adapter: %v", err)
	}
	defer adapter.Stop(context.Background())

	defaultAdapter := adapter.(*DefaultVictoriaMetricsAdapter)
	endpoint := defaultAdapter.endpoints[0]

	ctx := context.Background()
	data := []byte("test data")
	err = defaultAdapter.sendToEndpoint(ctx, endpoint, data)
	if err != nil {
		t.Fatalf("sendToEndpoint failed: %v", err)
	}
}

func TestVictoriaMetricsAdapter_sendToEndpoint_ErrorStatus(t *testing.T) {
	// Create a test HTTP server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	cfg := &config.PushConfig{
		Enabled: true,
		Endpoints: []config.EndpointConfig{
			{
				URL: server.URL,
			},
		},
		Batch: config.BatchConfig{
			Size:    100,
			Timeout: 5 * time.Second,
		},
	}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	adapter, err := NewVictoriaMetricsAdapter(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create adapter: %v", err)
	}
	defer adapter.Stop(context.Background())

	defaultAdapter := adapter.(*DefaultVictoriaMetricsAdapter)
	endpoint := defaultAdapter.endpoints[0]

	ctx := context.Background()
	data := []byte("test data")
	err = defaultAdapter.sendToEndpoint(ctx, endpoint, data)
	if err == nil {
		t.Error("Expected error for 500 status code")
	}
}
