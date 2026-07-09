package probe

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/gdagil/vmprober/internal/types"
)

// startGRPCHealthServer starts a hermetic loopback gRPC server exposing the
// standard health service. The overall ("") status is set to SERVING and any
// services passed in statuses are registered with the given status. It returns
// the listener address and a cleanup function.
func startGRPCHealthServer(t *testing.T, statuses map[string]grpc_health_v1.HealthCheckResponse_ServingStatus) *net.TCPAddr {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	srv := grpc.NewServer()
	hs := health.NewServer()
	hs.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	for name, status := range statuses {
		hs.SetServingStatus(name, status)
	}
	grpc_health_v1.RegisterHealthServer(srv, hs)

	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	return lis.Addr().(*net.TCPAddr)
}

// grpcTarget builds a loopback gRPC target for the given address.
func grpcTarget(addr *net.TCPAddr) types.Target {
	return types.Target{
		Host:     addr.IP.String(),
		Port:     addr.Port,
		Protocol: types.ProbeTypeGRPC,
		Timeout:  2 * time.Second,
	}
}

// TestGRPCProbe_Execute_HealthyServing drives the happy path against a local
// health server reporting SERVING. It also sets metadata to cover the
// outgoing-metadata branch of Execute.
func TestGRPCProbe_Execute_HealthyServing(t *testing.T) {
	addr := startGRPCHealthServer(t, nil)

	config := &GRPCConfig{
		Service:        "",
		ExpectedStatus: "SERVING",
		Metadata:       map[string]string{"x-request-id": "abc-123"}, // exercises metadata branch
		Timeout:        2 * time.Second,
	}
	probe := NewGRPCProbe(config)
	defer probe.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := probe.Execute(ctx, grpcTarget(addr))
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success, "expected healthy SERVING probe, error: %s", result.Error)
	assert.Equal(t, types.ProbeTypeGRPC, result.Protocol)
	assert.Equal(t, "client", result.Role)
	assert.Equal(t, addr.Port, result.TargetPort)
	assert.Equal(t, addr.IP.String(), result.TargetIP)
	assert.GreaterOrEqual(t, result.RTT, time.Duration(0))
}

// TestGRPCProbe_Execute_StatusMismatch covers the branch where the reported
// status differs from ExpectedStatus (NOT_SERVING vs SERVING -> failure).
func TestGRPCProbe_Execute_StatusMismatch(t *testing.T) {
	addr := startGRPCHealthServer(t, map[string]grpc_health_v1.HealthCheckResponse_ServingStatus{
		"svc.Down": grpc_health_v1.HealthCheckResponse_NOT_SERVING,
	})

	config := &GRPCConfig{
		Service:        "svc.Down",
		ExpectedStatus: "SERVING",
		Timeout:        2 * time.Second,
	}
	probe := NewGRPCProbe(config)
	defer probe.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := probe.Execute(ctx, grpcTarget(addr))
	require.Error(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "unexpected health status")
	assert.Contains(t, result.Error, "NOT_SERVING")
}

// TestGRPCProbe_Execute_NotServingExpected covers the success branch where the
// reported status matches a non-default ExpectedStatus.
func TestGRPCProbe_Execute_NotServingExpected(t *testing.T) {
	addr := startGRPCHealthServer(t, map[string]grpc_health_v1.HealthCheckResponse_ServingStatus{
		"svc.Down": grpc_health_v1.HealthCheckResponse_NOT_SERVING,
	})

	config := &GRPCConfig{
		Service:        "svc.Down",
		ExpectedStatus: "NOT_SERVING",
		Timeout:        2 * time.Second,
	}
	probe := NewGRPCProbe(config)
	defer probe.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := probe.Execute(ctx, grpcTarget(addr))
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success, "expected NOT_SERVING to match, error: %s", result.Error)
}

// TestGRPCProbe_Execute_ServiceUnknown covers the codes.NotFound branches: a
// request for an unregistered service fails unless ExpectedStatus accepts it.
func TestGRPCProbe_Execute_ServiceUnknown(t *testing.T) {
	addr := startGRPCHealthServer(t, nil)

	t.Run("expected serving -> failure", func(t *testing.T) {
		config := &GRPCConfig{
			Service:        "does.not.Exist",
			ExpectedStatus: "SERVING",
			Timeout:        2 * time.Second,
		}
		probe := NewGRPCProbe(config)
		defer probe.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		result, err := probe.Execute(ctx, grpcTarget(addr))
		require.Error(t, err)
		require.NotNil(t, result)
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "service not found")
	})

	t.Run("expected service_unknown -> success", func(t *testing.T) {
		config := &GRPCConfig{
			Service:        "does.not.Exist",
			ExpectedStatus: "SERVICE_UNKNOWN",
			Timeout:        2 * time.Second,
		}
		probe := NewGRPCProbe(config)
		defer probe.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		result, err := probe.Execute(ctx, grpcTarget(addr))
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.Success, "SERVICE_UNKNOWN should be accepted, error: %s", result.Error)
	})
}

// TestGRPCProbe_Execute_Unreachable covers the connection-error path using a
// loopback port with no listener (hermetic, no external network).
func TestGRPCProbe_Execute_Unreachable(t *testing.T) {
	addr := closedTCPAddr(t)

	config := &GRPCConfig{
		Service:        "",
		ExpectedStatus: "SERVING",
		Timeout:        1 * time.Second,
	}
	probe := NewGRPCProbe(config)
	defer probe.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := probe.Execute(ctx, grpcTarget(addr))
	require.Error(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Success)
	assert.NotEmpty(t, result.Error)
}

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

func TestGRPCProbe_Execute_WithTLS(t *testing.T) {
	config := &GRPCConfig{
		Service:        "",
		ExpectedStatus: "SERVING",
		TLS:            true,
		TLSSkipVerify:  true,
		Timeout:        1 * time.Second,
	}
	probe := NewGRPCProbe(config)
	defer probe.Close()

	target := types.Target{
		Host:     "grpc.example.com",
		Port:     443,
		Protocol: types.ProbeTypeGRPC,
		Timeout:  1 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := probe.Execute(ctx, target)
	// May fail if server is unavailable, but should not panic
	if err != nil {
		t.Logf("GRPC probe with TLS failed (may be expected): %v", err)
	}
	if result != nil {
		t.Logf("GRPC probe result: Success=%v, TLS=%v", result.Success, result.TLS)
	}
}

func TestGRPCProbe_Execute_WithService(t *testing.T) {
	config := &GRPCConfig{
		Service:        "test.Service",
		ExpectedStatus: "SERVING",
		Timeout:        1 * time.Second,
	}
	probe := NewGRPCProbe(config)
	defer probe.Close()

	target := types.Target{
		Host:     "127.0.0.1",
		Port:     50051,
		Protocol: types.ProbeTypeGRPC,
		Timeout:  1 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := probe.Execute(ctx, target)
	// May fail if server is unavailable, but should not panic
	if err != nil {
		t.Logf("GRPC probe with service failed (may be expected): %v", err)
	}
	if result != nil {
		t.Logf("GRPC probe result: Success=%v", result.Success)
	}
}

func TestGRPCProbe_Execute_WithMetadata(t *testing.T) {
	config := &GRPCConfig{
		Service:        "",
		ExpectedStatus: "SERVING",
		Metadata: map[string]string{
			"authorization": "Bearer test-token",
			"x-request-id":  "test-123",
		},
		Timeout: 1 * time.Second,
	}
	probe := NewGRPCProbe(config)
	defer probe.Close()

	target := types.Target{
		Host:     "127.0.0.1",
		Port:     50051,
		Protocol: types.ProbeTypeGRPC,
		Timeout:  1 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := probe.Execute(ctx, target)
	// May fail if server is unavailable, but should not panic
	if err != nil {
		t.Logf("GRPC probe with metadata failed (may be expected): %v", err)
	}
	if result != nil {
		t.Logf("GRPC probe result: Success=%v", result.Success)
	}
}

func TestGRPCProbe_Execute_DefaultPort(t *testing.T) {
	config := &GRPCConfig{
		Service:        "",
		ExpectedStatus: "SERVING",
		Timeout:        1 * time.Second,
	}
	probe := NewGRPCProbe(config)
	defer probe.Close()

	target := types.Target{
		Host:     "grpc.example.com",
		Port:     0, // Should default to 443
		Protocol: types.ProbeTypeGRPC,
		Timeout:  1 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := probe.Execute(ctx, target)
	// May fail if server is unavailable, but should not panic
	if err != nil {
		t.Logf("GRPC probe with default port failed (may be expected): %v", err)
	}
	if result != nil {
		t.Logf("GRPC probe result: Success=%v", result.Success)
	}
}
