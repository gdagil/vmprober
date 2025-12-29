package e2e

// BaseConfig returns the base configuration for tests
func BaseConfig() string {
	return `listen:
  port: 8429
  host: "0.0.0.0"

pull:
  enabled: true
  path: "/metrics"
  timeout: 10s

push:
  enabled: true
  endpoints:
    - url: "http://localhost:8480/insert/0/prometheus/api/v1/import"
      headers: {}
      auth:
        type: "none"
  retry:
    max_attempts: 5
    backoff: "exponential"
    initial_delay: 1s
    max_delay: 60s
    multiplier: 2.0
  dedup:
    enabled: false
  batch:
    size: 100
    timeout: 10s
  remote_write:
    enabled: false

scheduler:
  concurrent: 10
  rps_limit: 100
  per_host_cap: 5
  jitter: 0.1
  timeouts:
    tcp: 5s
    udp: 3s
    icmp: 2s
    http: 10s
    https: 10s
    dns: 5s
  queue_size: 1000
  worker_timeout: 30s

probes:
  defaults:
    count: 1
    interval: 5s
    timeout: 2s
    payload_size: 64
  tcp:
    connect_timeout: 2s
    tls:
      enabled: false
    keep_alive:
      enabled: true
      period: 30s
  udp:
    payload_type: "random"
    payload_size: 64
    response_timeout: 2s
    max_packet_size: 1024
  icmp:
    library: "systicmp"
    sequence_start: 1
    ttl: 64
  http:
    method: "GET"
    path: "/"
    expected_status_code: 200
    follow_redirects: true
    timeout: 10s
  dns:
    query_type: "A"
    protocol: "udp"
    validate_answer: true
    recursion: true
    timeout: 5s

metrics:
  namespace: "vmprober"
  include_labels:
    - "job"
    - "instance"
    - "probe"
    - "target"
    - "proto"
  custom_labels:
    job: "blackbox/vmprober"
  buckets: [0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0]
  enable_process_metrics: false
  enable_go_metrics: false
  enable_job_metrics: true

wal:
  dir: ""

logging:
  level: "info"
  format: "text"
  output: "stdout"
  structured: true
  include_source: true

observability:
  pprof:
    enabled: false
    port: 6060
    host: "127.0.0.1"
  opencensus:
    enabled: false
    sampling_rate: 0.1
    exporters: []
  prometheus:
    enabled: false
    namespace: "vmprober"
    subsystem: "process"
  health_check:
    enabled: true
    timeout: 5s
    interval: 30s
`
}

// TCPProbesConfig returns configuration with TCP probes
func TCPProbesConfig() string {
	return BaseConfig() + `
targets:
  static:
    - host: "127.0.0.1"
      port: 22
      proto: "tcp"
      interval: 5s
      timeout: 2s
      labels:
        test: "e2e"
        target: "localhost-ssh"
    - host: "8.8.8.8"
      port: 53
      proto: "tcp"
      interval: 5s
      timeout: 2s
      labels:
        test: "e2e"
        target: "google-dns"
    - host: "google.com"
      port: 80
      proto: "tcp"
      interval: 5s
      timeout: 2s
      labels:
        test: "e2e"
        target: "google-http"
    - host: "google.com"
      port: 443
      proto: "tcp"
      interval: 5s
      timeout: 2s
      labels:
        test: "e2e"
        target: "google-https"
`
}

// HTTPProbesConfig returns configuration with HTTP/HTTPS probes
func HTTPProbesConfig() string {
	return BaseConfig() + `
targets:
  static:
    - host: "httpbin.org"
      port: 80
      proto: "http"
      interval: 10s
      timeout: 5s
      labels:
        test: "e2e"
        target: "httpbin-get"
      http:
        method: "GET"
        path: "/status/200"
        expected_status_code: 200
        follow_redirects: true
    - host: "httpbin.org"
      port: 80
      proto: "http"
      interval: 10s
      timeout: 5s
      labels:
        test: "e2e"
        target: "httpbin-post"
      http:
        method: "POST"
        path: "/post"
        body: '{"test": "data"}'
        expected_status_code: 200
        headers:
          Content-Type: "application/json"
    - host: "github.com"
      port: 443
      proto: "https"
      interval: 10s
      timeout: 5s
      labels:
        test: "e2e"
        target: "github"
      http:
        method: "GET"
        path: "/"
        expected_status_code: 200
        validate_cert: true
        follow_redirects: true
`
}

// DNSProbesConfig returns configuration with DNS probes
func DNSProbesConfig() string {
	return BaseConfig() + `
targets:
  static:
    - host: "8.8.8.8"
      port: 53
      proto: "dns"
      interval: 10s
      timeout: 5s
      labels:
        test: "e2e"
        target: "google-dns-a"
      dns:
        query_name: "google.com"
        query_type: "A"
        protocol: "udp"
        validate_answer: true
        recursion: true
    - host: "8.8.8.8"
      port: 53
      proto: "dns"
      interval: 10s
      timeout: 5s
      labels:
        test: "e2e"
        target: "google-dns-aaaa"
      dns:
        query_name: "google.com"
        query_type: "AAAA"
        protocol: "udp"
    - host: "1.1.1.1"
      port: 53
      proto: "dns"
      interval: 10s
      timeout: 5s
      labels:
        test: "e2e"
        target: "cloudflare-dns"
      dns:
        query_name: "cloudflare.com"
        query_type: "A"
        protocol: "udp"
`
}

// ICMPProbesConfig returns configuration with ICMP probes
func ICMPProbesConfig() string {
	return BaseConfig() + `
targets:
  static:
    - host: "8.8.8.8"
      proto: "icmp"
      interval: 5s
      timeout: 2s
      labels:
        test: "e2e"
        target: "google-dns-ping"
    - host: "1.1.1.1"
      proto: "icmp"
      interval: 5s
      timeout: 2s
      labels:
        test: "e2e"
        target: "cloudflare-ping"
`
}

// MixedProbesConfig returns configuration with mixed probe types
func MixedProbesConfig() string {
	return BaseConfig() + `
targets:
  static:
    # TCP probes
    - host: "127.0.0.1"
      port: 22
      proto: "tcp"
      interval: 5s
      timeout: 2s
      labels:
        test: "e2e"
        target: "localhost-ssh"
    - host: "8.8.8.8"
      port: 53
      proto: "tcp"
      interval: 5s
      timeout: 2s
      labels:
        test: "e2e"
        target: "google-dns-tcp"
    # DNS probes
    - host: "8.8.8.8"
      port: 53
      proto: "dns"
      interval: 10s
      timeout: 5s
      labels:
        test: "e2e"
        target: "google-dns"
      dns:
        query_name: "google.com"
        query_type: "A"
        protocol: "udp"
`
}

// HighLoadConfig returns configuration for high-load testing
func HighLoadConfig() string {
	return BaseConfig() + `
scheduler:
  concurrent: 50
  rps_limit: 500
  per_host_cap: 10
  jitter: 0.1
  queue_size: 5000
  worker_timeout: 30s

targets:
  static:
    - host: "127.0.0.1"
      port: 22
      proto: "tcp"
      interval: 1s
      timeout: 2s
      labels:
        test: "e2e-highload"
        target: "localhost-fast"
    - host: "8.8.8.8"
      port: 53
      proto: "tcp"
      interval: 1s
      timeout: 2s
      labels:
        test: "e2e-highload"
        target: "dns-fast"
    - host: "1.1.1.1"
      port: 53
      proto: "tcp"
      interval: 1s
      timeout: 2s
      labels:
        test: "e2e-highload"
        target: "cloudflare-fast"
`
}

// FailingProbesConfig returns configuration with probes that should fail
func FailingProbesConfig() string {
	return BaseConfig() + `
targets:
  static:
    # This should fail - port is closed
    - host: "127.0.0.1"
      port: 9999
      proto: "tcp"
      interval: 5s
      timeout: 2s
      labels:
        test: "e2e-fail"
        target: "localhost-closed"
    # This should succeed
    - host: "8.8.8.8"
      port: 53
      proto: "tcp"
      interval: 5s
      timeout: 2s
      labels:
        test: "e2e-fail"
        target: "dns-open"
`
}

