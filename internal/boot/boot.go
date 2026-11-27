// Package boot provides application bootstrap, configuration lifecycle
// management, and logging initialization for Rays.
package boot

import (
	"io"
	"log/slog"
	"maps"
	"reflect"
	"slices"
	"sync"
	"sync/atomic"

	"github.com/BeamFoundry/rays/internal/boot/config"
	"github.com/BeamFoundry/rays/internal/boot/logger"
	"github.com/BeamFoundry/rays/internal/boot/scheduler"
	"github.com/BeamFoundry/rays/internal/services"
	. "github.com/BeamFoundry/rays/pkg/core"
	. "github.com/BeamFoundry/rays/pkg/lang"
)

// BootLoader coordinates application initialization by orchestrating configuration
// loading, logging setup, scheduler initialization, and dependency service registration across bootstrap phases.
type BootLoader struct {
	*config.Config
	loggingProperties   *LoggingProperties
	schedulerProperties *SchedulerProperties
	handler             func(io.Writer, *slog.HandlerOptions) slog.Handler
	logger              *logger.Logger
	scheduler           *scheduler.Scheduler
	loggers             []string
	lock                sync.Mutex
	booted              atomic.Int32
}

// NewBootLoader creates and initializes a new BootLoader instance configured with
// default logging and scheduler properties and a quiet error-level logger for initial bootstrap.
func NewBootLoader() *BootLoader {
	cfg := config.NewConfig()
	loggingDefaults := map[string]any{
		"handler": LogHandler{}.New().DEFAULT.String(),
		"level":   LogLevel{}.New().INFO.String(),
		"source":  true,
		"format":  logger.DefaultFormat,
		"color":   LogColor{}.New().AUTO.String(),
	}
	//nolint:errcheck // default config load at path cannot fail
	cfg.LoadDefaultConfigAt(loggingDefaults, "rays.logging")

	schedulerDefaults := map[string]any{
		"pool": scheduler.DefaultPoolSize,
	}
	//nolint:errcheck // default config load at path cannot fail
	cfg.LoadDefaultConfigAt(schedulerDefaults, "rays.scheduler")

	quietLogger := logger.NewLogger()
	quietLogger.SetLogLevel(slog.LevelError)
	cfg.SetLogger(quietLogger)

	defaultScheduler := scheduler.NewScheduler(scheduler.DefaultPoolSize)
	defaultScheduler.SetLogger(quietLogger)
	services.Register(&services.Registry{
		Logger:    quietLogger,
		Scheduler: defaultScheduler,
	})

	return &BootLoader{
		Config:    cfg,
		logger:    quietLogger,
		scheduler: defaultScheduler,
		loggers:   []string{},
	}
}

func (this *BootLoader) isBooted() bool {
	return this.booted.Load() == 1
}

// SetLoggingHandler sets a custom logging handler factory function to override the
// default slog handler constructed during bootstrap. Calls after Boot are ignored.
func (this *BootLoader) SetLoggingHandler(handler func(io.Writer, *slog.HandlerOptions) slog.Handler) {
	if this.isBooted() {
		return
	}
	this.lock.Lock()
	defer this.lock.Unlock()
	this.handler = handler
}

// WrapLoggingHandler registers a wrapper function to decorate the active slog.Handler.
// Wrappers are applied in registration order around the configured logging handler,
// including any loggers already cloned for specific packages or types.
func (this *BootLoader) WrapLoggingHandler(wrapper func(slog.Handler) slog.Handler) {
	this.logger.WrapHandler(wrapper)
}

// LoggingProperties returns the unmarshaled logging configuration properties
// populated during the bootstrap sequence, or nil if Boot has not yet been invoked.
func (this *BootLoader) LoggingProperties() *LoggingProperties {
	return this.loggingProperties
}

// SchedulerProperties returns the unmarshaled scheduler configuration properties
// populated during the bootstrap sequence, or nil if Boot has not yet been invoked.
func (this *BootLoader) SchedulerProperties() *SchedulerProperties {
	return this.schedulerProperties
}

// Scheduler returns a TaskScheduler proxy backed by the scheduler managed by this BootLoader.
func (this *BootLoader) Scheduler() *TaskScheduler {
	this.lock.Lock()
	defer this.lock.Unlock()
	return NewTaskScheduler(this.scheduler)
}

// Boot executes the bootstrap sequence:
// 1. Initializes configuration files and profiles in a quiet phase.
// 2. Extracts and unmarshals the "rays.logging" configuration section to reconfigure the logger.
// 3. Extracts and unmarshals the "rays.scheduler" configuration section to initialize the task scheduler.
// 4. Registers the configured logger and scheduler in the global services registry.
// 5. Re-initializes the configuration container with full logging enabled.
//
// Returns an error if already booted or if configuration parsing or unmarshaling fails.
func (this *BootLoader) Boot() Error {
	if this.isBooted() {
		return NewError("InitializationError", "BootLoader is already booted")
	}
	this.lock.Lock()
	defer this.lock.Unlock()

	// 1. Initializing configuration in Phase 1 (quiet)
	if err := this.InitConfig(); err != nil {
		return err
	}

	// 2. Initializing logging properties from parsed config
	if cfg, err := this.GetSubConfig("rays.logging"); err == nil {
		this.loggingProperties = &LoggingProperties{}
		if err := cfg.Unmarshal(this.loggingProperties); err != nil {
			return err
		}
		level := this.loggingProperties.LogLevel()
		this.logger.SetLogLevel(level.Level())
		if this.loggingProperties.AddSource() {
			this.logger.AddSource()
		}
		if this.handler != nil {
			this.logger.SetHandler(this.handler)
		} else if handler := this.loggingProperties.LogHandler(); handler.Handler(this.loggingProperties.Format()) != nil {
			this.logger.SetHandler(handler.Handler(this.loggingProperties.Format()))
		}
		this.loggingProperties.LogColor().Apply(this.logger)
		this.loggers = slices.Sorted(maps.Keys(this.loggingProperties.MapLevels()))
		slices.Reverse(this.loggers)
	}
	this.logger.SetDefault()

	// 3. Initializing scheduler properties from parsed config
	if cfg, err := this.GetSubConfig("rays.scheduler"); err == nil {
		this.schedulerProperties = &SchedulerProperties{}
		if err := cfg.Unmarshal(this.schedulerProperties); err != nil {
			return err
		}
		this.scheduler.Stop()
		this.scheduler = scheduler.NewScheduler(this.schedulerProperties.Pool())
		this.scheduler.SetLogger(this.logger)
	}

	// 4. Register configured logger and scheduler into services
	services.Register(&services.Registry{
		Logger:    this.logger,
		Scheduler: this.scheduler,
	})

	// 5. Reinitialize Config with the correct parameters (Phase 4)
	this.Clear()
	this.SetLogger(this.logger)
	if err := this.InitConfig(); err != nil {
		return err
	}

	this.booted.Store(1)
	this.logger.Debug("Config, Logging & Scheduler initialized")
	return nil
}

// Logger returns the root application logger as a core.Logging instance.
func (this *BootLoader) Logger() Logging {
	return &logging{this.logger}
}

// LoggerForClass returns a specialized core.Logging instance configured for the package
// or type of the provided struct pointer, respecting package-specific log level overrides.
func (this *BootLoader) LoggerForClass(class any) Logging {
	rtype := reflect.TypeOf(class)
	return this.LoggerForType(rtype)
}

// matchLogLevel searches this.loggers for the first matching override for any of the given types,
// evaluating candidate types in the order supplied and choosing the most specific matching override.
func (this *BootLoader) matchLogLevel(types ...reflect.Type) (LogLevel, bool) {
	if this.loggingProperties == nil {
		return LogLevel{}, false
	}

	type typeInfo struct {
		pkg  string
		name string
	}

	var infos []typeInfo
	for _, rtype := range types {
		if rtype == nil {
			continue
		}
		if rtype.Kind() == reflect.Pointer && rtype.Elem().Kind() == reflect.Struct {
			infos = append(infos, typeInfo{
				pkg:  rtype.Elem().PkgPath(),
				name: rtype.Elem().PkgPath() + "/" + rtype.Elem().Name(),
			})
		} else if rtype.Kind() == reflect.Struct || rtype.Kind() == reflect.Interface {
			infos = append(infos, typeInfo{
				pkg:  rtype.PkgPath(),
				name: rtype.PkgPath() + "/" + rtype.Name(),
			})
		}
	}

	if len(infos) == 0 {
		return LogLevel{}, false
	}

	for _, info := range infos {
		for _, level := range this.loggers {
			if (!String(level).EndsWith("/") || !String(info.pkg).StartsWith(level)) && info.pkg != level && info.name != level {
				continue
			}
			return this.loggingProperties.MapLevels()[level], true
		}
	}
	return LogLevel{}, false
}

// LoggerForType returns a specialized core.Logging instance configured for the provided
// reflect.Type, applying the most specific log level override defined for its package path or type.
func (this *BootLoader) LoggerForType(rtype reflect.Type) Logging {
	if logLevel, ok := this.matchLogLevel(rtype); ok {
		this.logger.Trace("Creating specific new logger", "type", rtype.String())
		lg := this.logger.Clone()
		lg.SetLogLevel(logLevel.Level())
		return &logging{lg}
	}
	return this.Logger()
}

// LoggerForTypes returns a specialized core.Logging instance configured for the first
// matching reflect.Type among candidate types, applying the most specific log level override.
func (this *BootLoader) LoggerForTypes(types ...reflect.Type) Logging {
	if logLevel, ok := this.matchLogLevel(types...); ok {
		this.logger.Trace("Creating specific new logger for types")
		lg := this.logger.Clone()
		lg.SetLogLevel(logLevel.Level())
		return &logging{lg}
	}
	return this.Logger()
}

// Reset restores the BootLoader, its underlying configuration container, logging,
// and scheduler state back to their initial unbooted condition.
func (this *BootLoader) Reset() {
	this.lock.Lock()
	oldScheduler := this.scheduler
	this.lock.Unlock()

	// Stop running tasks without holding this.lock to prevent deadlocks if a task
	// accesses BootLoader methods while Reset is waiting for tasks to finish.
	oldScheduler.Stop()

	this.lock.Lock()
	defer this.lock.Unlock()
	this.Config.Reset()
	this.handler = nil
	this.loggingProperties = nil
	this.schedulerProperties = nil
	this.loggers = []string{}
	this.logger.Reset()
	this.logger.SetLogLevel(slog.LevelError)
	this.logger.SetDefault()
	this.SetLogger(this.logger)
	this.scheduler = scheduler.NewScheduler(scheduler.DefaultPoolSize)
	this.scheduler.SetLogger(this.logger)
	this.booted.Store(0)
	services.Register(&services.Registry{
		Logger:    this.logger,
		Scheduler: this.scheduler,
	})
}
