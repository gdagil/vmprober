# Installation and Quick Start

## Requirements

- Go 1.21 or higher
- Linux/macOS/Windows
- For ICMP probes: root privileges (Linux) or administrator (Windows)

## Installation from Source

### Cloning Repository

```bash
git clone https://github.com/gdagil/vmprober.git
cd vmprober
```

### Installing Dependencies

```bash
make deps
# or
go mod download
go mod tidy
```

### Build

```bash
make build
```

The binary will be created in `bin/vmprober`.

### Build with Version

```bash
VERSION=1.0.0 make build
```

## Docker Installation

### Building Image

```bash
make docker-build
# or
docker build -t vmprober:latest .
```

### Running Container

```bash
make docker-run
# or
docker run --rm -p 8429:8429 \
  -v $(pwd)/config.yaml.example:/etc/vmprober/config.yaml \
  vmprober:latest --config=/etc/vmprober/config.yaml
```

## Quick Start

### 1. Create Configuration

Copy the example configuration:

```bash
cp config.yaml.example config.yaml
```

Edit `config.yaml` to suit your needs.

### 2. Run Application

```bash
./bin/vmprober --config=config.yaml
```

### 3. Verify Operation

Open in browser or via curl:

```bash
# Health check
curl http://localhost:8429/health

# Metrics
curl http://localhost:8429/metrics

# Readiness check
curl http://localhost:8429/ready
```

## Default Configuration

If no configuration path is specified, VMProber looks for `config.yaml.example` in the current directory.

## Command Line Parameters

```bash
./bin/vmprober --help
```

Available parameters:

- `--config` - Path to configuration file (default: `config.yaml.example`)
- `--log-level` - Log level: `debug`, `info`, `warn`, `error` (default: `info`)

## Installation Verification

After starting, verify:

1. **Logs** - Should contain messages about component startup
2. **HTTP endpoints** - `/health` and `/ready` should return 200 OK
3. **Metrics** - `/metrics` should return Prometheus metrics

Verification example:

```bash
# Health check
curl -s http://localhost:8429/health | jq

# Metrics check
curl -s http://localhost:8429/metrics | grep vmprober
```

## Production Deployment

See [Deployment Guide](deployment.md) for detailed information about:
- Docker Compose
- Kubernetes
- Systemd service
- Monitoring and alerting

## Next Steps

- [Configuration](configuration.md) - VMProber configuration
- [Usage Examples](examples.md) - Practical examples
- [Architecture](architecture.md) - Understanding internal workings
