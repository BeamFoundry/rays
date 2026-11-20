package tx_test

import (
	"context"
	"sync"
	"testing"

	"github.com/timandy/routine"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	_ "github.com/BeamFoundry/rays"
	. "github.com/BeamFoundry/rays/pkg/core"
	. "github.com/BeamFoundry/rays/pkg/lang"
	. "github.com/BeamFoundry/rays/pkg/testing"
	. "github.com/BeamFoundry/rays/pkg/tx"
)

func TestTx(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Pkg/Tx Suite")
}

type ctxKey string

type mockResource struct {
	//nolint:containedctx // Stores bound context for test verification
	ctx        context.Context
	name       string
	started    bool
	committed  bool
	rolledBack bool
	ended      bool
}

type mockTransactional struct {
	txManager   TransactionManager
	startErr    Error
	commitErr   Error
	rollbackErr Error
	local       routine.ThreadLocal[*mockResource]
	lastSession routine.ThreadLocal[*mockResource]
	name        string
	reentrantRb bool
}

var _ Transactional[mockResource] = &mockTransactional{}

func newMockTransactional(name string, txManager TransactionManager) *mockTransactional {
	return &mockTransactional{
		name:        name,
		txManager:   txManager,
		local:       routine.NewThreadLocal[*mockResource](),
		lastSession: routine.NewThreadLocal[*mockResource](),
	}
}

func (this *mockTransactional) Start() Error {
	if this.startErr != nil {
		return this.startErr
	}
	res := &mockResource{
		name:    this.name,
		ctx:     this.txManager.GetContext(),
		started: true,
	}
	this.local.Set(res)
	this.lastSession.Set(res)
	return nil
}

func (this *mockTransactional) Commit() Error {
	if this.commitErr != nil {
		return this.commitErr
	}
	if res := this.local.Get(); res != nil {
		res.committed = true
	}
	return nil
}

func (this *mockTransactional) Rollback() Error {
	if this.reentrantRb && this.txManager != nil {
		this.txManager.SetError(NewError("ReentrantError", "reentrant SetError during Rollback"))
	}
	if this.rollbackErr != nil {
		return this.rollbackErr
	}
	if res := this.local.Get(); res != nil {
		res.rolledBack = true
	}
	return nil
}

func (this *mockTransactional) End() {
	if res := this.local.Get(); res != nil {
		res.ended = true
	}
	this.local.Remove()
}

func (this *mockTransactional) SetError(err Error) {
	if this.txManager != nil {
		this.txManager.SetError(err)
	}
}

func (this *mockTransactional) Error() Error {
	if this.txManager != nil {
		return this.txManager.Error()
	}
	return nil
}

func (this *mockTransactional) Get() *mockResource {
	if res := this.local.Get(); res != nil {
		return res
	}
	if this.txManager != nil && this.txManager.IsActive() {
		if err := this.txManager.Enlist(this); err != nil {
			return &mockResource{
				name: this.name,
				ctx:  this.txManager.GetContext(),
			}
		}
		return this.local.Get()
	}
	return &mockResource{
		name: this.name,
		ctx:  this.txManager.GetContext(),
	}
}

var _ = Describe("DefaultTransactionManager", func() {
	var (
		txMgr   TransactionManager
		sqlTx   *mockTransactional
		mongoTx *mockTransactional
	)

	BeforeEach(func() {
		GetContainer().EnableScanning()
		Expect(GetContainer().Initialize()).To(Succeed())
		txMgr = GetBeamDefaultInstance[TransactionManager]()
		sqlTx = newMockTransactional("sql", txMgr)
		mongoTx = newMockTransactional("mongo", txMgr)
	})

	AfterEach(func() {
		txMgr.End()
		txMgr.ClearContext()
	})

	Context("Goroutine-Local Context Management", func() {
		It("defaults to context.Background() and supports SetContext, GetContext, and ClearContext", func() {
			Expect(txMgr.GetContext()).To(Equal(context.Background()))

			customCtx := context.WithValue(context.Background(), ctxKey("reqID"), "12345")
			txMgr.SetContext(customCtx)
			Expect(txMgr.GetContext()).To(Equal(customCtx))
			Expect(txMgr.GetContext().Value(ctxKey("reqID"))).To(Equal("12345"))

			// Setting nil falls back to context.Background()
			//nolint:staticcheck // Intentionally testing nil context fallback
			txMgr.SetContext(nil)
			Expect(txMgr.GetContext()).To(Equal(context.Background()))

			txMgr.SetContext(customCtx)
			txMgr.ClearContext()
			Expect(txMgr.GetContext()).To(Equal(context.Background()))
		})

		It("propagates bound context to Transactional[T] providers in both non-transactional and transactional modes", func() {
			customCtx := context.WithValue(context.Background(), ctxKey("tenant"), "acme")
			txMgr.SetContext(customCtx)

			// Non-transactional Get()
			res := sqlTx.Get()
			Expect(res.started).To(BeFalse())
			Expect(res.ctx.Value(ctxKey("tenant"))).To(Equal("acme"))

			// Transactional Get()
			Expect(txMgr.Start()).To(Succeed())
			txRes := sqlTx.Get()
			Expect(txRes.started).To(BeTrue())
			Expect(txRes.ctx.Value(ctxKey("tenant"))).To(Equal("acme"))
			Expect(txMgr.Commit()).To(Succeed())
			txMgr.End()
			Expect(txRes.committed).To(BeTrue())
			Expect(txRes.ended).To(BeTrue())
		})
	})

	Context("Coordinated Multi-Provider Transactions", func() {
		It("commits all lazily enlisted Transactional[T] providers on success", func() {
			err := txMgr.Transactional(func() Error {
				s := sqlTx.Get()
				m := mongoTx.Get()
				Expect(s.started).To(BeTrue())
				Expect(m.started).To(BeTrue())
				// Second Get() reuses existing enlisted session
				Expect(sqlTx.Get()).To(BeIdenticalTo(s))
				Expect(txMgr.Enlist(sqlTx)).To(Succeed())
				return nil
			})
			Expect(err).To(Succeed())
			Expect(sqlTx.lastSession.Get().committed).To(BeTrue())
			Expect(sqlTx.lastSession.Get().rolledBack).To(BeFalse())
			Expect(sqlTx.lastSession.Get().ended).To(BeTrue())
			Expect(mongoTx.lastSession.Get().committed).To(BeTrue())
			Expect(mongoTx.lastSession.Get().rolledBack).To(BeFalse())
			Expect(mongoTx.lastSession.Get().ended).To(BeTrue())
			Expect(txMgr.IsActive()).To(BeFalse())
		})

		It("immediately rolls back all participating providers when a repository reports SetError", func() {
			repoErr := NewError("DuplicateKeyError", "unique constraint violated")

			err := txMgr.Transactional(func() Error {
				s := sqlTx.Get()
				m := mongoTx.Get()
				Expect(s.started).To(BeTrue())
				Expect(m.started).To(BeTrue())

				// Simulate Mongo repository encountering an error and reporting it to mongoTx
				mongoTx.SetError(repoErr)

				// Both providers must already be rolled back immediately!
				Expect(s.rolledBack).To(BeTrue())
				Expect(m.rolledBack).To(BeTrue())
				Expect(mongoTx.Error()).To(Equal(repoErr))
				Expect(sqlTx.Error()).To(Equal(repoErr))

				// Even if the closure ignores the error and returns nil, Transactional must fail with repoErr
				return nil
			})

			Expect(err).To(Equal(repoErr))
			Expect(sqlTx.lastSession.Get().committed).To(BeFalse())
			Expect(sqlTx.lastSession.Get().rolledBack).To(BeTrue())
			Expect(sqlTx.lastSession.Get().ended).To(BeTrue())
			Expect(mongoTx.lastSession.Get().committed).To(BeFalse())
			Expect(mongoTx.lastSession.Get().rolledBack).To(BeTrue())
			Expect(mongoTx.lastSession.Get().ended).To(BeTrue())
		})

		It("rolls back all participating providers when Transactional closure returns an Error or panics", func() {
			actionErr := NewError("BusinessError", "insufficient funds")
			err := txMgr.Transactional(func() Error {
				_ = sqlTx.Get()
				_ = mongoTx.Get()
				return actionErr
			})
			Expect(err).To(Equal(actionErr))
			Expect(sqlTx.lastSession.Get().rolledBack).To(BeTrue())
			Expect(mongoTx.lastSession.Get().rolledBack).To(BeTrue())

			// Panic inside closure
			panicErr := txMgr.Transactional(func() Error {
				_ = sqlTx.Get()
				RaiseErr(NewError("PanicError", "unexpected failure"))
				return nil
			})
			Expect(panicErr).To(HaveOccurred())
			Expect(panicErr.Name()).To(Equal("PanicError"))
			Expect(sqlTx.lastSession.Get().rolledBack).To(BeTrue())
		})

		It("rolls back already-enlisted providers if a subsequent provider fails to Start during Enlist", func() {
			mongoTx.startErr = NewError("MongoStartError", "session pool exhausted")

			Expect(txMgr.Start()).To(Succeed())
			s := sqlTx.Get()
			Expect(s.started).To(BeTrue())

			// Enlisting mongoTx fails and must immediately roll back sqlTx
			enlistErr := txMgr.Enlist(mongoTx)
			Expect(enlistErr).To(Equal(mongoTx.startErr))
			Expect(s.rolledBack).To(BeTrue())
			Expect(txMgr.Error()).To(Equal(mongoTx.startErr))

			// Subsequent Enlist or Commit returns the recorded error
			thirdTx := newMockTransactional("redis", txMgr)
			Expect(txMgr.Enlist(thirdTx)).To(Equal(mongoTx.startErr))
			Expect(txMgr.Commit()).To(Equal(mongoTx.startErr))

			txMgr.End()
			Expect(s.ended).To(BeTrue())
		})

		It("rolls back the failing provider and remaining uncommitted providers if a provider fails during Commit", func() {
			firstTx := newMockTransactional("first", txMgr)
			sqlTx.commitErr = NewError("SqlCommitError", "deadlock on commit")

			err := txMgr.Transactional(func() Error {
				_ = firstTx.Get()
				_ = sqlTx.Get()
				_ = mongoTx.Get()
				return nil
			})
			Expect(err).To(Equal(sqlTx.commitErr))
			Expect(firstTx.lastSession.Get().committed).To(BeTrue())
			Expect(firstTx.lastSession.Get().rolledBack).To(BeFalse())
			Expect(sqlTx.lastSession.Get().committed).To(BeFalse())
			Expect(sqlTx.lastSession.Get().rolledBack).To(BeTrue())
			Expect(mongoTx.lastSession.Get().committed).To(BeFalse())
			Expect(mongoTx.lastSession.Get().rolledBack).To(BeTrue())
			Expect(mongoTx.lastSession.Get().ended).To(BeTrue())
		})

		It("automatically rolls back uncommitted transactions when End() is called directly", func() {
			Expect(txMgr.Start()).To(Succeed())
			s := sqlTx.Get()
			Expect(s.started).To(BeTrue())

			// Calling End() without Commit() or Rollback() must roll back s
			txMgr.End()
			Expect(s.rolledBack).To(BeTrue())
			Expect(s.ended).To(BeTrue())
			Expect(txMgr.IsActive()).To(BeFalse())
		})
	})

	Context("Edge Cases and Error Handling", func() {
		It("handles nil logger, inactive state errors, double Start, double Commit/Rollback, and provider rollback errors", func() {
			nilLoggerMgr := DefaultTransactionManager{}.New()
			Expect(nilLoggerMgr.IsActive()).To(BeFalse())
			Expect(nilLoggerMgr.Error()).To(Succeed())

			// Operations when no transaction is active
			Expect(nilLoggerMgr.Enlist(sqlTx)).To(HaveOccurred())
			Expect(nilLoggerMgr.Commit()).To(HaveOccurred())
			Expect(nilLoggerMgr.Rollback()).To(HaveOccurred())
			nilLoggerMgr.SetError(NewError("Ignored", "no active tx"))
			nilLoggerMgr.SetError(nil)
			nilLoggerMgr.End()

			// Start and double Start
			Expect(txMgr.Start()).To(Succeed())
			Expect(txMgr.Start()).To(HaveOccurred())
			Expect(txMgr.Transactional(func() Error { return nil })).To(HaveOccurred())

			// Enlist nil provider
			Expect(txMgr.Enlist(nil)).To(HaveOccurred())
			txMgr.SetError(nil)

			// Commit then double Commit / Rollback / Enlist on finalized tx
			_ = sqlTx.Get()
			Expect(txMgr.Commit()).To(Succeed())
			Expect(txMgr.Commit()).To(HaveOccurred())
			Expect(txMgr.Rollback()).To(HaveOccurred())
			Expect(txMgr.Enlist(mongoTx)).To(HaveOccurred())
			txMgr.End()

			// Manual Rollback then Commit / Enlist / idempotent Rollback
			Expect(txMgr.Start()).To(Succeed())
			_ = sqlTx.Get()
			Expect(txMgr.Rollback()).To(Succeed())
			Expect(txMgr.Rollback()).To(Succeed())
			Expect(txMgr.Commit()).To(HaveOccurred())
			Expect(txMgr.Enlist(mongoTx)).To(HaveOccurred())
			txMgr.End()

			// Provider Rollback returning an error
			sqlTx.rollbackErr = NewError("RollbackFailed", "connection lost")
			Expect(txMgr.Start()).To(Succeed())
			_ = sqlTx.Get()
			rbErr := txMgr.Rollback()
			Expect(rbErr).To(Equal(sqlTx.rollbackErr))
			Expect(txMgr.Error()).To(Equal(sqlTx.rollbackErr))
			txMgr.End()

			// Re-entrant SetError during Rollback does not infinitely recurse
			sqlTx.rollbackErr = nil
			sqlTx.reentrantRb = true
			Expect(txMgr.Start()).To(Succeed())
			_ = sqlTx.Get()
			Expect(txMgr.Rollback()).To(Succeed())
			Expect(txMgr.Error().Name()).To(Equal("ReentrantError"))
			txMgr.End()

			// Commit failure when remaining provider also fails Rollback
			sqlTx.rollbackErr = nil
			sqlTx.reentrantRb = false
			sqlTx.commitErr = NewError("CommitFailed", "disk full")
			mongoTx.rollbackErr = NewError("MongoRbFailed", "network error")
			Expect(txMgr.Start()).To(Succeed())
			_ = sqlTx.Get()
			_ = mongoTx.Get()
			Expect(txMgr.Commit()).To(Equal(sqlTx.commitErr))
			txMgr.End()
		})

		It("isolates transaction state and context across concurrent goroutines", func() {
			const workers = 10
			var wg sync.WaitGroup
			wg.Add(workers)

			for i := range workers {
				go func(id int) {
					defer GinkgoRecover()
					defer wg.Done()

					ctx := context.WithValue(context.Background(), ctxKey("worker"), id)
					txMgr.SetContext(ctx)
					defer txMgr.ClearContext()

					err := txMgr.Transactional(func() Error {
						s := sqlTx.Get()
						m := mongoTx.Get()
						Expect(s.ctx.Value(ctxKey("worker"))).To(Equal(id))
						Expect(m.ctx.Value(ctxKey("worker"))).To(Equal(id))
						if id%2 == 1 {
							sqlTx.SetError(NewError("OddWorkerError", "rollback odd worker"))
						}
						return nil
					})

					if id%2 == 1 {
						Expect(err).To(HaveOccurred())
					} else {
						Expect(err).To(Succeed())
					}
				}(i)
			}

			wg.Wait()
		})
	})
})
