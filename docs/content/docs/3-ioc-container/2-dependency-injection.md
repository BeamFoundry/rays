---
title: "Dependency Injection"
weight: 2
---

# Dependency Injection

Rays relies on struct tags to identify injection points.

## Basic Injection

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

## Private Field Superiority
Idiomatic Go encourages hiding internal state. Rays uses reflection to inject into private fields,
meaning your dependencies aren't exposed to other packages.

## Qualifier & Primary (Resolving Ambiguity)

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

### BeamAttributes

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
var _, _ = any(&ServerEvaluator{}).(BeamAttributeEvaluator)

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

## Optional Injection (`Optional[T]`)

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

## Multi-Binding (Slices & Maps)

Advanced architectural patterns (like the Strategy Pattern) rely on injecting *multiple* implementations of an interface
or struct. Rays natively supports multi-binding into slices and maps for both interface types and struct pointers.

### Slice Injection (`[]T`)

Injecting into a slice will populate it with all registered Beams that match the slice's underlying type.

### Map Injection (`map[string]T`)

Injecting into a map requires the key to be of type `string`. Rays will map the injected Beams by their Qualifier name.
The key will be the injected Beam's `Qualifier` value.

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

### Map Injection with custom grouping

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

