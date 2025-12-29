package normalizer

import (
	"runtime"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/gdagil/vmprober/internal/types"
)

// DataEnricher enriches events with additional information
type DataEnricher struct {
	logger *logrus.Logger
}

// NewDataEnricher creates a new enricher
func NewDataEnricher(logger *logrus.Logger) *DataEnricher {
	return &DataEnricher{
		logger: logger,
	}
}

// Enrich enriches event with data
func (e *DataEnricher) Enrich(event *types.NormalizedEvent) {
	// Add system metadata
	if event.Metadata == nil {
		event.Metadata = make(map[string]interface{})
	}

	event.Metadata["processed_at"] = time.Now()
	event.Metadata["go_version"] = runtime.Version()
	event.Metadata["num_goroutines"] = runtime.NumGoroutine()

	// Add additional labels
	if event.Labels == nil {
		event.Labels = make(map[string]string)
	}

	// Add timestamps
	event.Labels["timestamp"] = event.Timestamp.Format(time.RFC3339)
}

