package probe

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/gdagil/vmprober/internal/types"
	"github.com/gdagil/vmprober/pkg/interfaces"
)

// TCPProbe implements TCP probes
type TCPProbe struct {
	config *TCPConfig
}

// TCPConfig TCP probe configuration
type TCPConfig struct {
	ConnectTimeout time.Duration
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
}

// NewTCPProbe creates a new TCP probe
func NewTCPProbe(config *TCPConfig) interfaces.Probe {
	return &TCPProbe{
		config: config,
	}
}

// Execute performs a TCP probe
func (p *TCPProbe) Execute(ctx context.Context, target types.Target) (*types.ProbeResult, error) {
	start := time.Now()
	result := &types.ProbeResult{
		Protocol:  types.ProbeTypeTCP,
		Timestamp: start,
		Attempt:   1,
	}

	// DNS resolution
	addr := target.Host
	if target.Port > 0 {
		addr = net.JoinHostPort(target.Host, strconv.Itoa(target.Port))
	}

	// Create dialer with timeout
	dialer := &net.Dialer{
		Timeout: target.Timeout,
	}

	// Save hostname for metrics
	result.TargetHost = target.Host
	result.TargetPort = target.Port

	// Perform connection
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		result.Error = err.Error()
		result.RTT = time.Since(start)
		return result, err
	}
	defer conn.Close()

	// Successful connection
	result.Success = true
	result.RTT = time.Since(start)
	result.TargetIP = conn.RemoteAddr().(*net.TCPAddr).IP.String()
	// TargetPort already set above
	result.SourceIP = conn.LocalAddr().(*net.TCPAddr).IP.String()
	result.Role = "client"

	return result, nil
}

// Type returns the probe type
func (p *TCPProbe) Type() types.ProbeType {
	return types.ProbeTypeTCP
}

// Validate validates the configuration
func (p *TCPProbe) Validate(config interface{}) error {
	if p.config == nil {
		return fmt.Errorf("TCP config is nil")
	}
	return nil
}

// Close releases resources
func (p *TCPProbe) Close() error {
	return nil
}
