package logger

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
)

var defaultHandlerfunc = func(output io.Writer, options *slog.HandlerOptions) slog.Handler {
	return NewHandler(output, options)
}

// Handler is an slog.Handler implementation that formats log records
// using a compiled pattern-based layout engine with ANSI color and mode support.
type Handler struct {
	opts    slog.HandlerOptions
	w       io.Writer
	pattern *compiledPattern
	color   *bool
	format  string
	groups  []string
	attrs   []collectedAttr
	mu      sync.Mutex
}

// NewHandler creates a new Handler writing to w with DefaultFormat and the given options.
func NewHandler(w io.Writer, opts *slog.HandlerOptions) *Handler {
	return NewHandlerWithFormat(w, opts, DefaultFormat)
}

// NewHandlerWithFormat creates a new Handler writing to w using the specified pattern format.
// If format is empty, DefaultFormat is used.
func NewHandlerWithFormat(w io.Writer, opts *slog.HandlerOptions, format string) *Handler {
	if format == "" {
		format = DefaultFormat
	}
	h := &Handler{
		w:      w,
		format: format,
	}
	if opts != nil {
		h.opts = *opts
	}
	h.pattern = compilePattern(h.format)
	return h
}

// SetFormat updates the pattern format and recompiles the layout AST.
// If format is empty, DefaultFormat is used.
func (this *Handler) SetFormat(format string) {
	if format == "" {
		format = DefaultFormat
	}
	this.format = format
	this.pattern = compilePattern(format)
}

// GetFormat returns the currently configured pattern format string.
func (this *Handler) GetFormat() string {
	return this.format
}

// SetColor explicitly enables or disables ANSI color output.
func (this *Handler) SetColor(enabled bool) {
	this.color = &enabled
}

// EnableColor automatically enables ANSI colors if the output writer is a terminal.
func (this *Handler) EnableColor() {
	this.SetColor(isTerminal(this.w))
}

// IsColor reports whether ANSI color output is currently enabled.
// If not explicitly set via SetColor, it detects whether the output writer is a terminal.
func (this *Handler) IsColor() bool {
	if this.color != nil {
		return *this.color
	}
	return isTerminal(this.w)
}

func (this *Handler) clone() *Handler {
	return &Handler{
		w:       this.w,
		opts:    this.opts,
		format:  this.format,
		pattern: this.pattern,
		color:   this.color,
		attrs:   slices.Clone(this.attrs),
		groups:  slices.Clone(this.groups),
	}
}

// ReplaceAttr returns the attribute replacement function from HandlerOptions,
// or a default pass-through function if none was provided.
func (this *Handler) ReplaceAttr() func(groups []string, a slog.Attr) slog.Attr {
	if this.opts.ReplaceAttr == nil {
		return func(_ []string, a slog.Attr) slog.Attr { return a }
	}
	return this.opts.ReplaceAttr
}

// Enabled reports whether the handler emits log records at the specified level.
func (this *Handler) Enabled(_ context.Context, level slog.Level) bool {
	minLevel := slog.LevelInfo
	if this.opts.Level != nil {
		minLevel = this.opts.Level.Level()
	}
	return level >= minLevel
}

// Handle formats and writes the slog.Record to the destination writer according to the pattern AST.
//
//nolint:gocritic,errcheck // Have to respect interface, ignore write errors
func (this *Handler) Handle(_ context.Context, r slog.Record) error {
	buf := NewBuffer()
	defer buf.Free()

	allAttrs := make([]collectedAttr, 0, len(this.attrs)+r.NumAttrs())
	allAttrs = append(allAttrs, this.attrs...)

	rep := this.ReplaceAttr()
	groupPrefix := strings.Join(this.groups, ".")
	r.Attrs(func(attr slog.Attr) bool {
		collectAttrRecursive(&allAttrs, attr, groupPrefix, this.groups, rep)
		return true
	})

	ctx := renderContext{
		record:       r,
		rep:          rep,
		addSource:    this.opts.AddSource,
		isColor:      this.IsColor(),
		hasGrpInTree: this.pattern.hasGrpInTree,
	}

	if this.pattern.hasGrpInTree {
		for _, a := range allAttrs {
			if a.group == "" {
				ctx.topAttrs = append(ctx.topAttrs, a)
			}
		}
	} else {
		ctx.topAttrs = allAttrs
	}

	groupMap := make(map[string]int)
	for _, a := range allAttrs {
		if a.group != "" {
			if idx, found := groupMap[a.group]; found {
				ctx.groups[idx].attrs = append(ctx.groups[idx].attrs, a.attr)
			} else {
				groupMap[a.group] = len(ctx.groups)
				ctx.groups = append(ctx.groups, groupInfo{
					name:  a.group,
					attrs: []slog.Attr{a.attr},
				})
			}
		}
	}

	this.pattern.root.Render(&ctx, buf)

	if buf.Len() == 0 || (*buf)[buf.Len()-1] != '\n' {
		buf.WriteByte('\n')
	}

	this.mu.Lock()
	defer this.mu.Unlock()

	_, err := this.w.Write(*buf)
	//nolint:wrapcheck // untestable
	return err
}

// WithAttrs returns a new Handler with the given attributes appended to the handler's pre-formatted attributes.
func (this *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return this
	}
	h2 := this.clone()
	rep := this.ReplaceAttr()
	groupPrefix := strings.Join(this.groups, ".")
	for _, attr := range attrs {
		collectAttrRecursive(&h2.attrs, attr, groupPrefix, this.groups, rep)
	}
	return h2
}

// WithGroup returns a new Handler with the given group name appended to the current group hierarchy.
func (this *Handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return this
	}
	h2 := this.clone()
	h2.groups = append(h2.groups, name)
	return h2
}

func collectAttrRecursive(all *[]collectedAttr, attr slog.Attr, groupPrefix string, groups []string, rep func([]string, slog.Attr) slog.Attr) {
	attr.Value = attr.Value.Resolve()
	if rep != nil && attr.Value.Kind() != slog.KindGroup {
		attr = rep(groups, attr)
		attr.Value = attr.Value.Resolve()
	}

	if attr.Equal(slog.Attr{}) || attr.Key == "" {
		return
	}

	if attr.Value.Kind() == slog.KindGroup {
		newPrefix := attr.Key
		if groupPrefix != "" {
			newPrefix = groupPrefix + "." + attr.Key
		}
		newGroups := append(slices.Clone(groups), attr.Key)
		for _, ga := range attr.Value.Group() {
			collectAttrRecursive(all, ga, newPrefix, newGroups, rep)
		}
		return
	}

	*all = append(*all, collectedAttr{
		attr:  attr,
		group: groupPrefix,
	})
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
