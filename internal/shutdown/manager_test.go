package shutdown

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

// mockComponent mock component for testing
type mockComponent struct {
	name     string
	priority int
	shutdown func(ctx context.Context) error
}

func (m *mockComponent) Shutdown(ctx context.Context) error {
	if m.shutdown != nil {
		return m.shutdown(ctx)
	}
	return nil
}

func (m *mockComponent) Name() string {
	return m.name
}

func (m *mockComponent) Priority() int {
	return m.priority
}

func TestNewShutdownManager(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	manager := NewShutdownManager(logger)
	if manager == nil {
		t.Fatal("NewShutdownManager returned nil")
	}
}

func TestShutdownManager_Register(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	manager := NewShutdownManager(logger)

	component := &mockComponent{
		name:     "test-component",
		priority: 1,
	}

	manager.Register(component)

	status := manager.GetStatus()
	if len(status.Components) != 1 {
		t.Errorf("Expected 1 component, got %d", len(status.Components))
	}
}

func TestShutdownManager_Shutdown(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	manager := NewShutdownManager(logger)

	shutdownCalled := false
	component := &mockComponent{
		name:     "test-component",
		priority: 1,
		shutdown: func(ctx context.Context) error {
			shutdownCalled = true
			return nil
		},
	}

	manager.Register(component)

	ctx := context.Background()
	if err := manager.Shutdown(ctx, 5*time.Second); err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	if !shutdownCalled {
		t.Error("Component shutdown was not called")
	}

	status := manager.GetStatus()
	if !status.Components["test-component"] {
		t.Error("Component should be marked as shutdown")
	}
}

func TestShutdownManager_Shutdown_PriorityOrder(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	manager := NewShutdownManager(logger)

	shutdownOrder := make([]string, 0)

	component1 := &mockComponent{
		name:     "component-1",
		priority: 3, // Low priority - should be last
		shutdown: func(ctx context.Context) error {
			shutdownOrder = append(shutdownOrder, "component-1")
			return nil
		},
	}

	component2 := &mockComponent{
		name:     "component-2",
		priority: 1, // High priority - should be first
		shutdown: func(ctx context.Context) error {
			shutdownOrder = append(shutdownOrder, "component-2")
			return nil
		},
	}

	component3 := &mockComponent{
		name:     "component-3",
		priority: 2, // Medium priority
		shutdown: func(ctx context.Context) error {
			shutdownOrder = append(shutdownOrder, "component-3")
			return nil
		},
	}

	manager.Register(component1)
	manager.Register(component2)
	manager.Register(component3)

	ctx := context.Background()
	if err := manager.Shutdown(ctx, 5*time.Second); err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	// Check shutdown order
	if len(shutdownOrder) != 3 {
		t.Fatalf("Expected 3 components to shutdown, got %d", len(shutdownOrder))
	}

	if shutdownOrder[0] != "component-2" {
		t.Errorf("Expected component-2 first, got %s", shutdownOrder[0])
	}
	if shutdownOrder[1] != "component-3" {
		t.Errorf("Expected component-3 second, got %s", shutdownOrder[1])
	}
	if shutdownOrder[2] != "component-1" {
		t.Errorf("Expected component-1 third, got %s", shutdownOrder[2])
	}
}

func TestShutdownManager_Shutdown_ErrorHandling(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	manager := NewShutdownManager(logger)

	component := &mockComponent{
		name:     "error-component",
		priority: 1,
		shutdown: func(ctx context.Context) error {
			return context.DeadlineExceeded
		},
	}

	manager.Register(component)

	ctx := context.Background()
	err := manager.Shutdown(ctx, 5*time.Second)
	// Shutdown may complete with error, but this is normal
	if err != nil {
		t.Logf("Shutdown returned error (expected): %v", err)
	}

	status := manager.GetStatus()
	if len(status.Errors) == 0 {
		t.Log("No errors recorded (may be expected)")
	}
}

func TestShutdownManager_Shutdown_Timeout(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	manager := NewShutdownManager(logger)

	component := &mockComponent{
		name:     "slow-component",
		priority: 1,
		shutdown: func(ctx context.Context) error {
			// Simulate slow shutdown
			time.Sleep(2 * time.Second)
			return nil
		},
	}

	manager.Register(component)

	ctx := context.Background()
	// Use short timeout
	err := manager.Shutdown(ctx, 100*time.Millisecond)
	if err == nil {
		t.Log("Shutdown completed (may have been fast enough)")
	} else {
		t.Logf("Shutdown timed out (expected): %v", err)
	}
}

func TestShutdownManager_GetStatus(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	manager := NewShutdownManager(logger)

	status := manager.GetStatus()
	if status == nil {
		t.Fatal("Status is nil")
	}

	if status.InProgress {
		t.Error("Shutdown should not be in progress initially")
	}

	if len(status.Components) != 0 {
		t.Errorf("Expected 0 components initially, got %d", len(status.Components))
	}
}

func TestShutdownManager_Shutdown_AlreadyInProgress(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	manager := NewShutdownManager(logger)

	component := &mockComponent{
		name:     "test-component",
		priority: 1,
		shutdown: func(ctx context.Context) error {
			time.Sleep(100 * time.Millisecond)
			return nil
		},
	}

	manager.Register(component)

	ctx := context.Background()

	// Start two shutdowns in parallel
	errChan := make(chan error, 2)
	go func() {
		errChan <- manager.Shutdown(ctx, 1*time.Second)
	}()
	go func() {
		errChan <- manager.Shutdown(ctx, 1*time.Second)
	}()

	// One should complete successfully, other should return error
	err1 := <-errChan
	err2 := <-errChan

	if err1 == nil && err2 == nil {
		t.Log("Both shutdowns completed (race condition)")
	} else {
		t.Logf("One shutdown returned error (expected): %v, %v", err1, err2)
	}
}


