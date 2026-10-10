package container_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/BeamFoundry/rays/pkg/core"
	. "github.com/BeamFoundry/rays/pkg/lang"
	. "github.com/BeamFoundry/rays/pkg/testing"
)

type BeanWithOptional struct {
	Component
	bean Optional[*BeamWithValue] `@:"Inject"`
}

var _ = Describe("Optional", func() {
	It("can use Optional", func() {
		AddBeamFor((&BeamWithValue{}).WithValue("test"))
		AddBeamFor(&BeanWithOptional{})
		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())

		/*
			beans, err := rays.GetBeans[BeanWithOptional]()
			Expect(err).NotTo(HaveOccurred())
			Expect(beans.Size()).To(Equal(1))
			instance, ok := beans.One().Get()
			Expect(ok).To(BeTrue())
		*/

		instance := GetBeamDefaultInstance[*BeanWithOptional]()
		opt, ok := instance.bean.Get()
		Expect(ok).To(BeTrue())
		Expect(opt.GetValue()).To(Equal("test"))
	})
})
