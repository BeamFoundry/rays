package config

import (
	"github.com/BeamFoundry/rays/internal/boot/logger"
)

// Exported for internal package tests.
var (
	GetConfigLogger = func(c *Config) *logger.Logger {
		return c.logger
	}
	DeepCloneMap   = deepCloneMap
	DeepCloneSlice = deepCloneSlice
)
