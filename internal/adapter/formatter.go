package adapter

import (
	"fmt"
	"strings"

	"github.com/vmprober/vmprober/internal/types"
)

// Formatter интерфейс для форматирования метрик
type Formatter interface {
	Format(metrics []types.Metric) ([]byte, error)
}

// PrometheusFormatter форматирует метрики в Prometheus text format
type PrometheusFormatter struct{}

// NewPrometheusFormatter создает новый Prometheus formatter
func NewPrometheusFormatter() Formatter {
	return &PrometheusFormatter{}
}

// Format форматирует метрики в Prometheus text format
func (f *PrometheusFormatter) Format(metrics []types.Metric) ([]byte, error) {
	var builder strings.Builder

	for _, metric := range metrics {
		// Формирование метки
		labels := make([]string, 0, len(metric.Labels))
		for k, v := range metric.Labels {
			labels = append(labels, fmt.Sprintf(`%s="%s"`, escapeLabel(k), escapeLabel(v)))
		}

		labelStr := ""
		if len(labels) > 0 {
			labelStr = "{" + strings.Join(labels, ",") + "}"
		}

		// Формирование строки метрики
		switch metric.Type {
		case types.MetricTypeCounter:
			builder.WriteString(fmt.Sprintf("%s%s %f\n", metric.Name, labelStr, metric.Value))
		case types.MetricTypeGauge:
			builder.WriteString(fmt.Sprintf("%s%s %f\n", metric.Name, labelStr, metric.Value))
		case types.MetricTypeHistogram:
			// Запись счетчика
			builder.WriteString(fmt.Sprintf("%s_count%s %d\n", metric.Name, labelStr, metric.Count))
			// Запись суммы
			builder.WriteString(fmt.Sprintf("%s_sum%s %f\n", metric.Name, labelStr, metric.Sum))
			// Запись бакетов
			for i, bucket := range metric.Buckets {
				bucketLabel := labelStr
				if bucketLabel == "" {
					bucketLabel = fmt.Sprintf(`{le="%f"}`, bucket)
				} else {
					bucketLabel = strings.TrimSuffix(labelStr, "}") + fmt.Sprintf(`,le="%f"}`, bucket)
				}
				builder.WriteString(fmt.Sprintf("%s_bucket%s %d\n", metric.Name, bucketLabel, metric.Count))
				if i == len(metric.Buckets)-1 {
					// Последний бакет с +Inf
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
			// Запись счетчика
			builder.WriteString(fmt.Sprintf("%s_count%s %d\n", metric.Name, labelStr, metric.Count))
			// Запись суммы
			builder.WriteString(fmt.Sprintf("%s_sum%s %f\n", metric.Name, labelStr, metric.Sum))
			// Запись квантилей
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

// escapeLabel экранирует метки для Prometheus формата
func escapeLabel(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	return s
}

