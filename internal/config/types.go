package config

import (
	"time"

	"github.com/gdagil/vmprober/internal/types"
)

// Config is the main VMProber configuration
type Config struct {
	Listen        ListenConfig        `yaml:"listen" json:"listen"`
	Pull          PullConfig          `yaml:"pull" json:"pull"`
	Push          PushConfig          `yaml:"push" json:"push"`
	Scheduler     SchedulerConfig     `yaml:"scheduler" json:"scheduler"`
	Targets       TargetsConfig       `yaml:"targets" json:"targets"`
	Probes        ProbesConfig        `yaml:"probes" json:"probes"`
	Metrics       MetricsConfig       `yaml:"metrics" json:"metrics"`
	WAL           WALConfig           `yaml:"wal" json:"wal"`
	Logging       LoggingConfig       `yaml:"logging" json:"logging"`
	TLS           TLSConfig           `yaml:"tls" json:"tls"`
	Observability ObservabilityConfig `yaml:"observability" json:"observability"`

	// Metadata
	Version   string    `yaml:"-" json:"version,omitempty"`
	Source    string    `yaml:"-" json:"source,omitempty"`
	Hash      string    `yaml:"-" json:"hash,omitempty"`
	Timestamp time.Time `yaml:"-" json:"timestamp,omitempty"`
}

// ListenConfig is the HTTP server configuration
type ListenConfig struct {
	Port int              `yaml:"port" json:"port"`
	Host string           `yaml:"host" json:"host"`
	TLS  *TLSServerConfig `yaml:"tls,omitempty" json:"tls,omitempty"`
}

// TLSServerConfig is the TLS server configuration
type TLSServerConfig struct {
	Enabled    bool   `yaml:"enabled" json:"enabled"`
	CertFile   string `yaml:"cert_file" json:"cert_file"`
	KeyFile    string `yaml:"key_file" json:"key_file"`
	ClientAuth string `yaml:"client_auth" json:"client_auth"`
}

// PullConfig is the pull mode configuration
type PullConfig struct {
	Enabled bool          `yaml:"enabled" json:"enabled"`
	Path    string        `yaml:"path" json:"path"`
	Timeout time.Duration `yaml:"timeout" json:"timeout"`
}

// PushConfig is the push mode configuration
type PushConfig struct {
	Enabled     bool              `yaml:"enabled" json:"enabled"`
	Endpoints   []EndpointConfig  `yaml:"endpoints" json:"endpoints"`
	Retry       RetryConfig       `yaml:"retry" json:"retry"`
	Dedup       DedupConfig       `yaml:"dedup" json:"dedup"`
	Batch       BatchConfig       `yaml:"batch" json:"batch"`
	RemoteWrite RemoteWriteConfig `yaml:"remote_write" json:"remote_write"`
}

// EndpointConfig is the endpoint configuration
type EndpointConfig struct {
	URL     string            `yaml:"url" json:"url"`
	Headers map[string]string `yaml:"headers" json:"headers"`
	Auth    AuthConfig        `yaml:"auth" json:"auth"`
}

// AuthConfig is the authentication configuration
type AuthConfig struct {
	Type     string `yaml:"type" json:"type"`
	Token    string `yaml:"token" json:"token"`
	Username string `yaml:"username" json:"username"`
	Password string `yaml:"password" json:"password"`
}

// RetryConfig is the retry configuration
type RetryConfig struct {
	MaxAttempts  int           `yaml:"max_attempts" json:"max_attempts"`
	Backoff      string        `yaml:"backoff" json:"backoff"`
	InitialDelay time.Duration `yaml:"initial_delay" json:"initial_delay"`
	MaxDelay     time.Duration `yaml:"max_delay" json:"max_delay"`
	Multiplier   float64       `yaml:"multiplier" json:"multiplier"`
}

// DedupConfig is the deduplication configuration
type DedupConfig struct {
	Enabled bool          `yaml:"enabled" json:"enabled"`
	Window  time.Duration `yaml:"window" json:"window"`
	Keys    []string      `yaml:"keys" json:"keys"`
}

// BatchConfig is the batch sending configuration
type BatchConfig struct {
	Size    int           `yaml:"size" json:"size"`
	Timeout time.Duration `yaml:"timeout" json:"timeout"`
}

// RemoteWriteConfig is the RemoteWrite configuration
type RemoteWriteConfig struct {
	Enabled bool              `yaml:"enabled" json:"enabled"`
	URL     string            `yaml:"url" json:"url"`
	Headers map[string]string `yaml:"headers" json:"headers"`
}

// SchedulerConfig is the scheduler configuration
type SchedulerConfig struct {
	Concurrent    int                      `yaml:"concurrent" json:"concurrent"`
	RPSLimit      int                      `yaml:"rps_limit" json:"rps_limit"`
	PerHostCap    int                      `yaml:"per_host_cap" json:"per_host_cap"`
	Jitter        float64                  `yaml:"jitter" json:"jitter"`
	Timeouts      map[string]time.Duration `yaml:"timeouts" json:"timeouts"`
	QueueSize     int                      `yaml:"queue_size" json:"queue_size"`
	WorkerTimeout time.Duration            `yaml:"worker_timeout" json:"worker_timeout"`
}

// TargetsConfig is the targets configuration
type TargetsConfig struct {
	Static         []TargetConfig  `yaml:"static" json:"static"`
	Files          []FileConfig    `yaml:"files" json:"files"`
	URLs           []URLConfig     `yaml:"urls" json:"urls"`
	Commands       []CommandConfig `yaml:"commands" json:"commands"`
	ReloadInterval time.Duration   `yaml:"reload_interval" json:"reload_interval"`
	HotReload      bool            `yaml:"hot_reload" json:"hot_reload"`
}

// TargetConfig is the target configuration
type TargetConfig struct {
	Host      string            `yaml:"host" json:"host"`
	Port      int               `yaml:"port" json:"port"`
	Protocols []string          `yaml:"protocols" json:"protocols"` // List of protocols as strings
	Interval  time.Duration     `yaml:"interval" json:"interval"`
	Timeout   time.Duration     `yaml:"timeout" json:"timeout"`
	Labels    map[string]string `yaml:"labels" json:"labels"`
}

// UnmarshalYAML custom parser for correct handling of protocols array and proto field
func (tc *TargetConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	aux := &struct {
		Host      string            `yaml:"host"`
		Port      int               `yaml:"port"`
		Proto     string            `yaml:"proto"`     // Single protocol (for backward compatibility)
		Protocols []string          `yaml:"protocols"` // Array of protocols
		Interval  time.Duration     `yaml:"interval"`
		Timeout   time.Duration     `yaml:"timeout"`
		Labels    map[string]string `yaml:"labels"`
	}{}

	if err := unmarshal(aux); err != nil {
		return err
	}

	// Copy fields
	tc.Host = aux.Host
	tc.Port = aux.Port
	tc.Interval = aux.Interval
	tc.Timeout = aux.Timeout
	tc.Labels = aux.Labels

	// Priority: protocols > proto
	if len(aux.Protocols) > 0 {
		tc.Protocols = aux.Protocols
	} else if aux.Proto != "" {
		tc.Protocols = []string{aux.Proto}
	}

	return nil
}

// GetProtocols returns the list of protocols for the target
func (tc *TargetConfig) GetProtocols() []types.ProbeType {
	if len(tc.Protocols) > 0 {
		result := make([]types.ProbeType, 0, len(tc.Protocols))
		for _, p := range tc.Protocols {
			result = append(result, types.ProbeType(p))
		}
		return result
	}
	// Default to TCP if protocols not specified
	return []types.ProbeType{types.ProbeTypeTCP}
}

// FileConfig is the file source configuration
type FileConfig struct {
	Path           string        `yaml:"path" json:"path"`
	ReloadInterval time.Duration `yaml:"reload_interval" json:"reload_interval"`
	Watch          bool          `yaml:"watch" json:"watch"`
}

// URLConfig is the HTTP source configuration
type URLConfig struct {
	URL            string            `yaml:"url" json:"url"`
	ReloadInterval time.Duration     `yaml:"reload_interval" json:"reload_interval"`
	Headers        map[string]string `yaml:"headers" json:"headers"`
}

// CommandConfig is the command source configuration
type CommandConfig struct {
	Command   string        `yaml:"command" json:"command"`
	Interval  time.Duration `yaml:"interval" json:"interval"`
	ParseType string        `yaml:"parse_type" json:"parse_type"`
	Filter    string        `yaml:"filter" json:"filter"`
}

// ProbesConfig is the probes configuration
type ProbesConfig struct {
	Defaults map[string]interface{} `yaml:"defaults" json:"defaults"`
	TCP      TCPProbeConfig         `yaml:"tcp" json:"tcp"`
	UDP      UDPProbeConfig         `yaml:"udp" json:"udp"`
	ICMP     ICMPProbeConfig        `yaml:"icmp" json:"icmp"`
	HTTP     HTTPProbeConfig        `yaml:"http" json:"http"`
	DNS      DNSProbeConfig         `yaml:"dns" json:"dns"`
	GRPC     GRPCProbeConfig        `yaml:"grpc" json:"grpc"`
}

// TCPProbeConfig is the TCP probes configuration
type TCPProbeConfig struct {
	ConnectTimeout time.Duration   `yaml:"connect_timeout" json:"connect_timeout"`
	TLS            types.TLSConfig `yaml:"tls" json:"tls"`
	KeepAlive      KeepAliveConfig `yaml:"keep_alive" json:"keep_alive"`
}

// KeepAliveConfig is the keep-alive configuration
type KeepAliveConfig struct {
	Enabled bool          `yaml:"enabled" json:"enabled"`
	Period  time.Duration `yaml:"period" json:"period"`
}

// UDPProbeConfig is the UDP probes configuration
type UDPProbeConfig struct {
	PayloadType     string        `yaml:"payload_type" json:"payload_type"`
	PayloadSize     int           `yaml:"payload_size" json:"payload_size"`
	ResponseTimeout time.Duration `yaml:"response_timeout" json:"response_timeout"`
	MaxPacketSize   int           `yaml:"max_packet_size" json:"max_packet_size"`
}

// ICMPProbeConfig is the ICMP probes configuration
type ICMPProbeConfig struct {
	Library       string `yaml:"library" json:"library"`
	SequenceStart int    `yaml:"sequence_start" json:"sequence_start"`
	TTL           int    `yaml:"ttl" json:"ttl"`
}

// HTTPProbeConfig is the HTTP/HTTPS probes configuration
type HTTPProbeConfig struct {
	Method             string            `yaml:"method" json:"method"`                             // GET, POST, HEAD, PUT, etc.
	Path               string            `yaml:"path" json:"path"`                                 // URL path (default: "/")
	Headers            map[string]string `yaml:"headers" json:"headers"`                           // Custom headers
	Body               string            `yaml:"body" json:"body"`                                 // Request body for POST/PUT
	ExpectedStatusCode int               `yaml:"expected_status_code" json:"expected_status_code"` // Expected HTTP status (default: 200)
	ExpectedBody       string            `yaml:"expected_body" json:"expected_body"`               // Expected response body substring
	FollowRedirects    bool              `yaml:"follow_redirects" json:"follow_redirects"`         // Follow HTTP redirects (default: true)
	MaxRedirects       int               `yaml:"max_redirects" json:"max_redirects"`               // Maximum redirects to follow (default: 10)
	ValidateCert       bool              `yaml:"validate_cert" json:"validate_cert"`               // Validate TLS certificate (default: true)
	BasicAuth          *BasicAuthConfig  `yaml:"basic_auth,omitempty" json:"basic_auth"`           // Basic authentication
	BearerToken        string            `yaml:"bearer_token,omitempty" json:"bearer_token"`       // Bearer token for Authorization header
	Proxy              string            `yaml:"proxy,omitempty" json:"proxy"`                     // HTTP proxy URL
	Timeout            time.Duration     `yaml:"timeout" json:"timeout"`                           // Request timeout
}

// BasicAuthConfig is the Basic Auth configuration
type BasicAuthConfig struct {
	Username string `yaml:"username" json:"username"`
	Password string `yaml:"password" json:"password"`
}

// DNSProbeConfig is the DNS probes configuration
type DNSProbeConfig struct {
	QueryName       string        `yaml:"query_name" json:"query_name"`             // Domain to query
	QueryType       string        `yaml:"query_type" json:"query_type"`             // A, AAAA, CNAME, MX, TXT, NS, SOA, SRV, PTR (default: A)
	Protocol        string        `yaml:"protocol" json:"protocol"`                 // udp, tcp, tcp-tls/dot (default: udp)
	ExpectedRecords []string      `yaml:"expected_records" json:"expected_records"` // Expected records in response
	ValidateAnswer  bool          `yaml:"validate_answer" json:"validate_answer"`   // Validate RCODE is NOERROR (default: true)
	Recursion       bool          `yaml:"recursion" json:"recursion"`               // Request recursive resolution (default: true)
	Timeout         time.Duration `yaml:"timeout" json:"timeout"`                   // Query timeout
}

// GRPCProbeConfig is the gRPC probes configuration
type GRPCProbeConfig struct {
	Service         string            `yaml:"service" json:"service"`                   // Service name for health check (empty = overall health)
	TLS             bool              `yaml:"tls" json:"tls"`                           // Use TLS
	TLSSkipVerify   bool              `yaml:"tls_skip_verify" json:"tls_skip_verify"`   // Skip TLS certificate verification
	ServerName      string            `yaml:"server_name" json:"server_name"`           // TLS server name
	Metadata        map[string]string `yaml:"metadata" json:"metadata"`                 // gRPC metadata (headers)
	ExpectedStatus  string            `yaml:"expected_status" json:"expected_status"`   // Expected health status (SERVING, NOT_SERVING, UNKNOWN)
	ReflectionProbe bool              `yaml:"reflection_probe" json:"reflection_probe"` // Use gRPC reflection instead of health check
	Timeout         time.Duration     `yaml:"timeout" json:"timeout"`                   // Request timeout
}

// MetricsConfig is the metrics configuration
type MetricsConfig struct {
	Namespace            string            `yaml:"namespace" json:"namespace"`
	IncludeLabels        []string          `yaml:"include_labels" json:"include_labels"`
	CustomLabels         map[string]string `yaml:"custom_labels" json:"custom_labels"`
	Buckets              []float64         `yaml:"buckets" json:"buckets"`
	EnableProcessMetrics bool              `yaml:"enable_process_metrics" json:"enable_process_metrics"`
	EnableGoMetrics      bool              `yaml:"enable_go_metrics" json:"enable_go_metrics"`
	EnableJobMetrics     *bool             `yaml:"enable_job_metrics" json:"enable_job_metrics"`
}

// WALConfig is the WAL configuration
type WALConfig struct {
	Dir            string        `yaml:"dir" json:"dir"`
	MaxSize        string        `yaml:"max_size" json:"max_size"`
	MaxAge         time.Duration `yaml:"max_age" json:"max_age"`
	Retention      time.Duration `yaml:"retention" json:"retention"`
	Compression    string        `yaml:"compression" json:"compression"`
	SyncInterval   time.Duration `yaml:"sync_interval" json:"sync_interval"`
	BufferSize     string        `yaml:"buffer_size" json:"buffer_size"`
	SegmentSize    string        `yaml:"segment_size" json:"segment_size"`
	IndexCacheSize int           `yaml:"index_cache_size" json:"index_cache_size"`
}

// LoggingConfig is the logging configuration
type LoggingConfig struct {
	Level         string            `yaml:"level" json:"level"`
	Format        string            `yaml:"format" json:"format"`
	Output        string            `yaml:"output" json:"output"`
	File          FileLoggingConfig `yaml:"file" json:"file"`
	Structured    bool              `yaml:"structured" json:"structured"`
	IncludeSource bool              `yaml:"include_source" json:"include_source"`
}

// FileLoggingConfig is the file logging configuration
type FileLoggingConfig struct {
	Path       string `yaml:"path" json:"path"`
	MaxSize    string `yaml:"max_size" json:"max_size"`
	MaxBackups int    `yaml:"max_backups" json:"max_backups"`
	MaxAge     int    `yaml:"max_age" json:"max_age"`
	Compress   bool   `yaml:"compress" json:"compress"`
}

// TLSConfig is the TLS configuration
type TLSConfig struct {
	ClientCerts        ClientCertsConfig `yaml:"client_certs" json:"client_certs"`
	ServerCerts        ServerCertsConfig `yaml:"server_certs" json:"server_certs"`
	InsecureSkipVerify bool              `yaml:"insecure_skip_verify" json:"insecure_skip_verify"`
	MinVersion         string            `yaml:"min_version" json:"min_version"`
	MaxVersion         string            `yaml:"max_version" json:"max_version"`
	CipherSuites       []string          `yaml:"cipher_suites" json:"cipher_suites"`
}

// ClientCertsConfig is the client certificates configuration
type ClientCertsConfig struct {
	Enabled  bool   `yaml:"enabled" json:"enabled"`
	CertFile string `yaml:"cert_file" json:"cert_file"`
	KeyFile  string `yaml:"key_file" json:"key_file"`
	CAFile   string `yaml:"ca_file" json:"ca_file"`
}

// ServerCertsConfig is the server certificates configuration
type ServerCertsConfig struct {
	Enabled  bool   `yaml:"enabled" json:"enabled"`
	CertFile string `yaml:"cert_file" json:"cert_file"`
	KeyFile  string `yaml:"key_file" json:"key_file"`
	CAFile   string `yaml:"ca_file" json:"ca_file"`
}

// ObservabilityConfig is the observability configuration
type ObservabilityConfig struct {
	Pprof       PprofConfig       `yaml:"pprof" json:"pprof"`
	OpenCensus  OpenCensusConfig  `yaml:"opencensus" json:"opencensus"`
	Prometheus  PrometheusConfig  `yaml:"prometheus" json:"prometheus"`
	HealthCheck HealthCheckConfig `yaml:"health_check" json:"health_check"`
}

// PprofConfig is the pprof configuration
type PprofConfig struct {
	Enabled bool   `yaml:"enabled" json:"enabled"`
	Port    int    `yaml:"port" json:"port"`
	Host    string `yaml:"host" json:"host"`
}

// OpenCensusConfig is the OpenCensus configuration
type OpenCensusConfig struct {
	Enabled      bool                       `yaml:"enabled" json:"enabled"`
	SamplingRate float64                    `yaml:"sampling_rate" json:"sampling_rate"`
	Exporters    []OpenCensusExporterConfig `yaml:"exporters" json:"exporters"`
}

// OpenCensusExporterConfig is the OpenCensus exporter configuration
type OpenCensusExporterConfig struct {
	Type     string            `yaml:"type" json:"type"`
	Endpoint string            `yaml:"endpoint" json:"endpoint"`
	Headers  map[string]string `yaml:"headers" json:"headers"`
}

// PrometheusConfig is the Prometheus metrics configuration
type PrometheusConfig struct {
	Enabled   bool   `yaml:"enabled" json:"enabled"`
	Namespace string `yaml:"namespace" json:"namespace"`
	Subsystem string `yaml:"subsystem" json:"subsystem"`
}

// HealthCheckConfig is the health check configuration
type HealthCheckConfig struct {
	Enabled  bool          `yaml:"enabled" json:"enabled"`
	Timeout  time.Duration `yaml:"timeout" json:"timeout"`
	Interval time.Duration `yaml:"interval" json:"interval"`
}
