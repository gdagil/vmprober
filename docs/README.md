# VMProber Documentation

Welcome to VMProber documentation - a standalone Go application for monitoring host availability through TCP/UDP/ICMP probes with support for exporting metrics to Prometheus/VictoriaMetrics.

## Contents

### Getting Started
- [Installation and Quick Start](installation.md) - Installation, build, and first run
- [Configuration](configuration.md) - Detailed configuration description
- [Usage Examples](examples.md) - Practical examples

### Architecture
- [Architecture Overview](architecture.md) - Overall system architecture
- [Project Structure](project-structure.md) - Codebase organization
- [Module Interfaces](module-interfaces.md) - Component interfaces

### Components
- [Probe System](probe-system.md) - TCP, UDP, ICMP probes
- [Task Scheduler](scheduler-system.md) - Probe scheduling and execution
- [Metrics System](prometheus-metrics.md) - Prometheus metrics export
- [HTTP Server](http-server.md) - API endpoints and health checks
- [Configuration Module](configuration-module.md) - Configuration management
- [WAL System](wal-system.md) - Write-Ahead Log for reliability
- [VictoriaMetrics Adapter](victoriametrics-adapter.md) - Push mode export

### Operations
- [Deployment](deployment.md) - Docker, Kubernetes, systemd
- [Monitoring and Observability](observability-system.md) - Logging, metrics, tracing
- [Graceful Shutdown](graceful-shutdown.md) - Proper shutdown handling

### Development
- [Developer Guide](development.md) - Environment setup, testing
- [Data Structures](data-structures.md) - Types and data structures
- [Normalizer System](normalizer-system.md) - Probe result normalization

### Reference
- [API Reference](api-reference.md) - HTTP API endpoints
- [Metrics](metrics-reference.md) - Description of all metrics
- [Troubleshooting](troubleshooting.md) - Problem solving

## Quick Start

```bash
# Build
make build

# Run with example configuration
./bin/vmprober --config=config.yaml.example

# Docker
make docker-build
make docker-run
```

## Key Features

- ✅ **TCP/UDP/ICMP probes** - Port and host availability monitoring
- ✅ **Prometheus metrics** - Pull model via `/metrics` endpoint
- ✅ **VictoriaMetrics push** - Push model for metrics export
- ✅ **Task scheduler** - Automatic probe scheduling with intervals
- ✅ **Hot reload** - Configuration reload without restart
- ✅ **WAL** - Write-Ahead Log for fault tolerance
- ✅ **Graceful shutdown** - Proper shutdown handling
- ✅ **Observability** - Logging, metrics, pprof

## Versions

Current version: see `README.md` in project root

## Support

For questions and suggestions, create issues in the project repository: https://github.com/gdagil/vmprober
