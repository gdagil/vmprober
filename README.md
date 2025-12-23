# VMProber

[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org)

VMProber is a standalone Go application for monitoring host availability through TCP/UDP/ICMP probes with support for both pull and push models for exporting metrics to Prometheus/VictoriaMetrics.

**Repository:** https://github.com/gdagil/vmprober

## Features

- ✅ **TCP/UDP probes** - Port availability monitoring
- ✅ **Prometheus metrics** - Pull model via `/metrics` endpoint
- ✅ **Task scheduler** - Automatic probe scheduling with intervals
- ✅ **HTTP server** - Health checks and metrics
- ✅ **Configuration** - YAML configuration with hot reload
- ✅ **Graceful shutdown** - Proper shutdown handling

## Quick Start

### Build

```bash
make build
```

### Run

```bash
./bin/vmprober --config=config.yaml.example
```

### Docker

```bash
make docker-build
make docker-run
```

## Project Structure

```
vmprober/
├── cmd/vmprober/          # Main application entry point
├── internal/
│   ├── config/           # Configuration management
│   ├── probe/            # Probe system (TCP, UDP)
│   ├── scheduler/         # Task scheduler
│   ├── metrics/          # Prometheus metrics system
│   ├── server/           # HTTP server
│   └── types/            # Base types
├── pkg/interfaces/       # Public interfaces
├── configs/              # Configuration examples
└── docs/                 # Documentation
```

## Configuration

Configuration example is located in `config.yaml.example`. Main sections:

- `listen` - HTTP server settings
- `pull` - Pull mode (Prometheus scrape)
- `push` - Push mode (VictoriaMetrics)
- `scheduler` - Scheduler settings
- `targets` - List of monitoring targets
- `probes` - Default probe settings
- `metrics` - Metrics settings

## API Endpoints

- `GET /metrics` - Prometheus metrics
- `GET /health` - Health check
- `GET /ready` - Readiness check

## Metrics

VMProber exports the following metrics:

- `vmprober_probe_success_total` - Number of successful probes
- `vmprober_probe_failure_total` - Number of failed probes
- `vmprober_probe_rtt_seconds` - Response time (histogram)
- `vmprober_probe_attempts_total` - Total number of attempts
- `vmprober_probe_last_success_timestamp` - Timestamp of last successful probe

## Development

### Dependencies

```bash
make deps
```

### Testing

```bash
make test
make test-coverage
```

### Formatting

```bash
make fmt
```

## Architecture

The project is implemented with a modular architecture:

1. **Config Manager** - Configuration loading and validation
2. **Probe Factory** - Creating probes of various types
3. **Scheduler** - Task scheduling with priorities
4. **Metrics Collector** - Prometheus metrics collection and export
5. **HTTP Server** - Providing metrics and health checks

## Implementation Status

### ✅ Implemented

- [x] Basic project structure
- [x] Configuration module with hot reload
- [x] TCP and UDP probes
- [x] Task scheduler
- [x] Prometheus metrics system
- [x] HTTP server with endpoints
- [x] Main application file
- [x] Configuration example

### 🔄 In Development

- [ ] ICMP probes
- [ ] WAL system
- [ ] VictoriaMetrics adapter
- [ ] Result normalizer
- [ ] Extended observability
- [ ] Graceful shutdown improvements

## Contributing

We welcome contributions from the community! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for information on how to contribute.

## Authors

VMProber Team

## Acknowledgments

Thank you to all contributors who make VMProber better!
