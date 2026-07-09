package main

import (
	"os"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gdagil/vmprober/internal/config"
	"github.com/gdagil/vmprober/internal/probe"
	"github.com/gdagil/vmprober/internal/types"
)

func newLoggingConfig(level, format, output string) *config.Config {
	cfg := &config.Config{}
	cfg.Logging.Level = level
	cfg.Logging.Format = format
	cfg.Logging.Output = output
	return cfg
}

func TestSetupLogger_Level(t *testing.T) {
	tests := []struct {
		name        string
		cmdLogLevel string
		cfgLevel    string
		wantLevel   logrus.Level
	}{
		{name: "cmd level takes priority over config", cmdLogLevel: "debug", cfgLevel: "warn", wantLevel: logrus.DebugLevel},
		{name: "invalid cmd level falls back to info", cmdLogLevel: "not-a-level", cfgLevel: "error", wantLevel: logrus.InfoLevel},
		{name: "config level used when cmd empty", cmdLogLevel: "", cfgLevel: "warn", wantLevel: logrus.WarnLevel},
		{name: "invalid config level falls back to info", cmdLogLevel: "", cfgLevel: "bogus", wantLevel: logrus.InfoLevel},
		{name: "default info when both empty", cmdLogLevel: "", cfgLevel: "", wantLevel: logrus.InfoLevel},
		{name: "error level from config", cmdLogLevel: "", cfgLevel: "error", wantLevel: logrus.ErrorLevel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := newLoggingConfig(tt.cfgLevel, "json", "stdout")
			logger := setupLogger(cfg, tt.cmdLogLevel)
			require.NotNil(t, logger)
			assert.Equal(t, tt.wantLevel, logger.GetLevel())
		})
	}
}

func TestSetupLogger_Format(t *testing.T) {
	tests := []struct {
		name   string
		format string
		// wantJSON true expects JSONFormatter, false expects TextFormatter
		wantJSON bool
	}{
		{name: "text format", format: "text", wantJSON: false},
		{name: "json format", format: "json", wantJSON: true},
		{name: "unknown format defaults to json", format: "xml", wantJSON: true},
		{name: "empty format defaults to json", format: "", wantJSON: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := newLoggingConfig("info", tt.format, "stdout")
			logger := setupLogger(cfg, "")
			require.NotNil(t, logger)

			if tt.wantJSON {
				_, ok := logger.Formatter.(*logrus.JSONFormatter)
				assert.True(t, ok, "expected JSONFormatter, got %T", logger.Formatter)
			} else {
				_, ok := logger.Formatter.(*logrus.TextFormatter)
				assert.True(t, ok, "expected TextFormatter, got %T", logger.Formatter)
			}
		})
	}
}

func TestSetupLogger_Output(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   *os.File
	}{
		{name: "stderr", output: "stderr", want: os.Stderr},
		{name: "file falls back to stdout", output: "file", want: os.Stdout},
		{name: "stdout", output: "stdout", want: os.Stdout},
		{name: "unknown defaults to stdout", output: "somewhere", want: os.Stdout},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := newLoggingConfig("info", "json", tt.output)
			logger := setupLogger(cfg, "")
			require.NotNil(t, logger)
			assert.Equal(t, tt.want, logger.Out)
		})
	}
}

func TestBuildProbeConfig_TCP(t *testing.T) {
	probesCfg := &config.ProbesConfig{}
	schedulerCfg := &config.SchedulerConfig{}

	t.Run("explicit connect timeout is honored", func(t *testing.T) {
		probesCfg.TCP.ConnectTimeout = 7 * time.Second
		result := buildProbeConfig(types.ProbeTypeTCP, probesCfg, schedulerCfg)
		tcpCfg, ok := result.(*probe.TCPConfig)
		require.True(t, ok, "expected *probe.TCPConfig, got %T", result)
		assert.Equal(t, 7*time.Second, tcpCfg.ConnectTimeout)
		assert.Equal(t, 7*time.Second, tcpCfg.ReadTimeout)
		assert.Equal(t, 7*time.Second, tcpCfg.WriteTimeout)
	})

	t.Run("zero timeout uses scheduler timeout override", func(t *testing.T) {
		zeroProbes := &config.ProbesConfig{}
		sched := &config.SchedulerConfig{Timeouts: map[string]time.Duration{"tcp": 12 * time.Second}}
		result := buildProbeConfig(types.ProbeTypeTCP, zeroProbes, sched)
		tcpCfg := result.(*probe.TCPConfig)
		assert.Equal(t, 12*time.Second, tcpCfg.ConnectTimeout)
	})

	t.Run("zero timeout with no override uses default", func(t *testing.T) {
		zeroProbes := &config.ProbesConfig{}
		result := buildProbeConfig(types.ProbeTypeTCP, zeroProbes, &config.SchedulerConfig{})
		tcpCfg := result.(*probe.TCPConfig)
		assert.Equal(t, 5*time.Second, tcpCfg.ConnectTimeout)
	})
}

func TestBuildProbeConfig_UDP(t *testing.T) {
	t.Run("defaults are applied for zero values", func(t *testing.T) {
		result := buildProbeConfig(types.ProbeTypeUDP, &config.ProbesConfig{}, &config.SchedulerConfig{})
		udpCfg, ok := result.(*probe.UDPConfig)
		require.True(t, ok, "expected *probe.UDPConfig, got %T", result)
		assert.Equal(t, 64, udpCfg.PayloadSize)
		assert.Equal(t, 1024, udpCfg.MaxPacketSize)
		assert.Equal(t, 3*time.Second, udpCfg.ResponseTimeout)
	})

	t.Run("explicit values are honored", func(t *testing.T) {
		probesCfg := &config.ProbesConfig{}
		probesCfg.UDP.PayloadSize = 128
		probesCfg.UDP.MaxPacketSize = 2048
		probesCfg.UDP.ResponseTimeout = 9 * time.Second
		result := buildProbeConfig(types.ProbeTypeUDP, probesCfg, &config.SchedulerConfig{})
		udpCfg := result.(*probe.UDPConfig)
		assert.Equal(t, 128, udpCfg.PayloadSize)
		assert.Equal(t, 2048, udpCfg.MaxPacketSize)
		assert.Equal(t, 9*time.Second, udpCfg.ResponseTimeout)
	})

	t.Run("response timeout falls back to scheduler override", func(t *testing.T) {
		sched := &config.SchedulerConfig{Timeouts: map[string]time.Duration{"udp": 6 * time.Second}}
		result := buildProbeConfig(types.ProbeTypeUDP, &config.ProbesConfig{}, sched)
		udpCfg := result.(*probe.UDPConfig)
		assert.Equal(t, 6*time.Second, udpCfg.ResponseTimeout)
	})
}

func TestBuildProbeConfig_ICMP(t *testing.T) {
	t.Run("defaults are applied for zero values", func(t *testing.T) {
		result := buildProbeConfig(types.ProbeTypeICMP, &config.ProbesConfig{}, &config.SchedulerConfig{})
		icmpCfg, ok := result.(*probe.ICMPConfig)
		require.True(t, ok, "expected *probe.ICMPConfig, got %T", result)
		assert.Equal(t, 1, icmpCfg.SequenceStart)
		assert.Equal(t, 64, icmpCfg.TTL)
	})

	t.Run("explicit values are honored", func(t *testing.T) {
		probesCfg := &config.ProbesConfig{}
		probesCfg.ICMP.Library = "custom"
		probesCfg.ICMP.SequenceStart = 100
		probesCfg.ICMP.TTL = 32
		result := buildProbeConfig(types.ProbeTypeICMP, probesCfg, &config.SchedulerConfig{})
		icmpCfg := result.(*probe.ICMPConfig)
		assert.Equal(t, "custom", icmpCfg.Library)
		assert.Equal(t, 100, icmpCfg.SequenceStart)
		assert.Equal(t, 32, icmpCfg.TTL)
	})
}

func TestBuildProbeConfig_UnknownReturnsNil(t *testing.T) {
	result := buildProbeConfig(types.ProbeType("bogus"), &config.ProbesConfig{}, &config.SchedulerConfig{})
	assert.Nil(t, result)
}
