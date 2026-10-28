package core

import (
	"context"
	"reflect"

	. "github.com/BeamFoundry/rays/internal/utils"
	. "github.com/BeamFoundry/rays/pkg/lang"
)

// DefaultBeam is the default [Beam] implementation.
//
// As this is a central type in the container architecture, it has been chosen to be
// represented by the unique [Beam] interface, with a complete implementation (this).
//
// You normally won't have to ever create a Beam yourself, and if so you should use the
// [BeamBuilder] which will return a [DefaultBeam].
type DefaultBeam struct {
	attributes map[string]string
	rType      reflect.Type
	aType      reflect.Type
	foundry    func(context.Context) (any, Error)
	scope      BeamScope
}

var _ Beam = &DefaultBeam{}

// Attributes implements [Beam].
func (this *DefaultBeam) Attributes() map[string]string {
	return this.attributes
}

// Type() implements [Beam.Type].
func (this *DefaultBeam) Type() reflect.Type {
	return this.rType
}

// AnnotatedType() implements [Beam.AnnotatedType]
func (this *DefaultBeam) AnnotatedType() reflect.Type {
	if this.aType != nil {
		return this.aType
	}
	return this.rType
}

// Foundry() implements [Beam.Foundry].
func (this *DefaultBeam) Foundry() func(context.Context) (any, Error) {
	return this.foundry
}

// Scope() implements [Beam.Scope].
func (this *DefaultBeam) Scope() BeamScope {
	return this.scope
}

// BeamBuilder represents a struct used to create a new [Beam].
type BeamBuilder struct {
	beam     *DefaultBeam
	instance any
}

// NewBeamBuilder() will return a new [BeamBuilder].
//
// Its argument may be :
//
//  1. A pointer or nil-pointer to a struct.
//
//  2. An interface.
//
//  3. A [reflect.Type] of one of those.
//
// If this is a non-nil pointer to a struct and this beam happen to be a
// Singleton [Beam], this pointer will be returned, which can be useful
// when used in testing.
func NewBeamBuilder(ptr any) (*BeamBuilder, Error) {
	var reflectType reflect.Type
	var instance any
	if rt, ok := ptr.(reflect.Type); ok {
		reflectType = rt
		instance = nil
	} else {
		reflectType = reflect.TypeOf(ptr)
		instance = ptr
	}

	builder := &BeamBuilder{
		beam: &DefaultBeam{
			attributes: map[string]string{},
			rType:      reflectType,
			aType:      nil,
			scope:      BeamScope{}.New().SINGLETON,
			foundry:    nil,
		},
		instance: instance,
	}

	if reflectType.Kind() == reflect.Interface {
		return builder, nil
	}
	if reflectType.Kind() != reflect.Pointer || reflectType.Elem().Kind() != reflect.Struct {
		return nil, NewError("InvalidType", "Bean must be a pointer to a struct")
	}

	builder.beam.attributes["Qualifier"] = Uncapitalize(reflectType.Elem().Name())

	if field := FindFieldWithAnnotation[Prototype](reflectType); field != nil {
		builder.beam.scope = BeamScope{}.New().PROTOTYPE
	} else if field := FindFieldWithAnnotation[BeamScopeAnnotation](reflectType); field != nil {
		if name, ok := field.Tag.Lookup("Name"); ok {
			if name == "Prototype" {
				builder.beam.scope = BeamScope{}.New().PROTOTYPE
			}
		}
	}

	// Keep support for testing manually registering struct pointer.
	if _, ok := ptr.(reflect.Type); !ok {
		p := ptr
		if reflect.ValueOf(ptr).IsNil() {
			p = builder.createInstance()
		}

		foundry := func(context.Context) (any, Error) {
			if builder.beam.scope == (BeamScope{}.New().PROTOTYPE) {
				return reflect.New(reflectType.Elem()).Interface(), nil
			}
			return p, nil
		}

		builder.WithFoundry(foundry)
	}

	return builder, nil
}

// WithAnnotatedType() let you specify another type to look for annotations.
func (this *BeamBuilder) WithAnnotatedType(aType reflect.Type) *BeamBuilder {
	this.beam.aType = aType
	return this
}

// WithQualifier() let you give a name to this [Beam], which can be used when injecting.
func (this *BeamBuilder) WithQualifier(qualifier string) *BeamBuilder {
	this.beam.attributes["Qualifier"] = qualifier
	return this
}

// WithFoundry() let you specify a function to be used to instantiate a new instance of this Beam.
func (this *BeamBuilder) WithFoundry(foundry func(context.Context) (any, Error)) *BeamBuilder {
	this.beam.foundry = foundry
	return this
}

// WithScope() let you specify a scope for this [Beam]
func (this *BeamBuilder) WithScope(scope BeamScope) *BeamBuilder {
	this.beam.scope = scope
	return this
}

func (this *BeamBuilder) createInstance() any {
	rt := this.beam.rType
	v := reflect.New(rt.Elem())

	if mt, ok := rt.MethodByName("New"); ok {
		if mt.Type.NumIn() == 1 && mt.Type.NumOut() == 1 && mt.Type.Out(0) == rt {
			v = v.Method(mt.Index).Call([]reflect.Value{})[0]
		}
	}

	return v.Interface()
}

// Build() will finally return a [Beam].
func (this *BeamBuilder) Build() Beam {
	attribute := reflect.TypeFor[BeamAttribute]()
	evaluator := reflect.TypeFor[BeamAttributeEvaluator]()
	reflectType := this.beam.AnnotatedType()

	// var findAttributes func(reflectType reflect.Type)
	// findAttributes = func(reflectType reflect.Type) {
	for field := range reflectType.Elem().Fields() {
		if !field.Anonymous {
			continue
		}
		//nolint:gocritic // Keeping it for now
		/*
			if field.Type.Kind() == reflect.Struct {
				findAttributes(reflect.PointerTo(field.Type))
			}
		*/
		if field.Type.Implements(attribute) && field.Type.Kind() == reflect.Interface {
			if evaluators := FindTypesImplementingInterface(evaluator, field.Type); len(evaluators) == 1 {
				if eval, ok := reflect.New(evaluators[0].Elem()).Interface().(BeamAttributeEvaluator); ok {
					value := eval.EvaluateBeamAttribute(this.beam)
					if value != "" {
						this.beam.attributes[field.Name] = eval.EvaluateBeamAttribute(this.beam)
					}
				}
			}
		}
	}
	// }
	// findAttributes(reflectType)

	if this.beam.foundry != nil {
		return this.beam
	}

	if this.beam.scope == (BeamScope{}.New().PROTOTYPE) {
		this.beam.foundry = func(context.Context) (any, Error) { return this.createInstance(), nil }
	} else {
		if this.instance == nil {
			this.instance = this.createInstance()
		}
		this.beam.foundry = func(context.Context) (any, Error) { return this.instance, nil }
	}

	return this.beam
}
