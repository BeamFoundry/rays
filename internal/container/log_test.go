package container_test

import (
	"bytes"
	"context"
	"os"
	"reflect"
	"strings"
	_ "unsafe"

	"github.com/BeamFoundry/rays/internal/boot/logger"
	"github.com/BeamFoundry/rays/internal/container"
	. "github.com/BeamFoundry/rays/pkg/core"
	. "github.com/BeamFoundry/rays/pkg/testing"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = BeforeEach(func() {
	GetContainer().EnableScanning()
})

type BeamWithLogger struct {
	logger Logging `@:"Inject"`
}

var _ = Describe("Logging", func() {
	It("can do stuff", func() {
		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())
		instance := GetBeamDefaultInstance[Logging]()
		instance.Trace("logging")
		instance.TraceContext(context.Background(), "logging")
		instance.Fatal("logging")
		instance.FatalContext(context.Background(), "logging")
		origOpts := instance.GetHandlerOptions()
		opts := origOpts
		opts.AddSource = !opts.AddSource
		Expect(instance.GetHandlerOptions().AddSource).NotTo(Equal(opts.AddSource))
		instance.SetHandlerOptions(&opts)
		Expect(instance.GetHandlerOptions().AddSource).To(Equal(opts.AddSource))
		instance.SetHandlerOptions(&origOpts)
	})
	It("can inject itself", func() {
		err := GetContainer().LoadConfigAt(map[string]any{
			"handler": "DEFAULT",
			"level":   "DEBUG",
			"loggers": map[string]any{
				"github.com/BeamFoundry/rays/internal/container":      "DEBUG",
				"github.com/BeamFoundry/rays/internal/container_test": "DEBUG",
				"github.com/BeamFoundry/rays/pkg/testing":             "DEBUG",
			},
		}, "rays.logging")
		Expect(err).NotTo(HaveOccurred())
		_, err = GetContainer().AddBeamFor(&BeamWithLogger{})
		Expect(err).NotTo(HaveOccurred())
		GetContainer().EnableScanning()
		err = GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())

		beam := GetBeamDefaultInstance[*BeamWithLogger]()
		Expect(beam).NotTo(BeNil())
		Expect(beam.logger).NotTo(BeNil())
	})
})

var _ = Describe("LoggingProperties", func() {
	It("can set a slog/JSONHandler", func() {
		Expect(GetContainer().LoadConfigAt(map[string]any{"handler": "JSON", "level": "DEBUG"}, "rays.logging")).NotTo(HaveOccurred())
		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())

		h, ok := GetLogger().Handler().(*logger.Handler)
		Expect(ok).To(BeTrue())
		Expect(h.GetFormat()).To(Equal(logger.JSONFormat))
	})
	It("can set a slog/TextHandler", func() {
		Expect(GetContainer().LoadConfigAt(map[string]any{"handler": "TEXT", "level": "DEBUG"}, "rays.logging")).NotTo(HaveOccurred())
		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())

		h, ok := GetLogger().Handler().(*logger.Handler)
		Expect(ok).To(BeTrue())
		Expect(h.GetFormat()).To(Equal(logger.TextFormat))
	})

	It("can set a custom Handler", func() {
		Expect(GetContainer().LoadConfigAt(map[string]any{"handler": "DEFAULT", "level": "DEBUG"}, "rays.logging")).NotTo(HaveOccurred())
		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())

		h, ok := GetLogger().Handler().(*logger.Handler)
		Expect(ok).To(BeTrue())
		Expect(h.GetFormat()).To(Equal(logger.DefaultFormat))
	})

	It("can use With", func() {
		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())
		beam := GetBeamDefaultInstance[Logging]()
		beam.With("foo", "bar").Trace("log")
	})

	It("can use WithGroup", func() {
		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())
		beam := GetBeamDefaultInstance[Logging]()
		beam.WithGroup("foo").Trace("log")
	})

	It("can set level to TRACE", func() {
		Expect(GetContainer().LoadConfigAt(map[string]any{"level": "TRACE"}, "rays.logging")).NotTo(HaveOccurred())
		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())
	})

	It("can set level to DEBUG", func() {
		Expect(GetContainer().LoadConfigAt(map[string]any{"level": "DEBUG"}, "rays.logging")).NotTo(HaveOccurred())
		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())
	})

	It("can set level to INFO", func() {
		Expect(GetContainer().LoadConfigAt(map[string]any{"level": "INFO"}, "rays.logging")).NotTo(HaveOccurred())
		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())
	})

	It("can set level to WARN", func() {
		Expect(GetContainer().LoadConfigAt(map[string]any{"level": "WARN"}, "rays.logging")).NotTo(HaveOccurred())
		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())
	})

	It("can set level to ERROR", func() {
		Expect(GetContainer().LoadConfigAt(map[string]any{"level": "ERROR"}, "rays.logging")).NotTo(HaveOccurred())
		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())
	})

	It("can set level to FATAL", func() {
		Expect(GetContainer().LoadConfigAt(map[string]any{"level": "FATAL"}, "rays.logging")).NotTo(HaveOccurred())
		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())
	})

	It("can set custom format", func() {
		Expect(GetContainer().LoadConfigAt(map[string]any{"handler": "CUSTOM", "format": "%level(%-5v) %msg(%v)"}, "rays.logging")).NotTo(HaveOccurred())
		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())
		Expect(GetLogger().GetFormat()).To(Equal("%level(%-5v) %msg(%v)"))
	})

	It("defaults custom format to DefaultFormat", func() {
		Expect(GetContainer().LoadConfigAt(map[string]any{"handler": "CUSTOM", "format": ""}, "rays.logging")).NotTo(HaveOccurred())
		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())
		Expect(GetLogger().GetFormat()).To(Equal(logger.DefaultFormat))
	})

	It("can set handler to ECS", func() {
		Expect(GetContainer().LoadConfigAt(map[string]any{"handler": "ECS"}, "rays.logging")).NotTo(HaveOccurred())
		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())
		Expect(GetLogger().GetFormat()).To(Equal(logger.ECSFormat))
	})

	It("can set handler to GELF", func() {
		Expect(GetContainer().LoadConfigAt(map[string]any{"handler": "GELF"}, "rays.logging")).NotTo(HaveOccurred())
		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())
		Expect(GetLogger().GetFormat()).To(Equal(logger.GELFFormat))
	})

	It("can set handler to LOGSTASH", func() {
		Expect(GetContainer().LoadConfigAt(map[string]any{"handler": "LOGSTASH"}, "rays.logging")).NotTo(HaveOccurred())
		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())
		Expect(GetLogger().GetFormat()).To(Equal(logger.LogstashFormat))
	})

	It("can set color to ON", func() {
		Expect(GetContainer().LoadConfigAt(map[string]any{"color": "ON"}, "rays.logging")).NotTo(HaveOccurred())
		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())
		Expect(GetLogger().IsColor()).To(BeTrue())
	})

	It("can set color to OFF", func() {
		Expect(GetContainer().LoadConfigAt(map[string]any{"color": "OFF"}, "rays.logging")).NotTo(HaveOccurred())
		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())
		Expect(GetLogger().IsColor()).To(BeFalse())
	})

	It("can set color to AUTO", func() {
		origNoColor, hasNoColor := os.LookupEnv("NO_COLOR")
		origTerm, hasTerm := os.LookupEnv("TERM")
		origColorTerm, hasColorTerm := os.LookupEnv("COLORTERM")
		origArgs := os.Args
		//nolint:errcheck // restoring environment in test cleanup
		defer func() {
			if hasNoColor {
				os.Setenv("NO_COLOR", origNoColor)
			} else {
				os.Unsetenv("NO_COLOR")
			}
			if hasTerm {
				os.Setenv("TERM", origTerm)
			} else {
				os.Unsetenv("TERM")
			}
			if hasColorTerm {
				os.Setenv("COLORTERM", origColorTerm)
			} else {
				os.Unsetenv("COLORTERM")
			}
			os.Args = origArgs
			GetLogger().SetOutput(GinkgoWriter)
		}()

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

		termWriter := &fakeTerminalWriter{Buffer: &bytes.Buffer{}}
		GetLogger().SetOutput(termWriter)

		Expect(GetContainer().LoadConfigAt(map[string]any{"color": "AUTO"}, "rays.logging")).NotTo(HaveOccurred())
		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())
		Expect(GetLogger().IsColor()).To(BeTrue())

		// On non-terminal writer (like bytes.Buffer), AUTO detects false
		buf := &bytes.Buffer{}
		GetLogger().SetOutput(buf)
		Expect(GetLogger().IsColor()).To(BeFalse())

		// Forcing SetColor(true) forces color even on non-terminal
		GetLogger().SetColor(true)
		Expect(GetLogger().IsColor()).To(BeTrue())
	})

	It("tests LoggingProperties.Color and default LogHandler", func() {
		Expect(GetContainer().Initialize()).To(Succeed())
		props := GetBeamDefaultInstance[*container.LoggingProperties]()
		Expect(props.Color()).To(Equal(props.LogColor()))

		var unknownHandler container.LogHandler
		Expect(unknownHandler.Handler()).To(BeNil())
	})

	It("resolves interface and concrete overrides in LoggingConfiguration.Logger", func() {
		Expect(GetContainer().LoadConfigAt(map[string]any{
			"handler": "DEFAULT",
			"level":   "INFO",
			"loggers": map[string]any{
				"github.com/BeamFoundry/rays/internal/container_test/DummyLoggerInterface": "WARN",
				"github.com/BeamFoundry/rays/internal/container_test/DummyLoggerImpl":      "DEBUG",
			},
		}, "rays.logging")).NotTo(HaveOccurred())
		Expect(GetContainer().Initialize()).To(Succeed())

		logConfig := &container.LoggingConfiguration{}

		// 1. With concrete instance and interface parent type
		ctx1 := context.WithValue(context.Background(), PrototypeContextParentType, reflect.TypeFor[DummyLoggerInterface]())
		ctx1 = context.WithValue(ctx1, PrototypeContextParentInstance, &DummyLoggerImpl{})
		lg1 := logConfig.Logger(ctx1)
		Expect(lg1).NotTo(BeNil())

		// 2. With interface parent type but concrete instance unavailable
		ctx2 := context.WithValue(context.Background(), PrototypeContextParentType, reflect.TypeFor[DummyLoggerInterface]())
		lg2 := logConfig.Logger(ctx2)
		Expect(lg2).NotTo(BeNil())

		// 3. With concrete instance only (no parent type)
		ctx3 := context.WithValue(context.Background(), PrototypeContextParentInstance, &DummyLoggerImpl{})
		lg3 := logConfig.Logger(ctx3)
		Expect(lg3).NotTo(BeNil())

		// 4. With pointer-to-struct parent type
		ctx4 := context.WithValue(context.Background(), PrototypeContextParentType, reflect.TypeFor[*DummyLoggerImpl]())
		lg4 := logConfig.Logger(ctx4)
		Expect(lg4).NotTo(BeNil())

		// 5. Empty context
		lg5 := logConfig.Logger(context.Background())
		Expect(lg5).NotTo(BeNil())
	})
})

type DummyLoggerInterface interface {
	DoLog()
}

type DummyLoggerImpl struct {
	Logger Logging `@:"Inject"`
}

func (d *DummyLoggerImpl) DoLog() {
	if d.Logger != nil {
		d.Logger.Info("DoLog")
	}
}

type fakeTerminalWriter struct {
	*bytes.Buffer
}

func (f *fakeTerminalWriter) IsTerminal() bool { return true }
