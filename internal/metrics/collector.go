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
	"github.com/vmprober/vmprober/internal/types"
)

// Collector собирает и экспортирует метрики
type Collector struct {
	// Метрики проб - используем map для хранения метрик с лейблами
	probeSuccess      map[string]*metrics.Counter
	probeFailure      map[string]*metrics.Counter
	probeRTT          map[string]*metrics.Histogram
	probeAttempts     map[string]*metrics.Counter
	// Метрики джобов
	jobsTotal         *metrics.Gauge
	jobsRunning        *metrics.Gauge
	jobsFailed         *metrics.Gauge
	// Значения для метрик джобов
	jobsTotalValue    float64
	jobsRunningValue  float64
	jobsFailedValue   float64
	mu                 sync.RWMutex
	namespace          string
	enableJobMetrics   bool
}

// NewCollector создает новый коллектор метрик
func NewCollector(namespace string, enableJobMetrics bool) *Collector {
	if namespace == "" {
		namespace = "vmprober"
	}

	collector := &Collector{
		namespace:            namespace,
		enableJobMetrics:     enableJobMetrics,
		probeSuccess:         make(map[string]*metrics.Counter),
		probeFailure:         make(map[string]*metrics.Counter),
		probeRTT:             make(map[string]*metrics.Histogram),
		probeAttempts:        make(map[string]*metrics.Counter),
	}

	// Инициализация метрик джобов только если они включены
	// Используем Gauge с callback, который возвращает значение из переменной
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

// getMetricKey создает ключ для метрики с лейблами
// Использует стандартные лейблы Prometheus:
// - instance: hostname:port (например, "google.com:443")
// - target_ip: IP адрес (например, "142.250.109.100")
// - port: порт (например, "443")
// - protocol: протокол (например, "tcp")
func (c *Collector) getMetricKey(name string, instance, targetIP, port, protocol string) string {
	// Если instance пустой, используем targetIP:port
	if instance == "" {
		if port != "" && port != "0" {
			instance = fmt.Sprintf("%s:%s", targetIP, port)
		} else {
			instance = targetIP
		}
	}
	
	// Формируем лейблы согласно стандартам Prometheus
	labels := fmt.Sprintf(`instance="%s",target_ip="%s",port="%s",protocol="%s"`, 
		instance, targetIP, port, protocol)
	return fmt.Sprintf(`%s_%s{%s}`, c.namespace, name, labels)
}

// getOrCreateCounter получает или создает counter метрику
func (c *Collector) getOrCreateCounter(key string) *metrics.Counter {
	if counter, ok := c.probeAttempts[key]; ok {
		return counter
	}
	counter := metrics.NewCounter(key)
	c.probeAttempts[key] = counter
	return counter
}

// getOrCreateSuccessCounter получает или создает counter для успешных проб
func (c *Collector) getOrCreateSuccessCounter(key string) *metrics.Counter {
	if counter, ok := c.probeSuccess[key]; ok {
		return counter
	}
	counter := metrics.NewCounter(key)
	c.probeSuccess[key] = counter
	return counter
}

// getOrCreateFailureCounter получает или создает counter для неудачных проб
func (c *Collector) getOrCreateFailureCounter(key string) *metrics.Counter {
	if counter, ok := c.probeFailure[key]; ok {
		return counter
	}
	counter := metrics.NewCounter(key)
	c.probeFailure[key] = counter
	return counter
}

// getOrCreateHistogram получает или создает histogram метрику
func (c *Collector) getOrCreateHistogram(key string) *metrics.Histogram {
	if histogram, ok := c.probeRTT[key]; ok {
		return histogram
	}
	histogram := metrics.NewHistogram(key)
	c.probeRTT[key] = histogram
	return histogram
}

// Record записывает результат пробы
func (c *Collector) Record(ctx context.Context, result *types.ProbeResult) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Получаем значения для лейблов
	// instance = hostname:port (например, "google.com:443")
	// target_ip = IP адрес, на который фактически подключились (например, "142.250.109.100")
	// port = порт (например, "443")
	// protocol = протокол (например, "tcp")
	
	hostname := result.TargetHost // hostname из конфигурации (например, "google.com")
	targetIP := result.TargetIP    // IP адрес, на который подключились (например, "142.250.109.100")
	port := "0"
	if result.TargetPort > 0 {
		port = fmt.Sprintf("%d", result.TargetPort)
	}
	protocol := string(result.Protocol)

	// Формируем instance как hostname:port
	// Если hostname пустой, используем targetIP
	if hostname == "" {
		hostname = targetIP
	}
	
	// Если targetIP пустой, используем hostname
	if targetIP == "" {
		targetIP = "unknown"
	}
	
	// instance = hostname:port (стандартный формат Prometheus)
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

// UpdateJobMetrics обновляет метрики джобов на основе статистики планировщика
func (c *Collector) UpdateJobMetrics(totalJobs, runningJobs int, failedJobs int64) {
	if !c.enableJobMetrics {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Обновляем значения через переменные (callback будет возвращать эти значения)
	c.jobsTotalValue = float64(totalJobs)
	c.jobsRunningValue = float64(runningJobs)
	c.jobsFailedValue = float64(failedJobs)
}

// ExportMetrics экспортирует все метрики из collector в формат types.Metric
func (c *Collector) ExportMetrics() []types.Metric {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Используем WritePrometheus для получения всех метрик в Prometheus формате
	var buf bytes.Buffer
	metrics.WritePrometheus(&buf, true)
	promData := buf.String()

	// Парсим Prometheus формат и конвертируем в types.Metric
	return c.parsePrometheusMetrics(promData)
}

// parsePrometheusMetrics парсит Prometheus формат и конвертирует в types.Metric
func (c *Collector) parsePrometheusMetrics(promData string) []types.Metric {
	var result []types.Metric
	now := time.Now()

	// Регулярное выражение для парсинга Prometheus метрик
	// Формат: metric_name{label1="value1",label2="value2"} value
	// или: metric_name value (без лейблов)
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

		// Пропускаем служебные метрики VictoriaMetrics и Go runtime
		if strings.HasPrefix(metricName, "vm_") || 
		   strings.HasPrefix(metricName, "process_") || 
		   strings.HasPrefix(metricName, "go_") {
			continue
		}

		// Парсим значение
		value, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			continue
		}

		// Парсим лейблы
		labels := c.parseLabels(labelsStr)

		// Определяем тип метрики по имени
		metricType := types.MetricTypeGauge
		if strings.HasSuffix(metricName, "_total") && !strings.Contains(metricName, "_bucket") {
			metricType = types.MetricTypeCounter
		} else if strings.Contains(metricName, "_bucket") {
			metricType = types.MetricTypeHistogram
		} else if strings.HasSuffix(metricName, "_sum") || strings.HasSuffix(metricName, "_count") {
			metricType = types.MetricTypeHistogram
		}

		// Добавляем метрику
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

// parseLabels парсит строку лейблов в формате "key1=\"value1\",key2=\"value2\""
func (c *Collector) parseLabels(labelsStr string) map[string]string {
	labels := make(map[string]string)
	if labelsStr == "" {
		return labels
	}

	// Регулярное выражение для парсинга лейблов
	labelRegex := regexp.MustCompile(`([a-zA-Z_][a-zA-Z0-9_]*)\s*=\s*"([^"]*)"`)
	matches := labelRegex.FindAllStringSubmatch(labelsStr, -1)

	for _, match := range matches {
		if len(match) >= 3 {
			key := match[1]
			value := match[2]
			labels[key] = value
		}
	}

	return labels
}
