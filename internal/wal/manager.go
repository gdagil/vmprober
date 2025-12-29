package wal

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/gdagil/vmprober/internal/config"
	"github.com/gdagil/vmprober/internal/types"
)

// WALManager manages the WAL system
type WALManager interface {
	// Write writes a record to WAL
	Write(ctx context.Context, record *types.Record) error

	// Read reads records from WAL
	Read(ctx context.Context, filter WALFilter) ([]*types.Record, error)

	// Flush forcefully syncs all records
	Flush(ctx context.Context) error

	// Rotate performs rotation of the active segment
	Rotate(ctx context.Context) error

	// Close closes the WAL manager
	Close(ctx context.Context) error

	// GetStats returns WAL statistics
	GetStats() *WALStats

	// MarkSent marks a record as sent
	MarkSent(ctx context.Context, recordID string) error

	// GetUnsentRecords returns all unsent records
	GetUnsentRecords(ctx context.Context) ([]*types.Record, error)

	// DeleteSentRecords deletes all sent records older than given time
	DeleteSentRecords(ctx context.Context, olderThan time.Duration) error
}

// WALFilter is a filter for reading records
type WALFilter struct {
	StartTime time.Time
	EndTime   time.Time
	Type      string
	SeriesID  string
	Limit     int
}

// WALStats represents WAL statistics
type WALStats struct {
	TotalRecords    int64     `json:"total_records"`
	TotalSize       int64     `json:"total_size"`
	SegmentCount    int       `json:"segment_count"`
	ActiveSegmentID string    `json:"active_segment_id"`
	LastWriteTime   time.Time `json:"last_write_time"`
	LastRotateTime  time.Time `json:"last_rotate_time"`
}

// DefaultWALManager is the WAL manager implementation
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

// NewWALManager creates a new WAL manager
func NewWALManager(cfg *config.WALConfig, logger *logrus.Logger) (WALManager, error) {
	if cfg == nil {
		return nil, fmt.Errorf("WAL config is nil")
	}

	dir := cfg.Dir
	if dir == "" {
		dir = "/var/lib/vmprober/wal"
	}

	// Create directory if needed
	if err := os.MkdirAll(dir, 0o755); err != nil {
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

	// Recover existing segments
	if err := manager.recoverSegments(ctx); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to recover segments: %w", err)
	}

	// Create active segment
	if err := manager.createActiveSegment(ctx); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create active segment: %w", err)
	}

	// Start background tasks
	manager.wg.Add(2)
	go manager.rotationLoop()
	go manager.compactionLoop()

	return manager, nil
}

// Write writes a record to WAL
func (w *DefaultWALManager) Write(ctx context.Context, record *types.Record) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.activeSegment == nil {
		if err := w.createActiveSegment(ctx); err != nil {
			return err
		}
	}

	// Write to active segment
	if err := w.activeSegment.Write(ctx, record); err != nil {
		return fmt.Errorf("failed to write to active segment: %w", err)
	}

	// Update statistics
	w.stats.TotalRecords++
	w.stats.LastWriteTime = time.Now()

	// Check if rotation is needed
	if w.shouldRotate() {
		go func() {
			if err := w.Rotate(w.ctx); err != nil {
				w.logger.WithError(err).Error("Failed to rotate segment")
			}
		}()
	}

	return nil
}

// Read reads records from WAL
func (w *DefaultWALManager) Read(ctx context.Context, filter WALFilter) ([]*types.Record, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	var allRecords []*types.Record

	// Read from all segments
	for _, segment := range w.segments {
		records, err := segment.Read(ctx, filter)
		if err != nil {
			w.logger.WithError(err).Warn("Failed to read from segment")
			continue
		}
		allRecords = append(allRecords, records...)
	}

	// Read from active segment
	if w.activeSegment != nil {
		records, err := w.activeSegment.Read(ctx, filter)
		if err != nil {
			w.logger.WithError(err).Warn("Failed to read from active segment")
		} else {
			allRecords = append(allRecords, records...)
		}
	}

	// Apply limit
	if filter.Limit > 0 && len(allRecords) > filter.Limit {
		allRecords = allRecords[:filter.Limit]
	}

	return allRecords, nil
}

// Flush forcefully syncs all records
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

// Rotate performs rotation of the active segment
func (w *DefaultWALManager) Rotate(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.activeSegment == nil {
		return nil
	}

	// Close active segment
	if err := w.activeSegment.Close(ctx); err != nil {
		return fmt.Errorf("failed to close active segment: %w", err)
	}

	// Add to segments list
	w.segments = append(w.segments, w.activeSegment)
	w.stats.SegmentCount++
	w.stats.LastRotateTime = time.Now()

	// Create new active segment
	if err := w.createActiveSegment(ctx); err != nil {
		return fmt.Errorf("failed to create new active segment: %w", err)
	}

	return nil
}

// Close closes the WAL manager
func (w *DefaultWALManager) Close(ctx context.Context) error {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return nil
	}
	w.closed = true
	w.mu.Unlock()

	// Stop background goroutines
	if w.cancel != nil {
		w.cancel()
	}

	// Wait for background goroutines to finish
	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Goroutines finished
	case <-ctx.Done():
		// Context canceled, but continue closing
	case <-time.After(5 * time.Second):
		// Waiting timeout
		w.logger.Warn("Timeout waiting for background goroutines to finish")
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	// Close active segment
	if w.activeSegment != nil {
		if err := w.activeSegment.Close(ctx); err != nil {
			w.logger.WithError(err).Error("Failed to close active segment")
		}
	}

	// Close all segments
	for _, segment := range w.segments {
		if err := segment.Close(ctx); err != nil {
			w.logger.WithError(err).Error("Failed to close segment")
		}
	}

	return nil
}

// GetStats returns WAL statistics
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

// shouldRotate checks if rotation should be performed
func (w *DefaultWALManager) shouldRotate() bool {
	if w.activeSegment == nil {
		return true
	}

	// Check size
	maxSize := parseSize(w.config.SegmentSize)
	if maxSize > 0 && w.activeSegment.Size() >= maxSize {
		return true
	}

	// Check age
	maxAge := w.config.MaxAge
	if maxAge > 0 && time.Since(w.activeSegment.CreatedAt()) >= maxAge {
		return true
	}

	return false
}

// createActiveSegment creates a new active segment
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

// recoverSegments recovers existing segments
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
		segmentID := entry.Name()[:len(entry.Name())-4] // Remove .wal

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

// rotationLoop is the segment rotation loop
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

// compactionLoop is the old segment compression loop
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

// compactOldSegments compresses old segments
func (w *DefaultWALManager) compactOldSegments(ctx context.Context) {
	w.mu.Lock()
	defer w.mu.Unlock()

	for _, segment := range w.segments {
		if segment.IsCompressed() {
			continue
		}

		// Compress segments older than 1 hour
		if time.Since(segment.CreatedAt()) > time.Hour {
			if err := segment.Compress(ctx); err != nil {
				w.logger.WithError(err).Warn("Failed to compress segment")
			}
		}
	}
}

// parseSize parses size from string
func parseSize(s string) int64 {
	if s == "" {
		return 0
	}

	var size int64
	var unit string

	_, _ = fmt.Sscanf(s, "%d%s", &size, &unit)

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

// MarkSent marks a record as sent
func (w *DefaultWALManager) MarkSent(ctx context.Context, recordID string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Search for record in all segments and active segment
	// For optimization, we use a separate sent records index
	if w.activeSegment != nil {
		if err := w.activeSegment.MarkSent(ctx, recordID); err == nil {
			return nil
		}
	}

	for _, segment := range w.segments {
		if err := segment.MarkSent(ctx, recordID); err == nil {
			return nil
		}
	}

	return fmt.Errorf("record %s not found", recordID)
}

// GetUnsentRecords returns all unsent records
func (w *DefaultWALManager) GetUnsentRecords(ctx context.Context) ([]*types.Record, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	var allRecords []*types.Record

	// Read from all segments
	for _, segment := range w.segments {
		records, err := segment.GetUnsentRecords(ctx)
		if err != nil {
			w.logger.WithError(err).Warn("Failed to get unsent records from segment")
			continue
		}
		allRecords = append(allRecords, records...)
	}

	// Read from active segment
	if w.activeSegment != nil {
		records, err := w.activeSegment.GetUnsentRecords(ctx)
		if err != nil {
			w.logger.WithError(err).Warn("Failed to get unsent records from active segment")
		} else {
			allRecords = append(allRecords, records...)
		}
	}

	return allRecords, nil
}

// DeleteSentRecords deletes all sent records older than given time
func (w *DefaultWALManager) DeleteSentRecords(ctx context.Context, olderThan time.Duration) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	cutoff := time.Now().Add(-olderThan)
	var segmentsToRemove []*WALSegment

	// Check each segment
	for _, segment := range w.segments {
		// If all records in segment are sent and segment is older than cutoff
		allSent, err := segment.AllRecordsSent(ctx)
		if err != nil {
			w.logger.WithError(err).Warn("Failed to check if all records sent in segment")
			continue
		}

		if allSent && segment.CreatedAt().Before(cutoff) {
			segmentsToRemove = append(segmentsToRemove, segment)
		}
	}

	// Delete segments
	for _, segment := range segmentsToRemove {
		if err := segment.Close(ctx); err != nil {
			w.logger.WithError(err).Warn("Failed to close segment before deletion")
			continue
		}

		segmentPath := filepath.Join(w.dir, segment.ID+".wal")
		if err := os.Remove(segmentPath); err != nil {
			w.logger.WithError(err).Warn("Failed to delete segment file")
			continue
		}

		// Remove from segments list
		for i, s := range w.segments {
			if s.ID == segment.ID {
				w.segments = append(w.segments[:i], w.segments[i+1:]...)
				w.stats.SegmentCount--
				break
			}
		}

		w.logger.WithField("segment_id", segment.ID).Info("Deleted old segment with all records sent")
	}

	return nil
}
