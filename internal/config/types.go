package config

import (
	"time"

	"github.com/vmprober/vmprober/internal/types"
)

// Config основная конфигурация VMProber
type Config struct {
	Listen       ListenConfig       `yaml:"listen" json:"listen"`
	Pull         PullConfig         `yaml:"pull" json:"pull"`
	Push         PushConfig         `yaml:"push" json:"push"`
	Scheduler    SchedulerConfig    `yaml:"scheduler" json:"scheduler"`
	Targets      TargetsConfig      `yaml:"targets" json:"targets"`
	Probes       ProbesConfig       `yaml:"probes" json:"probes"`
	Metrics      MetricsConfig      `yaml:"metrics" json:"metrics"`
	WAL          WALConfig          `yaml:"wal" json:"wal"`
	Logging      LoggingConfig      `yaml:"logging" json:"logging"`
	TLS          TLSConfig          `yaml:"tls" json:"tls"`
	Observability ObservabilityConfig `yaml:"observability" json:"observability"`
	
	// Метаданные
	Version   string    `yaml:"-" json:"version,omitempty"`
	Source    string    `yaml:"-" json:"source,omitempty"`
	Hash      string    `yaml:"-" json:"hash,omitempty"`
	Timestamp time.Time `yaml:"-" json:"timestamp,omitempty"`
}

// ListenConfig конфигурация HTTP сервера
type ListenConfig struct {
	Port int              `yaml:"port" json:"port"`
	Host string           `yaml:"host" json:"host"`
	TLS  *TLSServerConfig `yaml:"tls,omitempty" json:"tls,omitempty"`
}

// TLSServerConfig конфигурация TLS сервера
type TLSServerConfig struct {
	Enabled    bool   `yaml:"enabled" json:"enabled"`
	CertFile   string `yaml:"cert_file" json:"cert_file"`
	KeyFile    string `yaml:"key_file" json:"key_file"`
	ClientAuth string `yaml:"client_auth" json:"client_auth"`
}

// PullConfig конфигурация pull режима
type PullConfig struct {
	Enabled bool          `yaml:"enabled" json:"enabled"`
	Path    string        `yaml:"path" json:"path"`
	Timeout time.Duration `yaml:"timeout" json:"timeout"`
}

// PushConfig конфигурация push режима
type PushConfig struct {
	Enabled     bool              `yaml:"enabled" json:"enabled"`
	Endpoints   []EndpointConfig  `yaml:"endpoints" json:"endpoints"`
	Retry       RetryConfig       `yaml:"retry" json:"retry"`
	Dedup       DedupConfig       `yaml:"dedup" json:"dedup"`
	Batch       BatchConfig       `yaml:"batch" json:"batch"`
	RemoteWrite RemoteWriteConfig `yaml:"remote_write" json:"remote_write"`
}

// EndpointConfig конфигурация endpoint
type EndpointConfig struct {
	URL     string            `yaml:"url" json:"url"`
	Headers map[string]string `yaml:"headers" json:"headers"`
	Auth    AuthConfig        `yaml:"auth" json:"auth"`
}

// AuthConfig конфигурация аутентификации
type AuthConfig struct {
	Type     string `yaml:"type" json:"type"`
	Token    string `yaml:"token" json:"token"`
	Username string `yaml:"username" json:"username"`
	Password string `yaml:"password" json:"password"`
}

// RetryConfig конфигурация retry
type RetryConfig struct {
	MaxAttempts  int           `yaml:"max_attempts" json:"max_attempts"`
	Backoff     string        `yaml:"backoff" json:"backoff"`
	InitialDelay time.Duration `yaml:"initial_delay" json:"initial_delay"`
	MaxDelay     time.Duration `yaml:"max_delay" json:"max_delay"`
	Multiplier   float64       `yaml:"multiplier" json:"multiplier"`
}

// DedupConfig конфигурация дедупликации
type DedupConfig struct {
	Enabled bool          `yaml:"enabled" json:"enabled"`
	Window  time.Duration `yaml:"window" json:"window"`
	Keys    []string      `yaml:"keys" json:"keys"`
}

// BatchConfig конфигурация batch отправки
type BatchConfig struct {
	Size    int           `yaml:"size" json:"size"`
	Timeout time.Duration `yaml:"timeout" json:"timeout"`
}

// RemoteWriteConfig конфигурация RemoteWrite
type RemoteWriteConfig struct {
	Enabled bool              `yaml:"enabled" json:"enabled"`
	URL     string            `yaml:"url" json:"url"`
	Headers map[string]string `yaml:"headers" json:"headers"`
}

// SchedulerConfig конфигурация планировщика
type SchedulerConfig struct {
	Concurrent    int                      `yaml:"concurrent" json:"concurrent"`
	RPSLimit      int                      `yaml:"rps_limit" json:"rps_limit"`
	PerHostCap    int                      `yaml:"per_host_cap" json:"per_host_cap"`
	Jitter        float64                  `yaml:"jitter" json:"jitter"`
	Timeouts      map[string]time.Duration `yaml:"timeouts" json:"timeouts"`
	QueueSize     int                      `yaml:"queue_size" json:"queue_size"`
	WorkerTimeout time.Duration            `yaml:"worker_timeout" json:"worker_timeout"`
}

// TargetsConfig конфигурация целей
type TargetsConfig struct {
	Static         []TargetConfig    `yaml:"static" json:"static"`
	Files          []FileConfig      `yaml:"files" json:"files"`
	URLs           []URLConfig       `yaml:"urls" json:"urls"`
	Commands       []CommandConfig   `yaml:"commands" json:"commands"`
	ReloadInterval time.Duration     `yaml:"reload_interval" json:"reload_interval"`
	HotReload      bool              `yaml:"hot_reload" json:"hot_reload"`
}

// TargetConfig конфигурация цели
type TargetConfig struct {
	Host     string            `yaml:"host" json:"host"`
	Port     int               `yaml:"port" json:"port"`
	Protocol types.ProbeType   `yaml:"proto" json:"proto"`
	Interval time.Duration      `yaml:"interval" json:"interval"`
	Timeout  time.Duration      `yaml:"timeout" json:"timeout"`
	Labels   map[string]string `yaml:"labels" json:"labels"`
}

// FileConfig конфигурация файлового источника
type FileConfig struct {
	Path          string        `yaml:"path" json:"path"`
	ReloadInterval time.Duration `yaml:"reload_interval" json:"reload_interval"`
	Watch         bool          `yaml:"watch" json:"watch"`
}

// URLConfig конфигурация HTTP источника
type URLConfig struct {
	URL           string            `yaml:"url" json:"url"`
	ReloadInterval time.Duration     `yaml:"reload_interval" json:"reload_interval"`
	Headers       map[string]string `yaml:"headers" json:"headers"`
}

// CommandConfig конфигурация командного источника
type CommandConfig struct {
	Command   string        `yaml:"command" json:"command"`
	Interval  time.Duration `yaml:"interval" json:"interval"`
	ParseType string        `yaml:"parse_type" json:"parse_type"`
	Filter    string        `yaml:"filter" json:"filter"`
}

// ProbesConfig конфигурация проб
type ProbesConfig struct {
	Defaults map[string]interface{} `yaml:"defaults" json:"defaults"`
	TCP      TCPProbeConfig         `yaml:"tcp" json:"tcp"`
	UDP      UDPProbeConfig         `yaml:"udp" json:"udp"`
	ICMP     ICMPProbeConfig        `yaml:"icmp" json:"icmp"`
}

// TCPProbeConfig конфигурация TCP проб
type TCPProbeConfig struct {
	ConnectTimeout time.Duration     `yaml:"connect_timeout" json:"connect_timeout"`
	TLS            types.TLSConfig   `yaml:"tls" json:"tls"`
	KeepAlive      KeepAliveConfig   `yaml:"keep_alive" json:"keep_alive"`
}

// KeepAliveConfig конфигурация keep-alive
type KeepAliveConfig struct {
	Enabled bool          `yaml:"enabled" json:"enabled"`
	Period  time.Duration `yaml:"period" json:"period"`
}

// UDPProbeConfig конфигурация UDP проб
type UDPProbeConfig struct {
	PayloadType     string        `yaml:"payload_type" json:"payload_type"`
	PayloadSize     int           `yaml:"payload_size" json:"payload_size"`
	ResponseTimeout time.Duration `yaml:"response_timeout" json:"response_timeout"`
	MaxPacketSize   int           `yaml:"max_packet_size" json:"max_packet_size"`
}

// ICMPProbeConfig конфигурация ICMP проб
type ICMPProbeConfig struct {
	Library       string `yaml:"library" json:"library"`
	SequenceStart int    `yaml:"sequence_start" json:"sequence_start"`
	TTL           int    `yaml:"ttl" json:"ttl"`
}

// MetricsConfig конфигурация метрик
type MetricsConfig struct {
	Namespace            string            `yaml:"namespace" json:"namespace"`
	IncludeLabels        []string          `yaml:"include_labels" json:"include_labels"`
	CustomLabels         map[string]string `yaml:"custom_labels" json:"custom_labels"`
	Buckets              []float64         `yaml:"buckets" json:"buckets"`
	EnableProcessMetrics bool              `yaml:"enable_process_metrics" json:"enable_process_metrics"`
	EnableGoMetrics      bool              `yaml:"enable_go_metrics" json:"enable_go_metrics"`
	EnableJobMetrics     *bool             `yaml:"enable_job_metrics" json:"enable_job_metrics"`
}

// WALConfig конфигурация WAL
type WALConfig struct {
	Dir             string        `yaml:"dir" json:"dir"`
	MaxSize         string        `yaml:"max_size" json:"max_size"`
	MaxAge          time.Duration `yaml:"max_age" json:"max_age"`
	Retention       time.Duration `yaml:"retention" json:"retention"`
	Compression     string        `yaml:"compression" json:"compression"`
	SyncInterval    time.Duration `yaml:"sync_interval" json:"sync_interval"`
	BufferSize      string        `yaml:"buffer_size" json:"buffer_size"`
	SegmentSize     string        `yaml:"segment_size" json:"segment_size"`
	IndexCacheSize  int           `yaml:"index_cache_size" json:"index_cache_size"`
}

// LoggingConfig конфигурация логирования
type LoggingConfig struct {
	Level        string            `yaml:"level" json:"level"`
	Format       string            `yaml:"format" json:"format"`
	Output       string            `yaml:"output" json:"output"`
	File         FileLoggingConfig `yaml:"file" json:"file"`
	Structured   bool              `yaml:"structured" json:"structured"`
	IncludeSource bool             `yaml:"include_source" json:"include_source"`
}

// FileLoggingConfig конфигурация файлового логирования
type FileLoggingConfig struct {
	Path       string `yaml:"path" json:"path"`
	MaxSize    string `yaml:"max_size" json:"max_size"`
	MaxBackups int    `yaml:"max_backups" json:"max_backups"`
	MaxAge     int    `yaml:"max_age" json:"max_age"`
	Compress   bool   `yaml:"compress" json:"compress"`
}

// TLSConfig конфигурация TLS
type TLSConfig struct {
	ClientCerts       ClientCertsConfig `yaml:"client_certs" json:"client_certs"`
	ServerCerts       ServerCertsConfig `yaml:"server_certs" json:"server_certs"`
	InsecureSkipVerify bool             `yaml:"insecure_skip_verify" json:"insecure_skip_verify"`
	MinVersion         string           `yaml:"min_version" json:"min_version"`
	MaxVersion         string           `yaml:"max_version" json:"max_version"`
	CipherSuites       []string         `yaml:"cipher_suites" json:"cipher_suites"`
}

// ClientCertsConfig конфигурация клиентских сертификатов
type ClientCertsConfig struct {
	Enabled  bool   `yaml:"enabled" json:"enabled"`
	CertFile string `yaml:"cert_file" json:"cert_file"`
	KeyFile  string `yaml:"key_file" json:"key_file"`
	CAFile   string `yaml:"ca_file" json:"ca_file"`
}

// ServerCertsConfig конфигурация серверных сертификатов
type ServerCertsConfig struct {
	Enabled  bool   `yaml:"enabled" json:"enabled"`
	CertFile string `yaml:"cert_file" json:"cert_file"`
	KeyFile  string `yaml:"key_file" json:"key_file"`
	CAFile   string `yaml:"ca_file" json:"ca_file"`
}

// ObservabilityConfig конфигурация наблюдаемости
type ObservabilityConfig struct {
	Pprof      PprofConfig      `yaml:"pprof" json:"pprof"`
	OpenCensus OpenCensusConfig `yaml:"opencensus" json:"opencensus"`
	Prometheus PrometheusConfig `yaml:"prometheus" json:"prometheus"`
	HealthCheck HealthCheckConfig `yaml:"health_check" json:"health_check"`
}

// PprofConfig конфигурация pprof
type PprofConfig struct {
	Enabled bool   `yaml:"enabled" json:"enabled"`
	Port    int    `yaml:"port" json:"port"`
	Host    string `yaml:"host" json:"host"`
}

// OpenCensusConfig конфигурация OpenCensus
type OpenCensusConfig struct {
	Enabled      bool                      `yaml:"enabled" json:"enabled"`
	SamplingRate float64                   `yaml:"sampling_rate" json:"sampling_rate"`
	Exporters    []OpenCensusExporterConfig `yaml:"exporters" json:"exporters"`
}

// OpenCensusExporterConfig конфигурация OpenCensus exporter
type OpenCensusExporterConfig struct {
	Type     string            `yaml:"type" json:"type"`
	Endpoint string            `yaml:"endpoint" json:"endpoint"`
	Headers  map[string]string `yaml:"headers" json:"headers"`
}

// PrometheusConfig конфигурация Prometheus метрик
type PrometheusConfig struct {
	Enabled   bool   `yaml:"enabled" json:"enabled"`
	Namespace string `yaml:"namespace" json:"namespace"`
	Subsystem string `yaml:"subsystem" json:"subsystem"`
}

// HealthCheckConfig конфигурация health check
type HealthCheckConfig struct {
	Enabled  bool          `yaml:"enabled" json:"enabled"`
	Timeout  time.Duration `yaml:"timeout" json:"timeout"`
	Interval time.Duration `yaml:"interval" json:"interval"`
}

