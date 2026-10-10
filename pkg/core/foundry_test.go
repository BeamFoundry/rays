package core_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/BeamFoundry/rays/pkg/core"
	. "github.com/BeamFoundry/rays/pkg/lang"
)

var _ = Describe("Foundry", func() {
	It("can initialize a valid Foundry", func() {
		s := "string"
		furnace := func(_ context.Context) (any, Error) {
			return &s, nil
		}

		foundry := NewFoundry[*string](furnace)
		forged, err := foundry.Forge(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(*forged).To(Equal(s))
	})
	It("can error on empty 'furnace'", func() {
		foundry := &Foundry[*string]{}
		_, err := foundry.Forge(context.Background())
		Expect(err).To(HaveOccurred())
	})
	It("can return an Error", func() {
		furnace := func(_ context.Context) (any, Error) {
			return nil, NewError("TestError", "Test")
		}
		foundry := NewFoundry[*string](furnace)
		_, err := foundry.Forge(context.Background())
		Expect(err).To(HaveOccurred())
	})
	It("can Error on wrong return type", func() {
		s := "string"
		furnace := func(_ context.Context) (any, Error) {
			return s, nil
		}

		foundry := NewFoundry[*string](furnace)
		_, err := foundry.Forge(context.Background())
		Expect(err).To(HaveOccurred())
	})
})
