package scheduler

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/gdagil/vmprober/internal/types"
)

func TestNewScheduler(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Disable logs in tests
	scheduler := NewScheduler(logger)
	if scheduler == nil {
		t.Fatal("NewScheduler returned nil")
	}

	stats := scheduler.GetStats()
	if stats.TotalJobs != 0 {
		t.Errorf("Expected 0 total jobs, got %d", stats.TotalJobs)
	}
}

func TestScheduler_Schedule(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	scheduler := NewScheduler(logger)

	job := &types.Job{
		ID:       "test-job-1",
		NextRun:  time.Now().Add(1 * time.Second),
		Interval: 30 * time.Second,
		Priority: 1,
		Target: types.Target{
			ID:       "test-target",
			Host:     "127.0.0.1",
			Port:     80,
			Protocol: types.ProbeTypeTCP,
		},
		CreatedAt: time.Now(),
	}

	ctx := context.Background()
	err := scheduler.Schedule(ctx, job)
	if err != nil {
		t.Fatalf("Schedule failed: %v", err)
	}

	stats := scheduler.GetStats()
	if stats.TotalJobs != 1 {
		t.Errorf("Expected 1 total job, got %d", stats.TotalJobs)
	}
	if stats.QueuedJobs != 1 {
		t.Errorf("Expected 1 queued job, got %d", stats.QueuedJobs)
	}
}

func TestScheduler_Schedule_MultipleJobs(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	scheduler := NewScheduler(logger)

	ctx := context.Background()
	for i := 0; i < 5; i++ {
		job := &types.Job{
			ID:       fmt.Sprintf("test-job-%d", i),
			NextRun:  time.Now().Add(time.Duration(i) * time.Second),
			Interval: 30 * time.Second,
			Priority: 1,
			Target: types.Target{
				ID:       "test-target",
				Host:     "127.0.0.1",
				Port:     80,
				Protocol: types.ProbeTypeTCP,
			},
			CreatedAt: time.Now(),
		}

		if err := scheduler.Schedule(ctx, job); err != nil {
			t.Fatalf("Schedule failed: %v", err)
		}
	}

	stats := scheduler.GetStats()
	if stats.TotalJobs != 5 {
		t.Errorf("Expected 5 total jobs, got %d", stats.TotalJobs)
	}
	if stats.QueuedJobs != 5 {
		t.Errorf("Expected 5 queued jobs, got %d", stats.QueuedJobs)
	}
}

func TestScheduler_StartStop(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	scheduler := NewScheduler(logger)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := scheduler.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Give time for scheduler to start
	time.Sleep(100 * time.Millisecond)

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer stopCancel()

	err = scheduler.Stop(stopCtx)
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}

func TestScheduler_GetJobChan(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	scheduler := NewScheduler(logger)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := scheduler.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer scheduler.Stop(context.Background())

	// Schedule a task that should execute immediately
	job := &types.Job{
		ID:       "test-job-immediate",
		NextRun:  time.Now().Add(-1 * time.Second), // In the past
		Interval: 30 * time.Second,
		Priority: 1,
		Target: types.Target{
			ID:       "test-target",
			Host:     "127.0.0.1",
			Port:     80,
			Protocol: types.ProbeTypeTCP,
		},
		CreatedAt: time.Now(),
	}

	if err := scheduler.Schedule(ctx, job); err != nil {
		t.Fatalf("Schedule failed: %v", err)
	}

	// Wait for task to appear in channel
	jobChan := scheduler.GetJobChan()
	select {
	case receivedJob := <-jobChan:
		if receivedJob.ID != job.ID {
			t.Errorf("Expected job ID %s, got %s", job.ID, receivedJob.ID)
		}
	case <-time.After(2 * time.Second):
		t.Error("Timeout waiting for job")
	}
}

func TestScheduler_ProcessJobs_Priority(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	scheduler := NewScheduler(logger)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := scheduler.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer scheduler.Stop(context.Background())

	// Schedule several tasks with different execution times
	now := time.Now()
	jobs := []*types.Job{
		{
			ID:       "job-1",
			NextRun:  now.Add(2 * time.Second),
			Interval: 30 * time.Second,
			Priority: 1,
			Target: types.Target{
				ID: "target-1",
			},
			CreatedAt: time.Now(),
		},
		{
			ID:       "job-2",
			NextRun:  now.Add(1 * time.Second),
			Interval: 30 * time.Second,
			Priority: 1,
			Target: types.Target{
				ID: "target-2",
			},
			CreatedAt: time.Now(),
		},
		{
			ID:       "job-3",
			NextRun:  now.Add(-1 * time.Second), // Should execute first
			Interval: 30 * time.Second,
			Priority: 1,
			Target: types.Target{
				ID: "target-3",
			},
			CreatedAt: time.Now(),
		},
	}

	for _, job := range jobs {
		if err := scheduler.Schedule(ctx, job); err != nil {
			t.Fatalf("Schedule failed: %v", err)
		}
	}

	// First task should be job-3 (earliest)
	jobChan := scheduler.GetJobChan()
	select {
	case firstJob := <-jobChan:
		if firstJob.ID != "job-3" {
			t.Errorf("Expected first job to be job-3, got %s", firstJob.ID)
		}
	case <-time.After(2 * time.Second):
		t.Error("Timeout waiting for first job")
	}
}

func TestScheduler_GetStats(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	scheduler := NewScheduler(logger)

	stats := scheduler.GetStats()
	if stats.TotalJobs != 0 {
		t.Errorf("Expected 0 total jobs initially, got %d", stats.TotalJobs)
	}

	ctx := context.Background()
	job := &types.Job{
		ID:       "test-job",
		NextRun:  time.Now(),
		Interval: 30 * time.Second,
		Priority: 1,
		Target: types.Target{
			ID: "test-target",
		},
		CreatedAt: time.Now(),
	}

	scheduler.Schedule(ctx, job)

	stats = scheduler.GetStats()
	if stats.TotalJobs != 1 {
		t.Errorf("Expected 1 total job, got %d", stats.TotalJobs)
	}
	if stats.QueuedJobs != 1 {
		t.Errorf("Expected 1 queued job, got %d", stats.QueuedJobs)
	}
}
