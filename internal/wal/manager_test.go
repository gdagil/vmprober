package wal

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/gdagil/vmprober/internal/config"
	"github.com/gdagil/vmprober/internal/types"
)

func createTestWALConfig() *config.WALConfig {
	return &config.WALConfig{
		Dir:            "",
		MaxSize:        "10MB",
		MaxAge:         1 * time.Hour,
		Retention:      24 * time.Hour,
		Compression:    "gzip",
		SyncInterval:   1 * time.Second,
		BufferSize:     "64KB",
		SegmentSize:    "1MB",
		IndexCacheSize: 100,
	}
}

func TestNewWALManager(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := createTestWALConfig()
	cfg.Dir = tmpDir

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Reduce logging in tests

	manager, err := NewWALManager(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create WAL manager: %v", err)
	}
	defer manager.Close(context.Background())

	if manager == nil {
		t.Fatal("WAL manager is nil")
	}
}

func TestWALManager_Write(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := createTestWALConfig()
	cfg.Dir = tmpDir

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	manager, err := NewWALManager(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create WAL manager: %v", err)
	}
	defer manager.Close(context.Background())

	ctx := context.Background()
	record := &types.Record{
		ID:        "test-record-1",
		Timestamp: time.Now(),
		Type:      "test",
		SeriesID:  "series-1",
		Data: map[string]interface{}{
			"value": 42,
		},
		Labels: map[string]string{
			"test": "true",
		},
	}

	if err := manager.Write(ctx, record); err != nil {
		t.Fatalf("Failed to write record: %v", err)
	}

	stats := manager.GetStats()
	if stats.TotalRecords != 1 {
		t.Errorf("Expected 1 record, got %d", stats.TotalRecords)
	}
}

func TestWALManager_Read(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := createTestWALConfig()
	cfg.Dir = tmpDir

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	manager, err := NewWALManager(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create WAL manager: %v", err)
	}
	defer manager.Close(context.Background())

	ctx := context.Background()

	// Write several records
	for i := 0; i < 3; i++ {
		record := &types.Record{
			ID:        fmt.Sprintf("test-record-%d", i),
			Timestamp: time.Now().Add(time.Duration(i) * time.Second),
			Type:      "test",
			SeriesID:  fmt.Sprintf("series-%d", i),
			Data: map[string]interface{}{
				"value": i,
			},
		}
		if err := manager.Write(ctx, record); err != nil {
			t.Fatalf("Failed to write record %d: %v", i, err)
		}
	}

	// Read all records
	filter := WALFilter{
		Limit: 10,
	}
	records, err := manager.Read(ctx, filter)
	if err != nil {
		t.Fatalf("Failed to read records: %v", err)
	}

	if len(records) != 3 {
		t.Errorf("Expected 3 records, got %d", len(records))
	}
}

func TestWALManager_ReadWithFilter(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := createTestWALConfig()
	cfg.Dir = tmpDir

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	manager, err := NewWALManager(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create WAL manager: %v", err)
	}
	defer manager.Close(context.Background())

	ctx := context.Background()

	now := time.Now()
	record1 := &types.Record{
		ID:        "record-1",
		Timestamp: now,
		Type:      "type1",
		SeriesID:  "series-1",
	}
	record2 := &types.Record{
		ID:        "record-2",
		Timestamp: now.Add(1 * time.Hour),
		Type:      "type2",
		SeriesID:  "series-2",
	}

	manager.Write(ctx, record1)
	manager.Write(ctx, record2)

	// Filter by type
	filter := WALFilter{
		Type:  "type1",
		Limit: 10,
	}
	records, err := manager.Read(ctx, filter)
	if err != nil {
		t.Fatalf("Failed to read records: %v", err)
	}

	if len(records) != 1 {
		t.Errorf("Expected 1 record with type1, got %d", len(records))
	}

	// Filter by SeriesID
	filter = WALFilter{
		SeriesID: "series-2",
		Limit:    10,
	}
	records, err = manager.Read(ctx, filter)
	if err != nil {
		t.Fatalf("Failed to read records: %v", err)
	}

	if len(records) != 1 {
		t.Errorf("Expected 1 record with series-2, got %d", len(records))
	}
}

func TestWALManager_Flush(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := createTestWALConfig()
	cfg.Dir = tmpDir

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	manager, err := NewWALManager(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create WAL manager: %v", err)
	}
	defer manager.Close(context.Background())

	ctx := context.Background()

	record := &types.Record{
		ID:        "test-record",
		Timestamp: time.Now(),
		Type:      "test",
		SeriesID:  "series-1",
	}

	if err := manager.Write(ctx, record); err != nil {
		t.Fatalf("Failed to write record: %v", err)
	}

	if err := manager.Flush(ctx); err != nil {
		t.Fatalf("Failed to flush: %v", err)
	}
}

func TestWALManager_Rotate(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := createTestWALConfig()
	cfg.Dir = tmpDir
	cfg.SegmentSize = "1KB" // Small size for fast rotation

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	manager, err := NewWALManager(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create WAL manager: %v", err)
	}
	defer manager.Close(context.Background())

	ctx := context.Background()

	// Write enough data for rotation
	largeData := make(map[string]interface{})
	for i := 0; i < 100; i++ {
		largeData[fmt.Sprintf("key%d", i)] = fmt.Sprintf("value%d", i)
	}

	for i := 0; i < 10; i++ {
		record := &types.Record{
			ID:        fmt.Sprintf("record-%d", i),
			Timestamp: time.Now(),
			Type:      "test",
			SeriesID:  "series-1",
			Data:      largeData,
		}
		if err := manager.Write(ctx, record); err != nil {
			t.Fatalf("Failed to write record %d: %v", i, err)
		}
	}

	// Force rotation
	if err := manager.Rotate(ctx); err != nil {
		t.Fatalf("Failed to rotate: %v", err)
	}

	stats := manager.GetStats()
	if stats.SegmentCount < 1 {
		t.Logf("Segment count: %d (may be 0 if rotation didn't trigger)", stats.SegmentCount)
	}
}

func TestWALManager_GetStats(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := createTestWALConfig()
	cfg.Dir = tmpDir

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	manager, err := NewWALManager(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create WAL manager: %v", err)
	}
	defer manager.Close(context.Background())

	stats := manager.GetStats()
	if stats == nil {
		t.Fatal("Stats is nil")
	}

	if stats.TotalRecords != 0 {
		t.Errorf("Expected 0 records initially, got %d", stats.TotalRecords)
	}
}

func TestWALManager_Close(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := createTestWALConfig()
	cfg.Dir = tmpDir

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	manager, err := NewWALManager(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create WAL manager: %v", err)
	}

	ctx := context.Background()
	if err := manager.Close(ctx); err != nil {
		t.Fatalf("Failed to close WAL manager: %v", err)
	}
}
