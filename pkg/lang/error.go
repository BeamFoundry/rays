package lang

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"path/filepath"
	"reflect"
	"runtime"
	"time"

	"github.com/BeamFoundry/rays/internal/logger"
)

// Error is an interface representing an advanced error type, as it implements the [builtin.error] interface.
//
// An [Error] has a name (like "IllegalArgumentError") and a message (ie. "You gave a wrong value of 'x' for the argument y.")
//
// It can hold some attributes and be logged via the container logging engine.
//
// It holds history of where it was created (frames), and produce a stacktrace, to be logged with the [Error] name and message.
//
// There is default implementations like [DefaultError] and [WrappedError], but you should use
// the functions [NewError], [NewErrorFrom] or [NewWrappedError] to create one.
//
// There is no need to have your own implementation, but it's open.
type Error interface {
	// Name() returns the error name.
	Name() string

	// Message() returns the error message.
	Message() string

	// Error() implements the error interface
	Error() string

	// AddAttribute() add attributes to the error to ge logged
	AddAttribute(key string, value any)

	// AddAttribute() add attributes to the error to ge logged
	AddAttributes(attr ...any)

	// Log() will log the error at the specified level, with optional arguments
	Log(level slog.Level, args ...any)

	// LogWithStackTrace() will log error with a stacktrace to its creation, with optional arguments.
	LogWithStackTrace(level slog.Level, args ...any)

	// LogError() will log the error at Error level, with optional arguments.
	LogError(args ...any)

	// LogErrorWithStackTrace() will log the error at Error level, wit hstacktrace and optional arguments
	LogErrorWithStackTrace(args ...any)

	// PrintStacktrace() will write the stacktrace on an [io.Writer]
	PrintStacktrace(out io.Writer)

	// Format() implements the [fmt.Formatter] interface.
	Format(state fmt.State, verb rune)

	// GomegaString() returns the error for Gomega
	GomegaString() string

	// MarshalJSON() will return a JSON version of this error.
	MarshalJSON() ([]byte, error)
}

// NilError represents an [Error] that will not log anything.
type NilError struct{}

func (this *NilError) Message() string                          { return "Nil" }
func (this *NilError) Name() string                             { return "NilError" }
func (this *NilError) Error() string                            { return "" }
func (this *NilError) AddAttribute(_ string, _ any)             {}
func (this *NilError) AddAttributes(_ ...any)                   {}
func (this *NilError) GomegaString() string                     { return "" }
func (this *NilError) Log(_ slog.Level, _ ...any)               {}
func (this *NilError) LogError(_ ...any)                        {}
func (this *NilError) LogErrorWithStackTrace(_ ...any)          {}
func (this *NilError) LogWithStackTrace(_ slog.Level, _ ...any) {}
func (this *NilError) PrintStacktrace(_ io.Writer)              {}
func (this *NilError) Format(_ fmt.State, _ rune)               {}
func (this *NilError) MarshalJSON() ([]byte, error)             { return []byte{}, nil }

var _ Error = &NilError{}

// DefaultError is the default implementation of the [Error] interface.
type DefaultError struct {
	name       string
	message    string
	frames     []runtime.Frame
	attributes []any
	pc         uintptr
}

func newError(name, message string, args ...any) *DefaultError {
	pcs := make([]uintptr, 32)
	n := runtime.Callers(3, pcs)
	maxPass := math.Min(float64(n), 100.0)
	pcs = pcs[:int(maxPass)] // pass only valid pcs to runtime.CallersFrames
	pc := pcs[0]
	fms := runtime.CallersFrames(pcs)
	var frames []runtime.Frame
	for {
		frame, _ := fms.Next()
		if reflect.ValueOf(frame).Equal(reflect.ValueOf(runtime.Frame{})) {
			break
		}
		frames = append(frames, frame)
	}
	return &DefaultError{name, message, frames, args, pc}
}

// NewError() creates a new Error with a name, message and optional key-paired arguments.
func NewError(name, message string, args ...any) *DefaultError {
	return newError(name, message, args...)
}

// Name() returns the error name.
func (this *DefaultError) Name() string {
	return this.name
}

// Message() returns the error message.
func (this *DefaultError) Message() string {
	return this.message
}

// Error() implements the error interface
func (this *DefaultError) Error() string {
	return fmt.Sprintf("%s: %s", this.name, this.message)
}

// AddAttribute() add attributes to the error to ge logged
func (this *DefaultError) AddAttribute(key string, value any) {
	this.attributes = append(this.attributes, key, value)
}

// AddAttribute() add attributes to the error to ge logged
func (this *DefaultError) AddAttributes(attr ...any) {
	this.attributes = append(this.attributes, attr...)
}

// GomegaString() returns the error for Gomega
func (this *DefaultError) GomegaString() string {
	out := bytes.NewBufferString("")
	this.PrintStacktrace(out)
	return out.String()
}

// PrintStacktrace() will write the stacktrace on an [io.Writer]
func (this *DefaultError) PrintStacktrace(out io.Writer) {
	for _, frame := range this.frames {
		fmt.Fprintf(out, "\n at %s(%s:%d)", frame.Function, filepath.Base(frame.File), frame.Line)
	}
}

// Formatter interface Format function
// %s or %v will print the error message
// Use %+v to also print the Stacktrace
func (this *DefaultError) Format(state fmt.State, verb rune) {
	switch verb {
	case 'v':
		if state.Flag('+') {
			// TODO: Review why we do not print the error message first
			this.PrintStacktrace(state)
		} else {
			fmt.Fprintf(state, "%s", this.Error())
		}
	default:
		fmt.Fprintf(state, "%s", this.Error())
	}
}

// Log() will log the error at the specified level, with optional arguments
func (this *DefaultError) Log(level slog.Level, args ...any) {
	if logger.GetLogger().Enabled(context.Background(), logger.LevelError) {
		this.logFrames(level, []runtime.Frame{}, args...)
	}
}

// LogWithStackTrace() will log error with a stacktrace to its creation, with optional arguments.
func (this *DefaultError) LogWithStackTrace(level slog.Level, args ...any) {
	if logger.GetLogger().Enabled(context.Background(), logger.LevelError) {
		this.logFrames(level, this.frames, args...)
	}
}

// LogError() will log the error at Error level, with optional arguments.
func (this *DefaultError) LogError(args ...any) {
	// TODO: Review why we use logFrames
	if logger.GetLogger().Enabled(context.Background(), logger.LevelError) {
		this.logFrames(slog.LevelError, []runtime.Frame{}, args...)
	}
}

// LogErrorWithStackTrace() will log the error at Error level, wit hstacktrace and optional arguments
func (this *DefaultError) LogErrorWithStackTrace(args ...any) {
	if logger.GetLogger().Enabled(context.Background(), logger.LevelError) {
		this.logFrames(slog.LevelError, this.frames, args...)
	}
}

// MarshalJSON() will return a JSON version of this error.
func (this *DefaultError) MarshalJSON() ([]byte, error) {
	st := struct {
		Name    string `json:"name"`
		Message string `json:"message"`
	}{this.name, this.message}
	//nolint:wrapcheck // untestable
	return json.Marshal(st)
}

func (this *DefaultError) logFrames(level slog.Level, frames []runtime.Frame, args ...any) {
	var out bytes.Buffer
	fmt.Fprintf(&out, "%s", this.Error())
	for _, frame := range frames {
		fmt.Fprintf(&out, "\n at %s(%s:%d)", frame.Function, filepath.Base(frame.File), frame.Line)
	}

	record := slog.Record{
		Time:    time.Now(),
		Message: out.String(),
		Level:   level,
		PC:      this.pc,
	}
	record.Add(this.attributes...)
	record.Add(args...)
	//nolint:errcheck // ignoring
	logger.GetLogger().Handler().Handle(context.Background(), record)
}

//---

// WrappedError is a structure permitting to wrap an error as an Error.
type WrappedError struct {
	*DefaultError
	wrapped error
}

// Unwrap() returns the wrapped error.
func (this *WrappedError) Unwrap() error {
	return this.wrapped
}

// NewWrappedError returns an Error with a name from and error.
func NewWrappedError(name string, err error, args ...any) Error {
	if err == nil {
		return nil
	}
	return &WrappedError{newError(name, err.Error(), args...), err}
}

// NewErrorFrom() creates a new [WrappedError] from an existing error.
//
// The error name will be "WrappedError", and the error Error() as its message.
func NewErrorFrom(err error) Error {
	if err == nil {
		return nil
	}
	return &WrappedError{newError("WrappedError", err.Error()), err}
}

// IsNilError will return if an [Error] is nil or not.
func IsNilError[T Error](err T) bool {
	if !reflect.ValueOf(err).IsValid() || reflect.ValueOf(err).IsNil() {
		return true
	}
	//nolint:gocritic // Keeping commentedOutCode
	// return reflect.TypeOf(err) == reflect.TypeFor[*NilError]()
	return false
}

// PanicError represents a named Error for PanicError
type PanicError struct {
	*DefaultError
}

// NewPanicError() creates a new PanicError with a message
func NewPanicError(message string) *PanicError {
	return &PanicError{newError("PanicError", message)}
}

// IndexOutOfBoundsError represents a named Error for IndexOutOfBoundsError
type IndexOutOfBoundsError struct {
	*DefaultError
}

// NewIndexOutOfBoundsError() creates a new IndexOutOfBoundsError with a message
func NewIndexOutOfBoundsError(message string) *IndexOutOfBoundsError {
	return &IndexOutOfBoundsError{NewError("IndexOutOfBoundsError", message)}
}

// PatternSyntaxError represents a named Error for PatternSyntaxError
type PatternSyntaxError struct {
	*DefaultError
}

// NewPatternSyntaxError() creates a new PatternSyntaxError with a message
func NewPatternSyntaxError(message string) *PatternSyntaxError {
	return &PatternSyntaxError{NewError("PatternSyntaxError", message)}
}
