package shutdown

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// ShutdownComponent interface for components with graceful shutdown
type ShutdownComponent interface {
	// Shutdown performs graceful shutdown of the component
	Shutdown(ctx context.Context) error

	// Name returns the component name
	Name() string

	// Priority returns shutdown priority (lower = higher priority)
	Priority() int
}

// ShutdownManager manages graceful shutdown
type ShutdownManager interface {
	// Register registers a component for shutdown
	Register(component ShutdownComponent)

	// Shutdown performs graceful shutdown of all components
	Shutdown(ctx context.Context, timeout time.Duration) error

	// GetStatus returns shutdown status
	GetStatus() *ShutdownStatus
}

// ShutdownStatus shutdown status
type ShutdownStatus struct {
	InProgress bool              `json:"in_progress"`
	Components map[string]bool   `json:"components"`
	StartTime  time.Time         `json:"start_time,omitempty"`
	EndTime    time.Time         `json:"end_time,omitempty"`
	Duration   time.Duration     `json:"duration,omitempty"`
	Errors     map[string]string `json:"errors,omitempty"`
}

// DefaultShutdownManager shutdown manager implementation
type DefaultShutdownManager struct {
	components []ShutdownComponent
	mu         sync.RWMutex
	logger     *logrus.Logger
	status     *ShutdownStatus
}

// NewShutdownManager creates a new shutdown manager
func NewShutdownManager(logger *logrus.Logger) ShutdownManager {
	return &DefaultShutdownManager{
		components: make([]ShutdownComponent, 0),
		logger:     logger,
		status: &ShutdownStatus{
			Components: make(map[string]bool),
			Errors:     make(map[string]string),
		},
	}
}

// Register registers a component for shutdown
func (m *DefaultShutdownManager) Register(component ShutdownComponent) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.components = append(m.components, component)
	m.status.Components[component.Name()] = false
}

// Shutdown performs graceful shutdown of all components
func (m *DefaultShutdownManager) Shutdown(ctx context.Context, timeout time.Duration) error {
	m.mu.Lock()
	if m.status.InProgress {
		m.mu.Unlock()
		return fmt.Errorf("shutdown already in progress")
	}
	m.status.InProgress = true
	m.status.StartTime = time.Now()
	m.mu.Unlock()

	defer func() {
		m.mu.Lock()
		m.status.InProgress = false
		m.status.EndTime = time.Now()
		m.status.Duration = time.Since(m.status.StartTime)
		m.mu.Unlock()
	}()

	m.logger.Info("Starting graceful shutdown")

	// Create context with timeout
	shutdownCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Sort components by priority
	m.mu.RLock()
	components := make([]ShutdownComponent, len(m.components))
	copy(components, m.components)
	m.mu.RUnlock()

	// Sort by priority (lower = higher priority)
	sort.Slice(components, func(i, j int) bool {
		return components[i].Priority() < components[j].Priority()
	})

	// Shutdown components by priority
	for _, component := range components {
		m.logger.WithField("component", component.Name()).Info("Shutting down component")

		componentCtx, componentCancel := context.WithTimeout(shutdownCtx, timeout/2)
		err := component.Shutdown(componentCtx)
		componentCancel()

		m.mu.Lock()
		m.status.Components[component.Name()] = true
		if err != nil {
			m.status.Errors[component.Name()] = err.Error()
			m.logger.WithError(err).WithField("component", component.Name()).Error("Failed to shutdown component")
		}
		m.mu.Unlock()

		// Check context
		if shutdownCtx.Err() != nil {
			m.logger.Warn("Shutdown timeout exceeded")
			return shutdownCtx.Err()
		}
	}

	m.logger.Info("Graceful shutdown completed")
	return nil
}

// GetStatus returns shutdown status
func (m *DefaultShutdownManager) GetStatus() *ShutdownStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := *m.status
	status.Components = make(map[string]bool)
	for k, v := range m.status.Components {
		status.Components[k] = v
	}
	status.Errors = make(map[string]string)
	for k, v := range m.status.Errors {
		status.Errors[k] = v
	}

	return &status
}
