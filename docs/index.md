---
layout: default
title: VMProber Documentation
---

# VMProber Documentation

Welcome to the VMProber documentation. VMProber is a standalone Go application for monitoring host availability through TCP/UDP/ICMP probes with support for both pull and push models for exporting metrics to Prometheus/VictoriaMetrics.

## Quick Navigation

### 📚 [Getting Started](getting-started/)
Start here if you're new to VMProber:
- [Installation](getting-started/installation) - How to install and build VMProber
- [Quick Start](getting-started/quick-start) - Get up and running in 5 minutes
- [Configuration Guide](getting-started/configuration) - Understanding configuration files
- [Basic Usage](getting-started/basic-usage) - Your first probes and metrics

### 🏗️ [Architecture](architecture/)
Understand how VMProber works:
- [System Overview](architecture/overview) - High-level system architecture
- [Design Principles](architecture/design-principles) - Architectural decisions and patterns

### 🔧 [Components](components/)
Deep dive into VMProber components:
- [Probe System](components/probes) - TCP, UDP, and ICMP probes

### 🚀 [Operations](operations/)
Deploy and operate VMProber:
- [Deployment](operations/deployment) - Docker, Kubernetes, systemd
- [Troubleshooting](operations/troubleshooting) - Common issues and solutions

### 👨‍💻 [Development](development/)
For contributors and developers:
- [Development Setup](development/setup) - Environment and tooling

### 📖 [Reference](reference/)
API and technical reference:
- [API Reference](reference/api) - HTTP API endpoints
- [Metrics Reference](reference/metrics) - All exported metrics

### 📘 [Guides](guides/)
Step-by-step guides for common tasks:
- [Monitoring Setup](guides/monitoring-setup) - Setting up Prometheus/Grafana

## Key Features

- ✅ **Multi-protocol probes** - TCP, UDP, and ICMP support
- ✅ **Dual export modes** - Pull (Prometheus) and Push (VictoriaMetrics)
- ✅ **Reliability** - WAL system for fault tolerance
- ✅ **Performance** - Efficient scheduling with rate limiting
- ✅ **Observability** - Comprehensive metrics, logging, and profiling
- ✅ **Production-ready** - Graceful shutdown, health checks, hot reload

## Getting Help

- **Documentation Issues**: Open an issue in the repository
- **Questions**: Check [Troubleshooting](operations/troubleshooting) first
- **Bugs**: Report via GitHub Issues
- **Feature Requests**: Open a GitHub Discussion

---

**Repository**: [https://github.com/gdagil/vmprober](https://github.com/gdagil/vmprober)

