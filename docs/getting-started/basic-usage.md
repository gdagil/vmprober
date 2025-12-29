# Basic Usage

Common usage patterns and examples for VMProber.

## Monitoring a Web Server

```yaml
targets:
  static:
    - host: "example.com"
      port: 80
      protocols: ["tcp"]          # Array of protocols
      interval: 30s
      timeout: 5s
      labels:
        service: "web"
        environment: "production"
```

## Monitoring HTTPS with TLS

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
      insecure_skip_verify: false
```

## Monitoring DNS Server with Multiple Protocols

```yaml
targets:
  static:
    - host: "8.8.8.8"
      port: 53
      protocols: ["udp", "tcp"]   # Probe both UDP and TCP
      interval: 60s
      timeout: 3s
      labels:
        service: "dns"
        provider: "google"
```

This creates two separate probe jobs: one for UDP and one for TCP, allowing you to monitor both protocols for the same target.

## Monitoring with ICMP

```yaml
targets:
  static:
    - host: "1.1.1.1"
      protocols: ["icmp"]
      interval: 30s
      timeout: 2s
      labels:
        service: "ping"
```

**Note**: ICMP probes require root/administrator privileges.

## Multiple Ports on One Host

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
    - host: "example.com"
      port: 22
      protocols: ["tcp"]
      interval: 60s
      labels:
        port: "ssh"
```

## Using File-Based Targets

Create `/etc/vmprober/targets.yaml`:

```yaml
- host: "service1.example.com"
  port: 8429
  protocols: ["tcp"]
  interval: 30s
  labels:
    service: "api"
- host: "service2.example.com"
  port: 9090
  protocols: ["tcp"]
  interval: 30s
  labels:
    service: "metrics"
```

Configure VMProber:

```yaml
targets:
  files:
    - path: "/etc/vmprober/targets.yaml"
      reload_interval: 1m
      watch: true
```

## Using HTTP Discovery

```yaml
targets:
  urls:
    - url: "http://service-discovery:8429/api/targets"
      reload_interval: 5m
      headers:
        Authorization: "Bearer ${DISCOVERY_TOKEN}"
```

## Rate Limiting

Control probe frequency:

```yaml
scheduler:
  concurrent: 50        # Max 50 concurrent probes
  rps_limit: 500        # Max 500 probes/second globally
  per_host_cap: 5      # Max 5 probes/second per host
  jitter: 0.2          # 20% jitter for load distribution
```

## Pushing to VictoriaMetrics

```yaml
push:
  enabled: true
  endpoints:
    - url: "http://victoria-metrics:8428/api/v1/import/prometheus"
      auth:
        type: "bearer"
        token: "${VM_TOKEN}"
  retry:
    max_attempts: 5
    backoff: "exponential"
    initial_delay: 1s
    max_delay: 60s
  batch:
    size: 1000
    timeout: 30s
```

## Enabling WAL for Reliability

```yaml
wal:
  dir: "/var/lib/vmprober/wal"
  max_size: "2GB"
  max_age: 168h
  retention: 7d
  compression: "gzip"
  sync_interval: 1s
```

## Structured Logging

```yaml
logging:
  level: "info"
  format: "json"
  output: "stdout"
  structured: true
  include_source: true
  file:
    path: "/var/log/vmprober.log"
    max_size: "100MB"
    max_backups: 10
    max_age: 30
    compress: true
```

## Debug Mode

Enable debug logging:

```yaml
logging:
  level: "debug"
  format: "json"
```

Or via command line:

```bash
./bin/vmprober --config=config.yaml --log-level=debug
```

## Health Checks

Check if VMProber is running:

```bash
curl http://localhost:8429/health
```

Check if ready to accept traffic:

```bash
curl http://localhost:8429/ready
```

## Viewing Metrics

Get all metrics:

```bash
curl http://localhost:8429/metrics
```

Filter specific metrics:

```bash
curl http://localhost:8429/metrics | grep vmprober_probe_success_total
```

## Next Steps

- [Configuration Guide](configuration.md) - Complete configuration reference
- [Operations: Deployment](../operations/deployment.md) - Production deployment
- [Guides: Monitoring Setup](../guides/monitoring-setup.md) - Set up Grafana
