package container

import (
	"context"
	"reflect"

	"github.com/BeamFoundry/rays/internal/boot"
	"github.com/BeamFoundry/rays/internal/services"
	. "github.com/BeamFoundry/rays/pkg/core"
	. "github.com/BeamFoundry/rays/pkg/lang"
)

type (
	LogLevel          = boot.LogLevel
	LogHandler        = boot.LogHandler
	LogColor          = boot.LogColor
	LoggingProperties = boot.LoggingProperties
)

// LoggingConfiguration is a Stereotype configuration bean that manages logger initialization and injection.
type LoggingConfiguration struct {
	Configuration
	properties *boot.LoggingProperties `@:"Inject"`
}

var _, _ = any(&LoggingConfiguration{}).(Stereotype)

// PostConstruct initializes the global logger format and color according to loaded logging properties.
func (this *LoggingConfiguration) PostConstruct() Error {
	services.Get().Logger.Debug("LoggingConfiguration::PostConstruct()", "properties", this.properties)
	return nil
}

// Logger returns a contextual Logging instance tailored to the injecting bean's package and type,
// resolving concrete instances for interface-typed beans to retain interface or package overrides.
func (this *LoggingConfiguration) Logger(ctx context.Context) Logging {
	var parentType reflect.Type
	if ct := ctx.Value(PrototypeContextParentType); ct != nil {
		if rt, ok := ct.(reflect.Type); ok {
			parentType = rt
		}
	}

	var concreteType reflect.Type
	if ci := ctx.Value(PrototypeContextParentInstance); ci != nil {
		concreteType = reflect.TypeOf(ci)
	}

	if parentType != nil && parentType.Kind() == reflect.Interface {
		if concreteType != nil {
			return GetContainer().LoggerForTypes(concreteType, parentType)
		}
		return GetContainer().LoggerForType(parentType)
	}

	if concreteType != nil {
		return GetContainer().LoggerForType(concreteType)
	}
	if parentType != nil {
		return GetContainer().LoggerForType(parentType)
	}
	return GetContainer().Logger()
}
