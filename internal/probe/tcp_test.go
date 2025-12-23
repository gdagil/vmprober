package probe

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/vmprober/vmprober/internal/types"
)

func TestNewTCPProbe(t *testing.T) {
	config := &TCPConfig{
		ConnectTimeout: 5 * time.Second,
	}
	
	probe := NewTCPProbe(config)
	if probe == nil {
		t.Fatal("NewTCPProbe returned nil")
	}
	
	if probe.Type() != types.ProbeTypeTCP {
		t.Errorf("Expected type TCP, got %s", probe.Type())
	}
}

func TestTCPProbe_Execute_Success(t *testing.T) {
	// Запускаем тестовый TCP сервер
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start test server: %v", err)
	}
	defer ln.Close()
	
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()
	
	addr := ln.Addr().(*net.TCPAddr)
	
	config := &TCPConfig{
		ConnectTimeout: 5 * time.Second,
	}
	probe := NewTCPProbe(config)
	defer probe.Close()
	
	target := types.Target{
		Host:     addr.IP.String(),
		Port:     addr.Port,
		Protocol: types.ProbeTypeTCP,
		Timeout:  5 * time.Second,
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	result, err := probe.Execute(ctx, target)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	
	if result == nil {
		t.Fatal("Result is nil")
	}
	if !result.Success {
		t.Error("Expected successful probe")
	}
	if result.RTT <= 0 {
		t.Error("Expected positive RTT")
	}
	if result.Protocol != types.ProbeTypeTCP {
		t.Errorf("Expected protocol TCP, got %s", result.Protocol)
	}
	if result.TargetIP != addr.IP.String() {
		t.Errorf("Expected target IP %s, got %s", addr.IP.String(), result.TargetIP)
	}
	if result.TargetPort != addr.Port {
		t.Errorf("Expected target port %d, got %d", addr.Port, result.TargetPort)
	}
}

func TestTCPProbe_Execute_ConnectionRefused(t *testing.T) {
	config := &TCPConfig{
		ConnectTimeout: 1 * time.Second,
	}
	probe := NewTCPProbe(config)
	defer probe.Close()
	
	// Используем порт, который точно не слушается
	target := types.Target{
		Host:     "127.0.0.1",
		Port:     65535,
		Protocol: types.ProbeTypeTCP,
		Timeout:  1 * time.Second,
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	result, err := probe.Execute(ctx, target)
	if err == nil {
		t.Error("Expected error for connection refused")
	}
	if result != nil && result.Success {
		t.Error("Expected unsuccessful probe")
	}
	if result != nil && result.Error == "" {
		t.Error("Expected error message in result")
	}
}

func TestTCPProbe_Execute_Timeout(t *testing.T) {
	config := &TCPConfig{
		ConnectTimeout: 100 * time.Millisecond,
	}
	probe := NewTCPProbe(config)
	defer probe.Close()
	
	// Используем несуществующий хост, который вызовет таймаут
	target := types.Target{
		Host:     "192.0.2.1", // Тестовый IP из RFC 5737
		Port:     80,
		Protocol: types.ProbeTypeTCP,
		Timeout:  100 * time.Millisecond,
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	result, err := probe.Execute(ctx, target)
	// Может быть ошибка или таймаут
	if result != nil && result.Success {
		t.Error("Expected unsuccessful probe")
	}
	if err == nil && result != nil && result.Error == "" {
		t.Error("Expected error message or timeout")
	}
}

func TestTCPProbe_Execute_InvalidHost(t *testing.T) {
	config := &TCPConfig{
		ConnectTimeout: 1 * time.Second,
	}
	probe := NewTCPProbe(config)
	defer probe.Close()
	
	target := types.Target{
		Host:     "invalid-host-name-that-does-not-exist.local",
		Port:     80,
		Protocol: types.ProbeTypeTCP,
		Timeout:  1 * time.Second,
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	result, err := probe.Execute(ctx, target)
	if err == nil && (result == nil || result.Error == "") {
		t.Error("Expected error for invalid host")
	}
}

func TestTCPProbe_Type(t *testing.T) {
	config := &TCPConfig{}
	probe := NewTCPProbe(config)
	
	if probe.Type() != types.ProbeTypeTCP {
		t.Errorf("Expected type TCP, got %s", probe.Type())
	}
}

func TestTCPProbe_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *TCPConfig
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  &TCPConfig{ConnectTimeout: 5 * time.Second},
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
			var probe *TCPProbe
			if tt.config != nil {
				probe = NewTCPProbe(tt.config).(*TCPProbe)
			} else {
				probe = &TCPProbe{config: nil}
			}
			
			err := probe.Validate(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTCPProbe_Close(t *testing.T) {
	config := &TCPConfig{}
	probe := NewTCPProbe(config)
	
	if err := probe.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}


