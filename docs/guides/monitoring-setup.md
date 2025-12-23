# Monitoring Setup

Set up monitoring for VMProber using Prometheus and Grafana.

## Prerequisites

- Prometheus installed and running
- Grafana installed and running (optional)
- VMProber running and accessible

## Prometheus Configuration

### 1. Add VMProber to Prometheus

Edit `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'vmprober'
    scrape_interval: 30s
    scrape_timeout: 10s
    static_configs:
      - targets: ['vmprober:8429']
    metrics_path: '/metrics'
```

### 2. Reload Prometheus

```bash
# If using systemd
sudo systemctl reload prometheus

# Or send SIGHUP
kill -HUP <prometheus-pid>
```

### 3. Verify Scraping

Check Prometheus targets page: `http://prometheus:9090/targets`

## Grafana Dashboard

### 1. Create Dashboard

1. Go to Grafana: `http://grafana:3000`
2. Create new dashboard
3. Add panels for key metrics

### 2. Key Panels

#### Success Rate

```promql
rate(vmprober_probe_success_total[5m]) / rate(vmprober_probe_attempts_total[5m])
```

#### Average RTT

```promql
rate(vmprober_probe_rtt_seconds_sum[5m]) / rate(vmprober_probe_rtt_seconds_count[5m])
```

#### 95th Percentile RTT

```promql
histogram_quantile(0.95, rate(vmprober_probe_rtt_seconds_bucket[5m]))
```

#### Failure Rate by Target

```promql
rate(vmprober_probe_failure_total[5m]) / rate(vmprober_probe_attempts_total[5m])
```

## Alerts

### High Failure Rate

```yaml
- alert: HighProbeFailureRate
  expr: |
    rate(vmprober_probe_failure_total[5m]) / 
    rate(vmprober_probe_attempts_total[5m]) > 0.1
  for: 5m
  annotations:
    summary: "High probe failure rate for {{ $labels.target }}"
```

### High Latency

```yaml
- alert: HighProbeLatency
  expr: |
    histogram_quantile(0.95, 
      rate(vmprober_probe_rtt_seconds_bucket[5m])
    ) > 1
  for: 5m
  annotations:
    summary: "High probe latency for {{ $labels.target }}"
```

## Service Discovery

For Kubernetes, use ServiceMonitor:

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: vmprober
spec:
  selector:
    matchLabels:
      app: vmprober
  endpoints:
  - port: http
    path: /metrics
    interval: 30s
```

## Next Steps

- [Operations: Monitoring](../operations/monitoring.md) - Advanced monitoring
- [Reference: Metrics](../reference/metrics.md) - All available metrics

