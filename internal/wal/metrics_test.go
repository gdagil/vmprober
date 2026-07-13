package wal

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/gdagil/vmprober/internal/config"
	"github.com/gdagil/vmprober/internal/types"
)

// TestWALMetrics verifies that Write/MarkSent feed the vmprober_wal_* metrics:
// bytes_written grows by the on-disk record size and MarkSent attributes the
// same size to bytes_sent exactly once.
func TestWALMetrics(t *testing.T) {
	cfg := &config.WALConfig{
		Dir:         t.TempDir(),
		SegmentSize: "64MB",
		Compression: "none",
	}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	manager, err := NewWALManager(cfg, logger)
	if err != nil {
		t.Fatalf("NewWALManager: %v", err)
	}
	ctx := context.Background()
	defer manager.Close(ctx)

	writtenBefore := walBytesWritten.Get()
	sentBefore := walBytesSent.Get()

	record := &types.Record{
		ID:        "metrics-test-record",
		Timestamp: time.Now(),
		Type:      "probe_result",
		Data:      map[string]interface{}{"probe": "tcp"},
	}
	if err := manager.Write(ctx, record); err != nil {
		t.Fatalf("Write: %v", err)
	}

	writtenDelta := walBytesWritten.Get() - writtenBefore
	if writtenDelta == 0 {
		t.Fatal("vmprober_wal_bytes_written did not grow after Write")
	}
	if walSegmentsGauge.Get() < 1 {
		t.Fatalf("vmprober_wal_segments_total = %v, want >= 1", walSegmentsGauge.Get())
	}

	if err := manager.MarkSent(ctx, record.ID); err != nil {
		t.Fatalf("MarkSent: %v", err)
	}
	sentDelta := walBytesSent.Get() - sentBefore
	if sentDelta != writtenDelta {
		t.Fatalf("vmprober_wal_bytes_sent grew by %d, want %d (the written size)", sentDelta, writtenDelta)
	}

	// Second MarkSent for the same record must not double-count.
	_ = manager.MarkSent(ctx, record.ID)
	if walBytesSent.Get()-sentBefore != sentDelta {
		t.Fatal("vmprober_wal_bytes_sent double-counted a re-sent record")
	}
}
