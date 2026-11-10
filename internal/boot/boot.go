package boot

import (
	"io"
	"log/slog"
	"maps"
	"reflect"
	"slices"
	"sync"

	"github.com/BeamFoundry/rays/internal/boot/config"
	"github.com/BeamFoundry/rays/internal/boot/logger"
	"github.com/BeamFoundry/rays/internal/services"
	. "github.com/BeamFoundry/rays/pkg/core"
	. "github.com/BeamFoundry/rays/pkg/lang"
)

type BootLoader struct {
	*config.Config
	loggingProperties *LoggingProperties
	handler           func(io.Writer, *slog.HandlerOptions) slog.Handler
	logger            *logger.Logger
	loggers           []string
	lock              sync.Mutex
}

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

	quietLogger := logger.NewLogger()
	quietLogger.SetLogLevel(slog.LevelError)
	cfg.SetLogger(quietLogger)
	services.Register(&services.Registry{
		Logger: quietLogger,
	})

	return &BootLoader{
		Config:  cfg,
		logger:  quietLogger,
		loggers: []string{},
	}
}

func (this *BootLoader) SetLoggingHandler(handler func(io.Writer, *slog.HandlerOptions) slog.Handler) {
	this.lock.Lock()
	defer this.lock.Unlock()
	this.handler = handler
}

func (this *BootLoader) LoggingProperties() *LoggingProperties {
	return this.loggingProperties
}

func (this *BootLoader) Boot() Error {
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

	// 3. Register configured logger into services
	services.Register(&services.Registry{
		Logger: this.logger,
	})

	// 4. Primary functionality hooks (future extensions)

	// 5. Reinitialize Config with the correct parameters (Phase 4)
	this.Clear()
	this.SetLogger(this.logger)
	//nolint:errcheck // Phase 1 has already validated configuration loading
	_ = this.InitConfig()

	this.logger.Debug("Config & Logging initialized")
	return nil
}

func (this *BootLoader) Logger() Logging {
	return &logging{this.logger}
}

func (this *BootLoader) LoggerForClass(class any) Logging {
	rtype := reflect.TypeOf(class)
	return this.LoggerForType(rtype)
}

func (this *BootLoader) LoggerForType(rtype reflect.Type) Logging {
	if this.loggingProperties == nil || rtype.Kind() != reflect.Pointer || rtype.Elem().Kind() != reflect.Struct {
		return this.Logger()
	}

	pkg := rtype.Elem().PkgPath()
	name := pkg + "/" + rtype.Elem().Name()
	for _, level := range this.loggers {
		if (!String(level).EndsWith("/") || !String(pkg).StartsWith(level)) && pkg != level && name != level {
			continue
		}
		this.logger.Trace("Creating specific new logger", "pkg", pkg, "name", name)
		lg := this.logger.Clone()
		logLevel := this.loggingProperties.MapLevels()[level]
		lg.SetLogLevel(logLevel.Level())
		return &logging{lg}
	}
	return this.Logger()
}

func (this *BootLoader) Reset() {
	this.lock.Lock()
	defer this.lock.Unlock()
	this.Config.Reset()
	this.handler = nil
	this.loggingProperties = nil
	this.loggers = []string{}
	this.logger.Reset()
	this.logger.SetLogLevel(slog.LevelError)
	this.SetLogger(this.logger)
	services.Register(&services.Registry{
		Logger: this.logger,
	})
}
