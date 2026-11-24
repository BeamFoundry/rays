package logger

import (
	"context"
	"log/slog"
	"slices"
	"sync"
)

// wrapperChain stores a thread-safe, versioned list of slog.Handler wrappers
// shared between a root Logger and all of its cloned instances.
type wrapperChain struct {
	wrappers []func(slog.Handler) slog.Handler
	version  uint64
	mu       sync.RWMutex
}

func (this *wrapperChain) add(fn func(slog.Handler) slog.Handler) {
	this.mu.Lock()
	defer this.mu.Unlock()
	this.wrappers = append(this.wrappers, fn)
	this.version++
}

func (this *wrapperChain) clear() {
	this.mu.Lock()
	defer this.mu.Unlock()
	this.wrappers = nil
	this.version++
}

func (this *wrapperChain) snapshot() (wrappers []func(slog.Handler) slog.Handler, version uint64) {
	this.mu.RLock()
	defer this.mu.RUnlock()
	if len(this.wrappers) == 0 {
		return nil, this.version
	}
	return slices.Clone(this.wrappers), this.version
}

type handlerOp struct {
	group string
	attrs []slog.Attr
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
	wrappers, ver := this.shared.snapshot()

	this.mu.RLock()
	if this.cached != nil && this.cachedVersion == ver {
		h := this.cached
		this.mu.RUnlock()
		return h
	}
	this.mu.RUnlock()

	this.mu.Lock()
	defer this.mu.Unlock()

	h := this.rootBase
	for _, wrap := range wrappers {
		h = wrap(h)
	}
	for _, op := range this.ops {
		if op.group != "" {
			h = h.WithGroup(op.group)
		} else {
			h = h.WithAttrs(op.attrs)
		}
	}
	this.cached = h
	this.cachedVersion = ver
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

// WithAttrs returns a new proxyHandler with the given attributes appended.
func (this *proxyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return this
	}
	ops := make([]handlerOp, len(this.ops)+1)
	copy(ops, this.ops)
	ops[len(this.ops)] = handlerOp{attrs: slices.Clone(attrs)}
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
