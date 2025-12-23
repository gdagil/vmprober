package interfaces

import (
	"context"

	"github.com/vmprober/vmprober/internal/types"
)

// Probe интерфейс для всех типов проб
type Probe interface {
	// Execute выполняет пробу
	Execute(ctx context.Context, target types.Target) (*types.ProbeResult, error)
	
	// Type возвращает тип пробы
	Type() types.ProbeType
	
	// Validate проверяет конфигурацию пробы
	Validate(config interface{}) error
	
	// Close освобождает ресурсы
	Close() error
}

// ProbeFactory фабрика для создания проб
type ProbeFactory interface {
	CreateProbe(probeType types.ProbeType, config interface{}) (Probe, error)
	GetSupportedTypes() []types.ProbeType
}

