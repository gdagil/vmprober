// Package types contains basic data structures and types for VMProber
package types

import (
	"time"
)

// ProbeType defines the probe type
type ProbeType string

const (
	ProbeTypeTCP   ProbeType = "tcp"
	ProbeTypeUDP   ProbeType = "udp"
	ProbeTypeICMP  ProbeType = "icmp"
	ProbeTypeHTTP  ProbeType = "http"
	ProbeTypeHTTPS ProbeType = "https"
	ProbeTypeDNS   ProbeType = "dns"
	ProbeTypeGRPC  ProbeType = "grpc"
)

// NetworkFamily defines the network address family
type NetworkFamily string

const (
	NetworkFamilyInet  NetworkFamily = "inet"  // IPv4
	NetworkFamilyInet6 NetworkFamily = "inet6" // IPv6
	NetworkFamilyAny   NetworkFamily = "any"   // Any
)

// ProbeResult represents the probe execution result
type ProbeResult struct {
	Success      bool          `json:"success"`
	RTT          time.Duration `json:"rtt"`
	Error        string        `json:"error,omitempty"`
	Attempt      int           `json:"attempt"`
	Timestamp    time.Time     `json:"timestamp"`
	SourceIP     string        `json:"source_ip,omitempty"`
	TargetHost   string        `json:"target_host,omitempty"` // Hostname (e.g., "google.com")
	TargetIP     string        `json:"target_ip,omitempty"`   // IP address (e.g., "142.250.109.100")
	TargetPort   int           `json:"target_port,omitempty"` // Port (e.g., 443)
	TLS          bool          `json:"tls,omitempty"`
	Protocol     ProbeType     `json:"protocol"`
	Role         string        `json:"role,omitempty"` // client/server
	SocketFamily string        `json:"socket_family,omitempty"`
	Payload      []byte        `json:"payload,omitempty"`
	Response     []byte        `json:"response,omitempty"`
	DNSResult    *DNSResult    `json:"dns_result,omitempty"`
}

// DNSResult DNS resolution result
type DNSResult struct {
	ResolvedIPs []string      `json:"resolved_ips"`
	LookupTime  time.Duration `json:"lookup_time"`
	TTL         time.Duration `json:"ttl,omitempty"`
	Error       string        `json:"error,omitempty"`
}

// Target represents a probe target
type Target struct {
	ID            string            `json:"id"`
	Host          string            `json:"host"`
	Port          int               `json:"port,omitempty"`
	Protocol      ProbeType         `json:"protocol"`
	Interval      time.Duration     `json:"interval"`
	Timeout       time.Duration     `json:"timeout"`
	Count         int               `json:"count"`
	Labels        map[string]string `json:"labels,omitempty"`
	NetworkFamily NetworkFamily     `json:"network_family,omitempty"`
	TLS           *TLSConfig        `json:"tls,omitempty"`
	UDP           *UDPConfig        `json:"udp,omitempty"`
	ICMP          *ICMPConfig       `json:"icmp,omitempty"`
	HTTP          *HTTPConfig       `json:"http,omitempty"`
	DNS           *DNSConfig        `json:"dns,omitempty"`
	GRPC          *GRPCConfig       `json:"grpc,omitempty"`
	Enabled       bool              `json:"enabled"`
	Priority      int               `json:"priority,omitempty"`
	CreatedAt     time.Time         `json:"created_at,omitempty"`
	UpdatedAt     time.Time         `json:"updated_at,omitempty"`
}

// TLSConfig TLS configuration
type TLSConfig struct {
	Enabled            bool     `json:"enabled"`
	InsecureSkipVerify bool     `json:"insecure_skip_verify"`
	ServerName         string   `json:"server_name"`
	MinVersion         string   `json:"min_version"`
	MaxVersion         string   `json:"max_version"`
	CipherSuites       []string `json:"cipher_suites"`
	RootCAs            string   `json:"root_cas"`
	ClientCert         string   `json:"client_cert,omitempty"`
	ClientKey          string   `json:"client_key,omitempty"`
}

// UDPConfig UDP configuration
type UDPConfig struct {
	PayloadType     string        `json:"payload_type"` // echo, random
	PayloadSize     int           `json:"payload_size"`
	ResponseTimeout time.Duration `json:"response_timeout"`
	MaxPacketSize   int           `json:"max_packet_size"`
	BindAddress     string        `json:"bind_address,omitempty"`
}

// ICMPConfig ICMP configuration
type ICMPConfig struct {
	Library       string `json:"library"` // systicmp, gopacket
	SequenceStart int    `json:"sequence_start"`
	TTL           int    `json:"ttl"`
	Data          []byte `json:"data,omitempty"`
}

// HTTPConfig HTTP/HTTPS probe configuration
type HTTPConfig struct {
	Method             string            `json:"method" yaml:"method"`                             // GET, POST, HEAD, PUT, etc.
	Path               string            `json:"path" yaml:"path"`                                 // URL path (e.g., "/health")
	Headers            map[string]string `json:"headers" yaml:"headers"`                           // Custom headers
	Body               string            `json:"body" yaml:"body"`                                 // Request body
	ExpectedStatusCode int               `json:"expected_status_code" yaml:"expected_status_code"` // Expected HTTP status (default: 200)
	ExpectedBody       string            `json:"expected_body" yaml:"expected_body"`               // Expected response body substring
	FollowRedirects    bool              `json:"follow_redirects" yaml:"follow_redirects"`         // Follow HTTP redirects
	MaxRedirects       int               `json:"max_redirects" yaml:"max_redirects"`               // Maximum redirects to follow
	ValidateCert       bool              `json:"validate_cert" yaml:"validate_cert"`               // Validate TLS certificate
	BasicAuth          *BasicAuth        `json:"basic_auth,omitempty" yaml:"basic_auth"`           // Basic authentication
	BearerToken        string            `json:"bearer_token,omitempty" yaml:"bearer_token"`       // Bearer token
	Proxy              string            `json:"proxy,omitempty" yaml:"proxy"`                     // HTTP proxy URL
}

// BasicAuth HTTP Basic Auth configuration
type BasicAuth struct {
	Username string `json:"username" yaml:"username"`
	Password string `json:"password" yaml:"password"`
}

// DNSConfig DNS probe configuration
type DNSConfig struct {
	QueryName       string   `json:"query_name" yaml:"query_name"`             // Domain to query
	QueryType       string   `json:"query_type" yaml:"query_type"`             // A, AAAA, CNAME, MX, TXT, NS, SOA, SRV, PTR
	Server          string   `json:"server" yaml:"server"`                     // DNS server address (e.g., "8.8.8.8:53")
	Protocol        string   `json:"protocol" yaml:"protocol"`                 // udp, tcp, tcp-tls (DoT)
	ExpectedRecords []string `json:"expected_records" yaml:"expected_records"` // Expected records in response
	ValidateAnswer  bool     `json:"validate_answer" yaml:"validate_answer"`   // Validate RCODE is NOERROR
	Recursion       bool     `json:"recursion" yaml:"recursion"`               // Request recursive resolution
}

// GRPCConfig gRPC probe configuration
type GRPCConfig struct {
	Service         string            `json:"service" yaml:"service"`                   // Service name for health check
	TLS             bool              `json:"tls" yaml:"tls"`                           // Use TLS
	TLSSkipVerify   bool              `json:"tls_skip_verify" yaml:"tls_skip_verify"`   // Skip TLS verification
	ServerName      string            `json:"server_name" yaml:"server_name"`           // TLS server name
	Metadata        map[string]string `json:"metadata" yaml:"metadata"`                 // gRPC metadata (headers)
	ExpectedStatus  string            `json:"expected_status" yaml:"expected_status"`   // Expected health status (SERVING, NOT_SERVING, UNKNOWN)
	ReflectionProbe bool              `json:"reflection_probe" yaml:"reflection_probe"` // Use gRPC reflection instead of health check
}

// Job represents a task for probe execution
type Job struct {
	ID         string        `json:"id"`
	Target     Target        `json:"target"`
	NextRun    time.Time     `json:"next_run"`
	Interval   time.Duration `json:"interval"`
	Jitter     float64       `json:"jitter"`
	RetryCount int           `json:"retry_count"`
	MaxRetries int           `json:"max_retries"`
	Priority   int           `json:"priority"`
	CreatedAt  time.Time     `json:"created_at"`
	Attempt    int           `json:"attempt"`
	// Probe statistics
	SuccessCount  int64     `json:"success_count"`
	FailedCount   int64     `json:"failed_count"`
	LastStatus    string    `json:"last_status"` // "up" or "down"
	LastProbeTime time.Time `json:"last_probe_time"`
}

// NormalizedEvent normalized event
type NormalizedEvent struct {
	Timestamp time.Time              `json:"timestamp"`
	SeriesID  string                 `json:"series_id"`
	Metrics   map[string]float64     `json:"metrics"`
	Labels    map[string]string      `json:"labels"`
	Tags      []string               `json:"tags"`
	Metadata  map[string]interface{} `json:"metadata"`
	Source    string                 `json:"source,omitempty"`
}

// Metric represents a metric
type Metric struct {
	Name      string              `json:"name"`
	Value     float64             `json:"value"`
	Timestamp time.Time           `json:"timestamp"`
	Labels    map[string]string   `json:"labels"`
	Type      MetricType          `json:"type"`
	Help      string              `json:"help,omitempty"`
	Buckets   []float64           `json:"buckets,omitempty"`
	Sum       float64             `json:"sum,omitempty"`
	Count     uint64              `json:"count,omitempty"`
	Quantiles map[float64]float64 `json:"quantiles,omitempty"`
}

// MetricType metric type
type MetricType string

const (
	MetricTypeGauge     MetricType = "gauge"
	MetricTypeCounter   MetricType = "counter"
	MetricTypeHistogram MetricType = "histogram"
	MetricTypeSummary   MetricType = "summary"
)

// Record represents a record in the storage
type Record struct {
	ID          string                 `json:"id"`
	Timestamp   time.Time              `json:"timestamp"`
	Type        string                 `json:"type"`
	Data        map[string]interface{} `json:"data"`
	Labels      map[string]string      `json:"labels,omitempty"`
	SeriesID    string                 `json:"series_id"`
	Compression string                 `json:"compression,omitempty"`
	Size        int64                  `json:"size,omitempty"`
	Sent        bool                   `json:"sent"`              // Successful send flag
	SentAt      time.Time              `json:"sent_at,omitempty"` // Send time
	Retries     int                    `json:"retries,omitempty"` // Number of send attempts
}

// ConfigUpdate configuration update event
// Config is defined in internal/config/types.go
type ConfigUpdate struct {
	Type      UpdateType     `json:"type"`
	OldConfig interface{}    `json:"old_config,omitempty"`
	NewConfig interface{}    `json:"new_config,omitempty"`
	Timestamp time.Time      `json:"timestamp"`
	Source    string         `json:"source"`
	Changes   []ConfigChange `json:"changes,omitempty"`
}

// UpdateType update type
type UpdateType string

const (
	UpdateTypeFull    UpdateType = "full"
	UpdateTypePartial UpdateType = "partial"
	UpdateTypeError   UpdateType = "error"
)

// ConfigChange change in configuration
type ConfigChange struct {
	Path     string      `json:"path"`
	OldValue interface{} `json:"old_value"`
	NewValue interface{} `json:"new_value"`
	Op       ChangeOp    `json:"op"`
}

// ChangeOp change operation
type ChangeOp string

const (
	ChangeOpAdd    ChangeOp = "add"
	ChangeOpUpdate ChangeOp = "update"
	ChangeOpDelete ChangeOp = "delete"
)

// HealthStatus system health status
type HealthStatus struct {
	Status    string                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Uptime    time.Duration          `json:"uptime"`
	Version   string                 `json:"version"`
	Checks    map[string]HealthCheck `json:"checks"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// HealthCheck component health check
type HealthCheck struct {
	Status    string        `json:"status"`
	Message   string        `json:"message,omitempty"`
	Timestamp time.Time     `json:"timestamp"`
	Duration  time.Duration `json:"duration,omitempty"`
	Details   interface{}   `json:"details,omitempty"`
}

// ReadyStatus system readiness status
type ReadyStatus struct {
	Ready      bool                   `json:"ready"`
	Timestamp  time.Time              `json:"timestamp"`
	Components map[string]ReadyCheck  `json:"components"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// ReadyCheck component readiness check
type ReadyCheck struct {
	Ready     bool        `json:"ready"`
	Message   string      `json:"message,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
	Details   interface{} `json:"details,omitempty"`
}

// RateLimitInfo rate limiting information
type RateLimitInfo struct {
	Key        string        `json:"key"`
	Rate       float64       `json:"rate"`
	Burst      int           `json:"burst"`
	Remaining  int           `json:"remaining"`
	ResetTime  time.Time     `json:"reset_time"`
	RetryAfter time.Duration `json:"retry_after,omitempty"`
}

// WorkerInfo worker information
type WorkerInfo struct {
	ID         string        `json:"id"`
	Status     WorkerStatus  `json:"status"`
	CurrentJob string        `json:"current_job,omitempty"`
	StartTime  time.Time     `json:"start_time"`
	JobsDone   int64         `json:"jobs_done"`
	JobsFailed int64         `json:"jobs_failed"`
	AvgTime    time.Duration `json:"avg_time"`
	Memory     int64         `json:"memory,omitempty"`
	CPU        float64       `json:"cpu,omitempty"`
}

// WorkerStatus worker status
type WorkerStatus string

const (
	WorkerStatusIdle    WorkerStatus = "idle"
	WorkerStatusRunning WorkerStatus = "running"
	WorkerStatusError   WorkerStatus = "error"
	WorkerStatusStopped WorkerStatus = "stopped"
)

// ProbeStats probe statistics
type ProbeStats struct {
	TotalProbes      int64         `json:"total_probes"`
	SuccessfulProbes int64         `json:"successful_probes"`
	FailedProbes     int64         `json:"failed_probes"`
	AvgRTT           time.Duration `json:"avg_rtt"`
	MinRTT           time.Duration `json:"min_rtt"`
	MaxRTT           time.Duration `json:"max_rtt"`
	SuccessRate      float64       `json:"success_rate"`
	CurrentRPS       float64       `json:"current_rps"`
	PeakRPS          float64       `json:"peak_rps"`
}

// SystemStats system statistics
type SystemStats struct {
	Timestamp   time.Time       `json:"timestamp"`
	Uptime      time.Duration   `json:"uptime"`
	Memory      MemoryStats     `json:"memory"`
	CPU         CPUStats        `json:"cpu"`
	Network     NetworkStats    `json:"network"`
	Disk        DiskStats       `json:"disk"`
	Goroutines  int             `json:"goroutines"`
	GC          GCStats         `json:"gc"`
	Connections ConnectionStats `json:"connections"`
}

// MemoryStats memory statistics
type MemoryStats struct {
	Alloc        uint64 `json:"alloc"`
	TotalAlloc   uint64 `json:"total_alloc"`
	Sys          uint64 `json:"sys"`
	Lookups      uint64 `json:"lookups"`
	Mallocs      uint64 `json:"mallocs"`
	Frees        uint64 `json:"frees"`
	HeapAlloc    uint64 `json:"heap_alloc"`
	HeapSys      uint64 `json:"heap_sys"`
	HeapIdle     uint64 `json:"heap_idle"`
	HeapInuse    uint64 `json:"heap_inuse"`
	HeapReleased uint64 `json:"heap_released"`
	StackInuse   uint64 `json:"stack_inuse"`
	StackSys     uint64 `json:"stack_sys"`
	MSpanInuse   uint64 `json:"mspan_inuse"`
	MSpanSys     uint64 `json:"mspan_sys"`
	MCacheInuse  uint64 `json:"mcache_inuse"`
	MCacheSys    uint64 `json:"mcache_sys"`
	BuckHashSys  uint64 `json:"buck_hash_sys"`
	GCSys        uint64 `json:"gc_sys"`
	OtherSys     uint64 `json:"other_sys"`
}

// CPUStats CPU statistics
type CPUStats struct {
	User    float64 `json:"user"`
	System  float64 `json:"system"`
	Idle    float64 `json:"idle"`
	Nice    float64 `json:"nice"`
	Iowait  float64 `json:"iowait"`
	Irq     float64 `json:"irq"`
	Softirq float64 `json:"softirq"`
	Steal   float64 `json:"steal"`
}

// NetworkStats network statistics
type NetworkStats struct {
	BytesSent   uint64 `json:"bytes_sent"`
	BytesRecv   uint64 `json:"bytes_recv"`
	PacketsSent uint64 `json:"packets_sent"`
	PacketsRecv uint64 `json:"packets_recv"`
	ErrorsIn    uint64 `json:"errors_in"`
	ErrorsOut   uint64 `json:"errors_out"`
	DropIn      uint64 `json:"drop_in"`
	DropOut     uint64 `json:"drop_out"`
}

// DiskStats disk statistics
type DiskStats struct {
	ReadBytes    uint64        `json:"read_bytes"`
	WriteBytes   uint64        `json:"write_bytes"`
	ReadCount    uint64        `json:"read_count"`
	WriteCount   uint64        `json:"write_count"`
	ReadTime     time.Duration `json:"read_time"`
	WriteTime    time.Duration `json:"write_time"`
	IOInProgress int64         `json:"io_in_progress"`
	IOTime       time.Duration `json:"io_time"`
}

// GCStats garbage collector statistics
type GCStats struct {
	NumGC         uint32        `json:"num_gc"`
	PauseTotal    time.Duration `json:"pause_total"`
	PauseAvg      time.Duration `json:"pause_avg"`
	PauseMin      time.Duration `json:"pause_min"`
	PauseMax      time.Duration `json:"pause_max"`
	LastGC        time.Time     `json:"last_gc"`
	GCCPUFraction float64       `json:"gc_cpu_fraction"`
}

// ConnectionStats connection statistics
type ConnectionStats struct {
	Active   int `json:"active"`
	Idle     int `json:"idle"`
	Waiting  int `json:"waiting"`
	MaxTotal int `json:"max_total"`
}

// ErrorInfo error information
type ErrorInfo struct {
	Code        string                 `json:"code"`
	Message     string                 `json:"message"`
	Cause       string                 `json:"cause,omitempty"`
	StackTrace  string                 `json:"stack_trace,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	Context     map[string]interface{} `json:"context,omitempty"`
	Recoverable bool                   `json:"recoverable"`
	Retryable   bool                   `json:"retryable"`
	RetryAfter  *time.Duration         `json:"retry_after,omitempty"`
}

// VersionInfo version information
type VersionInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"build_date"`
	GoVersion string `json:"go_version"`
}

// ConfigHash configuration hash for change detection
type ConfigHash struct {
	Hash      string    `json:"hash"`
	Timestamp time.Time `json:"timestamp"`
	Source    string    `json:"source"`
	Size      int64     `json:"size"`
}

// EventType event type
type EventType string

const (
	EventTypeProbeStart    EventType = "probe_start"
	EventTypeProbeComplete EventType = "probe_complete"
	EventTypeProbeError    EventType = "probe_error"
	EventTypeConfigReload  EventType = "config_reload"
	EventTypeSystemStart   EventType = "system_start"
	EventTypeSystemStop    EventType = "system_stop"
	EventTypeWorkerStart   EventType = "worker_start"
	EventTypeWorkerStop    EventType = "worker_stop"
	EventTypeMetricPushed  EventType = "metric_pushed"
	EventTypeWALWrite      EventType = "wal_write"
	EventTypeWALRead       EventType = "wal_read"
)

// Event system event
type Event struct {
	ID        string            `json:"id"`
	Type      EventType         `json:"type"`
	Timestamp time.Time         `json:"timestamp"`
	Source    string            `json:"source"`
	Data      interface{}       `json:"data,omitempty"`
	Labels    map[string]string `json:"labels,omitempty"`
	Level     LogLevel          `json:"level,omitempty"`
}

// LogLevel logging level
type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
	LogLevelFatal LogLevel = "fatal"
)
