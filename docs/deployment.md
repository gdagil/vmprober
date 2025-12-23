# Развертывание VMProber

Руководство по развертыванию VMProber в различных окружениях.

## Docker

### Базовый запуск

```bash
docker run -d \
  --name vmprober \
  -p 8429:8429 \
  -v $(pwd)/config.yaml:/etc/vmprober/config.yaml \
  vmprober:latest \
  --config=/etc/vmprober/config.yaml
```

### С volume для WAL

```bash
docker run -d \
  --name vmprober \
  -p 8429:8429 \
  -v $(pwd)/config.yaml:/etc/vmprober/config.yaml \
  -v vmprober-wal:/var/lib/vmprober/wal \
  vmprober:latest \
  --config=/etc/vmprober/config.yaml
```

### Docker Compose

Создайте `docker-compose.yml`:

```yaml
version: '3.8'

services:
  vmprober:
    image: vmprober:latest
    container_name: vmprober
    ports:
      - "8429:8429"
    volumes:
      - ./config.yaml:/etc/vmprober/config.yaml:ro
      - vmprober-wal:/var/lib/vmprober/wal
    command: ["--config=/etc/vmprober/config.yaml"]
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8429/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s

volumes:
  vmprober-wal:
```

Запуск:

```bash
docker-compose up -d
```

## Kubernetes

### ConfigMap

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: vmprober-config
  namespace: monitoring
data:
  config.yaml: |
    listen:
      port: 8429
      host: "0.0.0.0"
    targets:
      static:
        - host: "example.com"
          port: 80
          proto: "tcp"
          interval: 30s
    metrics:
      namespace: "vmprober"
```

### Secret (для токенов)

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: vmprober-secrets
  namespace: monitoring
type: Opaque
stringData:
  vm-token: "your-token-here"
```

### Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: vmprober
  namespace: monitoring
  labels:
    app: vmprober
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
        imagePullPolicy: IfNotPresent
        ports:
        - name: http
          containerPort: 8429
          protocol: TCP
        - name: pprof
          containerPort: 6060
          protocol: TCP
        volumeMounts:
        - name: config
          mountPath: /etc/vmprober
          readOnly: true
        - name: wal
          mountPath: /var/lib/vmprober/wal
        command: ["--config=/etc/vmprober/config.yaml"]
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
            port: http
          initialDelaySeconds: 30
          periodSeconds: 10
          timeoutSeconds: 5
        readinessProbe:
          httpGet:
            path: /ready
            port: http
          initialDelaySeconds: 10
          periodSeconds: 5
          timeoutSeconds: 3
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
  namespace: monitoring
  labels:
    app: vmprober
spec:
  type: ClusterIP
  ports:
  - port: 8429
    targetPort: http
    protocol: TCP
    name: http
  selector:
    app: vmprober
```

### ServiceMonitor (Prometheus Operator)

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: vmprober
  namespace: monitoring
  labels:
    app: vmprober
spec:
  selector:
    matchLabels:
      app: vmprober
  endpoints:
  - port: http
    path: /metrics
    interval: 30s
    scrapeTimeout: 10s
```

## Systemd

### Service файл

Создайте `/etc/systemd/system/vmprober.service`:

```ini
[Unit]
Description=VMProber - Network probe monitoring tool
After=network.target

[Service]
Type=simple
User=vmprober
Group=vmprober
WorkingDirectory=/opt/vmprober
ExecStart=/opt/vmprober/bin/vmprober --config=/etc/vmprober/config.yaml
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal
SyslogIdentifier=vmprober

# Security settings
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/vmprober/wal /var/log/vmprober

# Resource limits
LimitNOFILE=65536
LimitNPROC=4096

[Install]
WantedBy=multi-user.target
```

### Установка

```bash
# Создать пользователя
sudo useradd -r -s /bin/false vmprober

# Создать директории
sudo mkdir -p /opt/vmprober/bin
sudo mkdir -p /etc/vmprober
sudo mkdir -p /var/lib/vmprober/wal
sudo mkdir -p /var/log/vmprober

# Скопировать бинарник
sudo cp bin/vmprober /opt/vmprober/bin/
sudo chmod +x /opt/vmprober/bin/vmprober

# Скопировать конфигурацию
sudo cp config.yaml /etc/vmprober/

# Установить права
sudo chown -R vmprober:vmprober /opt/vmprober
sudo chown -R vmprober:vmprober /var/lib/vmprober
sudo chown -R vmprober:vmprober /var/log/vmprober

# Включить и запустить сервис
sudo systemctl daemon-reload
sudo systemctl enable vmprober
sudo systemctl start vmprober

# Проверить статус
sudo systemctl status vmprober
```

## Мониторинг

### Prometheus

Добавьте в `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'vmprober'
    scrape_interval: 30s
    static_configs:
      - targets: ['vmprober:8429']
    metrics_path: '/metrics'
```

### Grafana Dashboard

Создайте dashboard для визуализации метрик VMProber:

```json
{
  "dashboard": {
    "title": "VMProber",
    "panels": [
      {
        "title": "Probe Success Rate",
        "targets": [{
          "expr": "rate(vmprober_probe_success_total[5m]) / rate(vmprober_probe_attempts_total[5m])"
        }]
      },
      {
        "title": "Probe RTT",
        "targets": [{
          "expr": "histogram_quantile(0.95, rate(vmprober_probe_rtt_seconds_bucket[5m]))"
        }]
      }
    ]
  }
}
```

### Алерты

См. примеры в [examples.md](examples.md#мониторинг-и-алертинг).

## Безопасность

### Ограничение доступа

Используйте firewall для ограничения доступа к порту 8429:

```bash
# UFW
sudo ufw allow from 10.0.0.0/8 to any port 8429

# iptables
sudo iptables -A INPUT -p tcp --dport 8429 -s 10.0.0.0/8 -j ACCEPT
sudo iptables -A INPUT -p tcp --dport 8429 -j DROP
```

### TLS

Включите TLS в конфигурации:

```yaml
listen:
  port: 8443
  tls:
    enabled: true
    cert_file: "/etc/vmprober/cert.pem"
    key_file: "/etc/vmprober/key.pem"
```

### Network Policies (Kubernetes)

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: vmprober-netpol
  namespace: monitoring
spec:
  podSelector:
    matchLabels:
      app: vmprober
  policyTypes:
  - Ingress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: monitoring
    ports:
    - protocol: TCP
      port: 8429
```

## Масштабирование

### Горизонтальное масштабирование

VMProber можно запускать в нескольких экземплярах. Каждый экземпляр будет выполнять пробы независимо.

```yaml
# Kubernetes
spec:
  replicas: 3
```

### Вертикальное масштабирование

Увеличьте ресурсы при высокой нагрузке:

```yaml
resources:
  requests:
    memory: "256Mi"
    cpu: "200m"
  limits:
    memory: "1Gi"
    cpu: "1000m"
```

## Резервное копирование

### WAL данные

Регулярно создавайте резервные копии WAL:

```bash
# Backup script
#!/bin/bash
BACKUP_DIR="/backup/vmprober"
WAL_DIR="/var/lib/vmprober/wal"
DATE=$(date +%Y%m%d_%H%M%S)

mkdir -p "$BACKUP_DIR"
tar -czf "$BACKUP_DIR/wal_$DATE.tar.gz" "$WAL_DIR"
```

### Конфигурация

Храните конфигурацию в системе управления версиями (Git).

## Troubleshooting

### Проверка логов

```bash
# Docker
docker logs vmprober

# Kubernetes
kubectl logs -f deployment/vmprober

# Systemd
sudo journalctl -u vmprober -f
```

### Проверка метрик

```bash
curl http://localhost:8429/metrics | grep vmprober
```

### Проверка здоровья

```bash
curl http://localhost:8429/health
curl http://localhost:8429/ready
```

## См. также

- [Примеры использования](examples.md)
- [Конфигурация](configuration.md)
- [Troubleshooting](troubleshooting.md)

