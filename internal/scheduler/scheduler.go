package scheduler

import (
	"container/heap"
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vmprober/vmprober/internal/types"
)

// Scheduler управляет планированием задач
type Scheduler struct {
	jobs      *jobHeap
	allJobs   map[string]*types.Job // Хранилище всех джобов для UI
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	jobChan   chan *types.Job
	stats     *Stats
	logger    *logrus.Logger
}

// Stats статистика планировщика
type Stats struct {
	TotalJobs     int           `json:"total_jobs"`
	RunningJobs   int           `json:"running_jobs"`
	QueuedJobs    int           `json:"queued_jobs"`
	CompletedJobs int64         `json:"completed_jobs"`
	FailedJobs    int64         `json:"failed_jobs"`
	mu            sync.RWMutex  `json:"-"` // Не сериализуем mutex
}

// NewScheduler создает новый планировщик
func NewScheduler(logger *logrus.Logger) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	jh := &jobHeap{}
	heap.Init(jh)

	if logger == nil {
		logger = logrus.New()
		logger.SetLevel(logrus.InfoLevel)
	}

	return &Scheduler{
		jobs:    jh,
		allJobs: make(map[string]*types.Job),
		ctx:     ctx,
		cancel:  cancel,
		jobChan: make(chan *types.Job, 1000),
		stats:   &Stats{},
		logger:  logger,
	}
}

// Schedule добавляет задачу в планировщик
func (s *Scheduler) Schedule(ctx context.Context, job *types.Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Проверяем, существует ли джоб уже
	isNew := s.allJobs[job.ID] == nil
	
	s.logger.WithFields(logrus.Fields{
		"job_id":    job.ID,
		"target":    fmt.Sprintf("%s:%d", job.Target.Host, job.Target.Port),
		"protocol":  job.Target.Protocol,
		"next_run":  job.NextRun.Format(time.RFC3339),
		"interval":  job.Interval.String(),
		"is_new":    isNew,
	}).Debug("Scheduling job")
	
	// Сохраняем джоб в хранилище всех джобов
	s.allJobs[job.ID] = job
	
	heap.Push(s.jobs, job)
	s.stats.mu.Lock()
	if isNew {
		s.stats.TotalJobs++
		s.logger.WithField("job_id", job.ID).Debug("New job added to scheduler")
	}
	s.stats.QueuedJobs++
	queuedCount := s.stats.QueuedJobs
	s.stats.mu.Unlock()

	s.logger.WithFields(logrus.Fields{
		"job_id":      job.ID,
		"queued_jobs": queuedCount,
		"total_jobs":  s.stats.TotalJobs,
	}).Debug("Job scheduled successfully")

	return nil
}

// Reschedule перепланирует существующую задачу без увеличения счетчика TotalJobs
func (s *Scheduler) Reschedule(ctx context.Context, job *types.Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Обновляем джоб в хранилище
	if existingJob, exists := s.allJobs[job.ID]; exists {
		s.logger.WithFields(logrus.Fields{
			"job_id":    job.ID,
			"old_next": existingJob.NextRun.Format(time.RFC3339),
			"new_next": job.NextRun.Format(time.RFC3339),
			"interval":  job.Interval.String(),
		}).Debug("Rescheduling existing job")
		
		existingJob.NextRun = job.NextRun
		existingJob.Interval = job.Interval
		// Используем существующий джоб для heap
		job = existingJob
	} else {
		s.logger.WithFields(logrus.Fields{
			"job_id":   job.ID,
			"next_run": job.NextRun.Format(time.RFC3339),
		}).Debug("Rescheduling non-existent job, adding as new")
		
		// Если джоба нет, добавляем его как новый
		s.allJobs[job.ID] = job
		s.stats.mu.Lock()
		s.stats.TotalJobs++
		s.stats.mu.Unlock()
	}
	
	heap.Push(s.jobs, job)
	s.stats.mu.Lock()
	s.stats.QueuedJobs++
	queuedCount := s.stats.QueuedJobs
	s.stats.mu.Unlock()

	s.logger.WithFields(logrus.Fields{
		"job_id":      job.ID,
		"queued_jobs": queuedCount,
	}).Debug("Job rescheduled successfully")

	return nil
}

// Start запускает планировщик
func (s *Scheduler) Start(ctx context.Context) error {
	s.logger.Debug("Starting scheduler")
	s.wg.Add(1)
	go s.run(ctx)
	return nil
}

// Stop останавливает планировщик
func (s *Scheduler) Stop(ctx context.Context) error {
	s.logger.Debug("Stopping scheduler")
	s.cancel()
	s.wg.Wait()
	s.logger.Debug("Scheduler stopped")
	return nil
}

// GetJobChan возвращает канал для получения задач
func (s *Scheduler) GetJobChan() <-chan *types.Job {
	return s.jobChan
}

// GetStats возвращает статистику
func (s *Scheduler) GetStats() Stats {
	s.mu.RLock()
	// Пересчитываем FailedJobs на основе текущего состояния джобов
	// FailedJobs = количество джобов со статусом "down"
	failedJobsCount := int64(0)
	for _, job := range s.allJobs {
		if job.LastStatus == "down" {
			failedJobsCount++
		}
	}
	s.mu.RUnlock()
	
	s.stats.mu.Lock()
	s.stats.FailedJobs = failedJobsCount
	statsCopy := *s.stats
	s.stats.mu.Unlock()
	
	return statsCopy
}

// ExportMetrics экспортирует метрики шедулера в формат types.Metric
func (s *Scheduler) ExportMetrics() []types.Metric {
	stats := s.GetStats()
	now := time.Now()
	
	metrics := make([]types.Metric, 0, 3)
	
	// Общие метрики шедулера
	metrics = append(metrics, types.Metric{
		Name:      "vmprober_scheduler_jobs_total",
		Value:     float64(stats.TotalJobs),
		Timestamp: now,
		Labels:    make(map[string]string),
		Type:      types.MetricTypeGauge,
	})
	
	metrics = append(metrics, types.Metric{
		Name:      "vmprober_scheduler_jobs_running",
		Value:     float64(stats.RunningJobs),
		Timestamp: now,
		Labels:    make(map[string]string),
		Type:      types.MetricTypeGauge,
	})
	
	metrics = append(metrics, types.Metric{
		Name:      "vmprober_scheduler_jobs_failed",
		Value:     float64(stats.FailedJobs),
		Timestamp: now,
		Labels:    make(map[string]string),
		Type:      types.MetricTypeGauge,
	})
	
	return metrics
}

// GetAllJobs возвращает список всех джобов, отсортированный по ID для стабильного порядка
func (s *Scheduler) GetAllJobs() []*types.Job {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	jobs := make([]*types.Job, 0, len(s.allJobs))
	for _, job := range s.allJobs {
		// Создаем копию джоба для безопасности
		jobCopy := *job
		jobs = append(jobs, &jobCopy)
	}
	
	// Сортируем по ID для стабильного порядка
	sort.Slice(jobs, func(i, j int) bool {
		return jobs[i].ID < jobs[j].ID
	})
	
	return jobs
}

// MarkJobStarted отмечает начало выполнения джоба (вызывается когда джоб начинает выполняться)
func (s *Scheduler) MarkJobStarted(jobID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if _, exists := s.allJobs[jobID]; exists {
		s.stats.mu.Lock()
		s.stats.RunningJobs++
		runningCount := s.stats.RunningJobs
		s.stats.mu.Unlock()
		
		s.logger.WithFields(logrus.Fields{
			"job_id":    jobID,
			"running":   runningCount,
		}).Debug("Job started execution")
	} else {
		s.logger.WithField("job_id", jobID).Warn("MarkJobStarted called for non-existent job")
	}
}

// MarkJobCompleted отмечает джоб как завершенный (успешная проба)
func (s *Scheduler) MarkJobCompleted(jobID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Обновляем статистику джоба
	if job, exists := s.allJobs[jobID]; exists {
		job.SuccessCount++
		job.LastStatus = "up"
		job.LastProbeTime = time.Now()
		
		s.stats.mu.Lock()
		runningBefore := s.stats.RunningJobs
		s.stats.RunningJobs--
		if s.stats.RunningJobs < 0 {
			s.stats.RunningJobs = 0
		}
		s.stats.CompletedJobs++
		completedCount := s.stats.CompletedJobs
		runningCount := s.stats.RunningJobs
		s.stats.mu.Unlock()
		
		s.logger.WithFields(logrus.Fields{
			"job_id":         jobID,
			"success_count":  job.SuccessCount,
			"failed_count":   job.FailedCount,
			"running_before": runningBefore,
			"running_now":    runningCount,
			"completed":      completedCount,
		}).Debug("Job completed successfully")
	} else {
		s.logger.WithField("job_id", jobID).Warn("MarkJobCompleted called for non-existent job")
	}
}

// MarkJobFailed отмечает джоб как проваленный (неуспешная проба)
func (s *Scheduler) MarkJobFailed(jobID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Обновляем статистику джоба
	if job, exists := s.allJobs[jobID]; exists {
		job.FailedCount++
		job.LastStatus = "down"
		job.LastProbeTime = time.Now()
		
		s.stats.mu.Lock()
		runningBefore := s.stats.RunningJobs
		s.stats.RunningJobs--
		if s.stats.RunningJobs < 0 {
			s.stats.RunningJobs = 0
		}
		// Не увеличиваем FailedJobs здесь - он будет пересчитан в GetStats()
		// на основе LastStatus джобов
		runningCount := s.stats.RunningJobs
		s.stats.mu.Unlock()
		
		s.logger.WithFields(logrus.Fields{
			"job_id":        jobID,
			"success_count": job.SuccessCount,
			"failed_count":  job.FailedCount,
			"running_before": runningBefore,
			"running_now":    runningCount,
			"last_status":    job.LastStatus,
		}).Debug("Job failed")
	} else {
		s.logger.WithField("job_id", jobID).Warn("MarkJobFailed called for non-existent job")
	}
}

// run основной цикл планировщика
func (s *Scheduler) run(ctx context.Context) {
	defer s.wg.Done()

	s.logger.Debug("Scheduler run loop started")
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Debug("Scheduler run loop stopped (context cancelled)")
			return
		case <-s.ctx.Done():
			s.logger.Debug("Scheduler run loop stopped (scheduler context cancelled)")
			return
		case <-ticker.C:
			s.processJobs()
		}
	}
}

// processJobs обрабатывает готовые к выполнению задачи
func (s *Scheduler) processJobs() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	processedCount := 0
	
	for s.jobs.Len() > 0 {
		job := s.jobs.Peek().(*types.Job)
		if job.NextRun.After(now) {
			if processedCount > 0 {
				s.logger.WithFields(logrus.Fields{
					"processed": processedCount,
					"remaining": s.jobs.Len(),
					"next_run":  job.NextRun.Format(time.RFC3339),
				}).Debug("Processed jobs, waiting for next batch")
			}
			break
		}

		job = heap.Pop(s.jobs).(*types.Job)
		// Обновляем джоб в хранилище перед отправкой
		if storedJob, exists := s.allJobs[job.ID]; exists {
			storedJob.NextRun = job.NextRun
		}
		
		select {
		case s.jobChan <- job:
			s.stats.mu.Lock()
			s.stats.QueuedJobs--
			queuedCount := s.stats.QueuedJobs
			s.stats.mu.Unlock()
			
			processedCount++
			s.logger.WithFields(logrus.Fields{
				"job_id":      job.ID,
				"target":     fmt.Sprintf("%s:%d", job.Target.Host, job.Target.Port),
				"protocol":   job.Target.Protocol,
				"queued":     queuedCount,
			}).Debug("Job sent to execution channel")
		default:
			// Канал заполнен, возвращаем задачу обратно
			s.logger.WithFields(logrus.Fields{
				"job_id": job.ID,
				"channel_size": len(s.jobChan),
			}).Warn("Job channel full, returning job to queue")
			heap.Push(s.jobs, job)
			break
		}
	}
}

// jobHeap реализует heap.Interface для приоритетной очереди
type jobHeap struct {
	jobs []*types.Job
}

func (h jobHeap) Len() int { return len(h.jobs) }

func (h jobHeap) Less(i, j int) bool {
	return h.jobs[i].NextRun.Before(h.jobs[j].NextRun)
}

func (h jobHeap) Swap(i, j int) {
	h.jobs[i], h.jobs[j] = h.jobs[j], h.jobs[i]
}

func (h *jobHeap) Push(x interface{}) {
	h.jobs = append(h.jobs, x.(*types.Job))
}

func (h *jobHeap) Pop() interface{} {
	old := h.jobs
	n := len(old)
	job := old[n-1]
	h.jobs = old[0 : n-1]
	return job
}

func (h *jobHeap) Peek() interface{} {
	if len(h.jobs) == 0 {
		return nil
	}
	return h.jobs[0]
}

