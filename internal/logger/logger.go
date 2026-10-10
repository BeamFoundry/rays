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

func GetLogger() *Logger {
	return logger
}

type Logger struct {
	output io.Writer
	*slog.Logger
	options     *slog.HandlerOptions
	handlerFunc func(io.Writer, *slog.HandlerOptions) slog.Handler
	// logLevel    slog.Level
}

func (this Logger) New() *Logger {
	this.output = os.Stderr
	this.handlerFunc = defaultHandlerfunc
	this.options = defaultHandlerOptions()
	handler := this.handlerFunc(this.output, handlerOptions)
	this.Logger = slog.New(handler)
	return &this
}

func (this *Logger) Clone() *Logger {
	logger := &Logger{
		output:      this.output,
		handlerFunc: this.handlerFunc,
		options:     this.cloneOptions(),
	}
	handler := logger.handlerFunc(logger.output, logger.options)
	logger.Logger = slog.New(handler)
	return logger
}

func (this *Logger) getFramePointer(skip int) uintptr {
	pc := make([]uintptr, 32)
	runtime.Callers(4+skip, pc)
	return pc[0]
}

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

func (this *Logger) AddSource() {
	this.options.AddSource = true
	handler := this.handlerFunc(this.output, this.cloneOptions())
	this.Logger = slog.New(handler)
}

func (this *Logger) SetHandler(handlerFunc func(io.Writer, *slog.HandlerOptions) slog.Handler) {
	this.handlerFunc = handlerFunc
	handler := handlerFunc(this.output, this.cloneOptions())
	this.Logger = slog.New(handler)
}

func (this *Logger) SetLogLevel(level slog.Level) {
	this.options.Level = level
	handler := this.handlerFunc(this.output, this.cloneOptions())
	this.Logger = slog.New(handler)
}

func (this *Logger) SetOutput(output io.Writer) {
	this.output = output
	handler := this.handlerFunc(this.output, this.cloneOptions())
	this.Logger = slog.New(handler)
}

func (this *Logger) Reset() {
	this.handlerFunc = defaultHandlerfunc
	handlerOptions = defaultHandlerOptions()
	handler := this.handlerFunc(this.output, this.cloneOptions())
	this.Logger = slog.New(handler)
}

func (this *Logger) Trace(msg string, args ...any) {
	this.log(context.Background(), LevelTrace, msg, args...)
}

func (this *Logger) TraceContext(ctx context.Context, msg string, args ...any) {
	this.log(ctx, LevelTrace, msg, args...)
}

func (this *Logger) Printf(format string, v ...any) {
	this.log(context.Background(), LevelInfo, fmt.Sprintf(format, v...))
}

func (this Logger) With(args ...any) *Logger {
	this.Logger = this.Logger.With(args...)
	return &this
}

func (this Logger) WithGroup(name string) *Logger {
	this.Logger = this.Logger.WithGroup(name)
	return &this
}

func (this *Logger) GetHandlerOptions() *slog.HandlerOptions {
	return this.options
}

func (this *Logger) cloneOptions() *slog.HandlerOptions {
	opts := *this.options
	return &opts
}
