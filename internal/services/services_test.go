package services_test

import (
	"github.com/BeamFoundry/rays/internal/boot/logger"
	"github.com/BeamFoundry/rays/internal/services"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Services", func() {
	AfterEach(func() {
		services.Reset()
	})

	It("returns a non-nil registry by default", func() {
		reg := services.Get()
		Expect(reg).NotTo(BeNil())
		Expect(reg.Logger).NotTo(BeNil())
	})

	It("allows registering a custom registry", func() {
		customLogger := logger.NewLogger()
		customLogger.SetLogLevel(logger.LevelWarn)
		services.Register(&services.Registry{
			Logger: customLogger,
		})
		Expect(services.Get().Logger).To(Equal(customLogger))
	})

	It("ignores nil registrations", func() {
		regBefore := services.Get()
		services.Register(nil)
		Expect(services.Get()).To(Equal(regBefore))
	})

	It("can be reset to defaults", func() {
		customLogger := logger.NewLogger()
		services.Register(&services.Registry{
			Logger: customLogger,
		})
		Expect(services.Get().Logger).To(Equal(customLogger))

		services.Reset()
		Expect(services.Get().Logger).NotTo(Equal(customLogger))
		Expect(services.Get().Logger).NotTo(BeNil())
	})
})
