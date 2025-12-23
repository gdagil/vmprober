package metrics

import (
	"context"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/vmprober/vmprober/internal/types"
)

// Collector собирает и экспортирует метрики
type Collector struct {
	probeSuccess      *prometheus.CounterVec
	probeFailure      *prometheus.CounterVec
	probeRTT          *prometheus.HistogramVec
	probeAttempts     *prometheus.CounterVec
	lastSuccessTime   *prometheus.GaugeVec
	// Метрики джобов
	jobsTotal         prometheus.Gauge
	jobsRunning       prometheus.Gauge
	jobsQueued        prometheus.Gauge
	jobsCompleted     prometheus.Gauge
	jobsFailed        prometheus.Gauge
	mu                sync.RWMutex
	namespace         string
	enableJobMetrics  bool
}

// NewCollector создает новый коллектор метрик
func NewCollector(namespace string, enableJobMetrics bool) *Collector {
	if namespace == "" {
		namespace = "vmprober"
	}

	collector := &Collector{
		namespace:        namespace,
		enableJobMetrics: enableJobMetrics,
		probeSuccess: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "probe_success_total",
				Help:      "Total number of successful probes",
			},
			[]string{"target", "protocol"},
		),
		probeFailure: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "probe_failure_total",
				Help:      "Total number of failed probes",
			},
			[]string{"target", "protocol"},
		),
		probeRTT: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "probe_rtt_seconds",
				Help:      "Probe round-trip time in seconds",
				Buckets:   prometheus.ExponentialBuckets(0.001, 2, 10),
			},
			[]string{"target", "protocol"},
		),
		probeAttempts: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "probe_attempts_total",
				Help:      "Total number of probe attempts",
			},
			[]string{"target", "protocol"},
		),
		lastSuccessTime: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "probe_last_success_timestamp",
				Help:      "Timestamp of last successful probe",
			},
			[]string{"target", "protocol"},
		),
	}

	// Инициализация метрик джобов только если они включены
	if enableJobMetrics {
		collector.jobsTotal = promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "jobs_total",
				Help:      "Total number of jobs",
			},
		)
		collector.jobsRunning = promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "jobs_running",
				Help:      "Number of currently running jobs",
			},
		)
		collector.jobsQueued = promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "jobs_queued",
				Help:      "Number of jobs in queue",
			},
		)
		collector.jobsCompleted = promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "jobs_completed_total",
				Help:      "Total number of completed jobs",
			},
		)
		collector.jobsFailed = promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "jobs_failed_total",
				Help:      "Total number of failed jobs",
			},
		)
	}

	return collector
}

// Record записывает результат пробы
func (c *Collector) Record(ctx context.Context, result *types.ProbeResult) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	target := result.TargetIP
	if target == "" {
		target = "unknown"
	}
	protocol := string(result.Protocol)

	c.probeAttempts.WithLabelValues(target, protocol).Inc()

	if result.Success {
		c.probeSuccess.WithLabelValues(target, protocol).Inc()
		c.probeRTT.WithLabelValues(target, protocol).Observe(result.RTT.Seconds())
		c.lastSuccessTime.WithLabelValues(target, protocol).Set(float64(result.Timestamp.Unix()))
	} else {
		c.probeFailure.WithLabelValues(target, protocol).Inc()
	}

	return nil
}

// UpdateJobMetrics обновляет метрики джобов на основе статистики планировщика
func (c *Collector) UpdateJobMetrics(totalJobs, runningJobs, queuedJobs int, completedJobs, failedJobs int64) {
	if !c.enableJobMetrics {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.jobsTotal.Set(float64(totalJobs))
	c.jobsRunning.Set(float64(runningJobs))
	c.jobsQueued.Set(float64(queuedJobs))
	c.jobsCompleted.Set(float64(completedJobs))
	c.jobsFailed.Set(float64(failedJobs))
}

// GetRegistry возвращает Prometheus registry
func (c *Collector) GetRegistry() *prometheus.Registry {
	return prometheus.DefaultRegisterer.(*prometheus.Registry)
}

