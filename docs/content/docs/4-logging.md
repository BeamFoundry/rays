---
title: "4. Logging Framework"
weight: 4
---

# Logging Framework

Enterprise applications require structured, leveled, and contextual logging.

Rays provides a dedicated configurable, logging framework.

## The `Logging` Interface

Rays logging is powered by Go's standard `slog` library under the hood, providing high-performance, structured logging.
However, standard `slog` natively lacks a "Trace" level for highly granular debugging.

Rays extends the standard implementation by adding full support for the `Trace` level, including the corresponding `Trace()` and `TraceContext()` functions on the `Logging` interface.

### Configuration

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

### Basic Injection

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
