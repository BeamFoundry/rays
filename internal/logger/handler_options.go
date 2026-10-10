package logger

import (
	"fmt"
	"log/slog"
	"path/filepath"
)

const (
	LevelTrace = slog.Level(-8)
	LevelDebug = slog.LevelDebug
	LevelInfo  = slog.LevelInfo
	LevelWarn  = slog.LevelWarn
	LevelError = slog.LevelError
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
			case slog.LevelKey:
				if level, ok := a.Value.Any().(slog.Level); ok {
					switch level {
					case LevelTrace:
						a.Value = slog.StringValue("TRACE")
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
					default:
						a.Value = str("ERROR", level-LevelError)
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
