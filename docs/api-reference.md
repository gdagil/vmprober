# API Reference

VMProber предоставляет HTTP API для мониторинга метрик и проверки состояния системы.

## Базовый URL

По умолчанию: `http://localhost:8429`

## Endpoints

### GET /health

Проверка здоровья приложения.

**Response:**

```json
{
  "status": "healthy",
  "uptime": "1h23m45s"
}
```

**Status Codes:**
- `200 OK` - Приложение работает

### GET /ready

Проверка готовности приложения к обработке запросов.

**Response:**

```json
{
  "ready": true
}
```

**Status Codes:**
- `200 OK` - Приложение готово
- `503 Service Unavailable` - Приложение не готово

### GET /metrics

Экспорт метрик в формате Prometheus.

**Response:**

```
# HELP vmprober_probe_success_total Total number of successful probes
# TYPE vmprober_probe_success_total counter
vmprober_probe_success_total{protocol="tcp",target="example.com:80"} 1234

# HELP vmprober_probe_failure_total Total number of failed probes
# TYPE vmprober_probe_failure_total counter
vmprober_probe_failure_total{protocol="tcp",target="example.com:80"} 5

# HELP vmprober_probe_rtt_seconds Probe round-trip time in seconds
# TYPE vmprober_probe_rtt_seconds histogram
vmprober_probe_rtt_seconds_bucket{protocol="tcp",target="example.com:80",le="0.001"} 100
vmprober_probe_rtt_seconds_bucket{protocol="tcp",target="example.com:80",le="0.005"} 500
...
vmprober_probe_rtt_seconds_sum{protocol="tcp",target="example.com:80"} 12.34
vmprober_probe_rtt_seconds_count{protocol="tcp",target="example.com:80"} 1239

# HELP vmprober_probe_attempts_total Total number of probe attempts
# TYPE vmprober_probe_attempts_total counter
vmprober_probe_attempts_total{protocol="tcp",target="example.com:80"} 1239

# HELP vmprober_probe_last_success_timestamp Timestamp of last successful probe
# TYPE vmprober_probe_last_success_timestamp gauge
vmprober_probe_last_success_timestamp{protocol="tcp",target="example.com:80"} 1699123456.789
```

**Status Codes:**
- `200 OK` - Метрики успешно экспортированы

**Content-Type:** `text/plain; version=0.0.4; charset=utf-8`

## Метрики

### vmprober_probe_success_total

Тип: `counter`

Описание: Общее количество успешных проб.

Лейблы:
- `target` - Целевой хост и порт (например, "example.com:80")
- `protocol` - Протокол пробы ("tcp", "udp", "icmp")

### vmprober_probe_failure_total

Тип: `counter`

Описание: Общее количество неудачных проб.

Лейблы:
- `target` - Целевой хост и порт
- `protocol` - Протокол пробы

### vmprober_probe_rtt_seconds

Тип: `histogram`

Описание: Время отклика пробы в секундах (только для успешных проб).

Лейблы:
- `target` - Целевой хост и порт
- `protocol` - Протокол пробы

Buckets: `[0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0]`

### vmprober_probe_attempts_total

Тип: `counter`

Описание: Общее количество попыток проб (успешных и неудачных).

Лейблы:
- `target` - Целевой хост и порт
- `protocol` - Протокол пробы

### vmprober_probe_last_success_timestamp

Тип: `gauge`

Описание: Unix timestamp последней успешной пробы.

Лейблы:
- `target` - Целевой хост и порт
- `protocol` - Протокол пробы

## Примеры использования

### Проверка здоровья

```bash
curl http://localhost:8429/health
```

### Проверка готовности

```bash
curl http://localhost:8429/ready
```

### Получение метрик

```bash
curl http://localhost:8429/metrics
```

### Фильтрация метрик

```bash
curl http://localhost:8429/metrics | grep vmprober_probe_success_total
```

### Использование с Prometheus

```yaml
scrape_configs:
  - job_name: 'vmprober'
    scrape_interval: 30s
    static_configs:
      - targets: ['localhost:8429']
    metrics_path: '/metrics'
```

### Использование с VictoriaMetrics

```bash
curl http://localhost:8429/metrics | \
  curl -X POST \
    -H "Content-Type: text/plain" \
    --data-binary @- \
    http://victoria-metrics:8428/api/v1/import/prometheus
```

## Ошибки

### 404 Not Found

Возвращается для неизвестных endpoints.

### 503 Service Unavailable

Возвращается `/ready` когда приложение не готово к обработке запросов.

## Rate Limiting

В настоящее время rate limiting не применяется к HTTP endpoints. Это может быть изменено в будущих версиях.

## TLS

Если включен TLS (см. конфигурацию `listen.tls.enabled`), используйте HTTPS:

```bash
curl https://localhost:8429/health
```

## См. также

- [Метрики](metrics-reference.md) - Подробное описание метрик
- [Конфигурация](configuration.md) - Настройка HTTP сервера
- [HTTP сервер](http-server.md) - Внутренняя реализация

