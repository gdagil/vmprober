package probe

import (
	"fmt"

	"github.com/gdagil/vmprober/internal/types"
	"github.com/gdagil/vmprober/pkg/interfaces"
)

// Factory implements ProbeFactory
type Factory struct {
	creators map[types.ProbeType]func(interface{}) (interfaces.Probe, error)
}

// NewFactory creates a new probe factory
func NewFactory() *Factory {
	f := &Factory{
		creators: make(map[types.ProbeType]func(interface{}) (interfaces.Probe, error)),
	}

	// Register standard probe types
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

	f.creators[types.ProbeTypeICMP] = func(config interface{}) (interfaces.Probe, error) {
		icmpConfig, ok := config.(*ICMPConfig)
		if !ok && config != nil {
			return nil, fmt.Errorf("invalid ICMP config type")
		}
		return NewICMPProbe(icmpConfig), nil
	}

	// HTTP probe
	f.creators[types.ProbeTypeHTTP] = func(config interface{}) (interfaces.Probe, error) {
		httpConfig, ok := config.(*HTTPConfig)
		if !ok && config != nil {
			return nil, fmt.Errorf("invalid HTTP config type")
		}
		return NewHTTPProbe(httpConfig), nil
	}

	// HTTPS probe (uses the same HTTP probe with TLS)
	f.creators[types.ProbeTypeHTTPS] = func(config interface{}) (interfaces.Probe, error) {
		httpConfig, ok := config.(*HTTPConfig)
		if !ok && config != nil {
			// Create default config with TLS
			httpConfig = &HTTPConfig{TLS: true}
		} else if httpConfig != nil {
			httpConfig.TLS = true
		} else {
			httpConfig = &HTTPConfig{TLS: true}
		}
		return NewHTTPProbe(httpConfig), nil
	}

	// DNS probe
	f.creators[types.ProbeTypeDNS] = func(config interface{}) (interfaces.Probe, error) {
		dnsConfig, ok := config.(*DNSConfig)
		if !ok && config != nil {
			return nil, fmt.Errorf("invalid DNS config type")
		}
		return NewDNSProbe(dnsConfig), nil
	}

	// gRPC probe
	f.creators[types.ProbeTypeGRPC] = func(config interface{}) (interfaces.Probe, error) {
		grpcConfig, ok := config.(*GRPCConfig)
		if !ok && config != nil {
			return nil, fmt.Errorf("invalid gRPC config type")
		}
		return NewGRPCProbe(grpcConfig), nil
	}

	return f
}

// CreateProbe creates a probe of specified type
func (f *Factory) CreateProbe(probeType types.ProbeType, config interface{}) (interfaces.Probe, error) {
	creator, exists := f.creators[probeType]
	if !exists {
		return nil, fmt.Errorf("unsupported probe type: %s", probeType)
	}

	return creator(config)
}

// GetSupportedTypes returns supported probe types
func (f *Factory) GetSupportedTypes() []types.ProbeType {
	types := make([]types.ProbeType, 0, len(f.creators))
	for probeType := range f.creators {
		types = append(types, probeType)
	}
	return types
}
