# Справочник метрик

Полное описание всех метрик, экспортируемых VMProber.

## Обзор

VMProber экспортирует метрики в формате Prometheus через endpoint `/metrics`. Все метрики имеют префикс `vmprober` (настраивается через `metrics.namespace`).

## Метрики проб

### vmprober_probe_success_total

**Тип:** `counter`

**Описание:** Общее количество успешных проб.

**Лейблы:**
- `target` (string) - Целевой хост и порт в формате "host:port" или "host" для ICMP
- `protocol` (string) - Протокол пробы: "tcp", "udp", "icmp"

**Пример:**
```
vmprober_probe_success_total{protocol="tcp",target="example.com:80"} 1234
```

### vmprober_probe_failure_total

**Тип:** `counter`

**Описание:** Общее количество неудачных проб.

**Лейблы:**
- `target` (string) - Целевой хост и порт
- `protocol` (string) - Протокол пробы

**Пример:**
```
vmprober_probe_failure_total{protocol="tcp",target="example.com:80"} 5
```

### vmprober_probe_rtt_seconds

**Тип:** `histogram`

**Описание:** Время отклика пробы в секундах. Измеряется только для успешных проб.

**Лейблы:**
- `target` (string) - Целевой хост и порт
- `protocol` (string) - Протокол пробы

**Buckets:** `[0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0]`

**Пример:**
```
vmprober_probe_rtt_seconds_bucket{protocol="tcp",target="example.com:80",le="0.001"} 100
vmprober_probe_rtt_seconds_bucket{protocol="tcp",target="example.com:80",le="0.005"} 500
vmprober_probe_rtt_seconds_bucket{protocol="tcp",target="example.com:80",le="0.01"} 800
...
vmprober_probe_rtt_seconds_bucket{protocol="tcp",target="example.com:80",le="+Inf"} 1234
vmprober_probe_rtt_seconds_sum{protocol="tcp",target="example.com:80"} 12.34
vmprober_probe_rtt_seconds_count{protocol="tcp",target="example.com:80"} 1234
```

### vmprober_probe_attempts_total

**Тип:** `counter`

**Описание:** Общее количество попыток проб (успешных и неудачных).

**Лейблы:**
- `target` (string) - Целевой хост и порт
- `protocol` (string) - Протокол пробы

**Пример:**
```
vmprober_probe_attempts_total{protocol="tcp",target="example.com:80"} 1239
```

### vmprober_probe_last_success_timestamp

**Тип:** `gauge`

**Описание:** Unix timestamp (в секундах) последней успешной пробы.

**Лейблы:**
- `target` (string) - Целевой хост и порт
- `protocol` (string) - Протокол пробы

**Пример:**
```
vmprober_probe_last_success_timestamp{protocol="tcp",target="example.com:80"} 1699123456.789
```

## Дополнительные метрики

Если включены `enable_process_metrics` и `enable_go_metrics`, также экспортируются:

### Процесс метрики

- `process_cpu_seconds_total` - Общее время CPU процесса
- `process_resident_memory_bytes` - Использование памяти
- `process_open_fds` - Открытые файловые дескрипторы
- `process_start_time_seconds` - Время запуска процесса

### Go runtime метрики

- `go_goroutines` - Количество goroutines
- `go_memstats_alloc_bytes` - Выделенная память
- `go_memstats_sys_bytes` - Системная память
- `go_gc_duration_seconds` - Длительность GC

## Использование метрик

### Расчет success rate

```promql
rate(vmprober_probe_success_total[5m]) / rate(vmprober_probe_attempts_total[5m])
```

### Расчет failure rate

```promql
rate(vmprober_probe_failure_total[5m]) / rate(vmprober_probe_attempts_total[5m])
```

### Средний RTT

```promql
rate(vmprober_probe_rtt_seconds_sum[5m]) / rate(vmprober_probe_rtt_seconds_count[5m])
```

### 95-й перцентиль RTT

```promql
histogram_quantile(0.95, rate(vmprober_probe_rtt_seconds_bucket[5m]))
```

### Время с последней успешной пробы

```promql
time() - vmprober_probe_last_success_timestamp
```

### Количество активных целей

```promql
count(vmprober_probe_attempts_total) by (protocol)
```

## Примеры запросов Prometheus

### Топ-10 целей по количеству ошибок

```promql
topk(10, rate(vmprober_probe_failure_total[5m]))
```

### Цели с высоким RTT

```promql
histogram_quantile(0.95, rate(vmprober_probe_rtt_seconds_bucket[5m])) > 1
```

### Цели без успешных проб за последний час

```promql
(time() - vmprober_probe_last_success_timestamp) > 3600
```

### Success rate по протоколам

```promql
sum(rate(vmprober_probe_success_total[5m])) by (protocol) / 
sum(rate(vmprober_probe_attempts_total[5m])) by (protocol)
```

## Алерты

### Высокий failure rate

```yaml
- alert: HighProbeFailureRate
  expr: |
    rate(vmprober_probe_failure_total[5m]) / 
    rate(vmprober_probe_attempts_total[5m]) > 0.1
  for: 5m
  annotations:
    summary: "High probe failure rate for {{ $labels.target }}"
```

### Высокий RTT

```yaml
- alert: HighProbeLatency
  expr: |
    histogram_quantile(0.95, 
      rate(vmprober_probe_rtt_seconds_bucket[5m])
    ) > 1
  for: 5m
  annotations:
    summary: "High probe latency for {{ $labels.target }}"
```

### Нет успешных проб

```yaml
- alert: NoSuccessfulProbes
  expr: |
    (time() - vmprober_probe_last_success_timestamp) > 300
  for: 5m
  annotations:
    summary: "No successful probes for {{ $labels.target }} in 5 minutes"
```

## Grafana Dashboard

Пример панелей для Grafana:

### Success Rate Panel

```json
{
  "targets": [{
    "expr": "rate(vmprober_probe_success_total[5m]) / rate(vmprober_probe_attempts_total[5m])",
    "legendFormat": "{{target}}"
  }],
  "type": "graph",
  "title": "Probe Success Rate"
}
```

### RTT Histogram Panel

```json
{
  "targets": [{
    "expr": "histogram_quantile(0.95, rate(vmprober_probe_rtt_seconds_bucket[5m]))",
    "legendFormat": "p95 - {{target}}"
  }],
  "type": "graph",
  "title": "Probe RTT (95th percentile)"
}
```

### Failure Rate Table

```json
{
  "targets": [{
    "expr": "rate(vmprober_probe_failure_total[5m]) / rate(vmprober_probe_attempts_total[5m])",
    "format": "table"
  }],
  "type": "table",
  "title": "Failure Rate by Target"
}
```

## Настройка метрик

Метрики можно настроить через конфигурацию:

```yaml
metrics:
  namespace: "vmprober"  # Префикс для всех метрик
  include_labels:         # Лейблы для включения
    - "job"
    - "instance"
    - "probe"
    - "target"
    - "proto"
  custom_labels:          # Дополнительные статические лейблы
    environment: "production"
    region: "us-east"
  buckets:                # Buckets для histogram
    - 0.001
    - 0.005
    - 0.01
    - 0.025
    - 0.05
    - 0.1
    - 0.25
    - 0.5
    - 1.0
    - 2.5
    - 5.0
    - 10.0
  enable_process_metrics: true
  enable_go_metrics: true
```

## См. также

- [API Reference](api-reference.md) - HTTP API endpoints
- [Конфигурация](configuration.md) - Настройка метрик
- [Примеры использования](examples.md) - Практические примеры


