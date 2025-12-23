package probe

import (
	"testing"
	"time"

	"github.com/vmprober/vmprober/internal/types"
)

func TestNewFactory(t *testing.T) {
	factory := NewFactory()
	if factory == nil {
		t.Fatal("NewFactory returned nil")
	}
	
	supportedTypes := factory.GetSupportedTypes()
	if len(supportedTypes) == 0 {
		t.Error("Expected at least one supported probe type")
	}
	
	// Проверяем, что TCP, UDP и ICMP поддерживаются
	hasTCP := false
	hasUDP := false
	hasICMP := false
	for _, probeType := range supportedTypes {
		if probeType == types.ProbeTypeTCP {
			hasTCP = true
		}
		if probeType == types.ProbeTypeUDP {
			hasUDP = true
		}
		if probeType == types.ProbeTypeICMP {
			hasICMP = true
		}
	}
	
	if !hasTCP {
		t.Error("Expected TCP probe type to be supported")
	}
	if !hasUDP {
		t.Error("Expected UDP probe type to be supported")
	}
	if !hasICMP {
		t.Error("Expected ICMP probe type to be supported")
	}
}

func TestFactory_CreateProbe_TCP(t *testing.T) {
	factory := NewFactory()
	
	config := &TCPConfig{
		ConnectTimeout: 5 * time.Second,
	}
	
	probe, err := factory.CreateProbe(types.ProbeTypeTCP, config)
	if err != nil {
		t.Fatalf("CreateProbe failed: %v", err)
	}
	if probe == nil {
		t.Fatal("CreateProbe returned nil")
	}
	if probe.Type() != types.ProbeTypeTCP {
		t.Errorf("Expected TCP probe, got %s", probe.Type())
	}
	
	probe.Close()
}

func TestFactory_CreateProbe_UDP(t *testing.T) {
	factory := NewFactory()
	
	config := &UDPConfig{
		PayloadSize: 64,
	}
	
	probe, err := factory.CreateProbe(types.ProbeTypeUDP, config)
	if err != nil {
		t.Fatalf("CreateProbe failed: %v", err)
	}
	if probe == nil {
		t.Fatal("CreateProbe returned nil")
	}
	if probe.Type() != types.ProbeTypeUDP {
		t.Errorf("Expected UDP probe, got %s", probe.Type())
	}
	
	probe.Close()
}

func TestFactory_CreateProbe_ICMP(t *testing.T) {
	factory := NewFactory()
	
	probe, err := factory.CreateProbe(types.ProbeTypeICMP, nil)
	if err != nil {
		t.Fatalf("CreateProbe failed: %v", err)
	}
	if probe == nil {
		t.Fatal("CreateProbe returned nil")
	}
	if probe.Type() != types.ProbeTypeICMP {
		t.Errorf("Expected ICMP probe, got %s", probe.Type())
	}
	
	probe.Close()
}

func TestFactory_CreateProbe_UnsupportedType(t *testing.T) {
	factory := NewFactory()
	
	// Используем несуществующий тип пробы
	unsupportedType := types.ProbeType("unsupported")
	_, err := factory.CreateProbe(unsupportedType, nil)
	if err == nil {
		t.Error("Expected error for unsupported probe type")
	}
}

func TestFactory_CreateProbe_InvalidConfig(t *testing.T) {
	factory := NewFactory()
	
	// Передаем неправильный тип конфигурации
	_, err := factory.CreateProbe(types.ProbeTypeTCP, &UDPConfig{})
	if err == nil {
		t.Error("Expected error for invalid config type")
	}
	
	_, err = factory.CreateProbe(types.ProbeTypeUDP, &TCPConfig{})
	if err == nil {
		t.Error("Expected error for invalid config type")
	}
}

func TestFactory_CreateProbe_NilConfig(t *testing.T) {
	factory := NewFactory()
	
	// TCP с nil конфигурацией
	probe, err := factory.CreateProbe(types.ProbeTypeTCP, nil)
	if err == nil {
		t.Error("Expected error for nil TCP config")
	}
	if probe != nil {
		probe.Close()
	}
	
	// UDP с nil конфигурацией
	probe, err = factory.CreateProbe(types.ProbeTypeUDP, nil)
	if err == nil {
		t.Error("Expected error for nil UDP config")
	}
	if probe != nil {
		probe.Close()
	}
}

func TestFactory_GetSupportedTypes(t *testing.T) {
	factory := NewFactory()
	
	supportedTypes := factory.GetSupportedTypes()
	if len(supportedTypes) < 3 {
		t.Errorf("Expected at least 3 supported types, got %d", len(supportedTypes))
	}
	
	// Проверяем уникальность типов
	typeMap := make(map[types.ProbeType]bool)
	for _, probeType := range supportedTypes {
		if typeMap[probeType] {
			t.Errorf("Duplicate probe type: %s", probeType)
		}
		typeMap[probeType] = true
	}
}

