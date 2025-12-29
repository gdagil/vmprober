package probe

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/miekg/dns"

	"github.com/gdagil/vmprober/internal/types"
	"github.com/gdagil/vmprober/pkg/interfaces"
)

// DNSProbe implements DNS probes
type DNSProbe struct {
	config *DNSConfig
}

// DNSConfig is the DNS probe configuration
type DNSConfig struct {
	QueryName       string
	QueryType       string   // A, AAAA, CNAME, MX, TXT, NS, SOA, SRV, PTR
	Protocol        string   // udp, tcp, tcp-tls (DoT)
	ExpectedRecords []string // Expected DNS records
	ValidateAnswer  bool     // Validate that RCODE is NOERROR
	Recursion       bool     // Request recursive resolution
	Timeout         time.Duration
}

// NewDNSProbe creates a new DNS probe
func NewDNSProbe(config *DNSConfig) interfaces.Probe {
	if config == nil {
		config = &DNSConfig{}
	}

	// Set defaults
	if config.QueryType == "" {
		config.QueryType = "A"
	}
	if config.Protocol == "" {
		config.Protocol = "udp"
	}
	if config.Timeout == 0 {
		config.Timeout = 5 * time.Second
	}

	return &DNSProbe{
		config: config,
	}
}

// Execute executes a DNS probe
func (p *DNSProbe) Execute(ctx context.Context, target types.Target) (*types.ProbeResult, error) {
	start := time.Now()
	result := &types.ProbeResult{
		Protocol:   types.ProbeTypeDNS,
		Timestamp:  start,
		Attempt:    1,
		TargetHost: target.Host,
		TargetPort: target.Port,
	}

	// Determine DNS server
	server := target.Host
	port := target.Port
	if port == 0 {
		port = 53
	}

	// Determine query name
	queryName := p.config.QueryName
	if queryName == "" {
		queryName = target.Host // Use host as query name if not specified
	}

	// Add trailing dot if missing
	if !strings.HasSuffix(queryName, ".") {
		queryName = queryName + "."
	}

	// Determine record type
	qtype, err := p.parseQueryType(p.config.QueryType)
	if err != nil {
		result.Error = err.Error()
		result.RTT = time.Since(start)
		return result, err
	}

	// Create DNS message
	msg := new(dns.Msg)
	msg.SetQuestion(queryName, qtype)
	msg.RecursionDesired = p.config.Recursion

	// Form server address
	serverAddr := net.JoinHostPort(server, strconv.Itoa(port))

	// Create client
	client := new(dns.Client)
	client.Timeout = p.config.Timeout

	var resp *dns.Msg

	// Select protocol
	switch p.config.Protocol {
	case "udp":
		client.Net = "udp"
		resp, _, err = client.ExchangeContext(ctx, msg, serverAddr)
	case "tcp":
		client.Net = "tcp"
		resp, _, err = client.ExchangeContext(ctx, msg, serverAddr)
	case "tcp-tls", "dot":
		client.Net = "tcp-tls"
		client.TLSConfig = &tls.Config{
			InsecureSkipVerify: false,
		}
		if port == 53 {
			// DoT typically uses port 853
			serverAddr = net.JoinHostPort(server, "853")
		}
		resp, _, err = client.ExchangeContext(ctx, msg, serverAddr)
	default:
		err = fmt.Errorf("unsupported DNS protocol: %s", p.config.Protocol)
	}

	result.RTT = time.Since(start)

	if err != nil {
		result.Error = fmt.Sprintf("DNS query failed: %v", err)
		return result, err
	}

	if resp == nil {
		result.Error = "empty DNS response"
		return result, fmt.Errorf("empty DNS response")
	}

	// Check RCODE if validation is required
	if p.config.ValidateAnswer && resp.Rcode != dns.RcodeSuccess {
		result.Error = fmt.Sprintf("DNS RCODE error: %s", dns.RcodeToString[resp.Rcode])
		return result, fmt.Errorf("DNS RCODE error: %s", dns.RcodeToString[resp.Rcode])
	}

	// Extract answers
	var answers []string
	for _, ans := range resp.Answer {
		answers = append(answers, ans.String())
	}

	// Check expected records if specified
	if len(p.config.ExpectedRecords) > 0 {
		found := false
		for _, expected := range p.config.ExpectedRecords {
			for _, answer := range answers {
				if strings.Contains(answer, expected) {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			result.Error = "expected records not found in response"
			return result, fmt.Errorf("expected records not found in response")
		}
	}

	// Form DNS result
	var resolvedIPs []string
	for _, ans := range resp.Answer {
		switch record := ans.(type) {
		case *dns.A:
			resolvedIPs = append(resolvedIPs, record.A.String())
		case *dns.AAAA:
			resolvedIPs = append(resolvedIPs, record.AAAA.String())
		}
	}

	result.DNSResult = &types.DNSResult{
		ResolvedIPs: resolvedIPs,
		LookupTime:  result.RTT,
	}

	// Extract server IP
	addrs, err := net.LookupIP(server)
	if err == nil && len(addrs) > 0 {
		result.TargetIP = addrs[0].String()
	}

	result.Success = true
	result.TargetPort = port
	result.Role = "client"

	return result, nil
}

// parseQueryType parses DNS query type
func (p *DNSProbe) parseQueryType(qtype string) (uint16, error) {
	switch strings.ToUpper(qtype) {
	case "A":
		return dns.TypeA, nil
	case "AAAA":
		return dns.TypeAAAA, nil
	case "CNAME":
		return dns.TypeCNAME, nil
	case "MX":
		return dns.TypeMX, nil
	case "TXT":
		return dns.TypeTXT, nil
	case "NS":
		return dns.TypeNS, nil
	case "SOA":
		return dns.TypeSOA, nil
	case "SRV":
		return dns.TypeSRV, nil
	case "PTR":
		return dns.TypePTR, nil
	case "CAA":
		return dns.TypeCAA, nil
	case "DNSKEY":
		return dns.TypeDNSKEY, nil
	case "DS":
		return dns.TypeDS, nil
	default:
		return 0, fmt.Errorf("unsupported DNS query type: %s", qtype)
	}
}

// Type returns the probe type
func (p *DNSProbe) Type() types.ProbeType {
	return types.ProbeTypeDNS
}

// Validate validates the configuration
func (p *DNSProbe) Validate(config interface{}) error {
	if p.config == nil {
		return fmt.Errorf("DNS config is nil")
	}

	// Check query type
	_, err := p.parseQueryType(p.config.QueryType)
	if err != nil {
		return err
	}

	// Check protocol
	validProtocols := map[string]bool{
		"udp": true, "tcp": true, "tcp-tls": true, "dot": true,
	}
	if !validProtocols[p.config.Protocol] {
		return fmt.Errorf("invalid DNS protocol: %s", p.config.Protocol)
	}

	return nil
}

// Close releases resources
func (p *DNSProbe) Close() error {
	return nil
}

// BuildDNSConfigFromTarget creates DNS configuration from Target
func BuildDNSConfigFromTarget(target types.Target) *DNSConfig {
	config := &DNSConfig{
		QueryName:      "",
		QueryType:      "A",
		Protocol:       "udp",
		ValidateAnswer: true,
		Recursion:      true,
		Timeout:        target.Timeout,
	}

	if target.DNS != nil {
		if target.DNS.QueryName != "" {
			config.QueryName = target.DNS.QueryName
		}
		if target.DNS.QueryType != "" {
			config.QueryType = target.DNS.QueryType
		}
		if target.DNS.Protocol != "" {
			config.Protocol = target.DNS.Protocol
		}
		config.ExpectedRecords = target.DNS.ExpectedRecords
		config.ValidateAnswer = target.DNS.ValidateAnswer
		config.Recursion = target.DNS.Recursion
	}

	return config
}
