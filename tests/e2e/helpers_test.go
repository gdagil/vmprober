package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Test environment constants
const (
	dockerComposeFile = "../../docker-compose.yml"
	testConfigFile    = "test_config.yaml"
	vmInsertURL       = "http://localhost:8480"
	vmSelectURL       = "http://localhost:8481"
	vmStorageURL      = "http://localhost:8482"
	grafanaURL        = "http://localhost:3000"
	vmproberBinary    = "../../bin/vmprober"
)

// TestEnvironment holds shared test environment state
type TestEnvironment struct {
	ctx           context.Context
	cancel        context.CancelFunc
	vmproberCmd   *exec.Cmd
	configPath    string
	t             *testing.T
	dockerStarted bool
}

// NewTestEnvironment creates a new test environment
func NewTestEnvironment(t *testing.T, timeout time.Duration) *TestEnvironment {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	return &TestEnvironment{
		ctx:    ctx,
		cancel: cancel,
		t:      t,
	}
}

// Setup initializes the test environment with docker-compose services
func (e *TestEnvironment) Setup() error {
	// Check if docker-compose is available
	if !commandExists("docker-compose") && !commandExists("docker") {
		e.t.Skip("docker-compose or docker not found, skipping e2e test")
		return nil
	}

	e.t.Log("Starting docker-compose services...")
	if err := startDockerCompose(e.ctx); err != nil {
		return fmt.Errorf("failed to start docker-compose: %w", err)
	}
	e.dockerStarted = true

	e.t.Log("Waiting for services to be ready...")
	if err := waitForServices(e.ctx); err != nil {
		return fmt.Errorf("services failed to become ready: %w", err)
	}

	return nil
}

// Teardown cleans up the test environment
func (e *TestEnvironment) Teardown() {
	if e.vmproberCmd != nil && e.vmproberCmd.Process != nil {
		e.t.Log("Stopping vmprober...")
		e.vmproberCmd.Process.Kill()
		e.vmproberCmd.Wait()
	}

	if e.configPath != "" {
		os.Remove(e.configPath)
	}

	if e.dockerStarted {
		e.t.Log("Stopping docker-compose services...")
		stopDockerCompose(e.ctx)
	}

	e.cancel()
}

// StartVMProber starts vmprober with the given configuration.
func (e *TestEnvironment) StartVMProber(config, logLevel string) error {
	configPath := filepath.Join(os.TempDir(), fmt.Sprintf("vmprober_test_%d.yaml", time.Now().UnixNano()))
	if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	e.configPath = configPath

	e.t.Log("Starting vmprober locally...")
	e.vmproberCmd = exec.CommandContext(e.ctx, vmproberBinary, "-config", configPath, "-log-level", logLevel)
	e.vmproberCmd.Stdout = os.Stdout
	e.vmproberCmd.Stderr = os.Stderr

	if err := e.vmproberCmd.Start(); err != nil {
		return fmt.Errorf("failed to start vmprober: %w", err)
	}

	return nil
}

// WaitForProbes waits for probes to execute
func (e *TestEnvironment) WaitForProbes(duration time.Duration) {
	e.t.Logf("Waiting %v for probes to execute...", duration)
	time.Sleep(duration)
}

// Context returns the test context
func (e *TestEnvironment) Context() context.Context {
	return e.ctx
}

// startDockerCompose starts docker-compose services
func startDockerCompose(ctx context.Context) error {
	var cmd *exec.Cmd
	if commandExists("docker-compose") {
		cmd = exec.CommandContext(ctx, "docker-compose", "-f", dockerComposeFile, "up", "-d")
	} else {
		cmd = exec.CommandContext(ctx, "docker", "compose", "-f", dockerComposeFile, "up", "-d")
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// stopDockerCompose stops docker-compose services
func stopDockerCompose(ctx context.Context) {
	var cmd *exec.Cmd
	if commandExists("docker-compose") {
		cmd = exec.CommandContext(ctx, "docker-compose", "-f", dockerComposeFile, "down", "-v")
	} else {
		cmd = exec.CommandContext(ctx, "docker", "compose", "-f", dockerComposeFile, "down", "-v")
	}
	cmd.Run()
}

// waitForServices waits for all services to become ready
func waitForServices(ctx context.Context) error {
	services := map[string]string{
		"vmstorage": vmStorageURL + "/health",
		"vminsert":  vmInsertURL + "/health",
		"vmselect":  vmSelectURL + "/health",
		"grafana":   grafanaURL + "/api/health",
	}

	client := &http.Client{Timeout: 5 * time.Second}
	maxAttempts := 30
	attemptInterval := 2 * time.Second

	for service, url := range services {
		for i := 0; i < maxAttempts; i++ {
			req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
			if err != nil {
				return fmt.Errorf("failed to create request for %s: %w", service, err)
			}

			resp, err := client.Do(req)
			if err == nil && resp.StatusCode == http.StatusOK {
				resp.Body.Close()
				break
			}
			if resp != nil {
				resp.Body.Close()
			}

			if i == maxAttempts-1 {
				return fmt.Errorf("service %s failed to become ready after %d attempts", service, maxAttempts)
			}

			time.Sleep(attemptInterval)
		}
	}

	return nil
}

// commandExists checks if command exists in PATH
func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// VMQueryResult represents VictoriaMetrics query result
type VMQueryResult struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Value  []interface{}     `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

// VMLabelsResult represents VictoriaMetrics label values result
type VMLabelsResult struct {
	Status string   `json:"status"`
	Data   []string `json:"data"`
}

// QueryMetric queries a metric from VictoriaMetrics
func QueryMetric(ctx context.Context, metricName string) (*VMQueryResult, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	queryURL := vmSelectURL + "/select/0/prometheus/api/v1/query?query=" + metricName

	req, err := http.NewRequestWithContext(ctx, "GET", queryURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to query metrics: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	var result VMQueryResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// GetMetricNames returns all metric names from VictoriaMetrics
func GetMetricNames(ctx context.Context) ([]string, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	queryURL := vmSelectURL + "/select/0/prometheus/api/v1/label/__name__/values"

	req, err := http.NewRequestWithContext(ctx, "GET", queryURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to query metrics: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	var result VMLabelsResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("query failed with status: %s", result.Status)
	}

	return result.Data, nil
}

// VerifyMetricsExist verifies that specified metrics exist in VictoriaMetrics
func VerifyMetricsExist(ctx context.Context, expectedMetrics []string) error {
	metricNames, err := GetMetricNames(ctx)
	if err != nil {
		return err
	}

	foundMetrics := make(map[string]bool)
	for _, metric := range metricNames {
		foundMetrics[metric] = true
	}

	missingMetrics := []string{}
	for _, expected := range expectedMetrics {
		if !foundMetrics[expected] {
			missingMetrics = append(missingMetrics, expected)
		}
	}

	if len(missingMetrics) > 0 {
		return fmt.Errorf("missing metrics: %v", missingMetrics)
	}

	return nil
}

// VerifyMetricHasLabels verifies that a metric has specific labels
func VerifyMetricHasLabels(ctx context.Context, metricName string, expectedLabels []string) error {
	result, err := QueryMetric(ctx, metricName)
	if err != nil {
		return err
	}

	if result.Status != "success" {
		return fmt.Errorf("query failed with status: %s", result.Status)
	}

	if len(result.Data.Result) == 0 {
		return fmt.Errorf("no results found for metric %s", metricName)
	}

	// Check that at least one result has all expected labels
	for _, r := range result.Data.Result {
		hasAllLabels := true
		for _, label := range expectedLabels {
			if _, ok := r.Metric[label]; !ok {
				hasAllLabels = false
				break
			}
		}
		if hasAllLabels {
			return nil
		}
	}

	return fmt.Errorf("no metric %s found with all expected labels: %v", metricName, expectedLabels)
}

// RequireSetup is a helper that sets up the environment and fails the test if setup fails
func RequireSetup(t *testing.T, env *TestEnvironment) {
	require.NoError(t, env.Setup(), "Failed to setup test environment")
}
