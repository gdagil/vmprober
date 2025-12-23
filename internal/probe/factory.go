package probe

import (
	"fmt"

	"github.com/vmprober/vmprober/internal/types"
	"github.com/vmprober/vmprober/pkg/interfaces"
)

// Factory реализует ProbeFactory
type Factory struct {
	creators map[types.ProbeType]func(interface{}) (interfaces.Probe, error)
}

// NewFactory создает новую фабрику проб
func NewFactory() *Factory {
	f := &Factory{
		creators: make(map[types.ProbeType]func(interface{}) (interfaces.Probe, error)),
	}

	// Регистрация стандартных типов проб
	f.creators[types.ProbeTypeTCP] = func(config interface{}) (interfaces.Probe, error) {
		tcpConfig, ok := config.(*TCPConfig)
		if !ok {
			return nil, fmt.Errorf("invalid TCP config type")
		}
		return NewTCPProbe(tcpConfig), nil
	}

	f.creators[types.ProbeTypeUDP] = func(config interface{}) (interfaces.Probe, error) {
		udpConfig, ok := config.(*UDPConfig)
		if !ok {
			return nil, fmt.Errorf("invalid UDP config type")
		}
		return NewUDPProbe(udpConfig), nil
	}

	return f
}

// CreateProbe создает пробу указанного типа
func (f *Factory) CreateProbe(probeType types.ProbeType, config interface{}) (interfaces.Probe, error) {
	creator, exists := f.creators[probeType]
	if !exists {
		return nil, fmt.Errorf("unsupported probe type: %s", probeType)
	}

	return creator(config)
}

// GetSupportedTypes возвращает поддерживаемые типы проб
func (f *Factory) GetSupportedTypes() []types.ProbeType {
	types := make([]types.ProbeType, 0, len(f.creators))
	for probeType := range f.creators {
		types = append(types, probeType)
	}
	return types
}


