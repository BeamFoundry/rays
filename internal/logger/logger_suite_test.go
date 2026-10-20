package logger_test

import (
	"context"
	"io"
	"log/slog"
	"os"
	"runtime"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/BeamFoundry/rays/internal/logger"
	. "github.com/BeamFoundry/rays/pkg/core"
	. "github.com/BeamFoundry/rays/pkg/lang"
	. "github.com/BeamFoundry/rays/pkg/testing"
)

func TestLogging(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Internal/Logger Suite")
}

func getFramePointer() uintptr {
	pc := make([]uintptr, 32)
	runtime.Callers(3, pc)
	return pc[0]
}

func createRecord() slog.Record {
	return slog.Record{
		Time:    time.Now(),
		Message: "Testing",
		Level:   logger.LevelInfo,
		PC:      getFramePointer(),
	}
}

// Pre-1.25
func createSource() *slog.Source {
	pc := make([]uintptr, 32)
	runtime.Callers(3, pc)
	frames := runtime.CallersFrames(pc[0:1])
	if frame, ok := frames.Next(); ok {
		return &slog.Source{
			Function: frame.Function,
			File:     frame.File,
			Line:     frame.Line,
		}
	}
	return &slog.Source{}
}

var _ = AfterSuite(func() {
	GetLogger().SetOutput(GinkgoWriter)
	Expect(GetContainer().LoadDefaultConfigAt(map[string]any{"handler": "DEFAULT", "level": "DEBUG", "source": "true"}, "rays.logging")).NotTo(HaveOccurred())
})

var _ = AfterEach(func() {
	GetLogger().Reset()
	GetLogger().SetOutput(GinkgoWriter)
	GetLogger().SetLogLevel(LevelTrace)
	GetLogger().AddSource()
})

var _ = Describe("Logger", func() {
	Context("WithCustomHandler", func() {
		It("can set a custom Handler", func() {
			GetLogger().SetHandler(
				func(_ io.Writer, _ *slog.HandlerOptions) slog.Handler {
					return slog.NewTextHandler(os.Stderr, nil)
				},
			)
		})
	})

	Context("With DefaultHandler", func() {
		It("can log", func() {
			GetLogger().Trace("logging")
			GetLogger().Debug("logging")
			GetLogger().Info("logging")
			GetLogger().Warn("logging")
			GetLogger().Error("logging")
			GetLogger().Printf("logging")
			GetLogger().Log(context.Background(), slog.LevelInfo, "logging")
			GetLogger().LogAttrs(context.Background(), slog.LevelInfo, "logging")
			Expect(GetLogger().Handler()).NotTo(BeNil())
		})
		It("can log with Context", func() {
			GetLogger().TraceContext(context.Background(), "logging")
			GetLogger().DebugContext(context.Background(), "logging")
			GetLogger().InfoContext(context.Background(), "logging")
			GetLogger().WarnContext(context.Background(), "logging")
			GetLogger().ErrorContext(context.Background(), "logging")
		})
		It("can set default log level", func() {
			GetLogger().SetLogLevel(logger.LevelTrace)
		})
		It("can replace attributes", func() {
			replace := GetLogger().GetHandlerOptions().ReplaceAttr

			getAttrValue := func(level slog.Level) string {
				attr := slog.Any(slog.LevelKey, level)
				attr = replace([]string{}, attr)
				return attr.Value.String()
			}
			Expect(getAttrValue(logger.LevelTrace)).To(Equal("TRACE"))
			Expect(getAttrValue(slog.Level(-7))).To(Equal("TRACE+1"))
			Expect(getAttrValue(logger.LevelDebug)).To(Equal("DEBUG"))
			Expect(getAttrValue(logger.LevelInfo)).To(Equal("INFO"))
			Expect(getAttrValue(logger.LevelWarn)).To(Equal("WARN"))
			Expect(getAttrValue(logger.LevelError)).To(Equal("ERROR"))
		})
		It("can use With", func() {
			GetLogger().With("test", "test").Trace("Test logging")
		})
		It("can use WithGroup", func() {
			GetLogger().WithGroup("test").Trace("Test logging")
		})
		It("can be cloned", func() {
			Expect(GetLogger().Clone()).NotTo(BeNil())
		})
		It("can log Error with StackTrace", func() {
			err := NewError("TestError", "TestError")
			err.LogErrorWithStackTrace()
			err.LogWithStackTrace(slog.LevelError)
		})
	})
})

var _ = Describe("Buffer", func() {
	It("can create a buffer and use it (string)", func() {
		buf := logger.NewBuffer()
		s, err := buf.WriteString("custom Buffer")
		Expect(s).To(BeNumerically(">", 0))
		Expect(err).NotTo(HaveOccurred())
		Expect(buf.Len()).To(BeNumerically(">", 0))
		Expect(buf.String()).To(Equal("custom Buffer"))
		buf.Reset()
	})
	It("can create a buffer and use it ([]byte)", func() {
		buf := logger.NewBuffer()
		b := []byte("custom Buffer")
		s, err := buf.Write(b)
		Expect(s).To(BeNumerically(">", 0))
		Expect(err).NotTo(HaveOccurred())
		Expect(buf.Len()).To(BeNumerically(">", 0))
		Expect(buf.String()).To(Equal("custom Buffer"))
		buf.Reset()
	})
})

type PanicTextMarshaler struct{}

func (this *PanicTextMarshaler) MarshalText() (_ []byte, _ error) {
	panic("err")
}

func (this *PanicTextMarshaler) String() string {
	return "PanicTextMarshaler"
}

type ErrTextMarshaler struct{}

func (this *ErrTextMarshaler) MarshalText() (_ []byte, _ error) {
	return nil, NewError("ErrTextMarshaler", "Err")
}

func (this *ErrTextMarshaler) String() string {
	return "ErrTextMarshaler"
}

var _ = Describe("Handler", func() {
	It("can be created without HandlerOptions", func() {
		handler := logger.NewHandler(GinkgoWriter, nil)
		Expect(handler).NotTo(BeNil())
		err := handler.Handle(context.Background(), createRecord())
		Expect(err).NotTo(HaveOccurred())
	})
	It("can check if a level is enabled", func() {
		GetLogger().SetLogLevel(logger.LevelDebug)
		handler := logger.NewHandler(GinkgoWriter, GetLogger().GetHandlerOptions())
		Expect(handler.Enabled(context.Background(), logger.LevelTrace)).To(BeFalse())
		Expect(handler.Enabled(context.Background(), logger.LevelWarn)).To(BeTrue())
	})
	It("can log value", func() {
		type Lvl int
		opts := &slog.HandlerOptions{
			Level: logger.LevelInfo,
			ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
				switch a.Key {
				case slog.LevelKey:
					level, ok := a.Value.Any().(slog.Level)
					Expect(ok).To(BeTrue())
					a.Value = slog.AnyValue(Lvl(level))
				case slog.TimeKey:
					t, ok := a.Value.Any().(time.Time)
					Expect(ok).To(BeTrue())
					a.Value = slog.AnyValue(t.Format(time.RFC3339))
				default:
				}
				return a
			},
		}
		handler := logger.NewHandler(GinkgoWriter, opts)
		err := handler.Handle(context.Background(), createRecord())
		Expect(err).NotTo(HaveOccurred())
	})
	It("can log attributes", func() {
		handler := logger.NewHandler(GinkgoWriter, GetLogger().GetHandlerOptions())
		handler.WithAttrs([]slog.Attr{})
		handler.WithGroup("")

		type Bool struct {
			Enum[Bool]
			TRUE,
			FALSE Enum[Bool]
		}

		r := createRecord()
		r.AddAttrs(
			slog.Attr{},
			slog.Int64("int64", 1024),
			slog.Uint64("uint64", 1024),
			slog.Float64("float64", 1024.0),
			slog.Bool("bool", true),
			slog.Duration("duration", time.Second*60),
			slog.Time("time", time.Now()),
			slog.Any("enum", Bool{}.New().TRUE),
			slog.Any("panic", &PanicTextMarshaler{}),
			slog.Any("err", &ErrTextMarshaler{}),
			slog.Any("nil", (*PanicTextMarshaler)(nil)),
			slog.Any("src", createSource()),
			slog.Group(
				"group",
				slog.String("key", "value"),
			),
		)
		err := handler.Handle(context.Background(), r)
		Expect(err).NotTo(HaveOccurred())
		err = handler.WithGroup("testing").WithAttrs([]slog.Attr{slog.String("key", "value")}).Handle(context.Background(), r)
		Expect(err).NotTo(HaveOccurred())
	})
})

var _ = Describe("ReduceFunctionName", func() {
	doms := map[string]string{
		"github.com/BeamFoundry/rays/pkg/internal/container.(*Instance).GetFoo": "g/B/rays/p/i/container Instance",
		"github.com/BeamFoundry/rays/pkg/internal.(*Instance).GetBar":           "g/B/rays/p/internal Instance",
		"github.com/BeamFoundry/rays/internal.(*Instance).GetBar":               "g/B/rays/internal Instance",
		"github.com/BeamFoundry/rays.Instance.GetBar":                           "g/B/rays Instance",
		"github.com/BeamFoundry/rays/pkg/internal.init.func2.1":                 "g/B/rays/p/internal init",
		"log/slog/internal.logger.log":                                          "log/slog/internal logger",
	}
	It("can reduce a module name", func() {
		GetLogger().SetLogLevel(logger.LevelDebug)
		GetLogger().SetOutput(GinkgoWriter)
		GetLogger().Debug("testing ReduceFunctionName")
		for range 2 {
			for k, v := range doms {
				Expect(logger.ReduceFunctionName(k)).To(Equal(v))
			}
		}
	})
})
