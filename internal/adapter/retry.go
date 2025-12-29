package adapter

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/gdagil/vmprober/internal/config"
)

// RetryEngine manages retry logic
type RetryEngine struct {
	config     *config.RetryConfig
	logger     *logrus.Logger
	retryQueue chan retryTask
	mu         sync.RWMutex
	stats      *RetryStats
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
}

// RetryStats represents retry statistics
type RetryStats struct {
	TotalRetries      int64 `json:"total_retries"`
	SuccessfulRetries int64 `json:"successful_retries"`
	FailedRetries     int64 `json:"failed_retries"`
}

// retryTask represents a retry task
type retryTask struct {
	fn       func() error
	attempts int
	delay    time.Duration
	endpoint string
}

// NewRetryEngine creates a new retry engine
func NewRetryEngine(cfg *config.RetryConfig, logger *logrus.Logger) *RetryEngine {
	if cfg == nil {
		cfg = &config.RetryConfig{
			MaxAttempts:  3,
			Backoff:      "exponential",
			InitialDelay: 1 * time.Second,
			MaxDelay:     30 * time.Second,
			Multiplier:   2.0,
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	engine := &RetryEngine{
		config:     cfg,
		logger:     logger,
		retryQueue: make(chan retryTask, 100),
		stats:      &RetryStats{},
		ctx:        ctx,
		cancel:     cancel,
	}

	// Start worker for retry processing
	engine.wg.Add(1)
	go engine.retryWorker()

	return engine
}

// Stop stops the retry engine
func (r *RetryEngine) Stop() {
	r.cancel()
	r.wg.Wait()
	r.logger.Info("RetryEngine stopped")
}

// ShouldRetry checks if retry should be performed
func (r *RetryEngine) ShouldRetry(err error) bool {
	if err == nil {
		return false
	}

	// Check error type
	// In a real implementation, there would be more complex logic here
	return true
}

// ScheduleRetry schedules a retry
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

// retryWorker is a worker for retry processing
func (r *RetryEngine) retryWorker() {
	defer r.wg.Done()

	for {
		select {
		case <-r.ctx.Done():
			// Process remaining tasks in queue before exit
			for {
				select {
				case task := <-r.retryQueue:
					r.logger.WithField("endpoint", task.endpoint).Warn("Dropping retry task during shutdown")
				default:
					return
				}
			}
		case task, ok := <-r.retryQueue:
			if !ok {
				return
			}

			// Wait for delay with context awareness
			select {
			case <-r.ctx.Done():
				return
			case <-time.After(task.delay):
			}

			// Execute task
			task.attempts++
			r.logger.WithFields(logrus.Fields{
				"endpoint":     task.endpoint,
				"attempt":      task.attempts,
				"max_attempts": r.config.MaxAttempts,
			}).Info("Retrying metrics push")

			err := task.fn()

			if err != nil {
				if task.attempts < r.config.MaxAttempts {
					// Calculate new delay
					task.delay = r.calculateBackoff(task.attempts)
					// Re-add to queue
					select {
					case r.retryQueue <- task:
						r.logger.WithFields(logrus.Fields{
							"endpoint":     task.endpoint,
							"attempt":      task.attempts,
							"next_attempt": task.attempts + 1,
							"next_delay":   task.delay,
						}).Info("Scheduled next retry attempt")
					case <-r.ctx.Done():
						return
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
						"endpoint":     task.endpoint,
						"attempt":      task.attempts,
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

// calculateBackoff calculates delay for backoff
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
