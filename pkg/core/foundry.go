package core

import (
	"context"

	. "github.com/BeamFoundry/rays/pkg/lang"
)

// Foundry is one of the generic struct that can be directly injected with [lang.Optional].
//
// If a [Beam] of type T can be injected, this struct would have that [Beam] foundry injected and be accessible.
//
// This can be used to access and create [Prototype] [Beam].
type Foundry[T any] struct {
	beamType T
	furnace  func(context.Context) (any, Error)
}

// Forge will return an instance of type T. You can create a [context.Context] with your values added to it.
// if used as intended, injected, the container engine will add the following values to it :
//
//   - [PrototypeContextParentType] : [reflect.Type] of the parent object it is inject to.
//
//   - [PrototypeContextParentInstance] : (any) current instance of the parent object.
func (this *Foundry[T]) Forge(ctx context.Context) (T, Error) {
	if this.furnace == nil {
		return this.beamType, NewError("FoundryError", "Empty factory")
	} else if v, err := this.furnace(ctx); err != nil {
		return this.beamType, err
	} else if value, ok := v.(T); ok {
		return value, nil
	}
	return this.beamType, NewError("FoundryError", "Unable to cast")
}

// NewFoundry() will create a [Foundry] for a type T.
//
// Only used in testing. You shouldn't use this directly.
func NewFoundry[T any](furnace func(context.Context) (any, Error)) Foundry[T] {
	return Foundry[T]{furnace: furnace}
}
