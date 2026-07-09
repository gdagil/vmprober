package normalizer

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/gdagil/vmprober/internal/types"
)

func TestNewNormalizer(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	normalizer := NewNormalizer(logger)
	if normalizer == nil {
		t.Fatal("NewNormalizer returned nil")
	}

	ctx := context.Background()
	defer normalizer.Close(ctx)
}

func TestNormalizer_Normalize(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	normalizer := NewNormalizer(logger)
	defer normalizer.Close(context.Background())

	ctx := context.Background()
	result := &types.ProbeResult{
		Success:    true,
		RTT:        100 * time.Millisecond,
		Attempt:    1,
		Timestamp:  time.Now(),
		Protocol:   types.ProbeTypeTCP,
		TargetIP:   "192.168.1.1",
		TargetPort: 80,
		SourceIP:   "192.168.1.2",
		Role:       "client",
	}

	event, err := normalizer.Normalize(ctx, result)
	if err != nil {
		t.Fatalf("Normalize failed: %v", err)
	}

	if event == nil {
		t.Fatal("Event is nil")
	}

	if event.SeriesID == "" {
		t.Error("SeriesID is empty")
	}

	if len(event.Metrics) == 0 {
		t.Error("Metrics are empty")
	}

	if len(event.Labels) == 0 {
		t.Error("Labels are empty")
	}

	// Check presence of main metrics
	if _, ok := event.Metrics["vmprober_probe_rtt_seconds"]; !ok {
		t.Error("vmprober_probe_rtt_seconds metric not found")
	}
	if _, ok := event.Metrics["vmprober_probe_result"]; !ok {
		t.Error("vmprober_probe_result metric not found")
	}
	if _, ok := event.Metrics["vmprober_probe_duration_seconds"]; !ok {
		t.Error("vmprober_probe_duration_seconds metric not found")
	}
	if _, ok := event.Metrics["vmprober_probe_attempts_total"]; !ok {
		t.Error("vmprober_probe_attempts_total metric not found")
	}

	// Check presence of status label
	if status, ok := event.Labels["status"]; !ok {
		t.Error("status label not found")
	} else if status != "success" && status != "failed" {
		t.Errorf("Invalid status label value: %s", status)
	}
}

func TestNormalizer_NormalizeBatch(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	normalizer := NewNormalizer(logger)
	defer normalizer.Close(context.Background())

	ctx := context.Background()
	results := []*types.ProbeResult{
		{
			Success:   true,
			RTT:       50 * time.Millisecond,
			Timestamp: time.Now(),
			Protocol:  types.ProbeTypeTCP,
			TargetIP:  "192.168.1.1",
		},
		{
			Success:   false,
			RTT:       200 * time.Millisecond,
			Timestamp: time.Now(),
			Protocol:  types.ProbeTypeUDP,
			TargetIP:  "192.168.1.2",
		},
	}

	events, err := normalizer.NormalizeBatch(ctx, results)
	if err != nil {
		t.Fatalf("NormalizeBatch failed: %v", err)
	}

	if len(events) != 2 {
		t.Errorf("Expected 2 events, got %d", len(events))
	}
}

func TestNormalizer_Dedup(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	normalizer := NewNormalizer(logger)
	defer normalizer.Close(context.Background())

	ctx := context.Background()
	result := &types.ProbeResult{
		Success:    true,
		RTT:        100 * time.Millisecond,
		Timestamp:  time.Now(),
		Protocol:   types.ProbeTypeTCP,
		TargetIP:   "192.168.1.1",
		TargetPort: 80,
	}

	event, err := normalizer.Normalize(ctx, result)
	if err != nil {
		t.Fatalf("Normalize failed: %v", err)
	}

	// First check - should not be duplicate
	isDup, err := normalizer.Dedup(ctx, event)
	if err != nil {
		t.Fatalf("Dedup failed: %v", err)
	}
	if isDup {
		t.Error("First event should not be duplicate")
	}

	// Second check of same event - should be duplicate
	isDup, err = normalizer.Dedup(ctx, event)
	if err != nil {
		t.Fatalf("Dedup failed: %v", err)
	}
	if !isDup {
		t.Error("Second event should be duplicate")
	}
}

func TestNormalizer_Enrich(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	normalizer := NewNormalizer(logger)
	defer normalizer.Close(context.Background())

	ctx := context.Background()
	result := &types.ProbeResult{
		Success:   true,
		RTT:       100 * time.Millisecond,
		Timestamp: time.Now(),
		Protocol:  types.ProbeTypeTCP,
		TargetIP:  "192.168.1.1",
	}

	event, err := normalizer.Normalize(ctx, result)
	if err != nil {
		t.Fatalf("Normalize failed: %v", err)
	}

	// Enrich is called automatically in Normalize
	if event.Metadata == nil {
		t.Error("Metadata is nil after enrichment")
	}

	if len(event.Metadata) == 0 {
		t.Error("Metadata is empty after enrichment")
	}
}

func TestNormalizer_GetStats(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	normalizer := NewNormalizer(logger)
	defer normalizer.Close(context.Background())

	ctx := context.Background()
	result := &types.ProbeResult{
		Success:   true,
		RTT:       100 * time.Millisecond,
		Timestamp: time.Now(),
		Protocol:  types.ProbeTypeTCP,
		TargetIP:  "192.168.1.1",
	}

	normalizer.Normalize(ctx, result)

	stats := normalizer.GetStats()
	if stats == nil {
		t.Fatal("Stats is nil")
	}

	if stats.TotalNormalized == 0 {
		t.Error("Expected at least 1 normalized event")
	}
}

func TestNormalizer_Close(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	normalizer := NewNormalizer(logger)

	ctx := context.Background()
	if err := normalizer.Close(ctx); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

func TestNormalizer_SeriesIDGeneration(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	normalizer := NewNormalizer(logger)
	defer normalizer.Close(context.Background())

	ctx := context.Background()

	// Create two identical results
	result1 := &types.ProbeResult{
		Success:    true,
		RTT:        100 * time.Millisecond,
		Timestamp:  time.Now(),
		Protocol:   types.ProbeTypeTCP,
		TargetIP:   "192.168.1.1",
		TargetPort: 80,
		SourceIP:   "192.168.1.2",
	}

	result2 := &types.ProbeResult{
		Success:    true,
		RTT:        100 * time.Millisecond,
		Timestamp:  time.Now(),
		Protocol:   types.ProbeTypeTCP,
		TargetIP:   "192.168.1.1",
		TargetPort: 80,
		SourceIP:   "192.168.1.2",
	}

	event1, _ := normalizer.Normalize(ctx, result1)
	event2, _ := normalizer.Normalize(ctx, result2)

	if event1.SeriesID != event2.SeriesID {
		t.Error("Same results should generate same SeriesID")
	}

	// Different results should generate different SeriesID
	result3 := &types.ProbeResult{
		Success:    true,
		RTT:        100 * time.Millisecond,
		Timestamp:  time.Now(),
		Protocol:   types.ProbeTypeTCP,
		TargetIP:   "192.168.1.2", // Different IP
		TargetPort: 80,
		SourceIP:   "192.168.1.2",
	}

	event3, _ := normalizer.Normalize(ctx, result3)
	if event1.SeriesID == event3.SeriesID {
		t.Error("Different results should generate different SeriesID")
	}
}

// newTestDedupCache builds a DedupCache without starting the background
// cleanup goroutine, so tests can drive cleanupLoop deterministically with a
// controllable ticker and context.
func newTestDedupCache(window time.Duration, ticker *time.Ticker) *DedupCache {
	return &DedupCache{
		cache:         make(map[string]time.Time),
		window:        window,
		cleanupTicker: ticker,
	}
}

func TestDedupCache_CheckAndMark(t *testing.T) {
	// Use the no-goroutine test constructor so we don't leak an unstoppable
	// cleanupLoop goroutine (and its 1-minute ticker) for the whole test run.
	d := newTestDedupCache(time.Minute, time.NewTicker(time.Hour))
	defer d.cleanupTicker.Stop()

	now := time.Now()
	if d.Check("series-a", now) {
		t.Error("unmarked series should not be a duplicate")
	}

	d.Mark("series-a", now)
	if !d.Check("series-a", now.Add(time.Second)) {
		t.Error("marked series within window should be a duplicate")
	}

	// Outside the dedup window it should no longer be considered a duplicate.
	if d.Check("series-a", now.Add(2*time.Minute)) {
		t.Error("marked series outside window should not be a duplicate")
	}
}

func TestDedupCache_Cleanup(t *testing.T) {
	// A comfortably large window plus a stale entry backdated well beyond it
	// keeps Cleanup's expiry decision independent of test execution speed: the
	// fresh entry can never age out mid-test, and the stale one is always past
	// the window regardless of how long the test takes to run.
	d := newTestDedupCache(10*time.Second, time.NewTicker(time.Hour))
	defer d.cleanupTicker.Stop()

	now := time.Now()
	// Stale entry (backdated far past the window) and a fresh entry.
	d.Mark("stale", now.Add(-time.Hour))
	d.Mark("fresh", now)

	d.Cleanup(context.Background())

	d.mu.RLock()
	_, staleExists := d.cache["stale"]
	_, freshExists := d.cache["fresh"]
	d.mu.RUnlock()

	if staleExists {
		t.Error("stale entry should have been removed by Cleanup")
	}
	if !freshExists {
		t.Error("fresh entry should have been retained by Cleanup")
	}
}

// TestDedupCache_CleanupLoop drives cleanupLoop directly: the ticker branch
// triggers a real Cleanup (removing a stale entry), and cancelling the context
// exercises the ctx.Done() return path.
func TestDedupCache_CleanupLoop(t *testing.T) {
	ticker := time.NewTicker(5 * time.Millisecond)
	d := newTestDedupCache(10*time.Millisecond, ticker)
	defer ticker.Stop()

	// Insert a stale entry that the loop's Cleanup should evict.
	d.Mark("stale", time.Now().Add(-time.Second))

	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		d.cleanupLoop(ctx)
	}()

	// Wait until the ticker-driven Cleanup removes the stale entry.
	deadline := time.After(2 * time.Second)
	for {
		d.mu.RLock()
		_, exists := d.cache["stale"]
		d.mu.RUnlock()
		if !exists {
			break
		}
		select {
		case <-deadline:
			cancel()
			wg.Wait()
			t.Fatal("cleanupLoop did not evict stale entry within deadline")
		case <-time.After(2 * time.Millisecond):
		}
	}

	// Cancel to exercise the ctx.Done() branch and ensure the loop returns.
	cancel()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("cleanupLoop did not return after context cancellation")
	}
}
