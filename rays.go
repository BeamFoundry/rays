package rays

import (
	"github.com/BeamFoundry/rays/internal/container"
	"github.com/BeamFoundry/rays/pkg/core"
)

// ApplicationContext() returns a [core.ContainerConfiguration], letting you control Rays container initialization.
func ApplicationContext() core.ContainerConfiguration {
	return container.GetContainer()
}
