# Probe System

VMProber supports multiple probe types for comprehensive network and service monitoring.

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
      protocols: ["tcp"]
      interval: 30s
      timeout: 5s

probes:
  tcp:
    connect_timeout: 5s
    tls:
      enabled: false
      server_name: ""
```

---

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
      protocols: ["udp"]
      interval: 60s

probes:
  udp:
    payload_type: "random"  # random, echo, custom
    payload_size: 64
    response_timeout: 2s
```

---

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
      protocols: ["icmp"]
      interval: 30s

probes:
  icmp:
    library: "systicmp"  # systicmp, gopacket
    sequence_start: 1
    ttl: 64
```

**Note**: ICMP probes require root/administrator privileges.

---

### HTTP/HTTPS Probes
Perform HTTP requests to check web services availability and health.

**Features:**
- All HTTP methods (GET, POST, PUT, DELETE, HEAD, OPTIONS, PATCH)
- TLS/SSL with certificate validation
- Custom headers and request body
- Expected status code validation
- Response body substring matching
- Basic Auth and Bearer Token authentication
- HTTP proxy support
- Configurable redirect following

**Configuration:**
```yaml
targets:
  static:
    # Simple HTTP health check
    - host: "api.example.com"
      port: 80
      proto: "http"
      interval: 30s
      timeout: 10s
      http:
        method: "GET"
        path: "/health"
        expected_status_code: 200

    # HTTPS with custom headers
    - host: "secure.example.com"
      port: 443
      proto: "https"
      interval: 30s
      http:
        method: "GET"
        path: "/api/status"
        expected_status_code: 200
        validate_cert: true
        headers:
          Accept: "application/json"
          X-Api-Key: "your-api-key"

    # POST request with body
    - host: "api.example.com"
      port: 443
      proto: "https"
      interval: 60s
      http:
        method: "POST"
        path: "/api/health"
        body: '{"check": "full"}'
        expected_status_code: 200
        expected_body: "healthy"
        headers:
          Content-Type: "application/json"

    # With Basic Auth
    - host: "protected.example.com"
      port: 443
      proto: "https"
      interval: 60s
      http:
        method: "GET"
        path: "/admin/health"
        expected_status_code: 200
        basic_auth:
          username: "admin"
          password: "secret"

    # With Bearer Token
    - host: "api.example.com"
      port: 443
      proto: "https"
      interval: 60s
      http:
        method: "GET"
        path: "/api/protected"
        expected_status_code: 200
        bearer_token: "${API_TOKEN}"

    # Through HTTP proxy
    - host: "internal.example.com"
      port: 443
      proto: "https"
      http:
        method: "GET"
        path: "/health"
        validate_cert: false
        proxy: "http://proxy.corp:8080"

probes:
  http:
    method: "GET"
    path: "/"
    expected_status_code: 200
    follow_redirects: true
    max_redirects: 10
    validate_cert: true
    timeout: 10s
```

---

### DNS Probes
Check DNS server availability and validate DNS responses.

**Features:**
- Multiple record types: A, AAAA, CNAME, MX, TXT, NS, SOA, SRV, PTR, CAA, DNSKEY, DS
- UDP, TCP, and DNS-over-TLS (DoT) protocols
- Expected records validation
- RCODE validation
- Recursive query support

**Configuration:**
```yaml
targets:
  static:
    # Simple A record query
    - host: "8.8.8.8"
      port: 53
      proto: "dns"
      interval: 30s
      timeout: 5s
      dns:
        query_name: "google.com"
        query_type: "A"
        protocol: "udp"
        validate_answer: true
        recursion: true

    # AAAA record (IPv6)
    - host: "8.8.8.8"
      port: 53
      proto: "dns"
      interval: 60s
      dns:
        query_name: "google.com"
        query_type: "AAAA"

    # MX records with validation
    - host: "8.8.8.8"
      port: 53
      proto: "dns"
      interval: 120s
      dns:
        query_name: "gmail.com"
        query_type: "MX"
        expected_records: ["gmail-smtp-in.l.google.com"]

    # TXT records (SPF, DKIM, etc.)
    - host: "1.1.1.1"
      port: 53
      proto: "dns"
      interval: 300s
      dns:
        query_name: "example.com"
        query_type: "TXT"

    # DNS over TCP
    - host: "8.8.8.8"
      port: 53
      proto: "dns"
      dns:
        query_name: "google.com"
        query_type: "A"
        protocol: "tcp"

    # DNS-over-TLS (DoT)
    - host: "1.1.1.1"
      port: 853
      proto: "dns"
      interval: 60s
      dns:
        query_name: "cloudflare.com"
        query_type: "A"
        protocol: "tcp-tls"

probes:
  dns:
    query_type: "A"
    protocol: "udp"
    validate_answer: true
    recursion: true
    timeout: 5s
```

---

### gRPC Probes
Check gRPC service health using the standard Health Check Protocol.

**Features:**
- Standard gRPC Health Check Protocol (`grpc.health.v1.Health`)
- Per-service health checks
- TLS support with certificate validation
- Custom gRPC metadata (headers)
- Expected status validation (SERVING, NOT_SERVING, UNKNOWN, SERVICE_UNKNOWN)

**Configuration:**
```yaml
targets:
  static:
    # Overall health check (empty service)
    - host: "grpc.example.com"
      port: 50051
      proto: "grpc"
      interval: 15s
      timeout: 5s
      grpc:
        service: ""  # Empty = overall health
        expected_status: "SERVING"

    # Specific service health check
    - host: "grpc.example.com"
      port: 50051
      proto: "grpc"
      interval: 30s
      grpc:
        service: "user.UserService"
        expected_status: "SERVING"

    # gRPC with TLS
    - host: "secure-grpc.example.com"
      port: 443
      proto: "grpc"
      interval: 30s
      grpc:
        service: "api.HealthService"
        tls: true
        tls_skip_verify: false
        server_name: "secure-grpc.example.com"
        expected_status: "SERVING"

    # gRPC with metadata (headers)
    - host: "auth-grpc.example.com"
      port: 50051
      proto: "grpc"
      interval: 30s
      grpc:
        service: "auth.AuthService"
        expected_status: "SERVING"
        metadata:
          authorization: "Bearer ${GRPC_TOKEN}"
          x-request-id: "vmprober-health-check"

probes:
  grpc:
    service: ""
    tls: false
    tls_skip_verify: false
    expected_status: "SERVING"
    timeout: 10s
```

---

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
    
    - host: "api.example.com"
      port: 443
      protocols: ["tcp", "https"]  # Check both TCP and HTTP
      interval: 30s
      http:
        method: "GET"
        path: "/health"
```

This is useful when you want to monitor the same service using different protocols.

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


