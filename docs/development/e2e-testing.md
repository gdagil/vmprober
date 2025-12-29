# E2E Testing Guide

This document describes the end-to-end testing strategy and implementation for VMProber.

---

## Table of Contents

- [Overview](#overview)
- [Test Structure](#test-structure)
- [Running Tests](#running-tests)
- [Test Configuration](#test-configuration)
- [Docker Compose Services](#docker-compose-services)
- [Verified Metrics](#verified-metrics)
- [Issues Found & Fixed](#issues-found--fixed)
- [Known Limitations](#known-limitations)

---

## Overview

E2E tests validate the complete flow of metrics from VMProber to VictoriaMetrics, ensuring that:

1. VMProber correctly executes probes
2. Metrics are formatted in Prometheus text format
3. Metrics are successfully pushed to VictoriaMetrics
4. Metrics can be queried from VictoriaMetrics
5. Grafana dashboard displays data correctly

## Test Structure

### File Location

```
vmprober/
└── tests/
    └── e2e/
        ├── configs_test.go           # Test configurations for different scenarios
        ├── dns_probes_test.go        # DNS probe tests
        ├── failure_scenarios_test.go # Failure and high-load tests
        ├── grafana_test.go           # Grafana integration tests
        ├── helpers_test.go           # Common helpers and utilities
        ├── http_probes_test.go       # HTTP/HTTPS probe tests
        ├── metrics_flow_test.go      # Metrics flow tests
        └── tcp_probes_test.go        # TCP probe tests
```

### Test Categories

#### TCP Probe Tests (`tcp_probes_test.go`)

| Test | Description |
|------|-------------|
| `TestE2E_TCPProbes_Basic` | Basic TCP probe functionality |
| `TestE2E_TCPProbes_Labels` | Verifies correct metric labels |
| `TestE2E_TCPProbes_RTTHistogram` | RTT histogram recording |
| `TestE2E_TCPProbes_MultipleTargets` | Multiple TCP targets |

#### HTTP Probe Tests (`http_probes_test.go`)

| Test | Description |
|------|-------------|
| `TestE2E_HTTPProbes_Basic` | Basic HTTP probe functionality |
| `TestE2E_HTTPProbes_StatusCodes` | HTTP status code validation |
| `TestE2E_HTTPSProbes_TLS` | HTTPS with TLS validation |
| `TestE2E_HTTPProbes_Latency` | HTTP latency recording |

#### DNS Probe Tests (`dns_probes_test.go`)

| Test | Description |
|------|-------------|
| `TestE2E_DNSProbes_Basic` | Basic DNS probe functionality |
| `TestE2E_DNSProbes_QueryTypes` | Different DNS query types (A, AAAA) |
| `TestE2E_DNSProbes_MultipleServers` | Multiple DNS servers |
| `TestE2E_DNSProbes_Latency` | DNS latency recording |

#### Failure Scenario Tests (`failure_scenarios_test.go`)

| Test | Description |
|------|-------------|
| `TestE2E_FailingProbes_RecordFailures` | Failure metrics recording |
| `TestE2E_FailingProbes_SuccessRate` | Success rate with mixed results |
| `TestE2E_HighLoad_StabilityTest` | High-load stability test |
| `TestE2E_MixedProbes_ConcurrentExecution` | Concurrent probe execution |
| `TestE2E_Timeout_HandledGracefully` | Timeout handling |

#### Metrics Flow Tests (`metrics_flow_test.go`)

| Test | Description |
|------|-------------|
| `TestE2E_MetricsFlow` | Complete metrics flow validation |
| `TestE2E_AllMetricsCollected` | Collector metrics pushed to VM |
| `TestE2E_MetricsWithJobLabel` | Job label verification |
| `TestE2E_MetricsNamespace` | Namespace verification |

#### Grafana Tests (`grafana_test.go`)

| Test | Description |
|------|-------------|
| `TestE2E_Grafana_DashboardLoads` | Dashboard loading |
| `TestE2E_Grafana_DatasourceConfigured` | VictoriaMetrics datasource |
| `TestE2E_Grafana_QueryReturnsData` | Query returns data |
| `TestE2E_Grafana_JobLabelIssue` | Documents "No Data" issue |

### Metrics Flow Diagram

```
┌─────────┐     ┌──────────┐     ┌───────────────┐
│ VMProber│────▶│ vminsert │────▶│  vmstorage    │
└─────────┘     └──────────┘     └───────────────┘
                                        │
┌─────────┐     ┌──────────┐           │
│  Test   │◀────│ vmselect │◀──────────┘
└─────────┘     └──────────┘
```

## Running Tests

### Prerequisites

- Go 1.25+
- Docker and Docker Compose (or `docker compose`)
- Built VMProber binary (`bin/vmprober`)
- Available ports: 8480, 8481, 8482, 3000

### Commands

```bash
# Build VMProber binary first
make build

# Run all E2E tests
go test -v -timeout 10m ./tests/e2e/...

# Run specific test file
go test -v -timeout 10m ./tests/e2e/tcp_probes_test.go ./tests/e2e/helpers_test.go ./tests/e2e/configs_test.go

# Run specific test
go test -v -timeout 10m -run TestE2E_MetricsFlow ./tests/e2e/...

# Run with verbose output
go test -v -timeout 10m -count=1 ./tests/e2e/...

# Skip long-running tests
go test -v -timeout 5m -short ./tests/e2e/...

# Using make (if configured)
make e2e-test
```

### Troubleshooting Test Failures

| Issue | Solution |
|-------|----------|
| Services not starting | Check if ports 8480-8482, 3000 are available |
| Binary not found | Run `make build` first |
| Timeout errors | Increase test timeout with `-timeout 15m` |
| Metrics not found | Check vminsert logs for formatting errors |
| "No Data" in Grafana | Ensure `custom_labels.job` is set in config |

## Test Configuration

Tests use configuration templates defined in `configs_test.go`:

| Config Function | Description |
|-----------------|-------------|
| `BaseConfig()` | Base configuration with common settings |
| `TCPProbesConfig()` | TCP probe targets |
| `HTTPProbesConfig()` | HTTP/HTTPS probe targets |
| `DNSProbesConfig()` | DNS probe targets |
| `ICMPProbesConfig()` | ICMP probe targets |
| `MixedProbesConfig()` | Mixed probe types |
| `HighLoadConfig()` | High-load testing |
| `FailingProbesConfig()` | Targets that should fail |

### Example Configuration

```yaml
# HTTP Server
listen:
  port: 8429
  host: "0.0.0.0"

# Push mode for VictoriaMetrics
push:
  enabled: true
  endpoints:
    - url: "http://localhost:8480/insert/0/prometheus/api/v1/import"

# Test targets
targets:
  static:
    - host: "127.0.0.1"
      port: 22
      proto: "tcp"
      interval: 5s
      timeout: 2s
      labels:
        test: "e2e"

# Metrics configuration with job label
metrics:
  namespace: "vmprober"
  custom_labels:
    job: "blackbox/vmprober"
```

### Configuration Highlights

| Setting | Value | Reason |
|---------|-------|--------|
| Push interval | 5s | Fast feedback during tests |
| WAL | disabled | Simplifies test setup |
| Targets | localhost, 8.8.8.8 | Reliable, always-available endpoints |
| job label | blackbox/vmprober | Required for Grafana dashboard |

## Docker Compose Services

The E2E tests use the existing `docker-compose.yml` which provides:

| Service | Port | Description |
|---------|------|-------------|
| **vmstorage** | 8482 | VictoriaMetrics storage node |
| **vminsert** | 8480 | Metrics ingestion endpoint |
| **vmselect** | 8481 | Query endpoint for metrics |
| **grafana** | 3000 | Visualization dashboard |

### Service Health Checks

Tests verify service readiness before running:

```go
// Check vmselect is ready
resp, err := http.Get("http://localhost:8481/health")
if err != nil || resp.StatusCode != 200 {
    // Wait and retry...
}
```

## Verified Metrics

The E2E tests verify the following metrics exist in VictoriaMetrics:

### Probe Metrics

| Metric | Type | Labels |
|--------|------|--------|
| `vmprober_probe_attempts_total` | Counter | instance, target_ip, port, protocol, job |
| `vmprober_probe_success_total` | Counter | instance, target_ip, port, protocol, job |
| `vmprober_probe_failure_total` | Counter | instance, target_ip, port, protocol, job |
| `vmprober_probe_rtt_seconds` | Histogram | instance, target_ip, port, protocol, job |

### Job Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `vmprober_jobs_total` | Gauge | Total scheduled jobs |
| `vmprober_jobs_running` | Gauge | Currently executing jobs |
| `vmprober_jobs_failed` | Gauge | Failed job count |

### Verification Queries

```promql
# Check if probe metrics exist
vmprober_probe_attempts_total

# Get probe success rate
vmprober_probe_success_total / vmprober_probe_attempts_total

# Check RTT histogram percentile
histogram_quantile(0.99, vmprober_probe_rtt_seconds_bucket)

# Filter by job label (used by Grafana dashboard)
vmprober_probe_attempts_total{job="blackbox/vmprober"}
```

## Issues Found & Fixed

During E2E testing implementation, several issues were identified and resolved:

### 1. Missing Job Label in Metrics

| Issue | Fix |
|-------|-----|
| **Problem** | Metrics didn't include `job` label, causing "No Data" in Grafana dashboard |
| **Solution** | Modified `Collector` to accept and include `customLabels` from config |

### 2. Incorrect Data Format for vminsert

| Issue | Fix |
|-------|-----|
| **Problem** | Used JSON line format with Prometheus endpoint |
| **Solution** | Changed to Prometheus text format |

### 3. Wrong Content-Type Header

| Issue | Fix |
|-------|-----|
| **Problem** | Used `Content-Type: application/json` for Prometheus endpoint |
| **Solution** | Changed to `Content-Type: text/plain` |

### 4. Missing Namespace Filtering

| Issue | Fix |
|-------|-----|
| **Problem** | `ExportMetrics()` exported all metrics including Go runtime metrics |
| **Solution** | Added namespace filtering in `parsePrometheusMetrics()` |

## Known Limitations

### 1. Histogram Metric Parsing

Histogram metrics are parsed as separate metrics (`_bucket`, `_sum`, `_count`), which may result in incomplete information during export.

### 2. Network Dependencies

Tests require network connectivity to:
- `127.0.0.1:22` (SSH) - may fail if SSH is not running
- `8.8.8.8:53` (Google DNS) - requires internet access

### 3. Resource Requirements

Docker Compose services require:
- ~500MB RAM total
- ~1GB disk space for vmstorage data

---

## See Also

- [Setup Guide](setup.md) - Development environment setup
- [Configuration](../getting-started/configuration.md) - Configuration reference
- [Troubleshooting](../operations/troubleshooting.md) - Troubleshooting guide
- [Metrics Reference](../reference/metrics.md) - Metrics documentation

