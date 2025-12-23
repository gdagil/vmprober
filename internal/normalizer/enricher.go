package normalizer

import (
	"fmt"
	"runtime"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vmprober/vmprober/internal/types"
)

// DataEnricher обогащает события дополнительной информацией
type DataEnricher struct {
	logger *logrus.Logger
}

// NewDataEnricher создает новый enricher
func NewDataEnricher(logger *logrus.Logger) *DataEnricher {
	return &DataEnricher{
		logger: logger,
	}
}

// Enrich обогащает событие данными
func (e *DataEnricher) Enrich(event *types.NormalizedEvent) {
	// Добавление системных метаданных
	if event.Metadata == nil {
		event.Metadata = make(map[string]interface{})
	}

	event.Metadata["processed_at"] = time.Now()
	event.Metadata["go_version"] = runtime.Version()
	event.Metadata["num_goroutines"] = runtime.NumGoroutine()

	// Добавление дополнительных меток
	if event.Labels == nil {
		event.Labels = make(map[string]string)
	}

	// Добавление временных меток
	event.Labels["timestamp"] = event.Timestamp.Format(time.RFC3339)
	event.Labels["hour"] = fmt.Sprintf("%d", event.Timestamp.Hour())
	event.Labels["day_of_week"] = event.Timestamp.Weekday().String()
}

