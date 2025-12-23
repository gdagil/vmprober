# Troubleshooting

Common issues and solutions for VMProber.

## Application Won't Start

### Configuration Loading Error

**Symptoms:**
```
Failed to load configuration: failed to parse config: yaml: ...
```

**Solutions:**
1. Check YAML syntax: `yamllint config.yaml`
2. Validate configuration structure
3. Ensure file exists and is readable

### Port Already in Use

**Symptoms:**
```
Failed to start HTTP server: listen tcp :8429: bind: address already in use
```

**Solutions:**
1. Change port in configuration
2. Stop the process using the port: `lsof -i :8429 && kill <PID>`

## Metrics Not Exporting

### `/metrics` Endpoint Not Responding

**Solutions:**
1. Check HTTP server is running: `curl http://localhost:8429/health`
2. Check logs for errors
3. Ensure `pull.enabled: true` in configuration

### Metrics Are Empty

**Solutions:**
1. Verify targets are configured
2. Check logs for probe execution errors
3. Ensure targets are accessible

## Probes Not Executing

### All Probes Fail

**Solutions:**
1. Check target accessibility: `telnet example.com 80`
2. Check firewall rules
3. Increase timeout in configuration
4. Enable debug logging

### ICMP Probes Not Working

**Solutions:**
1. Check permissions (requires root/administrator)
2. Check system settings: `sysctl net.ipv4.ping_group_range`
3. Use alternative library in configuration

## Performance Issues

### High CPU Usage

**Solutions:**
1. Reduce concurrent probes
2. Increase intervals between probes
3. Use jitter for load distribution

### High Memory Usage

**Solutions:**
1. Reduce queue size
2. Configure WAL to limit size
3. Check for memory leaks using pprof

## WAL Issues

### WAL Write Errors

**Solutions:**
1. Check WAL directory permissions
2. Ensure directory exists
3. Change WAL path to accessible directory

### WAL Taking Too Much Space

**Solutions:**
1. Configure retention
2. Configure maximum size
3. Enable compression

## Push Mode Issues

### Metrics Not Sending to VictoriaMetrics

**Solutions:**
1. Check endpoint accessibility
2. Check authentication (tokens, credentials)
3. Enable debug logging
4. Check retry settings

## Debugging

### Enable Debug Logging

```yaml
logging:
  level: "debug"
  format: "json"
```

Or via command line:

```bash
./bin/vmprober --config=config.yaml --log-level=debug
```

### Using pprof

Enable in configuration:

```yaml
observability:
  pprof:
    enabled: true
    port: 6060
```

Profile:

```bash
go tool pprof http://localhost:6060/debug/pprof/profile
```

## Getting Help

If issues persist:

1. Collect information:
   - VMProber version
   - Configuration (without secrets)
   - Logs with debug level
   - Metrics (`/metrics`)
   - Environment information

2. Create an issue with:
   - Problem description
   - Steps to reproduce
   - Collected information

## See Also

- [Deployment](deployment.md) - Deployment guide
- [Monitoring](monitoring.md) - Monitoring setup

