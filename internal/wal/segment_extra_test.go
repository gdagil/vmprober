package wal

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/gdagil/vmprober/internal/types"
)

func TestValidatePath_Direct(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"valid absolute", filepath.Join(t.TempDir(), "seg.wal"), false},
		{"valid relative", "data/seg.wal", false},
		{"traversal prefix", "../../etc/passwd", true},
		{"traversal middle", "foo/../bar.wal", true},
		{"null byte", "seg\x00.wal", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePath(tt.path)
			if tt.wantErr && err == nil {
				t.Errorf("validatePath(%q) = nil, expected error", tt.path)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("validatePath(%q) = %v, expected nil", tt.path, err)
			}
		})
	}
}

func TestOpenWALSegment_InvalidPath(t *testing.T) {
	cfg := createTestSegmentConfig()
	if _, err := OpenWALSegment("bad", "../../../etc/passwd", cfg, NewCompressor(cfg.Compression), quietLogger()); err == nil {
		t.Error("expected error opening a segment with a traversal path")
	}
}

func TestOpenWALSegment_NonExistentFile(t *testing.T) {
	cfg := createTestSegmentConfig()
	missing := filepath.Join(t.TempDir(), "missing.wal")
	if _, err := OpenWALSegment("missing", missing, cfg, NewCompressor(cfg.Compression), quietLogger()); err == nil {
		t.Error("expected error opening a non-existent segment file")
	}
}

func TestOpenWALSegment_RoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := createTestSegmentConfig()
	ctx := context.Background()
	path := filepath.Join(tmpDir, "round-trip.wal")
	compressor := NewCompressor(cfg.Compression)

	// Create, write, close.
	seg, err := NewWALSegment("round-trip", path, cfg, compressor, quietLogger())
	if err != nil {
		t.Fatalf("failed to create segment: %v", err)
	}
	rec := &types.Record{ID: "rt-1", Timestamp: time.Now(), Type: "test", SeriesID: "s1"}
	if err := seg.Write(ctx, rec); err != nil {
		t.Fatalf("failed to write record: %v", err)
	}
	if err := seg.Close(ctx); err != nil {
		t.Fatalf("failed to close segment: %v", err)
	}

	// Reopen via OpenWALSegment: size and createdAt come from the file's stat.
	opened, err := OpenWALSegment("round-trip", path, cfg, compressor, quietLogger())
	if err != nil {
		t.Fatalf("failed to open existing segment: %v", err)
	}
	defer opened.Close(ctx)

	if opened.Size() <= 0 {
		t.Errorf("expected positive size from stat, got %d", opened.Size())
	}
	if opened.CreatedAt().IsZero() {
		t.Error("expected non-zero createdAt from file mod time")
	}

	recs, err := opened.Read(ctx, WALFilter{Limit: 10})
	if err != nil {
		t.Fatalf("failed to read from reopened segment: %v", err)
	}
	found := false
	for _, r := range recs {
		if r.ID == "rt-1" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected to read back rt-1 from reopened segment, got %d records", len(recs))
	}
}
