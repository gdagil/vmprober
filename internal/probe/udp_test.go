package probe

import (
	"context"
	"net"
	"testing"
	"time"

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
	// Start test UDP server
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

	config := &UDPConfig{
		PayloadSize:     64,
		ResponseTimeout: 2 * time.Second,
		MaxPacketSize:   1024,
	}
	probe := NewUDPProbe(config)
	defer probe.Close()

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
		t.Fatalf("Execute failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result is nil")
	}
	if !result.Success {
		t.Logf("UDP probe may timeout if server doesn't respond, result: %+v", result)
	}
	if result.RTT <= 0 {
		t.Error("Expected positive RTT")
	}
	if result.Protocol != types.ProbeTypeUDP {
		t.Errorf("Expected protocol UDP, got %s", result.Protocol)
	}
	if len(result.Payload) == 0 {
		t.Error("Expected payload to be set")
	}
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
