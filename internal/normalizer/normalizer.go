package normalizer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/gdagil/vmprober/internal/types"
)

// Normalizer interface for normalizing probe results
type Normalizer interface {
	// Normalize normalizes probe result
	Normalize(ctx context.Context, result *types.ProbeResult) (*types.NormalizedEvent, error)

	// NormalizeBatch normalizes a batch of results
	NormalizeBatch(ctx context.Context, results []*types.ProbeResult) ([]*types.NormalizedEvent, error)

	// Dedup checks for duplicates
	Dedup(ctx context.Context, event *types.NormalizedEvent) (bool, error)

	// Enrich enriches event with additional information
	Enrich(ctx context.Context, event *types.NormalizedEvent) error

	// GetStats returns normalizer statistics
	GetStats() *NormalizerStats

	// Close closes the normalizer
	Close(ctx context.Context) error
}

// NormalizerStats normalizer statistics
type NormalizerStats struct {
	TotalNormalized  int64         `json:"total_normalized"`
	TotalDeduped     int64         `json:"total_deduped"`
	TotalEnriched    int64         `json:"total_enriched"`
	AvgNormalizeTime time.Duration `json:"avg_normalize_time"`
}

// DefaultNormalizer normalizer implementation
type DefaultNormalizer struct {
	dedupCache *DedupCache
	enricher   *DataEnricher
	mu         sync.RWMutex
	logger     *logrus.Logger
	stats      *NormalizerStats
}

// NewNormalizer creates a new normalizer
func NewNormalizer(logger *logrus.Logger) Normalizer {
	return &DefaultNormalizer{
		dedupCache: NewDedupCache(10 * time.Minute),
		enricher:   NewDataEnricher(logger),
		logger:     logger,
		stats:      &NormalizerStats{},
	}
}

// Normalize normalizes probe result
func (n *DefaultNormalizer) Normalize(ctx context.Context, result *types.ProbeResult) (*types.NormalizedEvent, error) {
	start := time.Now()

	// Standardize result
	event := n.standardize(result)

	// Enrich with data
	if err := n.Enrich(ctx, event); err != nil {
		n.logger.WithError(err).Warn("Failed to enrich event")
	}

	// Update statistics
	n.mu.Lock()
	n.stats.TotalNormalized++
	n.stats.AvgNormalizeTime = time.Since(start)
	n.mu.Unlock()

	return event, nil
}

// NormalizeBatch normalizes a batch of results
func (n *DefaultNormalizer) NormalizeBatch(ctx context.Context, results []*types.ProbeResult) ([]*types.NormalizedEvent, error) {
	events := make([]*types.NormalizedEvent, 0, len(results))

	for _, result := range results {
		event, err := n.Normalize(ctx, result)
		if err != nil {
			n.logger.WithError(err).Warn("Failed to normalize result")
			continue
		}
		events = append(events, event)
	}

	return events, nil
}

// Dedup checks for duplicates
func (n *DefaultNormalizer) Dedup(ctx context.Context, event *types.NormalizedEvent) (bool, error) {
	isDuplicate := n.dedupCache.Check(event.SeriesID, event.Timestamp)

	if isDuplicate {
		n.mu.Lock()
		n.stats.TotalDeduped++
		n.mu.Unlock()
		return true, nil
	}

	// Mark event as processed
	n.dedupCache.Mark(event.SeriesID, event.Timestamp)

	return false, nil
}

// Enrich enriches event with additional information
func (n *DefaultNormalizer) Enrich(ctx context.Context, event *types.NormalizedEvent) error {
	n.enricher.Enrich(event)

	n.mu.Lock()
	n.stats.TotalEnriched++
	n.mu.Unlock()

	return nil
}

// GetStats returns normalizer statistics
func (n *DefaultNormalizer) GetStats() *NormalizerStats {
	n.mu.RLock()
	defer n.mu.RUnlock()

	stats := *n.stats
	return &stats
}

// Close closes the normalizer
func (n *DefaultNormalizer) Close(ctx context.Context) error {
	n.dedupCache.Cleanup(ctx)
	return nil
}

// standardize standardizes probe result
func (n *DefaultNormalizer) standardize(result *types.ProbeResult) *types.NormalizedEvent {
	event := &types.NormalizedEvent{
		Timestamp: result.Timestamp,
		Metrics:   make(map[string]float64),
		Labels:    make(map[string]string),
		Tags:      make([]string, 0),
		Metadata:  make(map[string]interface{}),
	}

	// Generate SeriesID
	event.SeriesID = n.generateSeriesID(result)

	// Set labels
	event.Labels["protocol"] = string(result.Protocol)
	event.Labels["target_ip"] = result.TargetIP
	if result.TargetPort > 0 {
		event.Labels["target_port"] = fmt.Sprintf("%d", result.TargetPort)
	}
	if result.SourceIP != "" {
		event.Labels["source_ip"] = result.SourceIP
	}
	if result.Role != "" {
		event.Labels["role"] = result.Role
	}
	if result.SocketFamily != "" {
		event.Labels["socket_family"] = result.SocketFamily
	}

	// Set status label
	if result.Success {
		event.Labels["status"] = "success"
	} else {
		event.Labels["status"] = "failed"
	}

	// Set metrics with vmprober_ prefix
	// RTT metric - always sent, will be 0 for failed
	event.Metrics["vmprober_probe_rtt_seconds"] = result.RTT.Seconds()

	// Probe result - 1.0 for successful, 1.0 for failed (for counting)
	event.Metrics["vmprober_probe_result"] = 1.0

	// Probe duration (total execution time)
	event.Metrics["vmprober_probe_duration_seconds"] = result.RTT.Seconds()

	// NB: the attempt number is deliberately NOT a metric — the name
	// vmprober_probe_attempts_total belongs to the collector's cumulative
	// counter, while result.Attempt (retry number) is not monotonic; pushing
	// it would poison rate() over the real counter's name. It is available in
	// event.Metadata["attempt"].

	// Set tags
	event.Tags = append(event.Tags, string(result.Protocol))
	if result.Success {
		event.Tags = append(event.Tags, "success")
	} else {
		event.Tags = append(event.Tags, "failure")
	}

	// Additional metrics
	// DNS lookup time (if available)
	if result.DNSResult != nil {
		event.Metrics["vmprober_probe_dns_lookup_seconds"] = result.DNSResult.LookupTime.Seconds()
		// Number of resolved IP addresses
		event.Metrics["vmprober_probe_dns_resolved_ips"] = float64(len(result.DNSResult.ResolvedIPs))
	}

	// Set metadata
	event.Metadata["attempt"] = result.Attempt
	if result.Error != "" {
		event.Metadata["error"] = result.Error
	}
	if result.DNSResult != nil {
		event.Metadata["dns_lookup_time"] = result.DNSResult.LookupTime.Seconds()
		event.Metadata["dns_resolved_ips"] = result.DNSResult.ResolvedIPs
	}

	return event
}

// generateSeriesID generates unique ID for metric series
func (n *DefaultNormalizer) generateSeriesID(result *types.ProbeResult) string {
	key := fmt.Sprintf("%s:%s:%d:%s",
		result.Protocol,
		result.TargetIP,
		result.TargetPort,
		result.SourceIP,
	)

	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])[:16]
}
