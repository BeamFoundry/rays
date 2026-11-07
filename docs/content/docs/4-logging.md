---
title: "4. Logging Framework"
weight: 4
---

# Logging Framework

Enterprise applications require structured, leveled, high-performance, and contextual logging.

Rays provides a dedicated, highly configurable logging framework built on top of Go's standard `log/slog` library, enriched with a compiled pattern layout engine, ANSI styling, extended log levels, and cloud-native log formatters.

---

## The `Logging` Interface

Rays logging extends standard `slog` by introducing both `Trace` (for highly granular debugging) and `Fatal` (for unrecoverable errors) levels, alongside the standard `Debug`, `Info`, `Warn`, and `Error` levels.

### Available Logging Methods

All logging levels provide both standard and context-aware variants:

| Method | Context Variant | Level Value | Description |
| :--- | :--- | :--- | :--- |
| `Trace(msg, ...args)` | `TraceContext(ctx, msg, ...args)` | `-8` | Highly granular diagnostic traces |
| `Debug(msg, ...args)` | `DebugContext(ctx, msg, ...args)` | `-4` | Detailed debugging information |
| `Info(msg, ...args)` | `InfoContext(ctx, msg, ...args)` | `0` | Informational operational events |
| `Warn(msg, ...args)` | `WarnContext(ctx, msg, ...args)` | `4` | Non-critical warnings and unusual events |
| `Error(msg, ...args)` | `ErrorContext(ctx, msg, ...args)` | `8` | Errors that impact execution |
| `Fatal(msg, ...args)` | `FatalContext(ctx, msg, ...args)` | `12` | Unrecoverable failures |

### Basic Injection

To use the logger, inject the `Logging` interface into your Beams using the `@:"Inject"` tag.

```go
package service

import (
    "context"

    . "github.com/BeamFoundry/rays/pkg/core"
    . "github.com/BeamFoundry/rays/pkg/lang"
)

type PaymentService struct {
    Service

    // Note: No pointer '*' because Logging is an interface
    logger Logging `@:"Inject"`
}

func (s *PaymentService) Process(ctx context.Context, amount float64) Error {
    s.logger.TraceContext(ctx, "Entering Process method", "amount", amount)

    if amount <= 0 {
        s.logger.WarnContext(ctx, "Invalid payment amount encountered", "amount", amount)
        return NewError("INVALID_AMOUNT", "amount must be greater than zero")
    }

    // Structured logging supports key-value pairs natively
    s.logger.InfoContext(ctx, "Payment processed successfully", "amount", amount, "currency", "USD")
    return nil
}
```

---

## Configuration

You can configure the logging framework globally and on a per-package basis using your `application.yaml` file under the `rays.logging` configuration block:

```yaml
rays:
  logging:
    # Minimum log level.
    # Valid values: "TRACE", "DEBUG", "INFO", "WARN", "ERROR", "FATAL".
    # Default: INFO
    level: DEBUG

    # Log handler format.
    # Valid values: "DEFAULT", "TEXT", "JSON", "CUSTOM", "ECS", "GELF", "LOGSTASH".
    # Default: DEFAULT
    handler: DEFAULT

    # Add source code location (reduced package, file, line) to each log entry.
    # Type: boolean
    # Default: true
    source: true

    # ANSI color output mode.
    # Valid values:
    #   AUTO - Enable color if output is an interactive terminal (TTY)
    #   ON   - Force color output even when redirected to files or pipes
    #   OFF  - Completely disable ANSI escape codes
    # Default: AUTO
    color: AUTO

    # Custom log format pattern (used when handler is CUSTOM or to override DEFAULT).
    # Default: DefaultFormat
    format: "%time(%v) %clr(%level(%-5v)) %source(%clr(%v){magenta} )%msg(%v)%clr(%attr( %k=%v)){faint}%clr(%grp( %g.%attr(%k=%v%nl( )))){faint}"

    # Override the log level for specific packages or structs.
    # - Exact match: matches only that package.
    # - Trailing slash ("/"): matches that package and all subpackages.
    # - Struct suffix: matches only logging injected into that specific struct.
    loggers:
      github.com/BeamFoundry/rays-sample/: INFO
      github.com/BeamFoundry/rays-sample/pkg/config: DEBUG
      github.com/BeamFoundry/rays-sample/pkg/service/PaymentService: TRACE
```

---

## Built-in Handlers

Rays provides several pre-configured log handlers covering local development, standard Unix environments, and cloud log aggregators:

| Handler | Description | Underlying Pattern |
| :--- | :--- | :--- |
| `DEFAULT` | Human-readable console format with ANSI colors and styles | `%time(%v) %clr(%level(%-5v)) %source(%clr(%v){magenta} )%msg(%v)%clr(%attr( %k=%v)){faint}%clr(%grp( %g.%attr(%k=%v%nl( )))){faint}` |
| `TEXT` | Standard `logfmt` key-value style | `%time(%k=%v )%level(%k=%v) %source(%k=%v )%msg(%k=%qv)%attr( %k=%qv)%grp(%attr( %g.%k=%qv))` |
| `JSON` | Compact, standard JSON log output | `{%time(%qk:%qv,)%level(%qk:%qv)%source(,%qk:%qv),%msg(%qk:%qv)%attr(,%qk:%qv)%grp(,%qg:{%attr(%qk:%qv%nl(,))})}` |
| `ECS` | Elastic Common Schema (ECS v1.2.0) JSON | `{%time("@timestamp":%"RFC3339"qv,)%level("log.level":%qv),%msg("message":%qv),"ecs.version":"1.2.0"%source(,"log.origin":%qv)%attr(,%qk:%qv)%grp(,%qg:{%attr(%qk:%qv%nl(,))})}` |
| `GELF` | Graylog Extended Log Format (GELF v1.1) JSON | `{"version":"1.1","host":"localhost",%time("timestamp":%"UnixMilli"v,)%level("level":%qv),%msg("short_message":%qv)%source(,"_source":%qv)%attr(,"_%k":%qv)%grp(,"_%g":{%attr(%qk:%qv%nl(,))})}` |
| `LOGSTASH` | Logstash JSON event format | `{"@version":"1",%time("@timestamp":%"RFC3339"qv,)%level("level":%qv),%msg("message":%qv)%source(,"source":%qv)%attr(,%qk:%qv)%grp(,%qg:{%attr(%qk:%qv%nl(,))})}` |
| `CUSTOM` | Custom format defined by the `format` configuration property | Defined via `format:` |

---

## Pattern Layout Engine

Rays features a compiled pattern layout engine. The format pattern string is parsed once into an Abstract Syntax Tree (AST) when the handler is initialized or when `SetFormat()` is called. During log execution, log records are written through the compiled AST directly into pooled byte buffers, eliminating runtime regular expressions and string re-parsing.

### Pattern Placeholders

| Placeholder | Description | Inner Value Targets |
| :--- | :--- | :--- |
| `%time(...)` | Renders the log record timestamp | `%v`, `%qv`, or custom layout `%"<layout>"v` |
| `%level(...)` | Renders the record log level | `%v`, `%-5v`, `%k=%v`, `%qv` |
| `%source(...)` | Renders the reduced source file and line (only if `source: true`) | `%v`, `%qv`, `%k=%v` |
| `%msg(...)` | Renders the log message | `%v`, `%qv`, `%k=%qv` |
| `%attr(...)` | Iterates over top-level structured attributes | `%k`, `%qk`, `%v`, `%qv` |
| `%grp(...)` | Iterates over attribute groups | `%g`, `%qg`, nested `%attr(...)` |
| `%nl(...)` | Renders its contents only if the current item is **not the last** | Static delimiter, e.g. `%nl(,)` or `%nl( )` |
| `%clr(...)` | Wraps inner content in ANSI color and mode escape sequences | See [ANSI Styling](#ansi-styling) below |

### Inner Target Specifiers

Inside placeholders like `%attr(...)` and `%grp(...)`, the following specifiers denote key, value, and group tokens:

- `%k`: Attribute key (raw string)
- `%qk`: Attribute key (quoted string, e.g. `"key"`)
- `%v`: Attribute or placeholder value (raw string)
- `%qv`: Attribute or placeholder value (quoted string, e.g. `"value"`)
- `%g`: Group name (raw string)
- `%qg`: Group name (quoted string, e.g. `"group"`)

### Padding and Truncation Modifiers

You can format placeholders with minimum widths, maximum widths, and alignment:

- `%-5v`: Left-align with minimum width 5 (pads with trailing spaces, e.g. `INFO `)
- `%5v`: Right-align with minimum width 5 (pads with leading spaces, e.g. ` INFO`)
- `%.10v`: Truncate to maximum 10 characters from the start (keeps trailing characters)
- `%.-10v`: Truncate to maximum 10 characters from the end (keeps leading characters)
- `%-5.10v`: Combined left-align minimum 5 characters, maximum 10 characters

---

## Time Formatting

The `%time(...)` placeholder supports custom formatting layouts specified in double quotes before the target specifier:

```
%time(%"<layout>"v)
%time(%"<layout>"qv)
```

### Predefined Epoch & Timestamp Layouts

| Layout Name | Description | Output Example |
| :--- | :--- | :--- |
| `"Unix"` | Epoch time in seconds | `1700000000` |
| `"UnixMilli"` | Epoch time with millisecond decimal precision (seconds.milliseconds) | `1700000000.123` |
| `"UnixMicro"` | Epoch time with microsecond decimal precision (seconds.microseconds) | `1700000000.123456` |
| `"UnixNano"` | Epoch time with nanosecond decimal precision (seconds.nanoseconds) | `1700000000.123456789` |
| `"RFC3339"` | Standard ISO-8601 / RFC 3339 format | `2026-11-05T12:00:00Z` |
| `"RFC3339Nano"` | RFC 3339 with nanoseconds | `2026-11-05T12:00:00.123456789Z` |
| `"DateTime"` | Standard date and time (`"2006-01-02 15:04:05"`) | `2026-11-05 12:00:00` |
| `"DateOnly"` | Date only (`"2006-01-02"`) | `2026-11-05` |
| `"TimeOnly"` | Time only (`"15:04:05"`) | `12:00:00` |
| `"ANSIC"` | ANSI C `asctime()` layout | `Thu Nov  5 12:00:00 2026` |
| `"Kitchen"` | 12-hour kitchen time format | `12:00PM` |
| `"StampMilli"` | Month, day, and time with milliseconds | `Nov  5 12:00:00.000` |

### Custom Go Time Layouts

Any standard Go reference time string (`2006-01-02 15:04:05`) can be passed directly as the layout:

```yaml
rays:
  logging:
    handler: CUSTOM
    format: '[%time(%"2006/01/02 - 15:04:05.000"v)] %level(%-5v) %msg(%v)'
```

When no custom layout is provided (`%time(%v)`), the handler's configured `ReplaceAttr` function or default timestamp format is used.

---

## ANSI Styling

Rays supports full ANSI styling with independent color and mode combinations.

### Syntax

```
%clr(<inner>)
%clr(<inner>){<styles>}
```

`<styles>` is a comma-separated list of zero or more **modes** and at most one **color** (e.g. `{cyan,bold}`, `{yellow,faint}`, `{magenta}`).

### Supported Modes

| Mode Name | ANSI Code | Description |
| :--- | :--- | :--- |
| `normal` | `0` | Normal style / reset (default) |
| `bold` | `1` | Bold or increased intensity |
| `faint` | `2` | Faint or decreased intensity |
| `italic` | `3` | Italicized text |
| `underline` | `4` | Underlined text |
| `blinky` | `5` | Blinking text |

### Supported Colors

| Color Name | Foreground Code |
| :--- | :--- |
| `black` | `30` |
| `red` | `31` |
| `green` | `32` |
| `yellow` | `33` |
| `blue` | `34` |
| `magenta` | `35` |
| `cyan` | `36` |
| `white` | `37` |
| `default` | `39` |

### Mode and Color Combinations

Styles can be freely combined. For example:

- `{cyan,bold}`: emits `\033[1;36m`
- `{yellow,faint}`: emits `\033[2;33m`
- `{underline,red}`: emits `\033[4;31m`
- `{faint}`: emits `\033[2m` (preserves terminal text color)
- `{normal}`: emits `\033[0m`

### Default Level Colors

When `%clr(...)` wraps `%level(...)` without an explicit color name, the level's default color is automatically selected:

| Level | Default Color | ANSI Code | Combined with `{bold}` | Combined with `{faint}` |
| :--- | :--- | :--- | :--- | :--- |
| `FATAL` | Red | `\033[31m` | `\033[1;31m` | `\033[2;31m` |
| `ERROR` | Red | `\033[31m` | `\033[1;31m` | `\033[2;31m` |
| `WARN` | Yellow | `\033[33m` | `\033[1;33m` | `\033[2;33m` |
| `INFO` | Green | `\033[32m` | `\033[1;32m` | `\033[2;32m` |
| `DEBUG` | Cyan | `\033[36m` | `\033[1;36m` | `\033[2;36m` |
| `TRACE` | Reset / Normal | `\033[0m` | `\033[1m` | `\033[2m` |

If you specify a mode only for `%level(...)` (such as `%clr(%level(%-5v)){bold}`), the mode is merged with the level's color! For example, `INFO` becomes bold green (`\033[1;32m`), `DEBUG` becomes bold cyan (`\033[1;36m`), and `WARN` becomes bold yellow (`\033[1;33m`).

For non-level elements (such as `%time`, `%source`, or `%attr`), specifying only a mode (e.g. `{faint}`) applies only that mode code (`\033[2m`) without forcing white or any other foreground color, correctly preserving the terminal's default color.

---

## The Default Console Format

The default format used by Rays is:

```
%time(%v) %clr(%level(%-5v)) %source(%clr(%v){magenta} )%msg(%v)%clr(%attr( %k=%v)){faint}%clr(%grp( %g.%attr(%k=%v%nl( )))){faint}
```

This configuration provides:
- **Timestamp**: clean timestamp without color interference.
- **Level**: fixed 5-character width, color-coded automatically by level (green for `INFO`, cyan for `DEBUG`, yellow for `WARN`, red for `ERROR`/`FATAL`, reset for `TRACE`).
- **Source**: concise reduced function, file, and line formatted in magenta.
- **Message**: clean, uncolored text for readability.
- **Attributes & Groups**: subtle faint text (`{faint}`) so key-value metadata is easily readable without distracting from the main log message.
