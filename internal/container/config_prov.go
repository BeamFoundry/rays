package container

import (
	"context"
	"reflect"

	"github.com/BeamFoundry/rays/internal/logger"

	. "github.com/BeamFoundry/rays/pkg/core"
	. "github.com/BeamFoundry/rays/pkg/lang"
	. "github.com/BeamFoundry/rays/pkg/util"
)

type ConfigurationBeamProvider struct {
	Configuration
}

var _ BeamProvider = &ConfigurationBeamProvider{}

var _, _ = any(&ConfigurationBeamProvider{}).(BeamProvider)

// ProvideBeam implements [core.BeamProvider].
func (this *ConfigurationBeamProvider) ProvideBeam(reflectType reflect.Type, container InitializingContext) (parent Beam, beams []Beam) {
	beams = []Beam{}
	builder, err := NewBeamBuilder(reflectType)
	if err != nil {
		err.LogError()
		return nil, beams
	}
	// TODO: Force it to be a singleton ???
	parent = builder.Build()

	logger.GetLogger().Debug("Evaluating registration beam with Configuration Stereotype", "beam", parent.Attributes()["Qualifier"], "methods", parent.Type().NumMethod())
	for mt := range parent.Type().Methods() {
		switch {
		case mt.Type.NumOut() == 0 || mt.Type.NumOut() > 2:
			continue
		case mt.Name == "New":
			continue
		case mt.Type.Out(0).Implements(reflect.TypeFor[error]()):
			continue
		default:
			switch mt.Type.Out(0).Kind() {
			case reflect.Pointer:
				if mt.Type.Out(0).Elem().Kind() != reflect.Struct {
					continue
				}
			case reflect.Interface:
			default:
				continue
			}
		}
		logger.GetLogger().Trace("Beam with Configuration Stereotype, looking method", "name", mt.Name, "func", mt.Type)
		qualifier := CaseUtils(mt.Name).Uncapitalize().String()
		annotatedType := parent.Type()

		scope := BeamScope{}.New().SINGLETON
		shift := 1

		aType, aValue := container.GetMethodAnnotedType(&mt)
		if aValue.IsValid() {
			logger.GetLogger().Debug("Beam has AnnotedValue", "qualifier", qualifier, "type", mt.Type.Out(0))
			annotatedType = aType
			shift++
		}

		switch {
		case mt.Type.NumIn() > shift && mt.Type.In(shift) == reflect.TypeFor[context.Context]():
			logger.GetLogger().Debug("Adding Prototype Beam", "qualifier", qualifier, "type", mt.Type.Out(0))
			scope = BeamScope{}.New().PROTOTYPE
		case mt.Type.NumIn() > shift:
			logger.GetLogger().Debug("Skipping Beam", "qualifier", qualifier, "type", mt.Type.Out(0))
			continue
		default:
			logger.GetLogger().Debug("Adding Singleton Beam", "qualifier", qualifier, "type", mt.Type.Out(0))
		}

		// TODO: Exclude if Prototype and type implements BeamPostConstructWithContext
		foundry := func(ctx context.Context) (any, Error) {
			instance := container.GetInstance(parent)
			md := reflect.ValueOf(instance).Method(mt.Index)
			in := []reflect.Value{}
			if aValue.IsValid() {
				//nolint:errcheck // Error may be not checked as "aValue" is created by GetMethodAnnotedType
				// and we are inside a foundry method. Injection impossibility will be detected before.
				container.InjectMethodAnnotedValue(aValue, instance)
				in = append(in, aValue)
			}
			if scope == (BeamScope{}.New().PROTOTYPE) {
				in = append(in, reflect.ValueOf(ctx))
			}
			ret := md.Call(in)
			if len(ret) > 1 {
				if err, ok := ret[1].Interface().(Error); ok && err != nil {
					return nil, err
				} else if err, ok := ret[1].Interface().(error); ok && err != nil {
					return nil, NewErrorFrom(err)
				}
			}
			return ret[0].Interface(), nil
		}
		if builder, err := NewBeamBuilder(mt.Type.Out(0)); err == nil {
			builder.WithQualifier(qualifier).WithFoundry(foundry).WithScope(scope)
			builder.WithAnnotatedType(annotatedType)
			beams = append(beams, builder.Build())
		}
	}
	return parent, beams
}
