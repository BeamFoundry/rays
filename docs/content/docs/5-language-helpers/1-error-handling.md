---
title: "Error Handling"
weight: 1
---

# Error Handling

If a container fails to boot, you need to know exactly *where* the wiring failed. Rays replaces standard Go `error` handling during the lifecycle with a richer `Error` interface that captures stack traces and key-value attributes automatically.

```go
import . "github.com/BeamFoundry/rays/pkg/lang"

func (s *UserService) PostConstruct() Error {
    if s.db == nil {
        err := NewError("WiringError", "Database pointer is nil")

        // AddAttribute does not return the Error object, so it cannot be chained.
        err.AddAttribute("component", "UserService")
        err.AddAttribute("action", "PostConstruct")

        return err
    }
    return nil
}
```

## Integration with Logging

Because Rays integrates its `lang` package directly with the `logging` framework (covered in Chapter 4), an `Error` can report itself seamlessly.

Instead of manually unpacking the error to log it, simply call `err.LogError()`. This function automatically extracts the error message, the stack trace, and all attached attributes, serializing them into your configured logging backend.

```go
import . "github.com/BeamFoundry/rays/pkg/lang"

func (s *PaymentService) Process(amount float64) Error {
    if amount <= 0 {
        err := NewError("ValidationError", "Payment amount must be greater than zero")
        err.AddAttribute("component", "PaymentService")
        err.AddAttribute("amount", amount)

        // Automatically logs the error with all context using the framework logger!
        err.LogError()

        return err
    }
    return nil
}
```

## The Try/Catch Pattern

While standard Go relies heavily on `if err != nil` checks, complex procedural logic can sometimes benefit from a more traditional exception-handling flow. The `lang` package provides a robust `Try/Catch` mechanism designed specifically around the framework's `Error` interface.

### Core Functions
* **`Try(try func(), catches ...ErrorHandler) Error`**: Executes the `try` function. If a panic occurs involving an `Error`, it evaluates the provided catches and returns the resulting `Error`. If no catch matches, the original error is returned automatically. Additionally, if standard Go code or a sub-function triggers a regular `panic()`, the `Try` block will safely catch it and wrap it into a new framework `Error` with the name "\"PanicError\"", ensuring your application does not crash unexpectedly.
* **`Catch[E Error](action func(E) Error) ErrorHandler`**: Catches an error based on its specific Go type.
* **`CatchName(name string, action func(Error) Error) ErrorHandler`**: Catches an error based on its internal string name.
* **`Finally(action func()) ErrorHandler`**: Executes an action after the `try` block finishes, regardless of success or failure.

### Raise Helpers
To easily trigger panics within a `Try` block when dealing with standard Go returns, use the `Raise` helpers:
* **`RaiseErr(err error) bool`**: Panics if `err` is not nil.
* **`RaiseVal[T any](t T, err error) T`**: Panics if `err` is not nil; otherwise, returns the object `T`. This is perfect for wrapping standard library calls like `RaiseVal(os.ReadFile("config.yaml"))`.

### Example Usage

Here is a complete example defining a custom exception and handling it cleanly using the `Try/Catch` flow.

```go
import (
    "fmt"
    . "github.com/BeamFoundry/rays/pkg/core"
    . "github.com/BeamFoundry/rays/pkg/lang"
)

// 1. Define a Custom Exception
type TestException struct {
    *DefaultError
}

func (e TestException) New() *TestException {
    e.DefaultError = NewError("TestException", "A test exception has occurred")
    return &e
}

type ProcessingService struct {
    Service
}

// 2. Use Try/Catch in Business Logic
func (s *ProcessingService) Process() Error {
    return Try(func() {

        // Simulate an action that throws an error
        RaiseErr(TestException{}.New())

        fmt.Println("This line will never be reached")

    }, CatchName("DatabaseError", func(e Error) Error {
        // Will not match "TestException"
        return e
    }), Catch(func(e *TestException) Error {
        // This WILL catch the specific TestException type
        fmt.Println("Caught specifically:", e.Message())
        return e
    }), Finally(func() {
        fmt.Println("Cleanup resources here...")
    }))
}
```
