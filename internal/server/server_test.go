package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func TestNewServer(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	server := NewServer("127.0.0.1", 8080, logger)
	if server == nil {
		t.Fatal("NewServer returned nil")
	}
}

func TestServer_StartStop(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	// Use random port for test
	server := NewServer("127.0.0.1", 0, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := server.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Give time for server to start
	time.Sleep(100 * time.Millisecond)

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer stopCancel()

	err = server.Stop(stopCtx)
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}

func TestServer_HealthEndpoint(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	// Find a free port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to find free port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	server := NewServer("127.0.0.1", port, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = server.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer server.Stop(context.Background())

	// Give time for server to start
	time.Sleep(200 * time.Millisecond)

	// Check /health endpoint
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/health", port))
	if err != nil {
		t.Fatalf("Failed to GET /health: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if resp.Header.Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", resp.Header.Get("Content-Type"))
	}
}

func TestServer_ReadyEndpoint(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	// Find a free port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to find free port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	server := NewServer("127.0.0.1", port, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = server.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer server.Stop(context.Background())

	// Give time for server to start
	time.Sleep(200 * time.Millisecond)

	// Check /ready endpoint
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/ready", port))
	if err != nil {
		t.Fatalf("Failed to GET /ready: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if resp.Header.Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", resp.Header.Get("Content-Type"))
	}
}

func TestServer_MetricsEndpoint(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	// Find a free port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to find free port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	server := NewServer("127.0.0.1", port, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = server.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer server.Stop(context.Background())

	// Give time for server to start
	time.Sleep(200 * time.Millisecond)

	// Check /metrics endpoint
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/metrics", port))
	if err != nil {
		t.Fatalf("Failed to GET /metrics: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestServer_ConcurrentRequests(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	// Find a free port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to find free port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	server := NewServer("127.0.0.1", port, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = server.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer server.Stop(context.Background())

	// Give time for server to start
	time.Sleep(200 * time.Millisecond)

	// Send several parallel requests
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	endpoints := []string{"/health", "/ready", "/metrics"}

	for _, endpoint := range endpoints {
		t.Run(endpoint, func(t *testing.T) {
			done := make(chan bool, 10)
			for i := 0; i < 10; i++ {
				go func() {
					resp, err := http.Get(baseURL + endpoint)
					if err == nil {
						resp.Body.Close()
					}
					done <- (err == nil && resp.StatusCode == http.StatusOK)
				}()
			}

			successCount := 0
			for i := 0; i < 10; i++ {
				if <-done {
					successCount++
				}
			}

			if successCount < 8 {
				t.Errorf("Expected at least 8 successful requests, got %d", successCount)
			}
		})
	}
}
