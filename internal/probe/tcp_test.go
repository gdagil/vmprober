package probe

import (
	"context"
	"errors"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gdagil/vmprober/internal/types"
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
	// Hermetic: a loopback TCP listener that accepts and immediately closes
	// every connection. The kernel completes the handshake from its accept
	// backlog, so the probe connects deterministically.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
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
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success, "expected successful probe, error: %s", result.Error)
	// The TCP probe measures only the connect handshake, which the kernel
	// completes from its accept backlog before the server ever calls Accept, so
	// no server-side delay can extend it. Combined with a coarse monotonic clock
	// (a loopback connect routinely measures exactly 0 on Windows), RTT has no
	// meaningful deterministic lower bound here. Assert instead that it is a sane,
	// populated value below the configured timeout - a huge or unset-to-garbage
	// RTT would fail. Lower-bound RTT coverage lives in the HTTP/UDP/gRPC success
	// tests, whose probes wait on a delayed server reply.
	assert.Less(t, result.RTT, target.Timeout)
	assert.Equal(t, types.ProbeTypeTCP, result.Protocol)
	assert.Equal(t, addr.IP.String(), result.TargetIP)
	assert.Equal(t, addr.Port, result.TargetPort)
	assert.Equal(t, "client", result.Role)
}

func TestTCPProbe_Execute_ConnectionRefused(t *testing.T) {
	config := &TCPConfig{
		ConnectTimeout: 1 * time.Second,
	}
	probe := NewTCPProbe(config)
	defer probe.Close()

	// Use a port that is definitely not listening
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
	// expired_context is hermetic + deterministic: reserve then release a loopback
	// port so nothing is listening, and drive Execute with a context whose deadline
	// is already in the past. This forces the dial-error path (result.Error set,
	// err != nil, Success false) without relying on external network reachability.
	t.Run("expired_context", func(t *testing.T) {
		addr := closedTCPAddr(t)

		probe := NewTCPProbe(&TCPConfig{ConnectTimeout: 100 * time.Millisecond})
		defer probe.Close()

		target := types.Target{
			Host:     addr.IP.String(),
			Port:     addr.Port,
			Protocol: types.ProbeTypeTCP,
			Timeout:  100 * time.Millisecond,
		}

		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
		defer cancel()

		result, err := probe.Execute(ctx, target)
		require.Error(t, err)
		require.NotNil(t, result)
		assert.False(t, result.Success)
		assert.NotEmpty(t, result.Error)
		assert.Equal(t, types.ProbeTypeTCP, result.Protocol)
		// Host/port are recorded before the dial attempt, even on failure.
		assert.Equal(t, addr.IP.String(), result.TargetHost)
		assert.Equal(t, addr.Port, result.TargetPort)
	})

	// dialer_timeout genuinely exercises net.Dialer.Timeout: with a *live* context
	// whose deadline is far beyond the probe timeout, only dialer.Timeout can abort
	// the dial. This requires a target that silently drops the SYN so the connect
	// runs to the timeout - 192.0.2.1 (RFC 5737 TEST-NET-1) is normally black-holed
	// and serves that role without being a real external dependency. A tiny timeout
	// against a live loopback listener can't substitute: the handshake usually wins
	// the race, so it does not deterministically hit the timeout path.
	//
	// Some environments (transparent proxies / captive portals) instead answer or
	// fast-refuse 192.0.2.1; there the dialer-timeout path cannot be reached, so we
	// skip rather than emit a false failure. The assertions run wherever the address
	// is genuinely black-holed (e.g. CI).
	t.Run("dialer_timeout", func(t *testing.T) {
		const probeTimeout = 300 * time.Millisecond

		probe := NewTCPProbe(&TCPConfig{ConnectTimeout: probeTimeout})
		defer probe.Close()

		target := types.Target{
			Host:     "192.0.2.1",
			Port:     80,
			Protocol: types.ProbeTypeTCP,
			Timeout:  probeTimeout,
		}

		// Context deadline (5s) is far beyond probeTimeout so that dialer.Timeout,
		// not the context, is what aborts the dial.
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		start := time.Now()
		result, err := probe.Execute(ctx, target)
		elapsed := time.Since(start)

		require.NotNil(t, result)
		if err == nil {
			t.Skipf("192.0.2.1 answered (elapsed %v) - not black-holed here, cannot exercise dialer.Timeout", elapsed)
		}
		var netErr net.Error
		isTimeout := (errors.As(err, &netErr) && netErr.Timeout()) ||
			strings.Contains(strings.ToLower(err.Error()), "timeout")
		if !isTimeout {
			t.Skipf("192.0.2.1 dial failed without timing out (%v) - environment does not black-hole it", err)
		}

		assert.False(t, result.Success)
		assert.Contains(t, strings.ToLower(result.Error), "timeout")
		// The dial must abort near probeTimeout, well below the 5s context deadline,
		// proving dialer.Timeout - not the context - fired.
		assert.Less(t, elapsed, 2*time.Second,
			"dial should time out near the %v probe timeout, not hang until the context deadline", probeTimeout)
	})
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

func TestTCPProbe_Execute_WithTLS(t *testing.T) {
	// TCPProbe performs a plain TCP connect and ignores target.TLS; this test
	// verifies a TLS-configured target still connects successfully against a
	// hermetic loopback listener (no external network).
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
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

	config := &TCPConfig{
		ConnectTimeout: 5 * time.Second,
	}
	probe := NewTCPProbe(config)
	defer probe.Close()

	addr := ln.Addr().(*net.TCPAddr)
	target := types.Target{
		Host:     addr.IP.String(),
		Port:     addr.Port,
		Protocol: types.ProbeTypeTCP,
		Timeout:  5 * time.Second,
		TLS: &types.TLSConfig{
			Enabled:            true,
			InsecureSkipVerify: true,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := probe.Execute(ctx, target)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success, "expected successful probe, error: %s", result.Error)
	// TCPProbe never negotiates TLS, so the result reflects a plain connection.
	assert.False(t, result.TLS)
}

func TestTCPProbe_Execute_IPv6(t *testing.T) {
	// Hermetic IPv6 loopback listener; skip where the host lacks IPv6.
	ln, err := net.Listen("tcp6", "[::1]:0")
	if err != nil {
		t.Skipf("IPv6 loopback unavailable: %v", err)
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

	// Some sandboxed/CI hosts allow binding ::1 but forbid connecting to it
	// (e.g. Windows WSAEACCES). Skip cleanly in that case instead of reporting
	// a false failure; the probe path itself is already covered over IPv4.
	if c, derr := net.DialTimeout("tcp6", addr.String(), time.Second); derr != nil {
		t.Skipf("IPv6 loopback connections not permitted here: %v", derr)
	} else {
		_ = c.Close()
	}

	config := &TCPConfig{
		ConnectTimeout: 5 * time.Second,
	}
	probe := NewTCPProbe(config)
	defer probe.Close()

	target := types.Target{
		Host:          "::1",
		Port:          addr.Port,
		Protocol:      types.ProbeTypeTCP,
		Timeout:       5 * time.Second,
		NetworkFamily: types.NetworkFamilyInet6,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := probe.Execute(ctx, target)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success, "expected successful probe, error: %s", result.Error)
}

func TestTCPProbe_Execute_WithZeroPort(t *testing.T) {
	config := &TCPConfig{
		ConnectTimeout: 5 * time.Second,
	}
	probe := NewTCPProbe(config)
	defer probe.Close()

	target := types.Target{
		Host:     "127.0.0.1",
		Port:     0, // Zero port
		Protocol: types.ProbeTypeTCP,
		Timeout:  5 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := probe.Execute(ctx, target)
	// May fail if connection is unavailable, but should not panic
	if err != nil {
		t.Logf("TCP probe with zero port failed (may be expected): %v", err)
	}
	if result != nil {
		t.Logf("TCP probe result: Success=%v", result.Success)
	}
}

func TestTCPProbe_Execute_WithReadWriteTimeout(t *testing.T) {
	config := &TCPConfig{
		ConnectTimeout: 5 * time.Second,
		ReadTimeout:    1 * time.Second,
		WriteTimeout:   1 * time.Second,
	}
	probe := NewTCPProbe(config)
	defer probe.Close()

	// Start a simple TCP server
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start TCP server: %v", err)
	}
	defer listener.Close()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	target := types.Target{
		Host:     "127.0.0.1",
		Port:     listener.Addr().(*net.TCPAddr).Port,
		Protocol: types.ProbeTypeTCP,
		Timeout:  5 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := probe.Execute(ctx, target)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected successful probe, got error: %s", result.Error)
	}
}
