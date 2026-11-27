package services

import (
	"sync/atomic"

	"github.com/BeamFoundry/rays/internal/boot/logger"
	"github.com/BeamFoundry/rays/internal/boot/scheduler"
)

// Registry holds references to cross-cutting services initialized during boot.
type Registry struct {
	Logger    *logger.Logger
	Scheduler *scheduler.Scheduler
}

var current atomic.Pointer[Registry]

func init() {
	Reset()
}

// Get returns the currently active service registry.
func Get() *Registry {
	return current.Load()
}

// Register sets the global service registry. If r is nil, it does nothing.
func Register(r *Registry) {
	if r != nil {
		current.Store(r)
	}
}

// Reset restores the default registry with a fallback Logger and Scheduler.
func Reset() {
	current.Store(&Registry{
		Logger:    logger.NewLogger(),
		Scheduler: scheduler.NewScheduler(scheduler.DefaultPoolSize),
	})
}
