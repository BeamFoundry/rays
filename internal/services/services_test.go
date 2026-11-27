package services_test

import (
	"github.com/BeamFoundry/rays/internal/boot/logger"
	"github.com/BeamFoundry/rays/internal/boot/scheduler"
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
		Expect(reg.Scheduler).NotTo(BeNil())
	})

	It("allows registering a custom registry", func() {
		customLogger := logger.NewLogger()
		customLogger.SetLogLevel(logger.LevelWarn)
		customScheduler := scheduler.NewScheduler(4)
		services.Register(&services.Registry{
			Logger:    customLogger,
			Scheduler: customScheduler,
		})
		Expect(services.Get().Logger).To(Equal(customLogger))
		Expect(services.Get().Scheduler).To(Equal(customScheduler))
	})

	It("ignores nil registrations", func() {
		regBefore := services.Get()
		services.Register(nil)
		Expect(services.Get()).To(Equal(regBefore))
	})

	It("can be reset to defaults", func() {
		customLogger := logger.NewLogger()
		customScheduler := scheduler.NewScheduler(2)
		services.Register(&services.Registry{
			Logger:    customLogger,
			Scheduler: customScheduler,
		})
		Expect(services.Get().Logger).To(Equal(customLogger))
		Expect(services.Get().Scheduler).To(Equal(customScheduler))

		services.Reset()
		Expect(services.Get().Logger).NotTo(Equal(customLogger))
		Expect(services.Get().Logger).NotTo(BeNil())
		Expect(services.Get().Scheduler).NotTo(Equal(customScheduler))
		Expect(services.Get().Scheduler).NotTo(BeNil())
	})
})
