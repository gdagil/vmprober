# Metrics Reference

Complete reference of all metrics exported by VMProber.

## Metric Naming

All metrics use the `vmprober` prefix (configurable via `metrics.namespace`).

## Probe Metrics

### vmprober_probe_success_total

**Type:** `counter`

**Description:** Total number of successful probes.

**Labels:**
- `instance` (string) - Target hostname and port from configuration (e.g., "google.com:443")
- `target_ip` (string) - IP address that was actually connected to (e.g., "142.250.109.100")
- `port` (string) - Target port (e.g., "443")
- `protocol` (string) - Probe protocol: "tcp", "udp", "icmp", "http", "https", "dns", "grpc"

**Example:**
```
vmprober_probe_success_total{instance="google.com:443",target_ip="142.250.109.100",port="443",protocol="tcp"} 1234
```

**Note:** When a hostname resolves to multiple IP addresses, you'll get separate metrics for each IP with the same `instance` label, allowing aggregation by instance or per-IP analysis.

### vmprober_probe_failure_total

**Type:** `counter`

**Description:** Total number of failed probes.

**Labels:**
- `instance` (string) - Target hostname and port from configuration
- `target_ip` (string) - IP address that was actually connected to
- `port` (string) - Target port
- `protocol` (string) - Probe protocol

**Example:**
```
vmprober_probe_failure_total{instance="google.com:443",target_ip="142.250.109.100",port="443",protocol="tcp"} 5
```

### vmprober_probe_rtt_seconds

**Type:** `histogram`

**Description:** Probe round-trip time in seconds. Measured only for successful probes.

**Labels:**
- `instance` (string) - Target hostname and port from configuration
- `target_ip` (string) - IP address that was actually connected to
- `port` (string) - Target port
- `protocol` (string) - Probe protocol

**Buckets:** `[0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0]`

**Example:**
```
vmprober_probe_rtt_seconds_bucket{instance="google.com:443",target_ip="142.250.109.100",port="443",protocol="tcp",le="0.001"} 100
vmprober_probe_rtt_seconds_bucket{instance="google.com:443",target_ip="142.250.109.100",port="443",protocol="tcp",le="0.005"} 500
vmprober_probe_rtt_seconds_sum{instance="google.com:443",target_ip="142.250.109.100",port="443",protocol="tcp"} 12.34
vmprober_probe_rtt_seconds_count{instance="google.com:443",target_ip="142.250.109.100",port="443",protocol="tcp"} 1234
```

### vmprober_probe_attempts_total

**Type:** `counter`

**Description:** Total number of probe attempts (successful and failed).

**Labels:**
- `instance` (string) - Target hostname and port from configuration
- `target_ip` (string) - IP address that was actually connected to
- `port` (string) - Target port
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

## Protocol-Specific Metrics

### HTTP/HTTPS Probes

For HTTP/HTTPS probes, the following additional information is available in labels:

```
vmprober_probe_success_total{instance="api.example.com:443",protocol="https",path="/health"} 1234
```

### DNS Probes

For DNS probes, the query type is included:

```
vmprober_probe_success_total{instance="8.8.8.8:53",protocol="dns",query_type="A",query_name="google.com"} 1234
```

### gRPC Probes

For gRPC probes, the service name is included:

```
vmprober_probe_success_total{instance="grpc.example.com:50051",protocol="grpc",service="user.UserService"} 1234
```

## See Also

- [API Reference](api.md) - HTTP API endpoints
- [Operations: Monitoring](../operations/monitoring.md) - Setting up monitoring


