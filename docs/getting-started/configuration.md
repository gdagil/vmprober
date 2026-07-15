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
    # TCP probe
    - host: "example.com"
      port: 80
      protocols: ["tcp"]
      interval: 30s
      timeout: 5s
      labels:
        service: "web"

    # Multiple protocols for the same target
    - host: "8.8.8.8"
      port: 53
      protocols: ["udp", "tcp"]   # Probe both UDP and TCP
      interval: 60s
      timeout: 3s
      labels:
        service: "dns"

    # HTTP/HTTPS probe
    - host: "api.example.com"
      port: 443
      proto: "https"
      interval: 30s
      http:
        method: "GET"
        path: "/health"
        expected_status_code: 200
        validate_cert: true
        headers:
          Accept: "application/json"

    # DNS probe
    - host: "8.8.8.8"
      port: 53
      proto: "dns"
      interval: 30s
      dns:
        query_name: "google.com"
        query_type: "A"
        protocol: "udp"

    # gRPC probe
    - host: "grpc.example.com"
      port: 50051
      proto: "grpc"
      interval: 15s
      grpc:
        service: "user.UserService"
        expected_status: "SERVING"

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

**Supported Protocols:**
- `tcp` - TCP connection check
- `udp` - UDP packet probe
- `icmp` - ICMP ping (requires root)
- `http` - HTTP request probe
- `https` - HTTPS request probe with TLS
- `dns` - DNS query probe
- `grpc` - gRPC health check probe

**Note**: The `protocols` field accepts an array of protocol names. For each protocol specified, VMProber will create a separate probe job. This allows monitoring the same target with multiple protocols simultaneously.

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
    http: 10s
    https: 10s
    dns: 5s
    grpc: 10s
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

  # TCP probe defaults
  tcp:
    connect_timeout: 5s
    tls:
      enabled: false
    keep_alive:
      enabled: true
      period: 30s

  # UDP probe defaults
  udp:
    payload_type: "random"
    payload_size: 64
    response_timeout: 2s
    max_packet_size: 1024

  # ICMP probe defaults
  icmp:
    library: "systicmp"
    sequence_start: 1
    ttl: 64

  # HTTP/HTTPS probe defaults
  http:
    method: "GET"
    path: "/"
    expected_status_code: 200
    follow_redirects: true
    max_redirects: 10
    validate_cert: true
    timeout: 10s
    headers:
      User-Agent: "vmprober/1.0"

  # DNS probe defaults
  dns:
    query_type: "A"         # A, AAAA, CNAME, MX, TXT, NS, SOA, SRV, PTR
    protocol: "udp"         # udp, tcp, tcp-tls (DoT)
    validate_answer: true
    recursion: true
    timeout: 5s

  # gRPC probe defaults
  grpc:
    service: ""             # Empty = overall health check
    tls: false
    tls_skip_verify: false
    expected_status: "SERVING"  # SERVING, NOT_SERVING, UNKNOWN
    timeout: 10s
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
    prober: "dc1-a"  # identity of this instance, defaults to the hostname
  buckets: [0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0]
  enable_process_metrics: true
  enable_go_metrics: true
```

Every instance automatically gets a `prober` label (hostname by default) so
that several vmprobers can push to the same VictoriaMetrics without their
series colliding. Override it via `custom_labels.prober` when running more
than one instance on the same host.

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
