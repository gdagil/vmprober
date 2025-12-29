package config

import (
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/gdagil/vmprober/internal/types"
)

func TestTargetConfig_GetProtocols(t *testing.T) {
	tests := []struct {
		name      string
		protocols []string
		expected  []types.ProbeType
	}{
		{
			name:      "single protocol",
			protocols: []string{"tcp"},
			expected:  []types.ProbeType{types.ProbeTypeTCP},
		},
		{
			name:      "multiple protocols",
			protocols: []string{"tcp", "udp", "icmp"},
			expected:  []types.ProbeType{types.ProbeTypeTCP, types.ProbeTypeUDP, types.ProbeTypeICMP},
		},
		{
			name:      "empty protocols",
			protocols: []string{},
			expected:  []types.ProbeType{types.ProbeTypeTCP}, // Default to TCP
		},
		{
			name:      "nil protocols",
			protocols: nil,
			expected:  []types.ProbeType{types.ProbeTypeTCP}, // Default to TCP
		},
		{
			name:      "http protocol",
			protocols: []string{"http"},
			expected:  []types.ProbeType{types.ProbeTypeHTTP},
		},
		{
			name:      "https protocol",
			protocols: []string{"https"},
			expected:  []types.ProbeType{types.ProbeTypeHTTPS},
		},
		{
			name:      "dns protocol",
			protocols: []string{"dns"},
			expected:  []types.ProbeType{types.ProbeTypeDNS},
		},
		{
			name:      "grpc protocol",
			protocols: []string{"grpc"},
			expected:  []types.ProbeType{types.ProbeTypeGRPC},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := &TargetConfig{
				Protocols: tt.protocols,
			}

			result := tc.GetProtocols()

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d protocols, got %d", len(tt.expected), len(result))
				return
			}

			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("Protocol[%d]: expected %s, got %s", i, expected, result[i])
				}
			}
		})
	}
}

func TestTargetConfig_UnmarshalYAML(t *testing.T) {
	tests := []struct {
		name      string
		yaml      string
		protocols []string
	}{
		{
			name: "protocols array",
			yaml: `
host: example.com
port: 80
protocols:
  - tcp
  - udp
`,
			protocols: []string{"tcp", "udp"},
		},
		{
			name: "proto field (backward compatibility)",
			yaml: `
host: example.com
port: 80
proto: tcp
`,
			protocols: []string{"tcp"},
		},
		{
			name: "protocols takes priority over proto",
			yaml: `
host: example.com
port: 80
proto: tcp
protocols:
  - udp
  - icmp
`,
			protocols: []string{"udp", "icmp"},
		},
		{
			name: "no protocols specified",
			yaml: `
host: example.com
port: 80
`,
			protocols: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var tc TargetConfig
			err := yaml.Unmarshal([]byte(tt.yaml), &tc)
			if err != nil {
				t.Fatalf("UnmarshalYAML failed: %v", err)
			}

			if tc.Host != "example.com" {
				t.Errorf("Expected host 'example.com', got '%s'", tc.Host)
			}

			if tc.Port != 80 {
				t.Errorf("Expected port 80, got %d", tc.Port)
			}

			if len(tc.Protocols) != len(tt.protocols) {
				t.Errorf("Expected %d protocols, got %d", len(tt.protocols), len(tc.Protocols))
				return
			}

			for i, expected := range tt.protocols {
				if tc.Protocols[i] != expected {
					t.Errorf("Protocol[%d]: expected %s, got %s", i, expected, tc.Protocols[i])
				}
			}
		})
	}
}

func TestTargetConfig_UnmarshalYAML_InvalidYAML(t *testing.T) {
	var tc TargetConfig
	invalidYAML := `host: example.com: port: [invalid`

	err := yaml.Unmarshal([]byte(invalidYAML), &tc)
	if err == nil {
		t.Error("Expected error for invalid YAML")
	}
}
