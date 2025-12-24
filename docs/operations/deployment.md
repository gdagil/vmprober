# Deployment

Deploy VMProber in various environments.

## Docker

### Basic Deployment

```bash
docker run -d \
  --name vmprober \
  -p 8429:8429 \
  -v $(pwd)/config.yaml:/etc/vmprober/config.yaml \
  vmprober:latest \
  --config=/etc/vmprober/config.yaml
```

### With WAL Volume

```bash
docker run -d \
  --name vmprober \
  -p 8429:8429 \
  -v $(pwd)/config.yaml:/etc/vmprober/config.yaml \
  -v vmprober-wal:/var/lib/vmprober/wal \
  vmprober:latest
```

### Docker Compose

```yaml
version: '3.8'
services:
  vmprober:
    image: vmprober:latest
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

volumes:
  vmprober-wal:
```

## Kubernetes

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
          protocols: ["tcp"]
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
          initialDelaySeconds: 10
          periodSeconds: 5
      volumes:
      - name: config
        configMap:
          name: vmprober-config
      - name: wal
        emptyDir: {}
```

## Systemd

Create `/etc/systemd/system/vmprober.service`:

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

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl enable vmprober
sudo systemctl start vmprober
```

## Production Considerations

### Resource Limits
- Memory: 128Mi - 512Mi depending on workload
- CPU: 100m - 500m
- Disk: 1GB+ for WAL (depends on retention)

### High Availability
- Run multiple instances
- Use load balancer for metrics endpoint
- Shared WAL storage (if needed)

### Security
- Use TLS for HTTP server
- Restrict network access
- Use secrets for tokens

## Next Steps

- [Configuration Management](configuration.md) - Managing configs in production
- [Monitoring](monitoring.md) - Setting up monitoring
- [Guides: Deployment Guide](../guides/deployment-guide.md) - Complete walkthrough


