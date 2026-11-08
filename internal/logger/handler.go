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
	mu      *sync.Mutex
	pattern *compiledPattern
	color   *bool
	format  string
	groups  []string
	attrs   []collectedAttr
	cfgMu   sync.RWMutex
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
		mu:     &sync.Mutex{},
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
	pat := compilePattern(format)
	this.cfgMu.Lock()
	defer this.cfgMu.Unlock()
	this.format = format
	this.pattern = pat
}

// GetFormat returns the currently configured pattern format string.
func (this *Handler) GetFormat() string {
	this.cfgMu.RLock()
	defer this.cfgMu.RUnlock()
	return this.format
}

// SetColor explicitly enables or disables ANSI color output.
func (this *Handler) SetColor(enabled bool) {
	this.cfgMu.Lock()
	defer this.cfgMu.Unlock()
	this.color = &enabled
}

// EnableColor automatically enables ANSI colors if the output writer is a terminal.
func (this *Handler) EnableColor() {
	enabled := isTerminal(this.w)
	this.cfgMu.Lock()
	defer this.cfgMu.Unlock()
	this.color = &enabled
}

// IsColor reports whether ANSI color output is currently enabled.
// If not explicitly set via SetColor, it detects whether the output writer is a terminal.
func (this *Handler) IsColor() bool {
	this.cfgMu.RLock()
	defer this.cfgMu.RUnlock()
	if this.color != nil {
		return *this.color
	}
	return isTerminal(this.w)
}

// clone returns a copy of the Handler sharing the original writer mutex.
func (this *Handler) clone() *Handler {
	this.cfgMu.RLock()
	defer this.cfgMu.RUnlock()
	return &Handler{
		w:       this.w,
		opts:    this.opts,
		format:  this.format,
		pattern: this.pattern,
		color:   this.color,
		attrs:   slices.Clone(this.attrs),
		groups:  slices.Clone(this.groups),
		mu:      this.mu,
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
	r.Attrs(func(attr slog.Attr) bool {
		collectAttrRecursive(&allAttrs, attr, this.groups, rep)
		return true
	})

	this.cfgMu.RLock()
	pattern := this.pattern
	format := this.format
	isColor := false
	if this.color != nil {
		isColor = *this.color
	} else {
		isColor = isTerminal(this.w)
	}
	this.cfgMu.RUnlock()

	isJson := strings.HasPrefix(strings.TrimSpace(format), "{")
	isGELF := strings.Contains(format, `"version":"1.1"`)

	ctx := renderContext{
		record:       r,
		rep:          rep,
		addSource:    this.opts.AddSource,
		isColor:      isColor,
		hasGrpInTree: pattern.hasGrpInTree,
		isJson:       isJson,
		isGELF:       isGELF,
		format:       format,
	}

	if pattern.hasGrpInTree {
		for _, a := range allAttrs {
			if len(a.groups) == 0 {
				ctx.topAttrs = append(ctx.topAttrs, a)
			}
		}
	} else {
		ctx.topAttrs = allAttrs
	}

	mode := groupModeText
	if isGELF {
		mode = groupModeGELF
	} else if isJson {
		mode = groupModeJSON
	}
	ctx.groups = buildGroupInfos(allAttrs, mode)

	pattern.root.Render(&ctx, buf)

	if isJson {
		cleanJsonCommas(buf)
	}

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
	rep := h2.ReplaceAttr()
	for _, attr := range attrs {
		collectAttrRecursive(&h2.attrs, attr, h2.groups, rep)
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

// collectAttrRecursive flattens nested slog attributes and groups into a slice of collectedAttr.
func collectAttrRecursive(all *[]collectedAttr, attr slog.Attr, groups []string, rep func([]string, slog.Attr) slog.Attr) {
	attr.Value = attr.Value.Resolve()
	if rep != nil && attr.Value.Kind() != slog.KindGroup {
		attr = rep(groups, attr)
		attr.Value = attr.Value.Resolve()
	}

	if attr.Equal(slog.Attr{}) {
		return
	}

	if attr.Value.Kind() == slog.KindGroup {
		if attr.Key == "" {
			for _, ga := range attr.Value.Group() {
				collectAttrRecursive(all, ga, groups, rep)
			}
			return
		}
		newGroups := append(slices.Clone(groups), attr.Key)
		for _, ga := range attr.Value.Group() {
			collectAttrRecursive(all, ga, newGroups, rep)
		}
		return
	}

	*all = append(*all, collectedAttr{
		attr:   attr,
		groups: slices.Clone(groups),
	})
}

// groupFormatMode defines how attribute groups are structured and formatted in output.
type groupFormatMode int

const (
	groupModeText groupFormatMode = iota
	groupModeJSON
	groupModeGELF
)

// buildGroupInfos organizes collected attributes and group hierarchy into groupInfo structures according to mode.
func buildGroupInfos(allAttrs []collectedAttr, mode groupFormatMode) []groupInfo {
	if mode == groupModeGELF {
		groupMap := make(map[string]int)
		var groups []groupInfo
		for _, a := range allAttrs {
			if len(a.groups) > 0 {
				key := strings.Join(a.groups, ".")
				if idx, found := groupMap[key]; found {
					groups[idx].attrs = append(groups[idx].attrs, a.attr)
				} else {
					groupMap[key] = len(groups)
					groups = append(groups, groupInfo{
						name:  key,
						attrs: []slog.Attr{a.attr},
					})
				}
			}
		}
		return groups
	}
	if mode == groupModeJSON {
		return buildNestedGroupTree(allAttrs)
	}
	groupMap := make(map[string]int)
	var groups []groupInfo
	for _, a := range allAttrs {
		if len(a.groups) > 0 {
			key := strings.Join(a.groups, ".")
			if idx, found := groupMap[key]; found {
				groups[idx].attrs = append(groups[idx].attrs, a.attr)
			} else {
				groupMap[key] = len(groups)
				groups = append(groups, groupInfo{
					name:  key,
					attrs: []slog.Attr{a.attr},
				})
			}
		}
	}
	return groups
}

// buildNestedGroupTree constructs a hierarchical tree of nested groups for JSON-formatted output.
func buildNestedGroupTree(allAttrs []collectedAttr) []groupInfo {
	type gNode struct {
		name     string
		attrs    []slog.Attr
		children []*gNode
	}
	var roots []*gNode

	findOrCreate := func(nodes *[]*gNode, name string) *gNode {
		for _, n := range *nodes {
			if n.name == name {
				return n
			}
		}
		n := &gNode{name: name}
		*nodes = append(*nodes, n)
		return n
	}

	for _, a := range allAttrs {
		if len(a.groups) == 0 {
			continue
		}
		currList := &roots
		var currNode *gNode
		for _, seg := range a.groups {
			currNode = findOrCreate(currList, seg)
			currList = &currNode.children
		}
		currNode.attrs = append(currNode.attrs, a.attr)
	}

	var convert func(n *gNode) groupInfo
	convert = func(n *gNode) groupInfo {
		gi := groupInfo{
			name:  n.name,
			attrs: n.attrs,
		}
		for _, ch := range n.children {
			gi.children = append(gi.children, convert(ch))
		}
		return gi
	}

	var res []groupInfo
	for _, r := range roots {
		res = append(res, convert(r))
	}
	return res
}

// cleanJsonCommas removes doubled or trailing commas from rendered JSON buffers after omitted fields.
func cleanJsonCommas(buf *Buffer) {
	b := *buf
	if len(b) < 2 || b[0] != '{' {
		return
	}
	out := make([]byte, 0, len(b))
	inString := false
	var prevNonSpace byte

	for i := 0; i < len(b); i++ {
		c := b[i]
		if inString {
			out = append(out, c)
			if c == '\\' && i+1 < len(b) {
				i++
				out = append(out, b[i])
				continue
			}
			if c == '"' {
				inString = false
				prevNonSpace = '"'
			}
			continue
		}

		if c == '"' {
			inString = true
			out = append(out, c)
			prevNonSpace = '"'
			continue
		}

		if c == ' ' || c == '\t' || c == '\r' || c == '\n' {
			out = append(out, c)
			continue
		}

		if c == ',' {
			if prevNonSpace == '{' || prevNonSpace == '[' || prevNonSpace == ',' {
				continue
			}
			j := i + 1
			for j < len(b) && (b[j] == ' ' || b[j] == '\t' || b[j] == '\r' || b[j] == '\n') {
				j++
			}
			if j < len(b) && (b[j] == '}' || b[j] == ']') {
				continue
			}
			out = append(out, c)
			prevNonSpace = ','
			continue
		}

		out = append(out, c)
		prevNonSpace = c
	}

	buf.Reset()
	//nolint:errcheck // buffer write cannot fail
	buf.Write(out)
}

// appendString writes a string to the buffer, escaping and quoting it if quote is true.
//
//nolint:errcheck,revive //ignoring buffer write.
func appendString(buf *Buffer, s string, quote bool) {
	if quote {
		appendJsonString(buf, s)
	} else {
		buf.WriteString(s)
	}
}

// appendSource formats and appends caller source code location (file and line) to the buffer.
//
//nolint:errcheck //ignoring buffer write.
func appendSource(buf *Buffer, src *slog.Source) {
	dir, file := filepath.Split(src.File)

	buf.WriteString(filepath.Join(filepath.Base(dir), file))
	buf.WriteByte(':')
	*buf = strconv.AppendInt(*buf, int64(src.Line), 10)
}
