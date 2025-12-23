package normalizer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vmprober/vmprober/internal/types"
)

// Normalizer интерфейс для нормализации результатов проб
type Normalizer interface {
	// Normalize нормализует результат пробы
	Normalize(ctx context.Context, result *types.ProbeResult) (*types.NormalizedEvent, error)

	// NormalizeBatch нормализует пакет результатов
	NormalizeBatch(ctx context.Context, results []*types.ProbeResult) ([]*types.NormalizedEvent, error)

	// Dedup проверяет на дубликаты
	Dedup(ctx context.Context, event *types.NormalizedEvent) (bool, error)

	// Enrich обогащает событие дополнительной информацией
	Enrich(ctx context.Context, event *types.NormalizedEvent) error

	// GetStats возвращает статистику нормализатора
	GetStats() *NormalizerStats

	// Close закрывает нормализатор
	Close(ctx context.Context) error
}

// NormalizerStats статистика нормализатора
type NormalizerStats struct {
	TotalNormalized int64         `json:"total_normalized"`
	TotalDeduped    int64         `json:"total_deduped"`
	TotalEnriched   int64         `json:"total_enriched"`
	AvgNormalizeTime time.Duration `json:"avg_normalize_time"`
}

// DefaultNormalizer реализация нормализатора
type DefaultNormalizer struct {
	dedupCache  *DedupCache
	enricher    *DataEnricher
	mu          sync.RWMutex
	logger      *logrus.Logger
	stats       *NormalizerStats
}

// NewNormalizer создает новый нормализатор
func NewNormalizer(logger *logrus.Logger) Normalizer {
	return &DefaultNormalizer{
		dedupCache: NewDedupCache(10 * time.Minute),
		enricher:   NewDataEnricher(logger),
		logger:     logger,
		stats:      &NormalizerStats{},
	}
}

// Normalize нормализует результат пробы
func (n *DefaultNormalizer) Normalize(ctx context.Context, result *types.ProbeResult) (*types.NormalizedEvent, error) {
	start := time.Now()

	// Стандартизация результата
	event := n.standardize(result)

	// Обогащение данными
	if err := n.Enrich(ctx, event); err != nil {
		n.logger.WithError(err).Warn("Failed to enrich event")
	}

	// Обновление статистики
	n.mu.Lock()
	n.stats.TotalNormalized++
	n.stats.AvgNormalizeTime = time.Since(start)
	n.mu.Unlock()

	return event, nil
}

// NormalizeBatch нормализует пакет результатов
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

// Dedup проверяет на дубликаты
func (n *DefaultNormalizer) Dedup(ctx context.Context, event *types.NormalizedEvent) (bool, error) {
	isDuplicate := n.dedupCache.Check(event.SeriesID, event.Timestamp)

	if isDuplicate {
		n.mu.Lock()
		n.stats.TotalDeduped++
		n.mu.Unlock()
		return true, nil
	}

	// Отметка события как обработанного
	n.dedupCache.Mark(event.SeriesID, event.Timestamp)

	return false, nil
}

// Enrich обогащает событие дополнительной информацией
func (n *DefaultNormalizer) Enrich(ctx context.Context, event *types.NormalizedEvent) error {
	n.enricher.Enrich(event)

	n.mu.Lock()
	n.stats.TotalEnriched++
	n.mu.Unlock()

	return nil
}

// GetStats возвращает статистику нормализатора
func (n *DefaultNormalizer) GetStats() *NormalizerStats {
	n.mu.RLock()
	defer n.mu.RUnlock()

	stats := *n.stats
	return &stats
}

// Close закрывает нормализатор
func (n *DefaultNormalizer) Close(ctx context.Context) error {
	n.dedupCache.Cleanup(ctx)
	return nil
}

// standardize стандартизирует результат пробы
func (n *DefaultNormalizer) standardize(result *types.ProbeResult) *types.NormalizedEvent {
	event := &types.NormalizedEvent{
		Timestamp: result.Timestamp,
		Metrics:   make(map[string]float64),
		Labels:    make(map[string]string),
		Tags:      make([]string, 0),
		Metadata:  make(map[string]interface{}),
	}

	// Генерация SeriesID
	event.SeriesID = n.generateSeriesID(result)

	// Установка меток
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

	// Установка метрик
	event.Metrics["rtt_seconds"] = result.RTT.Seconds()
	if result.Success {
		event.Metrics["success"] = 1.0
	} else {
		event.Metrics["success"] = 0.0
	}

	// Установка тегов
	event.Tags = append(event.Tags, string(result.Protocol))
	if result.Success {
		event.Tags = append(event.Tags, "success")
	} else {
		event.Tags = append(event.Tags, "failure")
	}

	// Установка метаданных
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

// generateSeriesID генерирует уникальный ID для серии метрик
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

