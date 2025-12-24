package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vmprober/vmprober/internal/adapter"
	"github.com/vmprober/vmprober/internal/config"
	internalmetrics "github.com/vmprober/vmprober/internal/metrics"
	"github.com/vmprober/vmprober/internal/normalizer"
	"github.com/vmprober/vmprober/internal/observability"
	"github.com/vmprober/vmprober/internal/probe"
	"github.com/vmprober/vmprober/internal/scheduler"
	"github.com/vmprober/vmprober/internal/server"
	"github.com/vmprober/vmprober/internal/shutdown"
	"github.com/vmprober/vmprober/internal/types"
	"github.com/vmprober/vmprober/internal/wal"
	"github.com/vmprober/vmprober/pkg/interfaces"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

// setupLogger настраивает логгер на основе конфигурации
func setupLogger(cfg *config.Config, cmdLogLevel string) *logrus.Logger {
	logger := logrus.New()

	// Приоритет: командная строка > конфигурация > по умолчанию
	var level logrus.Level
	var err error

	if cmdLogLevel != "" {
		level, err = logrus.ParseLevel(cmdLogLevel)
		if err != nil {
			level = logrus.InfoLevel
		}
	} else if cfg.Logging.Level != "" {
		level, err = logrus.ParseLevel(cfg.Logging.Level)
		if err != nil {
			level = logrus.InfoLevel
		}
	} else {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	// Настройка формата
	format := cfg.Logging.Format
	switch format {
	case "text":
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
			DisableColors: false,
		})
	case "json":
		logger.SetFormatter(&logrus.JSONFormatter{})
	default:
		// По умолчанию JSON, если формат не указан или неизвестен
		logger.SetFormatter(&logrus.JSONFormatter{})
	}

	// Настройка вывода
	switch cfg.Logging.Output {
	case "stderr":
		logger.SetOutput(os.Stderr)
	case "file":
		if cfg.Logging.File.Path != "" {
			// TODO: Реализовать файловый вывод с ротацией
			// Для простоты пока используем стандартный вывод
			logger.SetOutput(os.Stdout)
		} else {
			logger.SetOutput(os.Stdout)
		}
	default:
		// По умолчанию stdout
		logger.SetOutput(os.Stdout)
	}

	return logger
}

func main() {
	configPath := flag.String("config", "configs/config.yaml.example", "Path to configuration file")
	logLevel := flag.String("log-level", "", "Log level (debug, info, warn, error) - overrides config")
	flag.Parse()

	// Временный логгер для начальной загрузки конфигурации
	tempLogger := logrus.New()
	tempLogger.SetFormatter(&logrus.JSONFormatter{})
	tempLogger.SetLevel(logrus.InfoLevel)

	// Загрузка конфигурации
	cfgManager := config.NewManager(*configPath, tempLogger)
	cfg, err := cfgManager.Load(context.Background())
	if err != nil {
		tempLogger.WithError(err).Fatal("Failed to load configuration")
	}

	// Настройка логгера на основе конфигурации
	logger := setupLogger(cfg, *logLevel)

	// Обновляем логгер в менеджере конфигурации
	cfgManager.SetLogger(logger)

	logger.WithFields(logrus.Fields{
		"version":   Version,
		"buildTime": BuildTime,
		"gitCommit": GitCommit,
	}).Info("Starting VMProber")

	// Создание компонентов
	probeFactory := probe.NewFactory()
	enableJobMetrics := true
	if cfg.Metrics.EnableJobMetrics != nil {
		enableJobMetrics = *cfg.Metrics.EnableJobMetrics
	}
	metricsCollector := internalmetrics.NewCollector(cfg.Metrics.Namespace, enableJobMetrics)
	taskScheduler := scheduler.NewScheduler(logger)
	httpServer := server.NewServer(cfg.Listen.Host, cfg.Listen.Port, logger)

	// Создание WAL системы
	var walManager wal.WALManager
	if cfg.WAL.Dir != "" {
		walManager, err = wal.NewWALManager(&cfg.WAL, logger)
		if err != nil {
			logger.WithError(err).Warn("Failed to initialize WAL, continuing without WAL")
		} else {
			logger.Info("WAL system initialized")
		}
	}

	// Создание нормализатора
	resultNormalizer := normalizer.NewNormalizer(logger)

	// Создание VictoriaMetrics адаптера
	var vmAdapter adapter.VictoriaMetricsAdapter
	if cfg.Push.Enabled {
		vmAdapter, err = adapter.NewVictoriaMetricsAdapter(&cfg.Push, logger)
		if err != nil {
			logger.WithError(err).Warn("Failed to initialize VictoriaMetrics adapter, continuing without push")
		} else {
			if err := vmAdapter.Start(context.Background()); err != nil {
				logger.WithError(err).Warn("Failed to start VictoriaMetrics adapter")
			} else {
				logger.Info("VictoriaMetrics adapter started")
			}
		}

		// Настройка автоматической отправки метрик из VictoriaMetrics библиотеки
		// Используем первый endpoint из конфигурации
		// Примечание: InitPush может не поддерживать формат vminsert напрямую
		// Метрики из библиотеки будут отправляться через адаптер вместе с другими метриками
		// Если нужна прямая отправка, можно использовать правильный формат для single-node VM:
		// metrics.InitPush("http://localhost:8428/api/v1/import/prometheus", 10*time.Second, ...)
		// Но для vminsert лучше использовать адаптер
	}

	// Создание observability менеджера
	obsManager := observability.NewObservabilityManager(&cfg.Observability, logger)
	if err := obsManager.Start(context.Background()); err != nil {
		logger.WithError(err).Warn("Failed to start observability manager")
	}

	// Создание shutdown менеджера
	shutdownManager := shutdown.NewShutdownManager(logger)
	shutdownManager.Register(shutdown.NewServerComponent(httpServer))
	shutdownManager.Register(shutdown.NewSchedulerComponent(taskScheduler))
	if walManager != nil {
		shutdownManager.Register(shutdown.NewWALComponent(walManager))
	}
	if vmAdapter != nil {
		shutdownManager.Register(shutdown.NewAdapterComponent(vmAdapter))
	}
	shutdownManager.Register(shutdown.NewNormalizerComponent(resultNormalizer))
	shutdownManager.Register(shutdown.NewObservabilityComponent(obsManager))

	// Настройка сервера
	httpServer.SetScheduler(taskScheduler)
	// Metrics are now handled by VictoriaMetrics library directly

	// Запуск HTTP сервера
	if err := httpServer.Start(context.Background()); err != nil {
		logger.WithError(err).Fatal("Failed to start HTTP server")
	}

	// Запуск планировщика
	if err := taskScheduler.Start(context.Background()); err != nil {
		logger.WithError(err).Fatal("Failed to start scheduler")
	}

	// Обработка сигналов для graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Главный цикл обработки задач
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Загрузка целей из конфигурации
	for _, targetCfg := range cfg.Targets.Static {
		protocols := targetCfg.GetProtocols()

		// Если протоколы не найдены, пропускаем таргет
		if len(protocols) == 0 {
			logger.Warnf("No protocols specified for target %s:%d (raw: %v), skipping", targetCfg.Host, targetCfg.Port, targetCfg.Protocols)
			continue
		}

		logger.Debugf("Target %s:%d has protocols: %v", targetCfg.Host, targetCfg.Port, protocols)

		// Создаем джоб для каждого протокола
		for _, protocol := range protocols {
			// ID джоба включает протокол для уникальности
			jobID := fmt.Sprintf("%s:%d/%s", targetCfg.Host, targetCfg.Port, protocol)

			target := types.Target{
				ID:       jobID,
				Host:     targetCfg.Host,
				Port:     targetCfg.Port,
				Protocol: protocol,
				Interval: targetCfg.Interval,
				Timeout:  targetCfg.Timeout,
				Labels:   targetCfg.Labels,
				Enabled:  true,
			}

			if target.Interval == 0 {
				target.Interval = 30 * time.Second
			}
			if target.Timeout == 0 {
				target.Timeout = 5 * time.Second
			}

			job := &types.Job{
				ID:        jobID,
				Target:    target,
				NextRun:   time.Now(),
				Interval:  target.Interval,
				Priority:  1,
				CreatedAt: time.Now(),
			}

			if err := taskScheduler.Schedule(ctx, job); err != nil {
				logger.WithError(err).Errorf("Failed to schedule job for %s", jobID)
			}
		}
	}

	// Worker pool для выполнения проб
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case job := <-taskScheduler.GetJobChan():
				go func(j *types.Job) {
					taskScheduler.MarkJobStarted(j.ID)
					executeProbe(ctx, j, probeFactory, metricsCollector, resultNormalizer, walManager, vmAdapter, logger, taskScheduler, cfg.Probes, cfg.Scheduler, cfg)
				}(job)
			}
		}
	}()

	// Обновление метрик джобов периодически
	if enableJobMetrics {
		go func() {
			ticker := time.NewTicker(5 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					stats := taskScheduler.GetStats()
					metricsCollector.UpdateJobMetrics(
						stats.TotalJobs,
						stats.RunningJobs,
						stats.FailedJobs,
					)
				}
			}
		}()
	}

	// Периодическая отправка метрик из collector в vminsert
	if vmAdapter != nil {
		go func() {
			ticker := time.NewTicker(30 * time.Second) // Отправляем каждые 30 секунд
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					// Экспортируем все метрики из collector
					exportedMetrics := metricsCollector.ExportMetrics()
					if len(exportedMetrics) > 0 {
						if err := vmAdapter.Push(ctx, exportedMetrics); err != nil {
							logger.WithError(err).Warn("Failed to push collector metrics to VictoriaMetrics")
						} else {
							logger.Info("Pushed collector metrics to VictoriaMetrics")
						}
					}

					// Экспортируем метрики шедулера
					schedulerMetrics := taskScheduler.ExportMetrics()
					if len(schedulerMetrics) > 0 {
						if err := vmAdapter.Push(ctx, schedulerMetrics); err != nil {
							logger.WithError(err).Warn("Failed to push scheduler metrics to VictoriaMetrics")
						} else {
							logger.Info("Pushed scheduler metrics to VictoriaMetrics")
						}
					}
				}
			}
		}()
	}

	logger.Info("VMProber started successfully")

	// Ожидание сигнала завершения
	<-sigChan
	logger.Info("Shutting down VMProber...")

	// Graceful shutdown через менеджер
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := shutdownManager.Shutdown(shutdownCtx, 30*time.Second); err != nil {
		logger.WithError(err).Error("Error during graceful shutdown")
	}

	logger.Info("VMProber stopped")
}

// buildProbeConfig создает конфигурацию пробы из глобальной конфигурации
func buildProbeConfig(probeType types.ProbeType, probesConfig config.ProbesConfig, schedulerConfig config.SchedulerConfig) interface{} {
	switch probeType {
	case types.ProbeTypeTCP:
		connectTimeout := probesConfig.TCP.ConnectTimeout
		if connectTimeout == 0 {
			if timeout, ok := schedulerConfig.Timeouts["tcp"]; ok {
				connectTimeout = timeout
			} else {
				connectTimeout = 5 * time.Second
			}
		}
		return &probe.TCPConfig{
			ConnectTimeout: connectTimeout,
			ReadTimeout:    connectTimeout,
			WriteTimeout:   connectTimeout,
		}
	case types.ProbeTypeUDP:
		payloadSize := probesConfig.UDP.PayloadSize
		if payloadSize == 0 {
			payloadSize = 64
		}
		responseTimeout := probesConfig.UDP.ResponseTimeout
		if responseTimeout == 0 {
			if timeout, ok := schedulerConfig.Timeouts["udp"]; ok {
				responseTimeout = timeout
			} else {
				responseTimeout = 3 * time.Second
			}
		}
		maxPacketSize := probesConfig.UDP.MaxPacketSize
		if maxPacketSize == 0 {
			maxPacketSize = 1024
		}
		return &probe.UDPConfig{
			PayloadSize:     payloadSize,
			ResponseTimeout: responseTimeout,
			MaxPacketSize:   maxPacketSize,
		}
	case types.ProbeTypeICMP:
		sequenceStart := probesConfig.ICMP.SequenceStart
		if sequenceStart == 0 {
			sequenceStart = 1
		}
		ttl := probesConfig.ICMP.TTL
		if ttl == 0 {
			ttl = 64
		}
		return &probe.ICMPConfig{
			Library:       probesConfig.ICMP.Library,
			SequenceStart: sequenceStart,
			TTL:           ttl,
		}
	default:
		return nil
	}
}

// executeProbe выполняет пробу и перепланирует задачу
func executeProbe(
	ctx context.Context,
	job *types.Job,
	factory interfaces.ProbeFactory,
	collector *internalmetrics.Collector,
	normalizer normalizer.Normalizer,
	walManager wal.WALManager,
	vmAdapter adapter.VictoriaMetricsAdapter,
	logger *logrus.Logger,
	sched *scheduler.Scheduler,
	probesConfig config.ProbesConfig,
	schedulerConfig config.SchedulerConfig,
	cfg *config.Config,
) {
	// Создание конфигурации пробы
	probeConfig := buildProbeConfig(job.Target.Protocol, probesConfig, schedulerConfig)

	// Создание пробы
	probeInstance, err := factory.CreateProbe(job.Target.Protocol, probeConfig)
	if err != nil {
		logger.WithError(err).Errorf("Failed to create probe for %s", job.ID)
		sched.MarkJobFailed(job.ID)
		return
	}
	defer probeInstance.Close()

	// Выполнение пробы
	result, err := probeInstance.Execute(ctx, job.Target)
	if err != nil && result == nil {
		logger.WithError(err).Errorf("Probe execution failed for %s", job.ID)
		sched.MarkJobFailed(job.ID)
		return
	}

	// Нормализация результата
	if result != nil && normalizer != nil {
		event, err := normalizer.Normalize(ctx, result)
		if err != nil {
			logger.WithError(err).Warn("Failed to normalize result")
		} else {
			// Проверка на дубликаты (только если dedup включен)
			var isDup bool
			if cfg.Push.Dedup.Enabled {
				isDup, err = normalizer.Dedup(ctx, event)
				if err != nil {
					logger.WithError(err).Warn("Failed to check for duplicates")
				}
				if isDup {
					logger.Debug("Duplicate event detected, skipping")
				}
			}

			if !isDup {
				// Запись в WAL если доступен
				if walManager != nil {
					record := &types.Record{
						ID:        fmt.Sprintf("%s-%d", event.SeriesID, time.Now().UnixNano()),
						Timestamp: event.Timestamp,
						Type:      "probe_result",
						SeriesID:  event.SeriesID,
						Data: map[string]interface{}{
							"event": event,
						},
						Labels: event.Labels,
					}
					if err := walManager.Write(ctx, record); err != nil {
						logger.WithError(err).Warn("Failed to write to WAL")
					}
				}

				// Отправка в VictoriaMetrics если доступен
				if vmAdapter != nil {
					metrics := make([]types.Metric, 0)
					for name, value := range event.Metrics {
						// Определяем тип метрики по имени
						metricType := types.MetricTypeGauge
						if strings.HasSuffix(name, "_total") {
							metricType = types.MetricTypeCounter
						}

						// Копируем лейблы из события и добавляем custom_labels из конфигурации
						labels := make(map[string]string)
						for k, v := range event.Labels {
							labels[k] = v
						}
						// Добавляем custom_labels из конфигурации (перезаписывают существующие)
						for k, v := range cfg.Metrics.CustomLabels {
							labels[k] = v
						}

						metrics = append(metrics, types.Metric{
							Name:      name,
							Value:     value,
							Timestamp: event.Timestamp,
							Labels:    labels,
							Type:      metricType,
						})
					}
					if len(metrics) > 0 {
						if err := vmAdapter.Push(ctx, metrics); err != nil {
							logger.WithError(err).Warn("Failed to push metrics to VictoriaMetrics")
						}
					}
				}
			}
		}
	}

	// Запись метрик в коллектор (если результат есть)
	if result != nil {
		if err := collector.Record(ctx, result); err != nil {
			logger.WithError(err).Error("Failed to record metrics")
		}
	}

	// Отметка джоба как завершенного или проваленного
	if result != nil && result.Success {
		sched.MarkJobCompleted(job.ID)
	} else {
		// Если result == nil или result.Success == false, считаем провалом
		sched.MarkJobFailed(job.ID)
	}

	// Обновляем NextRun перед перепланированием
	job.NextRun = time.Now().Add(job.Interval)

	// Перепланирование задачи без увеличения счетчика TotalJobs
	if err := sched.Reschedule(ctx, job); err != nil {
		logger.WithError(err).Errorf("Failed to reschedule job %s", job.ID)
	}
}
