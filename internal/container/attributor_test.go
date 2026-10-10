package container_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/BeamFoundry/rays/pkg/core"
	. "github.com/BeamFoundry/rays/pkg/testing"
)

var _ = Describe("Attribuer", func() {
	It("can define attributes", func() {
		type PrimaryBeam struct {
			Component
			Primary
		}

		AddBeamFor(&PrimaryBeam{})
		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())

		beam := GetBeam[*PrimaryBeam]()
		attributes := beam.Attributes()

		Expect(attributes).To(HaveKey("Primary"))
	})

	It("can reject invalid tag attribute", func() {
		type BeamConfigurationWrongQualifierTag struct {
			Component
			Qualifier
		}

		AddBeamFor(&BeamConfigurationWrongQualifierTag{})
		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())

		beam := GetBeam[*BeamConfigurationWrongQualifierTag]()
		Expect(beam.Attributes()["Qualifier"]).To(Equal("beamConfigurationWrongQualifierTag"))
	})
})
