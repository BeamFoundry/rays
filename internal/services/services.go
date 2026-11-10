package services

import (
	"sync/atomic"

	"github.com/BeamFoundry/rays/internal/boot/logger"
)

// Registry holds references to cross-cutting services initialized during boot.
type Registry struct {
	Logger *logger.Logger
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

// Reset restores the default registry with a fallback Logger.
func Reset() {
	current.Store(&Registry{
		Logger: logger.NewLogger(),
	})
}
