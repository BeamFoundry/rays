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
	err       Error
	destroyed bool
}

func (this *preDestroyBeam) PreDestroy() Error {
	this.destroyed = true
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

var _ = Describe("Container Shutdown", func() {
	It("invokes PreDestroy on initialized singletons, skips prototypes and uninitialized beams, and returns first error", func() {
		first := &preDestroyBeam{err: NewError("ShutdownError", "first failure")}
		second := &preDestroyBeam{err: NewError("ShutdownError", "second failure")}
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
		Expect(shutdownErr.Message()).To(Equal("first failure"))
		Expect(first.destroyed).To(BeTrue())
		Expect(second.destroyed).To(BeTrue())
		Expect(proto.destroyed).To(BeFalse())
		Expect(uninit.destroyed).To(BeFalse())
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
