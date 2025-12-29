package probe

import (
	"context"
	"testing"
	"time"

	"github.com/gdagil/vmprober/internal/types"
)

func TestNewGRPCProbe(t *testing.T) {
	config := &GRPCConfig{
		Service:        "",
		ExpectedStatus: "SERVING",
	}

	probe := NewGRPCProbe(config)
	if probe == nil {
		t.Fatal("NewGRPCProbe returned nil")
	}

	if probe.Type() != types.ProbeTypeGRPC {
		t.Errorf("Expected type GRPC, got %s", probe.Type())
	}
}

func TestNewGRPCProbe_Defaults(t *testing.T) {
	probe := NewGRPCProbe(nil).(*GRPCProbe)

	if probe.config.ExpectedStatus != "SERVING" {
		t.Errorf("Expected default expected status SERVING, got %s", probe.config.ExpectedStatus)
	}
	if probe.config.Timeout != 10*time.Second {
		t.Errorf("Expected default timeout 10s, got %v", probe.config.Timeout)
	}
}

func TestGRPCProbe_Execute_ConnectionRefused(t *testing.T) {
	config := &GRPCConfig{
		Service:        "",
		ExpectedStatus: "SERVING",
		Timeout:        1 * time.Second,
	}
	probe := NewGRPCProbe(config)
	defer probe.Close()

	target := types.Target{
		Host:     "127.0.0.1",
		Port:     65533, // A port that is definitely not listening
		Protocol: types.ProbeTypeGRPC,
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

func TestGRPCProbe_Execute_Timeout(t *testing.T) {
	config := &GRPCConfig{
		Service:        "",
		ExpectedStatus: "SERVING",
		Timeout:        100 * time.Millisecond,
	}
	probe := NewGRPCProbe(config)
	defer probe.Close()

	target := types.Target{
		Host:     "192.0.2.1", // Test IP from RFC 5737
		Port:     50051,
		Protocol: types.ProbeTypeGRPC,
		Timeout:  100 * time.Millisecond,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	result, _ := probe.Execute(ctx, target)
	if result != nil && result.Success {
		t.Error("Expected unsuccessful probe due to timeout")
	}
}

func TestGRPCProbe_Execute_InvalidHost(t *testing.T) {
	config := &GRPCConfig{
		Service: "",
		Timeout: 1 * time.Second,
	}
	probe := NewGRPCProbe(config)
	defer probe.Close()

	target := types.Target{
		Host:     "invalid-host-name-that-does-not-exist.local",
		Port:     50051,
		Protocol: types.ProbeTypeGRPC,
		Timeout:  1 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := probe.Execute(ctx, target)
	if err == nil && (result == nil || result.Error == "") {
		t.Error("Expected error for invalid host")
	}
}

func TestGRPCProbe_Type(t *testing.T) {
	config := &GRPCConfig{}
	probe := NewGRPCProbe(config)

	if probe.Type() != types.ProbeTypeGRPC {
		t.Errorf("Expected type GRPC, got %s", probe.Type())
	}
}

func TestGRPCProbe_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *GRPCConfig
		wantErr bool
	}{
		{
			name: "valid config with SERVING",
			config: &GRPCConfig{
				ExpectedStatus: "SERVING",
			},
			wantErr: false,
		},
		{
			name: "valid config with NOT_SERVING",
			config: &GRPCConfig{
				ExpectedStatus: "NOT_SERVING",
			},
			wantErr: false,
		},
		{
			name: "valid config with UNKNOWN",
			config: &GRPCConfig{
				ExpectedStatus: "UNKNOWN",
			},
			wantErr: false,
		},
		{
			name: "valid config with SERVICE_UNKNOWN",
			config: &GRPCConfig{
				ExpectedStatus: "SERVICE_UNKNOWN",
			},
			wantErr: false,
		},
		{
			name: "valid config with empty status",
			config: &GRPCConfig{
				ExpectedStatus: "",
			},
			wantErr: false,
		},
		{
			name: "invalid expected status",
			config: &GRPCConfig{
				ExpectedStatus: "INVALID_STATUS",
			},
			wantErr: true,
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var probe *GRPCProbe
			if tt.config != nil {
				probe = NewGRPCProbe(tt.config).(*GRPCProbe)
			} else {
				probe = &GRPCProbe{config: nil}
			}

			err := probe.Validate(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGRPCProbe_Close(t *testing.T) {
	config := &GRPCConfig{}
	probe := NewGRPCProbe(config)

	if err := probe.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestBuildGRPCConfigFromTarget(t *testing.T) {
	target := types.Target{
		Host:     "grpc.example.com",
		Port:     443,
		Protocol: types.ProbeTypeGRPC,
		Timeout:  15 * time.Second,
		GRPC: &types.GRPCConfig{
			Service:       "user.UserService",
			TLS:           true,
			TLSSkipVerify: true,
			ServerName:    "grpc.example.com",
			Metadata: map[string]string{
				"authorization": "Bearer token",
			},
			ExpectedStatus:  "NOT_SERVING",
			ReflectionProbe: true,
		},
	}

	config := BuildGRPCConfigFromTarget(target)

	if config.Service != "user.UserService" {
		t.Errorf("Expected service user.UserService, got %s", config.Service)
	}
	if !config.TLS {
		t.Error("Expected TLS to be true")
	}
	if !config.TLSSkipVerify {
		t.Error("Expected TLSSkipVerify to be true")
	}
	if config.ServerName != "grpc.example.com" {
		t.Errorf("Expected server name grpc.example.com, got %s", config.ServerName)
	}
	if config.Metadata["authorization"] != "Bearer token" {
		t.Errorf("Expected authorization header, got %v", config.Metadata)
	}
	if config.ExpectedStatus != "NOT_SERVING" {
		t.Errorf("Expected status NOT_SERVING, got %s", config.ExpectedStatus)
	}
	if !config.ReflectionProbe {
		t.Error("Expected reflection probe to be true")
	}
}

func TestBuildGRPCConfigFromTarget_Defaults(t *testing.T) {
	target := types.Target{
		Host:     "grpc.example.com",
		Port:     50051,
		Protocol: types.ProbeTypeGRPC,
		Timeout:  5 * time.Second,
	}

	config := BuildGRPCConfigFromTarget(target)

	if config.Service != "" {
		t.Errorf("Expected default empty service, got %s", config.Service)
	}
	if config.TLS {
		t.Error("Expected default TLS to be false")
	}
	if config.TLSSkipVerify {
		t.Error("Expected default TLSSkipVerify to be false")
	}
	if config.ExpectedStatus != "SERVING" {
		t.Errorf("Expected default status SERVING, got %s", config.ExpectedStatus)
	}
}

func TestBuildGRPCConfigFromTarget_WithTLSConfig(t *testing.T) {
	target := types.Target{
		Host:     "grpc.example.com",
		Port:     443,
		Protocol: types.ProbeTypeGRPC,
		Timeout:  5 * time.Second,
		TLS: &types.TLSConfig{
			Enabled:            true,
			InsecureSkipVerify: true,
			ServerName:         "override.example.com",
		},
	}

	config := BuildGRPCConfigFromTarget(target)

	if !config.TLS {
		t.Error("Expected TLS to be true from TLSConfig")
	}
	if !config.TLSSkipVerify {
		t.Error("Expected TLSSkipVerify to be true from TLSConfig")
	}
	if config.ServerName != "override.example.com" {
		t.Errorf("Expected server name from TLSConfig, got %s", config.ServerName)
	}
}

func TestGRPCProbe_Config_Metadata(t *testing.T) {
	config := &GRPCConfig{
		Service: "test.Service",
		Metadata: map[string]string{
			"x-request-id":  "test-123",
			"authorization": "Bearer test-token",
		},
		Timeout: 5 * time.Second,
	}

	probe := NewGRPCProbe(config).(*GRPCProbe)

	if probe.config.Metadata["x-request-id"] != "test-123" {
		t.Error("Expected x-request-id metadata")
	}
	if probe.config.Metadata["authorization"] != "Bearer test-token" {
		t.Error("Expected authorization metadata")
	}
}

func TestGRPCProbe_TLSConfig(t *testing.T) {
	config := &GRPCConfig{
		Service:       "secure.Service",
		TLS:           true,
		TLSSkipVerify: false,
		ServerName:    "secure.example.com",
		Timeout:       5 * time.Second,
	}

	probe := NewGRPCProbe(config).(*GRPCProbe)

	if !probe.config.TLS {
		t.Error("Expected TLS to be enabled")
	}
	if probe.config.TLSSkipVerify {
		t.Error("Expected TLSSkipVerify to be false")
	}
	if probe.config.ServerName != "secure.example.com" {
		t.Errorf("Expected server name secure.example.com, got %s", probe.config.ServerName)
	}
}

