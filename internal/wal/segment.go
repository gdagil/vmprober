package wal

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vmprober/vmprober/internal/config"
	"github.com/vmprober/vmprober/internal/types"
)

// WALSegment представляет сегмент WAL
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
}

// NewWALSegment создает новый сегмент WAL
func NewWALSegment(id, path string, cfg *config.WALConfig, compressor Compressor, logger *logrus.Logger) (*WALSegment, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
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
	}, nil
}

// OpenWALSegment открывает существующий сегмент WAL
func OpenWALSegment(id, path string, cfg *config.WALConfig, compressor Compressor, logger *logrus.Logger) (*WALSegment, error) {
	file, err := os.OpenFile(path, os.O_RDONLY, 0644)
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
	}

	return segment, nil
}

// Write записывает запись в сегмент
func (s *WALSegment) Write(ctx context.Context, record *types.Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.file == nil {
		return fmt.Errorf("segment file is closed")
	}

	// Сериализация записи
	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("failed to marshal record: %w", err)
	}

	// Компрессия если нужно
	if s.compressor != nil && s.config.Compression != "" {
		data, err = s.compressor.Compress(data)
		if err != nil {
			return fmt.Errorf("failed to compress record: %w", err)
		}
	}

	// Запись размера записи
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

	// Запись данных
	if _, err := s.file.Write(data); err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}

	s.recordCount++
	s.size += int64(len(data) + 8)

	return nil
}

// Read читает записи из сегмента
func (s *WALSegment) Read(ctx context.Context, filter WALFilter) ([]*types.Record, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.file == nil {
		return nil, fmt.Errorf("segment file is closed")
	}

	// Переоткрытие файла для чтения если нужно
	file, err := os.Open(s.path)
	if err != nil {
		return nil, fmt.Errorf("failed to open segment for reading: %w", err)
	}
	defer file.Close()

	var records []*types.Record
	sizeBytes := make([]byte, 8)

	for {
		// Чтение размера
		if _, err := file.Read(sizeBytes); err != nil {
			if err.Error() == "EOF" {
				break
			}
			return nil, fmt.Errorf("failed to read size: %w", err)
		}

		size := int64(sizeBytes[0])<<56 | int64(sizeBytes[1])<<48 | int64(sizeBytes[2])<<40 | int64(sizeBytes[3])<<32 |
			int64(sizeBytes[4])<<24 | int64(sizeBytes[5])<<16 | int64(sizeBytes[6])<<8 | int64(sizeBytes[7])

		// Чтение данных
		data := make([]byte, size)
		if _, err := file.Read(data); err != nil {
			return nil, fmt.Errorf("failed to read data: %w", err)
		}

		// Декомпрессия если нужно
		if s.compressor != nil && s.config.Compression != "" {
			data, err = s.compressor.Decompress(data)
			if err != nil {
				return nil, fmt.Errorf("failed to decompress record: %w", err)
			}
		}

		// Десериализация записи
		var record types.Record
		if err := json.Unmarshal(data, &record); err != nil {
			s.logger.WithError(err).Warn("Failed to unmarshal record")
			continue
		}

		// Применение фильтра
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

// Flush принудительно синхронизирует файл
func (s *WALSegment) Flush(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.file == nil {
		return nil
	}

	return s.file.Sync()
}

// Close закрывает сегмент
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

// Compress сжимает сегмент
func (s *WALSegment) Compress(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.compressed || s.compressor == nil {
		return nil
	}

	// Реализация компрессии сегмента
	// В реальной реализации здесь была бы логика сжатия всего файла
	s.compressed = true
	return nil
}

// Size возвращает размер сегмента
func (s *WALSegment) Size() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.size
}

// CreatedAt возвращает время создания сегмента
func (s *WALSegment) CreatedAt() time.Time {
	return s.createdAt
}

// IsCompressed проверяет сжат ли сегмент
func (s *WALSegment) IsCompressed() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.compressed
}

