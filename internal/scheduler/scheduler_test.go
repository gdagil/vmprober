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

func TestScheduler_Reschedule(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	scheduler := NewScheduler(logger)

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

	// Schedule first time
	if err := scheduler.Schedule(ctx, job); err != nil {
		t.Fatalf("Schedule failed: %v", err)
	}

	stats := scheduler.GetStats()
	if stats.TotalJobs != 1 {
		t.Errorf("Expected 1 total job, got %d", stats.TotalJobs)
	}

	// Reschedule with new NextRun
	job.NextRun = time.Now().Add(1 * time.Minute)
	if err := scheduler.Reschedule(ctx, job); err != nil {
		t.Fatalf("Reschedule failed: %v", err)
	}

	// TotalJobs should not increase
	stats = scheduler.GetStats()
	if stats.TotalJobs != 1 {
		t.Errorf("Expected 1 total job after reschedule, got %d", stats.TotalJobs)
	}
}

func TestScheduler_Reschedule_NewJob(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	scheduler := NewScheduler(logger)

	ctx := context.Background()

	job := &types.Job{
		ID:       "test-job-new",
		NextRun:  time.Now(),
		Interval: 30 * time.Second,
		Priority: 1,
		Target: types.Target{
			ID: "test-target",
		},
		CreatedAt: time.Now(),
	}

	// Reschedule non-existent job (should add as new)
	if err := scheduler.Reschedule(ctx, job); err != nil {
		t.Fatalf("Reschedule failed: %v", err)
	}

	stats := scheduler.GetStats()
	if stats.TotalJobs != 1 {
		t.Errorf("Expected 1 total job, got %d", stats.TotalJobs)
	}
}

func TestScheduler_MarkJobStarted(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	scheduler := NewScheduler(logger)

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

	if err := scheduler.Schedule(ctx, job); err != nil {
		t.Fatalf("Schedule failed: %v", err)
	}

	// Mark job as started
	scheduler.MarkJobStarted(job.ID)

	stats := scheduler.GetStats()
	if stats.RunningJobs != 1 {
		t.Errorf("Expected 1 running job, got %d", stats.RunningJobs)
	}
}

func TestScheduler_MarkJobStarted_NonExistent(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	scheduler := NewScheduler(logger)

	// Mark non-existent job as started (should not panic)
	scheduler.MarkJobStarted("non-existent-job")

	stats := scheduler.GetStats()
	if stats.RunningJobs != 0 {
		t.Errorf("Expected 0 running jobs, got %d", stats.RunningJobs)
	}
}

func TestScheduler_MarkJobCompleted(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	scheduler := NewScheduler(logger)

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

	if err := scheduler.Schedule(ctx, job); err != nil {
		t.Fatalf("Schedule failed: %v", err)
	}

	scheduler.MarkJobStarted(job.ID)
	scheduler.MarkJobCompleted(job.ID)

	stats := scheduler.GetStats()
	if stats.RunningJobs != 0 {
		t.Errorf("Expected 0 running jobs, got %d", stats.RunningJobs)
	}
	if stats.CompletedJobs != 1 {
		t.Errorf("Expected 1 completed job, got %d", stats.CompletedJobs)
	}

	// Check job status
	jobs := scheduler.GetAllJobs()
	if len(jobs) != 1 {
		t.Fatalf("Expected 1 job, got %d", len(jobs))
	}
	if jobs[0].SuccessCount != 1 {
		t.Errorf("Expected success count 1, got %d", jobs[0].SuccessCount)
	}
	if jobs[0].LastStatus != "up" {
		t.Errorf("Expected last status 'up', got '%s'", jobs[0].LastStatus)
	}
}

func TestScheduler_MarkJobFailed(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	scheduler := NewScheduler(logger)

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

	if err := scheduler.Schedule(ctx, job); err != nil {
		t.Fatalf("Schedule failed: %v", err)
	}

	scheduler.MarkJobStarted(job.ID)
	scheduler.MarkJobFailed(job.ID)

	stats := scheduler.GetStats()
	if stats.RunningJobs != 0 {
		t.Errorf("Expected 0 running jobs, got %d", stats.RunningJobs)
	}

	// Check job status
	jobs := scheduler.GetAllJobs()
	if len(jobs) != 1 {
		t.Fatalf("Expected 1 job, got %d", len(jobs))
	}
	if jobs[0].FailedCount != 1 {
		t.Errorf("Expected failed count 1, got %d", jobs[0].FailedCount)
	}
	if jobs[0].LastStatus != "down" {
		t.Errorf("Expected last status 'down', got '%s'", jobs[0].LastStatus)
	}
}

func TestScheduler_MarkJobCompleted_NonExistent(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	scheduler := NewScheduler(logger)

	// Mark non-existent job as completed (should not panic)
	scheduler.MarkJobCompleted("non-existent-job")

	stats := scheduler.GetStats()
	if stats.CompletedJobs != 0 {
		t.Errorf("Expected 0 completed jobs, got %d", stats.CompletedJobs)
	}
}

func TestScheduler_MarkJobFailed_NonExistent(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	scheduler := NewScheduler(logger)

	// Mark non-existent job as failed (should not panic)
	scheduler.MarkJobFailed("non-existent-job")

	stats := scheduler.GetStats()
	if stats.FailedJobs != 0 {
		t.Errorf("Expected 0 failed jobs, got %d", stats.FailedJobs)
	}
}

func TestScheduler_GetAllJobs(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	scheduler := NewScheduler(logger)

	ctx := context.Background()

	// Schedule multiple jobs
	for i := 0; i < 5; i++ {
		job := &types.Job{
			ID:       fmt.Sprintf("test-job-%d", i),
			NextRun:  time.Now(),
			Interval: 30 * time.Second,
			Priority: 1,
			Target: types.Target{
				ID: fmt.Sprintf("test-target-%d", i),
			},
			CreatedAt: time.Now(),
		}
		if err := scheduler.Schedule(ctx, job); err != nil {
			t.Fatalf("Schedule failed: %v", err)
		}
	}

	jobs := scheduler.GetAllJobs()
	if len(jobs) != 5 {
		t.Errorf("Expected 5 jobs, got %d", len(jobs))
	}

	// Jobs should be sorted by ID
	for i := 1; i < len(jobs); i++ {
		if jobs[i].ID < jobs[i-1].ID {
			t.Errorf("Jobs are not sorted by ID")
		}
	}
}

func TestScheduler_ExportMetrics(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	scheduler := NewScheduler(logger)

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

	if err := scheduler.Schedule(ctx, job); err != nil {
		t.Fatalf("Schedule failed: %v", err)
	}

	metrics := scheduler.ExportMetrics()
	if len(metrics) == 0 {
		t.Error("Expected at least one metric")
	}

	// Check for expected metric names
	metricNames := make(map[string]bool)
	for _, metric := range metrics {
		metricNames[metric.Name] = true
	}

	expectedMetrics := []string{
		"vmprober_scheduler_jobs_total",
		"vmprober_scheduler_jobs_running",
		"vmprober_scheduler_jobs_failed",
	}

	for _, expected := range expectedMetrics {
		if !metricNames[expected] {
			t.Errorf("Expected metric %s not found", expected)
		}
	}
}

func TestScheduler_GetStats_FailedJobs(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	scheduler := NewScheduler(logger)

	ctx := context.Background()

	// Create jobs with different statuses
	job1 := &types.Job{
		ID:       "job-up",
		NextRun:  time.Now(),
		Interval: 30 * time.Second,
		Priority: 1,
		Target: types.Target{
			ID: "target-1",
		},
		CreatedAt: time.Now(),
	}

	job2 := &types.Job{
		ID:       "job-down",
		NextRun:  time.Now(),
		Interval: 30 * time.Second,
		Priority: 1,
		Target: types.Target{
			ID: "target-2",
		},
		CreatedAt: time.Now(),
	}

	if err := scheduler.Schedule(ctx, job1); err != nil {
		t.Fatalf("Schedule failed: %v", err)
	}
	if err := scheduler.Schedule(ctx, job2); err != nil {
		t.Fatalf("Schedule failed: %v", err)
	}

	// Mark one as up, one as down
	scheduler.MarkJobStarted(job1.ID)
	scheduler.MarkJobCompleted(job1.ID)

	scheduler.MarkJobStarted(job2.ID)
	scheduler.MarkJobFailed(job2.ID)

	stats := scheduler.GetStats()
	if stats.FailedJobs != 1 {
		t.Errorf("Expected 1 failed job, got %d", stats.FailedJobs)
	}
}
