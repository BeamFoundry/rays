package autoconfig

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/BeamFoundry/rays/pkg/core"
	. "github.com/BeamFoundry/rays/pkg/lang"
	. "github.com/BeamFoundry/rays/pkg/testing"
	. "github.com/BeamFoundry/rays/pkg/tx"
)

func TestAutoconfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Internal/Autoconfig Suite")
}

type customTransactionManager struct {
	*DefaultTransactionManager
}

var _ = Describe("DefaultConfiguration", func() {
	It("automatically provides TransactionManager Beam when scanning is enabled", func() {
		GetContainer().EnableScanning()
		Expect(GetContainer().Initialize()).To(Succeed())

		txMgr := GetBeamDefaultInstance[TransactionManager]()
		Expect(txMgr).NotTo(BeNil())
		Expect(txMgr.GetContext()).To(Equal(context.Background()))
		Expect(txMgr.Transactional(func() Error {
			Expect(txMgr.IsActive()).To(BeTrue())
			return nil
		})).To(Succeed())
	})

	It("backs off when a custom TransactionManager Beam is already registered", func() {
		custom := &customTransactionManager{
			DefaultTransactionManager: DefaultTransactionManager{}.New(),
		}
		AddBeamFor(custom)
		GetContainer().EnableScanning()
		Expect(GetContainer().Initialize()).To(Succeed())

		txMgr := GetBeamDefaultInstance[TransactionManager]()
		Expect(txMgr).To(BeIdenticalTo(custom))
	})

	It("constructs TransactionManager directly via TransactionConfiguration.TransactionManager", func() {
		cfg := &TransactionConfiguration{}
		mgr := cfg.TransactionManager(struct {
			ConditionalOnMissingBeam
		}{})
		Expect(mgr).NotTo(BeNil())
	})
})
