package logger

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"time"
)

const (
	// LevelTrace defines the trace log level (-8), more granular than LevelDebug.
	LevelTrace = slog.Level(-8)
	// LevelDebug defines the debug log level (-4).
	LevelDebug = slog.LevelDebug
	// LevelInfo defines the informational log level (0).
	LevelInfo = slog.LevelInfo
	// LevelWarn defines the warning log level (4).
	LevelWarn = slog.LevelWarn
	// LevelError defines the error log level (8).
	LevelError = slog.LevelError
	// LevelFatal defines the fatal log level (12), representing unrecoverable errors.
	LevelFatal = slog.Level(12)
)

var handlerOptions = defaultHandlerOptions()

func defaultHandlerOptions() *slog.HandlerOptions {
	return &slog.HandlerOptions{
		AddSource: true,
		Level:     LevelInfo,
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			str := func(base string, val slog.Level) slog.Value {
				if val == 0 {
					return slog.StringValue(base)
				}
				return slog.StringValue(fmt.Sprintf("%s%+d", base, val))
			}
			switch a.Key {
			case slog.TimeKey:
				if t, ok := a.Value.Any().(time.Time); ok {
					a.Value = slog.StringValue(t.Format("2006/01/02 15:04:05.000"))
				}
			case slog.LevelKey:
				if level, ok := a.Value.Any().(slog.Level); ok {
					switch level {
					case LevelTrace:
						a.Value = slog.StringValue("TRACE")
					case LevelFatal:
						a.Value = slog.StringValue("FATAL")
					default:
						a.Value = slog.StringValue(level.String())
					}
					switch {
					case level < LevelDebug:
						a.Value = str("TRACE", level-LevelTrace)
					case level < LevelInfo:
						a.Value = str("DEBUG", level-LevelDebug)
					case level < LevelWarn:
						a.Value = str("INFO", level-LevelInfo)
					case level < LevelError:
						a.Value = str("WARN", level-LevelWarn)
					case level < LevelFatal:
						a.Value = str("ERROR", level-LevelError)
					default:
						a.Value = str("FATAL", level-LevelFatal)
					}
				}
			case slog.SourceKey:
				if frame, ok := a.Value.Any().(*slog.Source); ok {
					src := fmt.Sprintf("%s(%s:%d)", ReduceFunctionName(frame.Function), filepath.Base(frame.File), frame.Line)
					a.Value = slog.StringValue(src)
				}
			default:
			}
			return a
		},
	}
}

/*
func GetHandlerOptions() *slog.HandlerOptions {
	return handlerOptions
}
*/
