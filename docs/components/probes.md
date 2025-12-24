# Probe System

VMProber supports multiple probe types for comprehensive network monitoring.

## Supported Probe Types

### TCP Probes
Connect to TCP ports to check service availability.

**Features:**
- TCP connection establishment
- TLS/SSL support
- Configurable timeouts
- IPv4 and IPv6 support

**Configuration:**
```yaml
targets:
  static:
    - host: "example.com"
      port: 80
      protocols: ["tcp"]          # Array of protocols
      interval: 30s
      timeout: 5s

probes:
  tcp:
    connect_timeout: 5s
    tls:
      enabled: false
      server_name: ""
```

### UDP Probes
Send UDP packets and optionally wait for responses.

**Features:**
- UDP packet sending
- Response detection
- Configurable payload
- Timeout handling

**Configuration:**
```yaml
targets:
  static:
    - host: "8.8.8.8"
      port: 53
      protocols: ["udp"]          # Array of protocols
      interval: 60s

probes:
  udp:
    payload_type: "random"  # random, echo, custom
    payload_size: 64
    response_timeout: 2s
```

**Multiple Protocols Example:**
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
```

### ICMP Probes
Ping hosts to check network connectivity.

**Features:**
- ICMP echo requests
- Round-trip time measurement
- IPv4 and IPv6 support
- Configurable TTL

**Configuration:**
```yaml
targets:
  static:
    - host: "1.1.1.1"
      protocols: ["icmp"]   # Array of protocols (single item for ICMP)
      interval: 30s

probes:
  icmp:
    library: "systicmp"  # systicmp, gopacket
    sequence_start: 1
    ttl: 64
```

**Note**: ICMP probes require root/administrator privileges.

## Probe Interface

All probes implement a common interface:

```go
type Probe interface {
    Execute(ctx context.Context, target Target) (*ProbeResult, error)
    Type() ProbeType
    Validate(config interface{}) error
    Close() error
}
```

## Probe Results

Each probe returns a `ProbeResult` containing:

- Success status
- Round-trip time (RTT)
- Error information (if any)
- Timestamp
- Source and target IPs
- Protocol-specific data

## Multiple Protocols per Target

You can specify multiple protocols for a single target. VMProber will create separate probe jobs for each protocol:

```yaml
targets:
  static:
    - host: "8.8.8.8"
      port: 53
      protocols: ["udp", "tcp"]   # Creates 2 separate jobs
      interval: 60s
      timeout: 3s
      labels:
        service: "dns"
```

This is useful when you want to monitor the same service using different protocols, such as checking both UDP and TCP connectivity for DNS servers.

## Probe Factory

Probes are created via a factory pattern:

```go
probe := factory.CreateProbe(ProbeTypeTCP, config)
```

## Adding New Probe Types

1. Implement the `Probe` interface
2. Register in the factory
3. Add configuration types
4. Add tests
5. Update documentation

See [Development: Contributing](../development/contributing.md) for details.

## Next Steps

- [Scheduler](scheduler.md) - How probes are scheduled
- [Metrics](metrics.md) - How probe results become metrics
- [Development: Contributing](../development/contributing.md) - Adding new probe types


