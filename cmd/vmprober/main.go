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

	"github.com/gdagil/vmprober/internal/adapter"
	"github.com/gdagil/vmprober/internal/config"
	internalmetrics "github.com/gdagil/vmprober/internal/metrics"
	"github.com/gdagil/vmprober/internal/normalizer"
	"github.com/gdagil/vmprober/internal/observability"
	"github.com/gdagil/vmprober/internal/probe"
	"github.com/gdagil/vmprober/internal/scheduler"
	"github.com/gdagil/vmprober/internal/server"
	"github.com/gdagil/vmprober/internal/shutdown"
	"github.com/gdagil/vmprober/internal/types"
	"github.com/gdagil/vmprober/internal/wal"
	"github.com/gdagil/vmprober/pkg/interfaces"
	"github.com/sirupsen/logrus"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

// setupLogger configures the logger based on configuration
func setupLogger(cfg *config.Config, cmdLogLevel string) *logrus.Logger {
	logger := logrus.New()

	// Priority: command line > config > default
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

	// Configure format
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
		// Default to JSON if format is not specified or unknown
		logger.SetFormatter(&logrus.JSONFormatter{})
	}

	// Configure output
	switch cfg.Logging.Output {
	case "stderr":
		logger.SetOutput(os.Stderr)
	case "file":
		if cfg.Logging.File.Path != "" {
			// TODO: Implement file output with rotation
			// For simplicity, using standard output for now
			logger.SetOutput(os.Stdout)
		} else {
			logger.SetOutput(os.Stdout)
		}
	default:
		// Default to stdout
		logger.SetOutput(os.Stdout)
	}

	return logger
}

func main() {
	configPath := flag.String("config", "configs/config.yaml.example", "Path to configuration file")
	logLevel := flag.String("log-level", "", "Log level (debug, info, warn, error) - overrides config")
	flag.Parse()

	// Temporary logger for initial configuration loading
	tempLogger := logrus.New()
	tempLogger.SetFormatter(&logrus.JSONFormatter{})
	tempLogger.SetLevel(logrus.InfoLevel)

	// Load configuration
	cfgManager := config.NewManager(*configPath, tempLogger)
	cfg, err := cfgManager.Load(context.Background())
	if err != nil {
		tempLogger.WithError(err).Fatal("Failed to load configuration")
	}

	// Configure logger based on configuration
	logger := setupLogger(cfg, *logLevel)

	// Update logger in configuration manager
	cfgManager.SetLogger(logger)

	logger.WithFields(logrus.Fields{
		"version":   Version,
		"buildTime": BuildTime,
		"gitCommit": GitCommit,
	}).Info("Starting VMProber")

	// Create components
	probeFactory := probe.NewFactory()
	enableJobMetrics := true
	if cfg.Metrics.EnableJobMetrics != nil {
		enableJobMetrics = *cfg.Metrics.EnableJobMetrics
	}
	metricsCollector := internalmetrics.NewCollector(cfg.Metrics.Namespace, enableJobMetrics, cfg.Metrics.CustomLabels)
	taskScheduler := scheduler.NewScheduler(logger)
	httpServer := server.NewServer(cfg.Listen.Host, cfg.Listen.Port, logger)

	// Create WAL system
	var walManager wal.WALManager
	if cfg.WAL.Dir != "" {
		walManager, err = wal.NewWALManager(&cfg.WAL, logger)
		if err != nil {
			logger.WithError(err).Warn("Failed to initialize WAL, continuing without WAL")
		} else {
			logger.Info("WAL system initialized")
		}
	}

	// Create normalizer
	resultNormalizer := normalizer.NewNormalizer(logger)

	// Create VictoriaMetrics adapter
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

		// Configure automatic metrics push from VictoriaMetrics library
		// Using the first endpoint from configuration
		// Note: InitPush may not support vminsert format directly
		// Metrics from the library will be sent through the adapter along with other metrics
		// For direct push, you can use the correct format for single-node VM:
		// metrics.InitPush("http://localhost:8428/api/v1/import/prometheus", 10*time.Second, ...)
		// But for vminsert it's better to use the adapter
	}

	// Create observability manager
	obsManager := observability.NewObservabilityManager(&cfg.Observability, logger)
	if err := obsManager.Start(context.Background()); err != nil {
		logger.WithError(err).Warn("Failed to start observability manager")
	}

	// Create shutdown manager
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

	// Configure server
	httpServer.SetScheduler(taskScheduler)
	// Metrics are now handled by VictoriaMetrics library directly

	// Start HTTP server
	if err := httpServer.Start(context.Background()); err != nil {
		logger.WithError(err).Fatal("Failed to start HTTP server")
	}

	// Start scheduler
	if err := taskScheduler.Start(context.Background()); err != nil {
		logger.WithError(err).Fatal("Failed to start scheduler")
	}

	// Handle signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Main task processing loop
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Load targets from configuration
	for _, targetCfg := range cfg.Targets.Static {
		protocols := targetCfg.GetProtocols()

		// If protocols not found, skip target
		if len(protocols) == 0 {
			logger.Warnf("No protocols specified for target %s:%d (raw: %v), skipping", targetCfg.Host, targetCfg.Port, targetCfg.Protocols)
			continue
		}

		logger.Debugf("Target %s:%d has protocols: %v", targetCfg.Host, targetCfg.Port, protocols)

		// Create a job for each protocol
		for _, protocol := range protocols {
			// Job ID includes protocol for uniqueness
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

	// Worker pool for probe execution (limited number of workers)
	workerCount := cfg.Scheduler.Concurrent
	if workerCount <= 0 {
		workerCount = 10 // Default value
	}

	// Semaphore to limit concurrently executing probes
	probeSemaphore := make(chan struct{}, workerCount)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case job := <-taskScheduler.GetJobChan():
				// Get a slot in semaphore (blocks if all workers are busy)
				select {
				case probeSemaphore <- struct{}{}:
					go func(j *types.Job) {
						defer func() { <-probeSemaphore }() // Release slot after completion
						taskScheduler.MarkJobStarted(j.ID)
						executeProbe(ctx, j, probeFactory, metricsCollector, resultNormalizer, walManager, vmAdapter, logger, taskScheduler, cfg.Probes, cfg.Scheduler, cfg)
					}(job)
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	// Replay unsent records from WAL at startup
	if walManager != nil && vmAdapter != nil {
		go replayWALRecords(ctx, walManager, vmAdapter, logger, cfg)

		// Periodic cleanup of old sent records from WAL
		go func() {
			cleanupInterval := cfg.WAL.Retention
			if cleanupInterval <= 0 {
				cleanupInterval = 24 * time.Hour // Default 24 hours
			}

			ticker := time.NewTicker(1 * time.Hour) // Check every hour
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					if err := walManager.DeleteSentRecords(ctx, cleanupInterval); err != nil {
						logger.WithError(err).Warn("Failed to cleanup old WAL records")
					} else {
						logger.Debug("WAL cleanup completed")
					}
				}
			}
		}()
	}

	// Update job metrics periodically
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

	// Periodic metrics push from collector to vminsert
	if vmAdapter != nil {
		go func() {
			ticker := time.NewTicker(30 * time.Second) // Push every 30 seconds
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					// Export all metrics from collector
					exportedMetrics := metricsCollector.ExportMetrics()
					if len(exportedMetrics) > 0 {
						if err := vmAdapter.Push(ctx, exportedMetrics); err != nil {
							logger.WithError(err).Warn("Failed to push collector metrics to VictoriaMetrics")
						} else {
							logger.Info("Pushed collector metrics to VictoriaMetrics")
						}
					}

					// Export scheduler metrics
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

	// Wait for shutdown signal
	<-sigChan
	logger.Info("Shutting down VMProber...")

	// Graceful shutdown through manager
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := shutdownManager.Shutdown(shutdownCtx, 30*time.Second); err != nil {
		logger.WithError(err).Error("Error during graceful shutdown")
	}

	logger.Info("VMProber stopped")
}

// replayWALRecords replays unsent records from WAL at application startup
func replayWALRecords(ctx context.Context, walManager wal.WALManager, vmAdapter adapter.VictoriaMetricsAdapter, logger *logrus.Logger, cfg *config.Config) {
	logger.Info("Starting WAL replay to send unsent records...")

	// Get all unsent records
	unsentRecords, err := walManager.GetUnsentRecords(ctx)
	if err != nil {
		logger.WithError(err).Error("Failed to get unsent records from WAL")
		return
	}

	if len(unsentRecords) == 0 {
		logger.Info("No unsent records found in WAL")
		return
	}

	logger.WithField("count", len(unsentRecords)).Info("Found unsent records in WAL, starting replay")

	// Counters for statistics
	successCount := 0
	failedCount := 0

	// Process each record
	for _, record := range unsentRecords {
		select {
		case <-ctx.Done():
			logger.Warn("WAL replay cancelled due to context cancellation")
			return
		default:
		}

		// Extract event from data
		eventData, ok := record.Data["event"]
		if !ok {
			logger.WithField("record_id", record.ID).Warn("Record has no event data, skipping")
			// Mark as sent to avoid processing again
			if err := walManager.MarkSent(ctx, record.ID); err != nil {
				logger.WithError(err).Warn("Failed to mark invalid record as sent")
			}
			continue
		}

		// Convert eventData back to NormalizedEvent
		eventMap, ok := eventData.(map[string]interface{})
		if !ok {
			logger.WithField("record_id", record.ID).Warn("Invalid event data format, skipping")
			if err := walManager.MarkSent(ctx, record.ID); err != nil {
				logger.WithError(err).Warn("Failed to mark invalid record as sent")
			}
			continue
		}

		// Create metrics from event
		metrics := make([]types.Metric, 0)

		// Extract metrics from event
		if metricsData, ok := eventMap["metrics"].(map[string]interface{}); ok {
			for name, value := range metricsData {
				var floatValue float64
				switch v := value.(type) {
				case float64:
					floatValue = v
				case int:
					floatValue = float64(v)
				case int64:
					floatValue = float64(v)
				default:
					continue
				}

				// Determine metric type
				metricType := types.MetricTypeGauge
				if strings.HasSuffix(name, "_total") {
					metricType = types.MetricTypeCounter
				}

				// Collect labels
				labels := make(map[string]string)
				if labelsData, ok := eventMap["labels"].(map[string]interface{}); ok {
					for k, v := range labelsData {
						if strVal, ok := v.(string); ok {
							labels[k] = strVal
						}
					}
				}
				// Add custom_labels from configuration
				for k, v := range cfg.Metrics.CustomLabels {
					labels[k] = v
				}

				metrics = append(metrics, types.Metric{
					Name:      name,
					Value:     floatValue,
					Timestamp: record.Timestamp,
					Labels:    labels,
					Type:      metricType,
				})
			}
		}

		if len(metrics) == 0 {
			logger.WithField("record_id", record.ID).Debug("No metrics to send for record, marking as sent")
			if err := walManager.MarkSent(ctx, record.ID); err != nil {
				logger.WithError(err).Warn("Failed to mark empty record as sent")
			}
			continue
		}

		// Send metrics
		if err := vmAdapter.Push(ctx, metrics); err != nil {
			logger.WithError(err).WithField("record_id", record.ID).Warn("Failed to push WAL record metrics")
			failedCount++
			// Don't mark as sent - will try again on next startup
			continue
		}

		// Mark record as sent
		if err := walManager.MarkSent(ctx, record.ID); err != nil {
			logger.WithError(err).WithField("record_id", record.ID).Warn("Failed to mark record as sent")
		} else {
			successCount++
		}
	}

	logger.WithFields(logrus.Fields{
		"total":   len(unsentRecords),
		"success": successCount,
		"failed":  failedCount,
	}).Info("WAL replay completed")

	// Clean up old sent records (older than 24 hours)
	if err := walManager.DeleteSentRecords(ctx, 24*time.Hour); err != nil {
		logger.WithError(err).Warn("Failed to delete old sent records")
	}
}

// buildProbeConfig creates probe configuration from global configuration
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

// executeProbe executes probe and reschedules task
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
	// Create probe configuration
	probeConfig := buildProbeConfig(job.Target.Protocol, probesConfig, schedulerConfig)

	// Create probe
	probeInstance, err := factory.CreateProbe(job.Target.Protocol, probeConfig)
	if err != nil {
		logger.WithError(err).Errorf("Failed to create probe for %s", job.ID)
		sched.MarkJobFailed(job.ID)
		return
	}
	defer probeInstance.Close()

	// Execute probe
	result, err := probeInstance.Execute(ctx, job.Target)
	if err != nil && result == nil {
		logger.WithError(err).Errorf("Probe execution failed for %s", job.ID)
		sched.MarkJobFailed(job.ID)
		return
	}

	// Normalize result
	if result != nil && normalizer != nil {
		event, err := normalizer.Normalize(ctx, result)
		if err != nil {
			logger.WithError(err).Warn("Failed to normalize result")
		} else {
			// Check for duplicates (only if dedup is enabled)
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
				// Generate record ID for WAL
				recordID := fmt.Sprintf("%s-%d", event.SeriesID, time.Now().UnixNano())

				// Write to WAL if available
				if walManager != nil {
					record := &types.Record{
						ID:        recordID,
						Timestamp: event.Timestamp,
						Type:      "probe_result",
						SeriesID:  event.SeriesID,
						Data: map[string]interface{}{
							"event": event,
						},
						Labels: event.Labels,
						Sent:   false,
					}
					if err := walManager.Write(ctx, record); err != nil {
						logger.WithError(err).Warn("Failed to write to WAL")
					}
				}

				// Send to VictoriaMetrics if available
				if vmAdapter != nil {
					metrics := make([]types.Metric, 0)
					for name, value := range event.Metrics {
						// Determine metric type by name
						metricType := types.MetricTypeGauge
						if strings.HasSuffix(name, "_total") {
							metricType = types.MetricTypeCounter
						}

						// Copy labels from event and add custom_labels from configuration
						labels := make(map[string]string)
						for k, v := range event.Labels {
							labels[k] = v
						}
						// Add custom_labels from configuration (override existing)
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
						} else {
							// Mark WAL record as sent
							if walManager != nil {
								if err := walManager.MarkSent(ctx, recordID); err != nil {
									logger.WithError(err).Debug("Failed to mark WAL record as sent")
								}
							}
						}
					}
				}
			}
		}
	}

	// Write metrics to collector (if result exists)
	if result != nil {
		if err := collector.Record(ctx, result); err != nil {
			logger.WithError(err).Error("Failed to record metrics")
		}
	}

	// Mark job as completed or failed
	if result != nil && result.Success {
		sched.MarkJobCompleted(job.ID)
	} else {
		// If result == nil or result.Success == false, consider it a failure
		sched.MarkJobFailed(job.ID)
	}

	// Update NextRun before rescheduling
	job.NextRun = time.Now().Add(job.Interval)

	// Reschedule task without incrementing TotalJobs counter
	if err := sched.Reschedule(ctx, job); err != nil {
		logger.WithError(err).Errorf("Failed to reschedule job %s", job.ID)
	}
}
