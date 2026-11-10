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

// Logger returns a contextual Logging instance tailored to the injecting bean's package and type.
func (this *LoggingConfiguration) Logger(ctx context.Context) Logging {
	if ct := ctx.Value(PrototypeContextParentType); ct != nil {
		if rt, ok := ct.(reflect.Type); ok {
			return GetContainer().LoggerForType(rt)
		}
	}
	return GetContainer().Logger()
}
