package wal

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vmprober/vmprober/internal/config"
	"github.com/vmprober/vmprober/internal/types"
)

// WALManager управляет WAL системой
type WALManager interface {
	// Write записывает запись в WAL
	Write(ctx context.Context, record *types.Record) error

	// Read читает записи из WAL
	Read(ctx context.Context, filter WALFilter) ([]*types.Record, error)

	// Flush принудительно синхронизирует все записи
	Flush(ctx context.Context) error

	// Rotate выполняет ротацию активного сегмента
	Rotate(ctx context.Context) error

	// Close закрывает WAL менеджер
	Close(ctx context.Context) error

	// GetStats возвращает статистику WAL
	GetStats() *WALStats
}

// WALFilter фильтр для чтения записей
type WALFilter struct {
	StartTime time.Time
	EndTime   time.Time
	Type      string
	SeriesID  string
	Limit     int
}

// WALStats статистика WAL
type WALStats struct {
	TotalRecords    int64         `json:"total_records"`
	TotalSize       int64         `json:"total_size"`
	SegmentCount    int           `json:"segment_count"`
	ActiveSegmentID string        `json:"active_segment_id"`
	LastWriteTime   time.Time     `json:"last_write_time"`
	LastRotateTime  time.Time     `json:"last_rotate_time"`
}

// DefaultWALManager реализация WAL менеджера
type DefaultWALManager struct {
	config        *config.WALConfig
	dir           string
	activeSegment *WALSegment
	segments      []*WALSegment
	mu            sync.RWMutex
	logger        *logrus.Logger
	stats         *WALStats
	compressor    Compressor
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	closed        bool
}

// NewWALManager создает новый WAL менеджер
func NewWALManager(cfg *config.WALConfig, logger *logrus.Logger) (WALManager, error) {
	if cfg == nil {
		return nil, fmt.Errorf("WAL config is nil")
	}

	dir := cfg.Dir
	if dir == "" {
		dir = "/var/lib/vmprober/wal"
	}

	// Создание директории если нужно
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create WAL directory: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	manager := &DefaultWALManager{
		config:     cfg,
		dir:        dir,
		segments:   make([]*WALSegment, 0),
		logger:     logger,
		stats:      &WALStats{},
		compressor: NewCompressor(cfg.Compression),
		ctx:        ctx,
		cancel:     cancel,
	}

	// Восстановление существующих сегментов
	if err := manager.recoverSegments(ctx); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to recover segments: %w", err)
	}

	// Создание активного сегмента
	if err := manager.createActiveSegment(ctx); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create active segment: %w", err)
	}

	// Запуск фоновых задач
	manager.wg.Add(2)
	go manager.rotationLoop()
	go manager.compactionLoop()

	return manager, nil
}

// Write записывает запись в WAL
func (w *DefaultWALManager) Write(ctx context.Context, record *types.Record) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.activeSegment == nil {
		if err := w.createActiveSegment(ctx); err != nil {
			return err
		}
	}

	// Запись в активный сегмент
	if err := w.activeSegment.Write(ctx, record); err != nil {
		return fmt.Errorf("failed to write to active segment: %w", err)
	}

	// Обновление статистики
	w.stats.TotalRecords++
	w.stats.LastWriteTime = time.Now()

	// Проверка необходимости ротации
	if w.shouldRotate() {
		go func() {
			if err := w.Rotate(w.ctx); err != nil {
				w.logger.WithError(err).Error("Failed to rotate segment")
			}
		}()
	}

	return nil
}

// Read читает записи из WAL
func (w *DefaultWALManager) Read(ctx context.Context, filter WALFilter) ([]*types.Record, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	var allRecords []*types.Record

	// Чтение из всех сегментов
	for _, segment := range w.segments {
		records, err := segment.Read(ctx, filter)
		if err != nil {
			w.logger.WithError(err).Warn("Failed to read from segment")
			continue
		}
		allRecords = append(allRecords, records...)
	}

	// Чтение из активного сегмента
	if w.activeSegment != nil {
		records, err := w.activeSegment.Read(ctx, filter)
		if err != nil {
			w.logger.WithError(err).Warn("Failed to read from active segment")
		} else {
			allRecords = append(allRecords, records...)
		}
	}

	// Применение лимита
	if filter.Limit > 0 && len(allRecords) > filter.Limit {
		allRecords = allRecords[:filter.Limit]
	}

	return allRecords, nil
}

// Flush принудительно синхронизирует все записи
func (w *DefaultWALManager) Flush(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.activeSegment != nil {
		if err := w.activeSegment.Flush(ctx); err != nil {
			return fmt.Errorf("failed to flush active segment: %w", err)
		}
	}

	return nil
}

// Rotate выполняет ротацию активного сегмента
func (w *DefaultWALManager) Rotate(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.activeSegment == nil {
		return nil
	}

	// Закрытие активного сегмента
	if err := w.activeSegment.Close(ctx); err != nil {
		return fmt.Errorf("failed to close active segment: %w", err)
	}

	// Добавление в список сегментов
	w.segments = append(w.segments, w.activeSegment)
	w.stats.SegmentCount++
	w.stats.LastRotateTime = time.Now()

	// Создание нового активного сегмента
	if err := w.createActiveSegment(ctx); err != nil {
		return fmt.Errorf("failed to create new active segment: %w", err)
	}

	return nil
}

// Close закрывает WAL менеджер
func (w *DefaultWALManager) Close(ctx context.Context) error {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return nil
	}
	w.closed = true
	w.mu.Unlock()

	// Остановка фоновых горутин
	if w.cancel != nil {
		w.cancel()
	}

	// Ожидание завершения фоновых горутин
	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Горутины завершились
	case <-ctx.Done():
		// Контекст отменен, но продолжаем закрытие
	case <-time.After(5 * time.Second):
		// Таймаут ожидания
		w.logger.Warn("Timeout waiting for background goroutines to finish")
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	// Закрытие активного сегмента
	if w.activeSegment != nil {
		if err := w.activeSegment.Close(ctx); err != nil {
			w.logger.WithError(err).Error("Failed to close active segment")
		}
	}

	// Закрытие всех сегментов
	for _, segment := range w.segments {
		if err := segment.Close(ctx); err != nil {
			w.logger.WithError(err).Error("Failed to close segment")
		}
	}

	return nil
}

// GetStats возвращает статистику WAL
func (w *DefaultWALManager) GetStats() *WALStats {
	w.mu.RLock()
	defer w.mu.RUnlock()

	stats := *w.stats
	if w.activeSegment != nil {
		stats.ActiveSegmentID = w.activeSegment.ID
		stats.TotalSize += w.activeSegment.Size()
	}

	for _, segment := range w.segments {
		stats.TotalSize += segment.Size()
	}

	return &stats
}

// shouldRotate проверяет нужно ли выполнить ротацию
func (w *DefaultWALManager) shouldRotate() bool {
	if w.activeSegment == nil {
		return true
	}

	// Проверка размера
	maxSize := parseSize(w.config.SegmentSize)
	if maxSize > 0 && w.activeSegment.Size() >= maxSize {
		return true
	}

	// Проверка возраста
	maxAge := w.config.MaxAge
	if maxAge > 0 && time.Since(w.activeSegment.CreatedAt()) >= maxAge {
		return true
	}

	return false
}

// createActiveSegment создает новый активный сегмент
func (w *DefaultWALManager) createActiveSegment(ctx context.Context) error {
	segmentID := fmt.Sprintf("segment-%d", time.Now().Unix())
	segmentPath := filepath.Join(w.dir, segmentID+".wal")

	segment, err := NewWALSegment(segmentID, segmentPath, w.config, w.compressor, w.logger)
	if err != nil {
		return fmt.Errorf("failed to create segment: %w", err)
	}

	w.activeSegment = segment
	w.stats.ActiveSegmentID = segmentID

	return nil
}

// recoverSegments восстанавливает существующие сегменты
func (w *DefaultWALManager) recoverSegments(ctx context.Context) error {
	entries, err := os.ReadDir(w.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if filepath.Ext(entry.Name()) != ".wal" {
			continue
		}

		segmentPath := filepath.Join(w.dir, entry.Name())
		segmentID := entry.Name()[:len(entry.Name())-4] // Убираем .wal

		segment, err := OpenWALSegment(segmentID, segmentPath, w.config, w.compressor, w.logger)
		if err != nil {
			w.logger.WithError(err).Warn("Failed to open segment", "segment", segmentID)
			continue
		}

		w.segments = append(w.segments, segment)
		w.stats.SegmentCount++
	}

	return nil
}

// rotationLoop цикл ротации сегментов
func (w *DefaultWALManager) rotationLoop() {
	defer w.wg.Done()
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			if w.shouldRotate() {
				if err := w.Rotate(w.ctx); err != nil {
					w.logger.WithError(err).Error("Failed to rotate segment in loop")
				}
			}
		}
	}
}

// compactionLoop цикл компрессии старых сегментов
func (w *DefaultWALManager) compactionLoop() {
	defer w.wg.Done()
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			w.compactOldSegments(w.ctx)
		}
	}
}

// compactOldSegments сжимает старые сегменты
func (w *DefaultWALManager) compactOldSegments(ctx context.Context) {
	w.mu.Lock()
	defer w.mu.Unlock()

	for _, segment := range w.segments {
		if segment.IsCompressed() {
			continue
		}

		// Сжатие сегментов старше 1 часа
		if time.Since(segment.CreatedAt()) > time.Hour {
			if err := segment.Compress(ctx); err != nil {
				w.logger.WithError(err).Warn("Failed to compress segment")
			}
		}
	}
}

// parseSize парсит размер из строки
func parseSize(s string) int64 {
	if s == "" {
		return 0
	}

	var size int64
	var unit string

	fmt.Sscanf(s, "%d%s", &size, &unit)

	switch unit {
	case "KB", "kb":
		return size * 1024
	case "MB", "mb":
		return size * 1024 * 1024
	case "GB", "gb":
		return size * 1024 * 1024 * 1024
	default:
		return size
	}
}

