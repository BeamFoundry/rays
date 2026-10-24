---
title: "Configuration & Profiles"
weight: 4
---

# Configuration & Profiles

In enterprise applications, configuration and environment management go hand-in-hand.

Rays provides a unified system to bind YAML configurations directly to your structs, override them based on active
profiles (e.g., `dev`, `prod`), and conditionally load specific components depending on the environment.

## 1. Binding Data

Embed `ConfigurationProperties` and define the YAML path via the `Name` tag.

**NOTE:** *Rays strictly enforces a 1-to-1 parity between your YAML files and your Go code. The `yaml` tag is **not
honored**. Only properties with the exact same case name as the Go struct field will be injected.*

### Valid types

All numeric types, `bool` and `string` are honored.

Rays includes native support for `time.Duration`. Text values like `"5000ms"` in the YAML file are automatically parsed
using Go's standard `time.ParseDuration` function.

Struct and struct pointers (sub-object mapping) are supported.

Struct implementing the `encoding.TextUnmarshaler` interface can also be used (string to struct mapping). The struct
type (not a pointer to that) can be used directly.

Slices of all the above types are supported.

Finally map of `string` and of `encoding.TextUnmarshaler` to all the above types and to other map of those two types are
supported.

### Validation

Validation is done using the [Validation](github.com/go-playground/validator) project, given you provides the
corresponding tags.

NOTE: To validate private fields on a struct, reflection structs will be constructed internally with public fields
instead. If using Cross-Field validation, fields should be indicated in their public form. Errors will be formatted
using the original private field instead.

### Example

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
## 2. Profile-Based Loading

Rays natively supports Spring-like environment profiles. You define a base configuration file, and then create environment-specific overrides using a suffix convention (`-<profile>.yaml`).

When the application boots, the base file is always loaded. If a specific profile is activated, its corresponding file is loaded *on top*, overriding the base values.

### The YAML Files

Place your configuration files in a directory (e.g., `config/`).

```yaml
# config/application.yaml (Loaded for everyone)
app:
  kafka:
    brokers: ["localhost:9092"]
    timeout: "5000ms"
    retry: true
```

```yaml
# config/application-prod.yaml (Loaded ONLY if the "prod" profile is active)
app:
  kafka:
    brokers: ["prod-kafka-1:9092", "prod-kafka-2:9092"]
    timeout: "2000ms"
```


## 3. Conditional Registration

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

## 4. The Application Entrypoint

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
