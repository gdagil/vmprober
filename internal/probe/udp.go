package probe

import (
	"context"
	"crypto/rand"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/gdagil/vmprober/internal/types"
	"github.com/gdagil/vmprober/pkg/interfaces"
)

// UDPProbe implements UDP probes
type UDPProbe struct {
	config *UDPConfig
}

// UDPConfig UDP probe configuration
type UDPConfig struct {
	PayloadSize     int
	ResponseTimeout time.Duration
	MaxPacketSize   int
}

// NewUDPProbe creates a new UDP probe
func NewUDPProbe(config *UDPConfig) interfaces.Probe {
	return &UDPProbe{
		config: config,
	}
}

// Execute performs a UDP probe
func (p *UDPProbe) Execute(ctx context.Context, target types.Target) (*types.ProbeResult, error) {
	start := time.Now()
	result := &types.ProbeResult{
		Protocol:  types.ProbeTypeUDP,
		Timestamp: start,
		Attempt:   1,
	}

	// Create UDP connection
	conn, err := net.ListenUDP("udp", nil)
	if err != nil {
		result.Error = fmt.Sprintf("failed to create UDP socket: %v", err)
		result.RTT = time.Since(start)
		return result, err
	}
	defer conn.Close()

	// Save hostname for metrics
	result.TargetHost = target.Host
	result.TargetPort = target.Port

	// Resolve address
	addr := net.JoinHostPort(target.Host, strconv.Itoa(target.Port))
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		result.Error = fmt.Sprintf("failed to resolve UDP address: %v", err)
		result.RTT = time.Since(start)
		return result, err
	}

	// Generate payload
	payloadSize := p.config.PayloadSize
	if payloadSize == 0 {
		payloadSize = 64
	}
	payload := make([]byte, payloadSize)
	if _, err := rand.Read(payload); err != nil {
		result.Error = fmt.Sprintf("failed to generate payload: %v", err)
		result.RTT = time.Since(start)
		return result, err
	}
	result.Payload = payload

	// Send packet
	if _, err := conn.WriteToUDP(payload, udpAddr); err != nil {
		result.Error = fmt.Sprintf("failed to send UDP packet: %v", err)
		result.RTT = time.Since(start)
		return result, err
	}

	// Set read timeout
	timeout := p.config.ResponseTimeout
	if timeout == 0 {
		timeout = target.Timeout
	}
	if err := conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		result.Error = fmt.Sprintf("failed to set read deadline: %v", err)
		result.RTT = time.Since(start)
		return result, err
	}

	// Attempt to read response
	buffer := make([]byte, p.config.MaxPacketSize)
	if p.config.MaxPacketSize == 0 {
		buffer = make([]byte, 1500)
	}

	_, _, err = conn.ReadFromUDP(buffer)
	if err != nil {
		// UDP may not receive a response, this is not always an error
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			result.Error = "UDP response timeout"
		} else {
			result.Error = fmt.Sprintf("failed to receive UDP response: %v", err)
		}
		result.RTT = time.Since(start)
		// Don't return error for UDP timeout, this is normal
		return result, nil
	}

	// Successful response
	result.Success = true
	result.RTT = time.Since(start)
	result.TargetIP = udpAddr.IP.String()
	// TargetPort already set above
	result.SourceIP = conn.LocalAddr().(*net.UDPAddr).IP.String()
	result.Role = "client"

	return result, nil
}

// Type returns the probe type
func (p *UDPProbe) Type() types.ProbeType {
	return types.ProbeTypeUDP
}

// Validate validates the configuration
func (p *UDPProbe) Validate(config interface{}) error {
	if p.config == nil {
		return fmt.Errorf("UDP config is nil")
	}
	return nil
}

// Close releases resources
func (p *UDPProbe) Close() error {
	return nil
}



