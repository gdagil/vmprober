package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"

	"github.com/gdagil/vmprober/internal/scheduler"
	"github.com/gdagil/vmprober/internal/types"
)

// newTestLogger returns a quiet logger for use in tests.
func newTestLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	return logger
}

// newTestLoggerWithHook returns a quiet logger plus a hook that captures the
// entries it emits, so tests can assert what was logged. Only Error-level and
// above are recorded (the logger level is ErrorLevel).
func newTestLoggerWithHook() (*logrus.Logger, *logrustest.Hook) {
	logger, hook := logrustest.NewNullLogger()
	logger.SetLevel(logrus.ErrorLevel)
	return logger, hook
}

// failingResponseWriter is an http.ResponseWriter whose Write always fails,
// used to exercise the error-logging branches of the handlers.
type failingResponseWriter struct {
	header http.Header
	status int
}

func (f *failingResponseWriter) Header() http.Header {
	if f.header == nil {
		f.header = make(http.Header)
	}
	return f.header
}

func (f *failingResponseWriter) Write([]byte) (int, error) {
	return 0, errors.New("simulated write failure")
}

func (f *failingResponseWriter) WriteHeader(statusCode int) {
	f.status = statusCode
}

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

func TestServer_SetScheduler(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	server := NewServer("127.0.0.1", 0, logger)

	// Create a mock scheduler
	scheduler := &scheduler.Scheduler{}
	server.SetScheduler(scheduler)

	if server.scheduler != scheduler {
		t.Error("Scheduler was not set correctly")
	}
}

func TestServer_JobsHandler(t *testing.T) {
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

	// Create scheduler and add some jobs
	sched := scheduler.NewScheduler(logger)
	server.SetScheduler(sched)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = server.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer server.Stop(context.Background())

	// Give time for server to start
	time.Sleep(200 * time.Millisecond)

	// Test /api/v1/jobs endpoint
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/api/v1/jobs", port))
	if err != nil {
		t.Fatalf("Failed to GET /api/v1/jobs: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if resp.Header.Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", resp.Header.Get("Content-Type"))
	}
}

func TestServer_JobsHandler_NoScheduler(t *testing.T) {
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
	// Don't set scheduler

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = server.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer server.Stop(context.Background())

	// Give time for server to start
	time.Sleep(200 * time.Millisecond)

	// Test /api/v1/jobs endpoint without scheduler
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/api/v1/jobs", port))
	if err != nil {
		t.Fatalf("Failed to GET /api/v1/jobs: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", resp.StatusCode)
	}
}

func TestServer_StatsHandler(t *testing.T) {
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

	// Create scheduler
	sched := scheduler.NewScheduler(logger)
	server.SetScheduler(sched)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = server.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer server.Stop(context.Background())

	// Give time for server to start
	time.Sleep(200 * time.Millisecond)

	// Test /api/v1/stats endpoint
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/api/v1/stats", port))
	if err != nil {
		t.Fatalf("Failed to GET /api/v1/stats: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if resp.Header.Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", resp.Header.Get("Content-Type"))
	}
}

func TestServer_StatsHandler_NoScheduler(t *testing.T) {
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
	// Don't set scheduler

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = server.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer server.Stop(context.Background())

	// Give time for server to start
	time.Sleep(200 * time.Millisecond)

	// Test /api/v1/stats endpoint without scheduler
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/api/v1/stats", port))
	if err != nil {
		t.Fatalf("Failed to GET /api/v1/stats: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", resp.StatusCode)
	}
}

func TestServer_TargetsHandler(t *testing.T) {
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

	// Create scheduler and add a job
	sched := scheduler.NewScheduler(logger)
	ctx := context.Background()

	job := &types.Job{
		ID:       "test-job-1",
		NextRun:  time.Now(),
		Interval: 30 * time.Second,
		Priority: 1,
		Target: types.Target{
			ID:       "test-target",
			Host:     "127.0.0.1",
			Port:     80,
			Protocol: types.ProbeTypeTCP,
		},
		CreatedAt: time.Now(),
	}

	if err := sched.Schedule(ctx, job); err != nil {
		t.Fatalf("Failed to schedule job: %v", err)
	}

	server.SetScheduler(sched)

	err = server.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer server.Stop(context.Background())

	// Give time for server to start
	time.Sleep(200 * time.Millisecond)

	// Test /api/v1/targets endpoint
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/api/v1/targets", port))
	if err != nil {
		t.Fatalf("Failed to GET /api/v1/targets: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if resp.Header.Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", resp.Header.Get("Content-Type"))
	}
}

func TestServer_TargetsHandler_NoScheduler(t *testing.T) {
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
	// Don't set scheduler

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = server.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer server.Stop(context.Background())

	// Give time for server to start
	time.Sleep(200 * time.Millisecond)

	// Test /api/v1/targets endpoint without scheduler
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/api/v1/targets", port))
	if err != nil {
		t.Fatalf("Failed to GET /api/v1/targets: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", resp.StatusCode)
	}
}

func TestServer_UIHandler(t *testing.T) {
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

	// Test root endpoint (UI)
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/", port))
	if err != nil {
		t.Fatalf("Failed to GET /: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if resp.Header.Get("Content-Type") != "text/html; charset=utf-8" {
		t.Errorf("Expected Content-Type text/html; charset=utf-8, got %s", resp.Header.Get("Content-Type"))
	}
}

func TestServer_UIHandler_NotFound(t *testing.T) {
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

	// Test non-root path (should return 404)
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/nonexistent", port))
	if err != nil {
		t.Fatalf("Failed to GET /nonexistent: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}
}

// TestServer_UIHandler_Direct drives uiHandler through the mux with an
// in-memory recorder (no network) for both the root and not-found branches.
func TestServer_UIHandler_Direct(t *testing.T) {
	server := NewServer("127.0.0.1", 0, newTestLogger())
	handler := server.httpServer.Handler

	tests := []struct {
		name            string
		path            string
		wantStatus      int
		wantContentType string
		wantBody        bool
	}{
		{
			name:            "root serves html",
			path:            "/",
			wantStatus:      http.StatusOK,
			wantContentType: "text/html; charset=utf-8",
			wantBody:        true,
		},
		{
			name:       "non-root returns 404",
			path:       "/does-not-exist",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "nested non-root returns 404",
			path:       "/api/v1/unknown",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("Expected status %d, got %d", tt.wantStatus, rec.Code)
			}
			if tt.wantContentType != "" && rec.Header().Get("Content-Type") != tt.wantContentType {
				t.Errorf("Expected Content-Type %q, got %q", tt.wantContentType, rec.Header().Get("Content-Type"))
			}
			if tt.wantBody && rec.Body.Len() == 0 {
				t.Error("Expected non-empty response body")
			}
		})
	}
}

// TestServer_UIHandler_WriteError exercises the branch where writing the HTML
// body fails; the handler should log and return without panicking.
func TestServer_UIHandler_WriteError(t *testing.T) {
	logger, hook := newTestLoggerWithHook()
	server := NewServer("127.0.0.1", 0, logger)

	fw := &failingResponseWriter{}
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	// Should not panic even though Write fails.
	server.uiHandler(fw, req)

	// The handler sets the content type header before writing.
	if got := fw.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Errorf("Expected Content-Type text/html; charset=utf-8, got %q", got)
	}

	// A 200 status must have been written before the body write failed.
	if fw.status != http.StatusOK {
		t.Errorf("Expected status %d written before write failure, got %d", http.StatusOK, fw.status)
	}

	// The write failure must be logged at Error level with the underlying error.
	entry := hook.LastEntry()
	if entry == nil {
		t.Fatal("Expected the write failure to be logged, got no log entry")
	}
	if entry.Level != logrus.ErrorLevel {
		t.Errorf("Expected Error level log, got %s", entry.Level)
	}
	if entry.Message != "Failed to write HTML content" {
		t.Errorf("Expected 'Failed to write HTML content' log, got %q", entry.Message)
	}
	if _, ok := entry.Data[logrus.ErrorKey]; !ok {
		t.Error("Expected the log entry to carry the underlying error")
	}
}

// TestServer_StaticAssets verifies the embedded static file server route,
// including a 404 for a missing asset.
func TestServer_StaticAssets(t *testing.T) {
	server := NewServer("127.0.0.1", 0, newTestLogger())
	handler := server.httpServer.Handler

	tests := []struct {
		name       string
		path       string
		wantStatus int
	}{
		{name: "favicon", path: "/static/favicon.svg", wantStatus: http.StatusOK},
		{name: "css asset", path: "/static/css/custom.css", wantStatus: http.StatusOK},
		{name: "js asset", path: "/static/js/app.js", wantStatus: http.StatusOK},
		{name: "missing asset", path: "/static/nope.txt", wantStatus: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("Expected status %d for %s, got %d", tt.wantStatus, tt.path, rec.Code)
			}
			if tt.wantStatus == http.StatusOK && rec.Body.Len() == 0 {
				t.Errorf("Expected non-empty body for %s", tt.path)
			}
		})
	}
}

// TestServer_TargetsHandler_StatusBranches covers the up/down/unknown status
// mapping in targetsHandler by scheduling jobs in each state.
func TestServer_TargetsHandler_StatusBranches(t *testing.T) {
	logger := newTestLogger()
	server := NewServer("127.0.0.1", 0, logger)

	sched := scheduler.NewScheduler(logger)
	ctx := context.Background()

	mkJob := func(id string) *types.Job {
		return &types.Job{
			ID:       id,
			NextRun:  time.Now(),
			Interval: 30 * time.Second,
			Priority: 1,
			Target: types.Target{
				ID:       id,
				Host:     "127.0.0.1",
				Port:     80,
				Protocol: types.ProbeTypeTCP,
			},
			CreatedAt: time.Now(),
		}
	}

	for _, id := range []string{"job-up", "job-down", "job-unknown"} {
		if err := sched.Schedule(ctx, mkJob(id)); err != nil {
			t.Fatalf("Failed to schedule %s: %v", id, err)
		}
	}

	// Drive the jobs into distinct states.
	sched.MarkJobCompleted("job-up") // LastStatus == "up"
	sched.MarkJobFailed("job-down")  // LastStatus == "down"
	// job-unknown left untouched -> LastStatus == "" -> "unknown"

	server.SetScheduler(sched)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/targets", nil)
	server.httpServer.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %q", ct)
	}

	var targets []map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &targets); err != nil {
		t.Fatalf("Failed to decode targets JSON: %v", err)
	}
	if len(targets) != 3 {
		t.Fatalf("Expected 3 targets, got %d", len(targets))
	}

	statusByID := make(map[string]string, len(targets))
	for _, tgt := range targets {
		id, _ := tgt["id"].(string)
		status, _ := tgt["status"].(string)
		statusByID[id] = status
	}

	want := map[string]string{
		"job-up":      "up",
		"job-down":    "down",
		"job-unknown": "unknown",
	}
	for id, wantStatus := range want {
		if got := statusByID[id]; got != wantStatus {
			t.Errorf("Job %s: expected status %q, got %q", id, wantStatus, got)
		}
	}
}

// TestServer_Handlers_EncodeError exercises the JSON-encoding error branches of
// the API handlers using a ResponseWriter whose Write always fails. The
// handlers should log the failure without panicking.
func TestServer_Handlers_EncodeError(t *testing.T) {
	logger, hook := newTestLoggerWithHook()
	server := NewServer("127.0.0.1", 0, logger)

	sched := scheduler.NewScheduler(logger)
	if err := sched.Schedule(context.Background(), &types.Job{
		ID:       "encode-job",
		NextRun:  time.Now(),
		Interval: 30 * time.Second,
		Target: types.Target{
			ID:       "encode-job",
			Host:     "127.0.0.1",
			Port:     80,
			Protocol: types.ProbeTypeTCP,
		},
		CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("Failed to schedule job: %v", err)
	}
	server.SetScheduler(sched)

	tests := []struct {
		name    string
		handler http.HandlerFunc
		path    string
	}{
		{name: "jobs", handler: server.jobsHandler, path: "/api/v1/jobs"},
		{name: "stats", handler: server.statsHandler, path: "/api/v1/stats"},
		{name: "targets", handler: server.targetsHandler, path: "/api/v1/targets"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hook.Reset()
			fw := &failingResponseWriter{}
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)

			// Must not panic even though encoding to the writer fails.
			tt.handler(fw, req)

			if got := fw.Header().Get("Content-Type"); got != "application/json" {
				t.Errorf("Expected Content-Type application/json, got %q", got)
			}

			// A 200 status must have been written before the encode write failed.
			if fw.status != http.StatusOK {
				t.Errorf("Expected status %d written before encode failure, got %d", http.StatusOK, fw.status)
			}

			// The encode failure must be logged at Error level with the error.
			entry := hook.LastEntry()
			if entry == nil {
				t.Fatalf("Expected the encode failure to be logged for %s, got no log entry", tt.name)
			}
			if entry.Level != logrus.ErrorLevel {
				t.Errorf("Expected Error level log, got %s", entry.Level)
			}
			if !strings.Contains(entry.Message, "Failed to encode") {
				t.Errorf("Expected a 'Failed to encode' log, got %q", entry.Message)
			}
			if _, ok := entry.Data[logrus.ErrorKey]; !ok {
				t.Error("Expected the log entry to carry the underlying error")
			}
		})
	}
}
