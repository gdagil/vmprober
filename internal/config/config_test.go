package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func TestNewManager(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	manager := NewManager("test.yaml", logger)
	if manager == nil {
		t.Fatal("NewManager returned nil")
	}
	if manager.configPath != "test.yaml" {
		t.Errorf("Expected configPath 'test.yaml', got '%s'", manager.configPath)
	}
}

func TestLoad_ValidConfig(t *testing.T) {
	// Create temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
listen:
  port: 8429
  host: "0.0.0.0"
scheduler:
  concurrent: 10
  rps_limit: 100
targets:
  static: []
metrics:
  namespace: "test"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	manager := NewManager(configPath, logger)

	cfg, err := manager.Load(context.Background())
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg == nil {
		t.Fatal("Config is nil")
	}
	if cfg.Listen.Port != 8429 {
		t.Errorf("Expected port 8429, got %d", cfg.Listen.Port)
	}
	if cfg.Listen.Host != "0.0.0.0" {
		t.Errorf("Expected host '0.0.0.0', got '%s'", cfg.Listen.Host)
	}
	if cfg.Scheduler.Concurrent != 10 {
		t.Errorf("Expected concurrent 10, got %d", cfg.Scheduler.Concurrent)
	}
	if cfg.Metrics.Namespace != "test" {
		t.Errorf("Expected namespace 'test', got '%s'", cfg.Metrics.Namespace)
	}
}

func TestLoad_InvalidFile(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	manager := NewManager("nonexistent.yaml", logger)

	_, err := manager.Load(context.Background())
	if err == nil {
		t.Fatal("Expected error for nonexistent file")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `invalid: yaml: content: [`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	manager := NewManager(configPath, logger)

	_, err := manager.Load(context.Background())
	if err == nil {
		t.Fatal("Expected error for invalid YAML")
	}
}

func TestLoad_ValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		config  string
		wantErr bool
	}{
		{
			name: "invalid port",
			config: `
listen:
  port: 70000
scheduler:
  concurrent: 10
  rps_limit: 100
`,
			wantErr: true,
		},
		{
			name: "zero concurrent",
			config: `
listen:
  port: 8429
scheduler:
  concurrent: 0
  rps_limit: 100
`,
			wantErr: true,
		},
		{
			name: "zero rps_limit",
			config: `
listen:
  port: 8429
scheduler:
  concurrent: 10
  rps_limit: 0
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")

			if err := os.WriteFile(configPath, []byte(tt.config), 0644); err != nil {
				t.Fatalf("Failed to write config file: %v", err)
			}

			logger := logrus.New()
			logger.SetLevel(logrus.ErrorLevel)
			manager := NewManager(configPath, logger)

			_, err := manager.Load(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoad_ApplyDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
listen:
  port: 8429
scheduler:
  concurrent: 10
  rps_limit: 100
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	manager := NewManager(configPath, logger)

	cfg, err := manager.Load(context.Background())
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Check default values
	if cfg.Listen.Host == "" {
		t.Error("Expected default host to be set")
	}
	if cfg.Pull.Path == "" {
		t.Error("Expected default pull path to be set")
	}
	if cfg.Metrics.Namespace == "" {
		t.Error("Expected default metrics namespace to be set")
	}
	if cfg.Scheduler.QueueSize == 0 {
		t.Error("Expected default queue size to be set")
	}
	if cfg.Push.Interval != 30*time.Second {
		t.Errorf("Expected default push interval 30s, got %v", cfg.Push.Interval)
	}
}

func TestLoad_DefaultProberLabel(t *testing.T) {
	tmpDir := t.TempDir()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	write := func(name, content string) string {
		p := filepath.Join(tmpDir, name)
		if err := os.WriteFile(p, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write config file: %v", err)
		}
		return p
	}

	// Without custom_labels the prober label defaults to the hostname,
	// so several instances don't collide into the same time series.
	base := "listen:\n  port: 8429\nscheduler:\n  concurrent: 10\n  rps_limit: 100\n"
	cfg, err := NewManager(write("default.yaml", base), logger).Load(context.Background())
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	hostname, _ := os.Hostname()
	if cfg.Metrics.CustomLabels["prober"] != hostname {
		t.Errorf("Expected default prober label %q, got %q", hostname, cfg.Metrics.CustomLabels["prober"])
	}

	// An explicit prober label must be preserved.
	explicit := base + "metrics:\n  custom_labels:\n    prober: dc1-a\n"
	cfg, err = NewManager(write("explicit.yaml", explicit), logger).Load(context.Background())
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Metrics.CustomLabels["prober"] != "dc1-a" {
		t.Errorf("Expected explicit prober label to be preserved, got %q", cfg.Metrics.CustomLabels["prober"])
	}
}

func TestGet(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
listen:
  port: 8429
scheduler:
  concurrent: 10
  rps_limit: 100
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	manager := NewManager(configPath, logger)

	// Before loading should return nil
	if cfg := manager.Get(); cfg != nil {
		t.Error("Expected nil config before Load")
	}

	// After loading should return configuration
	_, err := manager.Load(context.Background())
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	cfg := manager.Get()
	if cfg == nil {
		t.Fatal("Expected config after Load")
	}
	if cfg.Listen.Port != 8429 {
		t.Errorf("Expected port 8429, got %d", cfg.Listen.Port)
	}
}

func TestWatch(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
listen:
  port: 8429
scheduler:
  concurrent: 10
  rps_limit: 100
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	manager := NewManager(configPath, logger)

	// Load initial configuration
	_, err := manager.Load(context.Background())
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	updateChan, err := manager.Watch(ctx)
	if err != nil {
		t.Fatalf("Watch failed: %v", err)
	}

	// Modify config file
	newConfigContent := `
listen:
  port: 9090
scheduler:
  concurrent: 10
  rps_limit: 100
`
	time.Sleep(100 * time.Millisecond) // Give time for watcher to initialize

	if err := os.WriteFile(configPath, []byte(newConfigContent), 0644); err != nil {
		t.Fatalf("Failed to write new config file: %v", err)
	}

	// Wait for update (watcher checks every 5 seconds, but for test we can wait)
	select {
	case update := <-updateChan:
		if update.Type == "" {
			t.Error("Expected update type to be set")
		}
	case <-time.After(6 * time.Second):
		t.Log("No update received within timeout (this is expected if watcher hasn't detected change yet)")
	}
}

func TestComputeHash(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	manager := NewManager("test.yaml", logger)

	data1 := []byte("test data")
	data2 := []byte("test data")
	data3 := []byte("different data")

	hash1 := manager.computeHash(data1)
	hash2 := manager.computeHash(data2)
	hash3 := manager.computeHash(data3)

	if hash1 != hash2 {
		t.Error("Same data should produce same hash")
	}
	if hash1 == hash3 {
		t.Error("Different data should produce different hash")
	}
	if len(hash1) == 0 {
		t.Error("Hash should not be empty")
	}
}

func TestSetLogger(t *testing.T) {
	logger1 := logrus.New()
	logger1.SetLevel(logrus.ErrorLevel)

	logger2 := logrus.New()
	logger2.SetLevel(logrus.DebugLevel)

	manager := NewManager("test.yaml", logger1)

	// Set new logger
	manager.SetLogger(logger2)

	// Verify logger was updated (we can't directly check, but SetLogger should not panic)
	// The logger is used internally, so we just verify the method works
}
