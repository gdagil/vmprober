package adapter

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vmprober/vmprober/internal/config"
)

// RetryEngine управляет retry логикой
type RetryEngine struct {
	config     *config.RetryConfig
	logger     *logrus.Logger
	retryQueue chan retryTask
	mu         sync.RWMutex
	stats      *RetryStats
}

// RetryStats статистика retry
type RetryStats struct {
	TotalRetries      int64 `json:"total_retries"`
	SuccessfulRetries int64 `json:"successful_retries"`
	FailedRetries     int64 `json:"failed_retries"`
}

// retryTask задача для retry
type retryTask struct {
	fn       func() error
	attempts int
	delay    time.Duration
	endpoint string
}

// NewRetryEngine создает новый retry engine
func NewRetryEngine(cfg *config.RetryConfig, logger *logrus.Logger) *RetryEngine {
	if cfg == nil {
		cfg = &config.RetryConfig{
			MaxAttempts:  3,
			Backoff:     "exponential",
			InitialDelay: 1 * time.Second,
			MaxDelay:     30 * time.Second,
			Multiplier:   2.0,
		}
	}

	engine := &RetryEngine{
		config:     cfg,
		logger:     logger,
		retryQueue: make(chan retryTask, 100),
		stats:      &RetryStats{},
	}

	// Запуск воркера для обработки retry
	go engine.retryWorker(context.Background())

	return engine
}

// ShouldRetry проверяет нужно ли делать retry
func (r *RetryEngine) ShouldRetry(err error) bool {
	if err == nil {
		return false
	}

	// Проверка типа ошибки
	// В реальной реализации здесь была бы более сложная логика
	return true
}

// ScheduleRetry планирует retry
func (r *RetryEngine) ScheduleRetry(ctx context.Context, endpoint string, fn func() error) {
	task := retryTask{
		fn:       fn,
		attempts: 0,
		delay:    r.config.InitialDelay,
		endpoint: endpoint,
	}

	select {
	case r.retryQueue <- task:
		r.mu.Lock()
		r.stats.TotalRetries++
		r.mu.Unlock()
		r.logger.WithFields(logrus.Fields{
			"endpoint": endpoint,
			"attempt":  1,
			"delay":    task.delay,
		}).Info("Scheduled retry for failed metrics push")
	case <-ctx.Done():
		return
	default:
		r.logger.WithField("endpoint", endpoint).Warn("Retry queue is full, dropping retry task")
	}
}

// retryWorker воркер для обработки retry
func (r *RetryEngine) retryWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case task := <-r.retryQueue:
			// Ожидание задержки
			time.Sleep(task.delay)

			// Выполнение задачи
			task.attempts++
			r.logger.WithFields(logrus.Fields{
				"endpoint": task.endpoint,
				"attempt":  task.attempts,
				"max_attempts": r.config.MaxAttempts,
			}).Info("Retrying metrics push")
			
			err := task.fn()

			if err != nil {
				if task.attempts < r.config.MaxAttempts {
					// Вычисление новой задержки
					task.delay = r.calculateBackoff(task.attempts)
					// Повторное добавление в очередь
					select {
					case r.retryQueue <- task:
						r.logger.WithFields(logrus.Fields{
							"endpoint": task.endpoint,
							"attempt":  task.attempts,
							"next_attempt": task.attempts + 1,
							"next_delay": task.delay,
						}).Info("Scheduled next retry attempt")
					default:
						r.mu.Lock()
						r.stats.FailedRetries++
						r.mu.Unlock()
						r.logger.WithField("endpoint", task.endpoint).Warn("Retry queue is full, dropping retry task")
					}
				} else {
					r.mu.Lock()
					r.stats.FailedRetries++
					r.mu.Unlock()
					r.logger.WithError(err).WithFields(logrus.Fields{
						"endpoint": task.endpoint,
						"attempt":  task.attempts,
						"max_attempts": r.config.MaxAttempts,
					}).Error("Retry failed after max attempts")
				}
			} else {
				r.mu.Lock()
				r.stats.SuccessfulRetries++
				r.mu.Unlock()
				r.logger.WithFields(logrus.Fields{
					"endpoint": task.endpoint,
					"attempt":  task.attempts,
				}).Info("Retry succeeded")
			}
		}
	}
}

// calculateBackoff вычисляет задержку для backoff
func (r *RetryEngine) calculateBackoff(attempts int) time.Duration {
	var delay time.Duration

	switch r.config.Backoff {
	case "exponential":
		delay = time.Duration(float64(r.config.InitialDelay) * math.Pow(r.config.Multiplier, float64(attempts)))
	case "linear":
		delay = r.config.InitialDelay * time.Duration(attempts)
	case "fixed":
		delay = r.config.InitialDelay
	default:
		delay = r.config.InitialDelay
	}

	if delay > r.config.MaxDelay {
		delay = r.config.MaxDelay
	}

	return delay
}

