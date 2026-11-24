package logger

import (
	"context"
	"log/slog"
	"slices"
	"sync"
	"sync/atomic"
)

type wrapperSnapshot struct {
	wrappers []func(slog.Handler) slog.Handler
	version  uint64
}

// wrapperChain stores a thread-safe, versioned list of slog.Handler wrappers
// shared between a root Logger and all of its cloned instances.
type wrapperChain struct {
	current atomic.Pointer[wrapperSnapshot]
	mu      sync.Mutex
}

func (this *wrapperChain) add(fn func(slog.Handler) slog.Handler) {
	this.mu.Lock()
	defer this.mu.Unlock()
	prev := this.current.Load()
	var nextWrappers []func(slog.Handler) slog.Handler
	var nextVersion uint64 = 1
	if prev != nil {
		nextWrappers = make([]func(slog.Handler) slog.Handler, len(prev.wrappers)+1)
		copy(nextWrappers, prev.wrappers)
		nextVersion = prev.version + 1
	} else {
		nextWrappers = make([]func(slog.Handler) slog.Handler, 1)
	}
	nextWrappers[len(nextWrappers)-1] = fn
	this.current.Store(&wrapperSnapshot{
		wrappers: nextWrappers,
		version:  nextVersion,
	})
}

func (this *wrapperChain) clear() {
	this.mu.Lock()
	defer this.mu.Unlock()
	prev := this.current.Load()
	var nextVersion uint64 = 1
	if prev != nil {
		nextVersion = prev.version + 1
	}
	this.current.Store(&wrapperSnapshot{
		version: nextVersion,
	})
}

func (this *wrapperChain) snapshot() *wrapperSnapshot {
	if snap := this.current.Load(); snap != nil {
		return snap
	}
	return &wrapperSnapshot{}
}

type handlerOp struct {
	group string
	attrs []slog.Attr
}

func resolveAttr(attr slog.Attr) slog.Attr {
	attr.Value = attr.Value.Resolve()
	if attr.Value.Kind() == slog.KindGroup {
		group := attr.Value.Group()
		resolved := make([]slog.Attr, len(group))
		for i, a := range group {
			resolved[i] = resolveAttr(a)
		}
		attr.Value = slog.GroupValue(resolved...)
	}
	return attr
}

func resolveAttrs(attrs []slog.Attr) []slog.Attr {
	resolved := make([]slog.Attr, len(attrs))
	for i, a := range attrs {
		resolved[i] = resolveAttr(a)
	}
	return resolved
}

// proxyHandler delegates to a base slog.Handler decorated by a shared wrapperChain,
// lazily rebuilding the effective handler chain whenever new wrappers are registered.
type proxyHandler struct {
	rootBase      slog.Handler
	cached        slog.Handler
	shared        *wrapperChain
	ops           []handlerOp
	cachedVersion uint64
	mu            sync.RWMutex
}

func newProxyHandler(base slog.Handler, shared *wrapperChain) *proxyHandler {
	return &proxyHandler{
		rootBase: base,
		shared:   shared,
	}
}

func (this *proxyHandler) resolve() slog.Handler {
	snap := this.shared.snapshot()

	this.mu.RLock()
	if this.cached != nil && this.cachedVersion == snap.version {
		h := this.cached
		this.mu.RUnlock()
		return h
	}
	this.mu.RUnlock()

	this.mu.Lock()
	defer this.mu.Unlock()

	h := this.rootBase
	for _, wrap := range snap.wrappers {
		h = wrap(h)
	}
	for _, op := range this.ops {
		if op.group != "" {
			h = h.WithGroup(op.group)
		} else {
			h = h.WithAttrs(slices.Clone(op.attrs))
		}
	}
	this.cached = h
	this.cachedVersion = snap.version
	return h
}

// Enabled reports whether the effective handler emits records at the given level.
func (this *proxyHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return this.resolve().Enabled(ctx, level)
}

// Handle forwards the log record to the effective handler.
//
//nolint:gocritic,wrapcheck // Have to respect slog.Handler interface
func (this *proxyHandler) Handle(ctx context.Context, r slog.Record) error {
	return this.resolve().Handle(ctx, r)
}

// WithAttrs returns a new proxyHandler with the given attributes resolved and appended.
func (this *proxyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return this
	}
	ops := make([]handlerOp, len(this.ops)+1)
	copy(ops, this.ops)
	ops[len(this.ops)] = handlerOp{attrs: resolveAttrs(attrs)}
	return &proxyHandler{
		rootBase: this.rootBase,
		shared:   this.shared,
		ops:      ops,
	}
}

// WithGroup returns a new proxyHandler with the given group name appended.
func (this *proxyHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return this
	}
	ops := make([]handlerOp, len(this.ops)+1)
	copy(ops, this.ops)
	ops[len(this.ops)] = handlerOp{group: name}
	return &proxyHandler{
		rootBase: this.rootBase,
		shared:   this.shared,
		ops:      ops,
	}
}
