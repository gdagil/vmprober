package adapter

import (
	"strings"
	"testing"
	"time"

	"github.com/gdagil/vmprober/internal/types"
)

func TestPrometheusFormatter_Format_Gauge(t *testing.T) {
	formatter := NewPrometheusFormatter()

	metrics := []types.Metric{
		{
			Name:      "test_gauge",
			Value:     42.5,
			Timestamp: time.Now(),
			Labels: map[string]string{
				"label1": "value1",
				"label2": "value2",
			},
			Type: types.MetricTypeGauge,
		},
	}

	data, err := formatter.Format(metrics)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	result := string(data)
	if !strings.Contains(result, "test_gauge") {
		t.Error("Formatted data doesn't contain metric name")
	}
	if !strings.Contains(result, "42.5") {
		t.Error("Formatted data doesn't contain metric value")
	}
	if !strings.Contains(result, "label1") {
		t.Error("Formatted data doesn't contain labels")
	}
}

func TestPrometheusFormatter_Format_Counter(t *testing.T) {
	formatter := NewPrometheusFormatter()

	metrics := []types.Metric{
		{
			Name:      "test_counter",
			Value:     100.0,
			Timestamp: time.Now(),
			Type:      types.MetricTypeCounter,
		},
	}

	data, err := formatter.Format(metrics)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	result := string(data)
	if !strings.Contains(result, "test_counter") {
		t.Error("Formatted data doesn't contain metric name")
	}
	if !strings.Contains(result, "100") {
		t.Error("Formatted data doesn't contain metric value")
	}
}

func TestPrometheusFormatter_Format_Histogram(t *testing.T) {
	formatter := NewPrometheusFormatter()

	metrics := []types.Metric{
		{
			Name:      "test_histogram",
			Value:     0,
			Timestamp: time.Now(),
			Type:      types.MetricTypeHistogram,
			Count:     10,
			Sum:       100.0,
			Buckets:   []float64{0.1, 0.5, 1.0, 2.5, 5.0},
		},
	}

	data, err := formatter.Format(metrics)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	result := string(data)
	if !strings.Contains(result, "test_histogram_count") {
		t.Error("Formatted data doesn't contain histogram count")
	}
	if !strings.Contains(result, "test_histogram_sum") {
		t.Error("Formatted data doesn't contain histogram sum")
	}
	if !strings.Contains(result, "test_histogram_bucket") {
		t.Error("Formatted data doesn't contain histogram buckets")
	}
}

func TestPrometheusFormatter_Format_Summary(t *testing.T) {
	formatter := NewPrometheusFormatter()

	metrics := []types.Metric{
		{
			Name:      "test_summary",
			Value:     0,
			Timestamp: time.Now(),
			Type:      types.MetricTypeSummary,
			Count:     5,
			Sum:       50.0,
			Quantiles: map[float64]float64{
				0.5:  10.0,
				0.9:  20.0,
				0.99: 30.0,
			},
		},
	}

	data, err := formatter.Format(metrics)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	result := string(data)
	if !strings.Contains(result, "test_summary_count") {
		t.Error("Formatted data doesn't contain summary count")
	}
	if !strings.Contains(result, "test_summary_sum") {
		t.Error("Formatted data doesn't contain summary sum")
	}
}

func TestPrometheusFormatter_Format_MultipleMetrics(t *testing.T) {
	formatter := NewPrometheusFormatter()

	metrics := []types.Metric{
		{
			Name:      "metric1",
			Value:     1.0,
			Timestamp: time.Now(),
			Type:      types.MetricTypeGauge,
		},
		{
			Name:      "metric2",
			Value:     2.0,
			Timestamp: time.Now(),
			Type:      types.MetricTypeGauge,
		},
	}

	data, err := formatter.Format(metrics)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	result := string(data)
	if !strings.Contains(result, "metric1") {
		t.Error("Formatted data doesn't contain metric1")
	}
	if !strings.Contains(result, "metric2") {
		t.Error("Formatted data doesn't contain metric2")
	}
}

func TestPrometheusFormatter_Format_EmptyMetrics(t *testing.T) {
	formatter := NewPrometheusFormatter()

	metrics := []types.Metric{}

	data, err := formatter.Format(metrics)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	// For empty metrics list, output may be empty
	if data == nil {
		t.Error("Data should not be nil")
	}
	// Empty byte slice is valid for empty metrics
}
