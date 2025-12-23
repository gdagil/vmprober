package config

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vmprober/vmprober/internal/types"
	"gopkg.in/yaml.v3"
)

// Manager управляет конфигурацией VMProber
type Manager struct {
	config     *Config
	configPath string
	mu         sync.RWMutex
	logger     *logrus.Logger
	subscribers []chan types.ConfigUpdate
	watcher    *watcher
}

// NewManager создает новый менеджер конфигурации
func NewManager(configPath string, logger *logrus.Logger) *Manager {
	return &Manager{
		configPath:  configPath,
		logger:      logger,
		subscribers: make([]chan types.ConfigUpdate, 0),
	}
}

// Load загружает конфигурацию из файла
func (m *Manager) Load(ctx context.Context) (*Config, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	file, err := os.Open(m.configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Установка метаданных
	cfg.Source = m.configPath
	cfg.Timestamp = time.Now()
	cfg.Hash = m.computeHash(data)

	// Валидация
	if err := m.validate(&cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	// Применение значений по умолчанию
	m.applyDefaults(&cfg)

	oldConfig := m.config
	m.config = &cfg

	// Уведомление подписчиков об изменении
	if oldConfig != nil {
		m.notifySubscribers(types.ConfigUpdate{
			Type:      types.UpdateTypeFull,
			OldConfig: oldConfig,
			NewConfig: &cfg,
			Timestamp: time.Now(),
			Source:    m.configPath,
		})
	}

	m.logger.WithFields(logrus.Fields{
		"path":  m.configPath,
		"hash":  cfg.Hash[:8],
	}).Info("Configuration loaded successfully")

	return &cfg, nil
}

// Get возвращает текущую конфигурацию
func (m *Manager) Get() *Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}

// Watch подписывается на изменения конфигурации
func (m *Manager) Watch(ctx context.Context) (<-chan types.ConfigUpdate, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	ch := make(chan types.ConfigUpdate, 10)
	m.subscribers = append(m.subscribers, ch)

	// Запуск watcher если еще не запущен
	if m.watcher == nil {
		m.watcher = newWatcher(m.configPath, m.logger)
		go m.watcher.watch(ctx, func() {
			if _, err := m.Load(ctx); err != nil {
				m.logger.WithError(err).Error("Failed to reload configuration")
			}
		})
	}

	return ch, nil
}

// validate валидирует конфигурацию
func (m *Manager) validate(cfg *Config) error {
	if cfg.Listen.Port <= 0 || cfg.Listen.Port > 65535 {
		return fmt.Errorf("invalid listen port: %d", cfg.Listen.Port)
	}

	if cfg.Scheduler.Concurrent <= 0 {
		return fmt.Errorf("scheduler.concurrent must be > 0")
	}

	if cfg.Scheduler.RPSLimit <= 0 {
		return fmt.Errorf("scheduler.rps_limit must be > 0")
	}

	return nil
}

// applyDefaults применяет значения по умолчанию
func (m *Manager) applyDefaults(cfg *Config) {
	if cfg.Listen.Host == "" {
		cfg.Listen.Host = "0.0.0.0"
	}

	if cfg.Listen.Port == 0 {
		cfg.Listen.Port = 8429
	}

	if cfg.Pull.Path == "" {
		cfg.Pull.Path = "/metrics"
	}

	if cfg.Metrics.Namespace == "" {
		cfg.Metrics.Namespace = "vmprober"
	}

	// По умолчанию метрики джобов включены
	if cfg.Metrics.EnableJobMetrics == nil {
		enabled := true
		cfg.Metrics.EnableJobMetrics = &enabled
	}

	if cfg.Scheduler.QueueSize == 0 {
		cfg.Scheduler.QueueSize = 10000
	}

	if cfg.WAL.Dir == "" {
		cfg.WAL.Dir = "/var/lib/vmprober/wal"
	}
}

// computeHash вычисляет хеш конфигурации
func (m *Manager) computeHash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// notifySubscribers уведомляет подписчиков об изменении
func (m *Manager) notifySubscribers(update types.ConfigUpdate) {
	for _, ch := range m.subscribers {
		select {
		case ch <- update:
		default:
			// Пропуск если канал заполнен
		}
	}
}

// watcher отслеживает изменения файла конфигурации
type watcher struct {
	path   string
	logger *logrus.Logger
}

func newWatcher(path string, logger *logrus.Logger) *watcher {
	return &watcher{
		path:   path,
		logger: logger,
	}
}

func (w *watcher) watch(ctx context.Context, reload func()) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	var lastModTime time.Time
	stat, err := os.Stat(w.path)
	if err == nil {
		lastModTime = stat.ModTime()
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			stat, err := os.Stat(w.path)
			if err != nil {
				continue
			}

			if stat.ModTime().After(lastModTime) {
				lastModTime = stat.ModTime()
				w.logger.Info("Configuration file changed, reloading...")
				reload()
			}
		}
	}
}

