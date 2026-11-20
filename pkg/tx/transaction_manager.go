package tx

import (
	"context"
	"slices"

	"github.com/timandy/routine"

	. "github.com/BeamFoundry/rays/pkg/core"
	. "github.com/BeamFoundry/rays/pkg/lang"
)

const transactionErrorName = "TransactionError"

type transactionState struct {
	err         Error
	providers   []TransactionProvider
	committed   bool
	rolledBack  bool
	rollingBack bool
}

// DefaultTransactionManager is the default [TransactionManager] implementation,
// coordinating goroutine-local [context.Context] and transaction state across
// multiple [TransactionProvider] ([Transactional]) instances.
type DefaultTransactionManager struct {
	Manager
	logger   Logging `@:"Inject"`
	ctxLocal routine.ThreadLocal[context.Context]
	txLocal  routine.ThreadLocal[*transactionState]
}

var _ TransactionManager = &DefaultTransactionManager{}

// New creates and initializes a new [DefaultTransactionManager].
func (this DefaultTransactionManager) New() *DefaultTransactionManager {
	this.ctxLocal = routine.NewThreadLocalWithInitial(context.Background)
	this.txLocal = routine.NewThreadLocal[*transactionState]()
	return &this
}

// SetContext binds ctx to the current goroutine (defaults to [context.Background] if nil).
//
//nolint:contextcheck // Fallback to background context when caller passes nil
func (this *DefaultTransactionManager) SetContext(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}
	this.ctxLocal.Set(ctx)
}

// GetContext returns the [context.Context] bound to the current goroutine,
// or [context.Background] if none is set.
func (this *DefaultTransactionManager) GetContext() context.Context {
	return this.ctxLocal.Get()
}

// ClearContext resets the [context.Context] for the current goroutine back to default.
func (this *DefaultTransactionManager) ClearContext() {
	this.ctxLocal.Remove()
}

// Start begins a new coordinated transaction on the current goroutine.
func (this *DefaultTransactionManager) Start() Error {
	if this.txLocal.Get() != nil {
		return NewError(transactionErrorName, "Transaction is already active")
	}
	if this.logger != nil {
		this.logger.Trace("Start Transaction", "manager", "DefaultTransactionManager")
	}
	this.txLocal.Set(&transactionState{})
	return nil
}

// IsActive returns true if a transaction is currently active on the current goroutine.
func (this *DefaultTransactionManager) IsActive() bool {
	return this.txLocal.Get() != nil
}

// Enlist registers a [TransactionProvider] in the current goroutine's active transaction
// and starts its underlying transaction if not already enlisted.
func (this *DefaultTransactionManager) Enlist(provider TransactionProvider) Error {
	if provider == nil {
		return NewError(transactionErrorName, "Cannot enlist nil TransactionProvider")
	}
	state := this.txLocal.Get()
	if state == nil {
		return NewError(transactionErrorName, "No active transaction")
	}
	if state.err != nil {
		return state.err
	}
	if state.committed || state.rolledBack {
		return NewError(transactionErrorName, "Transaction already finalized")
	}
	if slices.Contains(state.providers, provider) {
		return nil
	}

	state.providers = append(state.providers, provider)
	if err := provider.Start(); err != nil {
		this.SetError(err)
		return err
	}
	return nil
}

// SetError records an [Error] on the current goroutine's active transaction and immediately
// triggers [TransactionProvider.Rollback] on all enlisted providers.
func (this *DefaultTransactionManager) SetError(err Error) {
	if err == nil {
		return
	}
	state := this.txLocal.Get()
	if state == nil {
		return
	}
	if state.err == nil {
		state.err = err
	}
	if !state.rolledBack && !state.committed && !state.rollingBack {
		//nolint:errcheck // Primary transaction error is already recorded in state.err
		this.rollbackState(state)
	}
}

// Error returns the [Error] recorded on the current goroutine's active transaction, or nil.
func (this *DefaultTransactionManager) Error() Error {
	state := this.txLocal.Get()
	if state == nil {
		return nil
	}
	return state.err
}

// Commit commits all enlisted [TransactionProvider] instances on the current goroutine.
func (this *DefaultTransactionManager) Commit() Error {
	state := this.txLocal.Get()
	if state == nil {
		return NewError(transactionErrorName, "No active transaction")
	}
	if state.err != nil {
		return state.err
	}
	if state.rolledBack {
		return NewError(transactionErrorName, "Transaction has already been rolled back")
	}
	if state.committed {
		return NewError(transactionErrorName, "Transaction has already been committed")
	}

	if this.logger != nil {
		this.logger.Trace("Commit Transaction", "manager", "DefaultTransactionManager", "providers", len(state.providers))
	}

	for i, provider := range state.providers {
		err := provider.Commit()
		if err == nil {
			if state.err != nil {
				return state.err
			}
			continue
		}
		state.err = err
		state.rollingBack = true
		for j := len(state.providers) - 1; j >= i; j-- {
			if rbErr := state.providers[j].Rollback(); rbErr != nil && this.logger != nil {
				this.logger.Error("Rollback failed after commit error", "error", rbErr.Error())
			}
		}
		state.rollingBack = false
		state.rolledBack = true
		return err
	}

	state.committed = true
	return nil
}

// Rollback rolls back all enlisted [TransactionProvider] instances on the current goroutine.
func (this *DefaultTransactionManager) Rollback() Error {
	state := this.txLocal.Get()
	if state == nil {
		return NewError(transactionErrorName, "No active transaction")
	}
	if state.committed {
		return NewError(transactionErrorName, "Transaction has already been committed")
	}
	if state.rolledBack || state.rollingBack {
		return nil
	}
	return this.rollbackState(state)
}

func (this *DefaultTransactionManager) rollbackState(state *transactionState) Error {
	if this.logger != nil {
		this.logger.Trace("Rollback Transaction", "manager", "DefaultTransactionManager", "providers", len(state.providers))
	}
	state.rollingBack = true
	defer func() {
		state.rollingBack = false
		state.rolledBack = true
	}()

	var firstErr Error
	for _, v := range slices.Backward(state.providers) {
		if err := v.Rollback(); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			if this.logger != nil {
				this.logger.Error("Provider rollback error", "error", err.Error())
			}
		}
	}
	if state.err == nil && firstErr != nil {
		state.err = firstErr
	}
	return firstErr
}

// End finalizes and cleans up the active transaction and all enlisted [TransactionProvider]
// instances on the current goroutine.
func (this *DefaultTransactionManager) End() {
	state := this.txLocal.Get()
	if state == nil {
		return
	}
	if this.logger != nil {
		this.logger.Trace("End Transaction", "manager", "DefaultTransactionManager", "providers", len(state.providers))
	}
	if !state.committed && !state.rolledBack && !state.rollingBack {
		//nolint:errcheck // Best-effort rollback on cleanup when transaction was not finalized
		this.rollbackState(state)
	}
	for _, v := range slices.Backward(state.providers) {
		v.End()
	}
	this.txLocal.Remove()
}

// Transactional executes action inside a coordinated transaction on the current goroutine,
// committing on success or rolling back if action returns an [Error], panics, or any
// participating provider reports an error via [TransactionManager.SetError].
func (this *DefaultTransactionManager) Transactional(action func() Error) Error {
	if err := this.Start(); err != nil {
		err.LogError()
		return err
	}
	defer this.End()

	err := Try(func() {
		RaiseErr(action())
	})
	if err != nil {
		this.SetError(err)
		return err
	}
	if txErr := this.Error(); txErr != nil {
		return txErr
	}
	if commitErr := this.Commit(); commitErr != nil {
		commitErr.LogError()
		return commitErr
	}
	return nil
}
