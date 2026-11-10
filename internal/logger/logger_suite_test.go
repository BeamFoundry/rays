package logger_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"os"
	"runtime"
	"slices"
	"strings"
	"sync"
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
		fixedTime := time.Date(2026, 11, 5, 12, 0, 0, 123456789, time.UTC)
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
		Expect(js["@timestamp"]).To(Equal("2026-11-05T12:00:00.123456789Z"))
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
		r.Add("user_id", "u123", "retry", 3, "flag", true)
		r.Add(slog.Group("details", slog.String("env", "prod")))
		Expect(h.Handle(context.Background(), r)).To(Succeed())

		output := buf.String()
		Expect(output).To(HaveSuffix("\n"))

		var js map[string]any
		Expect(json.Unmarshal([]byte(output), &js)).To(Succeed())
		Expect(js["version"]).To(Equal("1.1"))
		hostname := logger.GetEmitterIdentity()
		Expect(js["host"]).To(Equal(hostname))
		Expect(js["timestamp"]).To(BeNumerically("~", 1700000000.123, 0.0001))
		Expect(js["level"]).To(BeEquivalentTo(4))
		Expect(js["short_message"]).To(Equal("GELF message"))
		Expect(js["_source"]).NotTo(BeNil())
		Expect(js["_user_id"]).To(Equal("u123"))
		Expect(js["_retry"]).To(BeEquivalentTo(3))
		Expect(js["_flag"]).To(BeEquivalentTo(1))
		Expect(js["_details.env"]).To(Equal("prod"))
	})

	It("formats with LogstashFormat", func() {
		h := logger.NewHandlerWithFormat(buf, opts, logger.LogstashFormat)
		fixedTime := time.Date(2026, 11, 5, 12, 0, 0, 123456789, time.UTC)
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
		Expect(js["@timestamp"]).To(Equal("2026-11-05T12:00:00.123456789Z"))
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

		// On Logger with terminal output, isolated from ambient environment
		setupColorEnv()

		termOut := &fakeTerminalWriter{Buffer: bytes.NewBuffer(nil)}
		lg.Reset()
		lg.SetOutput(termOut)
		lg.EnableColor()
		Expect(lg.IsColor()).To(BeTrue())

		// Disabled with NO_COLOR=1
		Expect(os.Setenv("NO_COLOR", "1")).To(Succeed())
		lg.EnableColor()
		Expect(lg.IsColor()).To(BeFalse())

		// Disabled with TERM=dumb
		Expect(os.Unsetenv("NO_COLOR")).To(Succeed())
		Expect(os.Setenv("TERM", "dumb")).To(Succeed())
		Expect(os.Unsetenv("COLORTERM")).To(Succeed())
		lg.EnableColor()
		Expect(lg.IsColor()).To(BeFalse())
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

type fakeTerminalWriter struct {
	*bytes.Buffer
}

func (f *fakeTerminalWriter) IsTerminal() bool { return true }

type dummySetColorOnlyHandler struct {
	color bool
}

func (d *dummySetColorOnlyHandler) Enabled(context.Context, slog.Level) bool  { return true }
func (d *dummySetColorOnlyHandler) Handle(context.Context, slog.Record) error { return nil }
func (d *dummySetColorOnlyHandler) WithAttrs([]slog.Attr) slog.Handler        { return d }
func (d *dummySetColorOnlyHandler) WithGroup(string) slog.Handler             { return d }
func (d *dummySetColorOnlyHandler) SetColor(c bool)                           { d.color = c }

func setupColorEnv() {
	origNoColor, hasNoColor := os.LookupEnv("NO_COLOR")
	origTerm, hasTerm := os.LookupEnv("TERM")
	origColorTerm, hasColorTerm := os.LookupEnv("COLORTERM")
	origArgs := os.Args

	DeferCleanup(func() {
		if hasNoColor {
			Expect(os.Setenv("NO_COLOR", origNoColor)).To(Succeed())
		} else {
			Expect(os.Unsetenv("NO_COLOR")).To(Succeed())
		}
		if hasTerm {
			Expect(os.Setenv("TERM", origTerm)).To(Succeed())
		} else {
			Expect(os.Unsetenv("TERM")).To(Succeed())
		}
		if hasColorTerm {
			Expect(os.Setenv("COLORTERM", origColorTerm)).To(Succeed())
		} else {
			Expect(os.Unsetenv("COLORTERM")).To(Succeed())
		}
		os.Args = origArgs
	})

	var filteredArgs []string
	for _, arg := range os.Args {
		trimmed := strings.TrimLeft(arg, "-")
		if trimmed != "no-color" && trimmed != "ginkgo.no-color" &&
			!strings.HasPrefix(trimmed, "no-color=") && !strings.HasPrefix(trimmed, "ginkgo.no-color=") {
			filteredArgs = append(filteredArgs, arg)
		}
	}
	os.Args = filteredArgs

	Expect(os.Unsetenv("NO_COLOR")).To(Succeed())
	Expect(os.Setenv("TERM", "xterm-256color")).To(Succeed())
	Expect(os.Setenv("COLORTERM", "truecolor")).To(Succeed())
}

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

		termBuf := &fakeTerminalWriter{Buffer: bytes.NewBuffer(nil)}
		lg.SetOutput(termBuf)
		setupColorEnv()

		lg.EnableColor()
		Expect(lg.IsColor()).To(BeTrue())

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
		setupColorEnv()

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

	It("supports default color code 39 in custom pattern rendering", func() {
		h := logger.NewHandlerWithFormat(buf, nil, "%clr(%msg(%v)){default}\n")
		h.SetColor(true)
		r := slog.Record{Message: "default color test"}
		Expect(h.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(Equal("\033[39mdefault color test\033[0m\n"))
	})

	It("shares writer mutex across cloned handlers for serialized writes", func() {
		base := logger.NewHandler(buf, nil)
		clone1 := base.WithAttrs([]slog.Attr{slog.String("c", "1")})
		clone2 := base.WithGroup("grp")

		var wg sync.WaitGroup
		for range 20 {
			wg.Add(2)
			go func() {
				defer GinkgoRecover()
				defer wg.Done()
				Expect(clone1.Handle(context.Background(), slog.Record{Message: "msg1"})).To(Succeed())
			}()
			go func() {
				defer GinkgoRecover()
				defer wg.Done()
				Expect(clone2.Handle(context.Background(), slog.Record{Message: "msg2"})).To(Succeed())
			}()
		}
		wg.Wait()
		Expect(buf.Len()).To(BeNumerically(">", 0))
	})

	It("synchronizes SetFormat, SetColor, and Handle concurrently without race", func() {
		h := logger.NewHandler(buf, nil)
		var wg sync.WaitGroup
		for range 20 {
			wg.Add(3)
			go func() {
				defer wg.Done()
				h.SetFormat("%time(%v) %msg(%v)")
			}()
			go func() {
				defer wg.Done()
				h.SetColor(true)
			}()
			go func() {
				defer GinkgoRecover()
				defer wg.Done()
				Expect(h.Handle(context.Background(), slog.Record{Message: "concurrent"})).To(Succeed())
			}()
		}
		wg.Wait()
	})

	It("retains anonymous group members and empty-key non-group attributes in collectAttrRecursive", func() {
		h := logger.NewHandlerWithFormat(buf, nil, logger.JSONFormat)
		h.SetColor(false)
		r := slog.Record{Message: "test"}
		// Anonymous group with request_id
		r.Add(slog.Group("", slog.String("request_id", "req-456")))
		// Non-group attribute with empty key
		r.Add(slog.String("", "empty_key_val"))
		// Zero attribute (should be skipped)
		r.Add(slog.Attr{})
		Expect(h.Handle(context.Background(), r)).To(Succeed())

		var js map[string]any
		Expect(json.Unmarshal(buf.Bytes(), &js)).To(Succeed())
		Expect(js["request_id"]).To(Equal("req-456"))
		Expect(js[""]).To(Equal("empty_key_val"))
	})

	It("preserves group ancestry as separate path segments and keeps literal group keys distinct", func() {
		h := logger.NewHandlerWithFormat(buf, nil, logger.JSONFormat)
		h.SetColor(false)
		r := slog.Record{Message: "test"}
		// Nested groups: request -> user
		r.Add(slog.Group("request", slog.Group("user", slog.String("name", "alice"))))
		// Literal group key with dot
		r.Add(slog.Group("request.user", slog.String("role", "admin")))
		Expect(h.Handle(context.Background(), r)).To(Succeed())

		var js map[string]any
		Expect(json.Unmarshal(buf.Bytes(), &js)).To(Succeed())

		// Nested object built from segments
		req, ok := js["request"].(map[string]any)
		Expect(ok).To(BeTrue())
		user, ok := req["user"].(map[string]any)
		Expect(ok).To(BeTrue())
		Expect(user["name"]).To(Equal("alice"))

		// Literal group key distinct and not merged
		lit, ok := js["request.user"].(map[string]any)
		Expect(ok).To(BeTrue())
		Expect(lit["role"]).To(Equal("admin"))
	})

	It("emits valid JSON without doubled or trailing commas when level and message are omitted in all built-in JSON formats", func() {
		omitOpts := &slog.HandlerOptions{
			ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
				if a.Key == slog.LevelKey || a.Key == slog.MessageKey {
					return slog.Attr{} // omit level and message
				}
				return a
			},
		}

		formats := []string{
			logger.JSONFormat,
			logger.ECSFormat,
			logger.GELFFormat,
			logger.LogstashFormat,
		}

		for _, fmtStr := range formats {
			buf.Reset()
			h := logger.NewHandlerWithFormat(buf, omitOpts, fmtStr)
			h.SetColor(false)
			r := slog.Record{
				Time:    testTime,
				Message: "Omitted",
				Level:   slog.LevelInfo,
			}
			r.Add("app", "test", "num", 100)
			Expect(h.Handle(context.Background(), r)).To(Succeed())

			out := buf.String()
			Expect(out).NotTo(ContainSubstring(",,"))
			Expect(out).NotTo(ContainSubstring("{,"))
			Expect(out).NotTo(ContainSubstring(",}"))

			var js map[string]any
			Expect(json.Unmarshal([]byte(out), &js)).To(Succeed())
			Expect(js["level"]).To(BeNil())
			Expect(js["log.level"]).To(BeNil())
			Expect(js["message"]).To(BeNil())
			Expect(js["short_message"]).To(BeNil())
			if fmtStr == logger.GELFFormat {
				Expect(js["_app"]).To(Equal("test"))
			} else {
				Expect(js["app"]).To(Equal("test"))
			}
		}
	})

	It("emits structured source fields in ECSFormat", func() {
		h := logger.NewHandlerWithFormat(buf, &slog.HandlerOptions{AddSource: true}, logger.ECSFormat)
		h.SetColor(false)
		pc := make([]uintptr, 1)
		runtime.Callers(1, pc)
		r := slog.Record{
			Time:    testTime,
			Message: "ecs test",
			Level:   slog.LevelInfo,
			PC:      pc[0],
		}
		Expect(h.Handle(context.Background(), r)).To(Succeed())

		var js map[string]any
		Expect(json.Unmarshal(buf.Bytes(), &js)).To(Succeed())
		origin, ok := js["log.origin"].(map[string]any)
		Expect(ok).To(BeTrue())
		fileInfo, ok := origin["file"].(map[string]any)
		Expect(ok).To(BeTrue())
		Expect(fileInfo["name"]).NotTo(BeEmpty())
		Expect(fileInfo["line"]).To(BeNumerically(">", 0))
		Expect(origin["function"]).NotTo(BeEmpty())
	})

	It("recognizes quoted format layouts with unmatched parentheses in delimiter scanning", func() {
		// Time format containing unmatched parenthesis inside quotes
		fmtWithParen := `%time(%"(2006-01-02"v) %msg(%v)`
		h := logger.NewHandlerWithFormat(buf, nil, fmtWithParen)
		r := slog.Record{
			Time:    time.Date(2026, 11, 7, 0, 0, 0, 0, time.UTC),
			Message: "paren_msg",
		}
		Expect(h.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(Equal("(2026-11-07 paren_msg\n"))

		// Quoted format layout after width and precision modifiers
		buf.Reset()
		hMod := logger.NewHandlerWithFormat(buf, nil, `[%time(%-15.10"(2006-01-02"v)] %msg(%v)`)
		Expect(hMod.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(Equal("[2026-11-07     ] paren_msg\n"))

		// Apostrophe inside %msg(...) does not prevent locating the closing parenthesis
		buf.Reset()
		hApos := logger.NewHandlerWithFormat(buf, nil, "%msg(it's %v)")
		Expect(hApos.Handle(context.Background(), slog.Record{Message: "alive"})).To(Succeed())
		Expect(buf.String()).To(Equal("it's alive\n"))

		// Literal double quotes inside %msg(...)
		buf.Reset()
		hQuote := logger.NewHandlerWithFormat(buf, nil, `%msg("quoted" %v)`)
		Expect(hQuote.Handle(context.Background(), slog.Record{Message: "msg"})).To(Succeed())
		Expect(buf.String()).To(Equal(`"quoted" msg` + "\n"))
	})

	It("formats negative fractional times correctly in unixmilli, unix_micro, and unixnano", func() {
		// -500ms from epoch: instant is -0.500s
		tNeg := time.Unix(-1, 500_000_000).UTC()
		r := slog.Record{Time: tNeg, Message: "test"}

		buf.Reset()
		hMilli := logger.NewHandlerWithFormat(buf, nil, `%time(%"unixmilli"v)`)
		Expect(hMilli.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(Equal("-0.500\n"))

		buf.Reset()
		hMicro := logger.NewHandlerWithFormat(buf, nil, `%time(%"unix_micro"v)`)
		Expect(hMicro.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(Equal("-0.500000\n"))

		buf.Reset()
		hNano := logger.NewHandlerWithFormat(buf, nil, `%time(%"unixnano"v)`)
		Expect(hNano.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(Equal("-0.500000000\n"))

		// -1.250s
		tNeg2 := time.Unix(-2, 750_000_000).UTC()
		r2 := slog.Record{Time: tNeg2, Message: "test"}
		buf.Reset()
		Expect(hMilli.Handle(context.Background(), r2)).To(Succeed())
		Expect(buf.String()).To(Equal("-1.250\n"))
	})

	It("handles encoding.TextMarshaler errors with a diagnostic fallback in text and JSON", func() {
		bad := &ErrorTextMarshaler{}

		// Text format: fallback without quotes
		buf.Reset()
		hText := logger.NewHandlerWithFormat(buf, nil, "%attr(%k=%v)\n")
		rText := slog.Record{Message: "test"}
		rText.Add(slog.Any("bad_field", bad))
		Expect(hText.Handle(context.Background(), rText)).To(Succeed())
		Expect(buf.String()).To(ContainSubstring("bad_field=!ERROR: marshaler failure"))

		// JSON format: fallback properly quoted
		buf.Reset()
		hJSON := logger.NewHandlerWithFormat(buf, nil, logger.JSONFormat)
		hJSON.SetColor(false)
		rJSON := slog.Record{Message: "test"}
		rJSON.Add(slog.Any("bad_field", bad))
		Expect(hJSON.Handle(context.Background(), rJSON)).To(Succeed())
		var js map[string]any
		Expect(json.Unmarshal(buf.Bytes(), &js)).To(Succeed())
		Expect(js["bad_field"]).To(Equal("!ERROR: marshaler failure"))
	})

	It("suppresses timestamp when ReplaceAttr returns zero slog.TimeKey in custom layouts, ECS, GELF, and Logstash", func() {
		omitTimeOpts := &slog.HandlerOptions{
			ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
				if a.Key == slog.TimeKey {
					return slog.Attr{}
				}
				return a
			},
		}

		// Custom layout
		buf.Reset()
		hCustom := logger.NewHandlerWithFormat(buf, omitTimeOpts, `%time(%"RFC3339"v)%msg(%v)`)
		Expect(hCustom.Handle(context.Background(), slog.Record{Time: testTime, Message: "hello"})).To(Succeed())
		Expect(buf.String()).To(Equal("hello\n"))

		// ECSFormat
		buf.Reset()
		hECS := logger.NewHandlerWithFormat(buf, omitTimeOpts, logger.ECSFormat)
		Expect(hECS.Handle(context.Background(), slog.Record{Time: testTime, Message: "hello"})).To(Succeed())
		var jsECS map[string]any
		Expect(json.Unmarshal(buf.Bytes(), &jsECS)).To(Succeed())
		Expect(jsECS["@timestamp"]).To(BeNil())

		// GELFFormat
		buf.Reset()
		hGELF := logger.NewHandlerWithFormat(buf, omitTimeOpts, logger.GELFFormat)
		Expect(hGELF.Handle(context.Background(), slog.Record{Time: testTime, Message: "hello"})).To(Succeed())
		var jsGELF map[string]any
		Expect(json.Unmarshal(buf.Bytes(), &jsGELF)).To(Succeed())
		Expect(jsGELF["timestamp"]).To(BeNil())

		// LogstashFormat
		buf.Reset()
		hLogstash := logger.NewHandlerWithFormat(buf, omitTimeOpts, logger.LogstashFormat)
		Expect(hLogstash.Handle(context.Background(), slog.Record{Time: testTime, Message: "hello"})).To(Succeed())
		var jsLogstash map[string]any
		Expect(json.Unmarshal(buf.Bytes(), &jsLogstash)).To(Succeed())
		Expect(jsLogstash["@timestamp"]).To(BeNil())
	})

	It("encodes non-finite floats as quoted error strings in JSON", func() {
		h := logger.NewHandlerWithFormat(buf, nil, logger.JSONFormat)
		h.SetColor(false)
		r := slog.Record{Message: "floats"}
		r.Add(
			slog.Float64("nan", math.NaN()),
			slog.Float64("pos_inf", math.Inf(1)),
			slog.Float64("neg_inf", math.Inf(-1)),
		)
		Expect(h.Handle(context.Background(), r)).To(Succeed())

		var js map[string]any
		Expect(json.Unmarshal(buf.Bytes(), &js)).To(Succeed())
		Expect(js["nan"]).To(Equal("!FLOAT:NaN"))
		Expect(js["pos_inf"]).To(Equal("!FLOAT:+Inf"))
		Expect(js["neg_inf"]).To(Equal("!FLOAT:-Inf"))
	})

	It("encodes maps, json.Marshaler, and control characters as structured JSON in JSON output path", func() {
		h := logger.NewHandlerWithFormat(buf, nil, logger.JSONFormat)
		h.SetColor(false)
		r := slog.Record{Message: "structured"}
		r.Add(
			slog.Any("map_field", map[string]int{"alpha": 1, "beta": 2}),
			slog.Any("custom_json", &CustomJSONType{Name: "item1"}),
			slog.String("ctrl", "line\u0001text"),
		)
		Expect(h.Handle(context.Background(), r)).To(Succeed())

		var js map[string]any
		Expect(json.Unmarshal(buf.Bytes(), &js)).To(Succeed())
		m, ok := js["map_field"].(map[string]any)
		Expect(ok).To(BeTrue())
		Expect(m["alpha"]).To(BeEquivalentTo(1))
		Expect(m["beta"]).To(BeEquivalentTo(2))

		cj, ok := js["custom_json"].(map[string]any)
		Expect(ok).To(BeTrue())
		Expect(cj["custom"]).To(Equal("item1"))
		Expect(js["ctrl"]).To(Equal("line\u0001text"))
	})

	It("quotes keys and timestamps containing spaces or equals signs in TextFormat logfmt", func() {
		h := logger.NewHandlerWithFormat(buf, nil, logger.TextFormat)
		h.SetColor(false)
		tSpace := time.Date(2026, 11, 7, 12, 30, 0, 0, time.UTC)
		r := slog.Record{
			Time:    tSpace,
			Message: "logfmt test",
			Level:   slog.LevelInfo,
		}
		r.Add(
			slog.String("key with space", "val1"),
			slog.String("key=with=equals", "val2"),
		)
		Expect(h.Handle(context.Background(), r)).To(Succeed())

		out := buf.String()
		Expect(out).To(ContainSubstring(`"key with space"="val1"`))
		Expect(out).To(ContainSubstring(`"key=with=equals"="val2"`))
	})

	It("tracks explicit empty SetFormat independently and reapplies it on clone", func() {
		lg := logger.GetLogger().Clone()
		defer lg.Reset()

		lg.SetFormat("")
		Expect(lg.GetFormat()).To(Equal(logger.DefaultFormat))

		cloned := lg.Clone()
		Expect(cloned.GetFormat()).To(Equal(logger.DefaultFormat))
	})

	It("allows configuring emitter identity for GELF host", func() {
		logger.SetEmitterIdentity("custom-service-host")
		defer logger.SetEmitterIdentity("")

		h := logger.NewHandlerWithFormat(buf, nil, logger.GELFFormat)
		r := slog.Record{Message: "host test"}
		Expect(h.Handle(context.Background(), r)).To(Succeed())

		var js map[string]any
		Expect(json.Unmarshal(buf.Bytes(), &js)).To(Succeed())
		Expect(js["host"]).To(Equal("custom-service-host"))
	})

	It("evaluates IsColor on handler when color setting is unset", func() {
		h := logger.NewHandler(buf, nil)
		_ = h.IsColor()
	})

	It("merges multiple attributes in the same group in GELF format", func() {
		h := logger.NewHandlerWithFormat(buf, nil, logger.GELFFormat)
		r := slog.Record{Message: "gelf group test"}
		r.Add(slog.Group("my_group", slog.String("a", "1"), slog.String("b", "2"), slog.Bool("flag", false)))
		Expect(h.Handle(context.Background(), r)).To(Succeed())
		var js map[string]any
		Expect(json.Unmarshal(buf.Bytes(), &js)).To(Succeed())
		Expect(js["_my_group.a"]).To(Equal("1"))
		Expect(js["_my_group.b"]).To(Equal("2"))
		Expect(js["_my_group.flag"]).To(Equal(float64(0)))
	})

	It("handles edge cases in cleanJsonCommas", func() {
		bShort := logger.NewBuffer()
		_, err := bShort.WriteString("x")
		Expect(err).NotTo(HaveOccurred())
		logger.CleanJsonCommas(bShort)
		Expect(bShort.String()).To(Equal("x"))

		bWS := logger.NewBuffer()
		_, err = bWS.WriteString("{\n  \"a\": 1, \n}")
		Expect(err).NotTo(HaveOccurred())
		logger.CleanJsonCommas(bWS)
		Expect(bWS.String()).To(Equal("{\n  \"a\": 1 \n}"))
	})

	It("falls back to localhost when hostname lookup fails", func() {
		logger.SetHostnameFunc(func() (string, error) { return "", errors.New("hostname failed") })
		defer logger.SetHostnameFunc(os.Hostname)
		Expect(logger.GetEmitterIdentity()).To(Equal("localhost"))
	})

	It("handles unknown KeyNode key safely", func() {
		node := logger.NewUnknownKeyNode()
		b := logger.NewBuffer()
		node.Render(logger.NewRenderContext(), b)
		Expect(b.Len()).To(Equal(0))
	})

	It("renders grouped attributes in topAttrs when pattern lacks group directive", func() {
		h := logger.NewHandlerWithFormat(buf, nil, "%attr(%k=%v)\n")
		r := slog.Record{Message: "no grp directive"}
		r.Add(slog.Group("env", slog.String("mode", "prod")))
		Expect(h.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(Equal("mode=prod\n"))

		// In GELF mode without group directive
		buf.Reset()
		hGELFNoGrp := logger.NewHandlerWithFormat(buf, nil, `{"version":"1.1",%attr(%_qg=%v%nl( ))}`+"\n")
		rGELF := slog.Record{Message: "no grp directive gelf"}
		rGELF.Add(
			slog.Group("env", slog.String("mode", "prod")),
			slog.String("env.prod", "literal_val"),
			slog.Group("env", slog.Group("prod", slog.String("setting", "nested_val"))),
		)
		Expect(hGELFNoGrp.Handle(context.Background(), rGELF)).To(Succeed())
		Expect(buf.String()).To(ContainSubstring(`"_env_mode"=prod`))
		Expect(buf.String()).To(ContainSubstring(`"_env_x2eprod"=literal_val`))
		Expect(buf.String()).To(ContainSubstring(`"_env.prod_setting"=nested_val`))
	})

	It("renders deep nested groups and multiple child groups in JSON format", func() {
		h := logger.NewHandlerWithFormat(buf, nil, logger.JSONFormat)
		h.SetColor(false)
		r := slog.Record{Message: "deep nested"}
		r.Add(slog.Group("top",
			slog.Group("c1",
				slog.String("x", "1"),
				slog.String("y", "2"),
				slog.Group("sub", slog.String("z", "3")),
			),
			slog.Group("c2", slog.String("w", "4")),
		))
		Expect(h.Handle(context.Background(), r)).To(Succeed())
		var js map[string]any
		Expect(json.Unmarshal(buf.Bytes(), &js)).To(Succeed())
		top, ok := js["top"].(map[string]any)
		Expect(ok).To(BeTrue())
		c1, ok := top["c1"].(map[string]any)
		Expect(ok).To(BeTrue())
		Expect(c1["x"]).To(Equal("1"))
		Expect(c1["y"]).To(Equal("2"))
		sub, ok := c1["sub"].(map[string]any)
		Expect(ok).To(BeTrue())
		Expect(sub["z"]).To(Equal("3"))
		c2, ok := top["c2"].(map[string]any)
		Expect(ok).To(BeTrue())
		Expect(c2["w"]).To(Equal("4"))
	})

	It("supports prefixes in %qk and %q_gk format specifiers", func() {
		buf.Reset()
		hQk := logger.NewHandlerWithFormat(buf, nil, `[%attr(%"pfx_"qk=%v)]`)
		r := slog.Record{Message: "prefix test"}
		r.Add(slog.String("k1", "v1"))
		Expect(hQk.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(Equal(`["pfx_k1"=v1]` + "\n"))

		buf.Reset()
		hQgk := logger.NewHandlerWithFormat(buf, nil, `[%attr(%"custom"q_gk=%v)]`)
		Expect(hQgk.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(Equal(`["custom_k1"=v1]` + "\n"))
	})

	It("handles unknown ValueFormatNode target safely", func() {
		node := logger.NewUnknownValueNode()
		b := logger.NewBuffer()
		node.Render(logger.NewRenderContext(), b)
		Expect(b.Len()).To(Equal(0))
	})

	It("uses record level in GELF syslog severity when level attribute is replaced by string", func() {
		opts := &slog.HandlerOptions{
			ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
				if a.Key == slog.LevelKey {
					return slog.String("level", "WARN")
				}
				return a
			},
		}
		h := logger.NewHandlerWithFormat(buf, opts, logger.GELFFormat)
		r := slog.Record{Level: slog.LevelWarn, Message: "warn test"}
		Expect(h.Handle(context.Background(), r)).To(Succeed())
		var js map[string]any
		Expect(json.Unmarshal(buf.Bytes(), &js)).To(Succeed())
		Expect(js["level"]).To(Equal(float64(4)))
	})

	It("formats negative seconds with zero nanoseconds in formatUnixFractional", func() {
		tNegZero := time.Unix(-5, 0).UTC()
		Expect(logger.FormatUnixFractional(tNegZero, 3)).To(Equal("-5.000"))
	})

	It("maps all severity levels in syslogSeverity", func() {
		Expect(logger.SyslogSeverity(logger.LevelFatal)).To(Equal(2))
		Expect(logger.SyslogSeverity(slog.LevelError)).To(Equal(3))
		Expect(logger.SyslogSeverity(slog.LevelWarn)).To(Equal(4))
		Expect(logger.SyslogSeverity(slog.LevelInfo)).To(Equal(6))
		Expect(logger.SyslogSeverity(logger.LevelTrace)).To(Equal(7))
		Expect(logger.SyslogSeverity(slog.Level(-10))).To(Equal(7))
	})

	It("returns false for hasLevelNode(nil)", func() {
		Expect(logger.HasLevelNode(nil)).To(BeFalse())
	})

	It("validates isQuotedLayoutStart conditions", func() {
		Expect(logger.IsQuotedLayoutStart("foo", 0)).To(BeFalse())
		Expect(logger.IsQuotedLayoutStart("foo", 1)).To(BeFalse())
	})

	It("handles edge cases in findMatchingParen", func() {
		_, ok := logger.FindMatchingParen("abc", 5)
		Expect(ok).To(BeFalse())
		_, ok2 := logger.FindMatchingParen("abc", 0)
		Expect(ok2).To(BeFalse())

		// Escaped characters inside quoted layout
		idx, ok3 := logger.FindMatchingParen(`%time(%"2006\"01\"02"v)`, 5)
		Expect(ok3).To(BeTrue())
		Expect(idx).To(Equal(22))
	})

	It("falls back to strconv.Quote in formatJsonString if JSON marshaling fails", func() {
		logger.SetJsonMarshalFunc(func(any) ([]byte, error) { return nil, errors.New("marshal error") })
		defer logger.SetJsonMarshalFunc(json.Marshal)
		Expect(logger.FormatJsonString("test")).To(Equal(`"test"`))
	})

	It("handles nil pointer panics gracefully in appendFormattedValue", func() {
		b := logger.NewBuffer()
		var src *slog.Source
		logger.AppendFormattedValue(b, slog.AnyValue(src), true)
		Expect(b.String()).To(Equal("null"))

		b.Reset()
		logger.AppendFormattedValue(b, slog.AnyValue(src), false)
		Expect(b.String()).To(Equal("<nil>"))
	})

	It("handles slog.GroupValue in appendFormattedValue default case", func() {
		b := logger.NewBuffer()
		logger.AppendFormattedValue(b, slog.GroupValue(), false)
		Expect(b.Len()).To(Equal(0))
	})

	It("checks isTerminal with --no-color=true flag in os.Args", func() {
		origArgs := os.Args
		os.Args = append(slices.Clone(origArgs), "--no-color=true")
		defer func() { os.Args = origArgs }()
		Expect(logger.IsTerminal(buf)).To(BeFalse())
	})

	It("covers checkTerminalVT", func() {
		if runtime.GOOS == "windows" {
			Expect(logger.CheckTerminalVT(0)).To(BeFalse())
		} else {
			Expect(logger.CheckTerminalVT(0)).To(BeTrue())
		}
	})

	It("handles non-nil panics gracefully in appendFormattedValue", func() {
		b := logger.NewBuffer()
		logger.AppendFormattedValue(b, slog.AnyValue(NonNilPanicker{}), true)
		Expect(b.String()).To(ContainSubstring("!PANIC: boom"))
	})

	It("supports %av, %ak, and %ag automatic quoting", func() {
		h := logger.NewHandlerWithFormat(buf, nil, "%attr(%ak=%av%nl( ))")
		r := slog.Record{Time: testTime, Message: "test"}
		r.Add(
			slog.String("simple_key", "simple_val"),
			slog.String("key with space", "val with space"),
			slog.String("key=equals", "val=equals"),
			slog.String("empty_val", ""),
			slog.Int("num", 42),
			slog.Bool("flag", true),
		)
		Expect(h.Handle(context.Background(), r)).To(Succeed())
		out := buf.String()
		Expect(out).To(ContainSubstring("simple_key=simple_val"))
		Expect(out).To(ContainSubstring(`"key with space"="val with space"`))
		Expect(out).To(ContainSubstring(`"key=equals"="val=equals"`))
		Expect(out).To(ContainSubstring(`empty_val=""`))
		Expect(out).To(ContainSubstring("num=42"))
		Expect(out).To(ContainSubstring("flag=true"))
	})

	It("keeps %time(%v) unquoted and quotes %time(%av) when layout has spaces", func() {
		optsTime := &slog.HandlerOptions{
			ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
				if a.Key == slog.TimeKey {
					if t, ok := a.Value.Any().(time.Time); ok {
						a.Value = slog.StringValue(t.Format("2006/01/02 15:04:05.000"))
					}
				}
				return a
			},
		}

		buf.Reset()
		hRaw := logger.NewHandlerWithFormat(buf, optsTime, "%time(%v)%msg(%v)\n")
		r := slog.Record{Time: time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC), Message: "msg"}
		Expect(hRaw.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(Equal("2026/11/08 12:00:00.000msg\n"))

		buf.Reset()
		hAuto := logger.NewHandlerWithFormat(buf, optsTime, "%time(%av)%msg(%v)\n")
		Expect(hAuto.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(Equal("\"2026/11/08 12:00:00.000\"msg\n"))

		buf.Reset()
		hRFC := logger.NewHandlerWithFormat(buf, nil, `%time(%"RFC3339"av)%msg(%v)`+"\n")
		Expect(hRFC.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(Equal("2026-11-08T12:00:00Zmsg\n"))

		buf.Reset()
		hCustom := logger.NewHandlerWithFormat(buf, nil, `%time(%"2006/01/02 15:04:05"av)%msg(%v)`+"\n")
		Expect(hCustom.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(Equal("\"2026/11/08 12:00:00\"msg\n"))

		buf.Reset()
		hQV := logger.NewHandlerWithFormat(buf, nil, "%time(%qv)\n")
		Expect(hQV.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(Equal(`"` + r.Time.Format(time.RFC3339) + "\"\n"))
	})

	It("keeps raw %k and %g unquoted even with spaces", func() {
		buf.Reset()
		h := logger.NewHandlerWithFormat(buf, nil, "%grp(%g:%attr(%k=%v))")
		hGroup := h.WithGroup("my group")
		r := slog.Record{Time: testTime, Message: "test"}
		r.Add(slog.String("my key", "val"))
		Expect(hGroup.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(Equal("my group:my key=val\n"))
	})

	It("formats %ag, %a_k, and %a_gk with auto-quoting", func() {
		buf.Reset()
		hAutoGroup := logger.NewHandlerWithFormat(buf, nil, "%grp(%ag:%attr(%ak=%v))")
		hG := hAutoGroup.WithGroup("spaced group")
		r := slog.Record{Time: testTime, Message: "test"}
		r.Add(slog.String("key", "val"))
		Expect(hG.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(Equal("\"spaced group\":key=val\n"))

		buf.Reset()
		hG2 := hAutoGroup.WithGroup("simplegroup")
		Expect(hG2.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(Equal("simplegroup:key=val\n"))

		buf.Reset()
		hEmptyGroup := logger.NewHandlerWithFormat(buf, nil, "%ag%attr(%ak=%v)")
		rEmpty := slog.Record{Time: testTime, Message: "test"}
		rEmpty.Add(slog.String("key", "val"))
		Expect(hEmptyGroup.Handle(context.Background(), rEmpty)).To(Succeed())
		Expect(buf.String()).To(Equal("key=val\n"))

		buf.Reset()
		hPfxAK := logger.NewHandlerWithFormat(buf, nil, `%attr(%"pfx."ak=%v%nl( ))`)
		rPfx := slog.Record{Time: testTime, Message: "test"}
		rPfx.Add(slog.String("plain", "1"), slog.String("has space", "2"))
		Expect(hPfxAK.Handle(context.Background(), rPfx)).To(Succeed())
		Expect(buf.String()).To(ContainSubstring("pfx.plain=1"))
		Expect(buf.String()).To(ContainSubstring(`"pfx.has space"=2`))

		buf.Reset()
		hAK := logger.NewHandlerWithFormat(buf, nil, "%attr(%a_k=%v%nl( ))")
		rAK := slog.Record{Time: testTime, Message: "test"}
		rAK.Add(slog.String("plain", "1"), slog.String("has space", "2"))
		Expect(hAK.Handle(context.Background(), rAK)).To(Succeed())
		Expect(buf.String()).To(ContainSubstring("_plain=1"))
		Expect(buf.String()).To(ContainSubstring(`"_has space"=2`))

		buf.Reset()
		hAGK := logger.NewHandlerWithFormat(buf, nil, `%grp(%attr(%"g_"a_gk=%v%nl( )))`)
		hG3 := hAGK.WithGroup("grp")
		rAGK := slog.Record{Time: testTime, Message: "test"}
		rAGK.Add(slog.String("item", "1"), slog.String("spaced item", "2"))
		Expect(hG3.Handle(context.Background(), rAGK)).To(Succeed())
		Expect(buf.String()).To(ContainSubstring("g__grp_item=1"))
		Expect(buf.String()).To(ContainSubstring(`"g__grp_spaced item"=2`))

		buf.Reset()
		hNoGrpAGK := logger.NewHandlerWithFormat(buf, nil, "%attr(%a_gk=%v%nl( ))")
		rAGK2 := slog.Record{Time: testTime, Message: "test"}
		rAGK2.Add(slog.String("item", "1"), slog.String("spaced item", "2"))
		Expect(hNoGrpAGK.Handle(context.Background(), rAGK2)).To(Succeed())
		Expect(buf.String()).To(ContainSubstring("_item=1"))
		Expect(buf.String()).To(ContainSubstring(`"_spaced item"=2`))
	})

	It("formats grouped keys with %.ag, %.qg, and %.g and prefixed keys with %_k and %_g", func() {
		// %.ag in group
		buf.Reset()
		hAG := logger.NewHandlerWithFormat(buf, nil, `%grp(%attr(%.ag=%v%nl( )))`)
		hG := hAG.WithGroup("my group")
		r := slog.Record{Time: testTime, Message: "test"}
		r.Add(slog.String("plain", "1"), slog.String("has space", "2"))
		Expect(hG.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(ContainSubstring(`"my group.plain"=1`))
		Expect(buf.String()).To(ContainSubstring(`"my group.has space"=2`))

		// %.ag without group spaces
		buf.Reset()
		hG2 := hAG.WithGroup("grp")
		Expect(hG2.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(ContainSubstring("grp.plain=1"))
		Expect(buf.String()).To(ContainSubstring(`"grp.has space"=2`))

		// %.ag without group at top level
		buf.Reset()
		hTopAG := logger.NewHandlerWithFormat(buf, nil, `%attr(%"pfx.".ag=%v%nl( ))`)
		rTop := slog.Record{Time: testTime, Message: "test"}
		rTop.Add(slog.String("item", "1"), slog.String("spaced item", "2"))
		Expect(hTopAG.Handle(context.Background(), rTop)).To(Succeed())
		Expect(buf.String()).To(ContainSubstring("pfx.item=1"))
		Expect(buf.String()).To(ContainSubstring(`"pfx.spaced item"=2`))

		// %.qg always quotes
		buf.Reset()
		hQG := logger.NewHandlerWithFormat(buf, nil, `%grp(%attr(%"pfx.".qg=%v%nl( )))`)
		hGQG := hQG.WithGroup("grp")
		Expect(hGQG.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(ContainSubstring(`"pfx.grp.plain"=1`))
		Expect(buf.String()).To(ContainSubstring(`"pfx.grp.has space"=2`))

		// %.g raw unquoted
		buf.Reset()
		hGRaw := logger.NewHandlerWithFormat(buf, nil, `%grp(%attr(%"pfx.".g=%v%nl( )))`)
		hGG := hGRaw.WithGroup("grp")
		Expect(hGG.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(ContainSubstring("pfx.grp.plain=1"))
		Expect(buf.String()).To(ContainSubstring("pfx.grp.has space=2"))

		// %_k and %_g with underscore prefix and separation
		buf.Reset()
		hUnderscore := logger.NewHandlerWithFormat(buf, nil, `%attr(%_k=%v )%grp(%attr(%_g=%v%nl( )))`)
		hGU := hUnderscore.WithGroup("grp")
		Expect(hGU.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(ContainSubstring("_grp_plain=1"))
		Expect(buf.String()).To(ContainSubstring("_grp_has space=2"))

		// %_qk and %_qg quoted
		buf.Reset()
		hQUnderscore := logger.NewHandlerWithFormat(buf, nil, `%attr(%_qk=%v )%grp(%attr(%_qg=%v%nl( )))`)
		hGQU := hQUnderscore.WithGroup("grp")
		Expect(hGQU.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(ContainSubstring(`"_grp_plain"=1`))
		Expect(buf.String()).To(ContainSubstring(`"_grp_has_x20space"=2`))

		buf.Reset()
		Expect(hQUnderscore.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(ContainSubstring(`"_plain"=1`))
		Expect(buf.String()).To(ContainSubstring(`"_has_x20space"=2`))

		// %_ak auto-quoted key
		buf.Reset()
		hAK := logger.NewHandlerWithFormat(buf, nil, `%attr(%_ak=%v%nl( ))`)
		Expect(hAK.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(ContainSubstring("_plain=1"))
		Expect(buf.String()).To(ContainSubstring(`"_has space"=2`))

		// %"custom".k and %"custom"_g combination
		buf.Reset()
		hCustomCombo := logger.NewHandlerWithFormat(buf, nil, `%attr(%"custom".k=%v )%grp(%attr(%"custom"_g=%v%nl( )))`)
		rCombo := slog.Record{Time: testTime, Message: "test"}
		rCombo.Add(slog.String("top_item", "top_val"))
		rCombo.Add(slog.Group("grp", slog.String("plain", "1"), slog.String("has space", "2")))
		Expect(hCustomCombo.Handle(context.Background(), rCombo)).To(Succeed())
		Expect(buf.String()).To(ContainSubstring("custom.top_item=top_val"))
		Expect(buf.String()).To(ContainSubstring("custom_grp_plain=1"))

		// %"custom"k and %"custom"g
		buf.Reset()
		hCustomSimple := logger.NewHandlerWithFormat(buf, nil, `%attr(%"custom"k=%v )%grp(%"custom"g:%attr(%k=%v%nl( )))`)
		Expect(hCustomSimple.Handle(context.Background(), rCombo)).To(Succeed())
		Expect(buf.String()).To(ContainSubstring("customtop_item=top_val"))
		Expect(buf.String()).To(ContainSubstring("customgrp:plain=1"))

		// %"custom"_k and %"custom".g
		buf.Reset()
		hCustomCombo2 := logger.NewHandlerWithFormat(buf, nil, `%attr(%"custom"_k=%v )%grp(%attr(%"custom".g=%v%nl( )))`)
		Expect(hCustomCombo2.Handle(context.Background(), rCombo)).To(Succeed())
		Expect(buf.String()).To(ContainSubstring("custom_top_item=top_val"))
		Expect(buf.String()).To(ContainSubstring("custom.grp.plain=1"))

		// %.k, %.qk, %.ak dotted keys
		buf.Reset()
		hDotKey := logger.NewHandlerWithFormat(buf, nil, `%attr(%.k=%v %.qk=%v %.ak=%v %"pfx.".k=%v%nl( ))`)
		Expect(hDotKey.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(ContainSubstring(".plain=1 \".plain\"=1 .plain=1 pfx.plain=1"))
		Expect(buf.String()).To(ContainSubstring(".has space=2 \".has space\"=2 \".has space\"=2 pfx.has space=2"))

		// %grp with %"custom".g, %"custom.".g, %"custom"_g outside %attr
		buf.Reset()
		hGrpOutside := logger.NewHandlerWithFormat(buf, nil, `%grp(%"custom".g=1 %"custom.".g=2 %"custom"_g=3 %_g=4 )`)
		hGO := hGrpOutside.WithGroup("grp")
		Expect(hGO.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(Equal("custom.grp=1 custom.grp=2 custom_grp=3 _grp=4 \n"))

		// %k with prefix always prepends prefix, even if key already starts with it
		buf.Reset()
		hPfxK := logger.NewHandlerWithFormat(buf, nil, `%attr(%"pfx."k=%v%nl( ))`)
		rPfxK := slog.Record{Time: testTime, Message: "test"}
		rPfxK.Add(slog.String("item", "1"), slog.String("pfx.existing", "2"))
		Expect(hPfxK.Handle(context.Background(), rPfxK)).To(Succeed())
		Expect(buf.String()).To(ContainSubstring("pfx.item=1"))
		Expect(buf.String()).To(ContainSubstring("pfx.pfx.existing=2"))

		// %gk with prefix always prepends prefix, even if group already starts with it
		buf.Reset()
		hPfxGK := logger.NewHandlerWithFormat(buf, nil, `%grp(%attr(%"pfx."gk=%v%nl( )))`)
		hGPfx := hPfxGK.WithGroup("pfx.grp")
		Expect(hGPfx.Handle(context.Background(), rPfxK)).To(Succeed())
		Expect(buf.String()).To(ContainSubstring("pfx.pfx.grp.item=1"))
		Expect(buf.String()).To(ContainSubstring("pfx.pfx.grp.pfx.existing=2"))
	})

	It("applies precision truncation before quoting and width padding outside quotes for %av", func() {
		// %.-5av truncates from end: "hello world" -> keeps leading 5 "hello" (no space, unquoted)
		buf.Reset()
		h1 := logger.NewHandlerWithFormat(buf, nil, `%attr(%.-5av%nl( ))`)
		r1 := slog.Record{Time: testTime, Message: "test"}
		r1.Add(slog.String("k", "hello world"))
		Expect(h1.Handle(context.Background(), r1)).To(Succeed())
		Expect(buf.String()).To(Equal("hello\n"))

		// %.-7av truncates from end: "hello world" -> keeps leading 7 "hello w" (has space, quoted as "\"hello w\"")
		buf.Reset()
		h2 := logger.NewHandlerWithFormat(buf, nil, `%attr(%.-7av%nl( ))`)
		Expect(h2.Handle(context.Background(), r1)).To(Succeed())
		Expect(buf.String()).To(Equal("\"hello w\"\n"))

		// %.5av truncates from start: "hello world" -> keeps trailing 5 "world" (no space, unquoted)
		buf.Reset()
		h3 := logger.NewHandlerWithFormat(buf, nil, `%attr(%.5av%nl( ))`)
		Expect(h3.Handle(context.Background(), r1)).To(Succeed())
		Expect(buf.String()).To(Equal("world\n"))

		// %.7av truncates from start: "hello world" -> keeps trailing 7 "o world" (has space, quoted as "\"o world\"")
		buf.Reset()
		h3b := logger.NewHandlerWithFormat(buf, nil, `%attr(%.7av%nl( ))`)
		Expect(h3b.Handle(context.Background(), r1)).To(Succeed())
		Expect(buf.String()).To(Equal("\"o world\"\n"))

		// %-10.-7av truncates to "hello w", quotes to "\"hello w\"" (len 9), pads to width 10 left-aligned
		buf.Reset()
		h4 := logger.NewHandlerWithFormat(buf, nil, `[%attr(%-10.-7av)]`)
		Expect(h4.Handle(context.Background(), r1)).To(Succeed())
		Expect(buf.String()).To(Equal("[\"hello w\" ]\n"))

		// %10.-7av pads to width 10 right-aligned outside the quotes
		buf.Reset()
		h5 := logger.NewHandlerWithFormat(buf, nil, `[%attr(%10.-7av)]`)
		Expect(h5.Handle(context.Background(), r1)).To(Succeed())
		Expect(buf.String()).To(Equal("[ \"hello w\"]\n"))

		// %-10.-5av unquoted value padded to width 10 left-aligned
		buf.Reset()
		h6 := logger.NewHandlerWithFormat(buf, nil, `[%attr(%-10.-5av)]`)
		Expect(h6.Handle(context.Background(), r1)).To(Succeed())
		Expect(buf.String()).To(Equal("[hello     ]\n"))
	})

	It("formats TextFormat with quoted whitespace levels, sources, and single-quoted grouped keys", func() {
		optsWS := &slog.HandlerOptions{
			AddSource: true,
			ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
				if a.Key == slog.LevelKey {
					return slog.String(slog.LevelKey, "CUSTOM LEVEL")
				}
				if a.Key == slog.SourceKey {
					return slog.String(slog.SourceKey, "my dir/main.go:42")
				}
				return a
			},
		}
		hWS := logger.NewHandlerWithFormat(buf, optsWS, logger.TextFormat)
		hWS.SetColor(false)
		t := time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)
		r := slog.Record{
			Time:    t,
			Message: "test msg",
			Level:   slog.LevelInfo,
			PC:      getFramePointer(),
		}
		r.Add(slog.Group("my group", slog.String("sub key", "value")))
		Expect(hWS.Handle(context.Background(), r)).To(Succeed())
		out := buf.String()
		Expect(out).To(ContainSubstring(`level="CUSTOM LEVEL"`))
		Expect(out).To(ContainSubstring(`source="my dir/main.go:42"`))
		Expect(out).To(ContainSubstring(`"my group.sub key"="value"`))
	})

	It("covers needsQuoting branches", func() {
		Expect(logger.NeedsQuoting("")).To(BeTrue())
		Expect(logger.NeedsQuoting("hello")).To(BeFalse())
		Expect(logger.NeedsQuoting("hello world")).To(BeTrue())
		Expect(logger.NeedsQuoting("k=v")).To(BeTrue())
		Expect(logger.NeedsQuoting(`"quote"`)).To(BeTrue())
		Expect(logger.NeedsQuoting(string([]byte{0x7f}))).To(BeTrue())
		Expect(logger.NeedsQuoting("line\nbreak")).To(BeTrue())
		Expect(logger.NeedsQuoting("Montréal")).To(BeFalse())
	})

	It("supports custom source formatting and bypasses ReplaceAttr", func() {
		optsReplacedSource := &slog.HandlerOptions{
			AddSource: true,
			ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
				if a.Key == slog.SourceKey {
					return slog.String(slog.SourceKey, "replaced_source_string")
				}
				return a
			},
		}

		// %source(%"json"v) formats ReplaceAttr value if provided
		buf.Reset()
		hJSONSrc := logger.NewHandlerWithFormat(buf, optsReplacedSource, `%source(%"json"v)`+"\n")
		pc, _, _, _ := runtime.Caller(0)
		r := slog.Record{Time: testTime, Message: "src test", PC: pc}
		Expect(hJSONSrc.Handle(context.Background(), r)).To(Succeed())
		var strReplaced string
		Expect(json.Unmarshal(buf.Bytes(), &strReplaced)).To(Succeed())
		Expect(strReplaced).To(Equal("replaced_source_string"))

		// %source(%"json"v) without ReplaceAttr outputs structured JSON
		buf.Reset()
		hJSONDefault := logger.NewHandlerWithFormat(buf, &slog.HandlerOptions{AddSource: true}, `%source(%"json"v)`+"\n")
		Expect(hJSONDefault.Handle(context.Background(), r)).To(Succeed())
		var js map[string]any
		Expect(json.Unmarshal(buf.Bytes(), &js)).To(Succeed())
		fileInfo, ok := js["file"].(map[string]any)
		Expect(ok).To(BeTrue())
		Expect(fileInfo["name"]).To(Equal("logger_suite_test.go"))
		Expect(js["function"]).NotTo(BeEmpty())

		// %source(%"json"qv) quotes the JSON output
		buf.Reset()
		hJSONQVSrc := logger.NewHandlerWithFormat(buf, &slog.HandlerOptions{AddSource: true}, `%source(%"json"qv)`+"\n")
		Expect(hJSONQVSrc.Handle(context.Background(), r)).To(Succeed())
		var str string
		Expect(json.Unmarshal(buf.Bytes(), &str)).To(Succeed())
		Expect(str).To(ContainSubstring(`"file":{"name":"logger_suite_test.go"`))

		// %source(%"custom"v) default layout in formatSource
		buf.Reset()
		hCustomSrc := logger.NewHandlerWithFormat(buf, &slog.HandlerOptions{AddSource: true}, `%source(%"custom"v)`+"\n")
		Expect(hCustomSrc.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(ContainSubstring("logger_suite_test.go:"))

		// Default %source(%v) and %source(%qv) without custom layout
		buf.Reset()
		hDefSrc := logger.NewHandlerWithFormat(buf, &slog.HandlerOptions{AddSource: true}, `%source(%v) %source(%qv)`+"\n")
		Expect(hDefSrc.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(ContainSubstring("logger_suite_test.go:"))
		Expect(buf.String()).To(ContainSubstring(`) "`))
	})

	It("prevents field collision in GELFFormat between top-level and grouped attributes", func() {
		h := logger.NewHandlerWithFormat(buf, nil, logger.GELFFormat)
		r := slog.Record{Time: testTime, Message: "collision check"}
		r.Add(slog.String("user_id", "top_user_val"))
		r.Add(slog.String("user.id", "top_user_dot_val"))
		r.Add(slog.String("user_x2eid", "top_user_escape_val"))
		r.Add(slog.String("user_x2Eid", "top_user_upper_escape_val"))
		r.Add(slog.Group("user", slog.String("id", "grouped_user_val")))
		r.Add(slog.Group("user.profile", slog.String("id", "literal_dot_grp_val")))
		r.Add(slog.Group("user_x2eprofile", slog.String("id", "literal_escape_grp_val")))
		r.Add(slog.Group("user", slog.Group("profile", slog.String("id", "nested_grp_val"))))
		r.Add(slog.Group("user", slog.String("x2eid", "attr_escape_val")))
		Expect(h.Handle(context.Background(), r)).To(Succeed())

		var js map[string]any
		Expect(json.Unmarshal(buf.Bytes(), &js)).To(Succeed())
		Expect(js["_user_id"]).To(Equal("top_user_val"))
		Expect(js["_user_x2eid"]).To(Equal("top_user_dot_val"))
		Expect(js["_user_x5fx2eid"]).To(Equal("top_user_escape_val"))
		Expect(js["_user_x5fx2Eid"]).To(Equal("top_user_upper_escape_val"))
		Expect(js["_user.id"]).To(Equal("grouped_user_val"))
		Expect(js["_user_x2eprofile.id"]).To(Equal("literal_dot_grp_val"))
		Expect(js["_user_x5fx2eprofile.id"]).To(Equal("literal_escape_grp_val"))
		Expect(js["_user.profile.id"]).To(Equal("nested_grp_val"))
		Expect(js["_user.x2eid"]).To(Equal("attr_escape_val"))
	})

	It("JSON-encodes maps, structs, and slices as strings in GELFFormat additional fields", func() {
		h := logger.NewHandlerWithFormat(buf, nil, logger.GELFFormat)
		r := slog.Record{Time: testTime, Message: "gelf complex values"}
		var nilPtr *CustomJSONType
		var nilSrc *slog.Source
		var nilPanic *NilTextPanicker
		r.Add(
			slog.Any("meta_map", map[string]string{"env": "staging"}),
			slog.Any("meta_slice", []string{"alpha", "beta"}),
			slog.Any("meta_struct", CustomJSONType{Name: "custom_val"}),
			slog.String("plain_str", "hello"),
			slog.Int("count", 42),
			slog.Float64("score", 98.5),
			slog.Any("nil_untyped", nil),
			slog.Any("nil_ptr", nilPtr),
			slog.Any("nil_source", nilSrc),
			slog.Any("nil_panic", nilPanic),
		)
		Expect(h.Handle(context.Background(), r)).To(Succeed())

		var js map[string]any
		Expect(json.Unmarshal(buf.Bytes(), &js)).To(Succeed())
		Expect(js["_meta_map"]).To(Equal(`{"env":"staging"}`))
		Expect(js["_meta_slice"]).To(Equal(`["alpha","beta"]`))
		Expect(js["_meta_struct"]).To(Equal(`{"custom":"custom_val"}`))
		Expect(js["_plain_str"]).To(Equal("hello"))
		Expect(js["_count"]).To(BeEquivalentTo(42))
		Expect(js["_score"]).To(BeNumerically("~", 98.5, 0.001))
		Expect(js["_nil_untyped"]).To(Equal("null"))
		Expect(js["_nil_ptr"]).To(Equal("null"))
		Expect(js["_nil_source"]).To(Equal("null"))
		Expect(js["_nil_panic"]).To(Equal("null"))

		// Preserves JSON null outside GELF
		buf.Reset()
		hJSON := logger.NewHandlerWithFormat(buf, nil, logger.JSONFormat)
		Expect(hJSON.Handle(context.Background(), r)).To(Succeed())
		var jsJSON map[string]any
		Expect(json.Unmarshal(buf.Bytes(), &jsJSON)).To(Succeed())
		Expect(jsJSON["nil_untyped"]).To(BeNil())
		Expect(jsJSON["nil_ptr"]).To(BeNil())
		Expect(jsJSON["nil_source"]).To(BeNil())
		Expect(jsJSON["nil_panic"]).To(BeNil())
	})

	It("returns dotless or malformed function name unchanged in ReduceFunctionName without panicking", func() {
		Expect(logger.ReduceFunctionName("dotlessFunc")).To(Equal("dotlessFunc"))
		Expect(logger.ReduceFunctionName("")).To(Equal(""))
		Expect(logger.ReduceFunctionName("pkg.(*")).To(Equal("pkg.(*"))
		Expect(logger.ReduceFunctionName("pkg.(*foo")).To(Equal("pkg.(*foo"))
		Expect(logger.ReduceFunctionName("pkg.(*)")).To(Equal("pkg.(*)"))
		Expect(logger.ReduceFunctionName(".")).To(Equal("."))
		Expect(logger.ReduceFunctionName(".foo")).To(Equal(".foo"))
		Expect(logger.ReduceFunctionName("foo.")).To(Equal("foo."))

		// Verify malformed inputs are not stored in functionCache
		_, okDot := logger.GetFunctionCacheEntry(".")
		Expect(okDot).To(BeFalse())
		_, okStar := logger.GetFunctionCacheEntry("pkg.(*")
		Expect(okStar).To(BeFalse())
		_, okEmptyRecv := logger.GetFunctionCacheEntry("pkg.(*)")
		Expect(okEmptyRecv).To(BeFalse())

		// Verify valid function name is stored in functionCache and retrieved on subsequent calls
		validFn := "github.com/BeamFoundry/rays/pkg/internal/testpkg.MyFunc"
		reduced := logger.ReduceFunctionName(validFn)
		cachedVal, okValid := logger.GetFunctionCacheEntry(validFn)
		Expect(okValid).To(BeTrue())
		Expect(cachedVal).To(Equal(reduced))
		Expect(logger.ReduceFunctionName(validFn)).To(Equal(reduced))

		// Text rendering of record with malformed function name
		buf.Reset()
		optsMalformedSrc := &slog.HandlerOptions{
			AddSource: true,
			ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
				if a.Key == slog.SourceKey {
					return slog.Any(slog.SourceKey, &slog.Source{
						Function: "pkg.(*",
						File:     "server.go",
						Line:     42,
					})
				}
				return a
			},
		}
		hText := logger.NewHandlerWithFormat(buf, optsMalformedSrc, logger.TextFormat)
		r := slog.Record{Time: testTime, Message: "malformed func", PC: getFramePointer()}
		Expect(hText.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(ContainSubstring("pkg.(*(server.go:42)"))

		// Text rendering of record with empty pointer receiver preserves original function name
		buf.Reset()
		optsEmptyRecvSrc := &slog.HandlerOptions{
			AddSource: true,
			ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
				if a.Key == slog.SourceKey {
					return slog.Any(slog.SourceKey, &slog.Source{
						Function: "pkg.(*)",
						File:     "server.go",
						Line:     42,
					})
				}
				return a
			},
		}
		hTextEmpty := logger.NewHandlerWithFormat(buf, optsEmptyRecvSrc, logger.TextFormat)
		Expect(hTextEmpty.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(ContainSubstring("pkg.(*)(server.go:42)"))

		// Generic pointer method reduction
		Expect(logger.ReduceFunctionName("pkg.(*Box[...]).Get")).To(Equal("pkg.Box"))
		Expect(logger.ReduceFunctionName("github.com/BeamFoundry/rays/pkg.(*Box[...]).Get")).To(Equal("g/B/rays/pkg.Box"))

		// Text rendering of generic pointer method preserves method name and reduces correctly
		buf.Reset()
		optsGenericSrc := &slog.HandlerOptions{
			AddSource: true,
			ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
				if a.Key == slog.SourceKey {
					return slog.Any(slog.SourceKey, &slog.Source{
						Function: "pkg.(*Box[...]).Get",
						File:     "server.go",
						Line:     42,
					})
				}
				return a
			},
		}
		hTextGeneric := logger.NewHandlerWithFormat(buf, optsGenericSrc, logger.TextFormat)
		Expect(hTextGeneric.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(ContainSubstring("pkg.Box(server.go:42)"))

		// Generic value receiver method reduction
		Expect(logger.ReduceFunctionName("pkg.Box[...].Get")).To(Equal("pkg.Box"))
		Expect(logger.ReduceFunctionName("github.com/BeamFoundry/rays/pkg.Box[...].Get")).To(Equal("g/B/rays/pkg.Box"))

		// Text rendering of generic value receiver method preserves method name and reduces correctly
		buf.Reset()
		optsGenericValSrc := &slog.HandlerOptions{
			AddSource: true,
			ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
				if a.Key == slog.SourceKey {
					return slog.Any(slog.SourceKey, &slog.Source{
						Function: "pkg.Box[...].Get",
						File:     "server.go",
						Line:     42,
					})
				}
				return a
			},
		}
		hTextGenericVal := logger.NewHandlerWithFormat(buf, optsGenericValSrc, logger.TextFormat)
		Expect(hTextGenericVal.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(ContainSubstring("pkg.Box(server.go:42)"))

		// Generic method with package separator slashes inside type arguments
		Expect(logger.ReduceFunctionName("pkg.(*Box[example.com/model.Item]).Get")).To(Equal("pkg.Box[example.com/model.Item].Get"))
		Expect(logger.ReduceFunctionName("github.com/BeamFoundry/rays/pkg.(*Box[example.com/model.Item]).Get")).To(Equal("g/B/rays/pkg.Box[example.com/model.Item].Get"))
		Expect(logger.ReduceFunctionName("pkg.Box[example.com/model.Item].Get")).To(Equal("pkg.Box[example.com/model.Item].Get"))
		Expect(logger.ReduceFunctionName("github.com/BeamFoundry/rays/pkg.Box[example.com/model.Item].Get")).To(Equal("g/B/rays/pkg.Box[example.com/model.Item].Get"))

		// Text rendering of generic method with slash in type argument
		buf.Reset()
		optsGenericSlashSrc := &slog.HandlerOptions{
			AddSource: true,
			ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
				if a.Key == slog.SourceKey {
					return slog.Any(slog.SourceKey, &slog.Source{
						Function: "pkg.(*Box[example.com/model.Item]).Get",
						File:     "server.go",
						Line:     42,
					})
				}
				return a
			},
		}
		hTextGenericSlash := logger.NewHandlerWithFormat(buf, optsGenericSlashSrc, logger.TextFormat)
		Expect(hTextGenericSlash.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(ContainSubstring("pkg.Box[example.com/model.Item].Get(server.go:42)"))
	})

	It("bounds functionCache so distinct valid names cannot accumulate without limit", func() {
		logger.SetFunctionCacheLimit(logger.GetFunctionCacheLen() + 1)
		defer logger.ResetFunctionCacheLimit()

		fn1 := "github.com/BeamFoundry/rays/pkg/testpkg.FuncOne"
		fn2 := "github.com/BeamFoundry/rays/pkg/testpkg.FuncTwo"

		// First call fits within limit and is cached
		res1 := logger.ReduceFunctionName(fn1)
		Expect(res1).To(Equal("g/B/rays/p/testpkg.FuncOne"))
		cached, ok := logger.GetFunctionCacheEntry(fn1)
		Expect(ok).To(BeTrue())
		Expect(cached).To(Equal(res1))

		// Second call exceeds limit: formatting is preserved, but not retained in cache
		res2 := logger.ReduceFunctionName(fn2)
		Expect(res2).To(Equal("g/B/rays/p/testpkg.FuncTwo"))
		_, ok2 := logger.GetFunctionCacheEntry(fn2)
		Expect(ok2).To(BeFalse())
	})

	It("handles typed-nil *slog.Source in source formatting paths", func() {
		var nilSrc *slog.Source
		b := logger.NewBuffer()
		logger.AppendFormattedValue(b, slog.AnyValue(nilSrc), true)
		Expect(b.String()).To(Equal("null"))

		b.Reset()
		logger.AppendFormattedValue(b, slog.AnyValue(nilSrc), true, true)
		Expect(b.String()).To(Equal(`"null"`))

		b.Reset()
		logger.AppendFormattedValue(b, slog.AnyValue(nilSrc), false)
		Expect(b.String()).To(Equal("<nil>"))

		optsNilSrc := &slog.HandlerOptions{
			AddSource: true,
			ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
				if a.Key == slog.SourceKey {
					return slog.Any(slog.SourceKey, nilSrc)
				}
				return a
			},
		}
		buf.Reset()
		hJSONNilSrc := logger.NewHandlerWithFormat(buf, optsNilSrc, logger.JSONFormat)
		r := slog.Record{Time: testTime, Message: "nil src test", PC: getFramePointer()}
		Expect(hJSONNilSrc.Handle(context.Background(), r)).To(Succeed())
		var js map[string]any
		Expect(json.Unmarshal(buf.Bytes(), &js)).To(Succeed())
		Expect(js["source"]).To(BeNil())

		buf.Reset()
		hUnquotedNilSrc := logger.NewHandlerWithFormat(buf, optsNilSrc, `%source(%v)`+"\n")
		Expect(hUnquotedNilSrc.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(Equal("<nil>\n"))

		buf.Reset()
		hQNilSrc := logger.NewHandlerWithFormat(buf, optsNilSrc, `%source(%qv)`+"\n")
		Expect(hQNilSrc.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(Equal("null\n"))

		buf.Reset()
		hCustomNilSrc := logger.NewHandlerWithFormat(buf, optsNilSrc, `%source(%"custom"v)`+"\n")
		Expect(hCustomNilSrc.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(Equal("<nil>\n"))

		// Untyped nil in formatCurrentValue
		optsUntypedNil := &slog.HandlerOptions{
			AddSource: true,
			ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
				if a.Key == slog.SourceKey {
					return slog.Any(slog.SourceKey, nil)
				}
				return a
			},
		}
		buf.Reset()
		hUntypedNilV := logger.NewHandlerWithFormat(buf, optsUntypedNil, `%source(%v)`+"\n")
		Expect(hUntypedNilV.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(Equal("<nil>\n"))

		buf.Reset()
		hUntypedNilQV := logger.NewHandlerWithFormat(buf, optsUntypedNil, `%source(%qv)`+"\n")
		Expect(hUntypedNilQV.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(Equal("null\n"))

		// GELF JSON layouts encode nil *slog.Source as string "null"
		buf.Reset()
		hGELFNilSrc := logger.NewHandlerWithFormat(buf, optsNilSrc, logger.GELFFormat)
		Expect(hGELFNilSrc.Handle(context.Background(), r)).To(Succeed())
		var jsGELFNil map[string]any
		Expect(json.Unmarshal(buf.Bytes(), &jsGELFNil)).To(Succeed())
		Expect(jsGELFNil["_source"]).To(Equal("null"))

		// GELF JSON layouts encode untyped nil source as string "null"
		buf.Reset()
		hGELFUntypedNil := logger.NewHandlerWithFormat(buf, optsUntypedNil, logger.GELFFormat)
		Expect(hGELFUntypedNil.Handle(context.Background(), r)).To(Succeed())
		var jsGELFUntyped map[string]any
		Expect(json.Unmarshal(buf.Bytes(), &jsGELFUntyped)).To(Succeed())
		Expect(jsGELFUntyped["_source"]).To(Equal("null"))

		// Preserves existing output for %"json"qv without adding another layer of quotes
		buf.Reset()
		hJSONQVNilSrc := logger.NewHandlerWithFormat(buf, optsNilSrc, `%source(%"json"qv)`+"\n")
		Expect(hJSONQVNilSrc.Handle(context.Background(), r)).To(Succeed())
		var strNilSrc string
		Expect(json.Unmarshal(buf.Bytes(), &strNilSrc)).To(Succeed())
		Expect(strNilSrc).To(Equal("null"))

		// GELF layout with unquoted %v preserves <nil> representation
		buf.Reset()
		hGELFUnquotedNilSrc := logger.NewHandlerWithFormat(buf, optsNilSrc, `{"version":"1.1",%source("_source":%v)}`+"\n")
		Expect(hGELFUnquotedNilSrc.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(ContainSubstring(`"_source":<nil>`))
	})

	It("formats ReplaceAttr source value in JSON, ECS, and Logstash layouts without reconstructing PC", func() {
		optsCustomSrc := &slog.HandlerOptions{
			AddSource: true,
			ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
				if a.Key == slog.SourceKey {
					return slog.String(slog.SourceKey, "replaced/source.go:99")
				}
				return a
			},
		}
		r := slog.Record{Time: testTime, Message: "custom src test", PC: getFramePointer()}

		// JSONFormat
		buf.Reset()
		hJSON := logger.NewHandlerWithFormat(buf, optsCustomSrc, logger.JSONFormat)
		Expect(hJSON.Handle(context.Background(), r)).To(Succeed())
		var js map[string]any
		Expect(json.Unmarshal(buf.Bytes(), &js)).To(Succeed())
		Expect(js["source"]).To(Equal("replaced/source.go:99"))

		// ECSFormat
		buf.Reset()
		hECS := logger.NewHandlerWithFormat(buf, optsCustomSrc, logger.ECSFormat)
		Expect(hECS.Handle(context.Background(), r)).To(Succeed())
		var jsECS map[string]any
		Expect(json.Unmarshal(buf.Bytes(), &jsECS)).To(Succeed())
		Expect(jsECS["log.origin"]).To(Equal("replaced/source.go:99"))

		// LogstashFormat
		buf.Reset()
		hLogstash := logger.NewHandlerWithFormat(buf, optsCustomSrc, logger.LogstashFormat)
		Expect(hLogstash.Handle(context.Background(), r)).To(Succeed())
		var jsLogstash map[string]any
		Expect(json.Unmarshal(buf.Bytes(), &jsLogstash)).To(Succeed())
		Expect(jsLogstash["source"]).To(Equal("replaced/source.go:99"))

		// ReplaceAttr with custom struct in JSON layout
		optsStructSrc := &slog.HandlerOptions{
			AddSource: true,
			ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
				if a.Key == slog.SourceKey {
					return slog.Any(slog.SourceKey, CustomJSONType{Name: "custom_src"})
				}
				return a
			},
		}
		buf.Reset()
		hJSONStruct := logger.NewHandlerWithFormat(buf, optsStructSrc, logger.JSONFormat)
		Expect(hJSONStruct.Handle(context.Background(), r)).To(Succeed())
		var jsStruct map[string]any
		Expect(json.Unmarshal(buf.Bytes(), &jsStruct)).To(Succeed())
		srcObj, ok := jsStruct["source"].(map[string]any)
		Expect(ok).To(BeTrue())
		Expect(srcObj["custom"]).To(Equal("custom_src"))

		// ReplaceAttr with TestLogValuer resolving to *slog.Source in JSON, ECS, and Logstash
		optsValuerSrc := &slog.HandlerOptions{
			AddSource: true,
			ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
				if a.Key == slog.SourceKey {
					return slog.Any(slog.SourceKey, TestLogValuer{Val: &slog.Source{
						Function: "pkg.ValuerFunc",
						File:     "/path/to/valuer_file.go",
						Line:     42,
					}})
				}
				return a
			},
		}
		buf.Reset()
		hECSValuer := logger.NewHandlerWithFormat(buf, optsValuerSrc, logger.ECSFormat)
		Expect(hECSValuer.Handle(context.Background(), r)).To(Succeed())
		var jsECSValuer map[string]any
		Expect(json.Unmarshal(buf.Bytes(), &jsECSValuer)).To(Succeed())
		originValuer, ok := jsECSValuer["log.origin"].(map[string]any)
		Expect(ok).To(BeTrue())
		fileValuer, ok := originValuer["file"].(map[string]any)
		Expect(ok).To(BeTrue())
		Expect(fileValuer["name"]).To(Equal("valuer_file.go"))
		Expect(fileValuer["line"]).To(BeEquivalentTo(42))
		Expect(originValuer["function"]).To(Equal("pkg.ValuerFunc"))

		buf.Reset()
		hJSONValuer := logger.NewHandlerWithFormat(buf, optsValuerSrc, logger.JSONFormat)
		Expect(hJSONValuer.Handle(context.Background(), r)).To(Succeed())
		var jsJSONValuer map[string]any
		Expect(json.Unmarshal(buf.Bytes(), &jsJSONValuer)).To(Succeed())
		srcValuerJSON, ok := jsJSONValuer["source"].(map[string]any)
		Expect(ok).To(BeTrue())
		fileValuerJSON, ok := srcValuerJSON["file"].(map[string]any)
		Expect(ok).To(BeTrue())
		Expect(fileValuerJSON["name"]).To(Equal("valuer_file.go"))

		buf.Reset()
		hLogstashValuer := logger.NewHandlerWithFormat(buf, optsValuerSrc, logger.LogstashFormat)
		Expect(hLogstashValuer.Handle(context.Background(), r)).To(Succeed())
		var jsLogstashValuer map[string]any
		Expect(json.Unmarshal(buf.Bytes(), &jsLogstashValuer)).To(Succeed())
		srcValuerLogstash, ok := jsLogstashValuer["source"].(map[string]any)
		Expect(ok).To(BeTrue())
		fileValuerLogstash, ok := srcValuerLogstash["file"].(map[string]any)
		Expect(ok).To(BeTrue())
		Expect(fileValuerLogstash["name"]).To(Equal("valuer_file.go"))

		// ReplaceAttr with TestLogValuer resolving to string and struct
		optsValuerStr := &slog.HandlerOptions{
			AddSource: true,
			ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
				if a.Key == slog.SourceKey {
					return slog.Any(slog.SourceKey, TestLogValuer{Val: "valuer_string_src"})
				}
				return a
			},
		}
		buf.Reset()
		hJSONValuerStr := logger.NewHandlerWithFormat(buf, optsValuerStr, logger.JSONFormat)
		Expect(hJSONValuerStr.Handle(context.Background(), r)).To(Succeed())
		var jsValuerStr map[string]any
		Expect(json.Unmarshal(buf.Bytes(), &jsValuerStr)).To(Succeed())
		Expect(jsValuerStr["source"]).To(Equal("valuer_string_src"))

		// ReplaceAttr with slog.Source by value
		optsValSource := &slog.HandlerOptions{
			AddSource: true,
			ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
				if a.Key == slog.SourceKey {
					return slog.Any(slog.SourceKey, slog.Source{
						Function: "pkg.DirectValFunc",
						File:     "/path/to/direct_file.go",
						Line:     88,
					})
				}
				return a
			},
		}
		buf.Reset()
		hECSDirect := logger.NewHandlerWithFormat(buf, optsValSource, logger.ECSFormat)
		Expect(hECSDirect.Handle(context.Background(), r)).To(Succeed())
		var jsECSDirect map[string]any
		Expect(json.Unmarshal(buf.Bytes(), &jsECSDirect)).To(Succeed())
		originDirect, ok := jsECSDirect["log.origin"].(map[string]any)
		Expect(ok).To(BeTrue())
		fileDirect, ok := originDirect["file"].(map[string]any)
		Expect(ok).To(BeTrue())
		Expect(fileDirect["name"]).To(Equal("direct_file.go"))

		// Direct value source in unquoted and quoted text
		buf.Reset()
		hTextDirect := logger.NewHandlerWithFormat(buf, optsValSource, `%source(%"custom"v)`+"\n")
		Expect(hTextDirect.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(ContainSubstring("DirectValFunc(direct_file.go:88)"))

		buf.Reset()
		hTextCurrent := logger.NewHandlerWithFormat(buf, optsValSource, `%source(%v)`+"\n")
		Expect(hTextCurrent.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(ContainSubstring("DirectValFunc(direct_file.go:88)"))
	})

	It("preserves dotted and underscore separators for empty attribute keys inside groups", func() {
		// TextFormat emits trailing dot
		buf.Reset()
		hText := logger.NewHandlerWithFormat(buf, nil, logger.TextFormat)
		r := slog.Record{Time: testTime, Message: "empty key test"}
		r.Add(slog.Group("my_group", slog.String("", "empty_key_val")))
		Expect(hText.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(ContainSubstring(`my_group.="empty_key_val"`))

		// Underscore format emits trailing underscore
		buf.Reset()
		hUnderscore := logger.NewHandlerWithFormat(buf, nil, `%grp(%attr(,%_qg:%qv))`+"\n")
		Expect(hUnderscore.Handle(context.Background(), r)).To(Succeed())
		Expect(buf.String()).To(ContainSubstring(`,"_my_group_":"empty_key_val"`))
	})

	It("covers fallback and string formatting in formatSource", func() {
		optsNilVal := &slog.HandlerOptions{
			AddSource: true,
			ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
				if a.Key == slog.SourceKey {
					return slog.Any(slog.SourceKey, nil)
				}
				return a
			},
		}
		// ReplaceAttr replacement of slog.SourceKey with nil preserves null in JSON layout
		buf.Reset()
		hFallbackJSON := logger.NewHandlerWithFormat(buf, optsNilVal, logger.JSONFormat)
		rWithPC := slog.Record{Time: testTime, Message: "fb", PC: getFramePointer()}
		Expect(hFallbackJSON.Handle(context.Background(), rWithPC)).To(Succeed())
		var js map[string]any
		Expect(json.Unmarshal(buf.Bytes(), &js)).To(Succeed())
		val, exists := js["source"]
		Expect(exists).To(BeTrue())
		Expect(val).To(BeNil())

		// ReplaceAttr replacement of slog.SourceKey with nil preserves <nil> in unquoted custom layout
		buf.Reset()
		hFallbackCustom := logger.NewHandlerWithFormat(buf, optsNilVal, `%source(%"custom"v)`+"\n")
		Expect(hFallbackCustom.Handle(context.Background(), rWithPC)).To(Succeed())
		Expect(buf.String()).To(Equal("<nil>\n"))

		// Fallback when no replacement source attribute is present (key is empty)
		jsonFallback := logger.FormatSourceHelper("json", slog.Attr{}, getFramePointer())
		Expect(jsonFallback).To(ContainSubstring(`"file":{"name":`))
		customFallback := logger.FormatSourceHelper("custom", slog.Attr{}, getFramePointer())
		Expect(customFallback).To(ContainSubstring("("))
		Expect(customFallback).To(ContainSubstring(")"))

		// Fallback when no replacement source attribute is present and PC is 0
		Expect(logger.FormatSourceHelper("json", slog.Attr{}, 0)).To(Equal("null"))
		Expect(logger.FormatSourceHelper("custom", slog.Attr{}, 0)).To(Equal("<nil>"))

		// String replacement in unquoted custom layout
		optsStr := &slog.HandlerOptions{
			AddSource: true,
			ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
				if a.Key == slog.SourceKey {
					return slog.String(slog.SourceKey, "custom_str_source")
				}
				return a
			},
		}
		buf.Reset()
		hStrCustom := logger.NewHandlerWithFormat(buf, optsStr, `%source(%"custom"v)`+"\n")
		Expect(hStrCustom.Handle(context.Background(), rWithPC)).To(Succeed())
		Expect(buf.String()).To(Equal("custom_str_source\n"))
	})

	It("covers underscore-prefixed dotted keys and groups with custom prefixes", func() {
		buf.Reset()
		hDotUnder := logger.NewHandlerWithFormat(buf, nil, `%attr(%_.k=%v %"pfx"_.k=%v)`+"\n")
		rKey := slog.Record{Time: testTime, Message: "test"}
		rKey.Add(slog.String("key", "val"))
		Expect(hDotUnder.Handle(context.Background(), rKey)).To(Succeed())
		Expect(buf.String()).To(ContainSubstring("_.key=val"))
		Expect(buf.String()).To(ContainSubstring("pfx_.key=val"))

		// Inside group with custom prefix
		buf.Reset()
		hCombo := logger.NewHandlerWithFormat(buf, nil, `%grp(%attr(%"pfx"_.g=%v ))`+"\n")
		rGrp := slog.Record{Time: testTime, Message: "test"}
		rGrp.Add(slog.Group("grp", slog.String("plain", "1")))
		Expect(hCombo.Handle(context.Background(), rGrp)).To(Succeed())
		Expect(buf.String()).To(ContainSubstring("pfx_grp.plain=1"))

		// Outside attr in group
		buf.Reset()
		hGrpDirect := logger.NewHandlerWithFormat(buf, nil, `%grp(%_.g %"pfx"_.g)`+"\n")
		Expect(hGrpDirect.Handle(context.Background(), rGrp)).To(Succeed())
		Expect(buf.String()).To(ContainSubstring("_grp pfx_grp"))
	})

	It("sanitizes completed keys and groups with custom prefixes containing spaces in GELF mode", func() {
		buf.Reset()
		hPrefSpace := logger.NewHandlerWithFormat(buf, nil, `%attr(%"app prefix"_qk=%v )%grp(%attr(%"grp prefix"_qg=%v%nl( )))`+"\n")
		rPref := slog.Record{Time: testTime, Message: "prefix space test"}
		rPref.Add(slog.String("my key", "val1"))
		rPref.Add(slog.Group("my grp", slog.String("inner key", "val2")))
		Expect(hPrefSpace.Handle(context.Background(), rPref)).To(Succeed())
		Expect(buf.String()).To(ContainSubstring(`"app_x20prefix_my_x20key"=val1`))
		Expect(buf.String()).To(ContainSubstring(`"grp_x20prefix_my_x20grp_inner_x20key"=val2`))
	})

	It("handles nil pointer panics in appendFormattedValue with quoting", func() {
		b := logger.NewBuffer()
		var p *NilTextPanicker
		logger.AppendFormattedValue(b, slog.AnyValue(p), true)
		Expect(b.String()).To(Equal("null"))

		b.Reset()
		logger.AppendFormattedValue(b, slog.AnyValue(p), true, true)
		Expect(b.String()).To(Equal(`"null"`))

		b.Reset()
		logger.AppendFormattedValue(b, slog.AnyValue(p), false)
		Expect(b.String()).To(Equal("<nil>"))
	})
})

type ErrorTextMarshaler struct{}

func (e *ErrorTextMarshaler) MarshalText() ([]byte, error) {
	return nil, errors.New("marshaler failure")
}

type CustomJSONType struct {
	Name string `json:"name"`
}

func (c CustomJSONType) MarshalJSON() ([]byte, error) {
	return []byte(`{"custom":"` + c.Name + `"}`), nil
}

type TestLogValuer struct {
	Val any
}

func (v TestLogValuer) LogValue() slog.Value {
	return slog.AnyValue(v.Val)
}

type NilPointerPanicker struct{}

func (n *NilPointerPanicker) MarshalJSON() ([]byte, error) {
	_ = *n
	return []byte("{}"), nil
}

type NilStringPanicker struct{}

func (n *NilStringPanicker) String() string {
	_ = *n
	return ""
}

type NonNilPanicker struct{}

func (n NonNilPanicker) MarshalText() ([]byte, error) {
	panic("boom")
}

type NilTextPanicker struct{}

func (n *NilTextPanicker) MarshalText() ([]byte, error) {
	_ = *n
	return []byte("text"), nil
}
