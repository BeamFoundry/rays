package container

import (
	"context"
	"io"
	"log/slog"
	"maps"
	"reflect"
	"slices"

	"github.com/BeamFoundry/rays/internal/logger"
	. "github.com/BeamFoundry/rays/pkg/core"
	. "github.com/BeamFoundry/rays/pkg/lang"
)

type LogLevel struct {
	Enum[LogLevel]
	INFO, // Default level
	TRACE,
	DEBUG,
	WARN,
	ERROR Enum[LogLevel]
}

func (this *LogLevel) Level() slog.Level {
	switch this.Enum {
	case LogLevel{}.New().TRACE:
		return LevelTrace
	case LogLevel{}.New().DEBUG:
		return LevelDebug
	case LogLevel{}.New().WARN:
		return LevelWarn
	case LogLevel{}.New().ERROR:
		return LevelError
	default:
		return LevelInfo
	}
}

type LogHandler struct {
	Enum[LogHandler]
	DEFAULT,
	JSON,
	TEXT Enum[LogHandler]
}

func (this *LogHandler) Handler() func(io.Writer, *slog.HandlerOptions) slog.Handler {
	switch this.Enum {
	case LogHandler{}.New().JSON:
		return func(writer io.Writer, options *slog.HandlerOptions) slog.Handler {
			return slog.NewJSONHandler(writer, options)
		}
	case LogHandler{}.New().TEXT:
		return func(writer io.Writer, options *slog.HandlerOptions) slog.Handler {
			return slog.NewTextHandler(writer, options)
		}
	}
	return nil
}

type LoggingProperties struct {
	ConfigurationProperties `Name:"rays.logging"`
	// Loggers
	loggers map[string]LogLevel
	/* configuration keys */
	// Set rays.logging.handler to "DEFAULT", "TEXT" or "JSON"
	handler LogHandler
	// Set rays.logging.level to "TRACE", "DEBUG", "INFO", "WARN" or "ERROR".
	// Defaults to "INFO"
	level LogLevel
	// Add Logging source
	source bool
}

func (this *LoggingProperties) LogLevel() LogLevel {
	return this.level
}

func (this *LoggingProperties) LogHandler() LogHandler {
	return this.handler
}

func (this *LoggingProperties) AddSource() bool {
	return this.source
}

func (this *LoggingProperties) MapLevels() map[string]LogLevel {
	return this.loggers
}

type logging struct {
	*logger.Logger
}

func (this logging) With(args ...any) Logging {
	this.Logger = this.Logger.With(args...)
	return &this
}

func (this logging) WithGroup(name string) Logging {
	this.Logger = this.Logger.WithGroup(name)
	return &this
}

type LoggingConfiguration struct {
	Configuration
	properties *LoggingProperties `@:"Inject"`
	loggers    []string
}

var _, _ = any(&LoggingConfiguration{}).(Stereotype)

func (this *LoggingConfiguration) PostConstruct() Error {
	this.loggers = slices.Sorted(maps.Keys(this.properties.MapLevels()))
	slices.Reverse(this.loggers)
	logger.GetLogger().Debug("LoggingConfiguration::PostConstruct()", "loggers", this.loggers)
	return nil
}

//nolint:contextcheck //TODO: I should remove this check.
func (this *LoggingConfiguration) Logger(ctx context.Context) Logging {
	if ct := ctx.Value(PrototypeContextParentType); ct != nil {
		if rt, ok := ct.(reflect.Type); ok {
			if rt.Kind() == reflect.Pointer {
				rt = rt.Elem()
			}
			pkg := rt.PkgPath()
			name := pkg + "/" + rt.Name()
			getLogger := func(level string) Logging {
				logger.GetLogger().Trace("Creating specific new logger", "pkg", pkg, rt.PkgPath(), rt.Name())
				lg := logger.GetLogger().Clone()
				logLevel := this.properties.MapLevels()[level]
				lg.SetLogLevel(logLevel.Level())
				return &logging{lg}
			}
			for _, level := range this.loggers {
				if (String(level).EndsWith("/") && String(pkg).StartsWith(level)) || pkg == level || name == level {
					return getLogger(level)
				}
			}
		}
	}
	return &logging{logger.GetLogger()}
}

func init() {
	loggingProperties := map[string]any{
		"handler": LogHandler{}.New().DEFAULT.String(),
		"level":   LogLevel{}.New().INFO.String(),
		"source":  true,
	}

	//nolint:errcheck // it can not fail
	GetContainer().LoadDefaultConfigAt(loggingProperties, "rays.logging")
}
