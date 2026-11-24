package logger_test

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"strconv"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/BeamFoundry/rays/internal/boot/logger"
)

type spyHandler struct {
	inner     slog.Handler
	mu        *sync.Mutex
	records   *[]string
	tag       string
	withAttrs []slog.Attr
	groups    []string
}

func newSpyHandler(inner slog.Handler, tag string, records *[]string, mu *sync.Mutex) *spyHandler {
	return &spyHandler{
		inner:   inner,
		tag:     tag,
		records: records,
		mu:      mu,
	}
}

func (this *spyHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return this.inner.Enabled(ctx, level)
}

//nolint:gocritic,wrapcheck // Have to respect slog.Handler interface
func (this *spyHandler) Handle(ctx context.Context, r slog.Record) error {
	this.mu.Lock()
	*this.records = append(*this.records, this.tag+":"+r.Message)
	this.mu.Unlock()
	return this.inner.Handle(ctx, r)
}

func (this *spyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &spyHandler{
		inner:     this.inner.WithAttrs(attrs),
		mu:        this.mu,
		records:   this.records,
		tag:       this.tag,
		withAttrs: append(append([]slog.Attr(nil), this.withAttrs...), attrs...),
		groups:    this.groups,
	}
}

func (this *spyHandler) WithGroup(name string) slog.Handler {
	return &spyHandler{
		inner:     this.inner.WithGroup(name),
		mu:        this.mu,
		records:   this.records,
		tag:       this.tag,
		withAttrs: this.withAttrs,
		groups:    append(append([]string(nil), this.groups...), name),
	}
}

var _ = Describe("Logger WrapHandler proxy", func() {
	It("wraps the root logger and already-cloned loggers dynamically in registration order", func() {
		var buf bytes.Buffer
		lg := logger.NewLogger()
		lg.SetOutput(&buf)
		lg.SetColor(false)

		cloned := lg.Clone()
		withAttr := cloned.With("component", "auth")
		withGroup := withAttr.WithGroup("req")

		// No-op nil wrapper, empty With, empty WithGroup
		lg.WrapHandler(nil)
		Expect(withAttr.With().Handler()).NotTo(BeNil())
		Expect(withAttr.WithGroup("").Handler()).NotTo(BeNil())

		var mu sync.Mutex
		var seen []string

		lg.WrapHandler(func(next slog.Handler) slog.Handler {
			return newSpyHandler(next, "first", &seen, &mu)
		})
		lg.WrapHandler(func(next slog.Handler) slog.Handler {
			return newSpyHandler(next, "second", &seen, &mu)
		})

		lg.Info("root-msg")
		cloned.Info("cloned-msg")
		withGroup.Info("grouped-msg", "k", "v")

		Expect(seen).To(Equal([]string{
			"second:root-msg", "first:root-msg",
			"second:cloned-msg", "first:cloned-msg",
			"second:grouped-msg", "first:grouped-msg",
		}))
		out := buf.String()
		Expect(out).To(ContainSubstring("root-msg"))
		Expect(out).To(ContainSubstring("cloned-msg"))
		Expect(out).To(ContainSubstring("grouped-msg"))
		Expect(out).To(ContainSubstring("component=auth"))
		Expect(out).To(ContainSubstring("req.k=v"))

		// Reset clears wrappers
		lg.Reset()
		seen = nil
		lg.Info("after-reset")
		Expect(seen).To(BeEmpty())
	})

	It("handles concurrent resolve calls safely and nil/non-proxy handlers gracefully", func() {
		var zero logger.Logger
		zero.WrapHandler(func(next slog.Handler) slog.Handler { return next })
		Expect(zero.Handler()).To(BeNil())
		Expect(zero.GetFormat()).To(BeEmpty())
		zero.Reset()

		nonProxy := logger.Logger{Logger: slog.New(slog.DiscardHandler)}
		Expect(nonProxy.Handler()).NotTo(BeNil())
		Expect(nonProxy.GetFormat()).To(BeEmpty())
		nonProxy.SetHandler(func(w io.Writer, opts *slog.HandlerOptions) slog.Handler {
			return slog.NewTextHandler(w, opts)
		})
		// Force non-proxy Logger.Logger again, then Clone() so shared == nil in newProxyHandler
		nonProxy.Logger = slog.New(slog.DiscardHandler)
		nonProxy.SetOutput(&bytes.Buffer{})
		nonProxy.Logger = slog.New(slog.DiscardHandler)
		clonedFromNonProxy := nonProxy.Clone()
		clonedFromNonProxy.WrapHandler(func(next slog.Handler) slog.Handler { return next })
		clonedFromNonProxy.Info("from-non-proxy-clone")

		lg := logger.NewLogger()
		lg.SetOutput(&bytes.Buffer{})
		lg.SetHandler(func(w io.Writer, opts *slog.HandlerOptions) slog.Handler {
			return slog.NewTextHandler(w, opts)
		})
		lg.WrapHandler(func(next slog.Handler) slog.Handler {
			return next
		})

		// Directly exercise empty WithAttrs and WithGroup on proxyHandler
		rawProxy := lg.Logger.Handler()
		Expect(rawProxy.WithAttrs(nil)).To(Equal(rawProxy))
		Expect(rawProxy.WithGroup("")).To(Equal(rawProxy))

		start := make(chan struct{})
		var wg sync.WaitGroup
		for i := range 16 {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				<-start
				lg.Info("concurrent", "i", idx)
			}(i)
		}
		close(start)
		wg.Wait()
	})

	It("resolves LogValuer attributes (including inside groups) once at WithAttrs time", func() {
		var buf bytes.Buffer
		lg := logger.NewLogger()
		lg.SetOutput(&buf)
		lg.SetColor(false)

		calls := 0
		valuer := countingValuer{calls: &calls}

		child := lg.With(
			"top", valuer,
			slog.Group("nested", slog.Any("inner", valuer)),
		)
		Expect(calls).To(Equal(2))

		// Registering a new wrapper replays ops without re-evaluating LogValuer
		lg.WrapHandler(func(next slog.Handler) slog.Handler {
			return next
		})

		child.Info("valuer-msg")
		Expect(calls).To(Equal(2))
		out := buf.String()
		Expect(out).To(ContainSubstring("top=resolved-1"))
		Expect(out).To(ContainSubstring("nested.inner=resolved-2"))
	})
})

type countingValuer struct {
	calls *int
}

func (this countingValuer) LogValue() slog.Value {
	*this.calls++
	return slog.StringValue("resolved-" + strconv.Itoa(*this.calls))
}
