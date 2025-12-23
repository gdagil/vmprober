# VMProber Module Interfaces

## Обзор интерфейсов

VMProber построен на основе pluggable архитектуры с четко определенными интерфейсами для каждого модуля. Это обеспечивает расширяемость и возможность замены компонентов.

## Основные интерфейсы

### 1. Probe Interface

```go
// pkg/interfaces/probe.go
package interfaces

import (
    "context"
    "time"
)

// ProbeType определяет тип пробы
type ProbeType string

const (
    ProbeTypeTCP  ProbeType = "tcp"
    ProbeTypeUDP  ProbeType = "udp"
    ProbeTypeICMP ProbeType = "icmp"
)

// ProbeResult представляет результат выполнения пробы
type ProbeResult struct {
    Success     bool          `json:"success"`
    RTT         time.Duration `json:"rtt"`
    Error       string        `json:"error,omitempty"`
    Attempt     int           `json:"attempt"`
    Timestamp   time.Time     `json:"timestamp"`
    SourceIP    string        `json:"source_ip,omitempty"`
    TargetIP    string        `json:"target_ip,omitempty"`
    TargetPort  int           `json:"target_port,omitempty"`
    TLS         bool          `json:"tls,omitempty"`
    Protocol    ProbeType     `json:"protocol"`
    Role        string        `json:"role,omitempty"` // client/server
    SocketFamily string       `json:"socket_family,omitempty"` // inet/inet6
    Payload     []byte        `json:"payload,omitempty"`
    Response    []byte        `json:"response,omitempty"`
}

// Target представляет цель для пробы
type Target struct {
    Host         string            `json:"host"`
    Port         int               `json:"port,omitempty"`
    Protocol     ProbeType         `json:"protocol"`
    Interval     time.Duration     `json:"interval"`
    Timeout      time.Duration     `json:"timeout"`
    Count        int               `json:"count"`
    Labels       map[string]string `json:"labels,omitempty"`
    NetworkFamily string           `json:"network_family,omitempty"` // inet/inet6/any
    TLS          *TLSConfig        `json:"tls,omitempty"`
    UDP          *UDPConfig        `json:"udp,omitempty"`
    ICMP         *ICMPConfig       `json:"icmp,omitempty"`
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
}

// UDPConfig конфигурация UDP
type UDPConfig struct {
    PayloadType   string `json:"payload_type"` // echo, random
    PayloadSize   int    `json:"payload_size"`
    ResponseTimeout time.Duration `json:"response_timeout"`
    MaxPacketSize int    `json:"max_packet_size"`
}

// ICMPConfig конфигурация ICMP
type ICMPConfig struct {
    Library       string `json:"library"` // systicmp, gopacket
    SequenceStart int    `json:"sequence_start"`
    TTL           int    `json:"ttl"`
}

// Probe интерфейс для всех типов проб
type Probe interface {
    // Execute выполняет пробу
    Execute(ctx context.Context, target Target) (*ProbeResult, error)
    
    // Type возвращает тип пробы
    Type() ProbeType
    
    // Validate проверяет конфигурацию пробы
    Validate(config interface{}) error
    
    // Close освобождает ресурсы
    Close() error
}

// ProbeFactory фабрика для создания проб
type ProbeFactory interface {
    CreateProbe(probeType ProbeType, config interface{}) (Probe, error)
    GetSupportedTypes() []ProbeType
}
```

### 2. Scheduler Interface

```go
// pkg/interfaces/scheduler.go
package interfaces

import (
    "context"
    "time"
)

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
}

// Scheduler интерфейс планировщика
type Scheduler interface {
    // Schedule добавляет задачу в планировщик
    Schedule(ctx context.Context, job Job) error
    
    // Unschedule удаляет задачу из планировщика
    Unschedule(ctx context.Context, jobID string) error
    
    // Start запускает планировщик
    Start(ctx context.Context) error
    
    // Stop останавливает планировщик
    Stop(ctx context.Context) error
    
    // GetStats возвращает статистику планировщика
    GetStats() SchedulerStats
}

// SchedulerStats статистика планировщика
type SchedulerStats struct {
    TotalJobs      int           `json:"total_jobs"`
    RunningJobs    int           `json:"running_jobs"`
    QueuedJobs     int           `json:"queued_jobs"`
    CompletedJobs  int64         `json:"completed_jobs"`
    FailedJobs     int64         `json:"failed_jobs"`
    AvgExecutionTime time.Duration `json:"avg_execution_time"`
    RPS            float64       `json:"rps"`
    QueueSize      int           `json:"queue_size"`
}

// WorkerPool интерфейс пула воркеров
type WorkerPool interface {
    // Submit отправляет задачу на выполнение
    Submit(ctx context.Context, job Job, handler JobHandler) error
    
    // Start запускает воркер пул
    Start(ctx context.Context) error
    
    // Stop останавливает воркер пул
    Stop(ctx context.Context) error
    
    // GetStats возвращает статистику воркер пула
    GetStats() WorkerPoolStats
}

// JobHandler обработчик задач
type JobHandler interface {
    Handle(ctx context.Context, job Job) error
}

// WorkerPoolStats статистика воркер пула
type WorkerPoolStats struct {
    TotalWorkers    int           `json:"total_workers"`
    ActiveWorkers   int           `json:"active_workers"`
    IdleWorkers     int           `json:"idle_workers"`
    CompletedJobs   int64         `json:"completed_jobs"`
    FailedJobs      int64         `json:"failed_jobs"`
    AvgJobTime      time.Duration `json:"avg_job_time"`
    QueueSize       int           `json:"queue_size"`
    RPS             float64       `json:"rps"`
}

// RateLimiter интерфейс rate limiter
type RateLimiter interface {
    // Allow проверяет разрешено ли выполнение
    Allow(ctx context.Context, key string) (bool, time.Duration)
    
    // SetRate устанавливает лимит rate
    SetRate(key string, rate float64, burst int) error
    
    // Remove удаляет ключ из rate limiter
    Remove(key string) error
}
```

### 3. Storage Interface

```go
// pkg/interfaces/storage.go
package interfaces

import (
    "context"
    "time"
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
}

// Storage интерфейс хранилища
type Storage interface {
    // Write записывает записи в хранилище
    Write(ctx context.Context, records []Record) error
    
    // Read читает записи из хранилища
    Read(ctx context.Context, since time.Time, limit int) ([]Record, error)
    
    // Delete удаляет записи старше указанного времени
    Delete(ctx context.Context, before time.Time) error
    
    // Close закрывает хранилище
    Close(ctx context.Context) error
    
    // GetStats возвращает статистику хранилища
    GetStats() StorageStats
}

// WAL интерфейс Write-Ahead Log
type WAL interface {
    // Write записывает записи в WAL
    Write(ctx context.Context, records []Record) error
    
    // Read читает записи из WAL
    Read(ctx context.Context, fromID string, limit int) ([]Record, error)
    
    // Truncate усекает WAL до указанного ID
    Truncate(ctx context.Context, toID string) error
    
    // Sync синхронизирует WAL с диском
    Sync(ctx context.Context) error
    
    // Close закрывает WAL
    Close(ctx context.Context) error
    
    // GetStats возвращает статистику WAL
    GetStats() WALStats
}

// StorageStats статистика хранилища
type StorageStats struct {
    TotalRecords    int64         `json:"total_records"`
    TotalSize       int64         `json:"total_size"`
    OldestRecord    time.Time     `json:"oldest_record"`
    NewestRecord    time.Time     `json:"newest_record"`
    WriteRate       float64       `json:"write_rate"`
    ReadRate        float64       `json:"read_rate"`
    CompressionRatio float64      `json:"compression_ratio"`
}

// WALStats статистика WAL
type WALStats struct {
    TotalSegments   int           `json:"total_segments"`
    ActiveSegment   string        `json:"active_segment"`
    TotalSize       int64         `json:"total_size"`
    OldestSegment   time.Time     `json:"oldest_segment"`
    WriteRate       float64       `json:"write_rate"`
    SyncLatency     time.Duration `json:"sync_latency"`
}
```

### 4. Adapter Interface

```go
// pkg/interfaces/adapter.go
package interfaces

import (
    "context"
    "time"
)

// Metric представляет метрику
type Metric struct {
    Name        string            `json:"name"`
    Value       float64           `json:"value"`
    Timestamp   time.Time         `json:"timestamp"`
    Labels      map[string]string `json:"labels"`
    Type        MetricType        `json:"type"`
}

// MetricType тип метрики
type MetricType string

const (
    MetricTypeGauge     MetricType = "gauge"
    MetricTypeCounter   MetricType = "counter"
    MetricTypeHistogram MetricType = "histogram"
    MetricTypeSummary   MetricType = "summary"
)

// Adapter интерфейс адаптера для отправки метрик
type Adapter interface {
    // Push отправляет метрики
    Push(ctx context.Context, metrics []Metric) error
    
    // Flush принудительно отправляет все буферизованные метрики
    Flush(ctx context.Context) error
    
    // Close закрывает адаптер
    Close(ctx context.Context) error
    
    // GetStats возвращает статистику адаптера
    GetStats() AdapterStats
}

// RetryConfig конфигурация retry
type RetryConfig struct {
    MaxAttempts  int           `json:"max_attempts"`
    Backoff      string        `json:"backoff"` // linear, exponential, fixed
    InitialDelay time.Duration `json:"initial_delay"`
    MaxDelay     time.Duration `json:"max_delay"`
    Multiplier   float64       `json:"multiplier"`
}

// AdapterStats статистика адаптера
type AdapterStats struct {
    TotalPushed      int64         `json:"total_pushed"`
    TotalFailed      int64         `json:"total_failed"`
    RetryCount       int64         `json:"retry_count"`
    AvgPushTime      time.Duration `json:"avg_push_time"`
    QueueSize        int           `json:"queue_size"`
    LastPushTime     time.Time     `json:"last_push_time"`
    SuccessRate      float64       `json:"success_rate"`
}

// VictoriaMetricsAdapter специализированный адаптер для VictoriaMetrics
type VictoriaMetricsAdapter interface {
    Adapter
    
    // ImportPrometheus импортирует метрики в формате Prometheus
    ImportPrometheus(ctx context.Context, data []byte) error
	
    // ImportText импортирует метрики в текстовом формате
    ImportText(ctx context.Context, data []byte) error
}

// RemoteWriteAdapter адаптер для RemoteWrite протокола
type RemoteWriteAdapter interface {
    Adapter
    
    // WriteRequest создает Remote Write запрос
    WriteRequest(ctx context.Context, metrics []Metric) ([]byte, error)
	
    // ParseResponse парсит ответ Remote Write
    ParseResponse(ctx context.Context, data []byte) error
}
```

### 5. ConfigProvider Interface

```go
// pkg/interfaces/config.go
package interfaces

import (
    "context"
    "time"
)

// ConfigProvider интерфейс провайдера конфигурации
type ConfigProvider interface {
    // Load загружает конфигурацию
    Load(ctx context.Context) (*Config, error)
	
    // Watch отслеживает изменения в конфигурации
    Watch(ctx context.Context) (<-chan ConfigUpdate, <-chan error)
	
    // Validate валидирует конфигурацию
    Validate(ctx context.Context, config *Config) error
	
    // Close закрывает провайдер
    Close(ctx context.Context) error
}

// ConfigUpdate событие обновления конфигурации
type ConfigUpdate struct {
    Type        UpdateType    `json:"type"`
    OldConfig   *Config       `json:"old_config,omitempty"`
    NewConfig   *Config       `json:"new_config,omitempty"`
    Timestamp   time.Time     `json:"timestamp"`
    Source      string        `json:"source"`
}

// UpdateType тип обновления
type UpdateType string

const (
    UpdateTypeFull    UpdateType = "full"
    UpdateTypePartial UpdateType = "partial"
    UpdateTypeError   UpdateType = "error"
)

// Config основная конфигурация VMProber
type Config struct {
    Listen       ListenConfig       `json:"listen"`
    Pull         PullConfig         `json:"pull"`
    Push         PushConfig         `json:"push"`
    Scheduler    SchedulerConfig    `json:"scheduler"`
    Targets      TargetsConfig      `json:"targets"`
    Probes       ProbesConfig       `json:"probes"`
    Metrics      MetricsConfig      `json:"metrics"`
    WAL          WALConfig          `json:"wal"`
    Logging      LoggingConfig      `json:"logging"`
    TLS          TLSConfig          `json:"tls"`
    Observability ObservabilityConfig `json:"observability"`
}

// ListenConfig конфигурация HTTP сервера
type ListenConfig struct {
    Port    int           `json:"port"`
    Host    string        `json:"host"`
    TLS     *TLSServerConfig `json:"tls,omitempty"`
}

// PullConfig конфигурация pull режима
type PullConfig struct {
    Enabled bool   `json:"enabled"`
    Path    string `json:"path"`
    Timeout time.Duration `json:"timeout"`
}

// PushConfig конфигурация push режима
type PushConfig struct {
    Enabled     bool              `json:"enabled"`
    Endpoints   []EndpointConfig  `json:"endpoints"`
    Retry       RetryConfig       `json:"retry"`
    Dedup       DedupConfig       `json:"dedup"`
    Batch       BatchConfig       `json:"batch"`
    RemoteWrite RemoteWriteConfig `json:"remote_write"`
}

// EndpointConfig конфигурация endpoint
type EndpointConfig struct {
    URL     string            `json:"url"`
    Headers map[string]string `json:"headers"`
    Auth    AuthConfig        `json:"auth"`
}

// AuthConfig конфигурация аутентификации
type AuthConfig struct {
    Type     string `json:"type"` // bearer, basic, none
    Token    string `json:"token"`
    Username string `json:"username"`
    Password string `json:"password"`
}

// DedupConfig конфигурация дедупликации
type DedupConfig struct {
    Enabled bool     `json:"enabled"`
    Window  time.Duration `json:"window"`
    Keys    []string `json:"keys"`
}

// BatchConfig конфигурация batch отправки
type BatchConfig struct {
    Size    int           `json:"size"`
    Timeout time.Duration `json:"timeout"`
}

// RemoteWriteConfig конфигурация RemoteWrite
type RemoteWriteConfig struct {
    Enabled bool              `json:"enabled"`
    URL     string            `json:"url"`
    Headers map[string]string `json:"headers"`
}

// SchedulerConfig конфигурация планировщика
type SchedulerConfig struct {
    Concurrent   int                    `json:"concurrent"`
    RPSLimit     int                    `json:"rps_limit"`
    PerHostCap   int                    `json:"per_host_cap"`
    Jitter       float64                `json:"jitter"`
    Timeouts     map[string]time.Duration `json:"timeouts"`
    QueueSize    int                    `json:"queue_size"`
    WorkerTimeout time.Duration         `json:"worker_timeout"`
}

// TargetsConfig конфигурация целей
type TargetsConfig struct {
    Static    []TargetConfig    `json:"static"`
    Files     []FileConfig      `json:"files"`
    URLs      []URLConfig       `json:"urls"`
    Commands  []CommandConfig   `json:"commands"`
    ReloadInterval time.Duration `json:"reload_interval"`
    HotReload bool              `json:"hot_reload"`
}

// TargetConfig конфигурация цели
type TargetConfig struct {
    Host      string            `json:"host"`
    Port      int               `json:"port"`
    Protocol  ProbeType         `json:"protocol"`
    Interval  time.Duration     `json:"interval"`
    Timeout   time.Duration     `json:"timeout"`
    Labels    map[string]string `json:"labels"`
}

// FileConfig конфигурация файлового источника
type FileConfig struct {
    Path          string        `json:"path"`
    ReloadInterval time.Duration `json:"reload_interval"`
    Watch         bool          `json:"watch"`
}

// URLConfig конфигурация HTTP источника
type URLConfig struct {
    URL           string            `json:"url"`
    ReloadInterval time.Duration     `json:"reload_interval"`
    Headers       map[string]string `json:"headers"`
}

// CommandConfig конфигурация командного источника
type CommandConfig struct {
    Command   string        `json:"command"`
    Interval  time.Duration `json:"interval"`
    ParseType string        `json:"parse_type"`
    Filter    string        `json:"filter"`
}

// ProbesConfig конфигурация проб
type ProbesConfig struct {
    Defaults map[string]interface{} `json:"defaults"`
    TCP      TCPConfig              `json:"tcp"`
    UDP      UDPConfig              `json:"udp"`
    ICMP     ICMPConfig             `json:"icmp"`
}

// TCPConfig конфигурация TCP проб
type TCPConfig struct {
    ConnectTimeout time.Duration `json:"connect_timeout"`
    TLS            TLSConfig     `json:"tls"`
    KeepAlive      KeepAliveConfig `json:"keep_alive"`
}

// KeepAliveConfig конфигурация keep-alive
type KeepAliveConfig struct {
    Enabled bool          `json:"enabled"`
    Period  time.Duration `json:"period"`
}

// MetricsConfig конфигурация метрик
type MetricsConfig struct {
    Namespace        string            `json:"namespace"`
    IncludeLabels    []string          `json:"include_labels"`
    CustomLabels     map[string]string `json:"custom_labels"`
    Buckets          []float64         `json:"buckets"`
    EnableProcessMetrics bool          `json:"enable_process_metrics"`
    EnableGoMetrics   bool            `json:"enable_go_metrics"`
}

// WALConfig конфигурация WAL
type WALConfig struct {
    Dir             string        `json:"dir"`
    MaxSize         string        `json:"max_size"`
    MaxAge          time.Duration `json:"max_age"`
    Retention       time.Duration `json:"retention"`
    Compression     string        `json:"compression"`
    SyncInterval    time.Duration `json:"sync_interval"`
    BufferSize      string        `json:"buffer_size"`
    SegmentSize     string        `json:"segment_size"`
    IndexCacheSize  int           `json:"index_cache_size"`
}

// LoggingConfig конфигурация логирования
type LoggingConfig struct {
    Level      string        `json:"level"`
    Format     string        `json:"format"`
    Output     string        `json:"output"`
    File       FileLoggingConfig `json:"file"`
    Structured bool          `json:"structured"`
    IncludeSource bool       `json:"include_source"`
}

// FileLoggingConfig конфигурация файлового логирования
type FileLoggingConfig struct {
    Path       string `json:"path"`
    MaxSize    string `json:"max_size"`
    MaxBackups int    `json:"max_backups"`
    MaxAge     int    `json:"max_age"`
    Compress   bool   `json:"compress"`
}

// TLSConfig конфигурация TLS
type TLSConfig struct {
    ClientCerts ClientCertsConfig `json:"client_certs"`
    ServerCerts ServerCertsConfig `json:"server_certs"`
    InsecureSkipVerify bool      `json:"insecure_skip_verify"`
    MinVersion         string    `json:"min_version"`
    MaxVersion         string    `json:"max_version"`
    CipherSuites       []string  `json:"cipher_suites"`
}

// ClientCertsConfig конфигурация клиентских сертификатов
type ClientCertsConfig struct {
    Enabled  bool   `json:"enabled"`
    CertFile string `json:"cert_file"`
    KeyFile  string `json:"key_file"`
    CAFile   string `json:"ca_file"`
}

// ServerCertsConfig конфигурация серверных сертификатов
type ServerCertsConfig struct {
    Enabled  bool   `json:"enabled"`
    CertFile string `json:"cert_file"`
    KeyFile  string `json:"key_file"`
    CAFile   string `json:"ca_file"`
}

// TLSServerConfig конфигурация TLS сервера
type TLSServerConfig struct {
    Enabled     bool   `json:"enabled"`
    CertFile    string `json:"cert_file"`
    KeyFile     string `json:"key_file"`
    ClientAuth  string `json:"client_auth"`
}

// ObservabilityConfig конфигурация наблюдаемости
type ObservabilityConfig struct {
    Pprof     PprofConfig     `json:"pprof"`
    OpenCensus OpenCensusConfig `json:"opencensus"`
    Prometheus PrometheusConfig `json:"prometheus"`
    HealthCheck HealthCheckConfig `json:"health_check"`
}

// PprofConfig конфигурация pprof
type PprofConfig struct {
    Enabled bool   `json:"enabled"`
    Port    int    `json:"port"`
    Host    string `json:"host"`
}

// OpenCensusConfig конфигурация OpenCensus
type OpenCensusConfig struct {
    Enabled       bool                    `json:"enabled"`
    SamplingRate  float64                 `json:"sampling_rate"`
    Exporters     []OpenCensusExporterConfig `json:"exporters"`
}

// OpenCensusExporterConfig конфигурация OpenCensus exporter
type OpenCensusExporterConfig struct {
    Type     string            `json:"type"`
    Endpoint string            `json:"endpoint"`
    Headers  map[string]string `json:"headers"`
}

// PrometheusConfig конфигурация Prometheus метрик
type PrometheusConfig struct {
    Enabled    bool   `json:"enabled"`
    Namespace  string `json:"namespace"`
    Subsystem  string `json:"subsystem"`
}

// HealthCheckConfig конфигурация health check
type HealthCheckConfig struct {
    Enabled  bool          `json:"enabled"`
    Timeout  time.Duration `json:"timeout"`
    Interval time.Duration `json:"interval"`
}
```

## Интерфейсы для расширения

### Normalizer Interface

```go
// pkg/interfaces/normalizer.go
package interfaces

import (
    "context"
)

// Normalizer интерфейс нормализатора результатов
type Normalizer interface {
    // Normalize нормализует результат пробы
    Normalize(ctx context.Context, result *ProbeResult) (*NormalizedEvent, error)
    
    // Dedup проверяет на дубликаты
    Dedup(ctx context.Context, event *NormalizedEvent) (bool, error)
    
    // Enrich обогащает событие дополнительной информацией
    Enrich(ctx context.Context, event *NormalizedEvent) error
}

// NormalizedEvent нормализованное событие
type NormalizedEvent struct {
    Timestamp   time.Time              `json:"timestamp"`
    SeriesID    string                 `json:"series_id"`
    Metrics     map[string]float64     `json:"metrics"`
    Labels      map[string]string      `json:"labels"`
    Tags        []string               `json:"tags"`
    Metadata    map[string]interface{} `json:"metadata"`
}
```

### Metrics Collector Interface

```go
// pkg/interfaces/metrics.go
package interfaces

import (
    "context"
)

// MetricsCollector интерфейс коллектора метрик
type MetricsCollector interface {
    // Record записывает метрику
    Record(ctx context.Context, metric Metric) error
    
    // RecordBatch записывает пакет метрик
    RecordBatch(ctx context.Context, metrics []Metric) error
    
    // GetMetrics возвращает все метрики
    GetMetrics(ctx context.Context) ([]Metric, error)
    
    // Register регистрирует коллектор
    Register(ctx context.Context) error
    
    // Unregister отменяет регистрацию коллектора
    Unregister(ctx context.Context) error
}
```

## Принципы проектирования интерфейсов

### 1. Контекстная передача
Все методы принимают `context.Context` для корректного управления жизненным циклом операций.

### 2. Обработка ошибок
Интерфейсы используют стандартные Go ошибки с возможностью расширения через специальные типы.

### 3. Статистика и мониторинг
Каждый интерфейс предоставляет метод `GetStats()` для мониторинга состояния.

### 4. Graceful shutdown
Все интерфейсы поддерживают корректное закрытие через контекст.

### 5. Расширяемость
Интерфейсы спроектированы для легкого добавления новых реализаций без изменения существующего кода.

### 6. Типобезопасность
Использование конкретных типов вместо `interface{}` где это возможно.

### 7. Производительность
Интерфейсы оптимизированы для минимального аллокации и максимальной производительности.