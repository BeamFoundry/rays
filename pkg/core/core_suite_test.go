package core_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/BeamFoundry/rays/pkg/core"
	. "github.com/BeamFoundry/rays/pkg/testing"
)

func TestCore(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Pkg/Core Suite")
}

type keyValueType string

var keyValue keyValueType = "value"

type BeamWithPrototypeConfiguration struct {
	Configuration
}

func (this *BeamWithPrototypeConfiguration) BeamWithValue(ctx context.Context) *BeamWithValue {
	if value, ok := ctx.Value(keyValue).(string); ok {
		return (&BeamWithValue{}).WithValue(value)
	}
	return &BeamWithValue{}
}

var _ = Describe("Foundry", func() {
	type BeamWithFoundry struct {
		Component
		foundry Foundry[*BeamWithValue] `@:"Inject"`
	}
	It("can use Foundry", func() {
		AddBeamFor(&BeamWithPrototypeConfiguration{})
		AddBeamFor(&BeamWithFoundry{})

		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())

		instance := GetBeamDefaultInstance[*BeamWithFoundry]()

		beam, err := instance.foundry.Forge(context.WithValue(context.Background(), keyValue, "beam1"))
		Expect(err).NotTo(HaveOccurred())
		Expect(beam.GetValue()).To(Equal("beam1"))
	})
})
