# Metrics Reference

Complete reference of all metrics exported by VMProber.

## Metric Naming

All metrics use the `vmprober` prefix (configurable via `metrics.namespace`).

## Probe Metrics

### vmprober_probe_success_total

**Type:** `counter`

**Description:** Total number of successful probes.

**Labels:**
- `target` (string) - Target host and port
- `protocol` (string) - Probe protocol: "tcp", "udp", "icmp"

**Example:**
```
vmprober_probe_success_total{protocol="tcp",target="example.com:80"} 1234
```

### vmprober_probe_failure_total

**Type:** `counter`

**Description:** Total number of failed probes.

**Labels:**
- `target` (string) - Target host and port
- `protocol` (string) - Probe protocol

**Example:**
```
vmprober_probe_failure_total{protocol="tcp",target="example.com:80"} 5
```

### vmprober_probe_rtt_seconds

**Type:** `histogram`

**Description:** Probe round-trip time in seconds. Measured only for successful probes.

**Labels:**
- `target` (string) - Target host and port
- `protocol` (string) - Probe protocol

**Buckets:** `[0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0]`

**Example:**
```
vmprober_probe_rtt_seconds_bucket{protocol="tcp",target="example.com:80",le="0.001"} 100
vmprober_probe_rtt_seconds_bucket{protocol="tcp",target="example.com:80",le="0.005"} 500
vmprober_probe_rtt_seconds_sum{protocol="tcp",target="example.com:80"} 12.34
vmprober_probe_rtt_seconds_count{protocol="tcp",target="example.com:80"} 1234
```

### vmprober_probe_attempts_total

**Type:** `counter`

**Description:** Total number of probe attempts (successful and failed).

**Labels:**
- `target` (string) - Target host and port
- `protocol` (string) - Probe protocol

### vmprober_probe_last_success_timestamp

**Type:** `gauge`

**Description:** Unix timestamp (in seconds) of last successful probe.

**Labels:**
- `target` (string) - Target host and port
- `protocol` (string) - Probe protocol

## System Metrics

If `enable_process_metrics` and `enable_go_metrics` are enabled:

- `process_cpu_seconds_total` - Total CPU time
- `process_resident_memory_bytes` - Memory usage
- `go_goroutines` - Number of goroutines
- `go_memstats_alloc_bytes` - Allocated memory

## Common Queries

### Success Rate

```promql
rate(vmprober_probe_success_total[5m]) / rate(vmprober_probe_attempts_total[5m])
```

### Average RTT

```promql
rate(vmprober_probe_rtt_seconds_sum[5m]) / rate(vmprober_probe_rtt_seconds_count[5m])
```

### 95th Percentile RTT

```promql
histogram_quantile(0.95, rate(vmprober_probe_rtt_seconds_bucket[5m]))
```

## See Also

- [API Reference](api.md) - HTTP API endpoints
- [Operations: Monitoring](../operations/monitoring.md) - Setting up monitoring


