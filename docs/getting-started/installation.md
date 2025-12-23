# Installation

This guide covers different ways to install and build VMProber.

## Requirements

- **Go**: Version 1.21 or higher
- **Operating System**: Linux, macOS, or Windows
- **For ICMP probes**: Root privileges (Linux) or Administrator (Windows)
- **Make**: For using Makefile commands (optional)

## Installation Methods

### From Source

#### 1. Clone the Repository

```bash
git clone https://github.com/gdagil/vmprober.git
cd vmprober
```

#### 2. Install Dependencies

```bash
make deps
# or manually:
go mod download
go mod tidy
```

#### 3. Build

```bash
# Standard build
make build

# Build with version
VERSION=1.0.0 make build

# Cross-compilation
GOOS=linux GOARCH=amd64 make build
```

The binary will be created in `bin/vmprober`.

### Using Docker

#### Build Docker Image

```bash
make docker-build
# or
docker build -t vmprober:latest .
```

#### Run Container

```bash
make docker-run
# or
docker run --rm -p 8429:8429 \
  -v $(pwd)/config.yaml:/etc/vmprober/config.yaml \
  vmprober:latest --config=/etc/vmprober/config.yaml
```

### Using Pre-built Binaries

Pre-built binaries are available in the [Releases](https://github.com/gdagil/vmprober/releases) section.

1. Download the appropriate binary for your platform
2. Make it executable: `chmod +x vmprober`
3. Move to a directory in your PATH: `sudo mv vmprober /usr/local/bin/`

## Verification

After installation, verify it works:

```bash
# Check version
./bin/vmprober --version

# Check help
./bin/vmprober --help

# Test run (will use default config if available)
./bin/vmprober --config=config.yaml.example
```

## Next Steps

- [Quick Start](quick-start.md) - Get VMProber running with your first probe
- [Configuration Guide](configuration.md) - Learn about configuration options


