package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime"
	"time"
)

var logger = Logger{}.New()

// GetLogger returns the global default Logger instance.
func GetLogger() *Logger {
	return logger
}

// Logger wraps slog.Logger with support for custom log levels (Trace, Fatal),
// dynamic format patterns, ANSI color modes, and handler configuration.
type Logger struct {
	*slog.Logger
	options     *slog.HandlerOptions
	handlerFunc func(io.Writer, *slog.HandlerOptions) slog.Handler
	color       *bool
	output      io.Writer
	format      string
}

// New initializes and returns a new Logger writing to os.Stderr with default settings.
func (this Logger) New() *Logger {
	this.output = os.Stderr
	this.handlerFunc = defaultHandlerfunc
	this.options = defaultHandlerOptions()
	handler := this.handlerFunc(this.output, handlerOptions)
	this.Logger = slog.New(handler)
	return &this
}

// Clone creates a duplicate copy of the Logger with independent configuration.
func (this *Logger) Clone() *Logger {
	logger := &Logger{
		output:      this.output,
		handlerFunc: this.handlerFunc,
		options:     this.cloneOptions(),
		format:      this.format,
		color:       this.color,
	}
	handler := logger.handlerFunc(logger.output, logger.options)
	handler = logger.applyHandlerSettings(handler)
	logger.Logger = slog.New(handler)
	return logger
}

func (this *Logger) applyHandlerSettings(handler slog.Handler) slog.Handler {
	if this.format != "" {
		if h, ok := handler.(interface{ SetFormat(string) }); ok {
			h.SetFormat(this.format)
		}
	}
	if this.color != nil {
		if h, ok := handler.(interface{ SetColor(bool) }); ok {
			h.SetColor(*this.color)
		}
	} else {
		if h, ok := handler.(interface{ EnableColor() }); ok {
			h.EnableColor()
		}
	}
	return handler
}

func (this *Logger) getFramePointer(skip int) uintptr {
	pc := make([]uintptr, 32)
	runtime.Callers(4+skip, pc)
	return pc[0]
}

// SetDefault sets this Logger as the standard library's default slog logger.
func (this *Logger) SetDefault() {
	slog.SetDefault(this.Logger)
}

func (this *Logger) log(ctx context.Context, level slog.Level, msg string, args ...any) {
	if this.Enabled(ctx, level) {
		record := slog.Record{
			Time:    time.Now(),
			Message: msg,
			Level:   level,
			PC:      this.getFramePointer(0),
		}
		record.Add(args...)
		//nolint:errcheck // Not tested
		this.Handler().Handle(ctx, record)
	}
}

// AddSource enables source file and line number reporting in log outputs.
func (this *Logger) AddSource() {
	this.options.AddSource = true
	handler := this.handlerFunc(this.output, this.cloneOptions())
	handler = this.applyHandlerSettings(handler)
	this.Logger = slog.New(handler)
}

// SetHandler configures a new handler factory function and rebuilds the underlying slog.Logger.
func (this *Logger) SetHandler(handlerFunc func(io.Writer, *slog.HandlerOptions) slog.Handler) {
	this.handlerFunc = handlerFunc
	handler := handlerFunc(this.output, this.cloneOptions())
	handler = this.applyHandlerSettings(handler)
	this.Logger = slog.New(handler)
}

// SetLogLevel sets the minimum logging level required to emit records.
func (this *Logger) SetLogLevel(level slog.Level) {
	this.options.Level = level
	handler := this.handlerFunc(this.output, this.cloneOptions())
	handler = this.applyHandlerSettings(handler)
	this.Logger = slog.New(handler)
}

// SetOutput redirects logging output to the specified io.Writer.
func (this *Logger) SetOutput(output io.Writer) {
	this.output = output
	handler := this.handlerFunc(this.output, this.cloneOptions())
	handler = this.applyHandlerSettings(handler)
	this.Logger = slog.New(handler)
}

// SetFormat configures the pattern layout format on the underlying handler if supported.
func (this *Logger) SetFormat(format string) {
	this.format = format
	if h, ok := this.Handler().(interface{ SetFormat(string) }); ok {
		h.SetFormat(format)
	}
}

// GetFormat returns the currently configured pattern layout format string.
func (this *Logger) GetFormat() string {
	if h, ok := this.Handler().(interface{ GetFormat() string }); ok {
		return h.GetFormat()
	}
	return this.format
}

// SetColor explicitly forces ANSI color output on or off.
func (this *Logger) SetColor(enabled bool) {
	this.color = &enabled
	if h, ok := this.Handler().(interface{ SetColor(bool) }); ok {
		h.SetColor(enabled)
	}
}

// EnableColor sets ANSI color output to automatic detection based on terminal capability.
func (this *Logger) EnableColor() {
	this.color = nil
	switch h := this.Handler().(type) {
	case interface{ EnableColor() }:
		h.EnableColor()
	case interface{ SetColor(bool) }:
		h.SetColor(isTerminal(this.output))
	default:
	}
}

// IsColor reports whether ANSI color output is enabled.
func (this *Logger) IsColor() bool {
	if h, ok := this.Handler().(interface{ IsColor() bool }); ok {
		return h.IsColor()
	}
	if this.color != nil {
		return *this.color
	}
	return isTerminal(this.output)
}

// Reset restores the logger to its default handler, format, color mode, and options.
func (this *Logger) Reset() {
	this.format = ""
	this.color = nil
	this.handlerFunc = defaultHandlerfunc
	handlerOptions = defaultHandlerOptions()
	handler := this.handlerFunc(this.output, this.cloneOptions())
	this.Logger = slog.New(handler)
}

// Fatal logs a message at LevelFatal using context.Background().
func (this *Logger) Fatal(msg string, args ...any) {
	this.log(context.Background(), LevelFatal, msg, args...)
}

// FatalContext logs a message at LevelFatal with the provided context.
func (this *Logger) FatalContext(ctx context.Context, msg string, args ...any) {
	this.log(ctx, LevelFatal, msg, args...)
}

// Trace logs a message at LevelTrace using context.Background().
func (this *Logger) Trace(msg string, args ...any) {
	this.log(context.Background(), LevelTrace, msg, args...)
}

// TraceContext logs a message at LevelTrace with the provided context.
func (this *Logger) TraceContext(ctx context.Context, msg string, args ...any) {
	this.log(ctx, LevelTrace, msg, args...)
}

// Printf formats and logs an informational message at LevelInfo.
func (this *Logger) Printf(format string, v ...any) {
	this.log(context.Background(), LevelInfo, fmt.Sprintf(format, v...))
}

// With returns a new Logger that includes the given attributes in each subsequent log output.
func (this Logger) With(args ...any) *Logger {
	this.Logger = this.Logger.With(args...)
	return &this
}

// WithGroup returns a new Logger that starts a group with the given name.
func (this Logger) WithGroup(name string) *Logger {
	this.Logger = this.Logger.WithGroup(name)
	return &this
}

// GetHandlerOptions returns the underlying slog.HandlerOptions.
func (this *Logger) GetHandlerOptions() *slog.HandlerOptions {
	return this.options
}

func (this *Logger) cloneOptions() *slog.HandlerOptions {
	opts := *this.options
	return &opts
}
