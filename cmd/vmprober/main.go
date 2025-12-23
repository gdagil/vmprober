package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vmprober/vmprober/internal/adapter"
	"github.com/vmprober/vmprober/internal/config"
	"github.com/vmprober/vmprober/internal/metrics"
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

func main() {
	configPath := flag.String("config", "configs/config.yaml.example", "Path to configuration file")
	logLevel := flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	flag.Parse()

	// Инициализация логгера
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	level, err := logrus.ParseLevel(*logLevel)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	logger.WithFields(logrus.Fields{
		"version":   Version,
		"buildTime": BuildTime,
		"gitCommit": GitCommit,
	}).Info("Starting VMProber")

	// Загрузка конфигурации
	cfgManager := config.NewManager(*configPath, logger)
	cfg, err := cfgManager.Load(context.Background())
	if err != nil {
		logger.WithError(err).Fatal("Failed to load configuration")
	}

	// Создание компонентов
	probeFactory := probe.NewFactory()
	enableJobMetrics := true
	if cfg.Metrics.EnableJobMetrics != nil {
		enableJobMetrics = *cfg.Metrics.EnableJobMetrics
	}
	metricsCollector := metrics.NewCollector(cfg.Metrics.Namespace, enableJobMetrics)
	taskScheduler := scheduler.NewScheduler()
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
	httpServer.SetMetricsCollector(metricsCollector)

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
		target := types.Target{
			ID:       fmt.Sprintf("%s:%d", targetCfg.Host, targetCfg.Port),
			Host:     targetCfg.Host,
			Port:     targetCfg.Port,
			Protocol: targetCfg.Protocol,
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
			ID:        target.ID,
			Target:    target,
			NextRun:   time.Now(),
			Interval:  target.Interval,
			Priority:  1,
			CreatedAt: time.Now(),
		}

		if err := taskScheduler.Schedule(ctx, job); err != nil {
			logger.WithError(err).Errorf("Failed to schedule job for %s", target.ID)
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
					executeProbe(ctx, j, probeFactory, metricsCollector, resultNormalizer, walManager, vmAdapter, logger, taskScheduler)
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
						stats.QueuedJobs,
						stats.CompletedJobs,
						stats.FailedJobs,
					)
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

// executeProbe выполняет пробу и перепланирует задачу
func executeProbe(
	ctx context.Context,
	job *types.Job,
	factory interfaces.ProbeFactory,
	collector *metrics.Collector,
	normalizer normalizer.Normalizer,
	walManager wal.WALManager,
	vmAdapter adapter.VictoriaMetricsAdapter,
	logger *logrus.Logger,
	sched *scheduler.Scheduler,
) {
	// Создание пробы
	probeInstance, err := factory.CreateProbe(job.Target.Protocol, nil)
	if err != nil {
		logger.WithError(err).Errorf("Failed to create probe for %s", job.ID)
		return
	}
	defer probeInstance.Close()

	// Выполнение пробы
	result, err := probeInstance.Execute(ctx, job.Target)
	if err != nil && result == nil {
		logger.WithError(err).Errorf("Probe execution failed for %s", job.ID)
		sched.MarkJobFailed()
		return
	}

	// Нормализация результата
	if result != nil && normalizer != nil {
		event, err := normalizer.Normalize(ctx, result)
		if err != nil {
			logger.WithError(err).Warn("Failed to normalize result")
		} else {
			// Проверка на дубликаты
			isDup, err := normalizer.Dedup(ctx, event)
			if err != nil {
				logger.WithError(err).Warn("Failed to check for duplicates")
			}
			if isDup {
				logger.Debug("Duplicate event detected, skipping")
			} else {
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
						metrics = append(metrics, types.Metric{
							Name:      name,
							Value:     value,
							Timestamp: event.Timestamp,
							Labels:    event.Labels,
							Type:      types.MetricTypeGauge,
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

	// Запись метрик в коллектор
	if err := collector.Record(ctx, result); err != nil {
		logger.WithError(err).Error("Failed to record metrics")
	}

	// Отметка джоба как завершенного
	if result != nil && result.Success {
		sched.MarkJobCompleted()
	} else {
		sched.MarkJobFailed()
	}

	// Обновляем NextRun перед перепланированием
	job.NextRun = time.Now().Add(job.Interval)

	// Перепланирование задачи (она уже есть в allJobs, просто обновим NextRun)
	if err := sched.Schedule(ctx, job); err != nil {
		logger.WithError(err).Errorf("Failed to reschedule job %s", job.ID)
	}
}
