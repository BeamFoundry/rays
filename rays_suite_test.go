package rays_test

import (
	"testing"

	"github.com/BeamFoundry/rays"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestRays(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Rays Suite")
}

var _ = Describe("ApplicationContext", func() {
	Expect(rays.ApplicationContext()).NotTo(BeNil())
})
