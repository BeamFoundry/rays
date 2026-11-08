package logger

import (
	"log/slog"
)

// Exported for internal package tests.
var (
	CheckTerminalVT      = checkTerminalVT
	CleanJsonCommas      = cleanJsonCommas
	FormatUnixFractional = formatUnixFractional
	SyslogSeverity       = syslogSeverity
	HasLevelNode         = hasLevelNode
	IsQuotedLayoutStart  = isQuotedLayoutStart
	FindMatchingParen    = findMatchingParen
	FormatJsonString     = formatJsonString
	AppendFormattedValue = appendFormattedValue
	IsTerminal           = isTerminal
	NeedsQuoting         = needsQuoting
	SetHostnameFunc      = func(f func() (string, error)) { hostnameFunc = f }
	SetJsonMarshalFunc   = func(f func(any) ([]byte, error)) { jsonMarshalFunc = f }
	NewUnknownKeyNode    = func() PatternNode { return &KeyNode{key: "unknown"} }
	NewUnknownValueNode  = func() PatternNode { return &ValueFormatNode{target: "unknown"} }
	NewRenderContext     = func() *renderContext {
		return &renderContext{rep: func(_ []string, a slog.Attr) slog.Attr { return a }}
	}
	FormatSourceHelper = func(timeFormat string, attr slog.Attr, pc uintptr) string {
		node := &ValueFormatNode{timeFormat: timeFormat}
		ctx := &renderContext{
			currentAttr: attr,
			record:      slog.Record{PC: pc},
		}
		return node.formatSource(ctx)
	}
)
