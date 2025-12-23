package probe

import (
	"context"
	"crypto/rand"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/vmprober/vmprober/internal/types"
	"github.com/vmprober/vmprober/pkg/interfaces"
)

// UDPProbe реализует UDP пробы
type UDPProbe struct {
	config *UDPConfig
}

// UDPConfig конфигурация UDP пробы
type UDPConfig struct {
	PayloadSize     int
	ResponseTimeout time.Duration
	MaxPacketSize   int
}

// NewUDPProbe создает новый UDP probe
func NewUDPProbe(config *UDPConfig) interfaces.Probe {
	return &UDPProbe{
		config: config,
	}
}

// Execute выполняет UDP пробу
func (p *UDPProbe) Execute(ctx context.Context, target types.Target) (*types.ProbeResult, error) {
	start := time.Now()
	result := &types.ProbeResult{
		Protocol:  types.ProbeTypeUDP,
		Timestamp: start,
		Attempt:   1,
	}

	// Создание UDP соединения
	conn, err := net.ListenUDP("udp", nil)
	if err != nil {
		result.Error = fmt.Sprintf("failed to create UDP socket: %v", err)
		result.RTT = time.Since(start)
		return result, err
	}
	defer conn.Close()

	// Разрешение адреса
	addr := net.JoinHostPort(target.Host, strconv.Itoa(target.Port))
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		result.Error = fmt.Sprintf("failed to resolve UDP address: %v", err)
		result.RTT = time.Since(start)
		return result, err
	}

	// Генерация payload
	payloadSize := p.config.PayloadSize
	if payloadSize == 0 {
		payloadSize = 64
	}
	payload := make([]byte, payloadSize)
	if _, err := rand.Read(payload); err != nil {
		result.Error = fmt.Sprintf("failed to generate payload: %v", err)
		result.RTT = time.Since(start)
		return result, err
	}
	result.Payload = payload

	// Отправка пакета
	if _, err := conn.WriteToUDP(payload, udpAddr); err != nil {
		result.Error = fmt.Sprintf("failed to send UDP packet: %v", err)
		result.RTT = time.Since(start)
		return result, err
	}

	// Установка таймаута для чтения
	timeout := p.config.ResponseTimeout
	if timeout == 0 {
		timeout = target.Timeout
	}
	if err := conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		result.Error = fmt.Sprintf("failed to set read deadline: %v", err)
		result.RTT = time.Since(start)
		return result, err
	}

	// Попытка чтения ответа
	buffer := make([]byte, p.config.MaxPacketSize)
	if p.config.MaxPacketSize == 0 {
		buffer = make([]byte, 1500)
	}

	_, _, err = conn.ReadFromUDP(buffer)
	if err != nil {
		// UDP может не получить ответ, это не всегда ошибка
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			result.Error = "UDP response timeout"
		} else {
			result.Error = fmt.Sprintf("failed to receive UDP response: %v", err)
		}
		result.RTT = time.Since(start)
		// Не возвращаем ошибку для UDP timeout, это нормально
		return result, nil
	}

	// Успешный ответ
	result.Success = true
	result.RTT = time.Since(start)
	result.TargetIP = udpAddr.IP.String()
	result.TargetPort = udpAddr.Port
	result.SourceIP = conn.LocalAddr().(*net.UDPAddr).IP.String()
	result.Role = "client"

	return result, nil
}

// Type возвращает тип пробы
func (p *UDPProbe) Type() types.ProbeType {
	return types.ProbeTypeUDP
}

// Validate проверяет конфигурацию
func (p *UDPProbe) Validate(config interface{}) error {
	if p.config == nil {
		return fmt.Errorf("UDP config is nil")
	}
	return nil
}

// Close освобождает ресурсы
func (p *UDPProbe) Close() error {
	return nil
}


