# Configuration Guide

VMProber uses YAML configuration files to control all aspects of operation.

## Configuration File Location

VMProber looks for configuration in this order:

1. Command-line argument: `--config=path/to/config.yaml`
2. Environment variable: `VMPROBER_CONFIG=path/to/config.yaml`
3. Default: `config.yaml` in current directory

## Basic Configuration Structure

```yaml
# HTTP Server
listen:
  port: 8429
  host: "0.0.0.0"

# Pull Mode (Prometheus scrape)
pull:
  enabled: true
  path: "/metrics"

# Push Mode (VictoriaMetrics)
push:
  enabled: false
  endpoints: []

# Scheduler
scheduler:
  concurrent: 100
  rps_limit: 1000

# Targets to monitor
targets:
  static: []

# Probe defaults
probes:
  defaults:
    timeout: 5s

# Metrics
metrics:
  namespace: "vmprober"

# Logging
logging:
  level: "info"
  format: "json"
```

## Configuration Sections

### HTTP Server (`listen`)

Controls the HTTP server that serves metrics and health checks.

```yaml
listen:
  port: 8429              # Port to listen on
  host: "0.0.0.0"         # Host to bind to
  tls:
    enabled: false        # Enable TLS
    cert_file: ""         # Certificate file path
    key_file: ""          # Key file path
```

### Targets (`targets`)

Define what to monitor. Supports multiple sources:

```yaml
targets:
  # Static targets (defined in config)
  static:
    - host: "example.com"
      port: 80
      proto: "tcp"
      interval: 30s
      timeout: 5s
      labels:
        service: "web"
  
  # File-based targets
  files:
    - path: "/etc/vmprober/targets.yaml"
      reload_interval: 1m
      watch: true
  
  # HTTP-based targets
  urls:
    - url: "http://discovery:8429/targets"
      reload_interval: 5m
  
  # Command-based targets
  commands:
    - command: "/usr/bin/get-targets.sh"
      interval: 10m
```

### Scheduler (`scheduler`)

Controls how probes are scheduled and executed.

```yaml
scheduler:
  concurrent: 100        # Max concurrent probes
  rps_limit: 1000       # Global RPS limit
  per_host_cap: 10      # Max probes per host
  jitter: 0.1           # Load distribution (0.0-1.0)
  timeouts:
    tcp: 5s
    udp: 3s
    icmp: 2s
  queue_size: 10000
```

### Probes (`probes`)

Default settings for different probe types.

```yaml
probes:
  defaults:
    count: 3
    interval: 30s
    timeout: 5s
  
  tcp:
    connect_timeout: 5s
    tls:
      enabled: false
  
  udp:
    payload_type: "random"
    payload_size: 64
  
  icmp:
    library: "systicmp"
    ttl: 64
```

### Metrics (`metrics`)

Metrics collection and export settings.

```yaml
metrics:
  namespace: "vmprober"
  include_labels:
    - "job"
    - "instance"
    - "probe"
  custom_labels:
    environment: "production"
  buckets: [0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0]
  enable_process_metrics: true
  enable_go_metrics: true
```

### Push Mode (`push`)

VictoriaMetrics push configuration.

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

### WAL (`wal`)

Write-Ahead Log configuration for reliability.

```yaml
wal:
  dir: "/var/lib/vmprober/wal"
  max_size: "1GB"
  max_age: 168h
  retention: 30d
  compression: "gzip"
  sync_interval: 1s
```

### Logging (`logging`)

Logging configuration.

```yaml
logging:
  level: "info"          # debug, info, warn, error
  format: "json"         # json, text
  output: "stdout"       # stdout, stderr, file
  file:
    path: "/var/log/vmprober.log"
    max_size: "100MB"
    max_backups: 10
```

## Hot Reload

VMProber supports hot reload of configuration:

```yaml
targets:
  hot_reload: true
```

When enabled, VMProber watches the config file and reloads automatically when changes are detected.

## Environment Variables

You can use environment variables in configuration:

```yaml
push:
  endpoints:
    - url: "${VM_ENDPOINT}"
      auth:
        token: "${VM_TOKEN}"
```

## Validation

VMProber validates configuration on startup. Invalid configuration will prevent startup with clear error messages.

## Complete Example

See `config.yaml.example` in the project root for a complete configuration example.

## Next Steps

- [Reference: Configuration Reference](../reference/configuration-reference.md) - Complete configuration options
- [Operations: Configuration Management](../operations/configuration.md) - Managing configs in production


