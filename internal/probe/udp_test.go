package probe

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gdagil/vmprober/internal/types"
)

func TestNewUDPProbe(t *testing.T) {
	config := &UDPConfig{
		PayloadSize:     64,
		ResponseTimeout: 2 * time.Second,
		MaxPacketSize:   1024,
	}

	probe := NewUDPProbe(config)
	if probe == nil {
		t.Fatal("NewUDPProbe returned nil")
	}

	if probe.Type() != types.ProbeTypeUDP {
		t.Errorf("Expected type UDP, got %s", probe.Type())
	}
}

func TestUDPProbe_Execute_Success(t *testing.T) {
	// Hermetic loopback echo server guarantees a response, so the success path
	// is exercised deterministically (loopback UDP does not drop packets). The
	// server waits echoDelay before replying so the measured RTT has a real,
	// deterministic lower bound even on hosts with a coarse monotonic clock
	// (notably Windows, where a bare RTT > 0 check is flaky).
	const echoDelay = 20 * time.Millisecond
	serverAddr, cleanup := startUDPServer(t, true, echoDelay)
	defer cleanup()

	config := &UDPConfig{
		PayloadSize:     64,
		ResponseTimeout: 2 * time.Second,
		MaxPacketSize:   1024,
	}
	probe := NewUDPProbe(config)
	defer probe.Close()

	target := types.Target{
		Host:     "127.0.0.1",
		Port:     serverAddr.Port,
		Protocol: types.ProbeTypeUDP,
		Timeout:  2 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := probe.Execute(ctx, target)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success, "expected successful probe, error: %s", result.Error)
	// The server slept echoDelay before echoing, so RTT must reflect at least
	// most of that wait. 10ms is a safe deterministic floor below the 20ms delay.
	assert.GreaterOrEqual(t, result.RTT, 10*time.Millisecond)
	assert.Equal(t, types.ProbeTypeUDP, result.Protocol)
	assert.Len(t, result.Payload, 64)
	assert.Equal(t, "client", result.Role)
	assert.Equal(t, serverAddr.Port, result.TargetPort)
}

func TestUDPProbe_Execute_ResponseTimeout(t *testing.T) {
	// A live-but-silent server keeps the socket open (no ICMP port-unreachable),
	// so ReadFromUDP hits the read deadline and the "UDP response timeout" branch
	// is covered deterministically. Timeouts are reported as non-errors for UDP.
	serverAddr, cleanup := startUDPServer(t, false, 0)
	defer cleanup()

	config := &UDPConfig{
		PayloadSize:     32,
		ResponseTimeout: 150 * time.Millisecond,
		MaxPacketSize:   1024,
	}
	probe := NewUDPProbe(config)
	defer probe.Close()

	target := types.Target{
		Host:     "127.0.0.1",
		Port:     serverAddr.Port,
		Protocol: types.ProbeTypeUDP,
		Timeout:  150 * time.Millisecond,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err := probe.Execute(ctx, target)
	require.NoError(t, err) // UDP timeout is not treated as an error
	require.NotNil(t, result)
	assert.False(t, result.Success)
	assert.Equal(t, "UDP response timeout", result.Error)
}

func TestUDPProbe_Execute_DefaultMaxPacketSize(t *testing.T) {
	// MaxPacketSize 0 exercises the default (1500-byte) receive-buffer branch.
	serverAddr, cleanup := startUDPServer(t, true, 0)
	defer cleanup()

	config := &UDPConfig{
		PayloadSize:     16,
		ResponseTimeout: 2 * time.Second,
		MaxPacketSize:   0, // use default buffer
	}
	probe := NewUDPProbe(config)
	defer probe.Close()

	target := types.Target{
		Host:     "127.0.0.1",
		Port:     serverAddr.Port,
		Protocol: types.ProbeTypeUDP,
		Timeout:  2 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := probe.Execute(ctx, target)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success, "expected successful probe, error: %s", result.Error)
}

func TestUDPProbe_Execute_NoResponse(t *testing.T) {
	config := &UDPConfig{
		PayloadSize:     64,
		ResponseTimeout: 100 * time.Millisecond,
		MaxPacketSize:   1024,
	}
	probe := NewUDPProbe(config)
	defer probe.Close()

	// Use a port that doesn't respond
	target := types.Target{
		Host:     "127.0.0.1",
		Port:     65535,
		Protocol: types.ProbeTypeUDP,
		Timeout:  100 * time.Millisecond,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	result, err := probe.Execute(ctx, target)
	// UDP may not receive a response, this is normal
	if err != nil {
		t.Logf("UDP probe error (expected for no response): %v", err)
	}
	if result != nil {
		if result.Success {
			t.Error("Expected unsuccessful probe for no response")
		}
		if result.Error == "" {
			t.Error("Expected error message for timeout")
		}
	}
}

func TestUDPProbe_Execute_InvalidHost(t *testing.T) {
	config := &UDPConfig{
		PayloadSize:     64,
		ResponseTimeout: 1 * time.Second,
		MaxPacketSize:   1024,
	}
	probe := NewUDPProbe(config)
	defer probe.Close()

	target := types.Target{
		Host:     "invalid-host-name-that-does-not-exist.local",
		Port:     53,
		Protocol: types.ProbeTypeUDP,
		Timeout:  1 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := probe.Execute(ctx, target)
	if err == nil && (result == nil || result.Error == "") {
		t.Error("Expected error for invalid host")
	}
}

func TestUDPProbe_Type(t *testing.T) {
	config := &UDPConfig{}
	probe := NewUDPProbe(config)

	if probe.Type() != types.ProbeTypeUDP {
		t.Errorf("Expected type UDP, got %s", probe.Type())
	}
}

func TestUDPProbe_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *UDPConfig
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  &UDPConfig{PayloadSize: 64},
			wantErr: false,
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var probe *UDPProbe
			if tt.config != nil {
				probe = NewUDPProbe(tt.config).(*UDPProbe)
			} else {
				probe = &UDPProbe{config: nil}
			}

			err := probe.Validate(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUDPProbe_Close(t *testing.T) {
	config := &UDPConfig{}
	probe := NewUDPProbe(config)

	if err := probe.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestUDPProbe_DefaultPayloadSize(t *testing.T) {
	config := &UDPConfig{
		PayloadSize: 0, // Should use default
	}
	probe := NewUDPProbe(config)
	defer probe.Close()

	// Start UDP server
	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to resolve UDP address: %v", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		t.Fatalf("Failed to start UDP server: %v", err)
	}
	defer conn.Close()

	target := types.Target{
		Host:     "127.0.0.1",
		Port:     conn.LocalAddr().(*net.UDPAddr).Port,
		Protocol: types.ProbeTypeUDP,
		Timeout:  1 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, _ := probe.Execute(ctx, target)
	if result != nil && len(result.Payload) == 0 {
		t.Error("Expected default payload size to be used")
	}
}

func TestUDPProbe_Execute_WithResponse(t *testing.T) {
	config := &UDPConfig{
		PayloadSize:     64,
		ResponseTimeout: 1 * time.Second,
		MaxPacketSize:   1024,
	}
	probe := NewUDPProbe(config)
	defer probe.Close()

	// Start UDP echo server
	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to resolve UDP address: %v", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		t.Fatalf("Failed to start UDP server: %v", err)
	}
	defer conn.Close()

	// Echo server
	go func() {
		buffer := make([]byte, 1024)
		for {
			n, clientAddr, err := conn.ReadFromUDP(buffer)
			if err != nil {
				return
			}
			conn.WriteToUDP(buffer[:n], clientAddr)
		}
	}()

	target := types.Target{
		Host:     "127.0.0.1",
		Port:     conn.LocalAddr().(*net.UDPAddr).Port,
		Protocol: types.ProbeTypeUDP,
		Timeout:  2 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := probe.Execute(ctx, target)
	if err != nil {
		t.Logf("UDP probe error (may be expected): %v", err)
	}
	if result != nil {
		t.Logf("UDP probe result: Success=%v, PayloadLen=%d, ResponseLen=%d",
			result.Success, len(result.Payload), len(result.Response))
	}
}

func TestUDPProbe_Execute_LargePayload(t *testing.T) {
	config := &UDPConfig{
		PayloadSize:     512,
		ResponseTimeout: 1 * time.Second,
		MaxPacketSize:   2048,
	}
	probe := NewUDPProbe(config)
	defer probe.Close()

	target := types.Target{
		Host:     "127.0.0.1",
		Port:     53,
		Protocol: types.ProbeTypeUDP,
		Timeout:  1 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err := probe.Execute(ctx, target)
	// May fail if no response, but should not panic
	if err != nil {
		t.Logf("UDP probe with large payload failed (may be expected): %v", err)
	}
	if result != nil {
		t.Logf("UDP probe result: Success=%v, PayloadLen=%d", result.Success, len(result.Payload))
	}
}

func TestUDPProbe_Execute_WithMaxPacketSize(t *testing.T) {
	config := &UDPConfig{
		PayloadSize:     64,
		ResponseTimeout: 1 * time.Second,
		MaxPacketSize:   512,
	}
	probe := NewUDPProbe(config)
	defer probe.Close()

	target := types.Target{
		Host:     "127.0.0.1",
		Port:     53,
		Protocol: types.ProbeTypeUDP,
		Timeout:  1 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err := probe.Execute(ctx, target)
	// May fail if no response, but should not panic
	if err != nil {
		t.Logf("UDP probe with max packet size failed (may be expected): %v", err)
	}
	if result != nil {
		t.Logf("UDP probe result: Success=%v", result.Success)
	}
}

func TestUDPProbe_Execute_WithResponseTimeout(t *testing.T) {
	config := &UDPConfig{
		PayloadSize:     64,
		ResponseTimeout: 100 * time.Millisecond,
		MaxPacketSize:   1024,
	}
	probe := NewUDPProbe(config)
	defer probe.Close()

	target := types.Target{
		Host:     "127.0.0.1",
		Port:     53,
		Protocol: types.ProbeTypeUDP,
		Timeout:  1 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err := probe.Execute(ctx, target)
	// Should timeout, but should not panic
	if err != nil {
		t.Logf("UDP probe timeout (expected): %v", err)
	}
	if result != nil {
		t.Logf("UDP probe result: Success=%v, Error=%s", result.Success, result.Error)
	}
}
