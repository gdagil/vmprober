package scheduler

import (
	"container/heap"
	"context"
	"sort"
	"sync"
	"time"

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
}

// Stats статистика планировщика
type Stats struct {
	TotalJobs     int           `json:"total_jobs"`
	RunningJobs   int           `json:"running_jobs"`
	QueuedJobs    int           `json:"queued_jobs"`
	CompletedJobs int64         `json:"completed_jobs"`
	FailedJobs    int64         `json:"failed_jobs"`
	mu            sync.RWMutex
}

// NewScheduler создает новый планировщик
func NewScheduler() *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	jh := &jobHeap{}
	heap.Init(jh)

	return &Scheduler{
		jobs:    jh,
		allJobs: make(map[string]*types.Job),
		ctx:     ctx,
		cancel:  cancel,
		jobChan: make(chan *types.Job, 1000),
		stats:   &Stats{},
	}
}

// Schedule добавляет задачу в планировщик
func (s *Scheduler) Schedule(ctx context.Context, job *types.Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Сохраняем джоб в хранилище всех джобов
	s.allJobs[job.ID] = job
	
	heap.Push(s.jobs, job)
	s.stats.mu.Lock()
	s.stats.TotalJobs++
	s.stats.QueuedJobs++
	s.stats.mu.Unlock()

	return nil
}

// Start запускает планировщик
func (s *Scheduler) Start(ctx context.Context) error {
	s.wg.Add(1)
	go s.run(ctx)
	return nil
}

// Stop останавливает планировщик
func (s *Scheduler) Stop(ctx context.Context) error {
	s.cancel()
	s.wg.Wait()
	return nil
}

// GetJobChan возвращает канал для получения задач
func (s *Scheduler) GetJobChan() <-chan *types.Job {
	return s.jobChan
}

// GetStats возвращает статистику
func (s *Scheduler) GetStats() Stats {
	s.stats.mu.RLock()
	defer s.stats.mu.RUnlock()
	return *s.stats
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

// MarkJobCompleted отмечает джоб как завершенный
func (s *Scheduler) MarkJobCompleted() {
	s.stats.mu.Lock()
	defer s.stats.mu.Unlock()
	s.stats.RunningJobs--
	if s.stats.RunningJobs < 0 {
		s.stats.RunningJobs = 0
	}
	s.stats.CompletedJobs++
}

// MarkJobFailed отмечает джоб как проваленный
func (s *Scheduler) MarkJobFailed() {
	s.stats.mu.Lock()
	defer s.stats.mu.Unlock()
	s.stats.RunningJobs--
	if s.stats.RunningJobs < 0 {
		s.stats.RunningJobs = 0
	}
	s.stats.FailedJobs++
}

// run основной цикл планировщика
func (s *Scheduler) run(ctx context.Context) {
	defer s.wg.Done()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.ctx.Done():
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
	for s.jobs.Len() > 0 {
		job := s.jobs.Peek().(*types.Job)
		if job.NextRun.After(now) {
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
			s.stats.RunningJobs++
			s.stats.mu.Unlock()
		default:
			// Канал заполнен, возвращаем задачу обратно
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

