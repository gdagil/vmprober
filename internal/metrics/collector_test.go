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
