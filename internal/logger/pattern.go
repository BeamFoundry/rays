package logger

import (
	"encoding"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"golang.org/x/term"
)

const (
	// DefaultFormat is the default log pattern format using colored output.
	DefaultFormat = "%time(%v) %clr(%level(%-5v)) %source(%clr(%v){magenta} )%msg(%v)%clr(%attr( %k=%v)){faint}%clr(%grp( %g.%attr(%k=%v%nl( )))){faint}"
	// TextFormat is the standard slog logfmt-style text pattern.
	TextFormat = `%time(%ak=%"2006-01-02T15:04:05.999Z07:00"v )%level(%ak=%av) %source(%ak=%av )%msg(%ak=%qv)%attr( %ak=%qv)%grp(%attr( %.ag=%qv))`
	// JSONFormat is the standard JSON log format.
	JSONFormat = `{%time(%qk:%"RFC3339Nano"qv)%level(,%qk:%qv)%source(,%qk:%"json"v)%msg(,%qk:%qv)%attr(,%qk:%qv)%grp(,%qg:{%attr(%qk:%qv%nl(,))})}`
	// ECSFormat is the Elastic Common Schema (ECS) v1.2.0 JSON log format.
	ECSFormat = `{%time("@timestamp":%"RFC3339Nano"qv,)%level("log.level":%qv,)%msg("message":%qv,)"ecs.version":"1.2.0"%source(,"log.origin":%"json"v)%attr(,%qk:%qv)%grp(,%qg:{%attr(%qk:%qv%nl(,))})}`
	// GELFFormat is the Graylog Extended Log Format (GELF) v1.1 JSON format.
	GELFFormat = `{"version":"1.1",%host("host":%qv)%time(,"timestamp":%"UnixMilli"v)%level(,"level":%"syslog"v)%msg(,"short_message":%qv)%source(,"_source":%qv)%attr(,%_qk:%qv)%grp(%attr(,%_.qg:%qv))}`
	// LogstashFormat is the Logstash JSON event format.
	LogstashFormat = `{"@version":"1"%time(,"@timestamp":%"RFC3339Nano"qv)%level(,"level":%qv)%msg(,"message":%qv)%source(,"source":%"json"v)%attr(,%qk:%qv)%grp(,%qg:{%attr(%qk:%qv%nl(,))})}`
	keyTime        = "time"
	keyLevel       = "level"
	keyMsg         = "msg"
	keySource      = "source"
	keyHost        = "host"
	ansiReset      = "\033[0m"
	strNull        = "null"
	strNilText     = "<nil>"
)

// collectedAttr represents a flattened log attribute along with its group path segments.
type collectedAttr struct {
	attr   slog.Attr
	groups []string
}

// groupInfo represents a logging group containing its formatted name, attributes, and child groups.
type groupInfo struct {
	name     string
	attrs    []slog.Attr
	children []groupInfo
}

var (
	emitterIdentityMu sync.RWMutex
	emitterIdentity   string
	hostnameFunc      = os.Hostname
)

// SetEmitterIdentity sets the emitter identity (hostname) used in log formats like GELF.
func SetEmitterIdentity(identity string) {
	emitterIdentityMu.Lock()
	defer emitterIdentityMu.Unlock()
	emitterIdentity = identity
}

// GetEmitterIdentity returns the currently configured emitter identity, or the system hostname if unset.
func GetEmitterIdentity() string {
	emitterIdentityMu.RLock()
	id := emitterIdentity
	emitterIdentityMu.RUnlock()
	if id != "" {
		return id
	}
	if h, err := hostnameFunc(); err == nil && h != "" {
		return h
	}
	return "localhost"
}

// PatternNode represents a node in the compiled log pattern AST.
type PatternNode interface {
	Render(ctx *renderContext, buf *Buffer)
}

// LiteralNode renders fixed static text.
type LiteralNode struct {
	text string
}

// Render writes literal static text to the output buffer.
func (n *LiteralNode) Render(_ *renderContext, buf *Buffer) {
	//nolint:errcheck // buffer write cannot fail
	buf.WriteString(n.text)
}

// CompositeNode renders an ordered sequence of child pattern nodes.
type CompositeNode struct {
	nodes []PatternNode
}

// Render evaluates and writes all child pattern nodes sequentially to the output buffer.
func (n *CompositeNode) Render(ctx *renderContext, buf *Buffer) {
	for _, child := range n.nodes {
		child.Render(ctx, buf)
	}
}

// ColorNode wraps an inner pattern node with ANSI mode and color styling.
type ColorNode struct {
	inner         PatternNode
	levelCodes    map[int]string
	staticCode    string
	hasColor      bool
	isValid       bool
	hasLevelInner bool
}

// newColorNode creates a ColorNode wrapping an inner pattern node with parsed ANSI styles.
func newColorNode(inner PatternNode, spec string) *ColorNode {
	n := &ColorNode{
		inner:         inner,
		hasLevelInner: hasLevelNode(inner),
	}
	n.parseSpec(spec)
	return n
}

// parseSpec parses a color style specification string into foreground colors and text mode codes.
func (n *ColorNode) parseSpec(spec string) {
	trimmed := strings.TrimSpace(spec)
	if trimmed == "" {
		n.isValid = true
		n.buildCodes(nil, 0)
		return
	}

	parts := strings.Split(trimmed, ",")
	var modes []int
	var colorCode int
	hasColor := false
	recognizedAny := false

	for _, p := range parts {
		token := strings.ToLower(strings.TrimSpace(p))
		token = strings.ReplaceAll(token, "-", "_")
		if token == "" {
			continue
		}

		if mode, ok := ansiModes[token]; ok {
			recognizedAny = true
			if !slices.Contains(modes, mode) {
				modes = append(modes, mode)
			}
			continue
		}

		if code, ok := colorCodes[token]; ok {
			recognizedAny = true
			hasColor = true
			colorCode = code
			continue
		}
	}

	if !recognizedAny {
		n.isValid = false
		return
	}

	n.isValid = true
	n.hasColor = hasColor

	var finalModes []int
	for _, m := range modes {
		if m != 0 {
			finalModes = append(finalModes, m)
		}
	}
	if len(finalModes) == 0 && slices.Contains(modes, 0) {
		finalModes = []int{0}
	}

	n.buildCodes(finalModes, colorCode)
}

// buildCodes resolves ANSI escape codes for the specified color and style modes.
func (n *ColorNode) buildCodes(modes []int, colorCode int) {
	if n.hasColor {
		n.staticCode = buildAnsiCode(modes, colorCode)
	} else {
		n.staticCode = buildAnsiCode(modes, 0)
		n.levelCodes = map[int]string{
			31: buildAnsiCode(modes, 31),
			32: buildAnsiCode(modes, 32),
			33: buildAnsiCode(modes, 33),
			34: buildAnsiCode(modes, 34),
			36: buildAnsiCode(modes, 36),
			0:  buildAnsiCode(modes, 0),
		}
	}
}

// Render evaluates the inner pattern node wrapped in ANSI escape codes if color is enabled.
func (n *ColorNode) Render(ctx *renderContext, buf *Buffer) {
	if !ctx.isColor {
		n.inner.Render(ctx, buf)
		return
	}
	tmp := NewBuffer()
	defer tmp.Free()
	n.inner.Render(ctx, tmp)
	if tmp.Len() == 0 {
		return
	}
	startCode, endCode := n.ansiCodes(ctx)
	if startCode == "" {
		//nolint:errcheck // buffer write cannot fail
		buf.Write(*tmp)
		return
	}
	//nolint:errcheck // buffer writes cannot fail
	buf.WriteString(startCode)
	//nolint:errcheck // buffer writes cannot fail
	buf.Write(*tmp)
	//nolint:errcheck // buffer writes cannot fail
	buf.WriteString(endCode)
}

// ansiCodes returns the ANSI escape sequence for the given record level and styling.
func (n *ColorNode) ansiCodes(ctx *renderContext) (startCode, endCode string) {
	if !n.isValid {
		return "", ""
	}
	if n.hasColor {
		return n.staticCode, ansiReset
	}
	isForLevel := n.hasLevelInner || (ctx.isKey && ctx.currentAttr.Key == slog.LevelKey)
	if isForLevel {
		c := levelColorCode(ctx.record.Level)
		return n.levelCodes[c], ansiReset
	}
	return n.staticCode, ansiReset
}

// KeyNode renders standard slog record attributes (time, level, source, message).
type KeyNode struct {
	inner PatternNode
	key   string
}

// Render binds the current record field or attribute key and evaluates the inner pattern node.
func (n *KeyNode) Render(ctx *renderContext, buf *Buffer) {
	if n.key == keySource && !ctx.addSource {
		return
	}
	var a slog.Attr
	switch n.key {
	case keyHost:
		host := GetEmitterIdentity()
		a = ctx.rep(nil, slog.String("host", host))
	case keyTime:
		if ctx.record.Time.IsZero() {
			return
		}
		val := ctx.record.Time.Round(0)
		repAttr := ctx.rep(nil, slog.Time(slog.TimeKey, val))
		if repAttr.Equal(slog.Attr{}) || repAttr.Key == "" {
			return
		}
		if hasTimeFormat(n.inner) {
			a = slog.Time(slog.TimeKey, val)
		} else {
			a = repAttr
		}
	case keyLevel:
		a = ctx.rep(nil, slog.Any(slog.LevelKey, ctx.record.Level))
	case keySource:
		if ctx.record.PC == 0 {
			return
		}
		fs := runtime.CallersFrames([]uintptr{ctx.record.PC})
		f, _ := fs.Next()
		if f.File == "" {
			return
		}
		src := &slog.Source{
			Function: f.Function,
			File:     f.File,
			Line:     f.Line,
		}
		a = ctx.rep(nil, slog.Any(slog.SourceKey, src))
	case keyMsg:
		a = ctx.rep(nil, slog.String(slog.MessageKey, ctx.record.Message))
	default:
		return
	}

	if a.Key == "" {
		return
	}

	keyCtx := *ctx
	keyCtx.currentAttr = a
	keyCtx.isKey = true
	n.inner.Render(&keyCtx, buf)
}

// AttrNode renders structured log attributes.
type AttrNode struct {
	inner PatternNode
}

// Render iterates over collected attributes, formatting each attribute with the inner pattern.
func (n *AttrNode) Render(ctx *renderContext, buf *Buffer) {
	if ctx.inGroup {
		attrs := ctx.groupAttrs
		for i, attr := range attrs {
			attrCtx := *ctx
			attrCtx.currentAttr = attr
			attrCtx.isKey = false
			attrCtx.inAttr = true
			attrCtx.isLast = (i == len(attrs)-1) && len(ctx.groupChildren) == 0
			n.inner.Render(&attrCtx, buf)
		}
		if ctx.isJson && len(ctx.groupChildren) > 0 {
			renderChildGroupsJSON(buf, ctx.groupChildren, len(attrs) > 0)
		}
		return
	}

	attrs := ctx.topAttrs
	for i, item := range attrs {
		attrCtx := *ctx
		attrCtx.currentAttr = item.attr
		if len(item.groups) > 0 {
			if ctx.isGELF {
				encodedGroups := make([]string, len(item.groups))
				for idx, g := range item.groups {
					encodedGroups[idx] = encodeGELFKey(g)
				}
				attrCtx.currentGroup = strings.Join(encodedGroups, ".")
			} else {
				attrCtx.currentGroup = strings.Join(item.groups, ".")
			}
		} else {
			attrCtx.currentGroup = ""
		}
		attrCtx.isKey = false
		attrCtx.inAttr = true
		attrCtx.isLast = (i == len(attrs)-1)
		n.inner.Render(&attrCtx, buf)
	}
}

// renderChildGroupsJSON renders nested child groups as a JSON object structure.
//
//nolint:errcheck,revive // buffer write cannot fail; helper for child group JSON rendering
func renderChildGroupsJSON(buf *Buffer, children []groupInfo, hasPriorAttrs bool) {
	for i, child := range children {
		if hasPriorAttrs || i > 0 {
			buf.WriteByte(',')
		}
		appendJsonString(buf, child.name)
		buf.WriteByte(':')
		buf.WriteByte('{')
		for j, attr := range child.attrs {
			if j > 0 {
				buf.WriteByte(',')
			}
			appendJsonString(buf, attr.Key)
			buf.WriteByte(':')
			appendFormattedValue(buf, attr.Value, true)
		}
		if len(child.children) > 0 {
			renderChildGroupsJSON(buf, child.children, len(child.attrs) > 0)
		}
		buf.WriteByte('}')
	}
}

// GroupNode renders attribute groups.
type GroupNode struct {
	inner PatternNode
}

// Render formats structured groups using the inner pattern or nested JSON/GELF structures.
func (n *GroupNode) Render(ctx *renderContext, buf *Buffer) {
	groups := ctx.groups
	for i, grp := range groups {
		grpCtx := *ctx
		grpCtx.inGroup = true
		grpCtx.currentGroup = grp.name
		grpCtx.groupAttrs = grp.attrs
		grpCtx.groupChildren = grp.children
		grpCtx.isLast = (i == len(groups)-1)
		n.inner.Render(&grpCtx, buf)
	}
}

// NotLastNode renders text only if the current element is not the last one in an iteration.
type NotLastNode struct {
	text string
}

// Render writes separator text if the current element is not the last one in an iteration.
func (n *NotLastNode) Render(ctx *renderContext, buf *Buffer) {
	if !ctx.isLast {
		//nolint:errcheck // buffer write cannot fail
		buf.WriteString(n.text)
	}
}

// ValueFormatNode formats attribute keys, values, and groups with optional padding, truncation, quoting, or custom layouts.
type ValueFormatNode struct {
	target     string
	prefix     string
	timeFormat string
	minWidth   int
	maxWidth   int
	leftAlign  bool
	hasMax     bool
	truncEnd   bool
}

// Render formats attribute keys, values, or group names applying padding, truncation, or alignment.
func (n *ValueFormatNode) Render(ctx *renderContext, buf *Buffer) {
	var s string
	switch n.target {
	case "k", "qk", "ak", ".k", ".qk", ".ak", "_k", "_qk", "_ak", "q_k", "a_k", "_.k", "_.qk", "_.ak":
		s = n.renderKey(ctx)
	case "v":
		s = n.formatTimeOrValue(ctx, false)
	case "qv":
		s = n.formatTimeOrValue(ctx, true)
	case "av":
		raw := n.formatTimeOrValue(ctx, false)
		runes := []rune(raw)
		if n.hasMax && len(runes) > n.maxWidth {
			if n.truncEnd {
				runes = runes[:n.maxWidth]
			} else {
				runes = runes[len(runes)-n.maxWidth:]
			}
			raw = string(runes)
		}
		if needsQuoting(raw) {
			s = formatJsonString(raw)
		} else {
			s = raw
		}
		sRunes := []rune(s)
		if len(sRunes) < n.minWidth {
			padding := strings.Repeat(" ", n.minWidth-len(sRunes))
			if n.leftAlign {
				s += padding
			} else {
				s = padding + s
			}
		}
		//nolint:errcheck // buffer write cannot fail
		buf.WriteString(s)
		return
	default:
		s = n.renderGroup(ctx)
	}
	s = applyFormatModifier(s, n.minWidth, n.leftAlign, n.hasMax, n.maxWidth, n.truncEnd)
	//nolint:errcheck // buffer write cannot fail
	buf.WriteString(s)
}

// renderKey formats a single attribute or record key with optional prefix and quoting.
func (n *ValueFormatNode) renderKey(ctx *renderContext) string {
	k := ctx.currentAttr.Key
	switch {
	case strings.HasPrefix(n.target, "_."):
		k = "_." + k
		if n.prefix != "" {
			k = n.prefix + k
		}
	case strings.Contains(n.target, "."):
		k = "." + k
		if n.prefix != "" {
			if strings.HasSuffix(n.prefix, ".") {
				k = n.prefix + ctx.currentAttr.Key
			} else {
				k = n.prefix + k
			}
		}
	case strings.Contains(n.target, "_"):
		k = "_" + k
		if n.prefix != "" {
			k = n.prefix + k
		}
	default:
		if n.prefix != "" {
			k = n.prefix + k
		}
	}

	if strings.Contains(n.target, "_") && strings.Contains(n.target, "q") {
		k = encodeGELFKey(k)
	}

	switch {
	case strings.Contains(n.target, "q"):
		return formatJsonString(k)
	case strings.Contains(n.target, "a"):
		if needsQuoting(k) {
			return formatJsonString(k)
		}
		return k
	default:
		return k
	}
}

// renderGroup formats group names or group-qualified attribute keys with optional prefixes and quoting.
func (n *ValueFormatNode) renderGroup(ctx *renderContext) string {
	isDotted := strings.Contains(n.target, ".") || n.target == "gk" || n.target == "qgk" || n.target == "agk"
	isUnderscore := strings.Contains(n.target, "_")
	isGELFEncoding := isUnderscore && strings.Contains(n.target, "q")

	prefix := n.prefix
	attrKey := ctx.currentAttr.Key
	groupName := ctx.currentGroup

	if isGELFEncoding {
		if prefix != "" {
			prefix = encodeGELFKey(prefix)
		}
		attrKey = encodeGELFKey(attrKey)
	}

	var name string
	if ctx.inAttr && (isDotted || isUnderscore) {
		switch {
		case isDotted && isUnderscore:
			name = attrKey
			if groupName != "" {
				name = groupName + "." + attrKey
			}
			name = "_" + name
			if prefix != "" {
				name = prefix + name
			}
		case isDotted:
			name = attrKey
			if groupName != "" {
				name = groupName + "." + attrKey
			}
			if prefix != "" {
				if strings.HasSuffix(prefix, ".") {
					name = prefix + name
				} else {
					name = prefix + "." + name
				}
			}
		default:
			name = attrKey
			if groupName != "" {
				name = groupName + "_" + attrKey
			}
			name = "_" + name
			if prefix != "" {
				name = prefix + name
			}
		}
	} else {
		name = groupName
		if name != "" {
			switch {
			case isDotted && isUnderscore:
				name = "_" + name
				if prefix != "" {
					name = prefix + name
				}
			case isDotted:
				if prefix != "" {
					if strings.HasSuffix(prefix, ".") {
						name = prefix + name
					} else {
						name = prefix + "." + name
					}
				}
			case isUnderscore:
				name = "_" + name
				if prefix != "" {
					name = prefix + name
				}
			default:
				if prefix != "" {
					name = prefix + name
				}
			}
		}
	}

	if name == "" && !ctx.inAttr {
		return ""
	}

	isQuoted := strings.Contains(n.target, "q")
	isAuto := strings.Contains(n.target, "a")

	switch {
	case isQuoted:
		return formatJsonString(name)
	case isAuto:
		if needsQuoting(name) {
			return formatJsonString(name)
		}
		return name
	default:
		return name
	}
}

// formatTimeOrValue formats a timestamp or generic attribute value according to the configured layout.
//
//nolint:revive // quote flag differentiates between %v and %qv
func (n *ValueFormatNode) formatTimeOrValue(ctx *renderContext, quote bool) string {
	if n.timeFormat != "" {
		if strings.EqualFold(n.timeFormat, "syslog") && ctx.isKey && ctx.currentAttr.Key == slog.LevelKey {
			lvl, ok := ctx.currentAttr.Value.Any().(slog.Level)
			if !ok {
				lvl = ctx.record.Level
			}
			return strconv.Itoa(syslogSeverity(lvl))
		}
		var formatted string
		switch {
		case ctx.isKey && ctx.currentAttr.Key == slog.SourceKey:
			formatted = n.formatSource(ctx)
		case ctx.isKey && ctx.currentAttr.Key == slog.TimeKey:
			formatted = formatTime(ctx.record.Time, n.timeFormat)
		case ctx.currentAttr.Value.Kind() == slog.KindTime:
			formatted = formatTime(ctx.currentAttr.Value.Time(), n.timeFormat)
		default:
			formatted = ctx.formatCurrentValue(false)
		}
		if quote {
			return formatJsonString(formatted)
		}
		return formatted
	}
	return ctx.formatCurrentValue(quote)
}

// formatSource formats caller source information according to a custom layout, using ReplaceAttr value if available.
func (n *ValueFormatNode) formatSource(ctx *renderContext) string {
	isJsonLayout := strings.ToLower(strings.TrimSpace(n.timeFormat)) == "json"
	val := ctx.currentAttr.Value.Resolve()

	if ctx.currentAttr.Key == slog.SourceKey {
		if val.Any() == nil {
			if isJsonLayout {
				return strNull
			}
			return strNilText
		}
		if s, ok := val.Any().(*slog.Source); ok && s == nil {
			if isJsonLayout {
				return strNull
			}
			return strNilText
		}
		if val.Kind() == slog.KindString {
			if isJsonLayout {
				return formatJsonString(val.String())
			}
			return val.String()
		}
		var src *slog.Source
		switch s := val.Any().(type) {
		case *slog.Source:
			src = s
		case slog.Source:
			src = &s
		default:
		}
		if src != nil {
			if isJsonLayout {
				return fmt.Sprintf(`{"file":{"name":%s,"line":%d},"function":%s}`,
					formatJsonString(filepath.Base(src.File)),
					src.Line,
					formatJsonString(src.Function),
				)
			}
			return fmt.Sprintf("%s(%s:%d)", ReduceFunctionName(src.Function), filepath.Base(src.File), src.Line)
		}
		return formatValue(val, isJsonLayout)
	}

	if ctx.record.PC == 0 {
		if isJsonLayout {
			return strNull
		}
		return strNilText
	}
	fs := runtime.CallersFrames([]uintptr{ctx.record.PC})
	f, _ := fs.Next()
	src := &slog.Source{
		Function: f.Function,
		File:     f.File,
		Line:     f.Line,
	}
	if isJsonLayout {
		return fmt.Sprintf(`{"file":{"name":%s,"line":%d},"function":%s}`,
			formatJsonString(filepath.Base(src.File)),
			src.Line,
			formatJsonString(src.Function),
		)
	}
	return fmt.Sprintf("%s(%s:%d)", ReduceFunctionName(src.Function), filepath.Base(src.File), src.Line)
}

// isHexDigit reports whether a byte is a valid hexadecimal character.
func isHexDigit(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

// hasLiteralGELFEscape reports whether the string at index i begins a literal GELF escape sequence (_xHH).
// Escape-like sequences in valid keys are escaped too (by escaping the leading underscore to _x5f)
// to prevent collisions with encoded characters; for example, an attribute key user_x2Eid is emitted
// as _user_x5fx2Eid so consumers can query it.
func hasLiteralGELFEscape(s string, i int) bool {
	return s[i] == '_' && i+3 < len(s) && s[i+1] == 'x' && isHexDigit(s[i+2]) && isHexDigit(s[i+3])
}

// isGELFKeyChar reports whether a character is valid in GELF field names ([a-zA-Z0-9_-]).
func isGELFKeyChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-'
}

// encodeGELFKey encodes characters invalid in GELF field names into non-colliding hex escape sequences.
// Escape-like sequences in valid keys are escaped too; for example, an attribute key user_x2Eid is emitted
// as _user_x5fx2Eid so consumers can query it.
func encodeGELFKey(s string) string {
	needsEncoding := false
	for i := 0; i < len(s); i++ {
		if !isGELFKeyChar(s[i]) || hasLiteralGELFEscape(s, i) {
			needsEncoding = true
			break
		}
	}
	if !needsEncoding {
		return s
	}
	const hexDigits = "0123456789abcdef"
	buf := make([]byte, 0, len(s)+8)
	for i := 0; i < len(s); i++ {
		c := s[i]
		if hasLiteralGELFEscape(s, i) {
			buf = append(buf, '_', 'x', '5', 'f')
			continue
		}
		if isGELFKeyChar(c) {
			buf = append(buf, c)
		} else {
			buf = append(buf, '_', 'x', hexDigits[c>>4], hexDigits[c&0x0f])
		}
	}
	return string(buf)
}

// needsQuoting reports whether a key, group, or value string requires quoting in logfmt/text format.
func needsQuoting(s string) bool {
	if s == "" {
		return true
	}
	for _, r := range s {
		if r <= ' ' || r == '=' || r == '"' || r == 0x7f || unicode.IsSpace(r) || unicode.IsControl(r) {
			return true
		}
	}
	return false
}

// formatUnixFractional formats seconds and sub-second nanoseconds as a signed decimal fraction.
func formatUnixFractional(t time.Time, decimals int) string {
	sec := t.Unix()
	nsec := int64(t.Nanosecond())
	if sec >= 0 {
		var frac int64
		switch decimals {
		case 3:
			frac = nsec / 1_000_000
		case 6:
			frac = nsec / 1_000
		default:
			frac = nsec
		}
		return fmt.Sprintf("%d.%0*d", sec, decimals, frac)
	}

	if nsec == 0 {
		return fmt.Sprintf("%d.%0*d", sec, decimals, 0)
	}

	wholeSec := -sec - 1
	remNsec := 1_000_000_000 - nsec
	var frac int64
	switch decimals {
	case 3:
		frac = remNsec / 1_000_000
	case 6:
		frac = remNsec / 1_000
	default:
		frac = remNsec
	}
	return fmt.Sprintf("-%d.%0*d", wholeSec, decimals, frac)
}

// syslogSeverity maps a slog.Level to its corresponding numeric RFC 5424 syslog severity.
func syslogSeverity(level slog.Level) int {
	switch {
	case level >= LevelFatal:
		return 2
	case level >= slog.LevelError:
		return 3
	case level >= slog.LevelWarn:
		return 4
	case level >= slog.LevelInfo:
		return 6
	case level >= LevelTrace:
		return 7
	default:
		return 7
	}
}

// formatTime formats a time instant using standard or predefined epoch layouts.
func formatTime(t time.Time, layout string) string {
	switch strings.ToLower(strings.TrimSpace(layout)) {
	case "unix":
		return strconv.FormatInt(t.Unix(), 10)
	case "unixmilli", "unix_milli":
		return formatUnixFractional(t, 3)
	case "unixmicro", "unix_micro":
		return formatUnixFractional(t, 6)
	case "unixnano", "unix_nano":
		return formatUnixFractional(t, 9)
	case "rfc3339", "iso8601", "iso_8601", "iso-8601":
		return t.Format(time.RFC3339)
	case "rfc3339nano", "rfc3339_nano":
		return t.Format(time.RFC3339Nano)
	case "datetime", "date_time":
		return t.Format(time.DateTime)
	case "dateonly", "date_only":
		return t.Format(time.DateOnly)
	case "timeonly", "time_only":
		return t.Format(time.TimeOnly)
	case "ansic":
		return t.Format(time.ANSIC)
	case "unixdate", "unix_date":
		return t.Format(time.UnixDate)
	case "rubydate", "ruby_date":
		return t.Format(time.RubyDate)
	case "rfc822":
		return t.Format(time.RFC822)
	case "rfc822z":
		return t.Format(time.RFC822Z)
	case "rfc850":
		return t.Format(time.RFC850)
	case "rfc1123":
		return t.Format(time.RFC1123)
	case "rfc1123z":
		return t.Format(time.RFC1123Z)
	case "kitchen":
		return t.Format(time.Kitchen)
	case "stamp":
		return t.Format(time.Stamp)
	case "stampmilli", "stamp_milli":
		return t.Format(time.StampMilli)
	case "stampmicro", "stamp_micro":
		return t.Format(time.StampMicro)
	case "stampnano", "stamp_nano":
		return t.Format(time.StampNano)
	default:
		return t.Format(layout)
	}
}

// compiledPattern holds the root AST node and metadata flags of a parsed pattern.
type compiledPattern struct {
	root         PatternNode
	hasGrpInTree bool
}

// compilePattern parses a pattern string into an AST and inspects its structural features.
func compilePattern(format string) *compiledPattern {
	root := parsePattern(format)
	return &compiledPattern{
		root:         root,
		hasGrpInTree: hasGroupNode(root),
	}
}

// hasGroupNode reports whether the AST contains any %grp node.
func hasGroupNode(node PatternNode) bool {
	switch n := node.(type) {
	case *GroupNode:
		return true
	case *CompositeNode:
		return slices.ContainsFunc(n.nodes, hasGroupNode)
	case *ColorNode:
		return hasGroupNode(n.inner)
	case *KeyNode:
		return hasGroupNode(n.inner)
	case *AttrNode:
		return hasGroupNode(n.inner)
	default:
		return false
	}
}

// hasLevelNode reports whether the AST contains any %level node.
func hasLevelNode(node PatternNode) bool {
	if node == nil {
		return false
	}
	switch n := node.(type) {
	case *KeyNode:
		if n.key == keyLevel {
			return true
		}
		return hasLevelNode(n.inner)
	case *CompositeNode:
		return slices.ContainsFunc(n.nodes, hasLevelNode)
	case *ColorNode:
		return hasLevelNode(n.inner)
	case *AttrNode:
		return hasLevelNode(n.inner)
	case *GroupNode:
		return hasLevelNode(n.inner)
	default:
		return false
	}
}

// hasTimeFormat reports whether the AST contains a custom time layout specifier.
func hasTimeFormat(node PatternNode) bool {
	switch n := node.(type) {
	case *ValueFormatNode:
		return n.timeFormat != ""
	case *CompositeNode:
		return slices.ContainsFunc(n.nodes, hasTimeFormat)
	case *ColorNode:
		return hasTimeFormat(n.inner)
	case *KeyNode:
		return hasTimeFormat(n.inner)
	case *AttrNode:
		return hasTimeFormat(n.inner)
	case *GroupNode:
		return hasTimeFormat(n.inner)
	default:
		return false
	}
}

var formatSpecRegex = regexp.MustCompile(`^%(?:"([^"]*)")?(-)?(\d+)?(?:\.(-)?(\d+))?(?:"([^"]*)")?(\.qg|\.ag|\.g|\.qk|\.ak|\.k|_\.qg|_\.ag|_\.g|_\.qk|_\.ak|_\.k|_qg|_ag|_g|_qk|_ak|_k|q_gk|q_g_k|q_k|a_gk|a_g_k|a_k|q_g|a_g|agk|qgk|gk|qk|qv|qg|ak|av|ag|k|v|g)`)

// matchFormatSpec parses a value or key format placeholder (%[-][min][.[-]max]["layout"]target).
func matchFormatSpec(s string) (*ValueFormatNode, int, bool) {
	m := formatSpecRegex.FindStringSubmatch(s)
	if m == nil {
		return nil, 0, false
	}
	matchedLen := len(m[0])
	prefix := m[1]
	if prefix == "" {
		prefix = m[6]
	}
	timeFormat := ""
	target := m[7]
	if target == "v" || target == "qv" || target == "av" {
		timeFormat = prefix
	}
	leftAlign := m[2] == "-"
	minWidth := 0
	if m[3] != "" {
		if val, err := strconv.Atoi(m[3]); err == nil {
			minWidth = val
		}
	}
	truncEnd := m[4] == "-"
	hasMax := false
	maxWidth := 0
	if m[5] != "" {
		if val, err := strconv.Atoi(m[5]); err == nil {
			hasMax = true
			maxWidth = val
		}
	}
	return &ValueFormatNode{
		target:     target,
		prefix:     prefix,
		timeFormat: timeFormat,
		minWidth:   minWidth,
		leftAlign:  leftAlign,
		hasMax:     hasMax,
		maxWidth:   maxWidth,
		truncEnd:   truncEnd,
	}, matchedLen, true
}

// matchKeyPrefix matches a known field placeholder prefix such as %time( or %level(.
func matchKeyPrefix(s string) (key string, prefixLen int, ok bool) {
	prefixes := []struct {
		prefix string
		key    string
	}{
		{"%time(", keyTime},
		{"%level(", keyLevel},
		{"%msg(", keyMsg},
		{"%source(", keySource},
		{"%host(", keyHost},
		{"%t(", keyTime},
		{"%l(", keyLevel},
		{"%m(", keyMsg},
		{"%s(", keySource},
		{"%h(", keyHost},
	}
	for _, p := range prefixes {
		if strings.HasPrefix(s, p.prefix) {
			return p.key, len(p.prefix), true
		}
	}
	return "", 0, false
}

// isQuotedLayoutStart reports whether a double quote marks the start of a format layout specifier.
func isQuotedLayoutStart(s string, quoteIdx int) bool {
	if quoteIdx == 0 || s[quoteIdx] != '"' {
		return false
	}
	j := quoteIdx - 1
	for j >= 0 && (s[j] >= '0' && s[j] <= '9' || s[j] == '-' || s[j] == '.') {
		j--
	}
	if j >= 0 && s[j] == '%' {
		pctCount := 0
		for k := j; k >= 0 && s[k] == '%'; k-- {
			pctCount++
		}
		return pctCount%2 == 1
	}
	return false
}

// findMatchingParen scans for the matching closing parenthesis while respecting quoted layouts.
func findMatchingParen(s string, openParenIdx int) (int, bool) {
	if openParenIdx >= len(s) || s[openParenIdx] != '(' {
		return -1, false
	}
	depth := 1
	inQuote := false
	for i := openParenIdx + 1; i < len(s); i++ {
		c := s[i]
		if inQuote {
			if c == '\\' && i+1 < len(s) {
				i++
				continue
			}
			if c == '"' {
				inQuote = false
			}
			continue
		}
		if c == '"' && isQuotedLayoutStart(s, i) {
			inQuote = true
			continue
		}
		switch c {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i, true
			}
		default:
		}
	}
	return -1, false
}

// parsePattern parses a format string into a tree of PatternNodes.
func parsePattern(format string) PatternNode {
	var nodes []PatternNode
	i := 0
	for i < len(format) {
		if format[i] != '%' {
			next := strings.IndexByte(format[i:], '%')
			if next == -1 {
				nodes = append(nodes, &LiteralNode{text: format[i:]})
				break
			}
			nodes = append(nodes, &LiteralNode{text: format[i : i+next]})
			i += next
			continue
		}

		if strings.HasPrefix(format[i:], "%%") {
			nodes = append(nodes, &LiteralNode{text: "%"})
			i += 2
			continue
		}

		if strings.HasPrefix(format[i:], "%clr(") {
			closeParen, ok := findMatchingParen(format, i+4)
			if ok {
				innerStr := format[i+5 : closeParen]
				i = closeParen + 1
				color := ""
				if i < len(format) && format[i] == '{' {
					closeBrace := strings.IndexByte(format[i:], '}')
					if closeBrace != -1 {
						color = format[i+1 : i+closeBrace]
						i = i + closeBrace + 1
					}
				}
				innerNode := parsePattern(innerStr)
				nodes = append(nodes, newColorNode(innerNode, color))
				continue
			}
		}

		if key, prefixLen, ok := matchKeyPrefix(format[i:]); ok {
			closeParen, ok2 := findMatchingParen(format, i+prefixLen-1)
			if ok2 {
				innerStr := format[i+prefixLen : closeParen]
				i = closeParen + 1
				innerNode := parsePattern(innerStr)
				nodes = append(nodes, &KeyNode{key: key, inner: innerNode})
				continue
			}
		}

		if strings.HasPrefix(format[i:], "%attr(") {
			closeParen, ok := findMatchingParen(format, i+5)
			if ok {
				innerStr := format[i+6 : closeParen]
				i = closeParen + 1
				innerNode := parsePattern(innerStr)
				nodes = append(nodes, &AttrNode{inner: innerNode})
				continue
			}
		}

		if strings.HasPrefix(format[i:], "%grp(") {
			closeParen, ok := findMatchingParen(format, i+4)
			if ok {
				innerStr := format[i+5 : closeParen]
				i = closeParen + 1
				innerNode := parsePattern(innerStr)
				nodes = append(nodes, &GroupNode{inner: innerNode})
				continue
			}
		}

		if strings.HasPrefix(format[i:], "%nl(") {
			closeParen, ok := findMatchingParen(format, i+3)
			if ok {
				text := format[i+4 : closeParen]
				i = closeParen + 1
				nodes = append(nodes, &NotLastNode{text: text})
				continue
			}
		}

		if spec, matchedLen, ok := matchFormatSpec(format[i:]); ok {
			nodes = append(nodes, spec)
			i += matchedLen
			continue
		}

		nodes = append(nodes, &LiteralNode{text: "%"})
		i++
	}

	if len(nodes) == 1 {
		return nodes[0]
	}
	return &CompositeNode{nodes: nodes}
}

// renderContext holds runtime rendering state and metadata for evaluating pattern ASTs.
type renderContext struct {
	currentAttr   slog.Attr
	rep           func(groups []string, a slog.Attr) slog.Attr
	currentGroup  string
	format        string
	record        slog.Record
	groupChildren []groupInfo
	topAttrs      []collectedAttr
	groups        []groupInfo
	groupAttrs    []slog.Attr
	hasGrpInTree  bool
	addSource     bool
	isColor       bool
	isKey         bool
	isLast        bool
	inGroup       bool
	inAttr        bool
	isJson        bool
	isGELF        bool
}

// formatCurrentValue formats the current attribute or record value as text or JSON.
//
//nolint:revive // quote flag is required to differentiate between %v and %qv
func (ctx *renderContext) formatCurrentValue(quote bool) string {
	if ctx.isGELF && ctx.currentAttr.Value.Kind() == slog.KindBool {
		if ctx.currentAttr.Value.Bool() {
			return "1"
		}
		return "0"
	}

	if ctx.isKey {
		v := ctx.currentAttr.Value.Resolve()
		switch ctx.currentAttr.Key {
		case slog.TimeKey:
			if v.Kind() == slog.KindString {
				if quote {
					return formatJsonString(v.String())
				}
				return v.String()
			}
			if v.Kind() == slog.KindTime {
				formatted := v.Time().Format(time.RFC3339)
				if quote {
					return formatJsonString(formatted)
				}
				return formatted
			}
			s := v.String()
			if quote {
				return formatJsonString(s)
			}
			return s
		case slog.LevelKey:
			if v.Kind() == slog.KindString {
				if quote {
					return formatJsonString(v.String())
				}
				return v.String()
			}
			if lvl, ok := v.Any().(slog.Level); ok {
				var s string
				switch lvl {
				case LevelFatal:
					s = "FATAL"
				case LevelTrace:
					s = "TRACE"
				default:
					s = lvl.String()
				}
				if quote {
					return formatJsonString(s)
				}
				return s
			}
			s := v.String()
			if quote {
				return formatJsonString(s)
			}
			return s
		case slog.SourceKey:
			if v.Any() == nil {
				return formatValue(v, quote, ctx.isGELF)
			}
			if s, ok := v.Any().(*slog.Source); ok && s == nil {
				return formatValue(v, quote, ctx.isGELF)
			}
			if v.Kind() == slog.KindString {
				if quote {
					return formatJsonString(v.String())
				}
				return v.String()
			}
			var src *slog.Source
			switch s := v.Any().(type) {
			case *slog.Source:
				src = s
			case slog.Source:
				src = &s
			default:
			}
			if src != nil {
				formatted := fmt.Sprintf("%s(%s:%d)", ReduceFunctionName(src.Function), filepath.Base(src.File), src.Line)
				if quote {
					return formatJsonString(formatted)
				}
				return formatted
			}
			s := fmt.Sprintf("%+v", v.Any())
			if quote {
				return formatJsonString(s)
			}
			return s
		case slog.MessageKey:
			if quote {
				return formatJsonString(v.String())
			}
			return v.String()
		default:
			return formatValue(v, quote, ctx.isGELF)
		}
	}
	return formatValue(ctx.currentAttr.Value, quote, ctx.isGELF)
}

// applyFormatModifier applies minimum width, maximum width, alignment, and truncation to a string.
//
//nolint:revive // boolean flags control left alignment, maximum length, and truncation direction
func applyFormatModifier(s string, minWidth int, leftAlign, hasMax bool, maxWidth int, truncEnd bool) string {
	runes := []rune(s)
	if hasMax && len(runes) > maxWidth {
		if truncEnd {
			runes = runes[:maxWidth]
		} else {
			runes = runes[len(runes)-maxWidth:]
		}
		s = string(runes)
	}
	if len(runes) < minWidth {
		padding := strings.Repeat(" ", minWidth-len(runes))
		if leftAlign {
			s += padding
		} else {
			s = padding + s
		}
	}
	return s
}

var jsonMarshalFunc = json.Marshal

// formatJsonString escapes and quotes a string for JSON output.
func formatJsonString(s string) string {
	b, err := jsonMarshalFunc(s)
	if err == nil {
		return string(b)
	}
	return strconv.Quote(s)
}

// appendJsonString appends an escaped, quoted string to the buffer.
func appendJsonString(buf *Buffer, s string) {
	//nolint:errcheck // buffer write cannot fail
	buf.WriteString(formatJsonString(s))
}

// formatValue formats an arbitrary slog.Value into a string representation.
func formatValue(v slog.Value, quote bool, isGELF ...bool) string {
	buf := NewBuffer()
	defer buf.Free()
	appendFormattedValue(buf, v, quote, isGELF...)
	return string(*buf)
}

// appendNilValue renders a nil value into the buffer using a GELF-aware format.
// As an exception for the GELF specification (where all additional fields must be strings or numbers),
// nil values are encoded as a JSON string ("null") in GELF mode when quoted. Outside GELF,
// nil is preserved as JSON null when quoted, or <nil> when unquoted in text/logfmt mode.
//
//nolint:revive // boolean flags control quoting and GELF mode
func appendNilValue(buf *Buffer, quote, gelfMode bool) {
	if quote {
		if gelfMode {
			appendJsonString(buf, strNull)
		} else {
			//nolint:errcheck // buffer write cannot fail
			buf.WriteString(strNull)
		}
	} else {
		//nolint:errcheck // buffer write cannot fail
		buf.WriteString(strNilText)
	}
}

// appendFormattedValue renders a structured value into the buffer in JSON or text format.
//
//nolint:revive // quote controls whether values like strings are quoted
func appendFormattedValue(buf *Buffer, v slog.Value, quote bool, isGELF ...bool) {
	gelfMode := len(isGELF) > 0 && isGELF[0]
	switch v.Kind() {
	case slog.KindString:
		appendString(buf, v.String(), quote)
	case slog.KindInt64:
		*buf = strconv.AppendInt(*buf, v.Int64(), 10)
	case slog.KindUint64:
		*buf = strconv.AppendUint(*buf, v.Uint64(), 10)
	case slog.KindFloat64:
		f := v.Float64()
		if quote {
			if math.IsNaN(f) {
				appendString(buf, "!FLOAT:NaN", true)
				return
			}
			if math.IsInf(f, 1) {
				appendString(buf, "!FLOAT:+Inf", true)
				return
			}
			if math.IsInf(f, -1) {
				appendString(buf, "!FLOAT:-Inf", true)
				return
			}
		}
		*buf = strconv.AppendFloat(*buf, f, 'g', -1, 64)
	case slog.KindBool:
		*buf = strconv.AppendBool(*buf, v.Bool())
	case slog.KindDuration:
		appendString(buf, v.Duration().String(), quote)
	case slog.KindTime:
		appendString(buf, v.Time().String(), quote)
	case slog.KindAny:
		if v.Any() == nil {
			appendNilValue(buf, quote, gelfMode)
			return
		}
		defer func() {
			if r := recover(); r != nil {
				if val := reflect.ValueOf(v.Any()); val.Kind() == reflect.Pointer && val.IsNil() {
					appendNilValue(buf, quote, gelfMode)
					return
				}
				appendString(buf, fmt.Sprintf("!PANIC: %v", r), true)
			}
		}()

		switch cv := v.Any().(type) {
		case encoding.TextMarshaler:
			data, err := cv.MarshalText()
			if err != nil {
				appendString(buf, fmt.Sprintf("!ERROR: %v", err), quote)
				return
			}
			appendString(buf, string(data), quote)
		case *slog.Source:
			if cv == nil {
				appendNilValue(buf, quote, gelfMode)
				return
			}
			if quote {
				appendString(buf, fmt.Sprintf("%s:%d", filepath.Join(filepath.Base(filepath.Dir(cv.File)), filepath.Base(cv.File)), cv.Line), true)
			} else {
				appendSource(buf, cv)
			}
		default:
			val := reflect.ValueOf(cv)
			if val.Kind() == reflect.Pointer && val.IsNil() {
				appendNilValue(buf, quote, gelfMode)
				return
			}
			if quote {
				if jm, ok := cv.(json.Marshaler); ok {
					b, err := jm.MarshalJSON()
					if err == nil {
						if gelfMode {
							appendJsonString(buf, string(b))
						} else {
							//nolint:errcheck // buffer write cannot fail
							buf.WriteString(string(b))
						}
						return
					}
				}
				if val.Kind() == reflect.Map || val.Kind() == reflect.Slice || val.Kind() == reflect.Array || val.Kind() == reflect.Struct {
					b, err := json.Marshal(cv)
					if err == nil {
						if gelfMode {
							appendJsonString(buf, string(b))
						} else {
							//nolint:errcheck // buffer write cannot fail
							buf.WriteString(string(b))
						}
						return
					}
				}
			}
			appendString(buf, fmt.Sprintf("%+v", cv), quote)
		}
	default:
	}
}

var ansiModes = map[string]int{
	"normal":    0,
	"bold":      1,
	"faint":     2,
	"italic":    3,
	"underline": 4,
	"blinky":    5,
	"blink":     5,
}

var colorCodes = map[string]int{
	"black":          30,
	"red":            31,
	"green":          32,
	"yellow":         33,
	"blue":           34,
	"magenta":        35,
	"cyan":           36,
	"white":          37,
	"default":        39,
	"bright_black":   90,
	"bright_red":     91,
	"bright_green":   92,
	"bright_yellow":  93,
	"bright_blue":    94,
	"bright_magenta": 95,
	"bright_cyan":    96,
	"bright_white":   97,
}

// levelColorCode returns the default ANSI foreground color code for a given log level.
func levelColorCode(level slog.Level) int {
	switch {
	case level >= LevelError:
		return 31
	case level >= LevelWarn:
		return 33
	case level >= LevelInfo:
		return 32
	case level >= LevelDebug:
		return 36
	default:
		return 0
	}
}

// buildAnsiCode constructs an ANSI escape sequence string from text modes and color code.
//
//nolint:errcheck // strings.Builder write cannot fail
func buildAnsiCode(modes []int, color int) string {
	if len(modes) == 0 && color <= 0 {
		return "\033[0m"
	}
	var sb strings.Builder
	sb.WriteString("\033[")
	for i, m := range modes {
		if i > 0 {
			sb.WriteByte(';')
		}
		sb.WriteString(strconv.Itoa(m))
	}
	if color > 0 {
		if len(modes) > 0 {
			sb.WriteByte(';')
		}
		sb.WriteString(strconv.Itoa(color))
	}
	sb.WriteByte('m')
	return sb.String()
}

// isTerminal reports whether the output writer is an ANSI-capable terminal based on environment and writer capabilities.
func isTerminal(w io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	for _, arg := range os.Args {
		trimmed := strings.TrimLeft(arg, "-")
		if trimmed == "no-color" || trimmed == "ginkgo.no-color" ||
			strings.HasPrefix(trimmed, "no-color=") || strings.HasPrefix(trimmed, "ginkgo.no-color=") {
			return false
		}
	}
	if os.Getenv("TERM") == "dumb" && os.Getenv("COLORTERM") == "" {
		return false
	}
	if t, ok := w.(interface{ IsTerminal() bool }); ok {
		return t.IsTerminal()
	}
	if f, ok := w.(interface{ Fd() uintptr }); ok {
		fd := int(f.Fd())
		return term.IsTerminal(fd) && checkTerminalVT(fd)
	}
	if _, ok := w.(interface {
		TeeTo(io.Writer)
		ClearTeeWriters()
	}); ok {
		fd := int(os.Stdout.Fd())
		return term.IsTerminal(fd) && checkTerminalVT(fd)
	}
	return false
}
