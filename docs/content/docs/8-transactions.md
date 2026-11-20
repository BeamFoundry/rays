---
title: "8. Transaction Management"
weight: 8
---

# Transaction Management

Enterprise applications frequently span multiple repositories—and sometimes multiple data stores (such as relational databases via `rays-orm` and document stores via `rays-data-mongo`)—within a single business operation. Passing `context.Context` and database-specific transaction handles through every service and repository method signature quickly pollutes domain logic and couples services to underlying persistence drivers.

Rays solves this with a centralized, goroutine-local **Transaction Management** architecture built on top of `github.com/timandy/routine`:

- **Context-Free Signatures**: Bind a `context.Context` (with timeouts, cancellation, or tracing values) once per goroutine on `TransactionManager` instead of threading `ctx` through every method call.
- **Multi-Resource Coordination**: Coordinate transaction state across multiple `Transactional[T]` providers (e.g., GORM SQL + MongoDB) on the current goroutine.
- **Lazy Enlistment**: Data providers only open a transaction when a repository actually accesses them (`Get()`) during an active transaction on the current goroutine.
- **Automatic Cross-Provider Rollback**: Any error encountered in a repository is reported to its `Transactional[T]` provider and forwarded to `TransactionManager`, which immediately rolls back **all** participating `Transactional[T]` providers on that goroutine.

---

## Architecture & Data Flow

```
+-------------------------------------------------------------------+
|                   Controller / Middleware / Caller                 |
|         txManager.SetContext(ctx) / defer txManager.ClearContext() |
+---------------------------------+---------------------------------+
                                  |
                                  v
+-------------------------------------------------------------------+
|                           Service Layer                           |
|       txManager.Transactional(func() Error { ... })               |
|    or txManager.Start() / Commit() / Rollback() / defer End()     |
+----------------+---------------------------------+----------------+
                 |                                 |
                 v                                 v
    +-------------------------+       +-------------------------+
    |    SQL Repository       |       |   Mongo Repository      |
    |  Get() / SetError(err)  |       |  Get() / SetError(err)  |
    +------------+------------+       +------------+------------+
                 |                                 |
                 v                                 v
    +-------------------------+       +-------------------------+
    |   Transactional[gorm.DB]|       |Transactional[MongoSess] |
    |  (TransactionProvider)  |       |  (TransactionProvider)  |
    +------------+------------+       +------------+------------+
                 |                                 |
                 +----------------+----------------+
                                  |
          Enlist(this) / GetContext() / SetError(err)
          Start() / Commit() / Rollback() / End()
                                  |
                                  v
+-------------------------------------------------------------------+
|               TransactionManager (DefaultTransactionManager)       |
|       Goroutine-Local Context & Multi-Provider State Coordinator  |
+-------------------------------------------------------------------+
```

### Component Breakdown

| Component | Interface / Type | Package | Role |
| :--- | :--- | :--- | :--- |
| **Transaction Manager** | `TransactionManager` | `pkg/core` | Central coordinator injected into Services and `Transactional[T]` providers. Manages goroutine-local `context.Context`, lazy provider enlistment, commits, rollbacks, and error state. |
| **Default Implementation** | `DefaultTransactionManager` | `pkg/tx` | Built-in `TransactionManager` implementation backed by `routine.ThreadLocal`. Auto-configured via `ConditionalOnMissingBeam`. |
| **Transaction Provider** | `TransactionProvider` | `pkg/core` | Non-generic lifecycle contract (`Start`, `Commit`, `Rollback`, `End`, `SetError`, `Error`) implemented by all transactional data providers. |
| **Generic Provider** | `Transactional[T any]` | `pkg/core` | Embeds `TransactionProvider` and adds `Get() *T` for repositories to obtain the current goroutine's transactional resource handle. |

---

## Core Interfaces

### `TransactionManager` (`pkg/core`)

Services and middleware interact with the `TransactionManager` interface:

```go
type TransactionManager interface {
    Manager

    // Context management (goroutine-local)
    SetContext(ctx context.Context)
    GetContext() context.Context
    ClearContext()

    // Provider enlistment
    Enlist(provider TransactionProvider) Error

    // Transaction lifecycle (goroutine-local)
    Start() Error
    IsActive() bool
    Commit() Error
    Rollback() Error
    End()
    Transactional(action func() Error) Error

    // Error state & propagation
    SetError(err Error)
    Error() Error
}
```

| Method | Description |
| :--- | :--- |
| `SetContext(ctx)` | Binds `ctx` to the current goroutine (defaults to `context.Background()` if `nil`). |
| `GetContext()` | Returns the `context.Context` bound to the current goroutine, or `context.Background()`. |
| `ClearContext()` | Resets the current goroutine's `context.Context` back to `context.Background()`. |
| `Enlist(provider)` | Lazily registers a `TransactionProvider` in the current goroutine's active transaction and calls `provider.Start()`. |
| `Start()` | Starts a new coordinated transaction on the current goroutine. Returns an `Error` if a transaction is already active. |
| `IsActive()` | Returns `true` if a transaction is currently active on the current goroutine. |
| `Commit()` | Commits all enlisted `TransactionProvider` instances on the current goroutine. If any provider fails to commit, rolls back remaining uncommitted providers and returns the `Error`. |
| `Rollback()` | Rolls back all enlisted `TransactionProvider` instances on the current goroutine (idempotent if already rolled back). |
| `End()` | Finalizes the active transaction on the current goroutine. Automatically rolls back if neither `Commit()` nor `Rollback()` was called, invokes `End()` on all enlisted providers, and clears goroutine-local transaction state. |
| `Transactional(action)` | Starts a transaction, executes `action()`, and automatically commits on success or rolls back if `action()` returns an `Error`, panics (`RaiseErr`), or any repository reports `SetError(err)`. |
| `SetError(err)` | Records `err` on the current goroutine's active transaction and **immediately calls `Rollback()`** on all enlisted providers. |
| `Error()` | Returns the `Error` recorded on the current goroutine's active transaction, or `nil`. |

---

### `TransactionProvider` & `Transactional[T any]` (`pkg/core`)

Data modules (such as `rays-orm` and `rays-data-mongo`) implement `Transactional[T any]`, which embeds the non-generic `TransactionProvider` interface:

```go
type TransactionProvider interface {
    Start() Error
    Commit() Error
    Rollback() Error
    End()
    SetError(err Error)
    Error() Error
}

type Transactional[T any] interface {
    TransactionProvider
    Get() *T
}
```

---

## Using `TransactionManager` in Services

When you import `github.com/BeamFoundry/rays`, the container automatically registers `DefaultTransactionManager` as a singleton Beam implementing `TransactionManager`. Inject it directly into your `Service` structs using `@:"Inject"`.

### 1. Declarative Closure Style (`Transactional`)

The recommended way to run transactional business logic is `txManager.Transactional(...)`. It manages `Start()`, `Commit()`, `Rollback()`, panic recovery, and `End()` automatically:

```go
package services

import (
    . "github.com/BeamFoundry/rays/pkg/core"
    . "github.com/BeamFoundry/rays/pkg/lang"
)

type OrderService struct {
    Service
    txManager  TransactionManager `@:"Inject"`
    orderRepo  OrderRepository    `@:"Inject"`
    auditRepo  AuditRepository    `@:"Inject"`
}

func (this *OrderService) PlaceOrder(order *Order, audit *AuditEntry) Error {
    return this.txManager.Transactional(func() Error {
        // 1. SQL Repository (lazily enlists GormTransactionalManager)
        if _, err := this.orderRepo.Save(order); err != nil {
            return err
        }

        // 2. MongoDB Repository (lazily enlists MongoTransactionalManager)
        if _, err := this.auditRepo.Save(audit); err != nil {
            return err
        }

        return nil
    })
}
```

> **Note:** Even if a repository method does not return an `Error` in its signature (for example, a generated finder like `user := this.userRepo.FindByUsername("alice")`), any underlying database error is reported by the repository to `TransactionManager.SetError(err)`. This immediately rolls back all participating providers and causes `this.txManager.Transactional(...)` (or `this.txManager.Commit()`) to return that `Error`.

---

### 2. Manual Lifecycle Control (`Start` / `Commit` / `Rollback` / `End`)

For fine-grained imperative control, use `Start()` with `defer this.txManager.End()`:

```go
func (this *OrderService) CancelOrder(orderID uint) Error {
    if err := this.txManager.Start(); err != nil {
        return err
    }
    // End() guarantees cleanup and auto-rollbacks if Commit() is not reached
    defer this.txManager.End()

    order, ok := this.orderRepo.FindById(orderID).Get()
    if !ok {
        // Check if FindById failed due to a database error
        if err := this.txManager.Error(); err != nil {
            return err
        }
        _ = this.txManager.Rollback()
        return NewError("OrderNotFound", "Order does not exist")
    }

    order.Status = "CANCELLED"
    if _, err := this.orderRepo.Save(order); err != nil {
        _ = this.txManager.Rollback()
        return err
    }

    return this.txManager.Commit()
}
```

---

### 3. Binding a Goroutine-Local `context.Context`

To propagate request timeouts, cancellation signals, or tracing metadata to all database operations on the current goroutine (both transactional and non-transactional), call `SetContext(ctx)` and `defer ClearContext()`:

```go
func (this *OrderService) PlaceOrderWithTimeout(order *Order, timeout time.Duration) Error {
    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()

    this.txManager.SetContext(ctx)
    defer this.txManager.ClearContext()

    return this.txManager.Transactional(func() Error {
        _, err := this.orderRepo.Save(order)
        return err
    })
}
```

`SetContext` and `ClearContext` are independent of `Start()` and `End()`, allowing an HTTP middleware or controller to bind a request `context.Context` once per request across multiple service calls and transactions.

---

## Implementing a Custom `Transactional[T]` Provider

If you are building a custom persistence integration for Rays, implement `Transactional[T]` as a `Component` that injects `TransactionManager` and stores its active session in a `routine.ThreadLocal[*T]`:

```go
package mydriver

import (
    "github.com/timandy/routine"

    . "github.com/BeamFoundry/rays/pkg/core"
    . "github.com/BeamFoundry/rays/pkg/lang"
)

type MyTransactionalManager struct {
    Component
    client      *MyClient          `@:"Inject"`
    txManager   TransactionManager `@:"Inject"`
    logger      Logging            `@:"Inject"`
    threadLocal routine.ThreadLocal[*MySession]
}

var (
    _    Transactional[MySession] = &MyTransactionalManager{}
    _, _                          = any(&MyTransactionalManager{}).(Stereotype)
)

func (this *MyTransactionalManager) PostConstruct() Error {
    this.threadLocal = routine.NewThreadLocal[*MySession]()
    return nil
}

func (this *MyTransactionalManager) Start() Error {
    sess, err := this.client.BeginTx(this.txManager.GetContext())
    if err != nil {
        return NewWrappedError("TransactionError", err)
    }
    this.threadLocal.Set(sess)
    return nil
}

func (this *MyTransactionalManager) Get() *MySession {
    if sess := this.threadLocal.Get(); sess != nil {
        return sess
    }
    if this.txManager != nil && this.txManager.IsActive() {
        if err := this.txManager.Enlist(this); err != nil {
            sess := this.client.SessionWithContext(this.txManager.GetContext())
            sess.SetError(err)
            return sess
        }
        return this.threadLocal.Get()
    }
    return this.client.SessionWithContext(this.txManager.GetContext())
}

func (this *MyTransactionalManager) Commit() Error {
    if sess := this.threadLocal.Get(); sess != nil {
        if err := sess.Commit(); err != nil {
            return NewWrappedError("TransactionError", err)
        }
    }
    return nil
}

func (this *MyTransactionalManager) Rollback() Error {
    if sess := this.threadLocal.Get(); sess != nil {
        if err := sess.Rollback(); err != nil {
            return NewWrappedError("TransactionError", err)
        }
    }
    return nil
}

func (this *MyTransactionalManager) End() {
    if sess := this.threadLocal.Get(); sess != nil {
        sess.Close()
    }
    this.threadLocal.Remove()
}

func (this *MyTransactionalManager) SetError(err Error) {
    if this.txManager != nil {
        this.txManager.SetError(err)
    }
}

func (this *MyTransactionalManager) Error() Error {
    if this.txManager != nil {
        return this.txManager.Error()
    }
    return nil
}
```
