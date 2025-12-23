// Package types содержит базовые структуры данных и типы для VMProber
package types

import (
	"time"
)

// ProbeType определяет тип пробы
type ProbeType string

const (
	ProbeTypeTCP  ProbeType = "tcp"
	ProbeTypeUDP  ProbeType = "udp"
	ProbeTypeICMP ProbeType = "icmp"
)

// NetworkFamily определяет семейство сетевых адресов
type NetworkFamily string

const (
	NetworkFamilyInet  NetworkFamily = "inet"  // IPv4
	NetworkFamilyInet6 NetworkFamily = "inet6" // IPv6
	NetworkFamilyAny   NetworkFamily = "any"   // Любое
)

// ProbeResult представляет результат выполнения пробы
type ProbeResult struct {
	Success      bool          `json:"success"`
	RTT          time.Duration `json:"rtt"`
	Error        string        `json:"error,omitempty"`
	Attempt      int           `json:"attempt"`
	Timestamp    time.Time     `json:"timestamp"`
	SourceIP     string        `json:"source_ip,omitempty"`
	TargetIP     string        `json:"target_ip,omitempty"`
	TargetPort   int           `json:"target_port,omitempty"`
	TLS          bool          `json:"tls,omitempty"`
	Protocol     ProbeType     `json:"protocol"`
	Role         string        `json:"role,omitempty"` // client/server
	SocketFamily string        `json:"socket_family,omitempty"`
	Payload      []byte        `json:"payload,omitempty"`
	Response     []byte        `json:"response,omitempty"`
	DNSResult    *DNSResult    `json:"dns_result,omitempty"`
}

// DNSResult результат DNS разрешения
type DNSResult struct {
	ResolvedIPs []string       `json:"resolved_ips"`
	LookupTime  time.Duration  `json:"lookup_time"`
	TTL         time.Duration  `json:"ttl,omitempty"`
	Error       string         `json:"error,omitempty"`
}

// Target представляет цель для пробы
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
	Enabled       bool              `json:"enabled"`
	Priority      int               `json:"priority,omitempty"`
	CreatedAt     time.Time         `json:"created_at,omitempty"`
	UpdatedAt     time.Time         `json:"updated_at,omitempty"`
}

// TLSConfig конфигурация TLS
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

// UDPConfig конфигурация UDP
type UDPConfig struct {
	PayloadType     string        `json:"payload_type"` // echo, random
	PayloadSize     int           `json:"payload_size"`
	ResponseTimeout time.Duration `json:"response_timeout"`
	MaxPacketSize   int           `json:"max_packet_size"`
	BindAddress     string        `json:"bind_address,omitempty"`
}

// ICMPConfig конфигурация ICMP
type ICMPConfig struct {
	Library       string `json:"library"` // systicmp, gopacket
	SequenceStart int    `json:"sequence_start"`
	TTL           int    `json:"ttl"`
	Data          []byte `json:"data,omitempty"`
}

// Job представляет задачу для выполнения пробы
type Job struct {
	ID          string        `json:"id"`
	Target      Target        `json:"target"`
	NextRun     time.Time     `json:"next_run"`
	Interval    time.Duration `json:"interval"`
	Jitter      float64       `json:"jitter"`
	RetryCount  int           `json:"retry_count"`
	MaxRetries  int           `json:"max_retries"`
	Priority    int           `json:"priority"`
	CreatedAt   time.Time     `json:"created_at"`
	Attempt     int           `json:"attempt"`
}

// NormalizedEvent нормализованное событие
type NormalizedEvent struct {
	Timestamp   time.Time              `json:"timestamp"`
	SeriesID    string                 `json:"series_id"`
	Metrics     map[string]float64     `json:"metrics"`
	Labels      map[string]string      `json:"labels"`
	Tags        []string               `json:"tags"`
	Metadata    map[string]interface{} `json:"metadata"`
	Source      string                 `json:"source,omitempty"`
}

// Metric представляет метрику
type Metric struct {
	Name        string            `json:"name"`
	Value       float64           `json:"value"`
	Timestamp   time.Time         `json:"timestamp"`
	Labels      map[string]string `json:"labels"`
	Type        MetricType        `json:"type"`
	Help        string            `json:"help,omitempty"`
	Buckets     []float64         `json:"buckets,omitempty"`
	Sum         float64           `json:"sum,omitempty"`
	Count       uint64            `json:"count,omitempty"`
	Quantiles   map[float64]float64 `json:"quantiles,omitempty"`
}

// MetricType тип метрики
type MetricType string

const (
	MetricTypeGauge     MetricType = "gauge"
	MetricTypeCounter   MetricType = "counter"
	MetricTypeHistogram MetricType = "histogram"
	MetricTypeSummary   MetricType = "summary"
)

// Record представляет запись в хранилище
type Record struct {
	ID          string                 `json:"id"`
	Timestamp   time.Time              `json:"timestamp"`
	Type        string                 `json:"type"`
	Data        map[string]interface{} `json:"data"`
	Labels      map[string]string      `json:"labels,omitempty"`
	SeriesID    string                 `json:"series_id"`
	Compression string                 `json:"compression,omitempty"`
	Size        int64                  `json:"size,omitempty"`
}

// ConfigUpdate событие обновления конфигурации
// Config определен в internal/config/types.go
type ConfigUpdate struct {
	Type        UpdateType    `json:"type"`
	OldConfig   interface{}   `json:"old_config,omitempty"`
	NewConfig   interface{}   `json:"new_config,omitempty"`
	Timestamp   time.Time     `json:"timestamp"`
	Source      string        `json:"source"`
	Changes     []ConfigChange `json:"changes,omitempty"`
}

// UpdateType тип обновления
type UpdateType string

const (
	UpdateTypeFull    UpdateType = "full"
	UpdateTypePartial UpdateType = "partial"
	UpdateTypeError   UpdateType = "error"
)

// ConfigChange изменение в конфигурации
type ConfigChange struct {
	Path    string      `json:"path"`
	OldValue interface{} `json:"old_value"`
	NewValue interface{} `json:"new_value"`
	Op      ChangeOp    `json:"op"`
}

// ChangeOp операция изменения
type ChangeOp string

const (
	ChangeOpAdd    ChangeOp = "add"
	ChangeOpUpdate ChangeOp = "update"
	ChangeOpDelete ChangeOp = "delete"
)

// HealthStatus статус здоровья системы
type HealthStatus struct {
	Status      string                 `json:"status"`
	Timestamp   time.Time              `json:"timestamp"`
	Uptime      time.Duration          `json:"uptime"`
	Version     string                 `json:"version"`
	Checks      map[string]HealthCheck `json:"checks"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// HealthCheck проверка здоровья компонента
type HealthCheck struct {
	Status    string        `json:"status"`
	Message   string        `json:"message,omitempty"`
	Timestamp time.Time     `json:"timestamp"`
	Duration  time.Duration `json:"duration,omitempty"`
	Details   interface{}   `json:"details,omitempty"`
}

// ReadyStatus статус готовности системы
type ReadyStatus struct {
	Ready      bool                   `json:"ready"`
	Timestamp  time.Time              `json:"timestamp"`
	Components map[string]ReadyCheck  `json:"components"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// ReadyCheck проверка готовности компонента
type ReadyCheck struct {
	Ready     bool        `json:"ready"`
	Message   string      `json:"message,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
	Details   interface{} `json:"details,omitempty"`
}

// RateLimitInfo информация о rate limiting
type RateLimitInfo struct {
	Key        string        `json:"key"`
	Rate       float64       `json:"rate"`
	Burst      int           `json:"burst"`
	Remaining  int           `json:"remaining"`
	ResetTime  time.Time     `json:"reset_time"`
	RetryAfter time.Duration `json:"retry_after,omitempty"`
}

// WorkerInfo информация о воркере
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

// WorkerStatus статус воркера
type WorkerStatus string

const (
	WorkerStatusIdle    WorkerStatus = "idle"
	WorkerStatusRunning WorkerStatus = "running"
	WorkerStatusError   WorkerStatus = "error"
	WorkerStatusStopped WorkerStatus = "stopped"
)

// ProbeStats статистика проб
type ProbeStats struct {
	TotalProbes     int64         `json:"total_probes"`
	SuccessfulProbes int64        `json:"successful_probes"`
	FailedProbes    int64         `json:"failed_probes"`
	AvgRTT          time.Duration `json:"avg_rtt"`
	MinRTT          time.Duration `json:"min_rtt"`
	MaxRTT          time.Duration `json:"max_rtt"`
	SuccessRate     float64       `json:"success_rate"`
	CurrentRPS      float64       `json:"current_rps"`
	PeakRPS         float64       `json:"peak_rps"`
}

// SystemStats системная статистика
type SystemStats struct {
	Timestamp    time.Time       `json:"timestamp"`
	Uptime       time.Duration   `json:"uptime"`
	Memory       MemoryStats     `json:"memory"`
	CPU          CPUStats        `json:"cpu"`
	Network      NetworkStats    `json:"network"`
	Disk         DiskStats       `json:"disk"`
	Goroutines   int             `json:"goroutines"`
	GC           GCStats         `json:"gc"`
	Connections  ConnectionStats `json:"connections"`
}

// MemoryStats статистика памяти
type MemoryStats struct {
	Alloc       uint64 `json:"alloc"`
	TotalAlloc  uint64 `json:"total_alloc"`
	Sys         uint64 `json:"sys"`
	Lookups     uint64 `json:"lookups"`
	Mallocs     uint64 `json:"mallocs"`
	Frees       uint64 `json:"frees"`
	HeapAlloc   uint64 `json:"heap_alloc"`
	HeapSys     uint64 `json:"heap_sys"`
	HeapIdle    uint64 `json:"heap_idle"`
	HeapInuse   uint64 `json:"heap_inuse"`
	HeapReleased uint64 `json:"heap_released"`
	StackInuse  uint64 `json:"stack_inuse"`
	StackSys    uint64 `json:"stack_sys"`
	MSpanInuse  uint64 `json:"mspan_inuse"`
	MSpanSys    uint64 `json:"mspan_sys"`
	MCacheInuse uint64 `json:"mcache_inuse"`
	MCacheSys   uint64 `json:"mcache_sys"`
	BuckHashSys uint64 `json:"buck_hash_sys"`
	GCSys       uint64 `json:"gc_sys"`
	OtherSys    uint64 `json:"other_sys"`
}

// CPUStats статистика CPU
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

// NetworkStats сетевая статистика
type NetworkStats struct {
	BytesSent     uint64 `json:"bytes_sent"`
	BytesRecv     uint64 `json:"bytes_recv"`
	PacketsSent   uint64 `json:"packets_sent"`
	PacketsRecv   uint64 `json:"packets_recv"`
	ErrorsIn      uint64 `json:"errors_in"`
	ErrorsOut     uint64 `json:"errors_out"`
	DropIn        uint64 `json:"drop_in"`
	DropOut       uint64 `json:"drop_out"`
}

// DiskStats дисковая статистика
type DiskStats struct {
	ReadBytes    uint64         `json:"read_bytes"`
	WriteBytes   uint64         `json:"write_bytes"`
	ReadCount    uint64         `json:"read_count"`
	WriteCount   uint64         `json:"write_count"`
	ReadTime     time.Duration  `json:"read_time"`
	WriteTime    time.Duration  `json:"write_time"`
	IOInProgress int64          `json:"io_in_progress"`
	IOTime       time.Duration  `json:"io_time"`
}

// GCStats статистика сборщика мусора
type GCStats struct {
	NumGC         uint32        `json:"num_gc"`
	PauseTotal    time.Duration `json:"pause_total"`
	PauseAvg      time.Duration `json:"pause_avg"`
	PauseMin      time.Duration `json:"pause_min"`
	PauseMax      time.Duration `json:"pause_max"`
	LastGC        time.Time     `json:"last_gc"`
	GCCPUFraction float64       `json:"gc_cpu_fraction"`
}

// ConnectionStats статистика соединений
type ConnectionStats struct {
	Active   int `json:"active"`
	Idle     int `json:"idle"`
	Waiting  int `json:"waiting"`
	MaxTotal int `json:"max_total"`
}

// ErrorInfo информация об ошибке
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

// VersionInfo информация о версии
type VersionInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"build_date"`
	GoVersion string `json:"go_version"`
}

// ConfigHash хеш конфигурации для обнаружения изменений
type ConfigHash struct {
	Hash       string    `json:"hash"`
	Timestamp  time.Time `json:"timestamp"`
	Source     string    `json:"source"`
	Size       int64     `json:"size"`
}

// EventType тип события
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

// Event событие системы
type Event struct {
	ID        string     `json:"id"`
	Type      EventType  `json:"type"`
	Timestamp time.Time  `json:"timestamp"`
	Source    string     `json:"source"`
	Data      interface{} `json:"data,omitempty"`
	Labels    map[string]string `json:"labels,omitempty"`
	Level     LogLevel   `json:"level,omitempty"`
}

// LogLevel уровень логирования
type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
	LogLevelFatal LogLevel = "fatal"
)
