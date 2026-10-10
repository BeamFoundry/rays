package container_test

import (
	"context"
	"reflect"
	_ "unsafe"

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

		name := reflect.TypeOf(GetLogger().Handler()).String()

		Expect(name).To(Equal("*slog.JSONHandler"))
	})
	It("can set a slog/TextHandler", func() {
		Expect(GetContainer().LoadConfigAt(map[string]any{"handler": "TEXT", "level": "DEBUG"}, "rays.logging")).NotTo(HaveOccurred())
		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())

		name := reflect.TypeOf(GetLogger().Handler()).String()
		Expect(name).To(Equal("*slog.TextHandler"))
	})

	It("can set a custom Handler", func() {
		Expect(GetContainer().LoadConfigAt(map[string]any{"handler": "DEFAULT", "level": "DEBUG"}, "rays.logging")).NotTo(HaveOccurred())
		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())

		name := reflect.TypeOf(GetLogger().Handler()).String()
		Expect(name).NotTo(Equal("*slog.TextHandler"))
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
})
