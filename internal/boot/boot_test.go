package boot_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"reflect"
	"testing/fstest"

	"github.com/BeamFoundry/rays/internal/boot"
	"github.com/BeamFoundry/rays/internal/boot/logger"
	"github.com/BeamFoundry/rays/internal/services"
	. "github.com/BeamFoundry/rays/pkg/core"
	"github.com/BeamFoundry/rays/pkg/lang"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type DummyStruct struct{}

type failAfterLoggingInitFS struct {
	fs.FS
	bl *boot.BootLoader
}

func (f failAfterLoggingInitFS) Open(name string) (fs.File, error) {
	if f.bl.LoggingProperties() != nil {
		return nil, errors.New("phase 4 config reinit failed")
	}
	file, err := f.FS.Open(name)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", name, err)
	}
	return file, nil
}

var _ = BeforeSuite(func() {
	services.Get().Logger.SetOutput(GinkgoWriter)
})

var _ = Describe("BootLoader", func() {
	var bl *boot.BootLoader

	BeforeEach(func() {
		bl = boot.NewBootLoader()
		services.Get().Logger.SetOutput(GinkgoWriter)
	})

	AfterEach(func() {
		bl.Reset()
	})

	It("creates a BootLoader with default quiet logger and default config", func() {
		Expect(bl).NotTo(BeNil())
		Expect(bl.Logger()).NotTo(BeNil())
		Expect(bl.Config).NotTo(BeNil())
		Expect(bl.LoggerForType(reflect.TypeFor[*DummyStruct]())).NotTo(BeNil())
	})

	It("can boot successfully with default settings", func() {
		Expect(bl.Boot()).NotTo(HaveOccurred())
		Expect(bl.LoggingProperties()).NotTo(BeNil())
		Expect(bl.LoggingProperties().LogLevel().Enum).To(Equal(boot.LogLevel{}.New().INFO))
		Expect(bl.LoggingProperties().LogHandler().Enum).To(Equal(boot.LogHandler{}.New().DEFAULT))
		Expect(bl.LoggingProperties().LogColor().Enum).To(Equal(boot.LogColor{}.New().AUTO))
		Expect(bl.LoggingProperties().Color().Enum).To(Equal(boot.LogColor{}.New().AUTO))
		Expect(bl.LoggingProperties().AddSource()).To(BeTrue())
		Expect(bl.LoggingProperties().Format()).To(Equal(logger.DefaultFormat))
		Expect(bl.LoggingProperties().MapLevels()).To(BeEmpty())
	})

	It("can boot with custom logging properties and package-specific loggers", func() {
		cfg := map[string]any{
			"level":   "DEBUG",
			"handler": "JSON",
			"color":   "ON",
			"source":  true,
			"loggers": map[string]any{
				"github.com/BeamFoundry/rays/internal/boot/":                 "TRACE",
				"github.com/BeamFoundry/rays/internal/boot_test":             "DEBUG",
				"github.com/BeamFoundry/rays/internal/boot_test/DummyStruct": "WARN",
				"io/Reader": "ERROR",
			},
		}
		Expect(bl.LoadConfigAt(cfg, "rays.logging")).NotTo(HaveOccurred())
		Expect(bl.Boot()).NotTo(HaveOccurred())

		Expect(bl.Logger()).NotTo(BeNil())
		Expect(services.Get().Logger).NotTo(BeNil())

		// Test LoggerForType and LoggerForClass
		lg1 := bl.LoggerForClass(&DummyStruct{})
		Expect(lg1).NotTo(BeNil())

		lg2 := bl.LoggerForType(reflect.TypeFor[*DummyStruct]())
		Expect(lg2).NotTo(BeNil())

		// Test interface-typed logger
		lgReader := bl.LoggerForType(reflect.TypeFor[io.Reader]())
		Expect(lgReader).NotTo(BeNil())

		// Test LoggerForTypes candidate order precedence
		lgTypes := bl.LoggerForTypes(reflect.TypeFor[*DummyStruct](), reflect.TypeFor[io.Reader]())
		Expect(lgTypes).NotTo(BeNil())
		Expect(lgTypes.GetHandlerOptions().Level.Level()).To(Equal(slog.LevelWarn))

		lgTypesReversed := bl.LoggerForTypes(reflect.TypeFor[io.Reader](), reflect.TypeFor[*DummyStruct]())
		Expect(lgTypesReversed).NotTo(BeNil())
		Expect(lgTypesReversed.GetHandlerOptions().Level.Level()).To(Equal(slog.LevelError))

		lgTypesFallback := bl.LoggerForTypes(reflect.TypeFor[*bytes.Buffer](), reflect.TypeFor[io.Reader]())
		Expect(lgTypesFallback).NotTo(BeNil())
		Expect(lgTypesFallback.GetHandlerOptions().Level.Level()).To(Equal(slog.LevelError))

		// Test non-struct pointer and non-pointer
		Expect(bl.LoggerForType(reflect.TypeFor[string]())).NotTo(BeNil())
		Expect(bl.LoggerForClass("not a struct")).NotTo(BeNil())

		// Test struct that does not match any configured logger
		Expect(bl.LoggerForClass(&bytes.Buffer{})).NotTo(BeNil())
		Expect(bl.LoggerForTypes(reflect.TypeFor[bytes.Buffer]())).NotTo(BeNil())

		// Test nil type handling
		Expect(bl.LoggerForType(nil)).NotTo(BeNil())
		Expect(bl.LoggerForTypes(nil)).NotTo(BeNil())

		// Test With and WithGroup on logging wrapper
		withLg := lg1.With("key", "val")
		Expect(withLg).NotTo(BeNil())
		withGrp := lg1.WithGroup("testgroup")
		Expect(withGrp).NotTo(BeNil())
	})

	It("supports custom logging handler set via SetLoggingHandler", func() {
		called := false
		bl.SetLoggingHandler(func(w io.Writer, opts *slog.HandlerOptions) slog.Handler {
			called = true
			return slog.NewTextHandler(w, opts)
		})
		Expect(bl.Boot()).NotTo(HaveOccurred())
		Expect(called).To(BeTrue())
	})

	It("supports CUSTOM log handler with custom format", func() {
		cfg := map[string]any{
			"handler": "CUSTOM",
			"format":  "%msg(%v)",
		}
		Expect(bl.LoadConfigAt(cfg, "rays.logging")).NotTo(HaveOccurred())
		Expect(bl.Boot()).NotTo(HaveOccurred())
	})

	It("supports all LogHandler enum values", func() {
		h := boot.LogHandler{}.New()
		for _, enumVal := range []lang.Enum[boot.LogHandler]{
			h.DEFAULT,
			h.JSON,
			h.TEXT,
			h.CUSTOM,
			h.ECS,
			h.GELF,
			h.LOGSTASH,
		} {
			handlerObj := boot.LogHandler{Enum: enumVal}
			fn := handlerObj.Handler("custom-format")
			Expect(fn).NotTo(BeNil())
			handler := fn(io.Discard, nil)
			Expect(handler).NotTo(BeNil())
		}
		// Default / invalid
		invalidHandler := boot.LogHandler{}
		Expect(invalidHandler.Handler()).To(BeNil())
	})

	It("supports all LogLevel enum values", func() {
		l := boot.LogLevel{}.New()
		Expect((&boot.LogLevel{Enum: l.TRACE}).Level()).To(Equal(LevelTrace))
		Expect((&boot.LogLevel{Enum: l.DEBUG}).Level()).To(Equal(LevelDebug))
		Expect((&boot.LogLevel{Enum: l.WARN}).Level()).To(Equal(LevelWarn))
		Expect((&boot.LogLevel{Enum: l.ERROR}).Level()).To(Equal(LevelError))
		Expect((&boot.LogLevel{Enum: l.FATAL}).Level()).To(Equal(LevelFatal))
		Expect((&boot.LogLevel{Enum: l.INFO}).Level()).To(Equal(LevelInfo))
		invalidLevel := boot.LogLevel{}
		Expect(invalidLevel.Level()).To(Equal(LevelInfo))
	})

	It("supports all LogColor enum values", func() {
		lg := logger.NewLogger()
		lg.SetOutput(GinkgoWriter)
		c := boot.LogColor{}.New()
		boot.LogColor{Enum: c.ON}.Apply(lg)
		boot.LogColor{Enum: c.OFF}.Apply(lg)
		boot.LogColor{Enum: c.AUTO}.Apply(lg)
	})

	It("returns error when phase 1 config initialization fails", func() {
		// Calling InitConfig beforehand so Boot()'s InitConfig fails with already initialized
		Expect(bl.InitConfig()).NotTo(HaveOccurred())
		err := bl.Boot()
		Expect(err).To(HaveOccurred())
	})

	It("returns error when logging properties unmarshal fails", func() {
		cfg := map[string]any{
			"source": "invalid_bool",
		}
		Expect(bl.LoadConfigAt(cfg, "rays.logging")).NotTo(HaveOccurred())
		err := bl.Boot()
		Expect(err).To(HaveOccurred())
	})

	It("returns error when configuration file contains malformed YAML during boot", func() {
		malformedFS := fstest.MapFS{
			"application.yaml": &fstest.MapFile{
				Data: []byte("invalid: [yaml: foo"),
			},
		}
		Expect(bl.AddConfigFS(malformedFS)).NotTo(HaveOccurred())
		err := bl.Boot()
		Expect(err).To(HaveOccurred())
	})

	It("returns error when phase 4 config re-initialization fails after Clear", func() {
		fsys := failAfterLoggingInitFS{
			FS: fstest.MapFS{},
			bl: bl,
		}
		Expect(bl.AddConfigFS(fsys)).NotTo(HaveOccurred())
		err := bl.Boot()
		Expect(err).To(HaveOccurred())
	})

	It("can reset properly and restore standard-library default logger", func() {
		Expect(bl.Boot()).NotTo(HaveOccurred())
		bootedDefault := slog.Default()
		bl.Reset()
		Expect(bl.LoggingProperties()).To(BeNil())
		Expect(slog.Default()).NotTo(Equal(bootedDefault))
	})
})
