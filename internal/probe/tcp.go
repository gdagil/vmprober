package probe

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/vmprober/vmprober/internal/types"
	"github.com/vmprober/vmprober/pkg/interfaces"
)

// TCPProbe реализует TCP пробы
type TCPProbe struct {
	config *TCPConfig
}

// TCPConfig конфигурация TCP пробы
type TCPConfig struct {
	ConnectTimeout time.Duration
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
}

// NewTCPProbe создает новый TCP probe
func NewTCPProbe(config *TCPConfig) interfaces.Probe {
	return &TCPProbe{
		config: config,
	}
}

// Execute выполняет TCP пробу
func (p *TCPProbe) Execute(ctx context.Context, target types.Target) (*types.ProbeResult, error) {
	start := time.Now()
	result := &types.ProbeResult{
		Protocol:  types.ProbeTypeTCP,
		Timestamp: start,
		Attempt:   1,
	}

	// Разрешение DNS
	addr := target.Host
	if target.Port > 0 {
		addr = net.JoinHostPort(target.Host, strconv.Itoa(target.Port))
	}

	// Создание dialer с таймаутом
	dialer := &net.Dialer{
		Timeout: target.Timeout,
	}

	// Выполнение подключения
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		result.Error = err.Error()
		result.RTT = time.Since(start)
		return result, err
	}
	defer conn.Close()

	// Успешное подключение
	result.Success = true
	result.RTT = time.Since(start)
	result.TargetIP = conn.RemoteAddr().(*net.TCPAddr).IP.String()
	result.TargetPort = conn.RemoteAddr().(*net.TCPAddr).Port
	result.SourceIP = conn.LocalAddr().(*net.TCPAddr).IP.String()
	result.Role = "client"

	return result, nil
}

// Type возвращает тип пробы
func (p *TCPProbe) Type() types.ProbeType {
	return types.ProbeTypeTCP
}

// Validate проверяет конфигурацию
func (p *TCPProbe) Validate(config interface{}) error {
	if p.config == nil {
		return fmt.Errorf("TCP config is nil")
	}
	return nil
}

// Close освобождает ресурсы
func (p *TCPProbe) Close() error {
	return nil
}



