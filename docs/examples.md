# Примеры использования VMProber

## Базовые примеры

### Мониторинг веб-сервера

```yaml
targets:
  static:
    - host: "example.com"
      port: 80
      proto: "tcp"
      interval: 30s
      timeout: 5s
      labels:
        service: "web"
        environment: "production"
```

### Мониторинг HTTPS сервера

```yaml
targets:
  static:
    - host: "example.com"
      port: 443
      proto: "tcp"
      interval: 30s
      timeout: 5s
      labels:
        service: "web"
probes:
  tcp:
    tls:
      enabled: true
      server_name: "example.com"
```

### Мониторинг DNS сервера

```yaml
targets:
  static:
    - host: "8.8.8.8"
      port: 53
      proto: "udp"
      interval: 60s
      timeout: 3s
      labels:
        service: "dns"
        provider: "google"
```

### Мониторинг нескольких портов одного хоста

```yaml
targets:
  static:
    - host: "example.com"
      port: 80
      proto: "tcp"
      interval: 30s
      labels:
        port: "http"
    - host: "example.com"
      port: 443
      proto: "tcp"
      interval: 30s
      labels:
        port: "https"
    - host: "example.com"
      port: 22
      proto: "tcp"
      interval: 60s
      labels:
        port: "ssh"
```

## Продвинутые примеры

### Использование файлового источника целей

```yaml
targets:
  files:
    - path: "/etc/vmprober/targets.yaml"
      reload_interval: 1m
      watch: true
```

Формат файла `targets.yaml`:

```yaml
- host: "service1.example.com"
  port: 8429
  proto: "tcp"
  interval: 30s
  labels:
    service: "api"
- host: "service2.example.com"
  port: 9090
  proto: "tcp"
  interval: 30s
  labels:
    service: "metrics"
```

### Использование HTTP источника целей

```yaml
targets:
  urls:
    - url: "http://service-discovery:8429/api/targets"
      reload_interval: 5m
      headers:
        Authorization: "Bearer ${DISCOVERY_TOKEN}"
        X-Request-ID: "vmprober-1"
```

### Push метрики в VictoriaMetrics

```yaml
push:
  enabled: true
  endpoints:
    - url: "http://victoria-metrics:8428/api/v1/import/prometheus"
      auth:
        type: "bearer"
        token: "${VM_TOKEN}"
  retry:
    max_attempts: 5
    backoff: "exponential"
    initial_delay: 1s
    max_delay: 60s
  dedup:
    enabled: true
    window: 5m
  batch:
    size: 1000
    timeout: 30s
```

### Настройка rate limiting

```yaml
scheduler:
  concurrent: 50
  rps_limit: 500
  per_host_cap: 5
  jitter: 0.2
```

Это ограничит:
- Максимум 50 одновременных проб
- Максимум 500 проб в секунду глобально
- Максимум 5 проб в секунду на один хост
- Jitter 20% для распределения нагрузки

### Настройка WAL для надежности

```yaml
wal:
  dir: "/var/lib/vmprober/wal"
  max_size: "2GB"
  max_age: 168h
  retention: 7d
  compression: "gzip"
  sync_interval: 1s
```

### Структурированное логирование

```yaml
logging:
  level: "info"
  format: "json"
  output: "stdout"
  structured: true
  include_source: true
  file:
    path: "/var/log/vmprober.log"
    max_size: "100MB"
    max_backups: 10
    max_age: 30
    compress: true
```

## Docker примеры

### Базовый Docker Compose

```yaml
version: '3.8'
services:
  vmprober:
    image: vmprober:latest
    ports:
      - "8429:8429"
    volumes:
      - ./config.yaml:/etc/vmprober/config.yaml
      - ./wal:/var/lib/vmprober/wal
    command: ["--config=/etc/vmprober/config.yaml"]
    restart: unless-stopped
```

### С VictoriaMetrics

```yaml
version: '3.8'
services:
  vmprober:
    image: vmprober:latest
    ports:
      - "8429:8429"
    volumes:
      - ./config.yaml:/etc/vmprober/config.yaml
    environment:
      - VM_TOKEN=${VM_TOKEN}
    depends_on:
      - victoria-metrics
    restart: unless-stopped

  victoria-metrics:
    image: victoriametrics/victoria-metrics:latest
    ports:
      - "8428:8428"
    volumes:
      - vm-data:/victoria-metrics-data
    restart: unless-stopped

volumes:
  vm-data:
```

## Kubernetes примеры

### ConfigMap

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: vmprober-config
data:
  config.yaml: |
    listen:
      port: 8429
    targets:
      static:
        - host: "example.com"
          port: 80
          proto: "tcp"
          interval: 30s
```

### Deployment

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
        command: ["--config=/etc/vmprober/config.yaml"]
      volumes:
      - name: config
        configMap:
          name: vmprober-config
      - name: wal
        emptyDir: {}
```

### Service

```yaml
apiVersion: v1
kind: Service
metadata:
  name: vmprober
spec:
  selector:
    app: vmprober
  ports:
  - port: 8429
    targetPort: 8429
```

### ServiceMonitor для Prometheus

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: vmprober
spec:
  selector:
    matchLabels:
      app: vmprober
  endpoints:
  - port: http
    path: /metrics
    interval: 30s
```

## Интеграция с Prometheus

### Prometheus конфигурация

```yaml
scrape_configs:
  - job_name: 'vmprober'
    scrape_interval: 30s
    static_configs:
      - targets: ['vmprober:8429']
    metrics_path: '/metrics'
```

## Мониторинг и алертинг

### Prometheus алерты

```yaml
groups:
  - name: vmprober
    rules:
      - alert: VMProberDown
        expr: up{job="vmprober"} == 0
        for: 1m
        annotations:
          summary: "VMProber is down"

      - alert: HighProbeFailureRate
        expr: |
          rate(vmprober_probe_failure_total[5m]) / 
          rate(vmprober_probe_attempts_total[5m]) > 0.1
        for: 5m
        annotations:
          summary: "High probe failure rate"

      - alert: HighProbeLatency
        expr: |
          histogram_quantile(0.95, 
            rate(vmprober_probe_rtt_seconds_bucket[5m])
          ) > 1
        for: 5m
        annotations:
          summary: "High probe latency"
```

## Troubleshooting примеры

### Включение debug логирования

```yaml
logging:
  level: "debug"
  format: "json"
```

Запуск с debug уровнем:

```bash
./bin/vmprober --config=config.yaml --log-level=debug
```

### Проверка конфигурации

```bash
# Проверка синтаксиса YAML
yamllint config.yaml

# Проверка валидности конфигурации
./bin/vmprober --config=config.yaml --validate
```

### Мониторинг производительности

Включить pprof:

```yaml
observability:
  pprof:
    enabled: true
    port: 6060
    host: "127.0.0.1"
```

Профилирование:

```bash
go tool pprof http://localhost:6060/debug/pprof/profile
```

## См. также

- [Конфигурация](configuration.md) - Подробное описание конфигурации
- [Развертывание](deployment.md) - Руководство по развертыванию
- [Troubleshooting](troubleshooting.md) - Решение проблем

