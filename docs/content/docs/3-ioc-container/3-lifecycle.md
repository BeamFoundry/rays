---
title: "Lifecycle Management"
weight: 3
---

# Lifecycle Management

Rays uses a strict, deterministic phase model to ensure application stability.

## Phase 1: Instantiation & Wiring
1. The container scans for all embedded stereotypes.
2. It calls `New()` on structs that implement it to set up initial state.
3. It maps `ConfigurationProperties` from YAML.
4. It resolves all `@:"Inject"` tags and assigns pointers (or interface references).

*Note: Because no business logic runs in Phase 1, Rays natively handles **Circular Dependencies** between singletons.*

## Phase 2: Initialization (`PostConstruct`)

Once everything is wired, the container executes PostConstruct hooks. This is where you validate config, open connections, etc...

### Standard Hooks vs Hierarchical Hooks

* **Parent Types** implement `PostConstruct() Error`.
* **Embedded Types** implement `PostConstruct(parent any) Error`. The container will pass a pointer to the parent struct
that embedded it, which is incredibly powerful for building generic base modules.

Here is a complete example showing `New()`, embedded hooks, interface enforcement, and composition working together:

```go
import (
    "fmt"
    . "github.com/BeamFoundry/rays/pkg/core"
    . "github.com/BeamFoundry/rays/pkg/lang"
)

// 1. Define a contract for parents embedding the BaseWorker
type Worker interface {
    WorkerName() string
}

// 2. Define the reusable Base Module
type BaseWorker struct {
    logger Logging `@:"Inject"`
    worker Worker // Stored during PostConstruct for later use
}

// Inspect logs a formatted message using the internal logger and the parent's identity
func (b *BaseWorker) Inspect(message string) {
    b.logger.Info(fmt.Sprintf("[%s] %s", b.worker.WorkerName(), message))
}

// Embedded hook: receives the parent instance and verifies compliance
func (b *BaseWorker) PostConstruct(parent any) Error {
    worker, ok := parent.(Worker)
    if !ok {
        return NewError("LifecycleError", "Parent struct must implement the Worker interface")
    }

    // Store the strongly-typed parent reference
    b.worker = worker

    b.logger.Info(fmt.Sprintf("BaseWorker successfully attached to: %s", worker.WorkerName()))
    return nil
}

// 3. Define the actual implementer
type EmailWorker struct {
    Component
    BaseWorker // Triggers the hierarchical hook and provides the Inspect() method

    workerName string
}

// Instantiation Phase
func (e *EmailWorker) New() *EmailWorker {
    e.workerName = "TransactionalEmailWorker"
    return e
}

// Fulfills the Worker interface for BaseWorker
func (e *EmailWorker) WorkerName() string {
    return e.workerName
}

// Initialization Phase (Parent hook takes no arguments)
func (e *EmailWorker) PostConstruct() Error {
    // Because BaseWorker is embedded, EmailWorker inherits the Inspect() method!
    e.Inspect("EmailWorker is fully initialized and ready to process emails.")
    return nil
}
```

## Phase 3: Runtime (`BeamRunner`)
Beams that implement the `rays.BeamRunner` interface will have their `Run(wg *sync.WaitGroup)` method executed when `ctx.Run()` is called in `main.go`.

This hook is specifically designed for long-running processes (like HTTP servers, Kafka consumers, or periodic cron jobs). The WaitGroup keeps the main application thread alive while your background workers execute.

```go
import (
    "sync"
    "time"
    . "github.com/BeamFoundry/rays/pkg/core"
)

type MetricsWorker struct {
    Component
    logger Logging `@:"Inject"`
}

// Fulfills the BeamRunner interface
func (m *MetricsWorker) Run(wg *sync.WaitGroup) {
    wg.Add(1) // 1. Register this worker with the global WaitGroup

    go func() {
        defer wg.Done() // 2. Ensure we decrement the WaitGroup when finished

        m.logger.Info("Metrics Worker started...")

        // Background loop logic goes here...
        for {
            time.Sleep(10 * time.Second)
            // m.flushMetrics()
        }
    }()
}
```
