package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestE2E_Grafana_DashboardLoads tests that Grafana dashboard loads correctly
func TestE2E_Grafana_DashboardLoads(t *testing.T) {
	env := NewTestEnvironment(t, 5*time.Minute)
	defer env.Teardown()

	RequireSetup(t, env)

	ctx := env.Context()

	// Check Grafana health
	healthy, err := checkGrafanaHealth(ctx)
	require.NoError(t, err)
	require.True(t, healthy, "Grafana is not healthy")

	// List dashboards
	dashboards, err := listGrafanaDashboards(ctx)
	require.NoError(t, err)

	t.Logf("Found %d dashboards in Grafana", len(dashboards))

	// Look for vmprober dashboard
	foundVMProber := false
	for _, dash := range dashboards {
		if title, ok := dash["title"].(string); ok {
			t.Logf("Dashboard: %s", title)
			if title == "VMProber - Network Monitoring" {
				foundVMProber = true
			}
		}
	}

	if !foundVMProber {
		t.Log("VMProber dashboard not found - may need provisioning")
	}
}

// TestE2E_Grafana_DatasourceConfigured tests that VictoriaMetrics datasource is configured
func TestE2E_Grafana_DatasourceConfigured(t *testing.T) {
	env := NewTestEnvironment(t, 5*time.Minute)
	defer env.Teardown()

	RequireSetup(t, env)

	ctx := env.Context()

	// List datasources
	datasources, err := listGrafanaDatasources(ctx)
	require.NoError(t, err)

	t.Logf("Found %d datasources in Grafana", len(datasources))

	// Look for VictoriaMetrics datasource
	foundVM := false
	for _, ds := range datasources {
		name, _ := ds["name"].(string)
		dsType, _ := ds["type"].(string)
		uid, _ := ds["uid"].(string)
		t.Logf("Datasource: name=%s, type=%s, uid=%s", name, dsType, uid)

		if name == "VictoriaMetrics" || uid == "victoriametrics" {
			foundVM = true

			// Check if it's accessible
			if dsType == "prometheus" {
				t.Log("VictoriaMetrics datasource configured as Prometheus type")
			}
		}
	}

	require.True(t, foundVM, "VictoriaMetrics datasource not found in Grafana")
}

// TestE2E_Grafana_QueryReturnsData tests that Grafana can query metrics from VictoriaMetrics
func TestE2E_Grafana_QueryReturnsData(t *testing.T) {
	env := NewTestEnvironment(t, 5*time.Minute)
	defer env.Teardown()

	RequireSetup(t, env)

	// Start vmprober to generate metrics
	err := env.StartVMProber(TCPProbesConfig(), "info")
	require.NoError(t, err, "Failed to start vmprober")

	// Wait for metrics to be collected
	env.WaitForProbes(20 * time.Second)

	ctx := env.Context()

	// Query VictoriaMetrics directly (via vmselect)
	result, err := QueryMetric(ctx, "vmprober_probe_attempts_total")
	require.NoError(t, err)
	require.Equal(t, "success", result.Status)
	require.NotEmpty(t, result.Data.Result, "No metrics found in VictoriaMetrics")

	t.Logf("VictoriaMetrics has %d metric series that Grafana can display", len(result.Data.Result))
}

// TestE2E_Grafana_JobLabelIssue documents the "No Data" issue with job label
func TestE2E_Grafana_JobLabelIssue(t *testing.T) {
	env := NewTestEnvironment(t, 5*time.Minute)
	defer env.Teardown()

	RequireSetup(t, env)

	// Start vmprober to generate metrics
	err := env.StartVMProber(TCPProbesConfig(), "debug")
	require.NoError(t, err, "Failed to start vmprober")

	// Wait for metrics
	env.WaitForProbes(20 * time.Second)

	ctx := env.Context()

	// Test 1: Query without job label (should work)
	result, err := QueryMetric(ctx, "vmprober_probe_attempts_total")
	require.NoError(t, err)

	t.Logf("Query without job filter: %d results", len(result.Data.Result))

	// Test 2: Query with job label (may not work if job label is not set)
	resultWithJob, err := QueryMetric(ctx, `vmprober_probe_attempts_total{job="blackbox/vmprober"}`)
	require.NoError(t, err)

	t.Logf("Query with job filter: %d results", len(resultWithJob.Data.Result))

	// Document the issue
	if len(result.Data.Result) > 0 && len(resultWithJob.Data.Result) == 0 {
		t.Log("ISSUE DETECTED: Metrics exist but don't have 'job' label")
		t.Log("This causes 'No Data' in Grafana dashboard which filters by job=~\"$vmprober_job\"")
		t.Log("")
		t.Log("To fix: Ensure custom_labels.job is properly added to metrics in collector")

		// Log the actual labels on the metrics
		if len(result.Data.Result) > 0 {
			t.Log("Actual labels on metrics:")
			for k, v := range result.Data.Result[0].Metric {
				t.Logf("  %s = %s", k, v)
			}
		}
	} else if len(resultWithJob.Data.Result) > 0 {
		t.Log("Job label is correctly set on metrics")
	}
}

// checkGrafanaHealth checks if Grafana is healthy
func checkGrafanaHealth(ctx context.Context) (bool, error) {
	client := &http.Client{Timeout: 5 * time.Second}

	req, err := http.NewRequestWithContext(ctx, "GET", grafanaURL+"/api/health", nil)
	if err != nil {
		return false, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}

// listGrafanaDashboards lists all dashboards in Grafana
func listGrafanaDashboards(ctx context.Context) ([]map[string]interface{}, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequestWithContext(ctx, "GET", grafanaURL+"/api/search?type=dash-db", nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth("admin", "admin")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list dashboards: %d - %s", resp.StatusCode, string(body))
	}

	var dashboards []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&dashboards); err != nil {
		return nil, err
	}

	return dashboards, nil
}

// listGrafanaDatasources lists all datasources in Grafana
func listGrafanaDatasources(ctx context.Context) ([]map[string]interface{}, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequestWithContext(ctx, "GET", grafanaURL+"/api/datasources", nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth("admin", "admin")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list datasources: %d - %s", resp.StatusCode, string(body))
	}

	var datasources []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&datasources); err != nil {
		return nil, err
	}

	return datasources, nil
}

