package normalizer

import (
	"context"
	"sync"
	"time"
)

// DedupCache cache for deduplication
type DedupCache struct {
	cache      map[string]time.Time
	window     time.Duration
	mu         sync.RWMutex
	cleanupTicker *time.Ticker
}

// NewDedupCache creates a new deduplication cache
func NewDedupCache(window time.Duration) *DedupCache {
	cache := &DedupCache{
		cache:      make(map[string]time.Time),
		window:     window,
		cleanupTicker: time.NewTicker(1 * time.Minute),
	}

	// Start cleanup
	go cache.cleanupLoop(context.Background())

	return cache
}

// Check checks if the event is a duplicate
func (d *DedupCache) Check(seriesID string, timestamp time.Time) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()

	lastSeen, exists := d.cache[seriesID]
	if !exists {
		return false
	}

	// Check deduplication window
	return timestamp.Sub(lastSeen) < d.window
}

// Mark marks the event as processed
func (d *DedupCache) Mark(seriesID string, timestamp time.Time) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.cache[seriesID] = timestamp
}

// Cleanup clears outdated entries
func (d *DedupCache) Cleanup(ctx context.Context) {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()
	for seriesID, lastSeen := range d.cache {
		if now.Sub(lastSeen) > d.window {
			delete(d.cache, seriesID)
		}
	}
}

// cleanupLoop cache cleanup loop
func (d *DedupCache) cleanupLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-d.cleanupTicker.C:
			d.Cleanup(ctx)
		}
	}
}


