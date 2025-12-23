package shutdown

import (
	"context"

	"github.com/vmprober/vmprober/internal/adapter"
	"github.com/vmprober/vmprober/internal/normalizer"
	"github.com/vmprober/vmprober/internal/observability"
	"github.com/vmprober/vmprober/internal/scheduler"
	"github.com/vmprober/vmprober/internal/server"
	"github.com/vmprober/vmprober/internal/wal"
)

// ServerComponent обертка для HTTP сервера
type ServerComponent struct {
	server *server.Server
}

// NewServerComponent создает новую обертку для сервера
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
	return 1 // Высокий приоритет - останавливаем первым
}

// SchedulerComponent обертка для планировщика
type SchedulerComponent struct {
	scheduler *scheduler.Scheduler
}

// NewSchedulerComponent создает новую обертку для планировщика
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

// WALComponent обертка для WAL системы
type WALComponent struct {
	wal wal.WALManager
}

// NewWALComponent создает новую обертку для WAL
func NewWALComponent(w wal.WALManager) ShutdownComponent {
	return &WALComponent{wal: w}
}

func (c *WALComponent) Shutdown(ctx context.Context) error {
	// Flush перед закрытием
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

// AdapterComponent обертка для адаптера VictoriaMetrics
type AdapterComponent struct {
	adapter adapter.VictoriaMetricsAdapter
}

// NewAdapterComponent создает новую обертку для адаптера
func NewAdapterComponent(a adapter.VictoriaMetricsAdapter) ShutdownComponent {
	return &AdapterComponent{adapter: a}
}

func (c *AdapterComponent) Shutdown(ctx context.Context) error {
	// Flush перед остановкой
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

// NormalizerComponent обертка для нормализатора
type NormalizerComponent struct {
	normalizer normalizer.Normalizer
}

// NewNormalizerComponent создает новую обертку для нормализатора
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

// ObservabilityComponent обертка для observability менеджера
type ObservabilityComponent struct {
	obs observability.ObservabilityManager
}

// NewObservabilityComponent создает новую обертку для observability
func NewObservabilityComponent(o observability.ObservabilityManager) ShutdownComponent {
	return &ObservabilityComponent{obs: o}
}

func (c *ObservabilityComponent) Shutdown(ctx context.Context) error {
	return c.obs.Stop(ctx)
}

func (c *ObservabilityComponent) Name() string {
	return "observability"
}

func (c *ObservabilityComponent) Priority() int {
	return 6 // Низкий приоритет - останавливаем последним
}

