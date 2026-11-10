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
	SetHostnameFunc      = func(f func() (string, error)) {
		emitterIdentityMu.Lock()
		defer emitterIdentityMu.Unlock()
		hostnameFunc = f
		cachedHostname = ""
	}
	SetJsonMarshalFunc  = func(f func(any) ([]byte, error)) { jsonMarshalFunc = f }
	CloneOptions        = func(l *Logger) *slog.HandlerOptions { return l.cloneOptions() }
	NewUnknownKeyNode   = func() PatternNode { return &KeyNode{key: "unknown"} }
	NewUnknownValueNode = func() PatternNode { return &ValueFormatNode{target: "unknown"} }
	NewRenderContext    = func() *renderContext {
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
	GetFunctionCacheEntry = func(fn string) (string, bool) {
		functionLock.Lock()
		defer functionLock.Unlock()
		v, ok := functionCache[fn]
		return v, ok
	}
	SetFunctionCacheLimit = func(limit int) {
		functionLock.Lock()
		defer functionLock.Unlock()
		maxFunctionCacheSize = limit
	}
	ResetFunctionCacheLimit = func() {
		functionLock.Lock()
		defer functionLock.Unlock()
		maxFunctionCacheSize = defaultMaxFunctionCacheSize
	}
	GetFunctionCacheLen = func() int {
		functionLock.Lock()
		defer functionLock.Unlock()
		return len(functionCache)
	}
)
