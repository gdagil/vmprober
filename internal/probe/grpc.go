package probe

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"strconv"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/gdagil/vmprober/internal/types"
	"github.com/gdagil/vmprober/pkg/interfaces"
)

// GRPCProbe implements gRPC probes
type GRPCProbe struct {
	config *GRPCConfig
}

// GRPCConfig is the gRPC probe configuration
type GRPCConfig struct {
	Service         string            // Service name for health check (empty string = overall health check)
	TLS             bool              // Use TLS
	TLSSkipVerify   bool              // Skip certificate verification
	ServerName      string            // Server name for TLS
	Metadata        map[string]string // gRPC metadata (headers)
	ExpectedStatus  string            // Expected status (SERVING, NOT_SERVING, UNKNOWN, SERVICE_UNKNOWN)
	ReflectionProbe bool              // Use gRPC reflection instead of health check
	Timeout         time.Duration
}

// NewGRPCProbe creates a new gRPC probe
func NewGRPCProbe(config *GRPCConfig) interfaces.Probe {
	if config == nil {
		config = &GRPCConfig{}
	}

	// Set defaults
	if config.ExpectedStatus == "" {
		config.ExpectedStatus = "SERVING"
	}
	if config.Timeout == 0 {
		config.Timeout = 10 * time.Second
	}

	return &GRPCProbe{
		config: config,
	}
}

// Execute executes a gRPC probe
func (p *GRPCProbe) Execute(ctx context.Context, target types.Target) (*types.ProbeResult, error) {
	start := time.Now()
	result := &types.ProbeResult{
		Protocol:   types.ProbeTypeGRPC,
		Timestamp:  start,
		Attempt:    1,
		TargetHost: target.Host,
		TargetPort: target.Port,
	}

	// Form address
	port := target.Port
	if port == 0 {
		port = 443 // gRPC typically uses 443
	}

	addr := net.JoinHostPort(target.Host, strconv.Itoa(port))

	// Configure connection options
	var opts []grpc.DialOption

	if p.config.TLS {
		tlsConfig := &tls.Config{
			// #nosec G402 -- InsecureSkipVerify is configurable and may be needed for testing/internal services
			InsecureSkipVerify: p.config.TLSSkipVerify,
			MinVersion:         tls.VersionTLS12,
		}
		if p.config.ServerName != "" {
			tlsConfig.ServerName = p.config.ServerName
		}
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	// Set timeout
	dialCtx, dialCancel := context.WithTimeout(ctx, p.config.Timeout)
	defer dialCancel()

	// Connect
	//nolint:staticcheck // grpc.DialContext is deprecated but NewClient has different semantics
	conn, err := grpc.DialContext(dialCtx, addr, opts...)
	if err != nil {
		result.Error = fmt.Sprintf("failed to connect: %v", err)
		result.RTT = time.Since(start)
		return result, err
	}
	defer conn.Close()

	// Add metadata if present
	if len(p.config.Metadata) > 0 {
		md := metadata.New(p.config.Metadata)
		ctx = metadata.NewOutgoingContext(ctx, md)
	}

	// Create health check client
	healthClient := grpc_health_v1.NewHealthClient(conn)

	// Execute health check
	checkCtx, checkCancel := context.WithTimeout(ctx, p.config.Timeout)
	defer checkCancel()

	resp, err := healthClient.Check(checkCtx, &grpc_health_v1.HealthCheckRequest{
		Service: p.config.Service,
	})

	result.RTT = time.Since(start)

	if err != nil {
		// Check error type
		st, ok := status.FromError(err)
		if ok {
			switch st.Code() {
			case codes.NotFound:
				// Service not found, this may be normal if ExpectedStatus == "SERVICE_UNKNOWN"
				if p.config.ExpectedStatus == "SERVICE_UNKNOWN" {
					result.Success = true
					result.TargetPort = port
					result.Role = "client"
					result.TLS = p.config.TLS
					return result, nil
				}
				result.Error = fmt.Sprintf("service not found: %s", p.config.Service)
			case codes.Unavailable:
				result.Error = fmt.Sprintf("service unavailable: %v", st.Message())
			case codes.DeadlineExceeded:
				result.Error = "health check timeout"
			default:
				result.Error = fmt.Sprintf("health check failed: %v (code: %s)", st.Message(), st.Code())
			}
		} else {
			result.Error = fmt.Sprintf("health check failed: %v", err)
		}
		return result, err
	}

	// Check status
	actualStatus := resp.Status.String()
	if p.config.ExpectedStatus != "" && actualStatus != p.config.ExpectedStatus {
		result.Error = fmt.Sprintf("unexpected health status: got %s, expected %s", actualStatus, p.config.ExpectedStatus)
		return result, fmt.Errorf("unexpected health status: got %s, expected %s", actualStatus, p.config.ExpectedStatus)
	}

	// Success
	result.Success = true
	result.TargetPort = port
	result.Role = "client"
	result.TLS = p.config.TLS

	// Extract IP
	addrs, err := net.LookupIP(target.Host)
	if err == nil && len(addrs) > 0 {
		result.TargetIP = addrs[0].String()
	}

	return result, nil
}

// Type returns the probe type
func (p *GRPCProbe) Type() types.ProbeType {
	return types.ProbeTypeGRPC
}

// Validate validates the configuration
func (p *GRPCProbe) Validate(config interface{}) error {
	if p.config == nil {
		return fmt.Errorf("gRPC config is nil")
	}

	// Validate expected status
	validStatuses := map[string]bool{
		"SERVING":         true,
		"NOT_SERVING":     true,
		"UNKNOWN":         true,
		"SERVICE_UNKNOWN": true,
	}
	if p.config.ExpectedStatus != "" && !validStatuses[p.config.ExpectedStatus] {
		return fmt.Errorf("invalid expected status: %s", p.config.ExpectedStatus)
	}

	return nil
}

// Close releases resources
func (p *GRPCProbe) Close() error {
	return nil
}

// BuildGRPCConfigFromTarget creates gRPC configuration from Target
func BuildGRPCConfigFromTarget(target types.Target) *GRPCConfig {
	config := &GRPCConfig{
		Service:        "",
		TLS:            false,
		TLSSkipVerify:  false,
		ExpectedStatus: "SERVING",
		Timeout:        target.Timeout,
	}

	if target.GRPC != nil {
		config.Service = target.GRPC.Service
		config.TLS = target.GRPC.TLS
		config.TLSSkipVerify = target.GRPC.TLSSkipVerify
		config.ServerName = target.GRPC.ServerName
		config.Metadata = target.GRPC.Metadata
		if target.GRPC.ExpectedStatus != "" {
			config.ExpectedStatus = target.GRPC.ExpectedStatus
		}
		config.ReflectionProbe = target.GRPC.ReflectionProbe
	}

	// If using global TLS config
	if target.TLS != nil && target.TLS.Enabled {
		config.TLS = true
		config.TLSSkipVerify = target.TLS.InsecureSkipVerify
		if target.TLS.ServerName != "" {
			config.ServerName = target.TLS.ServerName
		}
	}

	return config
}
