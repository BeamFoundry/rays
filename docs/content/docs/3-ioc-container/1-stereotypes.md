---
title: "Stereotypes & Beams"
weight: 1
---

# Stereotypes & Beams

In Rays, a "Beam" is any object managed by the IoC container.
You declare a struct as a Beam by embedding a **Stereotype**.

## 🧹 Simplifying with Dot Imports

Before we dive in, a quick note on style. To keep your domain code clean and free of repetitive package prefixes,
it is highly recommended to use Go's dot import feature for the `core` and `lang` packages.

This allows you to use framework types like `Service`, `Component`, and `Error` directly as if they were
native keywords. We will use this convention in the following examples.

## The Marker Interface System

Under the hood, Rays relies on a **Marker Interface** pattern.

Every specific stereotype (`Component`, `Service`, etc.) implements a base interface called `Stereotype`.

When you call `ctx.Initialize()`, the framework uses Go's reflection to scan your module for any struct that
implements `Stereotype`.

By simply embedding a stereotype, your struct automatically satisfies this interface and becomes visible to the
IoC container.

**NOTE**: *In testing packages a call to `ctx.EnableScanning()` is required in each test to enable scanning for all
non-testing `Stereotype`. Testing `Stereotype` beams must be added manually. See [Testing](/docs/6-testing/) for more details.*

## Core Stereotypes

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

## Ensuring Runtime Detection (Compiler Stripping)

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
