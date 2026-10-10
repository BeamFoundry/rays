package container_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/BeamFoundry/rays/pkg/core"
	. "github.com/BeamFoundry/rays/pkg/testing"

	_ "github.com/BeamFoundry/rays/internal/container"
)

type LoggingConfig struct {
	level string
}

var _ = Describe("Environment", func() {
	It("can be obtained", func() {
		Expect(GetContainer().SetActiveProfiles([]string{"test"})).NotTo(HaveOccurred())
		GetContainer().EnableScanning()
		Expect(GetContainer().Initialize()).NotTo(HaveOccurred())

		env := GetBeamDefaultInstance[Environment]()
		Expect(env.ActiveProfiles()).To(ContainElement("test"))

		level, ok := env.ConfigValue("rays.logging.level")
		Expect(ok).To(BeTrue())
		Expect(level).To(Equal("DEBUG"))

		props := &LoggingConfig{}
		Expect(env.ConfigUnmarshall("rays.logging", props)).NotTo(HaveOccurred())
		Expect(props.level).To(Equal("DEBUG"))
	})
})
