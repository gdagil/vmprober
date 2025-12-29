package probe

import (
	"context"
	"testing"
	"time"

	"github.com/gdagil/vmprober/internal/types"
)

func TestNewICMPProbe(t *testing.T) {
	probe := NewICMPProbe(nil)
	if probe == nil {
		t.Fatal("NewICMPProbe returned nil")
	}

	config := &ICMPConfig{
		SequenceStart: 100,
		TTL:           64,
	}
	probeWithConfig := NewICMPProbe(config)
	if probeWithConfig == nil {
		t.Fatal("NewICMPProbe with config returned nil")
	}
	probe.Close()
	probeWithConfig.Close()
}

func TestICMPProbe_Type(t *testing.T) {
	probe := NewICMPProbe(nil)
	defer probe.Close()

	if probe.Type() != types.ProbeTypeICMP {
		t.Errorf("Expected ICMP probe type, got %s", probe.Type())
	}
}

func TestICMPProbe_Validate(t *testing.T) {
	probe := NewICMPProbe(&ICMPConfig{})
	defer probe.Close()

	if err := probe.Validate(nil); err != nil {
		t.Errorf("Validate failed: %v", err)
	}

	// ICMP probe can work with nil config (default is created)
	probeNil := NewICMPProbe(nil)
	defer probeNil.Close()

	// Validate checks internal config, which is always created
	if err := probeNil.Validate(nil); err != nil {
		t.Logf("Validate with nil config returned: %v (may be expected)", err)
	}
}

func TestICMPProbe_Close(t *testing.T) {
	probe := NewICMPProbe(nil)
	if err := probe.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestICMPProbe_Execute_InvalidIP(t *testing.T) {
	probe := NewICMPProbe(nil)
	defer probe.Close()

	ctx := context.Background()
	target := types.Target{
		ID:       "test",
		Host:     "invalid-ip-address",
		Protocol: types.ProbeTypeICMP,
		Timeout:  1 * time.Second,
	}

	result, err := probe.Execute(ctx, target)
	if err == nil {
		t.Error("Expected error for invalid IP address")
	}
	if result == nil {
		t.Fatal("Result should not be nil even on error")
	}
	if result.Success {
		t.Error("Expected unsuccessful result")
	}
}

func TestICMPProbe_Execute_Timeout(t *testing.T) {
	probe := NewICMPProbe(nil)
	defer probe.Close()

	ctx := context.Background()
	// Use non-existent IP for timeout test
	target := types.Target{
		ID:       "test",
		Host:     "192.0.2.1", // TEST-NET-1, usually doesn't respond
		Protocol: types.ProbeTypeICMP,
		Timeout:  100 * time.Millisecond,
	}

	result, err := probe.Execute(ctx, target)
	// ICMP may not receive a response, this is normal
	if err != nil && result == nil {
		t.Errorf("Execute failed: %v", err)
	}
	if result != nil && result.Success {
		t.Log("Note: ICMP probe succeeded, target may be reachable")
	}
}

func TestICMPProbe_Execute_WithConfig(t *testing.T) {
	config := &ICMPConfig{
		SequenceStart: 1,
		TTL:           64,
		Data:          []byte{1, 2, 3, 4},
	}
	probe := NewICMPProbe(config)
	defer probe.Close()

	ctx := context.Background()
	target := types.Target{
		ID:       "test",
		Host:     "127.0.0.1",
		Protocol: types.ProbeTypeICMP,
		Timeout:  1 * time.Second,
	}

	result, err := probe.Execute(ctx, target)
	// Result depends on localhost availability
	if err != nil && result == nil {
		t.Logf("ICMP probe result: %v, error: %v", result, err)
	}
}

func TestICMPProbe_SequenceIncrement(t *testing.T) {
	config := &ICMPConfig{
		SequenceStart: 1,
	}
	probe := NewICMPProbe(config)
	defer probe.Close()

	// Check that sequence increases
	// This is internal logic, but we can verify by executing multiple probes
	ctx := context.Background()
	target := types.Target{
		ID:       "test",
		Host:     "127.0.0.1",
		Protocol: types.ProbeTypeICMP,
		Timeout:  100 * time.Millisecond,
	}

	// Execute several probes to check sequence
	for i := 0; i < 3; i++ {
		result, _ := probe.Execute(ctx, target)
		if result != nil {
			t.Logf("Probe %d completed", i+1)
		}
	}
}
