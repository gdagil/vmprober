package adapter

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/gdagil/vmprober/internal/config"
	"github.com/gdagil/vmprober/internal/types"
)

// VictoriaMetricsAdapter is an adapter for sending metrics to VictoriaMetrics
type VictoriaMetricsAdapter interface {
	// Start starts the adapter
	Start(ctx context.Context) error

	// Stop stops the adapter
	Stop(ctx context.Context) error

	// Push sends metrics
	Push(ctx context.Context, metrics []types.Metric) error

	// Flush forcefully sends all buffered metrics
	Flush(ctx context.Context) error

	// GetStats returns adapter statistics.
	GetStats() *Stats
}

// Stats represents adapter statistics.
type Stats struct {
	TotalPushed  int64         `json:"total_pushed"`
	TotalFailed  int64         `json:"total_failed"`
	RetryCount   int64         `json:"retry_count"`
	AvgPushTime  time.Duration `json:"avg_push_time"`
	QueueSize    int           `json:"queue_size"`
	LastPushTime time.Time     `json:"last_push_time"`
	SuccessRate  float64       `json:"success_rate"`
}

// DefaultVictoriaMetricsAdapter is the VictoriaMetrics adapter implementation
type DefaultVictoriaMetricsAdapter struct {
	config      *config.PushConfig
	endpoints   []*Endpoint
	client      *http.Client
	batchQueue  chan []types.Metric
	mu          sync.RWMutex
	logger      *logrus.Logger
	stats       *Stats
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	formatter   Formatter
	retryEngine *RetryEngine
}

// Endpoint represents an endpoint for sending metrics
type Endpoint struct {
	URL     string
	Headers map[string]string
	Auth    *AuthConfig
}

// AuthConfig represents authentication configuration
type AuthConfig struct {
	Type     string
	Token    string
	Username string
	Password string
}

// NewVictoriaMetricsAdapter creates a new VictoriaMetrics adapter
func NewVictoriaMetricsAdapter(cfg *config.PushConfig, logger *logrus.Logger) (VictoriaMetricsAdapter, error) {
	if cfg == nil {
		return nil, fmt.Errorf("push config is nil")
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Configure HTTP client with timeouts
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	}
	client := &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
	}

	adapter := &DefaultVictoriaMetricsAdapter{
		config:      cfg,
		endpoints:   make([]*Endpoint, 0),
		client:      client,
		batchQueue:  make(chan []types.Metric, cfg.Batch.Size),
		logger:      logger,
		stats:       &Stats{},
		ctx:         ctx,
		cancel:      cancel,
		formatter:   NewJSONLineFormatter(), // Use JSON line format for vminsert (/api/v1/import)
		retryEngine: NewRetryEngine(&cfg.Retry, logger),
	}

	// Initialize endpoints
	if err := adapter.initializeEndpoints(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize endpoints: %w", err)
	}

	return adapter, nil
}

// Start starts the adapter
func (a *DefaultVictoriaMetricsAdapter) Start(ctx context.Context) error {
	// Start workers for batch processing
	workerCount := 3
	for i := 0; i < workerCount; i++ {
		a.wg.Add(1)
		go a.batchWorker(ctx)
	}

	a.logger.Info("VictoriaMetrics adapter started")
	return nil
}

// Stop stops the adapter.
func (a *DefaultVictoriaMetricsAdapter) Stop(_ context.Context) error {
	a.cancel()
	close(a.batchQueue)
	a.wg.Wait()

	// Stop retry engine
	if a.retryEngine != nil {
		a.retryEngine.Stop()
	}

	a.logger.Info("VictoriaMetrics adapter stopped")
	return nil
}

// Push sends metrics
func (a *DefaultVictoriaMetricsAdapter) Push(ctx context.Context, metrics []types.Metric) error {
	// Use timeout to avoid infinite blocking
	pushTimeout := 5 * time.Second
	if a.config.Batch.Timeout < pushTimeout {
		pushTimeout = a.config.Batch.Timeout
	}

	select {
	case a.batchQueue <- metrics:
		a.mu.Lock()
		a.stats.QueueSize = len(a.batchQueue)
		a.mu.Unlock()
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-a.ctx.Done():
		return fmt.Errorf("adapter is stopped")
	case <-time.After(pushTimeout):
		// Queue is full, trying to send directly
		a.logger.WithField("queue_size", len(a.batchQueue)).Warn("Batch queue is full, sending metrics directly")
		return a.sendBatch(ctx, metrics)
	}
}

// Flush forcefully sends all buffered metrics
func (a *DefaultVictoriaMetricsAdapter) Flush(ctx context.Context) error {
	// Send all metrics from queue
	for {
		select {
		case metrics := <-a.batchQueue:
			if err := a.sendBatch(ctx, metrics); err != nil {
				a.logger.WithError(err).Error("Failed to send batch during flush")
			}
		default:
			return nil
		}
	}
}

// GetStats returns adapter statistics.
func (a *DefaultVictoriaMetricsAdapter) GetStats() *Stats {
	a.mu.RLock()
	defer a.mu.RUnlock()

	stats := *a.stats
	stats.QueueSize = len(a.batchQueue)

	if stats.TotalPushed+stats.TotalFailed > 0 {
		stats.SuccessRate = float64(stats.TotalPushed) / float64(stats.TotalPushed+stats.TotalFailed) * 100
	}

	return &stats
}

// batchWorker is a worker for batch processing
func (a *DefaultVictoriaMetricsAdapter) batchWorker(ctx context.Context) {
	defer a.wg.Done()

	// Handle panics to prevent worker crash
	defer func() {
		if r := recover(); r != nil {
			a.logger.WithField("panic", r).Error("Panic in batchWorker, recovering")
		}
	}()

	batch := make([]types.Metric, 0, a.config.Batch.Size)
	ticker := time.NewTicker(a.config.Batch.Timeout)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Send remaining metrics
			if len(batch) > 0 {
				if err := a.sendBatch(ctx, batch); err != nil {
					a.logger.WithError(err).Error("Failed to send final batch")
				}
			}
			return
		case <-a.ctx.Done():
			// Send remaining metrics
			if len(batch) > 0 {
				if err := a.sendBatch(ctx, batch); err != nil {
					a.logger.WithError(err).Error("Failed to send final batch")
				}
			}
			return
		case metrics, ok := <-a.batchQueue:
			if !ok {
				// Channel closed, send remaining metrics
				if len(batch) > 0 {
					if err := a.sendBatch(ctx, batch); err != nil {
						a.logger.WithError(err).Error("Failed to send final batch after channel close")
					}
				}
				return
			}
			batch = append(batch, metrics...)
			if len(batch) >= a.config.Batch.Size {
				if err := a.sendBatch(ctx, batch); err != nil {
					a.logger.WithError(err).Error("Failed to send batch")
					// Continue working even on error to avoid blocking processing
				}
				batch = batch[:0]
			}
		case <-ticker.C:
			if len(batch) > 0 {
				if err := a.sendBatch(ctx, batch); err != nil {
					a.logger.WithError(err).Error("Failed to send batch on timeout")
					// Continue working even on error
				}
				batch = batch[:0]
			}
		}
	}
}

// sendBatch sends a batch of metrics
func (a *DefaultVictoriaMetricsAdapter) sendBatch(ctx context.Context, metrics []types.Metric) error {
	if len(metrics) == 0 {
		return nil
	}

	start := time.Now()

	// Format metrics
	data, err := a.formatter.Format(metrics)
	if err != nil {
		return fmt.Errorf("failed to format metrics: %w", err)
	}

	// Send to all endpoints
	var lastErr error
	successCount := 0
	for _, endpoint := range a.endpoints {
		if err := a.sendToEndpoint(ctx, endpoint, data); err != nil {
			lastErr = err
			a.logger.WithError(err).WithField("endpoint", endpoint.URL).Error("Failed to send metrics to endpoint")

			// Retry logic
			if a.retryEngine.ShouldRetry(err) {
				a.retryEngine.ScheduleRetry(ctx, endpoint.URL, func() error {
					return a.sendToEndpoint(ctx, endpoint, data)
				})
			}
		} else {
			successCount++
			a.logger.WithField("endpoint", endpoint.URL).Debug("Successfully sent metrics to endpoint")
		}
	}

	// If at least one endpoint successfully received metrics, consider it successful
	if successCount > 0 {
		lastErr = nil
	}

	// Update statistics
	a.mu.Lock()
	if lastErr == nil {
		a.stats.TotalPushed += int64(len(metrics))
	} else {
		a.stats.TotalFailed += int64(len(metrics))
	}
	a.stats.AvgPushTime = time.Since(start)
	a.stats.LastPushTime = time.Now()
	a.mu.Unlock()

	return lastErr
}

// sendToEndpoint sends data to an endpoint
func (a *DefaultVictoriaMetricsAdapter) sendToEndpoint(ctx context.Context, endpoint *Endpoint, data []byte) error {
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint.URL, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	// For JSON line format we use application/json
	// For vminsert endpoint /insert/0/prometheus/api/v1/import vmimport JSON format is used
	req.Header.Set("Content-Type", "application/json")
	for k, v := range endpoint.Headers {
		req.Header.Set(k, v)
	}

	// Authentication
	if endpoint.Auth != nil {
		switch endpoint.Auth.Type {
		case "bearer", "token":
			req.Header.Set("Authorization", "Bearer "+endpoint.Auth.Token)
		case "basic":
			req.SetBasicAuth(endpoint.Auth.Username, endpoint.Auth.Password)
		}
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("unexpected status code: %d, failed to read body: %w", resp.StatusCode, err)
		}
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// initializeEndpoints initializes endpoints
func (a *DefaultVictoriaMetricsAdapter) initializeEndpoints() error {
	for _, endpointCfg := range a.config.Endpoints {
		endpoint := &Endpoint{
			URL:     endpointCfg.URL,
			Headers: endpointCfg.Headers,
		}

		if endpointCfg.Auth.Type != "" {
			endpoint.Auth = &AuthConfig{
				Type:     endpointCfg.Auth.Type,
				Token:    endpointCfg.Auth.Token,
				Username: endpointCfg.Auth.Username,
				Password: endpointCfg.Auth.Password,
			}
		}

		a.endpoints = append(a.endpoints, endpoint)
	}

	if len(a.endpoints) == 0 {
		return fmt.Errorf("no endpoints configured")
	}

	return nil
}
