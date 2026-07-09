package wal

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/gdagil/vmprober/internal/config"
	"github.com/gdagil/vmprober/internal/types"
)

// quietLogger returns a logger that stays silent during tests.
func quietLogger() *logrus.Logger {
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	return l
}

// buildSegment creates a standalone segment on disk containing a single record.
// The segment's file name matches its ID so that DeleteSentRecords, which
// reconstructs the path from dir + ID + ".wal", targets the same file. The
// write handle is left open; callers (or manager.Close) are responsible for
// closing it so t.TempDir cleanup can run on Windows.
func buildSegment(t *testing.T, dir, id string, cfg *config.WALConfig, sent bool, createdAt time.Time) *WALSegment {
	t.Helper()
	seg, err := NewWALSegment(id, filepath.Join(dir, id+".wal"), cfg, NewCompressor(cfg.Compression), quietLogger())
	if err != nil {
		t.Fatalf("failed to build segment %s: %v", id, err)
	}
	rec := &types.Record{
		ID:        id + "-rec",
		Timestamp: time.Now(),
		Type:      "test",
		SeriesID:  "series",
		Sent:      sent,
	}
	if err := seg.Write(context.Background(), rec); err != nil {
		t.Fatalf("failed to write to segment %s: %v", id, err)
	}
	seg.createdAt = createdAt
	return seg
}

func TestNewWALManager_NilConfig(t *testing.T) {
	if _, err := NewWALManager(nil, quietLogger()); err == nil {
		t.Error("expected error when config is nil")
	}
}

func TestNewWALManager_MkdirFails(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a regular file, then point the WAL dir at a path *underneath* it.
	// MkdirAll cannot create a directory under a file, so it must fail on both
	// Windows and Linux.
	filePath := filepath.Join(tmpDir, "not-a-dir")
	if err := os.WriteFile(filePath, []byte("x"), 0o600); err != nil {
		t.Fatalf("failed to create blocking file: %v", err)
	}

	cfg := createTestWALConfig()
	cfg.Dir = filepath.Join(filePath, "wal")

	if _, err := NewWALManager(cfg, quietLogger()); err == nil {
		t.Error("expected error when WAL directory cannot be created")
	}
}

func TestWALManager_RecoverSegments_MissingDir(t *testing.T) {
	dm := &DefaultWALManager{
		dir:    filepath.Join(t.TempDir(), "does-not-exist"),
		logger: quietLogger(),
		stats:  &WALStats{},
	}

	if err := dm.recoverSegments(context.Background()); err != nil {
		t.Fatalf("recoverSegments on a missing dir should return nil, got %v", err)
	}
	if dm.stats.SegmentCount != 0 {
		t.Errorf("expected 0 segments recovered, got %d", dm.stats.SegmentCount)
	}
}

func TestWALManager_RecoverSegments_SkipsDirsAndNonWAL(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := createTestWALConfig()
	cfg.Dir = tmpDir
	ctx := context.Background()

	// A valid, closed .wal segment that recovery should pick up.
	pre, err := NewWALSegment("preexisting", filepath.Join(tmpDir, "preexisting.wal"), cfg, NewCompressor(cfg.Compression), quietLogger())
	if err != nil {
		t.Fatalf("failed to create preexisting segment: %v", err)
	}
	if err := pre.Write(ctx, &types.Record{ID: "r", Timestamp: time.Now(), Type: "test", SeriesID: "s"}); err != nil {
		t.Fatalf("failed to write preexisting record: %v", err)
	}
	if err := pre.Close(ctx); err != nil {
		t.Fatalf("failed to close preexisting segment: %v", err)
	}

	// Noise that recovery must skip: a subdirectory, a non-.wal file, and a
	// directory whose name ends in .wal (IsDir must win over the extension).
	if err := os.Mkdir(filepath.Join(tmpDir, "subdir"), 0o750); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "notes.txt"), []byte("ignore"), 0o600); err != nil {
		t.Fatalf("failed to create notes.txt: %v", err)
	}
	if err := os.Mkdir(filepath.Join(tmpDir, "dir-segment.wal"), 0o750); err != nil {
		t.Fatalf("failed to create dir-segment.wal: %v", err)
	}

	manager, err := NewWALManager(cfg, quietLogger())
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}
	defer manager.Close(ctx)

	stats := manager.GetStats()
	if stats.SegmentCount != 1 {
		t.Errorf("expected exactly 1 recovered segment, got %d", stats.SegmentCount)
	}
}

func TestWALManager_Recover_ReadsRecordsBack(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := createTestWALConfig()
	cfg.Dir = tmpDir
	ctx := context.Background()

	// First manager: write records across a rotation, then close.
	m1, err := NewWALManager(cfg, quietLogger())
	if err != nil {
		t.Fatalf("failed to create first manager: %v", err)
	}
	r1 := &types.Record{ID: "rec-1", Timestamp: time.Now(), Type: "test", SeriesID: "s1"}
	r2 := &types.Record{ID: "rec-2", Timestamp: time.Now(), Type: "test", SeriesID: "s2"}
	if err := m1.Write(ctx, r1); err != nil {
		t.Fatalf("failed to write rec-1: %v", err)
	}
	if err := m1.Rotate(ctx); err != nil {
		t.Fatalf("failed to rotate: %v", err)
	}
	if err := m1.Write(ctx, r2); err != nil {
		t.Fatalf("failed to write rec-2: %v", err)
	}
	if err := m1.Close(ctx); err != nil {
		t.Fatalf("failed to close first manager: %v", err)
	}

	// Second manager over the same dir must recover and read both records.
	m2, err := NewWALManager(cfg, quietLogger())
	if err != nil {
		t.Fatalf("failed to create second manager: %v", err)
	}
	defer m2.Close(ctx)

	recs, err := m2.Read(ctx, WALFilter{Limit: 100})
	if err != nil {
		t.Fatalf("failed to read recovered records: %v", err)
	}

	found := make(map[string]bool)
	for _, r := range recs {
		found[r.ID] = true
	}
	if !found["rec-1"] || !found["rec-2"] {
		t.Errorf("expected both rec-1 and rec-2 after recovery, got %v", found)
	}
}

func TestWALManager_MarkSent_SegmentsAndNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := createTestWALConfig()
	cfg.Dir = tmpDir
	ctx := context.Background()

	manager, err := NewWALManager(cfg, quietLogger())
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}
	defer manager.Close(ctx)
	dm := manager.(*DefaultWALManager)

	// Drop the active segment so MarkSent must fall through to the segments
	// loop. Close it first to release its file handle.
	dm.mu.Lock()
	active := dm.activeSegment
	dm.activeSegment = nil
	dm.mu.Unlock()
	if active != nil {
		if err := active.Close(ctx); err != nil {
			t.Fatalf("failed to close original active segment: %v", err)
		}
	}

	// Inject a rotated segment so the segments loop has something to mark.
	seg := buildSegment(t, tmpDir, "seg-loop", cfg, false, time.Now())
	defer seg.Close(ctx)
	dm.mu.Lock()
	dm.segments = []*WALSegment{seg}
	dm.mu.Unlock()

	// With no active segment, MarkSent succeeds via the segments loop.
	if err := manager.MarkSent(ctx, "any-id"); err != nil {
		t.Fatalf("MarkSent via segments loop should succeed, got %v", err)
	}

	// With no active segment and no segments, MarkSent reports not found.
	dm.mu.Lock()
	dm.segments = nil
	dm.mu.Unlock()
	if err := manager.MarkSent(ctx, "missing-id"); err == nil {
		t.Error("MarkSent should return an error when the record cannot be found")
	}
}

func TestWALManager_CompactOldSegments_CompressesOldSkipsRest(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := createTestWALConfig()
	cfg.Dir = tmpDir
	ctx := context.Background()

	manager, err := NewWALManager(cfg, quietLogger())
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}
	defer manager.Close(ctx)
	dm := manager.(*DefaultWALManager)

	old := time.Now().Add(-2 * time.Hour)
	segOld := buildSegment(t, tmpDir, "seg-old", cfg, false, old)
	segRecent := buildSegment(t, tmpDir, "seg-recent", cfg, false, time.Now())
	segCompressed := buildSegment(t, tmpDir, "seg-compressed", cfg, false, old)
	if err := segCompressed.Compress(ctx); err != nil {
		t.Fatalf("failed to pre-compress segment: %v", err)
	}

	dm.mu.Lock()
	dm.segments = []*WALSegment{segOld, segRecent, segCompressed}
	dm.mu.Unlock()

	dm.compactOldSegments(ctx)

	if !segOld.IsCompressed() {
		t.Error("old, uncompressed segment should have been compressed")
	}
	if segRecent.IsCompressed() {
		t.Error("recent segment should not have been compressed")
	}
	if !segCompressed.IsCompressed() {
		t.Error("already-compressed segment should remain compressed")
	}
}

func TestWALManager_DeleteSentRecords_RemovesOldFullySentSegments(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := createTestWALConfig()
	cfg.Dir = tmpDir
	ctx := context.Background()

	manager, err := NewWALManager(cfg, quietLogger())
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}
	defer manager.Close(ctx)
	dm := manager.(*DefaultWALManager)

	old := time.Now().Add(-2 * time.Hour)

	// Old and fully sent -> eligible for deletion.
	segDelete := buildSegment(t, tmpDir, "seg-old-sent", cfg, true, old)
	// Fully sent but recent -> kept (not older than cutoff).
	segRecent := buildSegment(t, tmpDir, "seg-recent-sent", cfg, true, time.Now())
	// Old but has an unsent record -> kept (not fully sent).
	segUnsent := buildSegment(t, tmpDir, "seg-old-unsent", cfg, false, old)

	dm.mu.Lock()
	dm.segments = []*WALSegment{segDelete, segRecent, segUnsent}
	dm.stats.SegmentCount = 3
	dm.mu.Unlock()

	if err := manager.DeleteSentRecords(ctx, time.Hour); err != nil {
		t.Fatalf("DeleteSentRecords failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(tmpDir, "seg-old-sent.wal")); !os.IsNotExist(err) {
		t.Errorf("expected seg-old-sent.wal to be deleted, stat error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "seg-recent-sent.wal")); err != nil {
		t.Errorf("seg-recent-sent.wal should still exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "seg-old-unsent.wal")); err != nil {
		t.Errorf("seg-old-unsent.wal should still exist: %v", err)
	}

	dm.mu.RLock()
	remaining := len(dm.segments)
	remainingIDs := make([]string, 0, remaining)
	for _, s := range dm.segments {
		remainingIDs = append(remainingIDs, s.ID)
	}
	remainingCount := dm.stats.SegmentCount
	dm.mu.RUnlock()

	if remaining != 2 {
		t.Errorf("expected 2 segments remaining, got %d (%v)", remaining, remainingIDs)
	}
	if remainingCount != 2 {
		t.Errorf("expected SegmentCount to drop to 2, got %d", remainingCount)
	}
	for _, id := range remainingIDs {
		if id == "seg-old-sent" {
			t.Error("seg-old-sent should have been removed from the segments list")
		}
	}
}

func TestWALManager_RotationLoop_RotatesOnTick(t *testing.T) {
	origRotation := rotationLoopInterval
	origCompaction := compactionLoopInterval
	rotationLoopInterval = 15 * time.Millisecond
	compactionLoopInterval = 15 * time.Millisecond
	defer func() {
		rotationLoopInterval = origRotation
		compactionLoopInterval = origCompaction
	}()

	tmpDir := t.TempDir()
	cfg := createTestWALConfig()
	cfg.Dir = tmpDir
	// A tiny MaxAge makes the active segment eligible for rotation almost
	// immediately, so the background rotation loop will rotate on its next tick.
	cfg.MaxAge = time.Millisecond

	manager, err := NewWALManager(cfg, quietLogger())
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}
	defer manager.Close(context.Background())

	deadline := time.Now().Add(2 * time.Second)
	rotated := false
	for time.Now().Before(deadline) {
		if manager.GetStats().SegmentCount >= 1 {
			rotated = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !rotated {
		t.Error("rotationLoop did not rotate the active segment within the timeout")
	}
}
