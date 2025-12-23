# VMProber Architecture

## Обзор системы

VMProber - это автономное Go-приложение для мониторинга доступности хостов через TCP/UDP/ICMP пробы с поддержкой как pull, так и push моделей экспорта метрик в Prometheus/VictoriaMetrics.

## Архитектурная диаграмма

```mermaid
graph TB
    subgraph "Configuration Layer"
        CFG[Config Provider]
        CFG_FILE[config.yaml]
        CFG_SOURCES[Target Sources]
        CFG_RELOAD[Hot Reload]
    end
    
    subgraph "Core Engine"
        SCHED[Scheduler]
        WORKER_POOL[Worker Pool]
        PROBE_ENGINE[Probe Engine]
    end
    
    subgraph "Probe Types"
        TCP_PROBE[TCP Connect]
        UDP_PROBE[UDP Send/Receive]
        ICMP_PROBE[ICMP Echo]
    end
    
    subgraph "Data Processing"
        NORMALIZER[Result Normalizer]
        METRICS_SYS[Metrics System]
        EVENT_BUS[Event Bus]
    end
    
    subgraph "Storage & Export"
        WAL[Write-Ahead Log]
        PUSH_ADAPTER[Push Adapter]
        VM_ENDPOINT[VictoriaMetrics]
    end
    
    subgraph "HTTP Layer"
        HTTP_SERVER[HTTP Server]
        METRICS_ENDPOINT[/metrics]
        HEALTH_ENDPOINT[/health]
        READY_ENDPOINT[/ready]
    end
    
    subgraph "Observability"
        LOGGING[Structured Logging]
        TELEMETRY[OpenTelemetry]
        PPROF[pprof]
    end
    
    %% Configuration Flow
    CFG_FILE --> CFG
    CFG_SOURCES --> CFG
    CFG_RELOAD --> CFG
    CFG --> SCHED
    
    %% Scheduling Flow
    SCHED --> WORKER_POOL
    WORKER_POOL --> PROBE_ENGINE
    
    %% Probe Execution
    PROBE_ENGINE --> TCP_PROBE
    PROBE_ENGINE --> UDP_PROBE
    PROBE_ENGINE --> ICMP_PROBE
    
    %% Data Flow
    TCP_PROBE --> NORMALIZER
    UDP_PROBE --> NORMALIZER
    ICMP_PROBE --> NORMALIZER
    
    NORMALIZER --> EVENT_BUS
    EVENT_BUS --> METRICS_SYS
    EVENT_BUS --> WAL
    
    %% Export Flow
    METRICS_SYS --> METRICS_ENDPOINT
    WAL --> PUSH_ADAPTER
    PUSH_ADAPTER --> VM_ENDPOINT
    
    %% HTTP Server
    HTTP_SERVER --> METRICS_ENDPOINT
    HTTP_SERVER --> HEALTH_ENDPOINT
    HTTP_SERVER --> READY_ENDPOINT
    
    %% Observability
    LOGGING --> HTTP_SERVER
    TELEMETRY --> HTTP_SERVER
    PPROF --> HTTP_SERVER
```

## Основные компоненты

### 1. Configuration Layer
- **Config Provider**: Загрузка и валидация конфигурации
- **Target Sources**: Статические списки, файлы, HTTP endpoints, команды
- **Hot Reload**: Динамическое обновление без перезапуска

### 2. Core Engine
- **Scheduler**: Планирование задач с jitter и лимитами
- **Worker Pool**: Управление конкурентностью и RPS лимитами
- **Probe Engine**: Выполнение различных типов проб

### 3. Probe Types
- **TCP Connect**: TCP соединения с TLS поддержкой
- **UDP Send/Receive**: UDP пакеты с эхо/случайным payload
- **ICMP Echo**: ICMP запросы через systicmp/gopacket

### 4. Data Processing
- **Result Normalizer**: Унификация результатов проб
- **Metrics System**: Генерация Prometheus метрик
- **Event Bus**: Асинхронная обработка событий

### 5. Storage & Export
- **Write-Ahead Log**: Отказоустойчивое хранение
- **Push Adapter**: Адаптер для VictoriaMetrics
- **Retry Logic**: Экспоненциальные ретраи с backoff

### 6. HTTP Layer
- **Metrics Endpoint**: `/metrics` для Prometheus scrape
- **Health Checks**: `/health` и `/ready` endpoints
- **TLS Support**: Опциональное HTTPS

### 7. Observability
- **Structured Logging**: Логирование с уровнями
- **OpenTelemetry**: Трассировка и метрики процесса
- **pprof**: Профилирование производительности

## Потоки данных

### Pull Mode (Prometheus scrape)
```
Probes → Normalizer → Metrics System → /metrics endpoint → Prometheus
```

### Push Mode (VictoriaMetrics)
```
Probes → Normalizer → WAL → Push Adapter → VictoriaMetrics API
                                    ↓
                              Retry Logic (on failure)
```

## Ключевые особенности

### Отказоустойчивость
- WAL для сохранения неотправленных метрик
- Graceful shutdown с дожиданием завершения проб
- Retry логика с экспоненциальной задержкой
- Дедупликация по ключам серий

### Производительность
- Ограниченная конкурентность воркеров
- RPS лимиты per host
- Буферизация и batch отправка
- Backpressure контроль

### Безопасность
- TLS для TCP соединений
- Системное хранилище сертификатов
- Таймауты и дедлайны
- IPv4/IPv6 поддержка

### Расширяемость
- Pluggable интерфейсы модулей
- Легкое добавление новых типов проб
- Конфигурируемые метрики и лейблы
- Модульная архитектура