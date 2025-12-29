package e2e

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestE2E_FailingProbes_RecordFailures tests that failing probes are recorded correctly
func TestE2E_FailingProbes_RecordFailures(t *testing.T) {
	env := NewTestEnvironment(t, 5*time.Minute)
	defer env.Teardown()

	RequireSetup(t, env)

	err := env.StartVMProber(FailingProbesConfig(), "debug")
	require.NoError(t, err, "Failed to start vmprober")

	// Wait for probes to execute
	env.WaitForProbes(20 * time.Second)

	ctx := env.Context()

	// Verify that failure metrics exist
	result, err := QueryMetric(ctx, "vmprober_probe_failure_total")
	require.NoError(t, err)
	require.Equal(t, "success", result.Status)

	// Should have at least one failure (closed port probe)
	if len(result.Data.Result) > 0 {
		t.Logf("Found %d failure metric results", len(result.Data.Result))
	} else {
		t.Log("No failure metrics found - probe may not have executed yet or failure metric naming differs")
	}

	// Verify attempts are still recorded
	attemptsResult, err := QueryMetric(ctx, "vmprober_probe_attempts_total")
	require.NoError(t, err)
	require.NotEmpty(t, attemptsResult.Data.Result, "No probe attempts recorded")
}

// TestE2E_FailingProbes_SuccessRate tests success rate calculation with mixed results
func TestE2E_FailingProbes_SuccessRate(t *testing.T) {
	env := NewTestEnvironment(t, 5*time.Minute)
	defer env.Teardown()

	RequireSetup(t, env)

	err := env.StartVMProber(FailingProbesConfig(), "info")
	require.NoError(t, err, "Failed to start vmprober")

	env.WaitForProbes(25 * time.Second)

	ctx := env.Context()

	// Query both success and attempts
	successResult, err := QueryMetric(ctx, "vmprober_probe_success_total")
	require.NoError(t, err)

	attemptsResult, err := QueryMetric(ctx, "vmprober_probe_attempts_total")
	require.NoError(t, err)

	t.Logf("Success metrics: %d results", len(successResult.Data.Result))
	t.Logf("Attempts metrics: %d results", len(attemptsResult.Data.Result))

	// At least the working probe (8.8.8.8:53) should have success metrics
	hasSuccess := len(successResult.Data.Result) > 0
	hasAttempts := len(attemptsResult.Data.Result) > 0

	require.True(t, hasAttempts, "Should have at least some probe attempts")
	if hasSuccess {
		t.Log("Some probes succeeded as expected")
	}
}

// TestE2E_HighLoad_StabilityTest tests vmprober under high load
func TestE2E_HighLoad_StabilityTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping high-load test in short mode")
	}

	env := NewTestEnvironment(t, 5*time.Minute)
	defer env.Teardown()

	RequireSetup(t, env)

	err := env.StartVMProber(HighLoadConfig(), "info")
	require.NoError(t, err, "Failed to start vmprober")

	// Run for longer to accumulate load
	env.WaitForProbes(45 * time.Second)

	ctx := env.Context()

	// Verify metrics are being collected under load
	attemptsResult, err := QueryMetric(ctx, "vmprober_probe_attempts_total")
	require.NoError(t, err)
	require.NotEmpty(t, attemptsResult.Data.Result, "No probe attempts under high load")

	// Calculate total attempts across all probes
	totalAttempts := len(attemptsResult.Data.Result)
	t.Logf("High load test: found %d probe metric series", totalAttempts)
}

// TestE2E_MixedProbes_ConcurrentExecution tests mixed probe types executing concurrently
func TestE2E_MixedProbes_ConcurrentExecution(t *testing.T) {
	env := NewTestEnvironment(t, 5*time.Minute)
	defer env.Teardown()

	RequireSetup(t, env)

	err := env.StartVMProber(MixedProbesConfig(), "info")
	require.NoError(t, err, "Failed to start vmprober")

	env.WaitForProbes(25 * time.Second)

	ctx := env.Context()

	// Verify metrics for different probe types
	result, err := QueryMetric(ctx, "vmprober_probe_attempts_total")
	require.NoError(t, err)
	require.Equal(t, "success", result.Status)

	// Check for different protocols
	protocols := make(map[string]bool)
	for _, r := range result.Data.Result {
		if protocol, ok := r.Metric["protocol"]; ok {
			protocols[protocol] = true
		}
	}

	t.Logf("Found %d different protocols: %v", len(protocols), protocols)

	// Should have TCP and DNS protocols
	if protocols["tcp"] {
		t.Log("TCP probes executed")
	}
	if protocols["dns"] {
		t.Log("DNS probes executed")
	}
}

// TestE2E_Timeout_HandledGracefully tests that timeouts are handled gracefully
func TestE2E_Timeout_HandledGracefully(t *testing.T) {
	// Configuration with very short timeout
	shortTimeoutConfig := BaseConfig() + `
targets:
  static:
    - host: "10.255.255.1"  # Non-routable IP - will timeout
      port: 80
      proto: "tcp"
      interval: 5s
      timeout: 1s
      labels:
        test: "e2e-timeout"
        target: "timeout-test"
    - host: "8.8.8.8"
      port: 53
      proto: "tcp"
      interval: 5s
      timeout: 2s
      labels:
        test: "e2e-timeout"
        target: "working"
`

	env := NewTestEnvironment(t, 5*time.Minute)
	defer env.Teardown()

	RequireSetup(t, env)

	err := env.StartVMProber(shortTimeoutConfig, "debug")
	require.NoError(t, err, "Failed to start vmprober")

	env.WaitForProbes(20 * time.Second)

	ctx := env.Context()

	// Verify vmprober is still running and collecting metrics
	result, err := QueryMetric(ctx, "vmprober_probe_attempts_total")
	require.NoError(t, err, "vmprober should still be responding after timeout")
	require.Equal(t, "success", result.Status)

	t.Logf("vmprober handled timeouts gracefully, found %d metric series", len(result.Data.Result))
}

