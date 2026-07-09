package probe

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gdagil/vmprober/internal/types"
)

// startDNSServer starts a hermetic loopback UDP DNS server using the given
// handler and returns its address and a cleanup function. It blocks until the
// server is ready to serve, so tests are race-free.
func startDNSServer(t *testing.T, handler dns.HandlerFunc) (*net.UDPAddr, func()) {
	t.Helper()
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	require.NoError(t, err)

	server := &dns.Server{PacketConn: pc, Handler: handler}
	started := make(chan struct{})
	server.NotifyStartedFunc = func() { close(started) }

	go func() { _ = server.ActivateAndServe() }()
	<-started

	return pc.LocalAddr().(*net.UDPAddr), func() { _ = server.Shutdown() }
}

func TestNewDNSProbe(t *testing.T) {
	config := &DNSConfig{
		QueryType: "A",
		Protocol:  "udp",
	}

	probe := NewDNSProbe(config)
	if probe == nil {
		t.Fatal("NewDNSProbe returned nil")
	}

	if probe.Type() != types.ProbeTypeDNS {
		t.Errorf("Expected type DNS, got %s", probe.Type())
	}
}

func TestNewDNSProbe_Defaults(t *testing.T) {
	probe := NewDNSProbe(nil).(*DNSProbe)

	if probe.config.QueryType != "A" {
		t.Errorf("Expected default query type A, got %s", probe.config.QueryType)
	}
	if probe.config.Protocol != "udp" {
		t.Errorf("Expected default protocol udp, got %s", probe.config.Protocol)
	}
	if probe.config.Timeout != 5*time.Second {
		t.Errorf("Expected default timeout 5s, got %v", probe.config.Timeout)
	}
}

func TestDNSProbe_Execute_Success(t *testing.T) {
	config := &DNSConfig{
		QueryName:      "google.com",
		QueryType:      "A",
		Protocol:       "udp",
		ValidateAnswer: true,
		Recursion:      true,
		Timeout:        5 * time.Second,
	}
	probe := NewDNSProbe(config)
	defer probe.Close()

	target := types.Target{
		Host:     "8.8.8.8",
		Port:     53,
		Protocol: types.ProbeTypeDNS,
		Timeout:  5 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := probe.Execute(ctx, target)

	// This test requires network access, so it may not work in isolated environments
	if err != nil {
		t.Skipf("Skipping test due to network error: %v", err)
	}

	if result == nil {
		t.Fatal("Result is nil")
	}
	if !result.Success {
		t.Errorf("Expected successful probe, got error: %s", result.Error)
	}
	if result.RTT <= 0 {
		t.Error("Expected positive RTT")
	}
	if result.Protocol != types.ProbeTypeDNS {
		t.Errorf("Expected protocol DNS, got %s", result.Protocol)
	}
	if result.DNSResult == nil {
		t.Error("Expected DNS result to be set")
	}
	if result.DNSResult != nil && len(result.DNSResult.ResolvedIPs) == 0 {
		t.Error("Expected at least one resolved IP")
	}
}

func TestDNSProbe_Execute_AAAA(t *testing.T) {
	config := &DNSConfig{
		QueryName:      "google.com",
		QueryType:      "AAAA",
		Protocol:       "udp",
		ValidateAnswer: true,
		Timeout:        5 * time.Second,
	}
	probe := NewDNSProbe(config)
	defer probe.Close()

	target := types.Target{
		Host:     "8.8.8.8",
		Port:     53,
		Protocol: types.ProbeTypeDNS,
		Timeout:  5 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := probe.Execute(ctx, target)
	if err != nil {
		t.Skipf("Skipping test due to network error: %v", err)
	}

	if result == nil {
		t.Fatal("Result is nil")
	}
	// AAAA query may return empty response if there's no IPv6
	if !result.Success {
		t.Logf("AAAA query may have empty result: %s", result.Error)
	}
}

func TestDNSProbe_Execute_MX(t *testing.T) {
	config := &DNSConfig{
		QueryName:      "gmail.com",
		QueryType:      "MX",
		Protocol:       "udp",
		ValidateAnswer: true,
		Timeout:        5 * time.Second,
	}
	probe := NewDNSProbe(config)
	defer probe.Close()

	target := types.Target{
		Host:     "8.8.8.8",
		Port:     53,
		Protocol: types.ProbeTypeDNS,
		Timeout:  5 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := probe.Execute(ctx, target)
	if err != nil {
		t.Skipf("Skipping test due to network error: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected successful probe, got error: %s", result.Error)
	}
}

func TestDNSProbe_Execute_TXT(t *testing.T) {
	config := &DNSConfig{
		QueryName: "google.com",
		QueryType: "TXT",
		Protocol:  "udp",
		Timeout:   5 * time.Second,
	}
	probe := NewDNSProbe(config)
	defer probe.Close()

	target := types.Target{
		Host:     "8.8.8.8",
		Port:     53,
		Protocol: types.ProbeTypeDNS,
		Timeout:  5 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := probe.Execute(ctx, target)
	if err != nil {
		t.Skipf("Skipping test due to network error: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected successful probe, got error: %s", result.Error)
	}
}

func TestDNSProbe_Execute_TCP(t *testing.T) {
	config := &DNSConfig{
		QueryName: "google.com",
		QueryType: "A",
		Protocol:  "tcp",
		Timeout:   5 * time.Second,
	}
	probe := NewDNSProbe(config)
	defer probe.Close()

	target := types.Target{
		Host:     "8.8.8.8",
		Port:     53,
		Protocol: types.ProbeTypeDNS,
		Timeout:  5 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := probe.Execute(ctx, target)
	if err != nil {
		t.Skipf("Skipping test due to network error: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected successful probe, got error: %s", result.Error)
	}
}

func TestDNSProbe_Execute_ExpectedRecords(t *testing.T) {
	config := &DNSConfig{
		QueryName:       "gmail.com",
		QueryType:       "MX",
		Protocol:        "udp",
		ExpectedRecords: []string{"google.com"},
		Timeout:         5 * time.Second,
	}
	probe := NewDNSProbe(config)
	defer probe.Close()

	target := types.Target{
		Host:     "8.8.8.8",
		Port:     53,
		Protocol: types.ProbeTypeDNS,
		Timeout:  5 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := probe.Execute(ctx, target)
	if err != nil {
		t.Skipf("Skipping test due to network error: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected successful probe, got error: %s", result.Error)
	}
}

func TestDNSProbe_Execute_ExpectedRecords_NotFound(t *testing.T) {
	config := &DNSConfig{
		QueryName:       "google.com",
		QueryType:       "A",
		Protocol:        "udp",
		ExpectedRecords: []string{"nonexistent-record-12345.com"},
		Timeout:         5 * time.Second,
	}
	probe := NewDNSProbe(config)
	defer probe.Close()

	target := types.Target{
		Host:     "8.8.8.8",
		Port:     53,
		Protocol: types.ProbeTypeDNS,
		Timeout:  5 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, _ := probe.Execute(ctx, target)

	// If query succeeded, check that expected record is not found
	if result != nil && result.Success {
		t.Error("Expected failed probe due to expected record not found")
	}
}

func TestDNSProbe_Execute_InvalidServer(t *testing.T) {
	// Hermetic + deterministic: point at a reserved-then-released loopback port
	// so the query cannot be answered. On loopback this yields a fast
	// connection-refused (ICMP port unreachable) or a read timeout - either way
	// Execute returns an error. The previous 192.0.2.1 version was flaky because
	// captive portals / DNS interceptors answer queries to arbitrary IPs.
	closed := closedUDPAddr(t)

	config := &DNSConfig{
		QueryName: "example.com",
		QueryType: "A",
		Protocol:  "udp",
		Timeout:   500 * time.Millisecond,
	}
	probe := NewDNSProbe(config)
	defer probe.Close()

	target := types.Target{
		Host:     closed.IP.String(),
		Port:     closed.Port,
		Protocol: types.ProbeTypeDNS,
		Timeout:  500 * time.Millisecond,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err := probe.Execute(ctx, target)
	require.Error(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Success)
	assert.NotEmpty(t, result.Error)
}

func TestDNSProbe_Execute_LocalServer_Success(t *testing.T) {
	// Hermetic DNS server that answers A queries with a fixed record.
	addr, cleanup := startDNSServer(t, func(w dns.ResponseWriter, r *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(r)
		m.Authoritative = true
		for _, q := range r.Question {
			if q.Qtype == dns.TypeA {
				rr, err := dns.NewRR(q.Name + " 3600 IN A 93.184.216.34")
				if err == nil {
					m.Answer = append(m.Answer, rr)
				}
			}
		}
		_ = w.WriteMsg(m)
	})
	defer cleanup()

	config := &DNSConfig{
		QueryName:      "example.com",
		QueryType:      "A",
		Protocol:       "udp",
		ValidateAnswer: true,
		Timeout:        2 * time.Second,
	}
	probe := NewDNSProbe(config)
	defer probe.Close()

	target := types.Target{
		Host:     addr.IP.String(),
		Port:     addr.Port,
		Protocol: types.ProbeTypeDNS,
		Timeout:  2 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := probe.Execute(ctx, target)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success, "expected successful probe, error: %s", result.Error)
	require.NotNil(t, result.DNSResult)
	assert.Contains(t, result.DNSResult.ResolvedIPs, "93.184.216.34")
	assert.Equal(t, addr.IP.String(), result.TargetIP)
	assert.Equal(t, "client", result.Role)
}

func TestDNSProbe_Execute_LocalServer_ServfailValidated(t *testing.T) {
	// Server returns SERVFAIL; with ValidateAnswer the probe must fail on RCODE.
	addr, cleanup := startDNSServer(t, func(w dns.ResponseWriter, r *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(r)
		m.Rcode = dns.RcodeServerFailure
		_ = w.WriteMsg(m)
	})
	defer cleanup()

	config := &DNSConfig{
		QueryName:      "example.com",
		QueryType:      "A",
		Protocol:       "udp",
		ValidateAnswer: true,
		Timeout:        2 * time.Second,
	}
	probe := NewDNSProbe(config)
	defer probe.Close()

	target := types.Target{
		Host:     addr.IP.String(),
		Port:     addr.Port,
		Protocol: types.ProbeTypeDNS,
		Timeout:  2 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := probe.Execute(ctx, target)
	require.Error(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "RCODE")
}

func TestDNSProbe_Execute_LocalServer_ExpectedRecords(t *testing.T) {
	handler := func(w dns.ResponseWriter, r *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(r)
		rr, _ := dns.NewRR(r.Question[0].Name + " 3600 IN A 203.0.113.7")
		m.Answer = append(m.Answer, rr)
		_ = w.WriteMsg(m)
	}
	addr, cleanup := startDNSServer(t, handler)
	defer cleanup()

	target := types.Target{
		Host:     addr.IP.String(),
		Port:     addr.Port,
		Protocol: types.ProbeTypeDNS,
		Timeout:  2 * time.Second,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	t.Run("found", func(t *testing.T) {
		probe := NewDNSProbe(&DNSConfig{
			QueryName:       "example.com",
			QueryType:       "A",
			Protocol:        "udp",
			ExpectedRecords: []string{"203.0.113.7"},
			Timeout:         2 * time.Second,
		})
		defer probe.Close()

		result, err := probe.Execute(ctx, target)
		require.NoError(t, err)
		assert.True(t, result.Success, "expected successful probe, error: %s", result.Error)
	})

	t.Run("not found", func(t *testing.T) {
		probe := NewDNSProbe(&DNSConfig{
			QueryName:       "example.com",
			QueryType:       "A",
			Protocol:        "udp",
			ExpectedRecords: []string{"198.51.100.42"},
			Timeout:         2 * time.Second,
		})
		defer probe.Close()

		result, err := probe.Execute(ctx, target)
		require.Error(t, err)
		require.NotNil(t, result)
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "expected records not found")
	})
}

func TestDNSProbe_Execute_NXDomain(t *testing.T) {
	config := &DNSConfig{
		QueryName:      "this-domain-definitely-does-not-exist-12345.com",
		QueryType:      "A",
		Protocol:       "udp",
		ValidateAnswer: true,
		Timeout:        5 * time.Second,
	}
	probe := NewDNSProbe(config)
	defer probe.Close()

	target := types.Target{
		Host:     "8.8.8.8",
		Port:     53,
		Protocol: types.ProbeTypeDNS,
		Timeout:  5 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, _ := probe.Execute(ctx, target)

	// With ValidateAnswer=true, NXDOMAIN should result in error
	if result != nil && result.Success {
		t.Error("Expected failed probe for NXDOMAIN")
	}
}

func TestDNSProbe_parseQueryType(t *testing.T) {
	probe := NewDNSProbe(nil).(*DNSProbe)

	tests := []struct {
		qtype   string
		wantErr bool
	}{
		{"A", false},
		{"AAAA", false},
		{"CNAME", false},
		{"MX", false},
		{"TXT", false},
		{"NS", false},
		{"SOA", false},
		{"SRV", false},
		{"PTR", false},
		{"CAA", false},
		{"DNSKEY", false},
		{"DS", false},
		{"INVALID", true},
		{"", true},
	}

	for _, tt := range tests {
		t.Run(tt.qtype, func(t *testing.T) {
			_, err := probe.parseQueryType(tt.qtype)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseQueryType(%s) error = %v, wantErr %v", tt.qtype, err, tt.wantErr)
			}
		})
	}
}

func TestDNSProbe_Type(t *testing.T) {
	config := &DNSConfig{}
	probe := NewDNSProbe(config)

	if probe.Type() != types.ProbeTypeDNS {
		t.Errorf("Expected type DNS, got %s", probe.Type())
	}
}

func TestDNSProbe_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *DNSConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &DNSConfig{
				QueryType: "A",
				Protocol:  "udp",
			},
			wantErr: false,
		},
		{
			name: "valid TCP config",
			config: &DNSConfig{
				QueryType: "A",
				Protocol:  "tcp",
			},
			wantErr: false,
		},
		{
			name: "valid DoT config",
			config: &DNSConfig{
				QueryType: "A",
				Protocol:  "tcp-tls",
			},
			wantErr: false,
		},
		{
			name: "invalid query type",
			config: &DNSConfig{
				QueryType: "INVALID",
				Protocol:  "udp",
			},
			wantErr: true,
		},
		{
			name: "invalid protocol",
			config: &DNSConfig{
				QueryType: "A",
				Protocol:  "invalid",
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
			var probe *DNSProbe
			if tt.config != nil {
				probe = NewDNSProbe(tt.config).(*DNSProbe)
			} else {
				probe = &DNSProbe{config: nil}
			}

			err := probe.Validate(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDNSProbe_Close(t *testing.T) {
	config := &DNSConfig{}
	probe := NewDNSProbe(config)

	if err := probe.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestBuildDNSConfigFromTarget(t *testing.T) {
	target := types.Target{
		Host:     "8.8.8.8",
		Port:     53,
		Protocol: types.ProbeTypeDNS,
		Timeout:  10 * time.Second,
		DNS: &types.DNSConfig{
			QueryName:       "example.com",
			QueryType:       "MX",
			Protocol:        "tcp",
			ExpectedRecords: []string{"mail.example.com"},
			ValidateAnswer:  true,
			Recursion:       false,
		},
	}

	config := BuildDNSConfigFromTarget(target)

	if config.QueryName != "example.com" {
		t.Errorf("Expected query name example.com, got %s", config.QueryName)
	}
	if config.QueryType != "MX" {
		t.Errorf("Expected query type MX, got %s", config.QueryType)
	}
	if config.Protocol != "tcp" {
		t.Errorf("Expected protocol tcp, got %s", config.Protocol)
	}
	if len(config.ExpectedRecords) != 1 || config.ExpectedRecords[0] != "mail.example.com" {
		t.Errorf("Expected expected records [mail.example.com], got %v", config.ExpectedRecords)
	}
	if !config.ValidateAnswer {
		t.Error("Expected validate answer to be true")
	}
	if config.Recursion {
		t.Error("Expected recursion to be false")
	}
}

func TestBuildDNSConfigFromTarget_Defaults(t *testing.T) {
	target := types.Target{
		Host:     "8.8.8.8",
		Port:     53,
		Protocol: types.ProbeTypeDNS,
		Timeout:  5 * time.Second,
	}

	config := BuildDNSConfigFromTarget(target)

	if config.QueryType != "A" {
		t.Errorf("Expected default query type A, got %s", config.QueryType)
	}
	if config.Protocol != "udp" {
		t.Errorf("Expected default protocol udp, got %s", config.Protocol)
	}
	if !config.ValidateAnswer {
		t.Error("Expected default validate answer to be true")
	}
	if !config.Recursion {
		t.Error("Expected default recursion to be true")
	}
}

func TestDNSProbe_Execute_TCPProtocol(t *testing.T) {
	config := &DNSConfig{
		QueryName: "google.com",
		QueryType: "A",
		Protocol:  "tcp",
		Timeout:   5 * time.Second,
	}
	probe := NewDNSProbe(config)
	defer probe.Close()

	target := types.Target{
		Host:     "8.8.8.8",
		Port:     53,
		Protocol: types.ProbeTypeDNS,
		Timeout:  5 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := probe.Execute(ctx, target)
	// May fail if DNS server is unavailable, but should not panic
	if err != nil {
		t.Logf("DNS probe with TCP failed (may be expected): %v", err)
	}
	if result != nil {
		t.Logf("DNS probe result: Success=%v, Error=%s", result.Success, result.Error)
	}
}

func TestDNSProbe_Execute_UnsupportedProtocol(t *testing.T) {
	config := &DNSConfig{
		QueryName: "google.com",
		QueryType: "A",
		Protocol:  "invalid-protocol",
		Timeout:   5 * time.Second,
	}
	probe := NewDNSProbe(config)
	defer probe.Close()

	target := types.Target{
		Host:     "8.8.8.8",
		Port:     53,
		Protocol: types.ProbeTypeDNS,
		Timeout:  5 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := probe.Execute(ctx, target)
	if err == nil {
		t.Error("Expected error for unsupported protocol")
	}
	if result != nil && result.Success {
		t.Error("Expected unsuccessful probe")
	}
}

func TestDNSProbe_Execute_WithExpectedRecords(t *testing.T) {
	config := &DNSConfig{
		QueryName:       "google.com",
		QueryType:       "A",
		Protocol:        "udp",
		ExpectedRecords: []string{"172.217"}, // Partial match for Google IPs
		Timeout:         5 * time.Second,
	}
	probe := NewDNSProbe(config)
	defer probe.Close()

	target := types.Target{
		Host:     "8.8.8.8",
		Port:     53,
		Protocol: types.ProbeTypeDNS,
		Timeout:  5 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := probe.Execute(ctx, target)
	// May fail if expected records not found, but should not panic
	if err != nil {
		t.Logf("DNS probe with expected records failed (may be expected): %v", err)
	}
	if result != nil {
		t.Logf("DNS probe result: Success=%v", result.Success)
	}
}

func TestDNSProbe_Execute_InvalidQueryType(t *testing.T) {
	config := &DNSConfig{
		QueryName: "google.com",
		QueryType: "INVALID",
		Protocol:  "udp",
		Timeout:   5 * time.Second,
	}
	probe := NewDNSProbe(config)
	defer probe.Close()

	target := types.Target{
		Host:     "8.8.8.8",
		Port:     53,
		Protocol: types.ProbeTypeDNS,
		Timeout:  5 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := probe.Execute(ctx, target)
	if err == nil {
		t.Error("Expected error for invalid query type")
	}
	if result != nil && result.Success {
		t.Error("Expected unsuccessful probe")
	}
}

func TestDNSProbe_Execute_QueryNameFromTarget(t *testing.T) {
	config := &DNSConfig{
		QueryType: "A",
		Protocol:  "udp",
		Timeout:   5 * time.Second,
		// QueryName not set - should use target.Host
	}
	probe := NewDNSProbe(config)
	defer probe.Close()

	target := types.Target{
		Host:     "google.com", // Will be used as query name
		Port:     53,
		Protocol: types.ProbeTypeDNS,
		Timeout:  5 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := probe.Execute(ctx, target)
	// May fail if DNS server is unavailable, but should not panic
	if err != nil {
		t.Logf("DNS probe failed (may be expected): %v", err)
	}
	if result != nil {
		t.Logf("DNS probe result: Success=%v", result.Success)
	}
}

func TestDNSProbe_Execute_DefaultPort(t *testing.T) {
	config := &DNSConfig{
		QueryName: "google.com",
		QueryType: "A",
		Protocol:  "udp",
		Timeout:   5 * time.Second,
	}
	probe := NewDNSProbe(config)
	defer probe.Close()

	target := types.Target{
		Host:     "8.8.8.8",
		Port:     0, // Should default to 53
		Protocol: types.ProbeTypeDNS,
		Timeout:  5 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := probe.Execute(ctx, target)
	// May fail if DNS server is unavailable, but should not panic
	if err != nil {
		t.Logf("DNS probe with default port failed (may be expected): %v", err)
	}
	if result != nil {
		t.Logf("DNS probe result: Success=%v", result.Success)
	}
}

func TestDNSProbe_Execute_WithDifferentQueryTypes(t *testing.T) {
	queryTypes := []string{"A", "AAAA", "MX", "TXT", "NS", "CNAME"}

	for _, qtype := range queryTypes {
		t.Run(qtype, func(t *testing.T) {
			config := &DNSConfig{
				QueryName: "google.com",
				QueryType: qtype,
				Protocol:  "udp",
				Timeout:   5 * time.Second,
			}
			probe := NewDNSProbe(config)
			defer probe.Close()

			target := types.Target{
				Host:     "8.8.8.8",
				Port:     53,
				Protocol: types.ProbeTypeDNS,
				Timeout:  5 * time.Second,
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			result, err := probe.Execute(ctx, target)
			// May fail if DNS server is unavailable, but should not panic
			if err != nil {
				t.Logf("DNS probe with query type %s failed (may be expected): %v", qtype, err)
			}
			if result != nil {
				t.Logf("DNS probe result for %s: Success=%v", qtype, result.Success)
			}
		})
	}
}

func TestDNSProbe_Execute_WithValidateAnswer(t *testing.T) {
	config := &DNSConfig{
		QueryName:      "google.com",
		QueryType:      "A",
		Protocol:       "udp",
		ValidateAnswer: true,
		Timeout:        5 * time.Second,
	}
	probe := NewDNSProbe(config)
	defer probe.Close()

	target := types.Target{
		Host:     "8.8.8.8",
		Port:     53,
		Protocol: types.ProbeTypeDNS,
		Timeout:  5 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := probe.Execute(ctx, target)
	// May fail if DNS server is unavailable, but should not panic
	if err != nil {
		t.Logf("DNS probe with validate answer failed (may be expected): %v", err)
	}
	if result != nil {
		t.Logf("DNS probe result: Success=%v", result.Success)
	}
}
