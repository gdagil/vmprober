package probe

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gdagil/vmprober/internal/types"
)

func TestNewHTTPProbe(t *testing.T) {
	config := &HTTPConfig{
		Method:             "GET",
		ExpectedStatusCode: 200,
	}

	probe := NewHTTPProbe(config)
	if probe == nil {
		t.Fatal("NewHTTPProbe returned nil")
	}

	if probe.Type() != types.ProbeTypeHTTP {
		t.Errorf("Expected type HTTP, got %s", probe.Type())
	}
}

func TestNewHTTPProbe_WithTLS(t *testing.T) {
	config := &HTTPConfig{
		Method: "GET",
		TLS:    true,
	}

	probe := NewHTTPProbe(config)
	if probe == nil {
		t.Fatal("NewHTTPProbe returned nil")
	}

	if probe.Type() != types.ProbeTypeHTTPS {
		t.Errorf("Expected type HTTPS, got %s", probe.Type())
	}
}

func TestNewHTTPProbe_Defaults(t *testing.T) {
	probe := NewHTTPProbe(nil).(*HTTPProbe)

	if probe.config.Method != "GET" {
		t.Errorf("Expected default method GET, got %s", probe.config.Method)
	}
	if probe.config.ExpectedStatusCode != 200 {
		t.Errorf("Expected default status code 200, got %d", probe.config.ExpectedStatusCode)
	}
	if probe.config.MaxRedirects != 10 {
		t.Errorf("Expected default max redirects 10, got %d", probe.config.MaxRedirects)
	}
	if probe.config.Timeout != 10*time.Second {
		t.Errorf("Expected default timeout 10s, got %v", probe.config.Timeout)
	}
}

func TestHTTPProbe_Execute_Success(t *testing.T) {
	// Start a test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	config := &HTTPConfig{
		Method:             "GET",
		ExpectedStatusCode: 200,
		Timeout:            5 * time.Second,
	}
	probe := NewHTTPProbe(config)
	defer probe.Close()

	// Parse test server URL
	target := types.Target{
		Host:     "127.0.0.1",
		Port:     getPortFromURL(server.URL),
		Protocol: types.ProbeTypeHTTP,
		Timeout:  5 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := probe.Execute(ctx, target)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
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
	if result.Protocol != types.ProbeTypeHTTP {
		t.Errorf("Expected protocol HTTP, got %s", result.Protocol)
	}
}

func TestHTTPProbe_Execute_WithPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("healthy"))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	config := &HTTPConfig{
		Method:             "GET",
		Path:               "/health",
		ExpectedStatusCode: 200,
		Timeout:            5 * time.Second,
	}
	probe := NewHTTPProbe(config)
	defer probe.Close()

	target := types.Target{
		Host:     "127.0.0.1",
		Port:     getPortFromURL(server.URL),
		Protocol: types.ProbeTypeHTTP,
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

func TestHTTPProbe_Execute_POST(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	config := &HTTPConfig{
		Method:             "POST",
		Body:               `{"test": "data"}`,
		ExpectedStatusCode: 201,
		Timeout:            5 * time.Second,
	}
	probe := NewHTTPProbe(config)
	defer probe.Close()

	target := types.Target{
		Host:     "127.0.0.1",
		Port:     getPortFromURL(server.URL),
		Protocol: types.ProbeTypeHTTP,
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

func TestHTTPProbe_Execute_ExpectedBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	}))
	defer server.Close()

	config := &HTTPConfig{
		Method:             "GET",
		ExpectedStatusCode: 200,
		ExpectedBody:       "healthy",
		Timeout:            5 * time.Second,
	}
	probe := NewHTTPProbe(config)
	defer probe.Close()

	target := types.Target{
		Host:     "127.0.0.1",
		Port:     getPortFromURL(server.URL),
		Protocol: types.ProbeTypeHTTP,
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

func TestHTTPProbe_Execute_ExpectedBody_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("something else"))
	}))
	defer server.Close()

	config := &HTTPConfig{
		Method:             "GET",
		ExpectedStatusCode: 200,
		ExpectedBody:       "healthy",
		Timeout:            5 * time.Second,
	}
	probe := NewHTTPProbe(config)
	defer probe.Close()

	target := types.Target{
		Host:     "127.0.0.1",
		Port:     getPortFromURL(server.URL),
		Protocol: types.ProbeTypeHTTP,
		Timeout:  5 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, _ := probe.Execute(ctx, target)
	if result.Success {
		t.Error("Expected failed probe due to body mismatch")
	}
	if result.Error == "" {
		t.Error("Expected error message")
	}
}

func TestHTTPProbe_Execute_WrongStatusCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	config := &HTTPConfig{
		Method:             "GET",
		ExpectedStatusCode: 200,
		Timeout:            5 * time.Second,
	}
	probe := NewHTTPProbe(config)
	defer probe.Close()

	target := types.Target{
		Host:     "127.0.0.1",
		Port:     getPortFromURL(server.URL),
		Protocol: types.ProbeTypeHTTP,
		Timeout:  5 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, _ := probe.Execute(ctx, target)
	if result.Success {
		t.Error("Expected failed probe due to status code mismatch")
	}
}

func TestHTTPProbe_Execute_BasicAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "admin" || pass != "secret" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &HTTPConfig{
		Method:             "GET",
		ExpectedStatusCode: 200,
		BasicAuth: &BasicAuthConfig{
			Username: "admin",
			Password: "secret",
		},
		Timeout: 5 * time.Second,
	}
	probe := NewHTTPProbe(config)
	defer probe.Close()

	target := types.Target{
		Host:     "127.0.0.1",
		Port:     getPortFromURL(server.URL),
		Protocol: types.ProbeTypeHTTP,
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

func TestHTTPProbe_Execute_BearerToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &HTTPConfig{
		Method:             "GET",
		ExpectedStatusCode: 200,
		BearerToken:        "test-token",
		Timeout:            5 * time.Second,
	}
	probe := NewHTTPProbe(config)
	defer probe.Close()

	target := types.Target{
		Host:     "127.0.0.1",
		Port:     getPortFromURL(server.URL),
		Protocol: types.ProbeTypeHTTP,
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

func TestHTTPProbe_Execute_CustomHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Custom-Header") != "test-value" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &HTTPConfig{
		Method:             "GET",
		ExpectedStatusCode: 200,
		Headers: map[string]string{
			"X-Custom-Header": "test-value",
		},
		Timeout: 5 * time.Second,
	}
	probe := NewHTTPProbe(config)
	defer probe.Close()

	target := types.Target{
		Host:     "127.0.0.1",
		Port:     getPortFromURL(server.URL),
		Protocol: types.ProbeTypeHTTP,
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

func TestHTTPProbe_Execute_NoRedirects(t *testing.T) {
	redirectCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if redirectCount < 3 {
			redirectCount++
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &HTTPConfig{
		Method:             "GET",
		ExpectedStatusCode: 302, // Expect redirect, not final result
		FollowRedirects:    false,
		Timeout:            5 * time.Second,
	}
	probe := NewHTTPProbe(config)
	defer probe.Close()

	target := types.Target{
		Host:     "127.0.0.1",
		Port:     getPortFromURL(server.URL),
		Protocol: types.ProbeTypeHTTP,
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

func TestHTTPProbe_Execute_ConnectionRefused(t *testing.T) {
	config := &HTTPConfig{
		Method:             "GET",
		ExpectedStatusCode: 200,
		Timeout:            1 * time.Second,
	}
	probe := NewHTTPProbe(config)
	defer probe.Close()

	target := types.Target{
		Host:     "127.0.0.1",
		Port:     65534, // A port that is definitely not listening
		Protocol: types.ProbeTypeHTTP,
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
}

func TestHTTPProbe_Type(t *testing.T) {
	config := &HTTPConfig{TLS: false}
	probe := NewHTTPProbe(config)
	if probe.Type() != types.ProbeTypeHTTP {
		t.Errorf("Expected type HTTP, got %s", probe.Type())
	}

	configTLS := &HTTPConfig{TLS: true}
	probeTLS := NewHTTPProbe(configTLS)
	if probeTLS.Type() != types.ProbeTypeHTTPS {
		t.Errorf("Expected type HTTPS, got %s", probeTLS.Type())
	}
}

func TestHTTPProbe_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *HTTPConfig
		wantErr bool
	}{
		{
			name:    "valid GET",
			config:  &HTTPConfig{Method: "GET"},
			wantErr: false,
		},
		{
			name:    "valid POST",
			config:  &HTTPConfig{Method: "POST"},
			wantErr: false,
		},
		{
			name:    "valid PUT",
			config:  &HTTPConfig{Method: "PUT"},
			wantErr: false,
		},
		{
			name:    "valid DELETE",
			config:  &HTTPConfig{Method: "DELETE"},
			wantErr: false,
		},
		{
			name:    "valid HEAD",
			config:  &HTTPConfig{Method: "HEAD"},
			wantErr: false,
		},
		{
			name:    "invalid method",
			config:  &HTTPConfig{Method: "INVALID"},
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
			var probe *HTTPProbe
			if tt.config != nil {
				probe = NewHTTPProbe(tt.config).(*HTTPProbe)
			} else {
				probe = &HTTPProbe{config: nil}
			}

			err := probe.Validate(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHTTPProbe_Close(t *testing.T) {
	config := &HTTPConfig{}
	probe := NewHTTPProbe(config)

	if err := probe.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestBuildHTTPConfigFromTarget(t *testing.T) {
	target := types.Target{
		Host:     "example.com",
		Port:     443,
		Protocol: types.ProbeTypeHTTPS,
		Timeout:  15 * time.Second,
		HTTP: &types.HTTPConfig{
			Method:             "POST",
			Path:               "/api/health",
			ExpectedStatusCode: 201,
			Headers: map[string]string{
				"X-Api-Key": "test",
			},
			Body:            `{"test": true}`,
			FollowRedirects: false,
			MaxRedirects:    5,
			ValidateCert:    true,
			BasicAuth: &types.BasicAuth{
				Username: "user",
				Password: "pass",
			},
		},
	}

	config := BuildHTTPConfigFromTarget(target)

	if config.Method != "POST" {
		t.Errorf("Expected method POST, got %s", config.Method)
	}
	if config.Path != "/api/health" {
		t.Errorf("Expected path /api/health, got %s", config.Path)
	}
	if config.ExpectedStatusCode != 201 {
		t.Errorf("Expected status code 201, got %d", config.ExpectedStatusCode)
	}
	if !config.TLS {
		t.Error("Expected TLS to be true for HTTPS")
	}
	if config.BasicAuth == nil {
		t.Error("Expected BasicAuth to be set")
	}
	if config.BasicAuth.Username != "user" {
		t.Errorf("Expected username 'user', got %s", config.BasicAuth.Username)
	}
}

func TestFormatHTTPTarget(t *testing.T) {
	tests := []struct {
		name     string
		target   types.Target
		expected string
	}{
		{
			name: "HTTP default port",
			target: types.Target{
				Host:     "example.com",
				Port:     0,
				Protocol: types.ProbeTypeHTTP,
			},
			expected: "http://example.com:80/",
		},
		{
			name: "HTTPS default port",
			target: types.Target{
				Host:     "example.com",
				Port:     0,
				Protocol: types.ProbeTypeHTTPS,
			},
			expected: "https://example.com:443/",
		},
		{
			name: "Custom port and path",
			target: types.Target{
				Host:     "example.com",
				Port:     8080,
				Protocol: types.ProbeTypeHTTP,
				HTTP: &types.HTTPConfig{
					Path: "/health",
				},
			},
			expected: "http://example.com:8080/health",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatHTTPTarget(tt.target)
			if result != tt.expected {
				t.Errorf("FormatHTTPTarget() = %s, want %s", result, tt.expected)
			}
		})
	}
}

// Helper function to extract port from URL
func getPortFromURL(urlStr string) int {
	// Simple parser for test URLs like "http://127.0.0.1:12345"
	for i := len(urlStr) - 1; i >= 0; i-- {
		if urlStr[i] == ':' {
			port := 0
			for j := i + 1; j < len(urlStr); j++ {
				if urlStr[j] >= '0' && urlStr[j] <= '9' {
					port = port*10 + int(urlStr[j]-'0')
				} else {
					break
				}
			}
			return port
		}
	}
	return 80
}

func TestHTTPProbe_Execute_WithBasicAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok || username != "user" || password != "pass" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	config := &HTTPConfig{
		Method:             "GET",
		ExpectedStatusCode: 200,
		BasicAuth: &BasicAuthConfig{
			Username: "user",
			Password: "pass",
		},
		Timeout: 5 * time.Second,
	}
	probe := NewHTTPProbe(config)
	defer probe.Close()

	target := types.Target{
		Host:     "127.0.0.1",
		Port:     getPortFromURL(server.URL),
		Protocol: types.ProbeTypeHTTP,
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

func TestHTTPProbe_Execute_WithBearerToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer test-token") {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	config := &HTTPConfig{
		Method:             "GET",
		ExpectedStatusCode: 200,
		BearerToken:        "test-token",
		Timeout:            5 * time.Second,
	}
	probe := NewHTTPProbe(config)
	defer probe.Close()

	target := types.Target{
		Host:     "127.0.0.1",
		Port:     getPortFromURL(server.URL),
		Protocol: types.ProbeTypeHTTP,
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

func TestHTTPProbe_Execute_WithExpectedBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("expected content"))
	}))
	defer server.Close()

	config := &HTTPConfig{
		Method:             "GET",
		ExpectedStatusCode: 200,
		ExpectedBody:       "expected",
		Timeout:            5 * time.Second,
	}
	probe := NewHTTPProbe(config)
	defer probe.Close()

	target := types.Target{
		Host:     "127.0.0.1",
		Port:     getPortFromURL(server.URL),
		Protocol: types.ProbeTypeHTTP,
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

func TestHTTPProbe_Execute_WithExpectedBodyNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("different content"))
	}))
	defer server.Close()

	config := &HTTPConfig{
		Method:             "GET",
		ExpectedStatusCode: 200,
		ExpectedBody:       "expected",
		Timeout:            5 * time.Second,
	}
	probe := NewHTTPProbe(config)
	defer probe.Close()

	target := types.Target{
		Host:     "127.0.0.1",
		Port:     getPortFromURL(server.URL),
		Protocol: types.ProbeTypeHTTP,
		Timeout:  5 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, _ := probe.Execute(ctx, target)
	// Should fail because expected body not found
	if result != nil && result.Success {
		t.Error("Expected unsuccessful probe when expected body not found")
	}
}

func TestHTTPProbe_Execute_WithCustomHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Custom-Header") != "custom-value" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	config := &HTTPConfig{
		Method:             "GET",
		ExpectedStatusCode: 200,
		Headers: map[string]string{
			"X-Custom-Header": "custom-value",
		},
		Timeout: 5 * time.Second,
	}
	probe := NewHTTPProbe(config)
	defer probe.Close()

	target := types.Target{
		Host:     "127.0.0.1",
		Port:     getPortFromURL(server.URL),
		Protocol: types.ProbeTypeHTTP,
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

func TestHTTPProbe_Execute_POSTWithBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != "test body" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	config := &HTTPConfig{
		Method:             "POST",
		ExpectedStatusCode: 200,
		Body:               "test body",
		Timeout:            5 * time.Second,
	}
	probe := NewHTTPProbe(config)
	defer probe.Close()

	target := types.Target{
		Host:     "127.0.0.1",
		Port:     getPortFromURL(server.URL),
		Protocol: types.ProbeTypeHTTP,
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

func TestHTTPProbe_Execute_MaxRedirects(t *testing.T) {
	redirectCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if redirectCount < 5 {
			redirectCount++
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &HTTPConfig{
		Method:             "GET",
		ExpectedStatusCode: 200,
		FollowRedirects:    true,
		MaxRedirects:       3, // Less than actual redirects
		Timeout:            5 * time.Second,
	}
	probe := NewHTTPProbe(config)
	defer probe.Close()

	target := types.Target{
		Host:     "127.0.0.1",
		Port:     getPortFromURL(server.URL),
		Protocol: types.ProbeTypeHTTP,
		Timeout:  5 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, _ := probe.Execute(ctx, target)
	// Should fail due to max redirects exceeded
	if result != nil && result.Success {
		t.Error("Expected unsuccessful probe when max redirects exceeded")
	}
}

func TestHTTPProbe_Execute_WrongStatusCode2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not Found"))
	}))
	defer server.Close()

	config := &HTTPConfig{
		Method:             "GET",
		ExpectedStatusCode: 200,
		Timeout:            5 * time.Second,
	}
	probe := NewHTTPProbe(config)
	defer probe.Close()

	target := types.Target{
		Host:     "127.0.0.1",
		Port:     getPortFromURL(server.URL),
		Protocol: types.ProbeTypeHTTP,
		Timeout:  5 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := probe.Execute(ctx, target)
	// Should fail due to wrong status code
	if err == nil {
		t.Error("Expected error for wrong status code")
	}
	if result != nil && result.Success {
		t.Error("Expected unsuccessful probe for wrong status code")
	}
}

func TestHTTPProbe_Execute_HTTPS(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	config := &HTTPConfig{
		Method:             "GET",
		ExpectedStatusCode: 200,
		TLS:                true,
		ValidateCert:       false, // Skip cert validation for test
		Timeout:            5 * time.Second,
	}
	probe := NewHTTPProbe(config)
	defer probe.Close()

	target := types.Target{
		Host:     "127.0.0.1",
		Port:     getPortFromURL(server.URL),
		Protocol: types.ProbeTypeHTTPS,
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
	if !result.TLS {
		t.Error("Expected TLS to be true for HTTPS probe")
	}
}

func TestHTTPProbe_Execute_WithDifferentMethods(t *testing.T) {
	methods := []string{"GET", "POST", "PUT", "DELETE", "HEAD", "PATCH"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != method {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("OK"))
			}))
			defer server.Close()

			config := &HTTPConfig{
				Method:             method,
				ExpectedStatusCode: 200,
				Timeout:            5 * time.Second,
			}
			probe := NewHTTPProbe(config)
			defer probe.Close()

			target := types.Target{
				Host:     "127.0.0.1",
				Port:     getPortFromURL(server.URL),
				Protocol: types.ProbeTypeHTTP,
				Timeout:  5 * time.Second,
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			result, err := probe.Execute(ctx, target)
			if err != nil {
				t.Logf("HTTP probe with method %s failed (may be expected): %v", method, err)
			}
			if result != nil {
				t.Logf("HTTP probe result for %s: Success=%v", method, result.Success)
			}
		})
	}
}

func TestHTTPProbe_Execute_WithDefaultUserAgent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userAgent := r.Header.Get("User-Agent")
		if userAgent == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	config := &HTTPConfig{
		Method:             "GET",
		ExpectedStatusCode: 200,
		Timeout:            5 * time.Second,
		// No User-Agent header set - should use default
	}
	probe := NewHTTPProbe(config)
	defer probe.Close()

	target := types.Target{
		Host:     "127.0.0.1",
		Port:     getPortFromURL(server.URL),
		Protocol: types.ProbeTypeHTTP,
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

func TestHTTPProbe_Execute_WithCustomUserAgent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userAgent := r.Header.Get("User-Agent")
		if userAgent != "custom-agent" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	config := &HTTPConfig{
		Method:             "GET",
		ExpectedStatusCode: 200,
		Headers: map[string]string{
			"User-Agent": "custom-agent",
		},
		Timeout: 5 * time.Second,
	}
	probe := NewHTTPProbe(config)
	defer probe.Close()

	target := types.Target{
		Host:     "127.0.0.1",
		Port:     getPortFromURL(server.URL),
		Protocol: types.ProbeTypeHTTP,
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
