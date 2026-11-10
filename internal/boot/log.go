package boot

import (
	"io"
	"log/slog"

	"github.com/BeamFoundry/rays/internal/boot/logger"
	. "github.com/BeamFoundry/rays/pkg/core"
	. "github.com/BeamFoundry/rays/pkg/lang"
)

// LogLevel defines the enumeration of supported logging severity levels.
type LogLevel struct {
	Enum[LogLevel]
	INFO, // Default level
	TRACE,
	DEBUG,
	WARN,
	ERROR,
	FATAL Enum[LogLevel]
}

// Level converts the LogLevel enumeration into the corresponding slog.Level value.
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
	case LogLevel{}.New().FATAL:
		return LevelFatal
	default:
		return LevelInfo
	}
}

// LogHandler defines the enumeration of supported logging handler implementations.
type LogHandler struct {
	Enum[LogHandler]
	DEFAULT,
	JSON,
	TEXT,
	ECS,
	GELF,
	LOGSTASH,
	CUSTOM Enum[LogHandler]
}

// Handler returns a factory function that constructs the slog.Handler corresponding to this LogHandler option.
func (this *LogHandler) Handler(format ...string) func(io.Writer, *slog.HandlerOptions) slog.Handler {
	customFormat := logger.DefaultFormat
	if len(format) > 0 && format[0] != "" {
		customFormat = format[0]
	}
	switch this.Enum {
	case LogHandler{}.New().DEFAULT:
		return func(writer io.Writer, options *slog.HandlerOptions) slog.Handler {
			return logger.NewHandlerWithFormat(writer, options, logger.DefaultFormat)
		}
	case LogHandler{}.New().JSON:
		return func(writer io.Writer, options *slog.HandlerOptions) slog.Handler {
			return logger.NewHandlerWithFormat(writer, options, logger.JSONFormat)
		}
	case LogHandler{}.New().TEXT:
		return func(writer io.Writer, options *slog.HandlerOptions) slog.Handler {
			return logger.NewHandlerWithFormat(writer, options, logger.TextFormat)
		}
	case LogHandler{}.New().CUSTOM:
		return func(writer io.Writer, options *slog.HandlerOptions) slog.Handler {
			return logger.NewHandlerWithFormat(writer, options, customFormat)
		}
	case LogHandler{}.New().ECS:
		return func(writer io.Writer, options *slog.HandlerOptions) slog.Handler {
			return logger.NewHandlerWithFormat(writer, options, logger.ECSFormat)
		}
	case LogHandler{}.New().GELF:
		return func(writer io.Writer, options *slog.HandlerOptions) slog.Handler {
			return logger.NewHandlerWithFormat(writer, options, logger.GELFFormat)
		}
	case LogHandler{}.New().LOGSTASH:
		return func(writer io.Writer, options *slog.HandlerOptions) slog.Handler {
			return logger.NewHandlerWithFormat(writer, options, logger.LogstashFormat)
		}
	default:
		return nil
	}
}

// LogColor defines the enumeration of terminal ANSI color output options (AUTO, ON, OFF).
type LogColor struct {
	Enum[LogColor]
	AUTO,
	ON,
	OFF Enum[LogColor]
}

// Apply configures the ANSI color output mode on the given logger based on the LogColor setting.
func (this LogColor) Apply(lg *logger.Logger) {
	switch this.Enum {
	case LogColor{}.New().ON:
		lg.SetColor(true)
	case LogColor{}.New().OFF:
		lg.SetColor(false)
	default:
		lg.EnableColor()
	}
}

// LoggingProperties holds configuration properties under the "rays.logging" namespace.
type LoggingProperties struct {
	ConfigurationProperties `Name:"rays.logging"`
	// Loggers
	loggers map[string]LogLevel
	// Log format for CUSTOM handler
	format string
	/* configuration keys */
	// Set rays.logging.handler to "DEFAULT", "TEXT", "JSON", "ECS", "GELF", "LOGSTASH" or "CUSTOM"
	handler LogHandler
	// Set rays.logging.level to "TRACE", "DEBUG", "INFO", "WARN", "ERROR" or "FATAL".
	// Defaults to "INFO"
	level LogLevel
	// Set rays.logging.color to "AUTO", "ON", or "OFF".
	// Defaults to "AUTO"
	color LogColor
	// Add Logging source
	source bool
}

// LogLevel returns the configured logging severity level.
func (this *LoggingProperties) LogLevel() LogLevel {
	return this.level
}

// LogHandler returns the configured log handler type.
func (this *LoggingProperties) LogHandler() LogHandler {
	return this.handler
}

// LogColor returns the configured ANSI color output mode.
func (this *LoggingProperties) LogColor() LogColor {
	return this.color
}

// Color returns the configured ANSI color output mode, mirroring LogColor.
func (this *LoggingProperties) Color() LogColor {
	return this.color
}

// AddSource reports whether caller source file and line information should be added to records.
func (this *LoggingProperties) AddSource() bool {
	return this.source
}

// Format returns the pattern format string configured for custom log handlers.
func (this *LoggingProperties) Format() string {
	return this.format
}

// MapLevels returns the mapping of package paths or logger names to their configured log levels.
func (this *LoggingProperties) MapLevels() map[string]LogLevel {
	return this.loggers
}

// logging wraps a logger.Logger instance to implement the core.Logging interface.
type logging struct {
	*logger.Logger
}

// With returns a new Logging instance with the provided key-value attributes appended.
func (this logging) With(args ...any) Logging {
	this.Logger = this.Logger.With(args...)
	return &this
}

// WithGroup returns a new Logging instance that qualifies subsequent attributes with the specified group name.
func (this logging) WithGroup(name string) Logging {
	this.Logger = this.Logger.WithGroup(name)
	return &this
}
