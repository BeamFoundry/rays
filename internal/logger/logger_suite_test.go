package logger_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
	Expect(GetContainer().LoadDefaultConfigAt(map[string]any{"handler": "DEFAULT", "level": "DEBUG", "source": "true", "color": "AUTO"}, "rays.logging")).NotTo(HaveOccurred())
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
			GetLogger().Fatal("logging")
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
			GetLogger().FatalContext(context.Background(), "logging")
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
			Expect(getAttrValue(slog.Level(9))).To(Equal("ERROR+1"))
			Expect(getAttrValue(logger.LevelFatal)).To(Equal("FATAL"))
			Expect(getAttrValue(slog.Level(13))).To(Equal("FATAL+1"))

			getTimeValue := func(t time.Time) string {
				attr := slog.Time(slog.TimeKey, t)
				attr = replace([]string{}, attr)
				return attr.Value.String()
			}
			testTime := time.Date(2026, 11, 5, 12, 30, 21, 123000000, time.UTC)
			Expect(getTimeValue(testTime)).To(Equal("2026/11/05 12:30:21.123"))
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
		Expect(handler.Enabled(context.Background(), logger.LevelFatal)).To(BeTrue())
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

var _ = Describe("PatternHandler", func() {
	var (
		buf  *bytes.Buffer
		opts *slog.HandlerOptions
	)

	BeforeEach(func() {
		buf = &bytes.Buffer{}
		opts = &slog.HandlerOptions{
			AddSource: true,
			Level:     logger.LevelTrace,
		}
	})

	It("formats percent escape %%", func() {
		h := logger.NewHandlerWithFormat(buf, opts, "%% %msg(%v)")
		h.SetColor(false)
		err := h.Handle(context.Background(), slog.Record{
			Message: "Hello",
			Level:   slog.LevelInfo,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(buf.String()).To(Equal("% Hello\n"))
	})

	It("formats keys with %k, %qk, %v, %qv", func() {
		h := logger.NewHandlerWithFormat(buf, opts, "%msg(%k=%v, quoted: %qk=%qv)")
		h.SetColor(false)
		err := h.Handle(context.Background(), slog.Record{
			Message: "Hello",
			Level:   slog.LevelInfo,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(buf.String()).To(Equal("msg=Hello, quoted: \"msg\"=\"Hello\"\n"))
	})

	It("formats keys with single letter shortcuts %t, %l, %m, %s", func() {
		h := logger.NewHandlerWithFormat(buf, opts, "%l(%v) %m(%v)")
		h.SetColor(false)
		err := h.Handle(context.Background(), slog.Record{
			Message: "Hello",
			Level:   slog.LevelInfo,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(buf.String()).To(Equal("INFO Hello\n"))
	})

	It("supports Spring padding and truncation modifiers", func() {
		h1 := logger.NewHandlerWithFormat(buf, opts, "[%level(%-5v)] %msg(%.4v)")
		h1.SetColor(false)
		err := h1.Handle(context.Background(), slog.Record{
			Message: "Testing",
			Level:   slog.LevelInfo,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(buf.String()).To(Equal("[INFO ] ting\n"))

		buf.Reset()
		h2 := logger.NewHandlerWithFormat(buf, opts, "[%level(%5v)] %msg(%.-4v)")
		h2.SetColor(false)
		err = h2.Handle(context.Background(), slog.Record{
			Message: "Testing",
			Level:   slog.LevelInfo,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(buf.String()).To(Equal("[ INFO] Test\n"))

		buf.Reset()
		h3 := logger.NewHandlerWithFormat(buf, opts, "%msg(%-6.-4v)")
		h3.SetColor(false)
		err = h3.Handle(context.Background(), slog.Record{
			Message: "Testing",
			Level:   slog.LevelInfo,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(buf.String()).To(Equal("Test  \n"))
	})

	It("respects AddSource option", func() {
		// When AddSource is false, %source and %s are not printed
		optsNoSource := &slog.HandlerOptions{AddSource: false}
		h1 := logger.NewHandlerWithFormat(buf, optsNoSource, "%level(%v) %source(%v )%msg(%v)")
		h1.SetColor(false)
		err := h1.Handle(context.Background(), slog.Record{
			Message: "Hello",
			Level:   slog.LevelInfo,
			PC:      getFramePointer(),
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(buf.String()).To(Equal("INFO Hello\n"))

		buf.Reset()
		// When AddSource is true, %source is printed
		h2 := logger.NewHandlerWithFormat(buf, opts, "%level(%v) %source(%v )%msg(%v)")
		h2.SetColor(false)
		err = h2.Handle(context.Background(), slog.Record{
			Message: "Hello",
			Level:   slog.LevelInfo,
			PC:      getFramePointer(),
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(buf.String()).To(ContainSubstring("INFO "))
		Expect(buf.String()).To(ContainSubstring(".go:"))
		Expect(buf.String()).To(ContainSubstring(" Hello\n"))
	})

	It("formats attributes with %attr and %nl", func() {
		h := logger.NewHandlerWithFormat(buf, opts, "%msg(%v): %attr(%k=%v%nl(, ))")
		h.SetColor(false)
		r := slog.Record{
			Message: "Attrs",
			Level:   slog.LevelInfo,
		}
		r.Add("a", 1, "b", 2, "c", "text")
		err := h.Handle(context.Background(), r)
		Expect(err).NotTo(HaveOccurred())
		Expect(buf.String()).To(Equal("Attrs: a=1, b=2, c=text\n"))

		buf.Reset()
		hQuoted := logger.NewHandlerWithFormat(buf, opts, "%msg(%v): %attr(%qk=%qv%nl( ))")
		hQuoted.SetColor(false)
		err = hQuoted.Handle(context.Background(), r)
		Expect(err).NotTo(HaveOccurred())
		Expect(buf.String()).To(Equal("Attrs: \"a\"=1 \"b\"=2 \"c\"=\"text\"\n"))
	})

	It("formats groups with %grp, %g, and %qg", func() {
		h := logger.NewHandlerWithFormat(buf, opts, "%msg(%v) %grp(%g: [%attr(%k=%v%nl(, ))]%nl( ))")
		h.SetColor(false)
		r := slog.Record{
			Message: "Groups",
			Level:   slog.LevelInfo,
		}
		r.Add(
			slog.Group("metric", slog.Int("cpu", 50), slog.Int("mem", 80)),
			slog.Group("geo", slog.String("city", "Paris")),
		)
		err := h.Handle(context.Background(), r)
		Expect(err).NotTo(HaveOccurred())
		Expect(buf.String()).To(Equal("Groups metric: [cpu=50, mem=80] geo: [city=Paris]\n"))
	})

	It("supports %g and %qg within %attr in a %grp block", func() {
		h := logger.NewHandlerWithFormat(buf, opts, "%msg(%v) %grp(%attr(%g.%k=%v%nl( )))")
		h.SetColor(false)
		r := slog.Record{
			Message: "TestGrp",
			Level:   slog.LevelInfo,
		}
		r.Add(slog.Group("metric", slog.Int("cpu", 50), slog.Int("mem", 80)))
		err := h.Handle(context.Background(), r)
		Expect(err).NotTo(HaveOccurred())
		Expect(buf.String()).To(Equal("TestGrp metric.cpu=50 metric.mem=80\n"))

		buf.Reset()
		hQuoted := logger.NewHandlerWithFormat(buf, opts, "%msg(%v) %grp(%attr(%qg.%qk=%qv%nl( )))")
		hQuoted.SetColor(false)
		err = hQuoted.Handle(context.Background(), r)
		Expect(err).NotTo(HaveOccurred())
		Expect(buf.String()).To(Equal("TestGrp \"metric\".\"cpu\"=50 \"metric\".\"mem\"=80\n"))
	})

	It("formats with TextFormat", func() {
		h := logger.NewHandlerWithFormat(buf, opts, logger.TextFormat)
		h.SetColor(false)
		t := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
		r := slog.Record{
			Time:    t,
			Message: "Hello World",
			Level:   slog.LevelInfo,
		}
		r.Add("app", "rays", "count", 42)
		r.Add(slog.Group("metric", slog.Int("cpu", 50)))
		err := h.Handle(context.Background(), r)
		Expect(err).NotTo(HaveOccurred())
		Expect(buf.String()).To(Equal("time=2026-11-05T12:00:00Z level=INFO msg=\"Hello World\" app=\"rays\" count=42 metric.cpu=50\n"))
	})

	It("formats with JSONFormat", func() {
		h := logger.NewHandlerWithFormat(buf, opts, logger.JSONFormat)
		h.SetColor(false)
		t := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
		r := slog.Record{
			Time:    t,
			Message: "Hello World",
			Level:   slog.LevelInfo,
		}
		r.Add("app", "rays", "count", 42, "flag", true)
		r.Add(slog.Group("metric", slog.Int("cpu", 50), slog.Int("mem", 80)))
		err := h.Handle(context.Background(), r)
		Expect(err).NotTo(HaveOccurred())

		output := buf.String()
		Expect(output).To(HaveSuffix("\n"))

		var js map[string]any
		Expect(json.Unmarshal([]byte(output), &js)).To(Succeed())
		Expect(js["time"]).To(Equal("2026-11-05T12:00:00Z"))
		Expect(js["level"]).To(Equal("INFO"))
		Expect(js["msg"]).To(Equal("Hello World"))
		Expect(js["app"]).To(Equal("rays"))
		Expect(js["count"]).To(BeEquivalentTo(42))
		Expect(js["flag"]).To(BeTrue())

		metric, ok := js["metric"].(map[string]any)
		Expect(ok).To(BeTrue())
		Expect(metric["cpu"]).To(BeEquivalentTo(50))
		Expect(metric["mem"]).To(BeEquivalentTo(80))
	})

	It("supports custom time format with %v and %qv", func() {
		fixedTime := time.Date(2026, 11, 5, 12, 34, 56, 789000000, time.UTC)
		r := slog.Record{
			Time:    fixedTime,
			Message: "Test",
			Level:   slog.LevelInfo,
		}

		// %time(%"RFC3339"v) unquoted
		h1 := logger.NewHandlerWithFormat(buf, opts, `%time(%"RFC3339"v) %msg(%v)`)
		Expect(h1.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(Equal("2026-11-05T12:34:56Z Test\n"))

		// %time(%"RFC3339"qv) quoted
		buf.Reset()
		h2 := logger.NewHandlerWithFormat(buf, opts, `%time(%"RFC3339"qv) %msg(%v)`)
		Expect(h2.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(Equal("\"2026-11-05T12:34:56Z\" Test\n"))

		// Custom Go format layout
		buf.Reset()
		h3 := logger.NewHandlerWithFormat(buf, opts, `%time(%"2006-01-02T15:04:05Z07:00"v) %msg(%v)`)
		Expect(h3.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(Equal("2026-11-05T12:34:56Z Test\n"))
	})

	It("supports predefined time layouts Unix, UnixMilli, UnixMicro, UnixNano", func() {
		fixedTime := time.Unix(1700000000, 123456789).UTC()
		r := slog.Record{
			Time:    fixedTime,
			Message: "Test",
			Level:   slog.LevelInfo,
		}

		// Unix: epoch seconds
		buf.Reset()
		hUnix := logger.NewHandlerWithFormat(buf, opts, `%time(%"Unix"v)`)
		Expect(hUnix.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(Equal("1700000000\n"))

		// UnixMilli: epoch seconds with 3 decimal digits
		buf.Reset()
		hMilli := logger.NewHandlerWithFormat(buf, opts, `%time(%"UnixMilli"v)`)
		Expect(hMilli.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(Equal("1700000000.123\n"))

		// UnixMicro: epoch seconds with 6 decimal digits
		buf.Reset()
		hMicro := logger.NewHandlerWithFormat(buf, opts, `%time(%"UnixMicro"v)`)
		Expect(hMicro.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(Equal("1700000000.123456\n"))

		// UnixNano: epoch seconds with 9 decimal digits
		buf.Reset()
		hNano := logger.NewHandlerWithFormat(buf, opts, `%time(%"UnixNano"v)`)
		Expect(hNano.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(Equal("1700000000.123456789\n"))
	})

	It("does not use ReplaceAttr when custom time format is specified", func() {
		fixedTime := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
		r := slog.Record{
			Time:    fixedTime,
			Message: "Test",
			Level:   slog.LevelInfo,
		}

		customOpts := &slog.HandlerOptions{
			ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
				if a.Key == slog.TimeKey {
					return slog.String(slog.TimeKey, "REPLACED_TIME")
				}
				return a
			},
		}

		// Without custom format: ReplaceAttr IS used
		buf.Reset()
		hDefault := logger.NewHandlerWithFormat(buf, customOpts, `%time(%v)`)
		Expect(hDefault.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(Equal("REPLACED_TIME\n"))

		// With custom format: ReplaceAttr is NOT used
		buf.Reset()
		hCustom := logger.NewHandlerWithFormat(buf, customOpts, `%time(%"RFC3339"v)`)
		Expect(hCustom.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(Equal("2026-11-05T12:00:00Z\n"))
	})

	It("formats with ECSFormat", func() {
		h := logger.NewHandlerWithFormat(buf, opts, logger.ECSFormat)
		fixedTime := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
		r := slog.Record{
			Time:    fixedTime,
			Message: "Application started",
			Level:   slog.LevelInfo,
			PC:      getFramePointer(),
		}
		r.Add("service", "rays", "count", 10)
		r.Add(slog.Group("metric", slog.Int("cpu", 25)))
		Expect(h.Handle(context.Background(), r)).To(Succeed())

		output := buf.String()
		Expect(output).To(HaveSuffix("\n"))

		var js map[string]any
		Expect(json.Unmarshal([]byte(output), &js)).To(Succeed())
		Expect(js["@timestamp"]).To(Equal("2026-11-05T12:00:00Z"))
		Expect(js["log.level"]).To(Equal("INFO"))
		Expect(js["message"]).To(Equal("Application started"))
		Expect(js["ecs.version"]).To(Equal("1.2.0"))
		Expect(js["log.origin"]).NotTo(BeNil())
		Expect(js["service"]).To(Equal("rays"))
		Expect(js["count"]).To(BeEquivalentTo(10))

		metric, ok := js["metric"].(map[string]any)
		Expect(ok).To(BeTrue())
		Expect(metric["cpu"]).To(BeEquivalentTo(25))
	})

	It("formats with GELFFormat", func() {
		h := logger.NewHandlerWithFormat(buf, opts, logger.GELFFormat)
		fixedTime := time.Unix(1700000000, 123000000).UTC()
		r := slog.Record{
			Time:    fixedTime,
			Message: "GELF message",
			Level:   slog.LevelWarn,
			PC:      getFramePointer(),
		}
		r.Add("user_id", "u123", "retry", 3)
		r.Add(slog.Group("details", slog.String("env", "prod")))
		Expect(h.Handle(context.Background(), r)).To(Succeed())

		output := buf.String()
		Expect(output).To(HaveSuffix("\n"))

		var js map[string]any
		Expect(json.Unmarshal([]byte(output), &js)).To(Succeed())
		Expect(js["version"]).To(Equal("1.1"))
		Expect(js["host"]).To(Equal("localhost"))
		Expect(js["timestamp"]).To(BeNumerically("~", 1700000000.123, 0.0001))
		Expect(js["level"]).To(Equal("WARN"))
		Expect(js["short_message"]).To(Equal("GELF message"))
		Expect(js["_source"]).NotTo(BeNil())
		Expect(js["_user_id"]).To(Equal("u123"))
		Expect(js["_retry"]).To(BeEquivalentTo(3))

		details, ok := js["_details"].(map[string]any)
		Expect(ok).To(BeTrue())
		Expect(details["env"]).To(Equal("prod"))
	})

	It("formats with LogstashFormat", func() {
		h := logger.NewHandlerWithFormat(buf, opts, logger.LogstashFormat)
		fixedTime := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
		r := slog.Record{
			Time:    fixedTime,
			Message: "Logstash message",
			Level:   slog.LevelError,
			PC:      getFramePointer(),
		}
		r.Add("app", "demo", "status", "failed")
		r.Add(slog.Group("tags", slog.String("env", "staging")))
		Expect(h.Handle(context.Background(), r)).To(Succeed())

		output := buf.String()
		Expect(output).To(HaveSuffix("\n"))

		var js map[string]any
		Expect(json.Unmarshal([]byte(output), &js)).To(Succeed())
		Expect(js["@version"]).To(Equal("1"))
		Expect(js["@timestamp"]).To(Equal("2026-11-05T12:00:00Z"))
		Expect(js["level"]).To(Equal("ERROR"))
		Expect(js["message"]).To(Equal("Logstash message"))
		Expect(js["source"]).NotTo(BeNil())
		Expect(js["app"]).To(Equal("demo"))
		Expect(js["status"]).To(Equal("failed"))

		tags, ok := js["tags"].(map[string]any)
		Expect(ok).To(BeTrue())
		Expect(tags["env"]).To(Equal("staging"))
	})

	It("formats with DefaultFormat", func() {
		h := logger.NewHandlerWithFormat(buf, opts, logger.DefaultFormat)
		h.SetColor(true)
		fixedTime := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
		r := slog.Record{
			Time:    fixedTime,
			Message: "Default message",
			Level:   slog.LevelInfo,
			PC:      getFramePointer(),
		}
		r.Add("app", "demo")
		r.Add(slog.Group("metric", slog.Int("cpu", 50)))
		Expect(h.Handle(context.Background(), r)).To(Succeed())

		output := buf.String()
		Expect(output).To(ContainSubstring("2026-11-05T12:00:00Z "))
		Expect(output).To(ContainSubstring("\033[32mINFO \033[0m "))
		Expect(output).To(ContainSubstring("\033[35m"))
		Expect(output).To(ContainSubstring("Default message"))
		Expect(output).To(ContainSubstring("\033[2m app=demo\033[0m"))
		Expect(output).To(ContainSubstring("\033[2m metric.cpu=50\033[0m"))
	})

	It("formats colors with %clr(EXPRESSION){COLOR}", func() {
		h := logger.NewHandlerWithFormat(buf, opts, "%clr(%level(%-5v)){cyan} %msg(%v)")
		// When color is enabled
		h.SetColor(true)
		Expect(h.IsColor()).To(BeTrue())
		err := h.Handle(context.Background(), slog.Record{
			Message: "Hello",
			Level:   slog.LevelInfo,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(buf.String()).To(Equal("\033[36mINFO \033[0m Hello\n"))

		// When color is disabled
		buf.Reset()
		h.SetColor(false)
		Expect(h.IsColor()).To(BeFalse())
		err = h.Handle(context.Background(), slog.Record{
			Message: "Hello",
			Level:   slog.LevelInfo,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(buf.String()).To(Equal("INFO  Hello\n"))
	})

	It("does not emit ANSI color codes when expression evaluates to empty", func() {
		optsNoSource := &slog.HandlerOptions{AddSource: false}
		h := logger.NewHandlerWithFormat(buf, optsNoSource, "%clr(%source(%v)){cyan}%msg(%v)")
		h.SetColor(true)
		err := h.Handle(context.Background(), slog.Record{
			Message: "Hello",
			Level:   slog.LevelInfo,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(buf.String()).To(Equal("Hello\n"))
	})

	It("formats colors with level-based color when {COLOR} is omitted", func() {
		h := logger.NewHandlerWithFormat(buf, opts, "%clr(%level(%-5v)) %msg(%v)")
		h.SetColor(true)

		// INFO -> green (\033[32m)
		err := h.Handle(context.Background(), slog.Record{Message: "InfoMsg", Level: slog.LevelInfo})
		Expect(err).NotTo(HaveOccurred())
		Expect(buf.String()).To(Equal("\033[32mINFO \033[0m InfoMsg\n"))

		// DEBUG -> cyan (\033[36m)
		buf.Reset()
		err = h.Handle(context.Background(), slog.Record{Message: "DebugMsg", Level: logger.LevelDebug})
		Expect(err).NotTo(HaveOccurred())
		Expect(buf.String()).To(Equal("\033[36mDEBUG\033[0m DebugMsg\n"))

		// TRACE -> no color (\033[0m)
		buf.Reset()
		err = h.Handle(context.Background(), slog.Record{Message: "TraceMsg", Level: logger.LevelTrace})
		Expect(err).NotTo(HaveOccurred())
		Expect(buf.String()).To(Equal("\033[0mTRACE\033[0m TraceMsg\n"))

		// WARN -> yellow (\033[33m)
		buf.Reset()
		err = h.Handle(context.Background(), slog.Record{Message: "WarnMsg", Level: slog.LevelWarn})
		Expect(err).NotTo(HaveOccurred())
		Expect(buf.String()).To(Equal("\033[33mWARN \033[0m WarnMsg\n"))

		// ERROR -> red (\033[31m)
		buf.Reset()
		err = h.Handle(context.Background(), slog.Record{Message: "ErrorMsg", Level: slog.LevelError})
		Expect(err).NotTo(HaveOccurred())
		Expect(buf.String()).To(Equal("\033[31mERROR\033[0m ErrorMsg\n"))

		// FATAL -> red (\033[31m)
		buf.Reset()
		err = h.Handle(context.Background(), slog.Record{Message: "FatalMsg", Level: logger.LevelFatal})
		Expect(err).NotTo(HaveOccurred())
		Expect(buf.String()).To(Equal("\033[31mFATAL\033[0m FatalMsg\n"))
	})

	It("supports ANSI modes normal, bold, faint, italic, underline, blinky", func() {
		h := logger.NewHandlerWithFormat(buf, opts, "%clr(%msg(%v)){normal}")
		h.SetColor(true)
		r := slog.Record{Message: "Test", Level: slog.LevelInfo}

		// {normal} -> \033[0m
		Expect(h.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(Equal("\033[0mTest\033[0m\n"))

		// {white} -> \033[37m
		buf.Reset()
		h.SetFormat("%clr(%msg(%v)){white}")
		Expect(h.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(Equal("\033[37mTest\033[0m\n"))

		// {white,normal} -> \033[0;37m
		buf.Reset()
		h.SetFormat("%clr(%msg(%v)){white,normal}")
		Expect(h.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(Equal("\033[0;37mTest\033[0m\n"))

		// {bold} -> \033[1m
		buf.Reset()
		h.SetFormat("%clr(%msg(%v)){bold}")
		Expect(h.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(Equal("\033[1mTest\033[0m\n"))

		// {white,bold} -> \033[1;37m
		buf.Reset()
		h.SetFormat("%clr(%msg(%v)){white,bold}")
		Expect(h.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(Equal("\033[1;37mTest\033[0m\n"))

		// {faint} -> \033[2m
		buf.Reset()
		h.SetFormat("%clr(%msg(%v)){faint}")
		Expect(h.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(Equal("\033[2mTest\033[0m\n"))

		// {white,faint} -> \033[2;37m
		buf.Reset()
		h.SetFormat("%clr(%msg(%v)){white,faint}")
		Expect(h.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(Equal("\033[2;37mTest\033[0m\n"))

		// {italic} -> \033[3m
		buf.Reset()
		h.SetFormat("%clr(%msg(%v)){italic}")
		Expect(h.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(Equal("\033[3mTest\033[0m\n"))

		// {white,italic} -> \033[3;37m
		buf.Reset()
		h.SetFormat("%clr(%msg(%v)){white,italic}")
		Expect(h.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(Equal("\033[3;37mTest\033[0m\n"))

		// {underline} -> \033[4m
		buf.Reset()
		h.SetFormat("%clr(%msg(%v)){underline}")
		Expect(h.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(Equal("\033[4mTest\033[0m\n"))

		// {white,underline} -> \033[4;37m
		buf.Reset()
		h.SetFormat("%clr(%msg(%v)){white,underline}")
		Expect(h.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(Equal("\033[4;37mTest\033[0m\n"))

		// {blinky} -> \033[5m
		buf.Reset()
		h.SetFormat("%clr(%msg(%v)){blinky}")
		Expect(h.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(Equal("\033[5mTest\033[0m\n"))

		// {white,blinky} -> \033[5;37m
		buf.Reset()
		h.SetFormat("%clr(%msg(%v)){white,blinky}")
		Expect(h.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(Equal("\033[5;37mTest\033[0m\n"))
	})

	It("combines mode with configured level color", func() {
		h := logger.NewHandlerWithFormat(buf, opts, "%clr(%level(%v)){faint}")
		h.SetColor(true)

		// WARN + {faint} -> \033[2;33m (yellow, faint)
		rWarn := slog.Record{Level: slog.LevelWarn}
		Expect(h.Handle(context.Background(), rWarn)).To(Succeed())
		Expect(buf.String()).To(Equal("\033[2;33mWARN\033[0m\n"))

		// INFO + {bold} -> \033[1;32m (green, bold)
		buf.Reset()
		h.SetFormat("%clr(%level(%v)){bold}")
		rInfo := slog.Record{Level: slog.LevelInfo}
		Expect(h.Handle(context.Background(), rInfo)).To(Succeed())
		Expect(buf.String()).To(Equal("\033[1;32mINFO\033[0m\n"))

		// DEBUG + {italic} -> \033[3;36m (cyan, italic)
		buf.Reset()
		h.SetFormat("%clr(%level(%v)){italic}")
		rDebug := slog.Record{Level: logger.LevelDebug}
		Expect(h.Handle(context.Background(), rDebug)).To(Succeed())
		Expect(buf.String()).To(Equal("\033[3;36mDEBUG\033[0m\n"))

		// ERROR + {underline} -> \033[4;31m (red, underline)
		buf.Reset()
		h.SetFormat("%clr(%level(%v)){underline}")
		rError := slog.Record{Level: slog.LevelError}
		Expect(h.Handle(context.Background(), rError)).To(Succeed())
		Expect(buf.String()).To(Equal("\033[4;31mERROR\033[0m\n"))

		// FATAL + {blinky} -> \033[5;31m (red, blinky)
		buf.Reset()
		h.SetFormat("%clr(%level(%v)){blinky}")
		rFatal := slog.Record{Level: logger.LevelFatal}
		Expect(h.Handle(context.Background(), rFatal)).To(Succeed())
		Expect(buf.String()).To(Equal("\033[5;31mFATAL\033[0m\n"))

		// TRACE + {faint} -> \033[2m (no color, faint)
		buf.Reset()
		h.SetFormat("%clr(%level(%v)){faint}")
		rTrace := slog.Record{Level: logger.LevelTrace}
		Expect(h.Handle(context.Background(), rTrace)).To(Succeed())
		Expect(buf.String()).To(Equal("\033[2mTRACE\033[0m\n"))
	})

	It("supports combining multiple modes and explicit color", func() {
		h := logger.NewHandlerWithFormat(buf, opts, "%clr(%msg(%v)){yellow,faint}")
		h.SetColor(true)
		r := slog.Record{Message: "Multi", Level: slog.LevelInfo}

		// {yellow,faint} -> \033[2;33m
		Expect(h.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(Equal("\033[2;33mMulti\033[0m\n"))

		// {bold,italic,red} -> \033[1;3;31m
		buf.Reset()
		h.SetFormat("%clr(%msg(%v)){bold,italic,red}")
		Expect(h.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(Equal("\033[1;3;31mMulti\033[0m\n"))

		// non-level with no color and no mode -> \033[0m
		buf.Reset()
		h.SetFormat("%clr(%msg(%v))")
		Expect(h.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(Equal("\033[0mMulti\033[0m\n"))
	})

	It("supports SetFormat and GetFormat on Handler and Logger", func() {
		h := logger.NewHandlerWithFormat(buf, opts, "%msg(%v)")
		Expect(h.GetFormat()).To(Equal("%msg(%v)"))
		h.SetFormat("%l(%v) %m(%v)")
		Expect(h.GetFormat()).To(Equal("%l(%v) %m(%v)"))

		lg := logger.GetLogger()
		lg.SetFormat("%l(%v) - %m(%v)")
		Expect(lg.GetFormat()).To(Equal("%l(%v) - %m(%v)"))
		lg.SetColor(true)
		lg.Reset()
	})

	It("supports EnableColor and forced SetColor", func() {
		// buf is bytes.Buffer, which is not a terminal
		h := logger.NewHandlerWithFormat(buf, opts, "%clr(%level(%v)){cyan} %msg(%v)")
		h.EnableColor()
		Expect(h.IsColor()).To(BeFalse())

		// SetColor(true) forces color even on non-terminal
		h.SetColor(true)
		Expect(h.IsColor()).To(BeTrue())

		err := h.Handle(context.Background(), slog.Record{Message: "Forced", Level: slog.LevelInfo})
		Expect(err).NotTo(HaveOccurred())
		Expect(buf.String()).To(Equal("\033[36mINFO\033[0m Forced\n"))

		// On Logger with non-terminal output (buf)
		lg := logger.GetLogger()
		lg.SetOutput(buf)
		lg.EnableColor()
		Expect(lg.IsColor()).To(BeFalse())
		lg.SetColor(true)
		Expect(lg.IsColor()).To(BeTrue())

		// On Logger with GinkgoWriter (terminal)
		lg.Reset()
		lg.SetOutput(GinkgoWriter)
		lg.EnableColor()
		Expect(lg.IsColor()).To(BeTrue())
	})
})

var _ = Describe("ReduceFunctionName", func() {
	doms := map[string]string{
		"github.com/BeamFoundry/rays/pkg/internal/container.(*Instance).GetFoo": "g/B/rays/p/i/container.Instance",
		"github.com/BeamFoundry/rays/pkg/internal.(*Instance).GetBar":           "g/B/rays/p/internal.Instance",
		"github.com/BeamFoundry/rays/internal.(*Instance).GetBar":               "g/B/rays/internal.Instance",
		"github.com/BeamFoundry/rays.Instance.GetBar":                           "g/B/rays.Instance",
		"github.com/BeamFoundry/rays/pkg/internal.init.func2.1":                 "g/B/rays/p/internal.init",
		"log/slog/internal.logger.log":                                          "log/slog/internal.logger",
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

type dummySetColorOnlyHandler struct {
	color bool
}

func (d *dummySetColorOnlyHandler) Enabled(context.Context, slog.Level) bool  { return true }
func (d *dummySetColorOnlyHandler) Handle(context.Context, slog.Record) error { return nil }
func (d *dummySetColorOnlyHandler) WithAttrs([]slog.Attr) slog.Handler        { return d }
func (d *dummySetColorOnlyHandler) WithGroup(string) slog.Handler             { return d }
func (d *dummySetColorOnlyHandler) SetColor(c bool)                           { d.color = c }

var _ = Describe("Logger and Pattern Coverage Edge Cases", func() {
	var buf *bytes.Buffer
	testTime := time.Date(2026, 11, 7, 12, 0, 0, 123456789, time.UTC)

	BeforeEach(func() {
		buf = &bytes.Buffer{}
	})

	It("covers empty format strings in NewHandlerWithFormat and SetFormat", func() {
		h := logger.NewHandlerWithFormat(buf, nil, "")
		Expect(h.GetFormat()).To(Equal(logger.DefaultFormat))
		h.SetFormat("")
		Expect(h.GetFormat()).To(Equal(logger.DefaultFormat))
	})

	It("covers Logger fallback when handler does not implement interfaces", func() {
		lg := logger.GetLogger().Clone()
		defer lg.Reset()

		lg.SetHandler(func(w io.Writer, opts *slog.HandlerOptions) slog.Handler {
			return slog.NewTextHandler(w, opts)
		})
		lg.SetFormat("custom_format")
		Expect(lg.GetFormat()).To(Equal("custom_format"))

		lg.SetColor(true)
		Expect(lg.IsColor()).To(BeTrue())

		lg.EnableColor()
		Expect(lg.IsColor()).To(BeTrue()) // output is GinkgoWriter (terminal)

		// Handler that only implements SetColor(bool) without EnableColor()
		dummy := &dummySetColorOnlyHandler{}
		lg.SetHandler(func(io.Writer, *slog.HandlerOptions) slog.Handler {
			return dummy
		})
		lg.EnableColor()
		Expect(dummy.color).To(BeTrue())

		// Clone with format and color preserved
		lg.SetColor(false)
		cloned := lg.Clone()
		Expect(cloned.IsColor()).To(BeFalse())
	})

	It("covers all predefined time layouts in formatTime", func() {
		layouts := []string{
			"unix", "unixmilli", "unixmicro", "unixnano",
			"rfc3339", "rfc3339nano", "datetime", "dateonly", "timeonly",
			"ansic", "unixdate", "rubydate", "rfc822", "rfc822z", "rfc850",
			"rfc1123", "rfc1123z", "kitchen", "stamp", "stampmilli", "stampmicro", "stampnano",
			"2006-01-02 15:04:05",
		}
		for _, layout := range layouts {
			h := logger.NewHandlerWithFormat(buf, nil, fmt.Sprintf("%%time(%%%qv)\n", layout))
			r := slog.Record{Time: testTime, Level: slog.LevelInfo, Message: "test"}
			Expect(h.Handle(context.Background(), r)).To(Succeed())
			Expect(buf.Len()).To(BeNumerically(">", 0))
			buf.Reset()
		}
	})

	It("covers time formatting on slog.Attr values and non-time values", func() {
		// slog.KindTime attr formatted with time layout (%v and %qv)
		h := logger.NewHandlerWithFormat(buf, nil, "%attr(%k=%\x22RFC3339\x22v %k=%\x22RFC3339\x22qv)\n")
		r := slog.Record{Time: testTime, Level: slog.LevelInfo, Message: "test"}
		r.Add(slog.Time("created", testTime))
		Expect(h.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(ContainSubstring("created=2026-11-07T12:00:00Z"))
		Expect(buf.String()).To(ContainSubstring("created=\"2026-11-07T12:00:00Z\""))

		// Non-time attr with time layout fallback
		buf.Reset()
		r2 := slog.Record{Time: testTime, Level: slog.LevelInfo, Message: "test"}
		r2.Add(slog.String("str_key", "hello"))
		Expect(h.Handle(context.Background(), r2)).To(Succeed())
		Expect(buf.String()).To(ContainSubstring("str_key=hello"))
	})

	It("covers KeyNode and ReplaceAttr edge cases", func() {
		// Zero time in record
		h := logger.NewHandlerWithFormat(buf, nil, "%time(%v)%msg(%v)\n")
		rZeroTime := slog.Record{Message: "msg_only"}
		Expect(h.Handle(context.Background(), rZeroTime)).To(Succeed())
		Expect(buf.String()).To(Equal("msg_only\n"))

		// Source with PC=0
		buf.Reset()
		hSource := logger.NewHandlerWithFormat(buf, &slog.HandlerOptions{AddSource: true}, "%source(%v)%msg(%v)\n")
		rNoPC := slog.Record{Message: "no_pc", PC: 0}
		Expect(hSource.Handle(context.Background(), rNoPC)).To(Succeed())
		Expect(buf.String()).To(Equal("no_pc\n"))

		// ReplaceAttr drops key (returns empty Attr)
		buf.Reset()
		optsDrop := &slog.HandlerOptions{
			ReplaceAttr: func(_ []string, _ slog.Attr) slog.Attr {
				return slog.Attr{}
			},
		}
		hDrop := logger.NewHandlerWithFormat(buf, optsDrop, "%time(%v)%level(%v)%msg(%v)\n")
		rDrop := slog.Record{Time: testTime, Level: slog.LevelInfo, Message: "dropped"}
		Expect(hDrop.Handle(context.Background(), rDrop)).To(Succeed())
		Expect(buf.String()).To(Equal("\n"))
	})

	It("covers formatCurrentValue branches for string and custom types in ReplaceAttr", func() {
		optsReplaced := &slog.HandlerOptions{
			ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
				switch a.Key {
				case slog.TimeKey:
					return slog.String(slog.TimeKey, "custom_time_str")
				case slog.LevelKey:
					return slog.String(slog.LevelKey, "custom_level_str")
				case slog.SourceKey:
					return slog.String(slog.SourceKey, "custom_source_str")
				default:
					return a
				}
			},
		}
		// Test %v and %qv with string replacements
		h := logger.NewHandlerWithFormat(buf, optsReplaced, "%time(%qv) %level(%qv) %msg(%qv)\n")
		r := slog.Record{Time: testTime, Level: slog.LevelInfo, Message: "test_msg"}
		Expect(h.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(Equal("\"custom_time_str\" \"custom_level_str\" \"test_msg\"\n"))

		// Non-string, non-standard types in ReplaceAttr (e.g. ints)
		buf.Reset()
		optsReplacedInts := &slog.HandlerOptions{
			AddSource: true,
			ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
				switch a.Key {
				case slog.TimeKey:
					return slog.Int(slog.TimeKey, 12345)
				case slog.LevelKey:
					return slog.Int(slog.LevelKey, 999)
				case slog.SourceKey:
					return slog.Int(slog.SourceKey, 888)
				default:
					return a
				}
			},
		}
		hInts := logger.NewHandlerWithFormat(buf, optsReplacedInts, "%time(%v) %level(%v) %source(%v) %time(%qv) %level(%qv) %source(%qv)\n")
		// Provide valid PC for source
		pc := make([]uintptr, 1)
		runtime.Callers(1, pc)
		Expect(hInts.Handle(context.Background(), slog.Record{Time: testTime, Level: slog.LevelInfo, PC: pc[0]})).To(Succeed())
		Expect(buf.String()).To(ContainSubstring("12345 999 888 \"12345\" \"999\" \"888\""))
	})

	It("covers appendFormattedValue for nil, nil pointers, sources, and custom structs", func() {
		h := logger.NewHandlerWithFormat(buf, nil, "%attr( %k=%v %k=%qv)\n")
		r := slog.Record{Time: testTime, Level: slog.LevelInfo, Message: "attrs"}

		var nilPtr *int
		r.Add(
			slog.Any("nil_val", nil),
			slog.Any("nil_ptr", nilPtr),
			slog.Any("src", &slog.Source{Function: "main.foo", File: "bar.go", Line: 99}),
			slog.Any("obj", struct{ X int }{X: 42}),
		)
		Expect(h.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(ContainSubstring("nil_val=<nil> nil_val=null"))
		Expect(buf.String()).To(ContainSubstring("nil_ptr=<nil> nil_ptr=null"))
		Expect(buf.String()).To(ContainSubstring("src=bar.go:99"))
		Expect(buf.String()).To(ContainSubstring("obj={X:42}"))
	})

	It("covers ColorNode empty output, invalid spec, and commas", func() {
		// Empty output inside %clr
		hEmpty := logger.NewHandlerWithFormat(buf, nil, "prefix%clr(%msg(%v)){cyan}suffix\n")
		hEmpty.SetColor(true)
		rEmpty := slog.Record{Message: ""}
		Expect(hEmpty.Handle(context.Background(), rEmpty)).To(Succeed())
		Expect(buf.String()).To(Equal("prefixsuffix\n"))

		// Invalid color/mode spec
		buf.Reset()
		hInvalid := logger.NewHandlerWithFormat(buf, nil, "%clr(%msg(%v)){invalid_style_name}\n")
		hInvalid.SetColor(true)
		rMsg := slog.Record{Message: "hello"}
		Expect(hInvalid.Handle(context.Background(), rMsg)).To(Succeed())
		Expect(buf.String()).To(Equal("hello\n"))

		// Spec with extra commas
		buf.Reset()
		hCommas := logger.NewHandlerWithFormat(buf, nil, "%clr(%msg(%v)){,bold,,cyan,}\n")
		hCommas.SetColor(true)
		Expect(hCommas.Handle(context.Background(), rMsg)).To(Succeed())
		Expect(buf.String()).To(Equal("\033[1;36mhello\033[0m\n"))
	})

	It("covers lone percent signs and unmatched parentheses in pattern parser", func() {
		hLone := logger.NewHandlerWithFormat(buf, nil, "100% pure %msg(%v)%\n")
		r := slog.Record{Message: "gold"}
		Expect(hLone.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(Equal("100% pure gold%\n"))

		// Unmatched parenthesis
		buf.Reset()
		hUnmatched := logger.NewHandlerWithFormat(buf, nil, "%time(%v\n")
		Expect(hUnmatched.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(Equal("%time(<nil>\n"))
	})

	It("covers isTerminal environment variables (NO_COLOR, TERM=dumb)", func() {
		h := logger.NewHandlerWithFormat(buf, nil, "%clr(%level(%v)) %msg(%v)\n")

		// NO_COLOR env var disables color
		Expect(os.Setenv("NO_COLOR", "1")).To(Succeed())
		h.EnableColor()
		Expect(h.IsColor()).To(BeFalse())
		Expect(os.Unsetenv("NO_COLOR")).To(Succeed())

		// TERM=dumb disables color
		Expect(os.Setenv("TERM", "dumb")).To(Succeed())
		oldColorTerm := os.Getenv("COLORTERM")
		Expect(os.Unsetenv("COLORTERM")).To(Succeed())
		h.EnableColor()
		Expect(h.IsColor()).To(BeFalse())
		Expect(os.Unsetenv("TERM")).To(Succeed())
		if oldColorTerm != "" {
			Expect(os.Setenv("COLORTERM", oldColorTerm)).To(Succeed())
		}

		// When output is os.Stderr (which implements Fd)
		hStderr := logger.NewHandlerWithFormat(os.Stderr, nil, "%msg(%v)")
		hStderr.EnableColor()
		_ = hStderr.IsColor()
	})

	It("covers additional AST traversal and formatting branches", func() {
		// Clone with format on Logger
		lg := logger.GetLogger().Clone()
		defer lg.Reset()
		lg.SetFormat("%time(%v) %msg(%v)")
		clonedWithFormat := lg.Clone()
		Expect(clonedWithFormat.GetFormat()).To(Equal("%time(%v) %msg(%v)"))

		// Nested %clr and AST node traversal in hasLevelNode and hasTimeFormat
		hNested := logger.NewHandlerWithFormat(buf, nil, "%clr(%clr(%level(%v))) %time(%time(%\"RFC3339\"v)) %time(%clr(%\"RFC3339\"v)) %time(%attr(%\"RFC3339\"v)) %time(%grp(%\"RFC3339\"v))\n")
		hNested.SetColor(true)
		Expect(hNested.Handle(context.Background(), slog.Record{Time: testTime, Level: slog.LevelInfo})).To(Succeed())

		// Source with PC that has no file (e.g. PC=1)
		buf.Reset()
		hSourceInvalidPC := logger.NewHandlerWithFormat(buf, &slog.HandlerOptions{AddSource: true}, "%source(%v)")
		Expect(hSourceInvalidPC.Handle(context.Background(), slog.Record{PC: 1})).To(Succeed())
		Expect(buf.String()).To(Equal("\n"))

		// Attribute with KindTime without custom layout (both %v and %qv)
		buf.Reset()
		hTimeAttr := logger.NewHandlerWithFormat(buf, nil, "%attr( %k=%v %k=%qv)\n")
		rTimeAttr := slog.Record{Message: "time_test"}
		rTimeAttr.Add(slog.Time("raw_time", testTime))
		Expect(hTimeAttr.Handle(context.Background(), rTimeAttr)).To(Succeed())
		Expect(buf.String()).To(ContainSubstring("raw_time=" + testTime.String()))

		// Panic recovery in appendFormattedValue
		buf.Reset()
		rPanic := slog.Record{Message: "panic_test"}
		rPanic.Add(slog.Any("panicker", &PanicTextMarshaler{}))
		Expect(hTimeAttr.Handle(context.Background(), rPanic)).To(Succeed())
		Expect(buf.String()).To(ContainSubstring("!PANIC"))

		// SourceKey with string value and %qv
		buf.Reset()
		optsSourceStr := &slog.HandlerOptions{
			AddSource: true,
			ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
				if a.Key == slog.SourceKey {
					return slog.String(slog.SourceKey, "my/custom/file.go:123")
				}
				return a
			},
		}
		hSourceStr := logger.NewHandlerWithFormat(buf, optsSourceStr, "%source(%qv)\n")
		pc := make([]uintptr, 1)
		runtime.Callers(1, pc)
		Expect(hSourceStr.Handle(context.Background(), slog.Record{PC: pc[0]})).To(Succeed())
		Expect(buf.String()).To(Equal("\"my/custom/file.go:123\"\n"))
	})
})
