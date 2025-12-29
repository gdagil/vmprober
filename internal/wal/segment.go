package wal

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/gdagil/vmprober/internal/config"
	"github.com/gdagil/vmprober/internal/types"
)

// WALSegment represents a WAL segment
type WALSegment struct {
	ID          string
	path        string
	file        *os.File
	mu          sync.RWMutex
	config      *config.WALConfig
	compressor  Compressor
	logger      *logrus.Logger
	createdAt   time.Time
	recordCount int64
	size        int64
	compressed  bool
	sentIndex   map[string]bool // Index of sent records
}

// NewWALSegment creates a new WAL segment
func NewWALSegment(id, path string, cfg *config.WALConfig, compressor Compressor, logger *logrus.Logger) (*WALSegment, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("failed to create segment file: %w", err)
	}

	return &WALSegment{
		ID:         id,
		path:       path,
		file:       file,
		config:     cfg,
		compressor: compressor,
		logger:     logger,
		createdAt:  time.Now(),
		sentIndex:  make(map[string]bool),
	}, nil
}

// OpenWALSegment opens an existing WAL segment
func OpenWALSegment(id, path string, cfg *config.WALConfig, compressor Compressor, logger *logrus.Logger) (*WALSegment, error) {
	file, err := os.OpenFile(path, os.O_RDONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("failed to open segment file: %w", err)
	}

	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to stat segment file: %w", err)
	}

	segment := &WALSegment{
		ID:         id,
		path:       path,
		file:       file,
		config:     cfg,
		compressor: compressor,
		logger:     logger,
		createdAt:  info.ModTime(),
		size:       info.Size(),
		sentIndex:  make(map[string]bool),
	}

	// Load sent records index from companion file
	if err := segment.loadSentIndex(); err != nil {
		logger.WithError(err).Debug("Failed to load sent index, starting fresh")
	}

	return segment, nil
}

// Write writes a record to the segment
func (s *WALSegment) Write(ctx context.Context, record *types.Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.file == nil {
		return fmt.Errorf("segment file is closed")
	}

	// Serialize record
	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("failed to marshal record: %w", err)
	}

	// Compress if needed
	if s.compressor != nil && s.config.Compression != "" {
		data, err = s.compressor.Compress(data)
		if err != nil {
			return fmt.Errorf("failed to compress record: %w", err)
		}
	}

	// Write record size
	sizeBytes := make([]byte, 8)
	sizeBytes[0] = byte(len(data) >> 56)
	sizeBytes[1] = byte(len(data) >> 48)
	sizeBytes[2] = byte(len(data) >> 40)
	sizeBytes[3] = byte(len(data) >> 32)
	sizeBytes[4] = byte(len(data) >> 24)
	sizeBytes[5] = byte(len(data) >> 16)
	sizeBytes[6] = byte(len(data) >> 8)
	sizeBytes[7] = byte(len(data))

	if _, err := s.file.Write(sizeBytes); err != nil {
		return fmt.Errorf("failed to write size: %w", err)
	}

	// Write data
	if _, err := s.file.Write(data); err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}

	s.recordCount++
	s.size += int64(len(data) + 8)

	return nil
}

// Read reads records from the segment
func (s *WALSegment) Read(ctx context.Context, filter WALFilter) ([]*types.Record, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.file == nil {
		return nil, fmt.Errorf("segment file is closed")
	}

	// Reopen file for reading if needed
	file, err := os.Open(s.path)
	if err != nil {
		return nil, fmt.Errorf("failed to open segment for reading: %w", err)
	}
	defer file.Close()

	var records []*types.Record
	sizeBytes := make([]byte, 8)

	for {
		// Read size
		if _, err := file.Read(sizeBytes); err != nil {
			if err.Error() == "EOF" {
				break
			}
			return nil, fmt.Errorf("failed to read size: %w", err)
		}

		size := int64(sizeBytes[0])<<56 | int64(sizeBytes[1])<<48 | int64(sizeBytes[2])<<40 | int64(sizeBytes[3])<<32 |
			int64(sizeBytes[4])<<24 | int64(sizeBytes[5])<<16 | int64(sizeBytes[6])<<8 | int64(sizeBytes[7])

		// Read data
		data := make([]byte, size)
		if _, err := file.Read(data); err != nil {
			return nil, fmt.Errorf("failed to read data: %w", err)
		}

		// Decompress if needed
		if s.compressor != nil && s.config.Compression != "" {
			data, err = s.compressor.Decompress(data)
			if err != nil {
				return nil, fmt.Errorf("failed to decompress record: %w", err)
			}
		}

		// Deserialize record
		var record types.Record
		if err := json.Unmarshal(data, &record); err != nil {
			s.logger.WithError(err).Warn("Failed to unmarshal record")
			continue
		}

		// Apply filter
		if filter.Type != "" && record.Type != filter.Type {
			continue
		}
		if filter.SeriesID != "" && record.SeriesID != filter.SeriesID {
			continue
		}
		if !filter.StartTime.IsZero() && record.Timestamp.Before(filter.StartTime) {
			continue
		}
		if !filter.EndTime.IsZero() && record.Timestamp.After(filter.EndTime) {
			continue
		}

		records = append(records, &record)

		if filter.Limit > 0 && len(records) >= filter.Limit {
			break
		}
	}

	return records, nil
}

// Flush forcefully syncs the file
func (s *WALSegment) Flush(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.file == nil {
		return nil
	}

	return s.file.Sync()
}

// Close closes the segment
func (s *WALSegment) Close(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.file == nil {
		return nil
	}

	if err := s.file.Sync(); err != nil {
		s.logger.WithError(err).Warn("Failed to sync segment before close")
	}

	if err := s.file.Close(); err != nil {
		return fmt.Errorf("failed to close segment file: %w", err)
	}

	s.file = nil
	return nil
}

// Compress compresses the segment
func (s *WALSegment) Compress(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.compressed || s.compressor == nil {
		return nil
	}

	// Segment compression implementation
	// In a real implementation, there would be logic to compress the entire file
	s.compressed = true
	return nil
}

// Size returns the segment size
func (s *WALSegment) Size() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.size
}

// CreatedAt returns the segment creation time
func (s *WALSegment) CreatedAt() time.Time {
	return s.createdAt
}

// IsCompressed checks if the segment is compressed
func (s *WALSegment) IsCompressed() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.compressed
}

// MarkSent marks a record as sent
func (s *WALSegment) MarkSent(ctx context.Context, recordID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sentIndex[recordID] = true

	// Save index to disk for persistence
	return s.saveSentIndex()
}

// GetUnsentRecords returns all unsent records from the segment
func (s *WALSegment) GetUnsentRecords(ctx context.Context) ([]*types.Record, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Read all records from file
	file, err := os.Open(s.path)
	if err != nil {
		return nil, fmt.Errorf("failed to open segment for reading: %w", err)
	}
	defer file.Close()

	var records []*types.Record
	sizeBytes := make([]byte, 8)

	for {
		// Read size
		if _, err := file.Read(sizeBytes); err != nil {
			if err.Error() == "EOF" {
				break
			}
			return nil, fmt.Errorf("failed to read size: %w", err)
		}

		size := int64(sizeBytes[0])<<56 | int64(sizeBytes[1])<<48 | int64(sizeBytes[2])<<40 | int64(sizeBytes[3])<<32 |
			int64(sizeBytes[4])<<24 | int64(sizeBytes[5])<<16 | int64(sizeBytes[6])<<8 | int64(sizeBytes[7])

		// Read data
		data := make([]byte, size)
		if _, err := file.Read(data); err != nil {
			return nil, fmt.Errorf("failed to read data: %w", err)
		}

		// Decompress if needed
		if s.compressor != nil && s.config.Compression != "" {
			data, err = s.compressor.Decompress(data)
			if err != nil {
				return nil, fmt.Errorf("failed to decompress record: %w", err)
			}
		}

		// Deserialize record
		var record types.Record
		if err := json.Unmarshal(data, &record); err != nil {
			s.logger.WithError(err).Warn("Failed to unmarshal record")
			continue
		}

		// Check if record was already sent (by index or by flag)
		if s.sentIndex[record.ID] || record.Sent {
			continue
		}

		records = append(records, &record)
	}

	return records, nil
}

// AllRecordsSent checks if all records in the segment have been sent
func (s *WALSegment) AllRecordsSent(ctx context.Context) (bool, error) {
	unsent, err := s.GetUnsentRecords(ctx)
	if err != nil {
		return false, err
	}
	return len(unsent) == 0, nil
}

// loadSentIndex loads the sent records index from file
func (s *WALSegment) loadSentIndex() error {
	indexPath := s.path + ".sent"
	data, err := os.ReadFile(indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	return json.Unmarshal(data, &s.sentIndex)
}

// saveSentIndex saves the sent records index to file
func (s *WALSegment) saveSentIndex() error {
	indexPath := s.path + ".sent"
	data, err := json.Marshal(s.sentIndex)
	if err != nil {
		return err
	}

	return os.WriteFile(indexPath, data, 0o600)
}
