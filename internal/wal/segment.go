package wal

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	idIndex     map[string]bool // Lazy index of record IDs in this segment (built on first containment check; segments are append-only)
}

// NewWALSegment creates a new WAL segment
func NewWALSegment(id, path string, cfg *config.WALConfig, compressor Compressor, logger *logrus.Logger) (*WALSegment, error) {
	// Validate path to prevent directory traversal
	if err := validatePath(path); err != nil {
		return nil, fmt.Errorf("invalid segment path: %w", err)
	}
	// Clean path after validation to ensure it's safe
	cleanPath := filepath.Clean(path)
	//nolint:gosec // Path is validated by validatePath() to prevent directory traversal
	file, err := os.OpenFile(cleanPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, fmt.Errorf("failed to create segment file: %w", err)
	}

	return &WALSegment{
		ID:         id,
		path:       cleanPath,
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
	// Validate path to prevent directory traversal
	if err := validatePath(path); err != nil {
		return nil, fmt.Errorf("invalid segment path: %w", err)
	}
	// Clean path after validation to ensure it's safe
	cleanPath := filepath.Clean(path)
	//nolint:gosec // Path is validated by validatePath() to prevent directory traversal
	file, err := os.OpenFile(cleanPath, os.O_RDONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("failed to open segment file: %w", err)
	}

	info, err := file.Stat()
	if err != nil {
		if closeErr := file.Close(); closeErr != nil {
			logger.WithError(closeErr).Warn("Failed to close file after stat error")
		}
		return nil, fmt.Errorf("failed to stat segment file: %w", err)
	}

	segment := &WALSegment{
		ID:         id,
		path:       cleanPath,
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

	// Keep the lazy ID index current so MarkSent containment checks stay O(1)
	// after the index has been built once.
	if s.idIndex != nil {
		s.idIndex[record.ID] = true
	}

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

// MarkSent marks a record as sent. It only succeeds for records this segment
// actually contains; marking an unknown ID returns an error so callers (and the
// manager) do not record the wrong segment as owning a record, which would leave
// the real record perpetually unsent and re-delivered.
func (s *WALSegment) MarkSent(ctx context.Context, recordID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	contains, err := s.containsRecordLocked(recordID)
	if err != nil {
		return err
	}
	if !contains {
		return fmt.Errorf("record %s not found in segment %s", recordID, s.ID)
	}

	s.sentIndex[recordID] = true

	// Save index to disk for persistence
	return s.saveSentIndex()
}

// containsRecordLocked reports whether the segment holds a record with the
// given ID. The caller must hold s.mu. The first call scans the segment's data
// file once to build an in-memory ID index (segments are append-only; Write
// keeps the index current afterwards), so repeated MarkSent calls — e.g. WAL
// replay marking every record of a segment — stay O(1) instead of re-reading
// the file per record.
func (s *WALSegment) containsRecordLocked(recordID string) (bool, error) {
	if s.idIndex == nil {
		idx, err := s.buildIDIndexLocked()
		if err != nil {
			return false, err
		}
		s.idIndex = idx
	}
	return s.idIndex[recordID], nil
}

// buildIDIndexLocked scans the segment's data file and collects all record IDs.
// The caller must hold s.mu. It opens a fresh read handle so it does not
// disturb the append handle used for writes, mirroring Read and
// GetUnsentRecords.
func (s *WALSegment) buildIDIndexLocked() (map[string]bool, error) {
	file, err := os.Open(s.path)
	if err != nil {
		return nil, fmt.Errorf("failed to open segment for reading: %w", err)
	}
	defer file.Close()

	idx := make(map[string]bool, s.recordCount)
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

		idx[record.ID] = true
	}

	return idx, nil
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
	// Validate path to prevent directory traversal
	if err := validatePath(indexPath); err != nil {
		return fmt.Errorf("invalid index path: %w", err)
	}
	// Clean path after validation to ensure it's safe
	cleanIndexPath := filepath.Clean(indexPath)
	//nolint:gosec // Path is validated by validatePath() to prevent directory traversal
	data, err := os.ReadFile(cleanIndexPath)
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

// validatePath validates that the path doesn't contain directory traversal attempts
func validatePath(path string) error {
	// Check for directory traversal attempts before cleaning
	if strings.Contains(path, "..") {
		return fmt.Errorf("path contains directory traversal: %s", path)
	}

	// Clean the path to resolve any "." components
	cleaned := filepath.Clean(path)

	// Ensure cleaned path doesn't contain ".." (shouldn't happen after Clean, but double-check)
	if strings.Contains(cleaned, "..") {
		return fmt.Errorf("path contains directory traversal after cleaning: %s", path)
	}

	// Check for null bytes which could be used in path injection
	if strings.Contains(path, "\x00") {
		return fmt.Errorf("path contains null byte: %s", path)
	}

	return nil
}
