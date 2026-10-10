package core

import (
	. "github.com/BeamFoundry/rays/pkg/lang"
)

// BeamScopes is an [github.com/BeamFoundry/rays/pkg/lang.Enum] struct
type BeamScopes struct {
	SINGLETON,
	PROTOTYPE Enum[BeamScopes]
}

// BeamScope is a type alias for Enum[BeamScopes].
type BeamScope = Enum[BeamScopes]

type BeamScopeAnnotation interface {
	_BeamScopeAnnotation() string
}

// Singleton is a marker interface to be used to identify a singleton [Beam].
//
// Completely optional, as it's the default scope of a [Beam].
type Singleton interface {
	BeamScopeAnnotation
	_Singleton()
}

// Prototype is a marker interface to be used to identify a prototype [Beam].
type Prototype interface {
	BeamScopeAnnotation
	_Prototype()
}

// PrototypeContextKeyType is the type representing a context key type for default
// [Prototype] context values.
type PrototypeContextKeyType string

const (
	// PrototypeContextParentType represent the key used in the context to indicate a parent [reflect.Type].
	//
	// Please see [BeamPostConstructWithContext].
	PrototypeContextParentType PrototypeContextKeyType = "prototypeContextParentType"

	// PrototypeContextParentInstance represent the key used in the context to indicate a parent instance (any).
	//
	// Please see [BeamPostConstructWithContext].
	PrototypeContextParentInstance PrototypeContextKeyType = "prototypeContextParentInstance"

	// PrototypeContextProxyType is a predefined key to be used in a context to represent an additional
	// level of parentship, between an [InnerBeamPostConstruct] embbeded structure parent and a [Prototype] Beam.
	PrototypeContextProxyType PrototypeContextKeyType = "prototypeContextProxyType"

	// PrototypeContextProxyInstance is a predefined key to be used in a context to represent an additional
	// level of parentship, between an [InnerBeamPostConstruct] embbeded structure parent and a [Prototype] Beam.
	PrototypeContextProxyInstance PrototypeContextKeyType = "prototypeContextProxyInstance"
)
