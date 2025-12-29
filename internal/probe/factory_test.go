package probe

import (
	"testing"
	"time"

	"github.com/gdagil/vmprober/internal/types"
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

	// Verify that all probe types are supported
	expectedTypes := map[types.ProbeType]bool{
		types.ProbeTypeTCP:   false,
		types.ProbeTypeUDP:   false,
		types.ProbeTypeICMP:  false,
		types.ProbeTypeHTTP:  false,
		types.ProbeTypeHTTPS: false,
		types.ProbeTypeDNS:   false,
		types.ProbeTypeGRPC:  false,
	}

	for _, probeType := range supportedTypes {
		if _, exists := expectedTypes[probeType]; exists {
			expectedTypes[probeType] = true
		}
	}

	for probeType, found := range expectedTypes {
		if !found {
			t.Errorf("Expected %s probe type to be supported", probeType)
		}
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

func TestFactory_CreateProbe_HTTP(t *testing.T) {
	factory := NewFactory()

	config := &HTTPConfig{
		Method:             "GET",
		ExpectedStatusCode: 200,
	}

	probe, err := factory.CreateProbe(types.ProbeTypeHTTP, config)
	if err != nil {
		t.Fatalf("CreateProbe failed: %v", err)
	}
	if probe == nil {
		t.Fatal("CreateProbe returned nil")
	}
	if probe.Type() != types.ProbeTypeHTTP {
		t.Errorf("Expected HTTP probe, got %s", probe.Type())
	}

	probe.Close()
}

func TestFactory_CreateProbe_HTTPS(t *testing.T) {
	factory := NewFactory()

	config := &HTTPConfig{
		Method: "GET",
		TLS:    true,
	}

	probe, err := factory.CreateProbe(types.ProbeTypeHTTPS, config)
	if err != nil {
		t.Fatalf("CreateProbe failed: %v", err)
	}
	if probe == nil {
		t.Fatal("CreateProbe returned nil")
	}
	if probe.Type() != types.ProbeTypeHTTPS {
		t.Errorf("Expected HTTPS probe, got %s", probe.Type())
	}

	probe.Close()
}

func TestFactory_CreateProbe_HTTPS_NilConfig(t *testing.T) {
	factory := NewFactory()

	// HTTPS with nil config should create a probe with TLS=true
	probe, err := factory.CreateProbe(types.ProbeTypeHTTPS, nil)
	if err != nil {
		t.Fatalf("CreateProbe failed: %v", err)
	}
	if probe == nil {
		t.Fatal("CreateProbe returned nil")
	}
	if probe.Type() != types.ProbeTypeHTTPS {
		t.Errorf("Expected HTTPS probe, got %s", probe.Type())
	}

	probe.Close()
}

func TestFactory_CreateProbe_DNS(t *testing.T) {
	factory := NewFactory()

	config := &DNSConfig{
		QueryType: "A",
		Protocol:  "udp",
	}

	probe, err := factory.CreateProbe(types.ProbeTypeDNS, config)
	if err != nil {
		t.Fatalf("CreateProbe failed: %v", err)
	}
	if probe == nil {
		t.Fatal("CreateProbe returned nil")
	}
	if probe.Type() != types.ProbeTypeDNS {
		t.Errorf("Expected DNS probe, got %s", probe.Type())
	}

	probe.Close()
}

func TestFactory_CreateProbe_DNS_NilConfig(t *testing.T) {
	factory := NewFactory()

	// DNS with nil config should use defaults
	probe, err := factory.CreateProbe(types.ProbeTypeDNS, nil)
	if err != nil {
		t.Fatalf("CreateProbe failed: %v", err)
	}
	if probe == nil {
		t.Fatal("CreateProbe returned nil")
	}
	if probe.Type() != types.ProbeTypeDNS {
		t.Errorf("Expected DNS probe, got %s", probe.Type())
	}

	probe.Close()
}

func TestFactory_CreateProbe_GRPC(t *testing.T) {
	factory := NewFactory()

	config := &GRPCConfig{
		Service:        "test.Service",
		ExpectedStatus: "SERVING",
	}

	probe, err := factory.CreateProbe(types.ProbeTypeGRPC, config)
	if err != nil {
		t.Fatalf("CreateProbe failed: %v", err)
	}
	if probe == nil {
		t.Fatal("CreateProbe returned nil")
	}
	if probe.Type() != types.ProbeTypeGRPC {
		t.Errorf("Expected GRPC probe, got %s", probe.Type())
	}

	probe.Close()
}

func TestFactory_CreateProbe_GRPC_NilConfig(t *testing.T) {
	factory := NewFactory()

	// gRPC with nil config should use defaults
	probe, err := factory.CreateProbe(types.ProbeTypeGRPC, nil)
	if err != nil {
		t.Fatalf("CreateProbe failed: %v", err)
	}
	if probe == nil {
		t.Fatal("CreateProbe returned nil")
	}
	if probe.Type() != types.ProbeTypeGRPC {
		t.Errorf("Expected GRPC probe, got %s", probe.Type())
	}

	probe.Close()
}

func TestFactory_CreateProbe_UnsupportedType(t *testing.T) {
	factory := NewFactory()

	// Use a non-existent probe type
	unsupportedType := types.ProbeType("unsupported")
	_, err := factory.CreateProbe(unsupportedType, nil)
	if err == nil {
		t.Error("Expected error for unsupported probe type")
	}
}

func TestFactory_CreateProbe_InvalidConfig(t *testing.T) {
	factory := NewFactory()

	// Pass incorrect config type
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

	// TCP with nil config
	probe, err := factory.CreateProbe(types.ProbeTypeTCP, nil)
	if err == nil {
		t.Error("Expected error for nil TCP config")
	}
	if probe != nil {
		probe.Close()
	}

	// UDP with nil config
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
	// Now we have 7 types: TCP, UDP, ICMP, HTTP, HTTPS, DNS, GRPC
	if len(supportedTypes) < 7 {
		t.Errorf("Expected at least 7 supported types, got %d", len(supportedTypes))
	}

	// Check type uniqueness
	typeMap := make(map[types.ProbeType]bool)
	for _, probeType := range supportedTypes {
		if typeMap[probeType] {
			t.Errorf("Duplicate probe type: %s", probeType)
		}
		typeMap[probeType] = true
	}
}

func TestFactory_CreateProbe_InvalidConfigTypes(t *testing.T) {
	factory := NewFactory()

	// Pass incorrect config types for new probes
	tests := []struct {
		name      string
		probeType types.ProbeType
		config    interface{}
	}{
		{"HTTP with TCP config", types.ProbeTypeHTTP, &TCPConfig{}},
		{"DNS with HTTP config", types.ProbeTypeDNS, &HTTPConfig{}},
		{"GRPC with DNS config", types.ProbeTypeGRPC, &DNSConfig{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := factory.CreateProbe(tt.probeType, tt.config)
			if err == nil {
				t.Errorf("Expected error for invalid config type in %s", tt.name)
			}
		})
	}
}
