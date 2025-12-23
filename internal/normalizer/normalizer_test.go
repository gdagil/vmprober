package normalizer

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vmprober/vmprober/internal/types"
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
		Success:   true,
		RTT:       100 * time.Millisecond,
		Attempt:   1,
		Timestamp: time.Now(),
		Protocol:  types.ProbeTypeTCP,
		TargetIP:  "192.168.1.1",
		TargetPort: 80,
		SourceIP:  "192.168.1.2",
		Role:      "client",
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

	// Проверяем наличие основных метрик
	if _, ok := event.Metrics["rtt_seconds"]; !ok {
		t.Error("rtt_seconds metric not found")
	}
	if _, ok := event.Metrics["success"]; !ok {
		t.Error("success metric not found")
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
		Success:   true,
		RTT:       100 * time.Millisecond,
		Timestamp: time.Now(),
		Protocol:  types.ProbeTypeTCP,
		TargetIP:  "192.168.1.1",
		TargetPort: 80,
	}

	event, err := normalizer.Normalize(ctx, result)
	if err != nil {
		t.Fatalf("Normalize failed: %v", err)
	}

	// Первая проверка - не должно быть дубликатом
	isDup, err := normalizer.Dedup(ctx, event)
	if err != nil {
		t.Fatalf("Dedup failed: %v", err)
	}
	if isDup {
		t.Error("First event should not be duplicate")
	}

	// Вторая проверка того же события - должно быть дубликатом
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

	// Enrich вызывается автоматически в Normalize
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

	// Создаем два одинаковых результата
	result1 := &types.ProbeResult{
		Success:   true,
		RTT:       100 * time.Millisecond,
		Timestamp: time.Now(),
		Protocol:  types.ProbeTypeTCP,
		TargetIP:  "192.168.1.1",
		TargetPort: 80,
		SourceIP:  "192.168.1.2",
	}

	result2 := &types.ProbeResult{
		Success:   true,
		RTT:       100 * time.Millisecond,
		Timestamp: time.Now(),
		Protocol:  types.ProbeTypeTCP,
		TargetIP:  "192.168.1.1",
		TargetPort: 80,
		SourceIP:  "192.168.1.2",
	}

	event1, _ := normalizer.Normalize(ctx, result1)
	event2, _ := normalizer.Normalize(ctx, result2)

	if event1.SeriesID != event2.SeriesID {
		t.Error("Same results should generate same SeriesID")
	}

	// Разные результаты должны генерировать разные SeriesID
	result3 := &types.ProbeResult{
		Success:   true,
		RTT:       100 * time.Millisecond,
		Timestamp: time.Now(),
		Protocol:  types.ProbeTypeTCP,
		TargetIP:  "192.168.1.2", // Другой IP
		TargetPort: 80,
		SourceIP:  "192.168.1.2",
	}

	event3, _ := normalizer.Normalize(ctx, result3)
	if event1.SeriesID == event3.SeriesID {
		t.Error("Different results should generate different SeriesID")
	}
}


