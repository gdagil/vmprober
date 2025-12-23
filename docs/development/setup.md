# Development Setup

Set up your development environment for VMProber.

## Requirements

- **Go**: Version 1.21 or higher
- **Make**: For Makefile commands
- **Git**: For version control
- **golangci-lint**: For linting (optional but recommended)

## Environment Setup

### 1. Clone Repository

```bash
git clone https://github.com/gdagil/vmprober.git
cd vmprober
```

### 2. Install Dependencies

```bash
make deps
```

### 3. Verify Installation

```bash
go version
make build
```

## Development Workflow

### Building

```bash
# Standard build
make build

# Build with version
VERSION=1.0.0 make build

# Cross-compilation
GOOS=linux GOARCH=amd64 make build
```

### Testing

```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run specific test
go test -v ./internal/probe/...

# Run with race detector
go test -race ./...
```

### Formatting and Linting

```bash
# Format code
make fmt

# Lint code
make lint
```

## IDE Setup

### VS Code

Recommended extensions:
- Go extension
- YAML extension
- Markdown extension

### GoLand

No additional setup needed.

## Debugging

### Using Delve

```bash
# Install
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug
dlv debug ./cmd/vmprober -- --config=config.yaml.example
```

### Using pprof

Enable in configuration and profile:

```bash
go tool pprof http://localhost:6060/debug/pprof/profile
```

## Next Steps

- [Project Structure](project-structure.md) - Codebase organization
- [Contributing](contributing.md) - How to contribute
- [Testing](testing.md) - Testing strategies

