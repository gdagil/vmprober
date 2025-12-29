package interfaces

import (
	"context"

	"github.com/gdagil/vmprober/internal/types"
)

// Probe interface for all probe types
type Probe interface {
	// Execute performs the probe
	Execute(ctx context.Context, target types.Target) (*types.ProbeResult, error)
	
	// Type returns the probe type
	Type() types.ProbeType
	
	// Validate validates probe configuration
	Validate(config interface{}) error
	
	// Close releases resources
	Close() error
}

// ProbeFactory factory for creating probes
type ProbeFactory interface {
	CreateProbe(probeType types.ProbeType, config interface{}) (Probe, error)
	GetSupportedTypes() []types.ProbeType
}

