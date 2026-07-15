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

	// NB: no per-event labels here. A label with per-event value (such as the
	// former "timestamp") turns every probe into a brand-new time series —
	// unbounded cardinality in VictoriaMetrics. The event time already travels
	// in event.Timestamp.
}
