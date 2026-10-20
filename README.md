# BeamFoundry Rays 🚀

[![beamfoundry](https://rays.bfdy.dev/rays-bfdy.jpeg)](https://rays.bfdy.dev/)

[![Go Version](https://img.shields.io/badge/Go-1.27+-00ADD8?style=flat&logo=go)](https://golang.org)
[![codecov](https://codecov.io/gh/BeamFoundry/rays/graph/badge.svg?token=KASD3RXOYT)](https://codecov.io/gh/BeamFoundry/rays)

**This is an extract of the full documentation available at [rays.bdfy.dev](https://rays.bfdy.dev/rays/).**

**Rays** is a powerful, enterprise-grade Dependency Injection (DI) and Inversion of Control (IoC) framework for Go, built by **BeamFoundry**.

Inspired by mature ecosystems like Spring Boot, Rays brings structured lifecycles, annotation-driven configuration, and robust dependency wiring
to Go, while respecting Go's idiomatic use of composition and interfaces. It eliminates boilerplate wiring, enforces clean boundaries, and makes
your Go applications highly testable.

## Introduction and Philosophy

Go is famous for its simplicity and explicit dependency passing. However, as Go applications scale to hundreds of services, repositories, and cross-cutting concerns, manual wiring (often done in massive `init()` functions or `main.go`) becomes fragile and difficult to maintain.

**BeamFoundry Rays solves this by introducing a robust Inversion of Control (IoC) container.**

### Core Principles

1. **Annotation-Driven, but Type-Safe:** We use struct tags (`@:"Inject"`) for declaring dependencies, but rely on Go's strict typing to ensure the container wires the correct pointers.
2. **Encapsulation Matters:** Unlike manual DI which forces you to expose dependencies via public fields or constructor arguments, Rays natively injects into **private fields**. Your internal state remains protected.
3. **Two-Phase Bootstrapping:** To safely handle circular dependencies—a common pain point in large systems—Rays separates instantiation/injection from initialization.
4. **Context-Aware Scoping:** Singletons are great for services, but short-lived state requires a different approach. Rays introduces `Foundry[T]` to bridge the gap between global services and request-scoped prototypes.

## Quickstart

### 📦 Installation

```bash
go get github.com/BeamFoundry/rays
```

### 🚀 The Minimal Application

Here is a complete example of a Rays application. It's not really functional but demonstrate the following :

- `Beam` are defined by marker `Stereotype` interfaces like `Component`, `Service` and `Application`.
- Dependency injection is done via struct tag.
- A `PostConstruct` method can be defined on any `Beam`. They will be called once an instance is injected.
- A custom `Error` type.
- A `Run` method can be defined on any `Beam`. It does not need to be on an `Application`.

Notice how `UserService` does not need a constructor taking `*Database` as an argument.

```go
package main

import (
    "github.com/BeamFoundry/rays/pkg/core"
    "github.com/BeamFoundry/rays/pkg/lang"
    "github.com/BeamFoundry/rays"
)

// 1. Define a Repository
type Database struct {
    core.Component // Stereotype
    dsn string
}

func (db *Database) New() *Database {
    db.dsn = "postgres://localhost/mydb"
    return db
}

// 2. Define a Service
type UserService struct {
    core.Service //
    db     *Database    `@:"Inject"` // Private field injection!
    logger core.Logging `@:"Inject"`
}

func (s *UserService) PostConstruct() lang.Error {
    s.logger.Info("UserService ready!","connection", s.db.dsn)
    return nil
}

// 3. Define an Application, hosting a Run method.
type BatchUserApplication struct {
    core.Application
    userService *UserService `@:"Inject"`
    logger      core.Logging `@:"Inject"`
}

// 4. Define a Run method
func (b *BatchUserApplication) Run(wg *sync.WaitGroup) {
    wg.Add(1) // 1. Register this worker with the global WaitGroup

    b.logger.Info("BatchUserApplication Run started...")

    go func() {
        defer wg.Done() // 2. Ensure we decrement the WaitGroup when finished

        b.logger.Info("BatchUserApplication is running...")

        // Background loop logic goes here...
        for {
            time.Sleep(10 * time.Second)
        }
    }()
}

// 5. Boot the Container
func main() {
    ctx := rays.ApplicationContext()

    if err := ctx.Initialize(); err != nil {
        err.LogErrorWithStackTrace()
        return
    }

    ctx.Run() // Runs application lifecycle
}
```

## Stereotypes and Beams

In Rays, a "Beam" is any object managed by the IoC container.
You declare a struct as a Beam by embedding a **Stereotype**.

### 🧹 Simplifying with Dot Imports

Before we dive in, a quick note on style. To keep your domain code clean and free of repetitive package prefixes,
it is highly recommended to use Go's dot import feature for the `core` and `lang` packages.

This allows you to use framework types like `Service`, `Component`, and `Error` directly as if they were
native keywords. We will use this convention in the following examples.

### The Marker Interface System

Under the hood, Rays relies on a **Marker Interface** pattern.

Every specific stereotype (`Component`, `Service`, etc.) implements a base interface called `Stereotype`.

When you call `ctx.Initialize()`, the framework uses Go's reflection to scan your module for any struct that
implements `Stereotype`.

By simply embedding a stereotype, your struct automatically satisfies this interface and becomes visible to the
IoC container.

**NOTE**: *In testing packages a call to `ctx.EnableScanning()` is required in each test to enable scanning for all
non-testing `Stereotype`. Testing `Stereotype` beams must be added manually. See [Testing](/docs/6-testing/) for more details.*

### Core Stereotypes

* **`Configuration`**: A factory struct that defines other Beams programmatically.
* **`ConfigurationProperties`**: Provides a way to map configuration files to struct.
* **`Component`**: The base stereotype for generic, stateless singleton objects.
* **`Repository`**: Indicates a data access object.
* **`Controller`**: Indicates a web controller.
* **`Service`**: Indicates a class holding business logic. Semantically identical to Component,
  but useful for architectural clarity.
* **`Application`**: Indicates an application, holding a `Run` method.

**NOTE**: *Rays will not provide its own web framework, but will provide existing web framework
connectors to use the `Controller` `Stereotype`. Similarly, there will be connectors for the `Repository` `Stereotype`.*

[//]: <> (* **`Prototype`**: Marks a struct as a non-singleton. A new instance is created every time it is requested via a Foundry.)

### Ensuring Runtime Detection (Compiler Stripping)

Because Rays relies on runtime reflection to instantiate Beams, Go's aggressive compiler and linker might not see
any direct usage of your struct in your source code.

If the compiler thinks a struct is "dead code", it will strip it from the final binary, making it invisible to the
framework's scanner!

To ensure your Beam is compiled in and discoverable at runtime, you must add a static cast at the package level.
This creates an explicit reference to the struct and verifies interface compliance.

```go
package config

import (
    "database/sql"
    . "github.com/BeamFoundry/rays/pkg/core"
)

type DatabaseConfig struct {
    Configuration
}

// Forces the compiler to retain DatabaseConfig in the binary
// and verifies it implements the Stereotype interface.
var _, _ = any(&DatabaseConfig{}).(Stereotype)

// The container calls this method and registers the returned pointer as a Beam.
func (c *DatabaseConfig) ProvidePostgres() *sql.DB {
    db, err := sql.Open("postgres", "...")
    if err != nil {
        return nil // Will not provide a Beam, injecting will fail.
    }
    return db
}
```

Alternatively you can use a provided go tool to generate a file named `zz_stereotype.go` in each package in your
project containing a `Stereotype` beam.

Add the tool to your `go.mod` file :

```bash
go get -tool github.com/BeamFoundry/rays/cmd/stereotyper
```

Update you project before running a test or compiling it :

```bash
go tool stereotyper
```

## Dependency Injection

Rays relies on struct tags to identify injection points.

### Basic Injection

Add `@:"Inject"` to any field that is a **pointer to a struct** or an **interface**.
The container will automatically find and wire the matching Beam by type.

**Note**: *You cannot inject into a concrete value type (like `db Database` instead of `db *Database`),
as this would break the singleton pattern by copying the struct by value.*

```go
import . "github.com/BeamFoundry/rays/pkg/core"

type OrderService struct {
    Service
    inventory *InventoryService `@:"Inject"`
    payment   PaymentGateway    `@:"Inject"` // Interfaces do not need a '*'
}
```

### Private Field Superiority
Idiomatic Go encourages hiding internal state. Rays uses reflection to inject into private fields,
meaning your dependencies aren't exposed to other packages.

### Qualifier and Primary (Resolving Ambiguity)

If you have multiple Beams of the same type, the container will panic with an ambiguity error.
Use `Qualifier` to differentiate them, and `Primary` to set a default fallback.

A default `Qualifier` is always defined for all beams. When using `Configuration`, the qualifier it
is taken from the method's name, uncapitialized. On it can be overwitten using the `Qualifier` interface
in an anonymous struct (a non-named type struct) as the first argument to the method.

```go
import . "github.com/BeamFoundry/rays/pkg/core"

type CloudConfig struct {
    Configuration
}

// Define identical types, marking one as Primary
func (c *CloudConfig) PrimaryStorage(
    _ struct {
        Primary
        Qualifier `Name:"aws"`
    },
) *Storage {
    return &Storage{provider: "AWS"}
}

func (c *CloudConfig) BackupStorage(
    _ struct {
        Qualifier `Name:"gcp"`
    },
) *Storage {
    return &Storage{provider: "GCP"}
}

// 1. Inject a specific one using the Qualifier tag syntax
type BackupService struct {
    Service
    store *Storage `@:"Inject,Qualifier{gcp}"`
}

// 2. Inject the primary one implicitly (no Qualifier needed)
type ImageService struct {
    Service
    store *Storage `@:"Inject"` // Automatically resolves to the PrimaryStorage (AWS)
}
```

#### BeamAttributes

Other attributes can be defined. Those are easily extendable.

For example the `Server` `BeamAttribute`, defined in the `core` package, with a corresponding
`BeamAttributeEvaluator` implementation (hidden in an internal package), lets you another attribute
in the injection tag.

```go
type Server interface {
	  BeamAttribute
	  _Server()
}

type ServerEvaluator struct {
	  Server
}

// Ensute type is compiled in.
var _, _ = any(andServerEvaluator{}).(BeamAttributeEvaluator)

// Ensure interface
var _ BeamAttributeEvaluator = &ServerEvaluator{}

// EvaluateBeamAttribute implements [BeamAttributeEvaluator].
func (q *ServerEvaluator) EvaluateBeamAttribute(beam Beam) string {
    // FindFieldForInterface() and GetFieldFirstTag() are defined in the [core] package.
    field := FindFieldForInterface[Server](beam.AnnotatedType())
    return GetFieldFirstTag(field, []string{"_", "Name", "Server"})
}
```

So example your could wire on the `Server` attribute :

```go
import . "github.com/BeamFoundry/rays/pkg/core"

type CloudConfig struct {
    Configuration
}

func (c *CloudConfig) AwsStorage(
    _ struct {
        Primary
    },
) *Storage {
    return &Storage{provider: "AWS"}
}

func (c *CloudConfig) GcpStorage(
    _ struct {
        Server `Name:"gcp"`
    },
) *Storage {
    return &Storage{provider: "GCP"}
}

type BackupService struct {
    Service
    store *Storage `@:"Inject,Server{gcp}"`
}

type ImageService struct {
    Service
    store *Storage `@:"Inject"`
}
```

### Optional Injection (`Optional[T]`)

By default, if you inject a dependency that does not exist, the container panics and stops booting.
If a dependency is truly optional, wrap it in `Optional[T]` (in the `lang` package).

```go
import (
    . "github.com/BeamFoundry/rays/pkg/core"
    . "github.com/BeamFoundry/rays/pkg/lang"
)

type MetricsService struct {
    Service
    tracer Optional[*DatadogTracer] `@:"Inject"`
}

func (m *MetricsService) Report() {
    if m.tracer.IsPresent() {
        m.tracer.Get().Send(...)
    }
}
```

### Multi-Binding (Slices and Maps)

Advanced architectural patterns (like the Strategy Pattern) rely on injecting *multiple* implementations of an interface or struct. Rays natively supports multi-binding into slices and maps for both interface types and struct pointers.

#### Slice Injection (`[]T`)

Injecting into a slice will populate it with all registered Beams that match the slice's underlying type.

#### Map Injection (`map[string]T`)

Injecting into a map requires the key to be of type `string`. Rays will map the injected Beams by their Qualifier name. The key will be the injected Beam's `Qualifier` value.

```go
import . "github.com/BeamFoundry/rays/pkg/core"

type CommandHandler interface {
    HandleCommand()
}

type ServerNotification struct {
    Configuration
    Server `Name:"primary"`
}

func (s *ServerNotification) Email() CommandHandler {
    ...
}

func (s *ServerNotification) Sms() CommandHandler {
    ...
}

type NotificationDispatcher struct {
    Service

    // Injects all available handlers as a slice
    allHandlers []CommandHandler `@:"Inject"`

    // Injects handlers mapped by their qualifier name ("email", "sms")
    handlerMap map[string]CommandHandler `@:"Inject"`
}

func (d *NotificationDispatcher) Dispatch(handlerName string) {
    if handler, exists := d.handlerMap[handlerName]; exists {
        handler.HandleCommand()
    }
}
```

#### Map Injection with custom grouping

Of course you can group on different attributes, an a default value can be specified from beams not having
the attribute defined.

```go
import . "github.com/BeamFoundry/rays/pkg/core"

type PrimaryServerNotification struct {
    Configuration
    Server `Name:"primary"`
}

func (p *PrimaryServerNotification) Email() CommandHandler {
    ...
}

func (p *PrimaryServerNotification) Sms() CommandHandler {
    ...
}

type SecondaryServerNotification struct {
    Configuration
}

func (s *SecondaryServerNotification) Log() CommandHandler {
    ...
}

type NotificationDispatcher struct {
    Service

    // Inject all handlers by Server attribute, defaulting to secondary
    handlers map[string][]CommandHandler `@:"Inject,GroupBy=Server,Default=secondary"`
}

func (d *NotificationDispatcher) Dispatch (serverName string) {
  if handlers, exists := d.handlers[serverName]; exists {
    for _, handler := range handlers {
      handler.HandleCommand()
    }
  }
}
```

## Configuration and Profiles

In enterprise applications, configuration and environment management go hand-in-hand.

Rays provides a unified system to bind YAML configurations directly to your structs,
override them based on active profiles (e.g., `dev`, `prod`), and conditionally load
specific components depending on the environment.

### 1. Binding Data

Embed `ConfigurationProperties` and define the YAML path via the `Name` tag.

**NOTE:** *Rays strictly enforces a 1-to-1 parity between your
YAML files and your Go code. The `yaml` tag is **not honored**. Only properties with
the exact same case name as the Go struct field will be injected.*

#### Valid types

All numeric types, `bool` and `string` are honored.

Rays includes native support for `time.Duration`.
Text values like `"5000ms"` in the YAML file are automatically parsed using Go's standard `time.ParseDuration` function.

Struct and struct pointers (sub-object mapping) are supported.

Struct implementing the `encoding.TextUnmarshaler` interface can also be used (string to struct mapping).
The struct type (not a pointer to that) can be used directly.

Slices of all the above types are supported.

Finally map of `string` and of `encoding.TextUnmarshaler` to all the above types and to other map of those two types are supported.

#### Validation

Validation is done using the [Validation](github.com/go-playground/validator) project, given you provides the corresponding tags.

NOTE: To validate private fields on a struct, reflection structs will be constructed internally with public fields instead.
If using Cross-Field validation, fields should be indicated in their public form. Errors will be formatted using the original private field instead.

#### Example

```go
// myrepo/myapp/services/kafka.go
package services

import (
    "time"
    . "github.com/BeamFoundry/rays/pkg/core"
    . "github.com/BeamFoundry/rays/pkg/lang"
)

// 1. Define the Configuration
type KafkaConfig struct {
    ConfigurationProperties `Name:"app.kafka"`

    // Exact same-case matching is required.
    //
    // Ensure it's not empty and has valid hostname/port string values.
    brokers []string `validate:"gt=0,dive,hostname_port"`

    // Automatically parses a duration string
    // Ensure a minimum value
    timeout time.Duration `validate:"gt=100ms"`
    retry   bool
}

// 2. Inject it into a Service
type KafkaServer struct {
    Service
    config *KafkaConfig `@:"Inject"`
    logger Logging      `@:"Inject"`
}

// 3. Initialize the service using the injected configuration
func (s *KafkaServer) PostConstruct() Error {
    s.logger.Info("Connecting to kafka", "brokers", len(s.config.brokers), "timeout", s.config.timeout)
    return nil
}
```
### 2. Profile-Based Loading

Rays natively supports Spring-like environment profiles. You define a base configuration file, and then create environment-specific overrides using a suffix convention (`-<profile>.yaml`).

When the application boots, the base file is always loaded. If a specific profile is activated, its corresponding file is loaded *on top*, overriding the base values.

#### The YAML Files

Place your configuration files in a directory (e.g., `config/`).

```yaml
## config/application.yaml (Loaded for everyone)
app:
  kafka:
    brokers: ["localhost:9092"]
    timeout: "5000ms"
    retry: true
```

```yaml
## config/application-prod.yaml (Loaded ONLY if the "prod" profile is active)
app:
  kafka:
    brokers: ["prod-kafka-1:9092", "prod-kafka-2:9092"]
    timeout: "2000ms"
```


### 3. Conditional Registration

Beyond just overriding values, profiles can completely alter the architecture of your application.

You can use anonymous struct parameters in your `Configuration` factories to conditionally load different Beams based on active profiles or YAML properties.

```go
// myrepo/myapp/services/cache.go
package services

import (
    . "github.com/BeamFoundry/rays/pkg/core"

    // Cache implementation
    "repo/cache"
)

// Example of a Cache configuration defined by Conditional
// It is meant to be injected into a not documented Beam.
type CacheConfig struct {
    Configuration
}

// Only loads if the "local" profile is explicitly active
// and Redis is not enabled
func (c *CacheConfig) LocalMemoryCache(
    _ struct {
        ConditionalOnProfile  `Name:"local"`
        ConditionalOnProperty `Name:"redis.enabled" HavingValue:"false" MatchIfMissing:"true"`
    },
) *cache.Cache {
    return &cache.Cache{type: "in-memory"}
}

// Only loads if the YAML file contains "redis.enabled: true"
func (c *CacheConfig) RedisCache(
    _ struct {
        ConditionalOnProperty `Name:"redis.enabled" MatchIfMissing:"false"`
    },
) *cache.Cache {
    return &cache.Cache{type: "redis"}
}

// Only loads if NO OTHER *Cache beam has been registered yet
func (c *CacheConfig) FallbackCache(
    _ struct {
        ConditionalOnMissingBeam
    },
) *cache.Cache {
    return &cache.Cache{type: "noop"}
}
```

### 4. The Application Entrypoint

In your `main.go`, you bring it all together: set the active profiles, load the configuration using a filesystem object (`fs.FS`),
and use an underscore import to ensure your isolated packages are compiled into the binary.

```go
// main.go
package main

import (
    "os"

    // 3rd-Party packages for main
    flag "github.com/spf13/pflag"

    // Rays
    "github.com/BeamFoundry/rays"

    // Application package imports
    // Blank import ensures the package is compiled and visible to the scanner
    _ "myrepo/myapp/services"
)

var profiles []string = []string{}

func main() {
    // Parsing flags
    flag.StringSliceVarP(&profiles, "profile", "p", []string{}, "Profile(s) to use, default : none")
    flag.Parse()

    // Getting rays ApplicationContext
    ctx := rays.ApplicationContext()

    // Tell the container we are running in production
    ctx.SetActiveProfiles(profiles)

    // Load configuration from the "config" directory.
    // This will load both "application.yaml" AND "application-prod.yaml"
    if stat, err := os.Stat("config"); os.IsExist(err) && stat.IsDir() {
        ctx.AddConfigFS(os.DirFS(config))
    }
  }

    // Initialize Container
    if err := ctx.Initialize(); err != nil {
        err.LogErrorWithStackTrace()
        os.Exit(1)
    }

    // Finally Call run
    ctx.Run()
}
```

## Logging Framework

Enterprise applications require structured, leveled, and contextual logging.

Rays provides a dedicated configurable, logging framework.

### The `Logging` Interface

Rays logging is powered by Go's standard `slog` library under the hood, providing high-performance, structured logging.
However, standard `slog` natively lacks a "Trace" level for highly granular debugging.

Rays extends the standard implementation by adding full support for the `Trace` level, including the corresponding `Trace()` and `TraceContext()` functions on the `Logging` interface.

#### Configuration

You can configure the logging framework globally or on a per-package basis using your `application.yaml` file under the `rays.logging` property.

```yaml
rays:
  logging:
    # The default minimum log level
    #
    # Valid values are "TRACE", "DEBUG", "INFO", "WARN", "ERROR" in that order.
    # A value of "INFO" for example will hide all logging from "TRACE" and "DEBUG"
    #
    # Type: text
    # Default: INFO
    level: DEBUG

    # The slog handler to use.
    #
    # Valid values are "DEFAULT", "TEXT", or "JSON".
    #
    # Type: text
    # Default: DEFAULT
    handler: DEFAULT

    # Add source location to each log entry.
    #
    # Type: boolean
    # Default: true
    source: true

    # Override the log level for specific packages.
    #
    # Before injecting a [Logging], a new one will be created and its default level will be changed
    # if the struct requesting the [Logging] has its package name referenced here.
    #
    # Type: Dictionary of "source: level"
    #
    # "source" is either a package name (the one of the struct injecting the [Logging]) or
    # the package name followed by a slash and the struct name.
    #
    # It should be noted that "level" will only apply to that "source"
    # and will not apply to its sub packages UNLESS it ends with a slash.
    #
    loggers:
      github.com/BeamFoundry/rays-sample/: INFO
      github.com/BeamFoundry/rays-sample/pkg/config: DEBUG
      github.com/BeamFoundry/rays-sample/pkg/config/DatabaseConfig: INFO
```

#### Basic Injection

To use the logger, inject the `Logging` interface into your Beams.

```go
import (
    . "github.com/BeamFoundry/rays/pkg/core"
    . "github.com/BeamFoundry/rays/pkg/lang"
)

type PaymentService struct {
    Service

    // Note: No pointer '*' because Logging is an interface
    logger Logging `@:"Inject"`
}

func (s *PaymentService) Process(amount float64) Error {
    s.logger.Trace("Entering Process method", "amount", amount) // Extremely granular logging

    // Logging supports structured key-value pairs natively
    s.logger.Info("Processing payment", "amount", amount)

    return nil
}
```

## Error Handling

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

### Integration with Logging

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

### The Try/Catch Pattern

While standard Go relies heavily on `if err != nil` checks, complex procedural logic can sometimes benefit from a more traditional exception-handling flow. The `lang` package provides a robust `Try/Catch` mechanism designed specifically around the framework's `Error` interface.

#### Core Functions
* **`Try(try func(), catches ...ErrorHandler) Error`**: Executes the `try` function. If a panic occurs involving an `Error`, it evaluates the provided catches and returns the resulting `Error`. If no catch matches, the original error is returned automatically. Additionally, if standard Go code or a sub-function triggers a regular `panic()`, the `Try` block will safely catch it and wrap it into a new framework `Error` with the name "\"PanicError\"", ensuring your application does not crash unexpectedly.
* **`Catch[E Error](action func(E) Error) ErrorHandler`**: Catches an error based on its specific Go type.
* **`CatchName(name string, action func(Error) Error) ErrorHandler`**: Catches an error based on its internal string name.
* **`Finally(action func()) ErrorHandler`**: Executes an action after the `try` block finishes, regardless of success or failure.

#### Raise Helpers
To easily trigger panics within a `Try` block when dealing with standard Go returns, use the `Raise` helpers:
* **`RaiseErr(err error) bool`**: Panics if `err` is not nil.
* **`RaiseVal[T any](t T, err error) T`**: Panics if `err` is not nil; otherwise, returns the object `T`. This is perfect for wrapping standard library calls like `RaiseVal(os.ReadFile("config.yaml"))`.

#### Example Usage

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

## Type-Safe Enums

Go does not have native, strict enums. This often leads to configuration bugs where an invalid string from a YAML file is happily parsed into a `string` field, only to fail deep within the application logic.

Rays solves this with `Enum[T]`. It strictly binds string values from YAML into custom Go types and validates them during the framework's Instantiation phase.

#### 1. Define the Enum Type
You can define your enums using a struct to hold the variants as fields.

```go
import . "github.com/BeamFoundry/rays/pkg/lang"

type LogLevels struct {
    DEBUG, INFO, ERROR Enum[LogLevels]
}

type LogLevel = Enum[LogLevels]
```

Alternatively, this can be simplified by embedding the generic enum type directly into the struct:

```go
import . "github.com/BeamFoundry/rays/pkg/lang"

type LogLevel struct {
    Enum[LogLevel]
    DEBUG, INFO, ERROR Enum[LogLevel]
}
```

#### 2. Use it in Configuration
Simply declare the field using your custom type. Note that it works seamlessly without wrapping the struct field in generic notation.

```go
import . "github.com/BeamFoundry/rays/pkg/core"

type AppConfig struct {
    ConfigurationProperties `Name:"app"`
    level LogLevel
}
```

#### 3. Safe Access in Business Logic
Because of the generic structuring, you must initialize the enum type by calling `.New()` to ensure its internal state is properly configured before accessing its variants.

You can then compare it directly to your configuration fields without needing a getter function.

```go
import (
    "fmt"
    . "github.com/BeamFoundry/rays/pkg/lang"
)

func (s *MyService) PostConstruct() Error {
    // Initialize the Enum structure and compare the property directly
    if s.config.level == (LogLevel{}.New().DEBUG) {
        fmt.Println("Debug mode is enabled!")
    }
    return nil
}
```
