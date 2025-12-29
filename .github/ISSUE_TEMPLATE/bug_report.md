---
name: Bug Report
about: Report a bug in VMProber
title: '[BUG] '
labels: bug
assignees: ''
---

## Bug Description

A clear and concise description of what the bug is.

## Component Affected

- [ ] HTTP Probe
- [ ] TCP Probe
- [ ] UDP Probe
- [ ] DNS Probe
- [ ] ICMP Probe
- [ ] gRPC Probe
- [ ] Scheduler
- [ ] Metrics Collector
- [ ] VictoriaMetrics Adapter
- [ ] WAL (Write-Ahead Log)
- [ ] Configuration
- [ ] Web UI
- [ ] Docker / Deployment
- [ ] Other: ___

## Steps to Reproduce

1. Configure vmprober with...
2. Start with command...
3. Observe...

## Expected Behavior

What you expected to happen.

## Actual Behavior

What actually happened.

## Environment

| Field | Value |
|-------|-------|
| VMProber Version | e.g. 1.0.0 |
| Go Version | e.g. 1.21.0 |
| OS | e.g. Ubuntu 22.04 |
| Architecture | e.g. amd64, arm64 |
| Deployment | e.g. binary, Docker, Docker Compose |

## Configuration

```yaml
# Relevant parts of your config.yaml (remove sensitive information)
probes:
  - name: example
    type: http
    target: https://example.com
    interval: 30s
```

## Logs

```
# vmprober logs (use --log-level=debug for more details)
```

## Metrics Output

```
# If applicable, include relevant metric output
```

## Possible Solution

If you have ideas on how to fix this, please describe them here.
