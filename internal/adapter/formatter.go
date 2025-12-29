package adapter

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gdagil/vmprober/internal/types"
)

// Formatter interface for formatting metrics
type Formatter interface {
	Format(metrics []types.Metric) ([]byte, error)
}

// PrometheusFormatter formats metrics in Prometheus text format
type PrometheusFormatter struct{}

// NewPrometheusFormatter creates a new Prometheus formatter
func NewPrometheusFormatter() Formatter {
	return &PrometheusFormatter{}
}

// Format formats metrics in Prometheus text format
func (f *PrometheusFormatter) Format(metrics []types.Metric) ([]byte, error) {
	var builder strings.Builder

	for _, metric := range metrics {
		// Format label
		labels := make([]string, 0, len(metric.Labels))
		for k, v := range metric.Labels {
			labels = append(labels, fmt.Sprintf(`%s="%s"`, escapeLabel(k), escapeLabel(v)))
		}

		labelStr := ""
		if len(labels) > 0 {
			labelStr = "{" + strings.Join(labels, ",") + "}"
		}

		// Format metric string
		switch metric.Type {
		case types.MetricTypeCounter:
			builder.WriteString(fmt.Sprintf("%s%s %f\n", metric.Name, labelStr, metric.Value))
		case types.MetricTypeGauge:
			builder.WriteString(fmt.Sprintf("%s%s %f\n", metric.Name, labelStr, metric.Value))
		case types.MetricTypeHistogram:
			// Write counter
			builder.WriteString(fmt.Sprintf("%s_count%s %d\n", metric.Name, labelStr, metric.Count))
			// Write sum
			builder.WriteString(fmt.Sprintf("%s_sum%s %f\n", metric.Name, labelStr, metric.Sum))
			// Write buckets
			for i, bucket := range metric.Buckets {
				bucketLabel := labelStr
				if bucketLabel == "" {
					bucketLabel = fmt.Sprintf(`{le="%f"}`, bucket)
				} else {
					bucketLabel = strings.TrimSuffix(labelStr, "}") + fmt.Sprintf(`,le="%f"}`, bucket)
				}
				builder.WriteString(fmt.Sprintf("%s_bucket%s %d\n", metric.Name, bucketLabel, metric.Count))
				if i == len(metric.Buckets)-1 {
					// Last bucket with +Inf
					infLabel := labelStr
					if infLabel == "" {
						infLabel = `{le="+Inf"}`
					} else {
						infLabel = strings.TrimSuffix(labelStr, "}") + `,le="+Inf"}`
					}
					builder.WriteString(fmt.Sprintf("%s_bucket%s %d\n", metric.Name, infLabel, metric.Count))
				}
			}
		case types.MetricTypeSummary:
			// Write counter
			builder.WriteString(fmt.Sprintf("%s_count%s %d\n", metric.Name, labelStr, metric.Count))
			// Write sum
			builder.WriteString(fmt.Sprintf("%s_sum%s %f\n", metric.Name, labelStr, metric.Sum))
			// Write quantiles
			for quantile, value := range metric.Quantiles {
				quantileLabel := labelStr
				if quantileLabel == "" {
					quantileLabel = fmt.Sprintf(`{quantile="%f"}`, quantile)
				} else {
					quantileLabel = strings.TrimSuffix(labelStr, "}") + fmt.Sprintf(`,quantile="%f"}`, quantile)
				}
				builder.WriteString(fmt.Sprintf("%s%s %f\n", metric.Name, quantileLabel, value))
			}
		}
	}

	return []byte(builder.String()), nil
}

// escapeLabel escapes labels for Prometheus format
func escapeLabel(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	return s
}

// JSONLineFormatter formats metrics in VictoriaMetrics JSON line format
type JSONLineFormatter struct{}

// NewJSONLineFormatter creates a new JSON line formatter
func NewJSONLineFormatter() Formatter {
	return &JSONLineFormatter{}
}

// Format formats metrics in VictoriaMetrics JSON line format
// Формат: {"metric":{"__name__":"metric_name","label1":"value1"},"values":[1.0],"timestamps":[1549891472010]}
func (f *JSONLineFormatter) Format(metrics []types.Metric) ([]byte, error) {
	var builder strings.Builder
	
	for _, metric := range metrics {
		// Format labels for JSON format
		metricLabels := make(map[string]string)
		metricLabels["__name__"] = metric.Name
		for k, v := range metric.Labels {
			metricLabels[k] = v
		}
		
		// Convert timestamp to milliseconds
		var timestamp int64
		if metric.Timestamp.IsZero() {
			// If timestamp is not set, use current time
			timestamp = time.Now().UnixMilli()
		} else {
			timestamp = metric.Timestamp.UnixMilli()
		}
		
		// Format JSON object
		jsonObj := map[string]interface{}{
			"metric":    metricLabels,
			"values":    []float64{metric.Value},
			"timestamps": []int64{timestamp},
		}
		
		// Serialize to JSON
		jsonBytes, err := json.Marshal(jsonObj)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metric to JSON: %w", err)
		}
		
		builder.Write(jsonBytes)
		builder.WriteString("\n")
	}
	
	return []byte(builder.String()), nil
}


