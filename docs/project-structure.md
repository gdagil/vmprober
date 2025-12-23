# VMProber Project Structure

## Обзор структуры проекта

VMProber организован как модульное Go-приложение с четким разделением ответственности между компонентами.

## Структура директорий

```
vmprober/
├── cmd/
│   └── vmprober/
│       └── main.go                 # Точка входа в приложение
├── internal/
│   ├── config/                     # Конфигурация и загрузка config.yaml
│   │   ├── config.go
│   │   ├── loader.go
│   │   ├── validator.go
│   │   └── types.go
│   ├── probes/                     # Реализация различных типов проб
│   │   ├── probe.go               # Базовый интерфейс Probe
│   │   ├── tcp.go                 # TCP connect пробы
│   │   ├── udp.go                 # UDP send/receive пробы
│   │   ├── icmp.go                # ICMP echo пробы
│   │   └── factory.go             # Фабрика проб
│   ├── scheduler/                  # Планировщик задач
│   │   ├── scheduler.go
│   │   ├── worker_pool.go
│   │   ├── rate_limiter.go
│   │   └── job_queue.go
│   ├── normalizer/                 # Нормализация результатов
│   │   ├── normalizer.go
│   │   ├── event.go
│   │   └── dedup.go
│   ├── metrics/                    # Система метрик Prometheus
│   │   ├── prometheus.go
│   │   ├── collector.go
│   │   ├── labels.go
│   │   └── registry.go
│   ├── storage/                    # WAL и постоянное хранилище
│   │   ├── wal.go                 # Write-Ahead Log
│   │   ├── segment.go
│   │   ├── index.go
│   │   └── compressor.go
│   ├── adapter/                    # Push адаптеры
│   │   ├── victoria_metrics.go
│   │   ├── remote_write.go
│   │   ├── http_client.go
│   │   └── retry.go
│   ├── server/                     # HTTP сервер
│   │   ├── server.go
│   │   ├── handlers.go
│   │   ├── middleware.go
│   │   └── tls.go
│   ├── logging/                    # Структурированное логирование
│   │   ├── logger.go
│   │   ├── levels.go
│   │   └── formatter.go
│   ├── telemetry/                  # Наблюдаемость
│   │   ├── tracer.go
│   │   ├── metrics.go
│   │   └── pprof.go
│   ├── targets/                    # Управление целями
│   │   ├── manager.go
│   │   ├── static.go
│   │   ├── file.go
│   │   ├── http.go
│   │   ├── command.go
│   │   └── reload.go
│   └── types/                      # Общие типы и интерфейсы
│       ├── probe_result.go
│       ├── probe_config.go
│       ├── metrics.go
│       └── errors.go
├── pkg/
│   ├── utils/                      # Утилиты
│   │   ├── time.go
│   │   ├── network.go
│   │   ├── crypto.go
│   │   └── sync.go
│   └── interfaces/                 # Публичные интерфейсы
│       ├── probe.go
│       ├── scheduler.go
│       ├── storage.go
│       └── adapter.go
├── configs/
│   ├── config.yaml.example        # Пример конфигурации
│   └── systemd/
│       └── vmprober.service       # Systemd unit файл
├── scripts/
│   ├── build.sh                   # Скрипт сборки
│   ├── test.sh                    # Скрипт тестирования
│   └── run.sh                     # Скрипт запуска
├── docs/
│   ├── api.md                     # API документация
│   ├── configuration.md           # Документация по конфигурации
│   ├── deployment.md              # Руководство по развертыванию
│   └── monitoring.md              # Документация по мониторингу
├── go.mod                         # Go модуль
├── go.sum                         # Go зависимости
├── Makefile                       # Make targets
├── Dockerfile                     # Docker образ
├── docker-compose.yml             # Docker Compose для разработки
└── README.md                      # Основная документация
```

## Основные модули

### 1. Configuration (`internal/config/`)
- Загрузка и валидация `config.yaml`
- Hot reload конфигурации
- Типы конфигурации для всех компонентов

### 2. Probes (`internal/probes/`)
- Интерфейс `Probe` для расширяемости
- Реализации TCP, UDP, ICMP проб
- Фабрика для создания проб по типу

### 3. Scheduler (`internal/scheduler/`)
- Планирование задач с jitter
- Worker pool с ограниченной конкурентностью
- Rate limiting per host

### 4. Normalizer (`internal/normalizer/`)
- Приведение результатов к единому формату
- Дедупликация событий
- Обогащение меток

### 5. Metrics (`internal/metrics/`)
- Prometheus метрики
- Gauge для состояния (true/false)
- Histogram для задержек
- Counter для ошибок

### 6. Storage (`internal/storage/`)
- Write-Ahead Log для отказоустойчивости
- Сегментированное хранение
- Компрессия и ротация

### 7. Adapter (`internal/adapter/`)
- Push в VictoriaMetrics
- RemoteWrite поддержка
- Retry логика с backoff

### 8. Server (`internal/server/`)
- HTTP сервер с `/metrics`, `/health`, `/ready`
- TLS поддержка
- Middleware для логирования и метрик

### 9. Targets (`internal/targets/`)
- Управление списком целей
- Динамическая перезагрузка
- Поддержка различных источников

### 10. Logging & Telemetry (`internal/logging/`, `internal/telemetry/`)
- Структурированное логирование
- OpenTelemetry интеграция
- pprof для профилирования

## Ключевые интерфейсы

### Probe Interface
```go
type Probe interface {
    Execute(ctx context.Context, target Target) (*ProbeResult, error)
    Type() ProbeType
    Validate(config ProbeConfig) error
}
```

### Scheduler Interface
```go
type Scheduler interface {
    Schedule(target Target, interval time.Duration)
    Start(ctx context.Context) error
    Stop() error
}
```

### Storage Interface
```go
type Storage interface {
    Write(ctx context.Context, records []Record) error
    Read(ctx context.Context, since time.Time) ([]Record, error)
    Close() error
}
```

### Adapter Interface
```go
type Adapter interface {
    Push(ctx context.Context, metrics []Metric) error
    Flush(ctx context.Context) error
    Close() error
}
```

## Принципы проектирования

### 1. Модульность
- Четкое разделение ответственности
- Минимальные зависимости между модулями
- Интерфейсы для decoupling

### 2. Расширяемость
- Pluggable архитектура
- Легкое добавление новых типов проб
- Конфигурируемые компоненты

### 3. Отказоустойчивость
- WAL для сохранения данных
- Graceful shutdown
- Retry логика

### 4. Производительность
- Ограниченная конкурентность
- Rate limiting
- Буферизация

### 5. Наблюдаемость
- Структурированное логирование
- Метрики процесса
- Трассировка