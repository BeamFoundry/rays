package logger

// https://github.com/lmittmann/tint/blob/main/handler.go
/*
BSD 3-Clause License

Copyright (c) 2023 lmittmann

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

1. Redistributions of source code must retain the above copyright notice, this
   list of conditions and the following disclaimer.

2. Redistributions in binary form must reproduce the above copyright notice,
   this list of conditions and the following disclaimer in the documentation
   and/or other materials provided with the distribution.

3. Neither the name of the copyright holder nor the names of its
   contributors may be used to endorse or promote products derived from
   this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE
FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/

import (
	"context"
	"encoding"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"sync"
	"time"
)

var defaultHandlerfunc = func(output io.Writer, options *slog.HandlerOptions) slog.Handler {
	return NewHandler(output, options)
}

type Handler struct {
	opts        slog.HandlerOptions
	w           io.Writer
	attrsPrefix string
	groupPrefix string
	groups      []string

	mu sync.Mutex
}

func NewHandler(w io.Writer, opts *slog.HandlerOptions) *Handler {
	h := &Handler{w: w}
	if opts != nil {
		h.opts = *opts
	}
	return h
}

func (this *Handler) clone() *Handler {
	return &Handler{
		attrsPrefix: this.attrsPrefix,
		groupPrefix: this.groupPrefix,
		groups:      this.groups,
		w:           this.w,
		opts:        this.opts,
	}
}

func (this *Handler) ReplaceAttr() func(groups []string, a slog.Attr) slog.Attr {
	if this.opts.ReplaceAttr == nil {
		return func(_ []string, a slog.Attr) slog.Attr { return a }
	}
	return this.opts.ReplaceAttr
}

func (this *Handler) Enabled(_ context.Context, level slog.Level) bool {
	minLevel := slog.LevelInfo
	if this.opts.Level != nil {
		minLevel = this.opts.Level.Level()
	}
	return level >= minLevel
}

//nolint:gocritic,errcheck // Have to respect interface, ignore write errors
func (this *Handler) Handle(_ context.Context, r slog.Record) error {
	// get a buffer from the sync pool
	buf := NewBuffer()
	defer buf.Free()

	rep := this.ReplaceAttr()

	// write time
	if !r.Time.IsZero() {
		val := r.Time.Round(0) // strip monotonic to match Attr behavior
		if a := rep(nil /* groups */, slog.Time(slog.TimeKey, val)); a.Key != "" {
			val := a.Value.Resolve()
			if val.Kind() == slog.KindTime {
				this.appendTime(buf, val.Time())
			} else {
				this.appendValue(buf, val, false)
			}
			buf.WriteByte(' ')
		}
	}

	// write level
	if a := rep(nil /* groups */, slog.Any(slog.LevelKey, r.Level)); a.Key != "" {
		val := a.Value.Resolve()
		if val.Kind() == slog.KindAny {
			if lvlVal, ok := val.Any().(slog.Level); ok {
				this.appendLevel(buf, lvlVal)
			} else {
				this.appendValue(buf, val, false)
			}
		} else {
			this.appendValue(buf, val, false)
		}
		buf.WriteByte(' ')
	}

	// write source
	if this.opts.AddSource {
		fs := runtime.CallersFrames([]uintptr{r.PC})
		f, _ := fs.Next()
		if f.File != "" {
			src := &slog.Source{
				Function: f.Function,
				File:     f.File,
				Line:     f.Line,
			}

			if a := rep(nil /* groups */, slog.Any(slog.SourceKey, src)); a.Key != "" {
				val := a.Value.Resolve()
				this.appendValue(buf, val, false)
				buf.WriteByte(' ')
			}
		}
	}

	// write message
	if a := rep(nil /* groups */, slog.String(slog.MessageKey, r.Message)); a.Key != "" {
		val := a.Value.Resolve()
		this.appendValue(buf, val, false)
		buf.WriteByte(' ')
	}

	// write handler attributes
	if len(this.attrsPrefix) > 0 {
		buf.WriteString(this.attrsPrefix)
	}

	// write attributes
	r.Attrs(func(attr slog.Attr) bool {
		this.appendAttr(buf, attr, this.groupPrefix, this.groups)
		return true
	})

	/*
		if len(*buf) == 0 {
			return nil
		}
	*/
	(*buf)[len(*buf)-1] = '\n' // replace last space with newline

	this.mu.Lock()
	defer this.mu.Unlock()

	_, err := this.w.Write(*buf)
	//nolint:wrapcheck // untestable
	return err
}

func (this *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return this
	}
	h2 := this.clone()

	buf := NewBuffer()
	defer buf.Free()

	// write attributes to buffer
	for _, attr := range attrs {
		this.appendAttr(buf, attr, this.groupPrefix, this.groups)
	}
	h2.attrsPrefix = this.attrsPrefix + string(*buf)
	return h2
}

func (this *Handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return this
	}
	h2 := this.clone()
	h2.groupPrefix += name + "."
	h2.groups = append(h2.groups, name)
	return h2
}

//nolint:errcheck //ignoring buffer write errors
func (this *Handler) appendAttr(buf *Buffer, attr slog.Attr, groupsPrefix string, groups []string) {
	attr.Value = attr.Value.Resolve()
	if rep := this.opts.ReplaceAttr; rep != nil && attr.Value.Kind() != slog.KindGroup {
		attr = rep(groups, attr)
		attr.Value = attr.Value.Resolve()
	}

	if attr.Equal(slog.Attr{}) {
		return
	}

	if attr.Value.Kind() == slog.KindGroup {
		if attr.Key != "" {
			groupsPrefix += attr.Key + "."
			groups = append(groups, attr.Key)
		}
		for _, groupAttr := range attr.Value.Group() {
			this.appendAttr(buf, groupAttr, groupsPrefix, groups)
		}
		return
	}

	this.appendKey(buf, attr.Key, groupsPrefix)
	this.appendValue(buf, attr.Value, true)
	buf.WriteByte(' ')
}

//nolint:errcheck //ignoring buffer write.
func (this *Handler) appendKey(buf *Buffer, key, groups string) {
	appendString(buf, groups+key, false)
	buf.WriteByte('=')
}

func (this *Handler) appendTime(buf *Buffer, t time.Time) {
	*buf = t.AppendFormat(*buf, "2006/01/02 15:04:05.000")
}

//nolint:errcheck //ignoring buffer write.
func (this *Handler) appendLevel(buf *Buffer, level slog.Level) {
	buf.WriteString(level.String())
}

func (this *Handler) appendValue(buf *Buffer, v slog.Value, quote bool) {
	switch v.Kind() {
	case slog.KindString:
		appendString(buf, v.String(), quote)
	case slog.KindInt64:
		*buf = strconv.AppendInt(*buf, v.Int64(), 10)
	case slog.KindUint64:
		*buf = strconv.AppendUint(*buf, v.Uint64(), 10)
	case slog.KindFloat64:
		*buf = strconv.AppendFloat(*buf, v.Float64(), 'g', -1, 64)
	case slog.KindBool:
		*buf = strconv.AppendBool(*buf, v.Bool())
	case slog.KindDuration:
		appendString(buf, v.Duration().String(), quote)
	case slog.KindTime:
		appendString(buf, v.Time().String(), quote)
	case slog.KindAny:
		defer func() {
			// Copied from log/slog/handler.go.
			if r := recover(); r != nil {
				// If it panics with a nil pointer, the most likely cases are
				// an encoding.TextMarshaler or error fails to guard against nil,
				// in which case "<nil>" seems to be the feasible choice.
				//
				// Adapted from the code in fmt/print.go.
				if v := reflect.ValueOf(v.Any()); v.Kind() == reflect.Pointer && v.IsNil() {
					appendString(buf, "<nil>", false)
					return
				}

				// Otherwise just print the original panic message.
				appendString(buf, fmt.Sprintf("!PANIC: %v", r), true)
			}
		}()

		switch cv := v.Any().(type) {
		case encoding.TextMarshaler:
			data, err := cv.MarshalText()
			if err != nil {
				break
			}
			appendString(buf, string(data), quote)
		case *slog.Source:
			appendSource(buf, cv)
		default:
			appendString(buf, fmt.Sprintf("%+v", cv), quote)
		}
	default:
	}
}

//nolint:errcheck,revive //ignoring buffer write.
func appendString(buf *Buffer, s string, quote bool) {
	if quote {
		buf.WriteString(strconv.Quote(s))
	} else {
		buf.WriteString(s)
	}
}

//nolint:errcheck //ignoring buffer write.
func appendSource(buf *Buffer, src *slog.Source) {
	dir, file := filepath.Split(src.File)

	buf.WriteString(filepath.Join(filepath.Base(dir), file))
	buf.WriteByte(':')
	*buf = strconv.AppendInt(*buf, int64(src.Line), 10)
}
