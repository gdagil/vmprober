# VMProber - Итоговая архитектурная документация

## Обзор проекта

VMProber - это автономное Go-приложение для мониторинга доступности хостов через TCP/UDP/ICMP пробы с поддержкой как pull, так и push моделей экспорта метрик в Prometheus/VictoriaMetrics. Система спроектирована с акцентом на надежность, производительность, расширяемость и comprehensive наблюдаемость.

## Ключевые особенности

### 🔍 **Мониторинг хостов**
- TCP connect пробы
- UDP send/receive пробы  
- ICMP echo пробы
- TLS поддержка для TCP
- IPv4/IPv6 поддержка
- Настраиваемые таймауты и payload

### 📊 **Экспорт метрик**
- **Pull модель**: Prometheus scrape endpoint `/metrics`
- **Push модель**: VictoriaMetrics через `/api/v1/import/prometheus` и RemoteWrite
- Comprehensive метрики: gauge, histogram, counter типы
- Единые имена и лейблы для всех метрик

### 🛡️ **Надежность**
- WAL (Write-Ahead Log) для отказоустойчивости
- Retry логика с экспоненциальной задержкой
- Circuit breaker паттерн
- Graceful shutdown с поэтапным завершением

### ⚡ **Производительность**
- Планировщик с jitter и rate limiting
- Worker pool с ограниченной конкурентностью
- Batch processing для эффективной отправки
- Connection pooling

### 🔧 **Расширяемость**
- Pluggable архитектура модулей
- Интерфейсы для всех ключевых компонентов
- Легкое добавление новых типов проб
- Hot reload конфигурации

### 📈 **Наблюдаемость**
- Structured logging с контекстом
- Comprehensive метрики процесса
- Distributed tracing (OpenTelemetry)
- Health checks и readiness probes
- Профилирование производительности

## Архитектурная диаграмма

```mermaid
graph TB
    subgraph "Configuration Layer"
        CONFIG[Config Manager]
        CONFIG_YAML[config.yaml]
        HOT_RELOAD[Hot Reload]
    end
    
    subgraph "Probe System"
        TCP_PROBE[TCP Probe]
        UDP_PROBE[UDP Probe]
        ICMP_PROBE[ICMP Probe]
        PROBE_FACTORY[Probe Factory]
    end
    
    subgraph "Scheduling System"
        SCHEDULER[Task Scheduler]
        WORKER_POOL[Worker Pool]
        RATE_LIMITER[Rate Limiter]
        PRIORITY_QUEUE[Priority Queue]
    end
    
    subgraph "Data Processing"
        NORMALIZER[Results Normalizer]
        DEDUPLICATOR[Event Deduplicator]
        ENRICHMENT[Data Enrichment]
    end
    
    subgraph "Metrics System"
        METRICS_COLLECTOR[Metrics Collector]
        PROMETHEUS_FORMATTER[Prometheus Formatter]
        LOCAL_METRICS[Local Metrics Storage]
    end
    
    subgraph "Storage & Reliability"
        WAL_SYSTEM[WAL System]
        RETRY_ENGINE[Retry Engine]
        BUFFER_MANAGER[Buffer Manager]
    end
    
    subgraph "Export Systems"
        HTTP_SERVER[HTTP Server]
        VM_ADAPTER[VictoriaMetrics Adapter]
        PULL_ENDPOINT[/metrics]
        PUSH_ENDPOINTS[Push Endpoints]
    end
    
    subgraph "Observability"
        LOG_MANAGER[Log Manager]
        TRACE_COLLECTOR[Trace Collector]
        HEALTH_MONITOR[Health Monitor]
        PROFILER[Profiler]
    end
    
    subgraph "Shutdown System"
        SIGNAL_HANDLER[Signal Handler]
        SHUTDOWN_ORCHESTRATOR[Shutdown Orchestrator]
        COMPONENT_SHUTDOWN[Component Shutdown]
    end
    
    %% Configuration Flow
    CONFIG_YAML --> CONFIG
    CONFIG --> HOT_RELOAD
    HOT_RELOAD --> SCHEDULER
    HOT_RELOAD --> PROBE_FACTORY
    
    %% Probe Flow
    PROBE_FACTORY --> TCP_PROBE
    PROBE_FACTORY --> UDP_PROBE
    PROBE_FACTORY --> ICMP_PROBE
    SCHEDULER --> WORKER_POOL
    WORKER_POOL --> TCP_PROBE
    WORKER_POOL --> UDP_PROBE
    WORKER_POOL --> ICMP_PROBE
    
    %% Data Flow
    TCP_PROBE --> NORMALIZER
    UDP_PROBE --> NORMALIZER
    ICMP_PROBE --> NORMALIZER
    NORMALIZER --> DEDUPLICATOR
    DEDUPLICATOR --> ENRICHMENT
    ENRICHMENT --> METRICS_COLLECTOR
    
    %% Metrics Flow
    METRICS_COLLECTOR --> PROMETHEUS_FORMATTER
    METRICS_COLLECTOR --> LOCAL_METRICS
    PROMETHEUS_FORMATTER --> HTTP_SERVER
    
    %% Storage Flow
    METRICS_COLLECTOR --> WAL_SYSTEM
    WAL_SYSTEM --> RETRY_ENGINE
    RETRY_ENGINE --> BUFFER_MANAGER
    BUFFER_MANAGER --> VM_ADAPTER
    
    %% Export Flow
    HTTP_SERVER --> PULL_ENDPOINT
    VM_ADAPTER --> PUSH_ENDPOINTS
    
    %% Observability Flow
    LOG_MANAGER --> TRACE_COLLECTOR
    TRACE_COLLECTOR --> HEALTH_MONITOR
    HEALTH_MONITOR --> PROFILER
    
    %% Shutdown Flow
    SIGNAL_HANDLER --> SHUTDOWN_ORCHESTRATOR
    SHUTDOWN_ORCHESTRATOR --> COMPONENT_SHUTDOWN
    COMPONENT_SHUTDOWN --> HTTP_SERVER
    COMPONENT_SHUTDOWN --> WAL_SYSTEM
    COMPONENT_SHUTDOWN --> VM_ADAPTER
```

## Структура проекта

```
vmprober/
├── cmd/
│   └── main.go                 # Точка входа приложения
├── internal/
│   ├── config/                 # Управление конфигурацией
│   ├── types/                  # Базовые типы и структуры
│   ├── probe/                  # Система проб
│   ├── scheduler/              # Планировщик задач
│   ├── normalizer/             # Нормализация результатов
│   ├── metrics/                # Система метрик
│   ├── storage/                # WAL и хранилище
│   ├── adapter/                # Push адаптеры
│   ├── server/                 # HTTP сервер
│   ├── observability/          # Логирование и трассировка
│   └── shutdown/               # Graceful shutdown
├── pkg/
│   └── interfaces/             # Интерфейсы модулей
├── configs/
│   └── config.yaml            # Пример конфигурации
├── docs/                      # Документация
│   ├── architecture.md        # Общая архитектура
│   ├── project-structure.md   # Структура проекта
│   ├── configuration.md       # Конфигурация
│   ├── module-interfaces.md   # Интерфейсы модулей
│   ├── data-structures.md     # Структуры данных
│   ├── configuration-module.md # Модуль конфигурации
│   ├── probe-system.md        # Система проб
│   ├── scheduler-system.md    # Планировщик
│   ├── normalizer-system.md   # Нормализатор
│   ├── prometheus-metrics.md  # Система метрик
│   ├── http-server.md         # HTTP сервер
│   ├── wal-system.md          # WAL система
│   ├── victoriametrics-adapter.md # VictoriaMetrics адаптер
│   ├── observability-system.md # Наблюдаемость
│   └── graceful-shutdown.md   # Graceful shutdown
├── scripts/                   # Скрипты сборки и развертывания
├── tests/                     # Тесты
└── Makefile                   # Сборочные цели
```

## Созданные компоненты

### 1. **Архитектура и планирование**
- ✅ [Архитектурная диаграмма](architecture.md) - Общий обзор системы
- ✅ [Структура проекта](project-structure.md) - Организация кода и модулей
- ✅ [Интерфейсы модулей](module-interfaces.md) - Pluggable архитектура

### 2. **Конфигурация и данные**
- ✅ [Конфигурационная структура](configuration.md) - config.yaml спецификация
- ✅ [Базовые структуры данных](data-structures.md) - Core типы и структуры
- ✅ [Модуль конфигурации](configuration-module.md) - Управление конфигурацией

### 3. **Система мониторинга**
- ✅ [Система проб](probe-system.md) - TCP/UDP/ICMP пробы
- ✅ [Планировщик задач](scheduler-system.md) - Jitter, rate limiting, worker pool
- ✅ [Нормализатор результатов](normalizer-system.md) - Обработка и обогащение данных

### 4. **Метрики и экспорт**
- ✅ [Система метрик Prometheus](prometheus-metrics.md) - Сбор и экспорт метрик
- ✅ [HTTP сервер](http-server.md) - /metrics, /health, /ready endpoints
- ✅ [WAL система](wal-system.md) - Отказоустойчивое хранение
- ✅ [VictoriaMetrics адаптер](victoriametrics-adapter.md) - Push экспорт

### 5. **Наблюдаемость и управление**
- ✅ [Система наблюдаемости](observability-system.md) - Логирование, трассировка, профилирование
- ✅ [Graceful shutdown](graceful-shutdown.md) - Корректное завершение работы

## Ключевые интерфейсы

### Probe Interface
```go
type Probe interface {
    Execute(ctx context.Context, target Target) (*ProbeResult, error)
    Validate(ctx context.Context, config ProbeConfig) error
    GetType() ProbeType
    GetSupportedFeatures() []Feature
}
```

### Scheduler Interface
```go
type Scheduler interface {
    Schedule(ctx context.Context, task *Task) error
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    GetStats() *SchedulerStats
}
```

### MetricsCollector Interface
```go
type MetricsCollector interface {
    Record(ctx context.Context, metric *Metric) error
    GetMetrics(ctx context.Context) ([]prometheus.Metric, error)
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
}
```

### Storage Interface
```go
type Storage interface {
    Write(ctx context.Context, record *Record) error
    Read(ctx context.Context, filter Filter) ([]*Record, error)
    Flush(ctx context.Context) error
    Close(ctx context.Context) error
}
```

## Конфигурация

### Основная структура config.yaml
```yaml
# Основные настройки
app:
  name: "vmprober"
  version: "1.0.0"
  environment: "production"

# HTTP сервер
listen:
  addr: "0.0.0.0"
  port: 8429
  read_timeout: 30s
  write_timeout: 30s
  tls_enabled: false

# Pull модель (Prometheus)
pull:
  enabled: true
  endpoint: "/metrics"

# Push модель (VictoriaMetrics)
push:
  enabled: true
  endpoints:
    - url: "http://victoriametrics:8428/api/v1/import/prometheus"
      priority: 1
  batch:
    max_size: 1000
    max_wait_time: 10s
  retry:
    max_attempts: 3
    initial_delay: 1s

# Планировщик
scheduler:
  concurrent: 100
  rps_limit: 1000.0
  per_host_cap: 10
  jitter: true
  timeouts:
    tcp: 5s
    udp: 5s
    icmp: 3s

# Цели мониторинга
targets:
  static:
    - host: "google.com"
      port: 80
      proto: "tcp"
      interval: 30s
    - host: "8.8.8.8"
      proto: "icmp"
      interval: 60s
  files:
    - pattern: "/etc/vmprober/targets/*.yaml"
      reload_interval: 300s

# Настройки проб
probes:
  defaults:
    count: 3
    interval: 1s
    timeout: 5s
    payload_size: 64
  tcp:
    connect_timeout: 5s
    tls_enabled: false
  udp:
    payload_type: "random"
    expect_response: true
  icmp:
    packet_size: 64
    ttl: 64

# Метрики
metrics:
  namespace: "vmprober"
  include_labels:
    - "job"
    - "instance"
    - "probe"
    - "target"
    - "proto"
  buckets: [0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0]

# WAL система
wal:
  directory: "/var/lib/vmprober/wal"
  max_segment_size: 100MB
  max_segment_age: 1h
  compression_enabled: true
  retry:
    max_attempts: 5
    initial_delay: 1s
    max_delay: 300s

# Наблюдаемость
observability:
  logging:
    level: "info"
    format: "json"
    output: "stdout"
  metrics:
    enabled: true
    namespace: "vmprober"
  tracing:
    enabled: false
  health:
    enabled: true

# Graceful shutdown
shutdown:
  enabled: true
  timeout:
    graceful: 30s
    force: 10s
  signals:
    signals:
      - "SIGTERM"
      - "SIGINT"
```

## Метрики

### Основные метрики VMProber

#### Probe Metrics
- `vmprober_probe_success_total` - Общее количество успешных проб
- `vmprober_probe_failure_total` - Общее количество неудачных проб
- `vmprober_probe_rtt_seconds` - Время отклика проб (histogram)
- `vmprober_probe_attempts_total` - Общее количество попыток
- `vmprober_probe_last_success_timestamp` - Время последней успешной пробы

#### System Metrics
- `vmprober_process_memory_usage_bytes` - Использование памяти процессом
- `vmprober_process_cpu_usage_percent` - Использование CPU процессом
- `vmprober_process_goroutines` - Количество горутин
- `vmprober_http_requests_total` - HTTP запросы к серверу
- `vmprober_http_request_duration_seconds` - Время обработки HTTP запросов

#### Storage Metrics
- `vmprober_wal_records_total` - Общее количество записей в WAL
- `vmprober_wal_disk_usage_bytes` - Использование диска WAL
- `vmprober_retry_attempts_total` - Общее количество попыток ретрая
- `vmprober_victoriametrics_push_total` - Отправки в VictoriaMetrics

## Развертывание

### Docker
```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o vmprober cmd/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/vmprober .
COPY configs/config.yaml .
CMD ["./vmprober"]
```

### Kubernetes
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: vmprober
spec:
  replicas: 1
  selector:
    matchLabels:
      app: vmprober
  template:
    metadata:
      labels:
        app: vmprober
    spec:
      containers:
      - name: vmprober
        image: vmprober:latest
        ports:
        - containerPort: 8429
        volumeMounts:
        - name: config
          mountPath: /etc/vmprober
        - name: wal
          mountPath: /var/lib/vmprober/wal
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8429
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8429
          initialDelaySeconds: 5
          periodSeconds: 5
      volumes:
      - name: config
        configMap:
          name: vmprober-config
      - name: wal
        persistentVolumeClaim:
          claimName: vmprober-wal
```

### Systemd Service
```ini
[Unit]
Description=VMProber - Host Monitoring Service
After=network.target

[Service]
Type=simple
User=vmprober
Group=vmprober
ExecStart=/usr/local/bin/vmprober --config=/etc/vmprober/config.yaml
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=vmprober

# Security settings
NoNewPrivileges=yes
PrivateTmp=yes
ProtectSystem=strict
ProtectHome=yes
ReadWritePaths=/var/lib/vmprober /var/log/vmprober

[Install]
WantedBy=multi-user.target
```

## Мониторинг и алерты

### Grafana Dashboard
Создайте dashboard в Grafana с основными панелями:
- Общий статус проб (success/failure rate)
- Время отклика по типам проб
- Использование ресурсов VMProber
- Статус WAL системы
- Статус push в VictoriaMetrics

### Alerting Rules
```yaml
groups:
- name: vmprober
  rules:
  - alert: HighProbeFailureRate
    expr: rate(vmprober_probe_failure_total[5m]) > 0.1
    for: 2m
    labels:
      severity: warning
    annotations:
      summary: "High probe failure rate detected"
      
  - alert: HighRTT
    expr: vmprober_probe_rtt_seconds{quantile="0.95"} > 1
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "High round-trip time detected"
      
  - alert: VMProberDown
    expr: up{job="vmprober"} == 0
    for: 1m
    labels:
      severity: critical
    annotations:
      summary: "VMProber is down"
```

## Производительность

### Рекомендуемые настройки

#### Для небольших развертываний (< 1000 целей)
```yaml
scheduler:
  concurrent: 50
  rps_limit: 500.0
  per_host_cap: 5

wal:
  max_segment_size: 50MB
  max_segment_age: 30m

push:
  batch:
    max_size: 500
    max_wait_time: 5s
```

#### Для средних развертываний (1000-10000 целей)
```yaml
scheduler:
  concurrent: 200
  rps_limit: 2000.0
  per_host_cap: 10

wal:
  max_segment_size: 200MB
  max_segment_age: 1h

push:
  batch:
    max_size: 2000
    max_wait_time: 10s
```

#### Для крупных развертываний (> 10000 целей)
```yaml
scheduler:
  concurrent: 500
  rps_limit: 5000.0
  per_host_cap: 20

wal:
  max_segment_size: 500MB
  max_segment_age: 2h

push:
  batch:
    max_size: 5000
    max_wait_time: 15s
```

## Безопасность

### Рекомендации по безопасности
1. **TLS**: Включите TLS для HTTP сервера в production
2. **Аутентификация**: Используйте bearer token или API key для защиты endpoints
3. **Сетевая изоляция**: Ограничьте доступ к VMProber через firewall
4. **Мониторинг доступа**: Логируйте все запросы к /metrics endpoint
5. **Обновления**: Регулярно обновляйте зависимости и base images

### Пример безопасной конфигурации
```yaml
listen:
  addr: "127.0.0.1"  # Только локальный доступ
  port: 8429
  tls_enabled: true
  tls_cert_file: "/etc/ssl/certs/vmprober.crt"
  tls_key_file: "/etc/ssl/private/vmprober.key"

auth:
  enabled: true
  type: "bearer"
  token_file: "/etc/vmprober/tokens.txt"

rate_limit:
  enabled: true
  rate: 100.0
  burst: 200
```

## Тестирование

### Unit Tests
```bash
go test ./internal/... -v
```

### Integration Tests
```bash
go test ./tests/integration/... -v
```

### Load Tests
```bash
# Тестирование с большим количеством целей
go test ./tests/load/... -v -count=1
```

### Chaos Engineering
```bash
# Тестирование отказоустойчивости
go test ./tests/chaos/... -v
```

## Устранение неисправностей

### Частые проблемы

#### Высокое использование памяти
- Уменьшите `scheduler.concurrent`
- Уменьшите `wal.max_segment_size`
- Включите компрессию WAL

#### Высокий CPU usage
- Уменьшите `scheduler.rps_limit`
- Оптимизируйте интервалы проб
- Проверьте настройки ICMP проб

#### Проблемы с сетью
- Проверьте firewall настройки
- Увеличьте timeout для медленных целей
- Используйте connection pooling

#### Проблемы с VictoriaMetrics
- Проверьте доступность endpoints
- Увеличьте retry attempts
- Проверьте аутентификацию

### Логи и отладка
```bash
# Включение debug логов
vmprober --log-level=debug

# Профилирование
curl http://localhost:8429/debug/pprof/

# Health check
curl http://localhost:8429/health

# Metrics
curl http://localhost:8429/metrics
```

## Заключение

VMProber представляет собой comprehensive решение для мониторинга доступности хостов с поддержкой современных практик DevOps:

- **Надежность**: WAL, retry логика, graceful shutdown
- **Производительность**: Оптимизированный планировщик и batch processing
- **Расширяемость**: Pluggable архитектура для легкого добавления функций
- **Наблюдаемость**: Comprehensive логирование, метрики и трассировка
- **Гибкость**: Поддержка как pull, так и push моделей экспорта

Система готова к production развертыванию и может масштабироваться от небольших до крупных инфраструктур.

## Ссылки на документацию

- [Архитектура](architecture.md)
- [Структура проекта](project-structure.md)
- [Конфигурация](configuration.md)
- [Интерфейсы модулей](module-interfaces.md)
- [Структуры данных](data-structures.md)
- [Модуль конфигурации](configuration-module.md)
- [Система проб](probe-system.md)
- [Планировщик](scheduler-system.md)
- [Нормализатор](normalizer-system.md)
- [Метрики Prometheus](prometheus-metrics.md)
- [HTTP сервер](http-server.md)
- [WAL система](wal-system.md)
- [VictoriaMetrics адаптер](victoriametrics-adapter.md)
- [Наблюдаемость](observability-system.md)
- [Graceful shutdown](graceful-shutdown.md)