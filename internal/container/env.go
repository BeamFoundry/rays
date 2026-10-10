package container

import (
	. "github.com/BeamFoundry/rays/pkg/core"
	"github.com/BeamFoundry/rays/pkg/lang"
)

type environment struct{}

var _ Environment = &environment{}

// ActiveProfiles implements [core.Environment].
func (e *environment) ActiveProfiles() []string {
	return GetContainer().ActiveProfiles()
}

// ConfigUnmarshall implements [core.Environment].
func (e *environment) ConfigUnmarshall(path string, object any) lang.Error {
	return GetContainer().ConfigUnmarshall(path, object)
}

// ConfigValue implements [core.Environment].
func (e *environment) ConfigValue(path string) (string, bool) {
	return GetContainer().ConfigValue(path)
}

type EnvironmentConfiguration struct {
	Configuration
}

func (this *EnvironmentConfiguration) Environment() Environment {
	return &environment{}
}

var _, _ = any(&EnvironmentConfiguration{}).(Stereotype)
