# API Reference

VMProber provides HTTP API endpoints for metrics and health checks.

## Base URL

Default: `http://localhost:8429`

## Endpoints

### GET /health

Health check endpoint.

**Response:**
```json
{
  "status": "healthy",
  "uptime": "1h23m45s"
}
```

**Status Codes:**
- `200 OK` - Application is healthy

### GET /ready

Readiness check endpoint.

**Response:**
```json
{
  "ready": true
}
```

**Status Codes:**
- `200 OK` - Application is ready
- `503 Service Unavailable` - Application is not ready

### GET /metrics

Prometheus metrics endpoint.

**Response:** Prometheus text format

**Status Codes:**
- `200 OK` - Metrics exported successfully

**Content-Type:** `text/plain; version=0.0.4; charset=utf-8`

## Example Usage

### Health Check

```bash
curl http://localhost:8429/health
```

### Readiness Check

```bash
curl http://localhost:8429/ready
```

### Get Metrics

```bash
curl http://localhost:8429/metrics
```

### Filter Metrics

```bash
curl http://localhost:8429/metrics | grep vmprober_probe_success_total
```

## Integration with Prometheus

Add to `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'vmprober'
    scrape_interval: 30s
    static_configs:
      - targets: ['localhost:8429']
    metrics_path: '/metrics'
```

## TLS

If TLS is enabled, use HTTPS:

```bash
curl https://localhost:8429/health
```

## See Also

- [Metrics Reference](metrics.md) - All exported metrics
- [Operations: Monitoring](../operations/monitoring.md) - Setting up monitoring
