package container_test

import (
	"flag"
	"io"
	"log/slog"
	"reflect"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/BeamFoundry/rays/internal/container"
	. "github.com/BeamFoundry/rays/pkg/testing"
)

func TestContainer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Internal/Container Suite")
}

type TestStruct struct{}

var _ = Describe("Container", func() {
	It("can detect a testing environment", func() {
		Expect(flag.Lookup("test.v")).NotTo(BeNil())
	})
	It("can get a beam parent", func() {
		AddBeamFor(&BeamConfiguration{})
		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())
		parent := GetBeam[*BeamConfiguration]()
		Expect(parent).NotTo(BeNil())
		beam := GetBeam[*BeamWithValue]()
		Expect(beam).NotTo(BeNil())
		instance := container.GetContainer().GetInstance(beam)
		Expect(instance).NotTo(BeNil())
	})
	It("can set logging handler", func() {
		GetContainer().SetLoggingHandler(func(writer io.Writer, options *slog.HandlerOptions) slog.Handler {
			return slog.NewJSONHandler(writer, options)
		})
		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())
	})
	It("can load a non default logging handler", func() {
		Expect(GetContainer().LoadConfigAt(map[string]any{"handler": "JSON"}, "rays.logging")).NotTo(HaveOccurred())
		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())
	})
	It("can retrieve a default logger", func() {
		config := map[string]any{
			"level": "DEBUG",
			"loggers": map[string]any{
				"github.com/BeamFoundry/rays/internal/container":      "DEBUG",
				"github.com/BeamFoundry/rays/internal/container_test": "DEBUG",
			},
		}
		Expect(GetContainer().LoadConfigAt(config, "rays.logging")).NotTo(HaveOccurred())
		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())
		Expect(container.GetContainer().Logger()).NotTo(BeNil())
		Expect(container.GetContainer().LoggerForClass(&TestStruct{})).NotTo(BeNil())
		Expect(container.GetContainer().LoggerForClass(&BeamWithValue{})).NotTo(BeNil())
		Expect(container.GetContainer().LoggerForClass("")).NotTo(BeNil())
		Expect(container.GetContainer().LoggerForType(reflect.TypeFor[*TestStruct]())).NotTo(BeNil())
		Expect(container.GetContainer().LoggerForType(reflect.TypeFor[string]())).NotTo(BeNil())
	})
})
