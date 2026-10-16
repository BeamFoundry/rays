package container

import (
	"context"
	"reflect"

	. "github.com/BeamFoundry/rays/pkg/core"
	. "github.com/BeamFoundry/rays/pkg/lang"
)

type ConfigurationPropertiesBeamProvider struct {
	ConfigurationProperties
}

var _ BeamProvider = &ConfigurationPropertiesBeamProvider{}

var _, _ = any(&ConfigurationPropertiesBeamProvider{}).(BeamProvider)

// ProvideBeam implements [core.BeamProvider].
func (this *ConfigurationPropertiesBeamProvider) ProvideBeam(reflectType reflect.Type, container InitializingContext) (parent Beam, children []Beam) {
	children = []Beam{}

	foundry := func(context.Context) (any, Error) {
		ptr := reflect.New(reflectType.Elem()).Interface()
		if field := FindFieldWithAnnotation[ConfigurationProperties](reflectType); field != nil {
			if path, ok := field.Tag.Lookup(NameAttribute); ok && path != "" {
				if err := container.ConfigUnmarshall(path, ptr); err != nil {
					if err.Name() == "KeyNotFoundError" {
						return ptr, nil
					}
				}
			}
		}
		return ptr, nil
	}
	//nolint:errcheck // Will not fail
	builder, _ := NewBeamBuilder(reflectType)
	parent = builder.WithFoundry(foundry).Build()

	return parent, children
}
