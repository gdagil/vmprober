package shutdown

import (
	"context"

	"github.com/gdagil/vmprober/internal/adapter"
	"github.com/gdagil/vmprober/internal/normalizer"
	"github.com/gdagil/vmprober/internal/observability"
	"github.com/gdagil/vmprober/internal/scheduler"
	"github.com/gdagil/vmprober/internal/server"
	"github.com/gdagil/vmprober/internal/wal"
)

// ServerComponent wrapper for HTTP server
type ServerComponent struct {
	server *server.Server
}

// NewServerComponent creates a new server wrapper
func NewServerComponent(s *server.Server) ShutdownComponent {
	return &ServerComponent{server: s}
}

func (c *ServerComponent) Shutdown(ctx context.Context) error {
	return c.server.Stop(ctx)
}

func (c *ServerComponent) Name() string {
	return "http_server"
}

func (c *ServerComponent) Priority() int {
	return 1 // High priority - stop first
}

// SchedulerComponent wrapper for scheduler
type SchedulerComponent struct {
	scheduler *scheduler.Scheduler
}

// NewSchedulerComponent creates a new scheduler wrapper
func NewSchedulerComponent(s *scheduler.Scheduler) ShutdownComponent {
	return &SchedulerComponent{scheduler: s}
}

func (c *SchedulerComponent) Shutdown(ctx context.Context) error {
	return c.scheduler.Stop(ctx)
}

func (c *SchedulerComponent) Name() string {
	return "scheduler"
}

func (c *SchedulerComponent) Priority() int {
	return 2
}

// WALComponent wrapper for WAL system
type WALComponent struct {
	wal wal.WALManager
}

// NewWALComponent creates a new WAL wrapper
func NewWALComponent(w wal.WALManager) ShutdownComponent {
	return &WALComponent{wal: w}
}

func (c *WALComponent) Shutdown(ctx context.Context) error {
	// Flush before closing
	if err := c.wal.Flush(ctx); err != nil {
		return err
	}
	return c.wal.Close(ctx)
}

func (c *WALComponent) Name() string {
	return "wal_system"
}

func (c *WALComponent) Priority() int {
	return 3
}

// AdapterComponent wrapper for VictoriaMetrics adapter
type AdapterComponent struct {
	adapter adapter.VictoriaMetricsAdapter
}

// NewAdapterComponent creates a new adapter wrapper
func NewAdapterComponent(a adapter.VictoriaMetricsAdapter) ShutdownComponent {
	return &AdapterComponent{adapter: a}
}

func (c *AdapterComponent) Shutdown(ctx context.Context) error {
	// Flush before stopping
	if err := c.adapter.Flush(ctx); err != nil {
		return err
	}
	return c.adapter.Stop(ctx)
}

func (c *AdapterComponent) Name() string {
	return "victoriametrics_adapter"
}

func (c *AdapterComponent) Priority() int {
	return 4
}

// NormalizerComponent wrapper for normalizer
type NormalizerComponent struct {
	normalizer normalizer.Normalizer
}

// NewNormalizerComponent creates a new normalizer wrapper
func NewNormalizerComponent(n normalizer.Normalizer) ShutdownComponent {
	return &NormalizerComponent{normalizer: n}
}

func (c *NormalizerComponent) Shutdown(ctx context.Context) error {
	return c.normalizer.Close(ctx)
}

func (c *NormalizerComponent) Name() string {
	return "normalizer"
}

func (c *NormalizerComponent) Priority() int {
	return 5
}

// ObservabilityComponent wrapper for observability manager.
type ObservabilityComponent struct {
	obs observability.Manager
}

// NewObservabilityComponent creates a new observability wrapper.
func NewObservabilityComponent(o observability.Manager) ShutdownComponent {
	return &ObservabilityComponent{obs: o}
}

func (c *ObservabilityComponent) Shutdown(ctx context.Context) error {
	return c.obs.Stop(ctx)
}

func (c *ObservabilityComponent) Name() string {
	return "observability"
}

func (c *ObservabilityComponent) Priority() int {
	return 6 // Low priority - stop last
}
