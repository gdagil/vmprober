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
	"github.com/vmprober/vmprober/internal/config"
	"github.com/vmprober/vmprober/internal/types"
)

// VictoriaMetricsAdapter адаптер для отправки метрик в VictoriaMetrics
type VictoriaMetricsAdapter interface {
	// Start запускает адаптер
	Start(ctx context.Context) error

	// Stop останавливает адаптер
	Stop(ctx context.Context) error

	// Push отправляет метрики
	Push(ctx context.Context, metrics []types.Metric) error

	// Flush принудительно отправляет все буферизованные метрики
	Flush(ctx context.Context) error

	// GetStats возвращает статистику адаптера
	GetStats() *AdapterStats
}

// AdapterStats статистика адаптера
type AdapterStats struct {
	TotalPushed   int64         `json:"total_pushed"`
	TotalFailed   int64         `json:"total_failed"`
	RetryCount    int64         `json:"retry_count"`
	AvgPushTime   time.Duration `json:"avg_push_time"`
	QueueSize     int           `json:"queue_size"`
	LastPushTime  time.Time     `json:"last_push_time"`
	SuccessRate   float64       `json:"success_rate"`
}

// DefaultVictoriaMetricsAdapter реализация адаптера VictoriaMetrics
type DefaultVictoriaMetricsAdapter struct {
	config      *config.PushConfig
	endpoints   []*Endpoint
	client      *http.Client
	batchQueue  chan []types.Metric
	mu          sync.RWMutex
	logger      *logrus.Logger
	stats       *AdapterStats
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	formatter   Formatter
	retryEngine *RetryEngine
}

// Endpoint представляет endpoint для отправки метрик
type Endpoint struct {
	URL     string
	Headers map[string]string
	Auth    *AuthConfig
}

// AuthConfig конфигурация аутентификации
type AuthConfig struct {
	Type     string
	Token    string
	Username string
	Password string
}

// NewVictoriaMetricsAdapter создает новый адаптер VictoriaMetrics
func NewVictoriaMetricsAdapter(cfg *config.PushConfig, logger *logrus.Logger) (VictoriaMetricsAdapter, error) {
	if cfg == nil {
		return nil, fmt.Errorf("push config is nil")
	}

	ctx, cancel := context.WithCancel(context.Background())

	adapter := &DefaultVictoriaMetricsAdapter{
		config:     cfg,
		endpoints:  make([]*Endpoint, 0),
		client:     &http.Client{Timeout: 30 * time.Second},
		batchQueue: make(chan []types.Metric, cfg.Batch.Size),
		logger:     logger,
		stats:      &AdapterStats{},
		ctx:        ctx,
		cancel:     cancel,
		formatter:  NewPrometheusFormatter(),
		retryEngine: NewRetryEngine(&cfg.Retry, logger),
	}

	// Инициализация endpoints
	if err := adapter.initializeEndpoints(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize endpoints: %w", err)
	}

	return adapter, nil
}

// Start запускает адаптер
func (a *DefaultVictoriaMetricsAdapter) Start(ctx context.Context) error {
	// Запуск воркеров для обработки батчей
	workerCount := 3
	for i := 0; i < workerCount; i++ {
		a.wg.Add(1)
		go a.batchWorker(ctx)
	}

	a.logger.Info("VictoriaMetrics adapter started")
	return nil
}

// Stop останавливает адаптер
func (a *DefaultVictoriaMetricsAdapter) Stop(ctx context.Context) error {
	a.cancel()
	close(a.batchQueue)
	a.wg.Wait()
	a.logger.Info("VictoriaMetrics adapter stopped")
	return nil
}

// Push отправляет метрики
func (a *DefaultVictoriaMetricsAdapter) Push(ctx context.Context, metrics []types.Metric) error {
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
	}
}

// Flush принудительно отправляет все буферизованные метрики
func (a *DefaultVictoriaMetricsAdapter) Flush(ctx context.Context) error {
	// Отправка всех метрик из очереди
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

// GetStats возвращает статистику адаптера
func (a *DefaultVictoriaMetricsAdapter) GetStats() *AdapterStats {
	a.mu.RLock()
	defer a.mu.RUnlock()

	stats := *a.stats
	stats.QueueSize = len(a.batchQueue)

	if stats.TotalPushed+stats.TotalFailed > 0 {
		stats.SuccessRate = float64(stats.TotalPushed) / float64(stats.TotalPushed+stats.TotalFailed) * 100
	}

	return &stats
}

// batchWorker воркер для обработки батчей
func (a *DefaultVictoriaMetricsAdapter) batchWorker(ctx context.Context) {
	defer a.wg.Done()

	batch := make([]types.Metric, 0, a.config.Batch.Size)
	ticker := time.NewTicker(a.config.Batch.Timeout)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Отправка оставшихся метрик
			if len(batch) > 0 {
				if err := a.sendBatch(ctx, batch); err != nil {
					a.logger.WithError(err).Error("Failed to send final batch")
				}
			}
			return
		case <-a.ctx.Done():
			// Отправка оставшихся метрик
			if len(batch) > 0 {
				if err := a.sendBatch(ctx, batch); err != nil {
					a.logger.WithError(err).Error("Failed to send final batch")
				}
			}
			return
		case metrics := <-a.batchQueue:
			batch = append(batch, metrics...)
			if len(batch) >= a.config.Batch.Size {
				if err := a.sendBatch(ctx, batch); err != nil {
					a.logger.WithError(err).Error("Failed to send batch")
				}
				batch = batch[:0]
			}
		case <-ticker.C:
			if len(batch) > 0 {
				if err := a.sendBatch(ctx, batch); err != nil {
					a.logger.WithError(err).Error("Failed to send batch on timeout")
				}
				batch = batch[:0]
			}
		}
	}
}

// sendBatch отправляет батч метрик
func (a *DefaultVictoriaMetricsAdapter) sendBatch(ctx context.Context, metrics []types.Metric) error {
	if len(metrics) == 0 {
		return nil
	}

	start := time.Now()

	// Форматирование метрик
	data, err := a.formatter.Format(metrics)
	if err != nil {
		return fmt.Errorf("failed to format metrics: %w", err)
	}

	// Отправка на все endpoints
	var lastErr error
	for _, endpoint := range a.endpoints {
		if err := a.sendToEndpoint(ctx, endpoint, data); err != nil {
			lastErr = err
			a.logger.WithError(err).Error("Failed to send to endpoint", "endpoint", endpoint.URL)

			// Retry логика
			if a.retryEngine.ShouldRetry(err) {
				a.retryEngine.ScheduleRetry(ctx, func() error {
					return a.sendToEndpoint(ctx, endpoint, data)
				})
			}
		}
	}

	// Обновление статистики
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

// sendToEndpoint отправляет данные на endpoint
func (a *DefaultVictoriaMetricsAdapter) sendToEndpoint(ctx context.Context, endpoint *Endpoint, data []byte) error {
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint.URL, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Установка заголовков
	req.Header.Set("Content-Type", "text/plain")
	for k, v := range endpoint.Headers {
		req.Header.Set(k, v)
	}

	// Аутентификация
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
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// initializeEndpoints инициализирует endpoints
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

