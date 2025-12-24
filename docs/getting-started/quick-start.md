# Quick Start

Get VMProber up and running in 5 minutes.

## Step 1: Create Configuration

Create a `config.yaml` file:

```yaml
listen:
  port: 8429
  host: "0.0.0.0"

targets:
  static:
    - host: "google.com"
      port: 80
      protocols: ["tcp"]          # Array of protocols
      interval: 30s
      timeout: 5s
      labels:
        service: "web"
        environment: "production"

metrics:
  namespace: "vmprober"
```

## Step 2: Start VMProber

```bash
./bin/vmprober --config=config.yaml
```

You should see output like:

```
INFO    starting VMProber
INFO    HTTP server listening on 0.0.0.0:8429
INFO    scheduler started
INFO    probe scheduled    target=google.com:80
```

## Step 3: Verify It's Working

### Check Health

```bash
curl http://localhost:8429/health
```

Expected response:

```json
{
  "status": "healthy",
  "uptime": "1m23s"
}
```

### Check Metrics

```bash
curl http://localhost:8429/metrics
```

You should see Prometheus metrics including:

```
vmprober_probe_attempts_total{protocol="tcp",target="google.com:80"} 5
vmprober_probe_success_total{protocol="tcp",target="google.com:80"} 5
vmprober_probe_rtt_seconds_bucket{protocol="tcp",target="google.com:80",le="0.1"} 5
```

## Step 4: Add More Targets

Edit `config.yaml` to add more targets:

```yaml
targets:
  static:
    - host: "google.com"
      port: 80
      protocols: ["tcp"]
      interval: 30s
    - host: "8.8.8.8"
      port: 53
      protocols: ["udp", "tcp"]   # Probe both UDP and TCP
      interval: 60s
      labels:
        service: "dns"
    - host: "1.1.1.1"
      protocols: ["icmp"]
      interval: 30s
      labels:
        service: "ping"
```

VMProber supports hot reload, so changes are picked up automatically.

## Step 5: Integrate with Prometheus

Add to your `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'vmprober'
    scrape_interval: 30s
    static_configs:
      - targets: ['localhost:8429']
    metrics_path: '/metrics'
```

## Common First Steps

### Monitor Multiple Ports

```yaml
targets:
  static:
    - host: "example.com"
      port: 80
      protocols: ["tcp"]
      interval: 30s
      labels:
        port: "http"
    - host: "example.com"
      port: 443
      protocols: ["tcp"]
      interval: 30s
      labels:
        port: "https"
```

### Use HTTPS/TLS

```yaml
targets:
  static:
    - host: "example.com"
      port: 443
      protocols: ["tcp"]
      interval: 30s

probes:
  tcp:
    tls:
      enabled: true
      server_name: "example.com"
```

### Configure Rate Limiting

```yaml
scheduler:
  concurrent: 50
  rps_limit: 500
  per_host_cap: 5
```

## What's Next?

- [Configuration Guide](configuration.md) - Learn all configuration options
- [Basic Usage](basic-usage.md) - Common usage patterns
- [Operations: Deployment](../operations/deployment.md) - Deploy to production
- [Guides: Monitoring Setup](../guides/monitoring-setup.md) - Set up Grafana dashboards


