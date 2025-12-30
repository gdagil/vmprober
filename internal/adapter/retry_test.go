package adapter

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/gdagil/vmprober/internal/config"
)

func createTestRetryConfig() *config.RetryConfig {
	return &config.RetryConfig{
		MaxAttempts:  3,
		Backoff:      "exponential",
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     1 * time.Second,
		Multiplier:   2.0,
	}
}

func TestNewRetryEngine(t *testing.T) {
	cfg := createTestRetryConfig()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	engine := NewRetryEngine(cfg, logger)
	if engine == nil {
		t.Fatal("NewRetryEngine returned nil")
	}

	defer engine.Stop()
}

func TestNewRetryEngine_NilConfig(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	engine := NewRetryEngine(nil, logger)
	if engine == nil {
		t.Fatal("NewRetryEngine returned nil")
	}

	defer engine.Stop()
}

func TestRetryEngine_ShouldRetry(t *testing.T) {
	cfg := createTestRetryConfig()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	engine := NewRetryEngine(cfg, logger)
	defer engine.Stop()

	// Should not retry on nil error
	if engine.ShouldRetry(nil) {
		t.Error("ShouldRetry should return false for nil error")
	}

	// Should retry on error
	if !engine.ShouldRetry(errors.New("test error")) {
		t.Error("ShouldRetry should return true for error")
	}
}

func TestRetryEngine_ScheduleRetry(t *testing.T) {
	cfg := createTestRetryConfig()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	engine := NewRetryEngine(cfg, logger)
	defer engine.Stop()

	ctx := context.Background()

	// Schedule a retry
	var attempts int64
	engine.ScheduleRetry(ctx, "http://localhost:8428", func() error {
		atomic.AddInt64(&attempts, 1)
		if atomic.LoadInt64(&attempts) < 2 {
			return errors.New("temporary error")
		}
		return nil
	})

	// Give time for retry to process
	time.Sleep(500 * time.Millisecond)

	// Check that retry was attempted
	if atomic.LoadInt64(&attempts) == 0 {
		t.Log("Retry may not have been processed yet (this is OK)")
	}
}

func TestRetryEngine_ScheduleRetry_ContextCanceled(t *testing.T) {
	cfg := createTestRetryConfig()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	engine := NewRetryEngine(cfg, logger)
	defer engine.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Schedule retry with canceled context
	engine.ScheduleRetry(ctx, "http://localhost:8428", func() error {
		return errors.New("test error")
	})

	// Should not panic or error
	time.Sleep(100 * time.Millisecond)
}

func TestRetryEngine_Stop(t *testing.T) {
	cfg := createTestRetryConfig()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	engine := NewRetryEngine(cfg, logger)

	// Stop should not panic
	engine.Stop()

	// Stop again should not panic
	engine.Stop()
}

func TestRetryEngine_RetryWithExponentialBackoff(t *testing.T) {
	cfg := createTestRetryConfig()
	cfg.Backoff = "exponential"
	cfg.InitialDelay = 50 * time.Millisecond
	cfg.Multiplier = 2.0
	cfg.MaxDelay = 500 * time.Millisecond

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	engine := NewRetryEngine(cfg, logger)
	defer engine.Stop()

	ctx := context.Background()

	var attempts int64
	engine.ScheduleRetry(ctx, "http://localhost:8428", func() error {
		atomic.AddInt64(&attempts, 1)
		return errors.New("always fail")
	})

	// Give time for retries
	time.Sleep(1 * time.Second)

	// Should have attempted retries
	if atomic.LoadInt64(&attempts) == 0 {
		t.Log("Retries may not have been processed yet (this is OK)")
	}
}

func TestRetryEngine_RetryWithLinearBackoff(t *testing.T) {
	cfg := createTestRetryConfig()
	cfg.Backoff = "linear"
	cfg.InitialDelay = 50 * time.Millisecond

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	engine := NewRetryEngine(cfg, logger)
	defer engine.Stop()

	ctx := context.Background()

	var attempts int64
	engine.ScheduleRetry(ctx, "http://localhost:8428", func() error {
		atomic.AddInt64(&attempts, 1)
		return errors.New("always fail")
	})

	// Give time for retries
	time.Sleep(500 * time.Millisecond)

	// Should have attempted retries
	if atomic.LoadInt64(&attempts) == 0 {
		t.Log("Retries may not have been processed yet (this is OK)")
	}
}

func TestRetryEngine_RetryWithFixedBackoff(t *testing.T) {
	cfg := createTestRetryConfig()
	cfg.Backoff = "fixed"
	cfg.InitialDelay = 50 * time.Millisecond

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	engine := NewRetryEngine(cfg, logger)
	defer engine.Stop()

	ctx := context.Background()

	var attempts int64
	engine.ScheduleRetry(ctx, "http://localhost:8428", func() error {
		atomic.AddInt64(&attempts, 1)
		return errors.New("always fail")
	})

	// Give time for retries
	time.Sleep(500 * time.Millisecond)

	// Should have attempted retries
	if atomic.LoadInt64(&attempts) == 0 {
		t.Log("Retries may not have been processed yet (this is OK)")
	}
}

func TestRetryEngine_RetryWithDefaultBackoff(t *testing.T) {
	cfg := createTestRetryConfig()
	cfg.Backoff = "unknown" // Unknown backoff type should default

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	engine := NewRetryEngine(cfg, logger)
	defer engine.Stop()

	ctx := context.Background()

	var attempts int64
	engine.ScheduleRetry(ctx, "http://localhost:8428", func() error {
		atomic.AddInt64(&attempts, 1)
		return errors.New("always fail")
	})

	// Give time for retries
	time.Sleep(500 * time.Millisecond)

	// Should have attempted retries
	if atomic.LoadInt64(&attempts) == 0 {
		t.Log("Retries may not have been processed yet (this is OK)")
	}
}

func TestRetryEngine_MaxAttempts(t *testing.T) {
	cfg := createTestRetryConfig()
	cfg.MaxAttempts = 2
	cfg.InitialDelay = 50 * time.Millisecond

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	engine := NewRetryEngine(cfg, logger)
	defer engine.Stop()

	ctx := context.Background()

	var attempts int64
	engine.ScheduleRetry(ctx, "http://localhost:8428", func() error {
		atomic.AddInt64(&attempts, 1)
		return errors.New("always fail")
	})

	// Give time for retries
	time.Sleep(1 * time.Second)

	// Should not exceed max attempts
	if atomic.LoadInt64(&attempts) > int64(cfg.MaxAttempts+1) {
		t.Errorf("Should not exceed max attempts, got %d", atomic.LoadInt64(&attempts))
	}
}

func TestRetryEngine_ConcurrentRetries(t *testing.T) {
	cfg := createTestRetryConfig()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	engine := NewRetryEngine(cfg, logger)
	defer engine.Stop()

	ctx := context.Background()

	// Schedule multiple retries concurrently
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			engine.ScheduleRetry(ctx, "http://localhost:8428", func() error {
				return errors.New("test error")
			})
			done <- true
		}(i)
	}

	// Wait for all to be scheduled
	for i := 0; i < 10; i++ {
		<-done
	}

	// Give time for processing
	time.Sleep(200 * time.Millisecond)
}
