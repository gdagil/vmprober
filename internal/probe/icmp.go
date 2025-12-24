package probe

import (
	"context"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/vmprober/vmprober/internal/types"
	"github.com/vmprober/vmprober/pkg/interfaces"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

// ICMPProbe реализует ICMP пробы
type ICMPProbe struct {
	config   *ICMPConfig
	socket   *icmp.PacketConn
	sequence uint16
	mu       sync.RWMutex
}

// ICMPConfig конфигурация ICMP пробы
type ICMPConfig struct {
	Library       string
	SequenceStart int
	TTL           int
	Data          []byte
}

// ICMPResponse структура для парсинга ICMP ответа
type ICMPResponse struct {
	Type     int
	Code     int
	ID       uint16
	Sequence uint16
	Data     []byte
}

// NewICMPProbe создает новый ICMP probe
func NewICMPProbe(config *ICMPConfig) interfaces.Probe {
	if config == nil {
		config = &ICMPConfig{
			SequenceStart: 1,
			TTL:           64,
		}
	}

	// Создание ICMP сокета
	var conn *icmp.PacketConn
	var err error

	// Пытаемся создать IPv4 сокет
	conn, err = icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		// Если не получилось, пробуем IPv6
		conn, err = icmp.ListenPacket("ip6:ipv6-icmp", "::")
		if err != nil {
			// Если и это не получилось, возвращаем probe без сокета
			// Сокет будет создан при первом выполнении
		}
	}

	return &ICMPProbe{
		config:   config,
		socket:   conn,
		sequence: uint16(config.SequenceStart),
	}
}

// Execute выполняет ICMP пробу
func (p *ICMPProbe) Execute(ctx context.Context, target types.Target) (*types.ProbeResult, error) {
	start := time.Now()
	result := &types.ProbeResult{
		Protocol:  types.ProbeTypeICMP,
		Timestamp: start,
		Attempt:   1,
	}

	// Инициализация сокета если нужно
	if p.socket == nil {
		if err := p.initSocket(); err != nil {
			result.Error = fmt.Sprintf("failed to initialize ICMP socket: %v", err)
			result.RTT = time.Since(start)
			return result, err
		}
	}

	// Сохраняем hostname для метрик (до DNS резолвинга)
	result.TargetHost = target.Host

	// Разрешение DNS если нужно
	if isHostname(target.Host) {
		resolvedIPs, lookupTime, err := p.resolveDNS(ctx, target.Host, target.NetworkFamily)
		if err != nil {
			result.Error = fmt.Sprintf("DNS resolution failed: %v", err)
			result.RTT = time.Since(start)
			return result, err
		}
		result.DNSResult = &types.DNSResult{
			ResolvedIPs: resolvedIPs,
			LookupTime:  lookupTime,
		}
		target.Host = resolvedIPs[0] // Используем первый IP
	}

	result.TargetIP = target.Host

	// Определение семейства адресов
	ip := net.ParseIP(target.Host)
	if ip == nil {
		result.Error = fmt.Sprintf("invalid IP address: %s", target.Host)
		result.RTT = time.Since(start)
		return result, fmt.Errorf("invalid IP address: %s", target.Host)
	}

	isIPv4 := ip.To4() != nil
	var network string

	if isIPv4 {
		network = "ip4:icmp"
		result.SocketFamily = "inet"
	} else {
		network = "ip6:ipv6-icmp"
		result.SocketFamily = "inet6"
	}

	// Создание нового сокета для этого запроса если нужно
	conn, err := icmp.ListenPacket(network, "0.0.0.0")
	if err != nil {
		result.Error = fmt.Sprintf("failed to create ICMP socket: %v", err)
		result.RTT = time.Since(start)
		return result, err
	}
	defer conn.Close()

	// Установка TTL если нужно
	if p.config.TTL > 0 {
		if isIPv4 {
			if err := conn.IPv4PacketConn().SetTTL(p.config.TTL); err != nil {
				result.Error = fmt.Sprintf("failed to set TTL: %v", err)
				result.RTT = time.Since(start)
				return result, err
			}
		} else {
			if err := conn.IPv6PacketConn().SetHopLimit(p.config.TTL); err != nil {
				result.Error = fmt.Sprintf("failed to set hop limit: %v", err)
				result.RTT = time.Since(start)
				return result, err
			}
		}
	}

	// Создание ICMP пакета
	icmpID := uint16(os.Getpid() & 0xFFFF)
	p.mu.Lock()
	p.sequence++
	if p.sequence == 0 {
		p.sequence = uint16(p.config.SequenceStart)
	}
	icmpSeq := p.sequence
	p.mu.Unlock()

	// Подготовка данных
	data := p.config.Data
	if data == nil {
		data = make([]byte, 56) // Стандартный размер для ping
		for i := range data {
			data[i] = byte(i)
		}
	}

	var msg icmp.Message
	if isIPv4 {
		msg = icmp.Message{
			Type: ipv4.ICMPTypeEcho,
			Code: 0,
			Body: &icmp.Echo{
				ID:   int(icmpID),
				Seq:  int(icmpSeq),
				Data: data,
			},
		}
	} else {
		msg = icmp.Message{
			Type: ipv6.ICMPTypeEchoRequest,
			Code: 0,
			Body: &icmp.Echo{
				ID:   int(icmpID),
				Seq:  int(icmpSeq),
				Data: data,
			},
		}
	}

	// Сериализация сообщения
	packet, err := msg.Marshal(nil)
	if err != nil {
		result.Error = fmt.Sprintf("failed to marshal ICMP packet: %v", err)
		result.RTT = time.Since(start)
		return result, err
	}
	result.Payload = packet

	// Создание адреса назначения
	var targetAddr net.Addr
	if isIPv4 {
		targetAddr = &net.IPAddr{IP: ip}
	} else {
		targetAddr = &net.IPAddr{IP: ip}
	}

	// Установка таймаута чтения
	if err := conn.SetReadDeadline(time.Now().Add(target.Timeout)); err != nil {
		result.Error = fmt.Sprintf("failed to set read deadline: %v", err)
		result.RTT = time.Since(start)
		return result, err
	}

	// Отправка пакета
	if _, err := conn.WriteTo(packet, targetAddr); err != nil {
		result.Error = fmt.Sprintf("failed to send ICMP packet: %v", err)
		result.RTT = time.Since(start)
		return result, err
	}

	// Получение ответа
	buffer := make([]byte, 1500) // Стандартный MTU
	n, peer, err := conn.ReadFrom(buffer)
	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			result.Error = "ICMP response timeout"
			result.RTT = time.Since(start)
			return result, nil // Timeout не является критической ошибкой
		}
		result.Error = fmt.Sprintf("failed to receive ICMP response: %v", err)
		result.RTT = time.Since(start)
		return result, err
	}

	// Парсинг ответа
	response, err := p.parseICMPResponse(buffer[:n], isIPv4)
	if err != nil {
		result.Error = fmt.Sprintf("failed to parse ICMP response: %v", err)
		result.RTT = time.Since(start)
		return result, err
	}

	// Проверка соответствия запроса и ответа
	if !p.isMatchingResponse(response, icmpID, icmpSeq) {
		result.Error = "ICMP response does not match request"
		result.RTT = time.Since(start)
		return result, nil
	}

	// Успешный ответ
	result.Success = true
	result.RTT = time.Since(start)
	result.Response = buffer[:n]
	result.SourceIP = peer.String()
	result.Role = "client"

	return result, nil
}

// initSocket инициализирует ICMP сокет
func (p *ICMPProbe) initSocket() error {
	conn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		conn, err = icmp.ListenPacket("ip6:ipv6-icmp", "::")
		if err != nil {
			return err
		}
	}
	p.socket = conn
	return nil
}

// resolveDNS разрешает DNS имя
func (p *ICMPProbe) resolveDNS(ctx context.Context, hostname string, family types.NetworkFamily) ([]string, time.Duration, error) {
	start := time.Now()
	var network string

	switch family {
	case types.NetworkFamilyInet:
		network = "ip4"
	case types.NetworkFamilyInet6:
		network = "ip6"
	default:
		network = "ip"
	}

	resolver := &net.Resolver{}
	ips, err := resolver.LookupIP(ctx, network, hostname)
	if err != nil {
		return nil, time.Since(start), err
	}

	result := make([]string, len(ips))
	for i, ip := range ips {
		result[i] = ip.String()
	}

	return result, time.Since(start), nil
}

// parseICMPResponse парсит ICMP ответ
func (p *ICMPProbe) parseICMPResponse(data []byte, isIPv4 bool) (*ICMPResponse, error) {
	var msg *icmp.Message
	var err error

	if isIPv4 {
		msg, err = icmp.ParseMessage(ipv4.ICMPTypeEchoReply.Protocol(), data)
	} else {
		msg, err = icmp.ParseMessage(ipv6.ICMPTypeEchoReply.Protocol(), data)
	}

	if err != nil {
		return nil, err
	}

	echo, ok := msg.Body.(*icmp.Echo)
	if !ok {
		return nil, fmt.Errorf("unexpected ICMP message type")
	}

	return &ICMPResponse{
		Type:     int(msg.Type.Protocol()),
		Code:     msg.Code,
		ID:       uint16(echo.ID),
		Sequence: uint16(echo.Seq),
		Data:     echo.Data,
	}, nil
}

// isMatchingResponse проверяет соответствие ответа запросу
func (p *ICMPProbe) isMatchingResponse(response *ICMPResponse, expectedID, expectedSeq uint16) bool {
	return response.ID == expectedID && response.Sequence == expectedSeq
}

// isHostname проверяет является ли строка hostname
func isHostname(s string) bool {
	ip := net.ParseIP(s)
	return ip == nil
}

// Type возвращает тип пробы
func (p *ICMPProbe) Type() types.ProbeType {
	return types.ProbeTypeICMP
}

// Validate проверяет конфигурацию
func (p *ICMPProbe) Validate(config interface{}) error {
	if p.config == nil {
		return fmt.Errorf("ICMP config is nil")
	}
	return nil
}

// Close освобождает ресурсы
func (p *ICMPProbe) Close() error {
	if p.socket != nil {
		return p.socket.Close()
	}
	return nil
}

