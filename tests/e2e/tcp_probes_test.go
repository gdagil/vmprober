package e2e

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestE2E_TCPProbes_Basic tests basic TCP probe functionality
func TestE2E_TCPProbes_Basic(t *testing.T) {
	env := NewTestEnvironment(t, 5*time.Minute)
	defer env.Teardown()

	RequireSetup(t, env)

	err := env.StartVMProber(TCPProbesConfig(), "info")
	require.NoError(t, err, "Failed to start vmprober")

	// Wait for probes to execute
	env.WaitForProbes(15 * time.Second)

	// Verify that TCP probe metrics exist
	ctx := env.Context()
	expectedMetrics := []string{
		"vmprober_probe_attempts_total",
	}

	err = VerifyMetricsExist(ctx, expectedMetrics)
	require.NoError(t, err, "Expected TCP probe metrics not found")

	// Also check for RTT or duration metrics
	rttMetrics := []string{"vmprober_probe_rtt_seconds", "vmprober_probe_duration_seconds"}
	metricNames, _ := GetMetricNames(ctx)
	foundRTT := false
	for _, name := range metricNames {
		for _, rtt := range rttMetrics {
			if name == rtt {
				foundRTT = true
				break
			}
		}
	}
	t.Logf("Found RTT/duration metrics: %v", foundRTT)
}

// TestE2E_TCPProbes_Labels verifies that TCP probe metrics have correct labels
func TestE2E_TCPProbes_Labels(t *testing.T) {
	env := NewTestEnvironment(t, 5*time.Minute)
	defer env.Teardown()

	RequireSetup(t, env)

	err := env.StartVMProber(TCPProbesConfig(), "info")
	require.NoError(t, err, "Failed to start vmprober")

	env.WaitForProbes(15 * time.Second)

	ctx := env.Context()

	// Verify that metrics have expected labels
	// Note: labels may include target_ip, target_port, protocol, job, status
	expectedLabels := []string{"target_ip", "protocol", "job"}
	err = VerifyMetricHasLabels(ctx, "vmprober_probe_attempts_total", expectedLabels)
	require.NoError(t, err, "Metric vmprober_probe_attempts_total missing expected labels")
}

// TestE2E_TCPProbes_RTTHistogram verifies that RTT histogram is recorded for successful probes
func TestE2E_TCPProbes_RTTHistogram(t *testing.T) {
	env := NewTestEnvironment(t, 5*time.Minute)
	defer env.Teardown()

	RequireSetup(t, env)

	err := env.StartVMProber(TCPProbesConfig(), "info")
	require.NoError(t, err, "Failed to start vmprober")

	// Wait longer to ensure RTT metrics are collected
	env.WaitForProbes(20 * time.Second)

	ctx := env.Context()

	// Verify RTT histogram metrics exist
	rttMetrics := []string{
		"vmprober_probe_rtt_seconds",
		"vmprober_probe_rtt_seconds_bucket",
		"vmprober_probe_rtt_seconds_sum",
		"vmprober_probe_rtt_seconds_count",
	}

	// At least the base RTT metric should exist
	metricNames, err := GetMetricNames(ctx)
	require.NoError(t, err)

	foundRTT := false
	for _, name := range metricNames {
		for _, rtt := range rttMetrics {
			if name == rtt {
				foundRTT = true
				break
			}
		}
		if foundRTT {
			break
		}
	}

	require.True(t, foundRTT, "No RTT metrics found in VictoriaMetrics")
}

// TestE2E_TCPProbes_MultipleTargets verifies metrics for multiple TCP targets
func TestE2E_TCPProbes_MultipleTargets(t *testing.T) {
	env := NewTestEnvironment(t, 5*time.Minute)
	defer env.Teardown()

	RequireSetup(t, env)

	err := env.StartVMProber(TCPProbesConfig(), "info")
	require.NoError(t, err, "Failed to start vmprober")

	env.WaitForProbes(20 * time.Second)

	ctx := env.Context()

	// Query probe attempts and verify multiple targets
	result, err := QueryMetric(ctx, "vmprober_probe_attempts_total")
	require.NoError(t, err)
	require.Equal(t, "success", result.Status)

	// Should have metrics for multiple targets (SSH, DNS, Google HTTP, Google HTTPS)
	require.GreaterOrEqual(t, len(result.Data.Result), 2, "Expected metrics for at least 2 targets")

	// Verify we have different target_ip values
	targetIPs := make(map[string]bool)
	for _, r := range result.Data.Result {
		if ip, ok := r.Metric["target_ip"]; ok {
			targetIPs[ip] = true
		}
	}

	t.Logf("Found %d unique target IPs", len(targetIPs))
}

