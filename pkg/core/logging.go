package core

import (
	"context"
	"log/slog"
)

const (
	LevelTrace = slog.Level(-8)
	LevelDebug = slog.LevelDebug
	LevelInfo  = slog.LevelInfo
	LevelWarn  = slog.LevelWarn
	LevelError = slog.LevelError
)

// Logging is an interface detailing the majority of function provided by [slog.Logging], adding [LevelTrace] support.
//
// The default implementation is hidden inside the container internal package, as it is initialized directly by the container early on.
//
// It is directly injectable :
//
//	import (
//	    . "github.com/BeamFoundry/rays/pkg/core"
//	    . "github.com/BeamFoundry/rays/pkg/lang"
//	)
//
//	type PaymentService struct {
//	    Service
//
//	    // Note: No pointer '*' because Logging is an interface
//	    logger Logging `@:"Inject"`
//	}
//
//	func (s *PaymentService) Process(amount float64) Error {
//	    s.logger.Trace("Entering Process method", "amount", amount) // Extremely granular logging
//
//	    // Logging supports structured key-value pairs natively
//	    s.logger.Info("Processing payment", "amount", amount)
//
//	    return nil
//	}
//
// It can be configured directly using the container configuration system:
//
//	rays:
//	  logging:
//	    # The default minimum log level
//	    #
//	    # Valid values are "TRACE", "DEBUG", "INFO", "WARN", "ERROR" in that order.
//	    # A value of "INFO" for example will hide all logging from "TRACE" and "DEBUG"
//	    #
//	    # Type: text
//	    # Default: INFO
//	    level: DEBUG
//
//	    # The slog handler to use.
//	    #
//	    # Valid values are "DEFAULT", "TEXT", or "JSON".
//	    #
//	    # Type: text
//	    # Default: DEFAULT
//	    handler: DEFAULT
//
//	    # Add source location to each log entry.
//	    #
//	    # Type: boolean
//	    # Default: true
//	    source: true
//
//	    # Override the log level for specific packages.
//	    #
//	    # Before injecting a [Logging], a new one will be created and its default level will be changed
//	    # if the struct requesting the [Logging] has its package name referenced here.
//	    #
//	    # Type: Dictionary of "source: level"
//	    #
//	    # "source" is either a package name, the one of the struct injecting the [Logging] interface, or
//	    # the package name followed by a slash and the struct name.
//	    #
//	    # It should be noted that "level" will only apply to that "source"
//	    # and will not apply to its sub packages UNLESS it ends with a slash.
//	    #
//	    loggers:
//	      github.com/BeamFoundry/rays-sample/: INFO
//	      github.com/BeamFoundry/rays-sample/pkg/config: DEBUG
//	      github.com/BeamFoundry/rays-sample/pkg/config/DatabaseConfig: WARN
type Logging interface {
	// Debug() logs at [LevelDebug].
	Debug(msg string, args ...any)
	// DebugContext() logs at [LevelDebug] with the given context.
	DebugContext(ctx context.Context, msg string, args ...any)
	// Error() logs at [LevelError].
	Error(msg string, args ...any)
	// ErrorContext() logs at [LevelError] with the given context.
	ErrorContext(ctx context.Context, msg string, args ...any)
	// Handler() returns the slog.Handler.
	Handler() slog.Handler
	// Info() logs at [LevelInfo].
	Info(msg string, args ...any)
	// InfoContext() logs at [LevelInfo] with the given context.
	InfoContext(ctx context.Context, msg string, args ...any)
	// Log() emits a log record with the current time and the given level and message.
	// The [slog.Record]'s [slog.Attrs] consist of the [Logging]'s attributes followed by the [slog.Attrs] specified by args.
	//
	// The attribute arguments are processed as follows:
	//
	//  - If an argument is an [slog.Attr], it is used as is.
	//  - If an argument is a string and this is not the last argument, the following argument is treated as the value and the two are combined into an [slog.Attr].
	//  - Otherwise, the argument is treated as a value with key "!BADKEY".
	Log(ctx context.Context, level slog.Level, msg string, args ...any)
	// LogAttrs() is a more efficient version of [Logging.Log] that accepts only [slog.Attrs].
	LogAttrs(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr)
	// Trace() logs at [LevelTrace].
	Trace(msg string, args ...any)
	// TraceContext() logs at [LevelTrace] with the given context.
	TraceContext(ctx context.Context, msg string, args ...any)
	// Warn() logs at [LevelWarn].
	Warn(msg string, args ...any)
	// WarnContext() logs at [LevelWarn] with the given context.
	WarnContext(ctx context.Context, msg string, args ...any)
	// With() returns a [Logging] that includes the given attributes in each output operation.
	// Arguments are converted to attributes as if by [Logging.Log].
	With(args ...any) Logging
	// WithGroup() returns a [Logging] that starts a group, if name is non-empty.
	// The keys of all attributes added to the [Logging] will be qualified by the given name.
	// (How that qualification happens depends on the Handler.WithGroup method of the [Logging]'s Handler.)
	//
	// If name is empty, WithGroup returns the receiver.
	WithGroup(name string) Logging
}
