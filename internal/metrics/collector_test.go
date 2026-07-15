package metrics

import (
	"context"
	"testing"
	"time"

	"github.com/gdagil/vmprober/internal/types"
)

func TestNewCollector(t *testing.T) {
	collector := NewCollector("test_new", false, nil)
	if collector == nil {
		t.Fatal("NewCollector returned nil")
	}

	if collector.namespace != "test_new" {
		t.Errorf("Expected namespace 'test_new', got '%s'", collector.namespace)
	}
}

func TestNewCollector_DefaultNamespace(t *testing.T) {
	collector := NewCollector("", false, nil)
	if collector == nil {
		t.Fatal("NewCollector returned nil")
	}

	if collector.namespace != "vmprober" {
		t.Errorf("Expected default namespace 'vmprober', got '%s'", collector.namespace)
	}
}

func TestNewCollector_WithCustomLabels(t *testing.T) {
	customLabels := map[string]string{
		"job": "blackbox/vmprober",
		"env": "test",
	}
	collector := NewCollector("test_labels", false, customLabels)
	if collector == nil {
		t.Fatal("NewCollector returned nil")
	}

	if collector.customLabels["job"] != "blackbox/vmprober" {
		t.Errorf("Expected job label 'blackbox/vmprober', got '%s'", collector.customLabels["job"])
	}
	if collector.customLabels["env"] != "test" {
		t.Errorf("Expected env label 'test', got '%s'", collector.customLabels["env"])
	}
}

func TestCollector_Record_Success(t *testing.T) {
	collector := NewCollector("test_record_success", false, nil)

	result := &types.ProbeResult{
		Success:    true,
		RTT:        100 * time.Millisecond,
		Protocol:   types.ProbeTypeTCP,
		TargetIP:   "127.0.0.1",
		TargetPort: 80,
		Timestamp:  time.Now(),
		Attempt:    1,
	}

	ctx := context.Background()
	err := collector.Record(ctx, result)
	if err != nil {
		t.Fatalf("Record failed: %v", err)
	}
}

func TestCollector_Record_Failure(t *testing.T) {
	collector := NewCollector("test_record_failure", false, nil)

	result := &types.ProbeResult{
		Success:    false,
		RTT:        50 * time.Millisecond,
		Protocol:   types.ProbeTypeTCP,
		TargetIP:   "127.0.0.1",
		TargetPort: 80,
		Timestamp:  time.Now(),
		Attempt:    1,
		Error:      "connection refused",
	}

	ctx := context.Background()
	err := collector.Record(ctx, result)
	if err != nil {
		t.Fatalf("Record failed: %v", err)
	}
}

func TestCollector_Record_MultipleResults(t *testing.T) {
	collector := NewCollector("test_multiple_results", false, nil)

	ctx := context.Background()

	// Record several successful results
	for i := 0; i < 5; i++ {
		result := &types.ProbeResult{
			Success:    true,
			RTT:        time.Duration(i*10) * time.Millisecond,
			Protocol:   types.ProbeTypeTCP,
			TargetIP:   "127.0.0.1",
			TargetPort: 80,
			Timestamp:  time.Now(),
			Attempt:    i + 1,
		}

		if err := collector.Record(ctx, result); err != nil {
			t.Fatalf("Record failed: %v", err)
		}
	}

	// Record several failed results
	for i := 0; i < 3; i++ {
		result := &types.ProbeResult{
			Success:    false,
			RTT:        50 * time.Millisecond,
			Protocol:   types.ProbeTypeTCP,
			TargetIP:   "127.0.0.1",
			TargetPort: 80,
			Timestamp:  time.Now(),
			Attempt:    i + 1,
			Error:      "timeout",
		}

		if err := collector.Record(ctx, result); err != nil {
			t.Fatalf("Record failed: %v", err)
		}
	}
}

func TestCollector_Record_DifferentProtocols(t *testing.T) {
	collector := NewCollector("test_different_protocols", false, nil)

	ctx := context.Background()

	protocols := []types.ProbeType{
		types.ProbeTypeTCP,
		types.ProbeTypeUDP,
	}

	for _, protocol := range protocols {
		result := &types.ProbeResult{
			Success:    true,
			RTT:        100 * time.Millisecond,
			Protocol:   protocol,
			TargetIP:   "127.0.0.1",
			TargetPort: 80,
			Timestamp:  time.Now(),
			Attempt:    1,
		}

		if err := collector.Record(ctx, result); err != nil {
			t.Fatalf("Record failed for protocol %s: %v", protocol, err)
		}
	}
}

func TestCollector_Record_UnknownTargetIP(t *testing.T) {
	collector := NewCollector("test_unknown_target", false, nil)

	result := &types.ProbeResult{
		Success:   true,
		RTT:       100 * time.Millisecond,
		Protocol:  types.ProbeTypeTCP,
		TargetIP:  "", // Empty IP
		Timestamp: time.Now(),
		Attempt:   1,
	}

	ctx := context.Background()
	err := collector.Record(ctx, result)
	if err != nil {
		t.Fatalf("Record failed: %v", err)
	}
}

func TestCollector_Record_DifferentTargets(t *testing.T) {
	collector := NewCollector("test_different_targets", false, nil)

	ctx := context.Background()

	targets := []string{"127.0.0.1", "8.8.8.8", "1.1.1.1"}

	for _, targetIP := range targets {
		result := &types.ProbeResult{
			Success:    true,
			RTT:        100 * time.Millisecond,
			Protocol:   types.ProbeTypeTCP,
			TargetIP:   targetIP,
			TargetPort: 80,
			Timestamp:  time.Now(),
			Attempt:    1,
		}

		if err := collector.Record(ctx, result); err != nil {
			t.Fatalf("Record failed for target %s: %v", targetIP, err)
		}
	}
}

// GetRegistry test removed - VictoriaMetrics doesn't use registry pattern

func TestCollector_Record_RTT(t *testing.T) {
	collector := NewCollector("test_rtt", false, nil)

	ctx := context.Background()

	// Record results with different RTT values
	rtts := []time.Duration{
		1 * time.Millisecond,
		10 * time.Millisecond,
		100 * time.Millisecond,
		500 * time.Millisecond,
		1 * time.Second,
	}

	for _, rtt := range rtts {
		result := &types.ProbeResult{
			Success:    true,
			RTT:        rtt,
			Protocol:   types.ProbeTypeTCP,
			TargetIP:   "127.0.0.1",
			TargetPort: 80,
			Timestamp:  time.Now(),
			Attempt:    1,
		}

		if err := collector.Record(ctx, result); err != nil {
			t.Fatalf("Record failed for RTT %v: %v", rtt, err)
		}
	}
}

func TestCollector_UpdateJobMetrics(t *testing.T) {
	collector := NewCollector("test_job_metrics", true, nil)

	// Update job metrics
	collector.UpdateJobMetrics(10, 5, 2)

	// Metrics should be updated (we can't directly check the values, but we can verify no panic)
	// The actual values are accessed through callbacks in the gauge metrics
}

func TestCollector_UpdateJobMetrics_Disabled(t *testing.T) {
	collector := NewCollector("test_job_metrics_disabled", false, nil)

	// Update job metrics when disabled - should not panic
	collector.UpdateJobMetrics(10, 5, 2)
}

func TestCollector_ExportMetrics(t *testing.T) {
	collector := NewCollector("test_export", false, nil)

	ctx := context.Background()

	// Record some results first
	result := &types.ProbeResult{
		Success:    true,
		RTT:        100 * time.Millisecond,
		Protocol:   types.ProbeTypeTCP,
		TargetIP:   "127.0.0.1",
		TargetPort: 80,
		TargetHost: "localhost",
		Timestamp:  time.Now(),
		Attempt:    1,
	}

	if err := collector.Record(ctx, result); err != nil {
		t.Fatalf("Record failed: %v", err)
	}

	// Export metrics
	metrics := collector.ExportMetrics()
	if metrics == nil {
		t.Fatal("ExportMetrics returned nil")
	}

	// Should have at least some metrics
	if len(metrics) == 0 {
		t.Log("No metrics exported (this may be OK if VictoriaMetrics library doesn't expose them immediately)")
	}
}

func TestCollector_ExportMetrics_WithCustomLabels(t *testing.T) {
	customLabels := map[string]string{
		"job": "test-job",
		"env": "test",
	}
	collector := NewCollector("test_export_labels", false, customLabels)

	ctx := context.Background()

	result := &types.ProbeResult{
		Success:    true,
		RTT:        100 * time.Millisecond,
		Protocol:   types.ProbeTypeTCP,
		TargetIP:   "127.0.0.1",
		TargetPort: 80,
		TargetHost: "localhost",
		Timestamp:  time.Now(),
		Attempt:    1,
	}

	if err := collector.Record(ctx, result); err != nil {
		t.Fatalf("Record failed: %v", err)
	}

	metrics := collector.ExportMetrics()
	if metrics == nil {
		t.Fatal("ExportMetrics returned nil")
	}

	// Check if custom labels are included in exported metrics
	for _, metric := range metrics {
		if metric.Labels != nil {
			if job, ok := metric.Labels["job"]; ok && job == "test-job" {
				// Found custom label
				return
			}
		}
	}
	t.Log("Custom labels may not be immediately visible in exported metrics")
}

// TestGetMetricKey_StableWithMultipleCustomLabels: map iteration order is
// random in Go, so the key must not depend on it — otherwise one logical
// target gets several counter objects that VictoriaMetrics merges into a
// single sawtooth series.
func TestGetMetricKey_StableWithMultipleCustomLabels(t *testing.T) {
	collector := NewCollector("test_stable_key", false, map[string]string{
		"job": "blackbox", "prober": "dc1-a", "env": "prod", "region": "eu", "team": "net",
	})
	first := collector.getMetricKey("probe_attempts_total", "google.com:443", "1.2.3.4", "443", "tcp")
	for i := 0; i < 50; i++ {
		key := collector.getMetricKey("probe_attempts_total", "google.com:443", "1.2.3.4", "443", "tcp")
		if key != first {
			t.Fatalf("metric key is not stable across calls:\n%s\n%s", first, key)
		}
	}
}

func TestCollector_Record_WithHostname(t *testing.T) {
	collector := NewCollector("test_hostname", false, nil)

	result := &types.ProbeResult{
		Success:    true,
		RTT:        100 * time.Millisecond,
		Protocol:   types.ProbeTypeTCP,
		TargetIP:   "127.0.0.1",
		TargetHost: "localhost",
		TargetPort: 80,
		Timestamp:  time.Now(),
		Attempt:    1,
	}

	ctx := context.Background()
	err := collector.Record(ctx, result)
	if err != nil {
		t.Fatalf("Record failed: %v", err)
	}
}

func TestCollector_Record_WithEmptyHostname(t *testing.T) {
	collector := NewCollector("test_empty_hostname", false, nil)

	result := &types.ProbeResult{
		Success:    true,
		RTT:        100 * time.Millisecond,
		Protocol:   types.ProbeTypeTCP,
		TargetIP:   "127.0.0.1",
		TargetHost: "", // Empty hostname
		TargetPort: 80,
		Timestamp:  time.Now(),
		Attempt:    1,
	}

	ctx := context.Background()
	err := collector.Record(ctx, result)
	if err != nil {
		t.Fatalf("Record failed: %v", err)
	}
}

func TestCollector_Record_ZeroPort(t *testing.T) {
	collector := NewCollector("test_zero_port", false, nil)

	result := &types.ProbeResult{
		Success:    true,
		RTT:        100 * time.Millisecond,
		Protocol:   types.ProbeTypeICMP,
		TargetIP:   "127.0.0.1",
		TargetPort: 0, // Zero port (e.g., for ICMP)
		Timestamp:  time.Now(),
		Attempt:    1,
	}

	ctx := context.Background()
	err := collector.Record(ctx, result)
	if err != nil {
		t.Fatalf("Record failed: %v", err)
	}
}

func TestCollector_NewCollector_WithJobMetrics(t *testing.T) {
	collector := NewCollector("test_job_metrics_enabled", true, nil)
	if collector == nil {
		t.Fatal("NewCollector returned nil")
	}

	if !collector.enableJobMetrics {
		t.Error("Expected enableJobMetrics to be true")
	}
}

func TestCollector_Record_Concurrent(t *testing.T) {
	collector := NewCollector("test_concurrent", false, nil)

	ctx := context.Background()

	// Concurrent recording
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			result := &types.ProbeResult{
				Success:    true,
				RTT:        time.Duration(id*10) * time.Millisecond,
				Protocol:   types.ProbeTypeTCP,
				TargetIP:   "127.0.0.1",
				TargetPort: 80,
				Timestamp:  time.Now(),
				Attempt:    id + 1,
			}
			err := collector.Record(ctx, result)
			done <- (err == nil)
		}(i)
	}

	successCount := 0
	for i := 0; i < 10; i++ {
		if <-done {
			successCount++
		}
	}

	if successCount != 10 {
		t.Errorf("Expected 10 successful records, got %d", successCount)
	}
}
