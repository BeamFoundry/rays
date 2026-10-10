package testing_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/BeamFoundry/rays/pkg/testing"
)

func TestTesting(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Pkg/Testing Suite")
}

var _ = Describe("Testing", func() {
	It("can declare a beam and get it back", func() {
		AddBeamFor((&BeamWithValue{}).WithValue("test"))
		Expect(GetBeam[*BeamWithValue]()).NotTo(BeNil())
		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())
		Expect(GetBeamDefaultInstance[*BeamWithValue]()).NotTo(BeNil())
		Expect(GetBeamScopedInstance[*BeamWithValue]()).NotTo(BeNil())
		GetBeamScopedInstance[*BeamWithValue]().SetValue("foo")
		Expect(GetBeamScopedInstance[*BeamWithValue]().GetValue()).To(Equal("foo"))
	})
	It("can not more than one Beam of a type", func() {
		AddBeamFor((&BeamWithValue{}).WithValue("test"))
		AddBeamFor((&BeamWithValue{}).WithValue("test"))
		Expect(GetBeam[*BeamWithValue]()).To(BeNil())
	})
	It("can create a new Beam", func() {
		Expect(NewBeamFor(&BeamWithValue{})).NotTo(BeNil())
	})
	It("can create a injected Beam", func() {
		AddBeamFor((&BeamWithValue{}).WithValue("test"))
		AddBeamFor(&BeamWithInject{})
		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())
		beam := GetBeamScopedInstance[*BeamWithInject]()
		Expect(beam.GetBeam().GetValue()).To(Equal("test"))
		Expect(beam.WithBeam((&BeamWithValue{}).WithValue("foo")).GetBeam().GetValue()).To(Equal("foo"))
		beam.SetBeam((&BeamWithValue{}).WithValue("bar"))
		Expect(beam.GetBeam().GetValue()).To(Equal("bar"))
	})
	It("can create a Beam with Foundry", func() {
		AddBeamFor(&BeamWithValue{})
		AddBeamFor(&BeamWithFoundry[*BeamWithValue]{})
		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())
		Expect(GetBeamScopedInstance[*BeamWithFoundry[*BeamWithValue]]().GetFoundry()).NotTo(BeNil())
	})
	It("can use ValErr", func() {
		f := func() (string, error) {
			return "foo", nil
		}
		Expect(ValErr(f())).To(Equal("foo"))
	})
})
