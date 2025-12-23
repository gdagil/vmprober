# Конфигурация VMProber

VMProber использует YAML конфигурацию для настройки всех аспектов работы приложения.

## Структура конфигурации

Основной файл конфигурации (`config.yaml`) содержит следующие секции:

- `listen` - Настройки HTTP сервера
- `pull` - Pull режим (Prometheus scrape)
- `push` - Push режим (VictoriaMetrics)
- `scheduler` - Настройки планировщика задач
- `targets` - Список целей для мониторинга
- `probes` - Настройки проб по умолчанию
- `metrics` - Настройки метрик
- `wal` - Настройки Write-Ahead Log
- `logging` - Настройки логирования
- `tls` - Настройки TLS
- `observability` - Настройки наблюдаемости

## HTTP сервер (`listen`)

```yaml
listen:
  port: 8429
  host: "0.0.0.0"
  tls:
    enabled: false
    cert_file: "/path/to/cert.pem"
    key_file: "/path/to/key.pem"
    client_auth: "none"  # none, optional, required
```

- `port` - Порт для прослушивания (1-65535)
- `host` - IP адрес для привязки (по умолчанию: "0.0.0.0")
- `tls.enabled` - Включить TLS (по умолчанию: false)

## Pull режим (`pull`)

Настройки для Prometheus scrape:

```yaml
pull:
  enabled: true
  path: "/metrics"
  timeout: 10s
```

- `enabled` - Включить pull режим (по умолчанию: true)
- `path` - Путь к метрикам (по умолчанию: "/metrics")
- `timeout` - Таймаут для scrape запросов

## Push режим (`push`)

Настройки для отправки метрик в VictoriaMetrics:

```yaml
push:
  enabled: false
  endpoints:
    - url: "http://victoria-metrics:8428/api/v1/import/prometheus"
      headers: {}
      auth:
        type: "none"  # none, basic, bearer, token
        username: ""
        password: ""
        token: ""
  retry:
    max_attempts: 5
    backoff: "exponential"  # exponential, linear, constant
    initial_delay: 1s
    max_delay: 60s
    multiplier: 2.0
  dedup:
    enabled: true
    window: 5m
    keys: ["job", "instance", "probe", "target", "proto"]
  batch:
    size: 1000
    timeout: 30s
  remote_write:
    enabled: false
    url: ""
    headers: {}
```

### Retry конфигурация

- `max_attempts` - Максимальное количество попыток
- `backoff` - Стратегия задержки: `exponential`, `linear`, `constant`
- `initial_delay` - Начальная задержка
- `max_delay` - Максимальная задержка
- `multiplier` - Множитель для exponential backoff

### Deduplication

- `enabled` - Включить дедупликацию
- `window` - Окно времени для дедупликации
- `keys` - Ключи для группировки метрик

## Планировщик (`scheduler`)

```yaml
scheduler:
  concurrent: 100        # Количество одновременных проб
  rps_limit: 1000       # Лимит запросов в секунду
  per_host_cap: 10      # Максимум проб на хост
  jitter: 0.1           # Jitter для распределения нагрузки (0.0-1.0)
  timeouts:
    tcp: 5s
    udp: 3s
    icmp: 2s
  queue_size: 10000     # Размер очереди задач
  worker_timeout: 30s   # Таймаут для воркера
```

- `concurrent` - Количество одновременных проб
- `rps_limit` - Глобальный лимит RPS
- `per_host_cap` - Максимум проб на один хост
- `jitter` - Случайная задержка для распределения нагрузки (0.0-1.0)
- `timeouts` - Таймауты по типам протоколов
- `queue_size` - Размер очереди задач

## Цели (`targets`)

### Статические цели

```yaml
targets:
  static:
    - host: "google.com"
      port: 80
      proto: "tcp"
      interval: 30s
      timeout: 5s
      labels:
        region: "us-east"
        service: "web"
    - host: "8.8.8.8"
      port: 53
      proto: "udp"
      interval: 60s
      timeout: 3s
      labels:
        service: "dns"
```

### Файловые источники

```yaml
targets:
  files:
    - path: "/etc/vmprober/targets.yaml"
      reload_interval: 1m
      watch: true
```

### HTTP источники

```yaml
targets:
  urls:
    - url: "http://discovery-service:8429/targets"
      reload_interval: 5m
      headers:
        Authorization: "Bearer token"
```

### Командные источники

```yaml
targets:
  commands:
    - command: "/usr/local/bin/get-targets.sh"
      interval: 10m
      parse_type: "yaml"  # yaml, json, prometheus
      filter: ""
```

### Общие настройки

```yaml
targets:
  reload_interval: 1m    # Интервал перезагрузки целей
  hot_reload: true       # Hot reload конфигурации
```

## Настройки проб (`probes`)

### Значения по умолчанию

```yaml
probes:
  defaults:
    count: 3              # Количество попыток
    interval: 30s         # Интервал между попытками
    timeout: 5s           # Таймаут пробы
    payload_size: 64      # Размер payload
```

### TCP пробы

```yaml
probes:
  tcp:
    connect_timeout: 5s
    tls:
      enabled: false
      insecure_skip_verify: false
      server_name: ""
      min_version: "1.2"
      max_version: "1.3"
    keep_alive:
      enabled: true
      period: 30s
```

### UDP пробы

```yaml
probes:
  udp:
    payload_type: "random"  # random, echo, custom
    payload_size: 64
    response_timeout: 2s
    max_packet_size: 1024
```

### ICMP пробы

```yaml
probes:
  icmp:
    library: "systicmp"  # systicmp, gopacket
    sequence_start: 1
    ttl: 64
```

## Метрики (`metrics`)

```yaml
metrics:
  namespace: "vmprober"
  include_labels:
    - "job"
    - "instance"
    - "probe"
    - "target"
    - "proto"
  custom_labels:
    environment: "production"
    region: "us-east"
  buckets: [0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0]
  enable_process_metrics: true
  enable_go_metrics: true
```

- `namespace` - Префикс для всех метрик
- `include_labels` - Лейблы для включения в метрики
- `custom_labels` - Дополнительные статические лейблы
- `buckets` - Buckets для histogram метрик
- `enable_process_metrics` - Включить системные метрики процесса
- `enable_go_metrics` - Включить Go runtime метрики

## WAL (`wal`)

```yaml
wal:
  dir: "/var/lib/vmprober/wal"
  max_size: "1GB"
  max_age: 168h
  retention: 30d
  compression: "gzip"  # none, gzip, snappy
  sync_interval: 1s
  buffer_size: "64KB"
  segment_size: "64MB"
  index_cache_size: 1000
```

## Логирование (`logging`)

```yaml
logging:
  level: "info"          # debug, info, warn, error
  format: "json"         # json, text
  output: "stdout"       # stdout, stderr, file
  file:
    path: "/var/log/vmprober.log"
    max_size: "100MB"
    max_backups: 10
    max_age: 30
    compress: true
  structured: true
  include_source: true
```

## TLS (`tls`)

```yaml
tls:
  client_certs:
    enabled: false
    cert_file: ""
    key_file: ""
    ca_file: ""
  server_certs:
    enabled: false
    cert_file: ""
    key_file: ""
    ca_file: ""
  insecure_skip_verify: false
  min_version: "1.2"
  max_version: "1.3"
  cipher_suites: []
```

## Наблюдаемость (`observability`)

```yaml
observability:
  pprof:
    enabled: true
    port: 6060
    host: "127.0.0.1"
  opencensus:
    enabled: false
    sampling_rate: 0.1
    exporters: []
  prometheus:
    enabled: true
    namespace: "vmprober"
    subsystem: "process"
  health_check:
    enabled: true
    timeout: 5s
    interval: 30s
```

## Hot Reload

VMProber поддерживает hot reload конфигурации. При изменении файла конфигурации:

1. Файл проверяется каждые 5 секунд
2. При обнаружении изменений конфигурация перезагружается
3. Подписчики уведомляются об изменениях
4. Новые цели добавляются, удаленные - удаляются

Для включения:

```yaml
targets:
  hot_reload: true
```

## Валидация

Конфигурация валидируется при загрузке:

- Проверка портов (1-65535)
- Проверка обязательных полей
- Проверка типов данных
- Проверка диапазонов значений

При ошибке валидации приложение не запустится.

## Примеры

См. `config.yaml.example` в корне проекта для полного примера конфигурации.

## См. также

- [Примеры использования](examples.md)
- [Модуль конфигурации](configuration-module.md)
