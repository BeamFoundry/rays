---
title: "Enum Types"
weight: 2
---

# Type-Safe Enums

Go does not have native, strict enums. This often leads to configuration bugs where an invalid string from a YAML file
is happily parsed into a `string` field, only to fail deep within the application logic.

Rays solves this with `Enum[T]`. It strictly binds string values from YAML into custom Go types and validates them
during the framework's Instantiation phase.

### 1. Define the Enum Type
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

### 2. Use it in Configuration
Simply declare the field using your custom type. Note that it works seamlessly without wrapping the struct field in generic notation.

```go
import . "github.com/BeamFoundry/rays/pkg/core"

type AppConfig struct {
    ConfigurationProperties `Name:"app"`
    level LogLevel
}
```

### 3. Safe Access in Business Logic
Because of the generic structuring, you must initialize the enum type by calling `.New()` to ensure its internal state
is properly configured before accessing its variants.

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
