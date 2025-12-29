package metrics

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/VictoriaMetrics/metrics"

	"github.com/gdagil/vmprober/internal/types"
)

// Collector collects and exports metrics
type Collector struct {
	// Probe metrics - use map to store metrics with labels
	probeSuccess  map[string]*metrics.Counter
	probeFailure  map[string]*metrics.Counter
	probeRTT      map[string]*metrics.Histogram
	probeAttempts map[string]*metrics.Counter
	// Job metrics
	jobsTotal   *metrics.Gauge
	jobsRunning *metrics.Gauge
	jobsFailed  *metrics.Gauge
	// Values for job metrics
	jobsTotalValue   float64
	jobsRunningValue float64
	jobsFailedValue  float64
	mu               sync.RWMutex
	namespace        string
	enableJobMetrics bool
	customLabels     map[string]string // Custom labels (e.g., job)
}

// NewCollector creates a new metrics collector
// customLabels - custom labels that are added to all metrics (e.g., job)
func NewCollector(namespace string, enableJobMetrics bool, customLabels map[string]string) *Collector {
	if namespace == "" {
		namespace = "vmprober"
	}

	// Copy customLabels for safety
	labels := make(map[string]string)
	for k, v := range customLabels {
		labels[k] = v
	}

	collector := &Collector{
		namespace:        namespace,
		enableJobMetrics: enableJobMetrics,
		customLabels:     labels,
		probeSuccess:     make(map[string]*metrics.Counter),
		probeFailure:     make(map[string]*metrics.Counter),
		probeRTT:         make(map[string]*metrics.Histogram),
		probeAttempts:    make(map[string]*metrics.Counter),
	}

	// Initialize job metrics only if they are enabled
	// Use Gauge with callback that returns value from variable
	if enableJobMetrics {
		collector.jobsTotal = metrics.NewGauge(fmt.Sprintf(`%s_jobs_total`, namespace), func() float64 {
			collector.mu.RLock()
			defer collector.mu.RUnlock()
			return collector.jobsTotalValue
		})
		collector.jobsRunning = metrics.NewGauge(fmt.Sprintf(`%s_jobs_running`, namespace), func() float64 {
			collector.mu.RLock()
			defer collector.mu.RUnlock()
			return collector.jobsRunningValue
		})
		collector.jobsFailed = metrics.NewGauge(fmt.Sprintf(`%s_jobs_failed`, namespace), func() float64 {
			collector.mu.RLock()
			defer collector.mu.RUnlock()
			return collector.jobsFailedValue
		})
	}

	return collector
}

// getMetricKey creates a key for metric with labels
// Uses standard Prometheus labels:
// - instance: hostname:port (e.g., "google.com:443")
// - target_ip: IP address (e.g., "142.250.109.100")
// - port: port (e.g., "443")
// - protocol: protocol (e.g., "tcp")
// Also adds customLabels (e.g., job)
func (c *Collector) getMetricKey(name string, instance, targetIP, port, protocol string) string {
	// If instance is empty, use targetIP:port
	if instance == "" {
		if port != "" && port != "0" {
			instance = fmt.Sprintf("%s:%s", targetIP, port)
		} else {
			instance = targetIP
		}
	}

	// Form base labels according to Prometheus standards
	labels := fmt.Sprintf(`instance="%s",target_ip="%s",port="%s",protocol="%s"`,
		instance, targetIP, port, protocol)

	// Add custom labels (e.g., job)
	for k, v := range c.customLabels {
		labels += fmt.Sprintf(`,%s="%s"`, k, v)
	}

	return fmt.Sprintf(`%s_%s{%s}`, c.namespace, name, labels)
}

// getOrCreateCounter gets or creates a counter metric
func (c *Collector) getOrCreateCounter(key string) *metrics.Counter {
	if counter, ok := c.probeAttempts[key]; ok {
		return counter
	}
	counter := metrics.NewCounter(key)
	c.probeAttempts[key] = counter
	return counter
}

// getOrCreateSuccessCounter gets or creates a counter for successful probes
func (c *Collector) getOrCreateSuccessCounter(key string) *metrics.Counter {
	if counter, ok := c.probeSuccess[key]; ok {
		return counter
	}
	counter := metrics.NewCounter(key)
	c.probeSuccess[key] = counter
	return counter
}

// getOrCreateFailureCounter gets or creates a counter for failed probes
func (c *Collector) getOrCreateFailureCounter(key string) *metrics.Counter {
	if counter, ok := c.probeFailure[key]; ok {
		return counter
	}
	counter := metrics.NewCounter(key)
	c.probeFailure[key] = counter
	return counter
}

// getOrCreateHistogram gets or creates a histogram metric
func (c *Collector) getOrCreateHistogram(key string) *metrics.Histogram {
	if histogram, ok := c.probeRTT[key]; ok {
		return histogram
	}
	histogram := metrics.NewHistogram(key)
	c.probeRTT[key] = histogram
	return histogram
}

// Record records a probe result
func (c *Collector) Record(ctx context.Context, result *types.ProbeResult) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Get values for labels
	// instance = hostname:port (e.g., "google.com:443")
	// target_ip = IP address that was actually connected to (e.g., "142.250.109.100")
	// port = port (e.g., "443")
	// protocol = protocol (e.g., "tcp")

	hostname := result.TargetHost // hostname from configuration (e.g., "google.com")
	targetIP := result.TargetIP   // IP address that was connected to (e.g., "142.250.109.100")
	port := "0"
	if result.TargetPort > 0 {
		port = fmt.Sprintf("%d", result.TargetPort)
	}
	protocol := string(result.Protocol)

	// Form instance as hostname:port
	// If hostname is empty, use targetIP
	if hostname == "" {
		hostname = targetIP
	}

	// If targetIP is empty, use hostname
	if targetIP == "" {
		targetIP = "unknown"
	}

	// instance = hostname:port (standard Prometheus format)
	instance := hostname
	if port != "0" && port != "" {
		instance = fmt.Sprintf("%s:%s", hostname, port)
	}

	attemptsKey := c.getMetricKey("probe_attempts_total", instance, targetIP, port, protocol)
	attemptsCounter := c.getOrCreateCounter(attemptsKey)
	attemptsCounter.Inc()

	if result.Success {
		successKey := c.getMetricKey("probe_success_total", instance, targetIP, port, protocol)
		successCounter := c.getOrCreateSuccessCounter(successKey)
		successCounter.Inc()

		rttKey := c.getMetricKey("probe_rtt_seconds", instance, targetIP, port, protocol)
		rttHistogram := c.getOrCreateHistogram(rttKey)
		rttHistogram.Update(result.RTT.Seconds())
	} else {
		failureKey := c.getMetricKey("probe_failure_total", instance, targetIP, port, protocol)
		failureCounter := c.getOrCreateFailureCounter(failureKey)
		failureCounter.Inc()
	}

	return nil
}

// UpdateJobMetrics updates job metrics based on scheduler statistics
func (c *Collector) UpdateJobMetrics(totalJobs, runningJobs int, failedJobs int64) {
	if !c.enableJobMetrics {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Update values through variables (callback will return these values)
	c.jobsTotalValue = float64(totalJobs)
	c.jobsRunningValue = float64(runningJobs)
	c.jobsFailedValue = float64(failedJobs)
}

// ExportMetrics exports all metrics from collector to types.Metric format
func (c *Collector) ExportMetrics() []types.Metric {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Use WritePrometheus to get all metrics in Prometheus format
	var buf bytes.Buffer
	metrics.WritePrometheus(&buf, true)
	promData := buf.String()

	// Parse Prometheus format and convert to types.Metric
	return c.parsePrometheusMetrics(promData)
}

// parsePrometheusMetrics parses Prometheus format and converts to types.Metric
func (c *Collector) parsePrometheusMetrics(promData string) []types.Metric {
	var result []types.Metric
	now := time.Now()

	// Regular expression for parsing Prometheus metrics
	// Format: metric_name{label1="value1",label2="value2"} value
	// or: metric_name value (without labels)
	metricRegex := regexp.MustCompile(`^([a-zA-Z_:][a-zA-Z0-9_:]*)\s*(?:\{([^}]*)\})?\s+([0-9.+-eE]+)(?:\s+(\d+))?\s*$`)

	lines := strings.Split(promData, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		matches := metricRegex.FindStringSubmatch(line)
		if len(matches) < 4 {
			continue
		}

		metricName := matches[1]
		labelsStr := matches[2]
		valueStr := matches[3]

		// Skip VictoriaMetrics and Go runtime service metrics
		if strings.HasPrefix(metricName, "vm_") ||
			strings.HasPrefix(metricName, "process_") ||
			strings.HasPrefix(metricName, "go_") {
			continue
		}

		// Filter only metrics from our namespace
		if !strings.HasPrefix(metricName, c.namespace+"_") {
			continue
		}

		// Parse value
		value, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			continue
		}

		// Parse labels
		labels := c.parseLabels(labelsStr)

		// Add custom labels if they don't exist
		for k, v := range c.customLabels {
			if _, exists := labels[k]; !exists {
				labels[k] = v
			}
		}

		// Determine metric type by name
		metricType := types.MetricTypeGauge
		if strings.HasSuffix(metricName, "_total") && !strings.Contains(metricName, "_bucket") {
			metricType = types.MetricTypeCounter
		} else if strings.Contains(metricName, "_bucket") {
			metricType = types.MetricTypeHistogram
		} else if strings.HasSuffix(metricName, "_sum") || strings.HasSuffix(metricName, "_count") {
			metricType = types.MetricTypeHistogram
		}

		// Add metric
		result = append(result, types.Metric{
			Name:      metricName,
			Value:     value,
			Timestamp: now,
			Labels:    labels,
			Type:      metricType,
		})
	}

	return result
}

// parseLabels parses a labels string in format "key1=\"value1\",key2=\"value2\""
func (c *Collector) parseLabels(labelsStr string) map[string]string {
	labels := make(map[string]string)
	if labelsStr == "" {
		return labels
	}

	// Regular expression for parsing labels
	labelRegex := regexp.MustCompile(`([a-zA-Z_][a-zA-Z0-9_]*)\s*=\s*"([^"]*)"`)
	matches := labelRegex.FindAllStringSubmatch(labelsStr, -1)

	for _, match := range matches {
		if len(match) >= 3 {
			key := match[1]
			value := match[2]
			// Skip service label le, which is used for Prometheus histogram buckets
			if key == "le" {
				continue
			}
			labels[key] = value
		}
	}

	return labels
}
