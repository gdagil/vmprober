package probe

import (
	"context"
	"testing"
	"time"

	"github.com/gdagil/vmprober/internal/types"
)

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
	config := &DNSConfig{
		QueryName: "google.com",
		QueryType: "A",
		Protocol:  "udp",
		Timeout:   1 * time.Second,
	}
	probe := NewDNSProbe(config)
	defer probe.Close()

	target := types.Target{
		Host:     "192.0.2.1", // Test IP from RFC 5737
		Port:     53,
		Protocol: types.ProbeTypeDNS,
		Timeout:  1 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err := probe.Execute(ctx, target)
	if err == nil && result != nil && result.Success {
		t.Error("Expected error for invalid DNS server")
	}
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
