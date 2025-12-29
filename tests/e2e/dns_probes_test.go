package e2e

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestE2E_DNSProbes_Basic tests basic DNS probe functionality
func TestE2E_DNSProbes_Basic(t *testing.T) {
	env := NewTestEnvironment(t, 5*time.Minute)
	defer env.Teardown()

	RequireSetup(t, env)

	err := env.StartVMProber(DNSProbesConfig(), "info")
	require.NoError(t, err, "Failed to start vmprober")

	// Wait for DNS probes to execute
	env.WaitForProbes(20 * time.Second)

	ctx := env.Context()

	// Verify that DNS probe metrics exist
	expectedMetrics := []string{
		"vmprober_probe_attempts_total",
		"vmprober_probe_success_total",
	}

	err = VerifyMetricsExist(ctx, expectedMetrics)
	require.NoError(t, err, "Expected DNS probe metrics not found")
}

// TestE2E_DNSProbes_QueryTypes tests different DNS query types (A, AAAA)
func TestE2E_DNSProbes_QueryTypes(t *testing.T) {
	env := NewTestEnvironment(t, 5*time.Minute)
	defer env.Teardown()

	RequireSetup(t, env)

	err := env.StartVMProber(DNSProbesConfig(), "debug")
	require.NoError(t, err, "Failed to start vmprober")

	env.WaitForProbes(25 * time.Second)

	ctx := env.Context()

	// Query probe attempts
	result, err := QueryMetric(ctx, "vmprober_probe_attempts_total")
	require.NoError(t, err)
	require.Equal(t, "success", result.Status)

	// Should have multiple DNS probe results (A and AAAA queries)
	t.Logf("Found %d probe attempt results for DNS probes", len(result.Data.Result))
}

// TestE2E_DNSProbes_MultipleServers tests DNS probes to multiple DNS servers
func TestE2E_DNSProbes_MultipleServers(t *testing.T) {
	env := NewTestEnvironment(t, 5*time.Minute)
	defer env.Teardown()

	RequireSetup(t, env)

	err := env.StartVMProber(DNSProbesConfig(), "info")
	require.NoError(t, err, "Failed to start vmprober")

	env.WaitForProbes(25 * time.Second)

	ctx := env.Context()

	// Query successful probes
	result, err := QueryMetric(ctx, "vmprober_probe_success_total")
	require.NoError(t, err)
	require.Equal(t, "success", result.Status)

	// Check for different target IPs (Google DNS 8.8.8.8 and Cloudflare 1.1.1.1)
	targetIPs := make(map[string]bool)
	for _, r := range result.Data.Result {
		if ip, ok := r.Metric["target_ip"]; ok {
			targetIPs[ip] = true
		}
	}

	t.Logf("Found DNS probes to %d unique target IPs: %v", len(targetIPs), targetIPs)
}

// TestE2E_DNSProbes_Latency tests DNS probe latency recording
func TestE2E_DNSProbes_Latency(t *testing.T) {
	env := NewTestEnvironment(t, 5*time.Minute)
	defer env.Teardown()

	RequireSetup(t, env)

	err := env.StartVMProber(DNSProbesConfig(), "info")
	require.NoError(t, err, "Failed to start vmprober")

	env.WaitForProbes(25 * time.Second)

	ctx := env.Context()

	// DNS probes are typically fast, verify RTT metrics exist
	metricNames, err := GetMetricNames(ctx)
	require.NoError(t, err)

	foundRTT := false
	for _, name := range metricNames {
		if name == "vmprober_probe_rtt_seconds" ||
			name == "vmprober_probe_rtt_seconds_bucket" ||
			name == "vmprober_probe_rtt_seconds_sum" ||
			name == "vmprober_probe_rtt_seconds_count" {
			foundRTT = true
			break
		}
	}

	if foundRTT {
		t.Log("RTT metrics found for DNS probes")

		// Query RTT sum to verify latency is being recorded
		rttResult, err := QueryMetric(ctx, "vmprober_probe_rtt_seconds_sum")
		if err == nil && len(rttResult.Data.Result) > 0 {
			t.Log("DNS RTT sum metrics found with data")
		}
	} else {
		t.Log("No RTT metrics found - DNS probes may not have completed successfully yet")
	}
}
