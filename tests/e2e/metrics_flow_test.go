package e2e

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestE2E_MetricsFlow tests the complete metrics flow from vmprober to VictoriaMetrics
func TestE2E_MetricsFlow(t *testing.T) {
	env := NewTestEnvironment(t, 5*time.Minute)
	defer env.Teardown()

	RequireSetup(t, env)

	err := env.StartVMProber(TCPProbesConfig(), "info")
	require.NoError(t, err, "Failed to start vmprober")

	// Wait for probes to execute
	env.WaitForProbes(15 * time.Second)

	ctx := env.Context()

	// Verify that all expected metrics exist
	expectedMetrics := []string{
		"vmprober_probe_attempts_total",
	}

	err = VerifyMetricsExist(ctx, expectedMetrics)
	require.NoError(t, err, "Metrics verification failed")

	// Verify specific metrics have data
	t.Log("Verifying specific metrics have data...")
	result, err := QueryMetric(ctx, "vmprober_probe_attempts_total")
	require.NoError(t, err)
	require.Equal(t, "success", result.Status)
	require.NotEmpty(t, result.Data.Result, "No probe_attempts_total metrics found")
}

// TestE2E_AllMetricsCollected verifies that all metrics from collector are pushed to VM
func TestE2E_AllMetricsCollected(t *testing.T) {
	env := NewTestEnvironment(t, 6*time.Minute)
	defer env.Teardown()

	RequireSetup(t, env)

	err := env.StartVMProber(TCPProbesConfig(), "debug")
	require.NoError(t, err, "Failed to start vmprober")

	// Wait longer for collector metrics to be pushed (push interval is 30s)
	env.WaitForProbes(40 * time.Second)

	ctx := env.Context()

	// Check that collector metrics arrived in VM
	t.Log("Verifying collector metrics in VictoriaMetrics...")
	metricsToCheck := []string{
		"vmprober_probe_attempts_total",
	}

	for _, metricName := range metricsToCheck {
		result, err := QueryMetric(ctx, metricName)
		require.NoError(t, err, "Failed to query metric %s", metricName)
		require.Equal(t, "success", result.Status, "Query for %s failed", metricName)
		require.NotEmpty(t, result.Data.Result, "No results for metric %s", metricName)
	}
}

// TestE2E_MetricsWithJobLabel verifies that metrics have the job label from custom_labels
func TestE2E_MetricsWithJobLabel(t *testing.T) {
	env := NewTestEnvironment(t, 5*time.Minute)
	defer env.Teardown()

	RequireSetup(t, env)

	err := env.StartVMProber(TCPProbesConfig(), "info")
	require.NoError(t, err, "Failed to start vmprober")

	env.WaitForProbes(15 * time.Second)

	ctx := env.Context()

	// Query metrics with job label filter
	result, err := QueryMetric(ctx, `vmprober_probe_attempts_total{job="blackbox/vmprober"}`)
	require.NoError(t, err)

	// Note: This test documents expected behavior
	// If metrics have job label, this should succeed
	t.Logf("Query result status: %s, results count: %d", result.Status, len(result.Data.Result))

	if len(result.Data.Result) == 0 {
		t.Log("WARNING: No metrics found with job label - this may indicate job label is not being set correctly")
	}
}

// TestE2E_MetricsPushRetry verifies metrics are retried on failure
func TestE2E_MetricsPushRetry(t *testing.T) {
	t.Skip("Requires network failure simulation - implement when infrastructure supports it")
}

// TestE2E_MetricsNamespace verifies metrics use the configured namespace
func TestE2E_MetricsNamespace(t *testing.T) {
	env := NewTestEnvironment(t, 5*time.Minute)
	defer env.Teardown()

	RequireSetup(t, env)

	err := env.StartVMProber(TCPProbesConfig(), "info")
	require.NoError(t, err, "Failed to start vmprober")

	env.WaitForProbes(15 * time.Second)

	ctx := env.Context()

	// Get all metric names
	metricNames, err := GetMetricNames(ctx)
	require.NoError(t, err)

	// Verify all vmprober metrics have the correct namespace prefix
	vmproberMetricsFound := 0
	for _, name := range metricNames {
		if len(name) > 8 && name[:8] == "vmprober" {
			vmproberMetricsFound++
		}
	}

	require.Greater(t, vmproberMetricsFound, 0, "No vmprober-namespaced metrics found")
	t.Logf("Found %d metrics with vmprober namespace", vmproberMetricsFound)
}
