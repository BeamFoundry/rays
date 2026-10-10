---
title: "2. Quickstart"
weight: 2
---

# Quickstart

## 📦 Installation

```bash
go get github.com/BeamFoundry/rays
```

## 🚀 The Minimal Application

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
