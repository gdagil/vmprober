package probe

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gdagil/vmprober/internal/types"
	"github.com/gdagil/vmprober/pkg/interfaces"
)

// HTTPProbe implements HTTP/HTTPS probes
type HTTPProbe struct {
	config *HTTPConfig
	client *http.Client
}

// HTTPConfig is the HTTP probe configuration
type HTTPConfig struct {
	Method             string
	Path               string
	Headers            map[string]string
	Body               string
	ExpectedStatusCode int
	ExpectedBody       string
	FollowRedirects    bool
	MaxRedirects       int
	ValidateCert       bool
	BasicAuth          *BasicAuthConfig
	BearerToken        string
	Proxy              string
	Timeout            time.Duration
	TLS                bool
}

// BasicAuthConfig is the Basic Auth configuration
type BasicAuthConfig struct {
	Username string
	Password string
}

// NewHTTPProbe creates a new HTTP probe
func NewHTTPProbe(config *HTTPConfig) interfaces.Probe {
	if config == nil {
		config = &HTTPConfig{}
	}

	// Set defaults
	if config.Method == "" {
		config.Method = "GET"
	}
	if config.ExpectedStatusCode == 0 {
		config.ExpectedStatusCode = 200
	}
	if config.MaxRedirects == 0 {
		config.MaxRedirects = 10
	}
	if config.Timeout == 0 {
		config.Timeout = 10 * time.Second
	}

	// Create TLS config
	tlsConfig := &tls.Config{
		InsecureSkipVerify: !config.ValidateCert,
	}

	// Create transport
	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
		DialContext: (&net.Dialer{
			Timeout:   config.Timeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	// Configure proxy if specified
	if config.Proxy != "" {
		proxyURL, err := url.Parse(config.Proxy)
		if err == nil {
			transport.Proxy = http.ProxyURL(proxyURL)
		}
	}

	// Create client
	client := &http.Client{
		Transport: transport,
		Timeout:   config.Timeout,
	}

	// Configure redirects
	if !config.FollowRedirects {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	} else {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			if len(via) >= config.MaxRedirects {
				return fmt.Errorf("stopped after %d redirects", config.MaxRedirects)
			}
			return nil
		}
	}

	return &HTTPProbe{
		config: config,
		client: client,
	}
}

// Execute executes an HTTP probe
func (p *HTTPProbe) Execute(ctx context.Context, target types.Target) (*types.ProbeResult, error) {
	start := time.Now()

	// Determine protocol
	protocol := types.ProbeTypeHTTP
	scheme := "http"
	if p.config.TLS || target.Protocol == types.ProbeTypeHTTPS {
		protocol = types.ProbeTypeHTTPS
		scheme = "https"
	}

	result := &types.ProbeResult{
		Protocol:   protocol,
		Timestamp:  start,
		Attempt:    1,
		TargetHost: target.Host,
		TargetPort: target.Port,
	}

	// Form URL
	port := target.Port
	if port == 0 {
		if scheme == "https" {
			port = 443
		} else {
			port = 80
		}
	}

	path := p.config.Path
	if path == "" {
		path = "/"
	}

	targetURL := fmt.Sprintf("%s://%s:%d%s", scheme, target.Host, port, path)

	// Create request
	var body io.Reader
	if p.config.Body != "" {
		body = strings.NewReader(p.config.Body)
	}

	req, err := http.NewRequestWithContext(ctx, p.config.Method, targetURL, body)
	if err != nil {
		result.Error = fmt.Sprintf("failed to create request: %v", err)
		result.RTT = time.Since(start)
		return result, err
	}

	// Set headers
	for key, value := range p.config.Headers {
		req.Header.Set(key, value)
	}

	// Set User-Agent if not specified
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "vmprober/1.0")
	}

	// Set authentication
	if p.config.BasicAuth != nil {
		req.SetBasicAuth(p.config.BasicAuth.Username, p.config.BasicAuth.Password)
	}
	if p.config.BearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+p.config.BearerToken)
	}

	// Execute request
	resp, err := p.client.Do(req)
	if err != nil {
		result.Error = fmt.Sprintf("request failed: %v", err)
		result.RTT = time.Since(start)
		return result, err
	}
	defer resp.Body.Close()

	// Read response body
	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024)) // Limit 1MB
	if err != nil {
		result.Error = fmt.Sprintf("failed to read response body: %v", err)
		result.RTT = time.Since(start)
		return result, err
	}

	result.RTT = time.Since(start)
	result.Response = bodyBytes

	// Check status code
	if p.config.ExpectedStatusCode > 0 && resp.StatusCode != p.config.ExpectedStatusCode {
		result.Error = fmt.Sprintf("unexpected status code: got %d, expected %d", resp.StatusCode, p.config.ExpectedStatusCode)
		return result, fmt.Errorf("unexpected status code: got %d, expected %d", resp.StatusCode, p.config.ExpectedStatusCode)
	}

	// Check response body
	if p.config.ExpectedBody != "" && !strings.Contains(string(bodyBytes), p.config.ExpectedBody) {
		result.Error = fmt.Sprintf("expected body substring not found: %s", p.config.ExpectedBody)
		return result, fmt.Errorf("expected body substring not found: %s", p.config.ExpectedBody)
	}

	// Success
	result.Success = true
	result.TLS = scheme == "https"

	// Extract IP address from connection
	if resp.TLS != nil {
		result.TargetIP = resp.TLS.ServerName
	}

	// Get local address through TCP conn
	// This is difficult to do through http.Client, so we use resolved IP
	addrs, err := net.LookupIP(target.Host)
	if err == nil && len(addrs) > 0 {
		result.TargetIP = addrs[0].String()
	}

	result.TargetPort = port
	result.Role = "client"

	return result, nil
}

// Type returns the probe type
func (p *HTTPProbe) Type() types.ProbeType {
	if p.config.TLS {
		return types.ProbeTypeHTTPS
	}
	return types.ProbeTypeHTTP
}

// Validate validates the configuration
func (p *HTTPProbe) Validate(config interface{}) error {
	if p.config == nil {
		return fmt.Errorf("HTTP config is nil")
	}

	// Check method
	validMethods := map[string]bool{
		"GET": true, "POST": true, "PUT": true, "DELETE": true,
		"HEAD": true, "OPTIONS": true, "PATCH": true,
	}
	if !validMethods[p.config.Method] {
		return fmt.Errorf("invalid HTTP method: %s", p.config.Method)
	}

	return nil
}

// Close releases resources
func (p *HTTPProbe) Close() error {
	if p.client != nil {
		p.client.CloseIdleConnections()
	}
	return nil
}

// BuildHTTPConfigFromTarget creates HTTP configuration from Target
func BuildHTTPConfigFromTarget(target types.Target) *HTTPConfig {
	config := &HTTPConfig{
		Method:             "GET",
		Path:               "/",
		ExpectedStatusCode: 200,
		FollowRedirects:    true,
		MaxRedirects:       10,
		ValidateCert:       true,
		Timeout:            target.Timeout,
		TLS:                target.Protocol == types.ProbeTypeHTTPS,
	}

	if target.HTTP != nil {
		if target.HTTP.Method != "" {
			config.Method = target.HTTP.Method
		}
		if target.HTTP.Path != "" {
			config.Path = target.HTTP.Path
		}
		if target.HTTP.ExpectedStatusCode > 0 {
			config.ExpectedStatusCode = target.HTTP.ExpectedStatusCode
		}
		config.Headers = target.HTTP.Headers
		config.Body = target.HTTP.Body
		config.ExpectedBody = target.HTTP.ExpectedBody
		config.FollowRedirects = target.HTTP.FollowRedirects
		if target.HTTP.MaxRedirects > 0 {
			config.MaxRedirects = target.HTTP.MaxRedirects
		}
		config.ValidateCert = target.HTTP.ValidateCert
		if target.HTTP.BasicAuth != nil {
			config.BasicAuth = &BasicAuthConfig{
				Username: target.HTTP.BasicAuth.Username,
				Password: target.HTTP.BasicAuth.Password,
			}
		}
		config.BearerToken = target.HTTP.BearerToken
		config.Proxy = target.HTTP.Proxy
	}

	// If using global TLS config
	if target.TLS != nil && target.TLS.Enabled {
		config.TLS = true
		config.ValidateCert = !target.TLS.InsecureSkipVerify
	}

	return config
}

// FormatHTTPTarget formats target for output
func FormatHTTPTarget(target types.Target) string {
	scheme := "http"
	if target.Protocol == types.ProbeTypeHTTPS {
		scheme = "https"
	}

	port := target.Port
	if port == 0 {
		if scheme == "https" {
			port = 443
		} else {
			port = 80
		}
	}

	path := "/"
	if target.HTTP != nil && target.HTTP.Path != "" {
		path = target.HTTP.Path
	}

	return fmt.Sprintf("%s://%s:%s%s", scheme, target.Host, strconv.Itoa(port), path)
}
