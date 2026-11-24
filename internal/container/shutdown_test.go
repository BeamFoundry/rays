package container_test

import (
	"sync"

	"github.com/BeamFoundry/rays/internal/container"
	. "github.com/BeamFoundry/rays/pkg/core"
	. "github.com/BeamFoundry/rays/pkg/lang"
	. "github.com/BeamFoundry/rays/pkg/testing"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type preDestroyBeam struct {
	Component
	IgnoreScanning
	recorded  *[]string
	err       Error
	destroyed bool
}

func (this *preDestroyBeam) PreDestroy() Error {
	this.destroyed = true
	if this.recorded != nil && this.err != nil {
		*this.recorded = append(*this.recorded, this.err.Message())
	}
	return this.err
}

type preDestroyProtoBeam struct {
	Component
	Prototype
	IgnoreScanning
	destroyed bool
}

func (this *preDestroyProtoBeam) PreDestroy() Error {
	this.destroyed = true
	return nil
}

type runnerWithPreDestroy struct {
	Component
	IgnoreScanning
	ran       bool
	destroyed bool
}

func (this *runnerWithPreDestroy) Run(wg *sync.WaitGroup) {
	wg.Go(func() {
		this.ran = true
	})
}

func (this *runnerWithPreDestroy) PreDestroy() Error {
	this.destroyed = true
	return nil
}

type ifaceOnly interface {
	DoWork()
}

type concreteWithPreDestroy struct {
	destroyed bool
}

func (this *concreteWithPreDestroy) DoWork() {}

func (this *concreteWithPreDestroy) PreDestroy() Error {
	this.destroyed = true
	return nil
}

type ifaceConfig struct {
	Configuration
	IgnoreScanning
}

func (this *ifaceConfig) ProvideIface() ifaceOnly {
	return &concreteWithPreDestroy{}
}

var _ = Describe("Container Shutdown", func() {
	It("invokes PreDestroy on initialized singletons, skips prototypes and uninitialized beams, and returns first recorded error", func() {
		var recorded []string
		first := &preDestroyBeam{err: NewError("ShutdownError", "first failure"), recorded: &recorded}
		second := &preDestroyBeam{err: NewError("ShutdownError", "second failure"), recorded: &recorded}
		proto := &preDestroyProtoBeam{}
		uninit := &preDestroyBeam{}

		_, err := GetContainer().AddBeamFor(first)
		Expect(err).NotTo(HaveOccurred())
		_, err = GetContainer().AddBeamFor(second)
		Expect(err).NotTo(HaveOccurred())
		_, err = GetContainer().AddBeamFor(proto)
		Expect(err).NotTo(HaveOccurred())

		Expect(GetContainer().Initialize()).NotTo(HaveOccurred())

		// Register an uninitialized beam after Initialize to verify uninitialized instances are skipped.
		uninitBeam, err := container.GetContainer().Registry.AddBeamFor(uninit)
		Expect(err).NotTo(HaveOccurred())
		Expect(uninitBeam).NotTo(BeNil())

		shutdownErr := GetContainer().Shutdown()
		Expect(shutdownErr).To(HaveOccurred())
		Expect(first.destroyed).To(BeTrue())
		Expect(second.destroyed).To(BeTrue())
		Expect(recorded).To(HaveLen(2))
		Expect(shutdownErr.Message()).To(Equal(recorded[0]))
		Expect(proto.destroyed).To(BeFalse())
		Expect(uninit.destroyed).To(BeFalse())
	})

	It("invokes PreDestroy when declared Beam.Type() is an interface not embedding BeamPreDestroy", func() {
		cfg := &ifaceConfig{}
		_, err := GetContainer().AddBeamFor(cfg)
		Expect(err).NotTo(HaveOccurred())
		Expect(GetContainer().Initialize()).NotTo(HaveOccurred())

		provided := GetBeamDefaultInstance[ifaceOnly]()
		concrete, ok := provided.(*concreteWithPreDestroy)
		Expect(ok).To(BeTrue())

		Expect(GetContainer().Shutdown()).NotTo(HaveOccurred())
		Expect(concrete.destroyed).To(BeTrue())
	})

	It("invokes Shutdown automatically after Run completes", func() {
		runner := &runnerWithPreDestroy{}
		_, err := GetContainer().AddBeamFor(runner)
		Expect(err).NotTo(HaveOccurred())
		Expect(GetContainer().Initialize()).NotTo(HaveOccurred())

		GetContainer().Run()
		Expect(runner.ran).To(BeTrue())
		Expect(runner.destroyed).To(BeTrue())
	})
})
