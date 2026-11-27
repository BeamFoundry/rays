package container_test

import (
	"reflect"
	"sync/atomic"
	"time"

	"github.com/BeamFoundry/rays/internal/container"
	. "github.com/BeamFoundry/rays/pkg/core"
	. "github.com/BeamFoundry/rays/pkg/testing"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type scheduledDep struct {
	Component
	IgnoreScanning
	label string
}

type scheduledService struct {
	Service
	IgnoreScanning
	injected       atomic.Value
	rateCount      atomic.Int32
	delayCount     atomic.Int32
	cronCount      atomic.Int32
	launchCount    atomic.Int32
	launchDelCount atomic.Int32
}

func (this *scheduledService) RateJob(
	args struct {
		Scheduled `fixedRate:"20ms"`
		dep       *scheduledDep `@:"Inject"`
	},
) {
	this.injected.Store(args.dep.label)
	this.rateCount.Add(1)
}

func (this *scheduledService) DelayJob(
	_ *struct {
		Scheduled `initialDelay:"15ms" fixedDelay:"20ms"`
	},
) {
	this.delayCount.Add(1)
}

func (this *scheduledService) CronJob(
	_ struct {
		Scheduled `cron:"0 0 0 1 1 *"`
	},
) {
	this.cronCount.Add(1)
}

func (this *scheduledService) LaunchJob(
	_ struct {
		Scheduled
	},
) {
	this.launchCount.Add(1)
}

func (this *scheduledService) LaunchDelayedJob(
	_ struct {
		Scheduled `initialDelay:"15ms"`
	},
) {
	this.launchDelCount.Add(1)
}

// Non-scheduled methods that must be skipped cleanly.
func (this *scheduledService) IgnoredNoArgs() {}

func (this *scheduledService) IgnoredWithReturn(
	_ struct {
		Scheduled `fixedRate:"10ms"`
	},
) int {
	return 1
}

func (this *scheduledService) IgnoredNonAnon(_ string)           {}
func (this *scheduledService) IgnoredAnonWithoutTag(_ struct{})  {}
func (this *scheduledService) IgnoredTwoArgs(_ struct{}, _ bool) {}

type scheduledPrototypeBeam struct {
	Component
	Prototype
	IgnoreScanning
	ran atomic.Bool
}

func (this *scheduledPrototypeBeam) ProtoJob(
	_ struct {
		Scheduled `fixedRate:"10ms"`
	},
) {
	this.ran.Store(true)
}

type plainNonStereotypeBeam struct {
	ran atomic.Bool
}

func (this *plainNonStereotypeBeam) Job(
	_ struct {
		Scheduled `fixedRate:"10ms"`
	},
) {
	this.ran.Store(true)
}

type scheduledChildConfig struct {
	Configuration
	IgnoreScanning
}

func (this *scheduledChildConfig) ProvideChild() *scheduledService {
	return &scheduledService{}
}

type unsupportedTagScheduledBeam struct {
	Service
	IgnoreScanning
}

func (this *unsupportedTagScheduledBeam) BadJob(
	_ struct {
		Scheduled `fixedRate:"10ms" extra:"nope"`
	},
) {
}

type multipleTagScheduledBeam struct {
	Service
	IgnoreScanning
}

func (this *multipleTagScheduledBeam) BadJob(
	_ struct {
		Scheduled `fixedRate:"10ms" cron:"* * * * * *"`
	},
) {
}

type missingDep struct {
	Component
	IgnoreScanning
}

type invalidInjectScheduledBeam struct {
	Service
	IgnoreScanning
}

func (this *invalidInjectScheduledBeam) BadJob(
	_ struct {
		Scheduled `fixedRate:"10ms"`
		missing   *missingDep `@:"Inject"`
	},
) {
}

type invalidParamScheduledBeam struct {
	Service
	IgnoreScanning
}

func (this *invalidParamScheduledBeam) BadJob(
	_ struct {
		Scheduled `fixedRate:"not-a-duration"`
	},
) {
}

type invalidInitialDelayScheduledBeam struct {
	Service
	IgnoreScanning
}

func (this *invalidInitialDelayScheduledBeam) BadJob(
	_ struct {
		Scheduled `fixedRate:"10ms" initialDelay:"not-a-duration"`
	},
) {
}

type invalidLaunchDelayScheduledBeam struct {
	Service
	IgnoreScanning
}

func (this *invalidLaunchDelayScheduledBeam) BadJob(
	_ struct {
		Scheduled `initialDelay:"not-a-duration"`
	},
) {
}

var _ = Describe("Container Scheduled Tasks", func() {
	It("discovers, injects, and runs Scheduled and LaunchTask methods only after Run is invoked, skipping non-eligible beams and methods", func() {
		dep := &scheduledDep{label: "injected-ok"}
		svc := &scheduledService{}
		proto := &scheduledPrototypeBeam{}
		plain := &plainNonStereotypeBeam{}
		childCfg := &scheduledChildConfig{}

		_, err := GetContainer().AddBeamFor(dep)
		Expect(err).NotTo(HaveOccurred())
		_, err = GetContainer().AddBeamFor(svc)
		Expect(err).NotTo(HaveOccurred())
		_, err = GetContainer().AddBeamFor(proto)
		Expect(err).NotTo(HaveOccurred())
		_, err = GetContainer().AddBeamFor(plain)
		Expect(err).NotTo(HaveOccurred())
		_, err = GetContainer().AddBeamFor(childCfg)
		Expect(err).NotTo(HaveOccurred())

		Expect(GetContainer().Initialize()).NotTo(HaveOccurred())

		// Scheduled tasks must NOT execute before Run() starts the scheduler.
		Consistently(svc.rateCount.Load, 40*time.Millisecond, 5*time.Millisecond).Should(Equal(int32(0)))
		Expect(svc.delayCount.Load()).To(Equal(int32(0)))
		Expect(svc.launchCount.Load()).To(Equal(int32(0)))
		Expect(svc.launchDelCount.Load()).To(Equal(int32(0)))

		runDone := make(chan struct{})
		go func() {
			GetContainer().Run()
			close(runDone)
		}()

		Eventually(svc.rateCount.Load, time.Second, 5*time.Millisecond).Should(BeNumerically(">=", 2))
		Eventually(svc.delayCount.Load, time.Second, 5*time.Millisecond).Should(BeNumerically(">=", 2))
		Eventually(svc.launchCount.Load, time.Second, 5*time.Millisecond).Should(Equal(int32(1)))
		Eventually(svc.launchDelCount.Load, time.Second, 5*time.Millisecond).Should(Equal(int32(1)))
		Expect(svc.injected.Load()).To(Equal("injected-ok"))
		Expect(proto.ran.Load()).To(BeFalse())
		Expect(plain.ran.Load()).To(BeFalse())

		Expect(GetContainer().Shutdown()).NotTo(HaveOccurred())
		Eventually(runDone, time.Second).Should(BeClosed())
	})

	It("returns an error when Scheduled contains an unsupported struct tag", func() {
		_, err := GetContainer().AddBeamFor(&unsupportedTagScheduledBeam{})
		Expect(err).NotTo(HaveOccurred())
		Expect(GetContainer().Initialize()).To(HaveOccurred())
	})

	It("returns an error on malformed Scheduled struct tag syntax", func() {
		Expect(container.ParseScheduledTag(reflect.StructTag("fixedRate"))).To(HaveOccurred())
		Expect(container.ParseScheduledTag(reflect.StructTag(`fixedRate:"unclosed`))).To(HaveOccurred())
	})

	It("returns an error when Scheduled tag specifies multiple schedule types", func() {
		_, err := GetContainer().AddBeamFor(&multipleTagScheduledBeam{})
		Expect(err).NotTo(HaveOccurred())
		Expect(GetContainer().Initialize()).To(HaveOccurred())
	})

	It("returns an error when anonymous struct dependency injection fails", func() {
		_, err := GetContainer().AddBeamFor(&invalidInjectScheduledBeam{})
		Expect(err).NotTo(HaveOccurred())
		Expect(GetContainer().Initialize()).To(HaveOccurred())
	})

	It("returns an error when Scheduled parameter is invalid", func() {
		_, err := GetContainer().AddBeamFor(&invalidParamScheduledBeam{})
		Expect(err).NotTo(HaveOccurred())
		Expect(GetContainer().Initialize()).To(HaveOccurred())
	})

	It("returns an error when Scheduled initialDelay is invalid", func() {
		_, err := GetContainer().AddBeamFor(&invalidInitialDelayScheduledBeam{})
		Expect(err).NotTo(HaveOccurred())
		Expect(GetContainer().Initialize()).To(HaveOccurred())
	})

	It("returns an error when bare Scheduled LaunchTask initialDelay is invalid", func() {
		_, err := GetContainer().AddBeamFor(&invalidLaunchDelayScheduledBeam{})
		Expect(err).NotTo(HaveOccurred())
		Expect(GetContainer().Initialize()).To(HaveOccurred())
	})
})
