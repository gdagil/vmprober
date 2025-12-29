---
name: Feature Request
about: Suggest a new feature for VMProber
title: '[FEATURE] '
labels: enhancement
assignees: ''
---

## Feature Description

A clear and concise description of the feature you'd like to see.

## Component

- [ ] New Probe Type
- [ ] Existing Probe Enhancement (HTTP, TCP, UDP, DNS, ICMP, gRPC)
- [ ] Scheduler
- [ ] Metrics / Monitoring
- [ ] VictoriaMetrics Integration
- [ ] Configuration
- [ ] Web UI / API
- [ ] Grafana Dashboard
- [ ] Docker / Deployment
- [ ] Other: ___

## Problem Statement

What problem does this feature solve? What use case does it address?

## Proposed Solution

Describe how you envision this feature working.

### Example Configuration

```yaml
# If applicable, show how the feature would be configured
probes:
  - name: example
    type: ...
    new_option: value
```

### Example Metrics

```
# If applicable, show what new metrics would be exposed
vmprober_probe_new_metric{name="example"} 1.0
```

## Alternatives Considered

Describe any alternative solutions or features you've considered.

## Use Cases

1. **Use Case 1**: Description
2. **Use Case 2**: Description

## Additional Context

Add any other context, mockups, or references about the feature request here.
