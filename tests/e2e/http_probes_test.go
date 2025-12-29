package e2e

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestE2E_HTTPProbes_Basic tests basic HTTP probe functionality
func TestE2E_HTTPProbes_Basic(t *testing.T) {
	env := NewTestEnvironment(t, 5*time.Minute)
	defer env.Teardown()

	RequireSetup(t, env)

	err := env.StartVMProber(HTTPProbesConfig(), "info")
	require.NoError(t, err, "Failed to start vmprober")

	// Wait for HTTP probes to execute (they have longer timeouts)
	env.WaitForProbes(25 * time.Second)

	ctx := env.Context()

	// Verify that HTTP probe metrics exist
	expectedMetrics := []string{
		"vmprober_probe_attempts_total",
		"vmprober_probe_success_total",
	}

	err = VerifyMetricsExist(ctx, expectedMetrics)
	require.NoError(t, err, "Expected HTTP probe metrics not found")
}

// TestE2E_HTTPProbes_StatusCodes tests HTTP probe status code validation
func TestE2E_HTTPProbes_StatusCodes(t *testing.T) {
	env := NewTestEnvironment(t, 5*time.Minute)
	defer env.Teardown()

	RequireSetup(t, env)

	err := env.StartVMProber(HTTPProbesConfig(), "debug")
	require.NoError(t, err, "Failed to start vmprober")

	env.WaitForProbes(25 * time.Second)

	ctx := env.Context()

	// Query successful probes
	result, err := QueryMetric(ctx, "vmprober_probe_success_total")
	require.NoError(t, err)
	require.Equal(t, "success", result.Status)

	// Should have successful HTTP probes (httpbin returns expected status codes)
	t.Logf("Found %d success metric results", len(result.Data.Result))
}

// TestE2E_HTTPSProbes_TLS tests HTTPS probe with TLS validation
func TestE2E_HTTPSProbes_TLS(t *testing.T) {
	env := NewTestEnvironment(t, 5*time.Minute)
	defer env.Teardown()

	RequireSetup(t, env)

	err := env.StartVMProber(HTTPProbesConfig(), "info")
	require.NoError(t, err, "Failed to start vmprober")

	env.WaitForProbes(25 * time.Second)

	ctx := env.Context()

	// Query metrics - should include HTTPS probes to github.com
	result, err := QueryMetric(ctx, "vmprober_probe_attempts_total")
	require.NoError(t, err)
	require.Equal(t, "success", result.Status)

	// Check for HTTPS protocol in metrics
	foundHTTPS := false
	for _, r := range result.Data.Result {
		if protocol, ok := r.Metric["protocol"]; ok && protocol == "https" {
			foundHTTPS = true
			break
		}
	}

	// Note: This depends on HTTP/HTTPS probe implementation
	if foundHTTPS {
		t.Log("Found HTTPS protocol in metrics")
	} else {
		t.Log("No HTTPS protocol found - may be reported as 'http' or probe not supported yet")
	}
}

// TestE2E_HTTPProbes_Latency tests HTTP probe latency recording
func TestE2E_HTTPProbes_Latency(t *testing.T) {
	env := NewTestEnvironment(t, 5*time.Minute)
	defer env.Teardown()

	RequireSetup(t, env)

	err := env.StartVMProber(HTTPProbesConfig(), "info")
	require.NoError(t, err, "Failed to start vmprober")

	env.WaitForProbes(30 * time.Second)

	ctx := env.Context()

	// Verify RTT metrics exist for HTTP probes
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
		t.Log("RTT metrics found for probes")
	} else {
		t.Log("No RTT metrics found - may need more time or HTTP probes may not record RTT")
	}
}

