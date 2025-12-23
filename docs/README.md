# VMProber Documentation

Welcome to the VMProber documentation. VMProber is a standalone Go application for monitoring host availability through TCP/UDP/ICMP probes with support for both pull and push models for exporting metrics to Prometheus/VictoriaMetrics.

## Documentation Structure

### 📚 [Getting Started](getting-started/)
Start here if you're new to VMProber:
- [Installation](getting-started/installation.md) - How to install and build VMProber
- [Quick Start](getting-started/quick-start.md) - Get up and running in 5 minutes
- [Configuration Guide](getting-started/configuration.md) - Understanding configuration files
- [Basic Usage](getting-started/basic-usage.md) - Your first probes and metrics

### 🏗️ [Architecture](architecture/)
Understand how VMProber works:
- [System Overview](architecture/overview.md) - High-level system architecture
- [Design Principles](architecture/design-principles.md) - Architectural decisions and patterns
- [Component Architecture](architecture/components.md) - Detailed component design
- [Data Flow](architecture/data-flow.md) - How data moves through the system

### 🔧 [Components](components/)
Deep dive into VMProber components:
- [Probe System](components/probes.md) - TCP, UDP, and ICMP probes
- [Scheduler](components/scheduler.md) - Task scheduling and execution
- [Metrics System](components/metrics.md) - Prometheus metrics collection
- [WAL System](components/wal.md) - Write-Ahead Log for reliability
- [VictoriaMetrics Adapter](components/victoriametrics-adapter.md) - Push mode export
- [Normalizer](components/normalizer.md) - Result normalization and deduplication
- [HTTP Server](components/http-server.md) - API endpoints and health checks
- [Observability](components/observability.md) - Logging, profiling, and monitoring

### 🚀 [Operations](operations/)
Deploy and operate VMProber:
- [Deployment](operations/deployment.md) - Docker, Kubernetes, systemd
- [Configuration Management](operations/configuration.md) - Managing configs in production
- [Monitoring](operations/monitoring.md) - Setting up monitoring and alerting
- [Troubleshooting](operations/troubleshooting.md) - Common issues and solutions
- [Performance Tuning](operations/performance.md) - Optimizing for your workload
- [Security](operations/security.md) - Security best practices

### 👨‍💻 [Development](development/)
For contributors and developers:
- [Development Setup](development/setup.md) - Environment and tooling
- [Project Structure](development/project-structure.md) - Codebase organization
- [Contributing Guide](development/contributing.md) - How to contribute
- [Testing](development/testing.md) - Testing strategies and practices
- [Code Style](development/code-style.md) - Coding standards and conventions

### 📖 [Reference](reference/)
API and technical reference:
- [API Reference](reference/api.md) - HTTP API endpoints
- [Metrics Reference](reference/metrics.md) - All exported metrics
- [Data Structures](reference/data-structures.md) - Types and structures
- [Configuration Reference](reference/configuration-reference.md) - Complete config options
- [CLI Reference](reference/cli.md) - Command-line options

### 📘 [Guides](guides/)
Step-by-step guides for common tasks:
- [Deployment Guide](guides/deployment-guide.md) - Complete deployment walkthrough
- [Monitoring Setup](guides/monitoring-setup.md) - Setting up Prometheus/Grafana
- [VictoriaMetrics Integration](guides/victoriametrics-integration.md) - Push mode setup
- [High Availability](guides/high-availability.md) - Running VMProber in HA mode
- [Migration Guide](guides/migration.md) - Upgrading between versions

## Quick Navigation

### I want to...

- **Install VMProber** → [Getting Started: Installation](getting-started/installation.md)
- **Configure my first probe** → [Getting Started: Quick Start](getting-started/quick-start.md)
- **Deploy to production** → [Operations: Deployment](operations/deployment.md)
- **Understand the architecture** → [Architecture: Overview](architecture/overview.md)
- **Troubleshoot an issue** → [Operations: Troubleshooting](operations/troubleshooting.md)
- **Contribute code** → [Development: Contributing](development/contributing.md)
- **See all metrics** → [Reference: Metrics](reference/metrics.md)
- **Set up monitoring** → [Guides: Monitoring Setup](guides/monitoring-setup.md)

## Key Features

- ✅ **Multi-protocol probes** - TCP, UDP, and ICMP support
- ✅ **Dual export modes** - Pull (Prometheus) and Push (VictoriaMetrics)
- ✅ **Reliability** - WAL system for fault tolerance
- ✅ **Performance** - Efficient scheduling with rate limiting
- ✅ **Observability** - Comprehensive metrics, logging, and profiling
- ✅ **Production-ready** - Graceful shutdown, health checks, hot reload

## Getting Help

- **Documentation Issues**: Open an issue in the repository
- **Questions**: Check [Troubleshooting](operations/troubleshooting.md) first
- **Bugs**: Report via GitHub Issues
- **Feature Requests**: Open a GitHub Discussion

## Version Information

Current documentation version: See project README for version information.

---

**Note**: This documentation is continuously updated. If you find any issues or have suggestions, please contribute!
