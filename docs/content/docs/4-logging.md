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

    # Custom log format pattern (used only when handler is CUSTOM).
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
| `TEXT` | Standard `logfmt` key-value style | `%time(%ak=%"2006-01-02T15:04:05.999Z07:00"v )%level(%ak=%av) %source(%ak=%av )%msg(%ak=%qv)%attr( %ak=%qv)%grp(%attr( %.ag=%qv))` |
| `JSON` | Compact, standard JSON log output | `{%time(%qk:%"RFC3339Nano"qv)%level(,%qk:%qv)%source(,%qk:%"json"v)%msg(,%qk:%qv)%attr(,%qk:%qv)%grp(,%qg:{%attr(%qk:%qv%nl(,))})}` |
| `ECS` | Elastic Common Schema (ECS v1.2.0) JSON | `{%time("@timestamp":%"RFC3339"qv,)%level("log.level":%qv),%msg("message":%qv),"ecs.version":"1.2.0"%source(,"log.origin":%"json"v)%attr(,%qk:%qv)%grp(,%qg:{%attr(%qk:%qv%nl(,))})}` |
| `GELF` | Graylog Extended Log Format (GELF v1.1) JSON | `{"version":"1.1",%host("host":%qv)%time(,"timestamp":%"UnixMilli"v)%level(,"level":%"syslog"v)%msg(,"short_message":%qv)%source(,"_source":%qv)%attr(,%_qk:%qv)%grp(%attr(,%_qg:%qv))}` |
| `LOGSTASH` | Logstash JSON event format | `{"@version":"1",%time("@timestamp":%"RFC3339"qv,)%level("level":%qv),%msg("message":%qv)%source(,"source":%"json"v)%attr(,%qk:%qv)%grp(,%qg:{%attr(%qk:%qv%nl(,))})}` |
| `CUSTOM` | Custom format defined by the `format` configuration property | Defined via `format:` |

---

## Pattern Layout Engine

Rays features a compiled pattern layout engine. The format pattern string is parsed once into an Abstract Syntax Tree (AST) when the handler is initialized or when `SetFormat()` is called. During log execution, log records are written through the compiled AST directly into pooled byte buffers, eliminating runtime regular expressions and string re-parsing.

### Pattern Placeholders

| Placeholder | Description | Inner Value Targets |
| :--- | :--- | :--- |
| `%time(...)` | Renders the log record timestamp | Generic placeholders `%v`, `%qv`, `%av`, or custom layout `%"<layout>"v`, `%"<layout>"qv`, `%"<layout>"av` |
| `%level(...)` | Renders the record log level | Generic placeholders `%v`, `%-5v`, `%k=%v`, `%qv`, `%av`, or syslog severity `%"syslog"v` |
| `%source(...)` | Renders the caller source location (only if `source: true`) | Generic placeholders `%v`, `%qv`, `%av`, `%k=%v`, or custom layout `%"json"v` |
| `%msg(...)` | Renders the log message | Generic placeholders `%v`, `%qv`, `%av`, `%k=%qv` |
| `%host(...)` | Renders the machine hostname (cached) | Generic placeholders `%v`, `%qv`, `%av`, `host=%qv` |
| `%attr(...)` | Iterates over top-level structured attributes | Generic placeholders `%k`, `%qk`, `%ak`, `%v`, `%qv`, `%av` and delimiter `%nl(...)` |
| `%grp(...)` | Iterates over attribute groups | Group placeholders `%g`, `%qg`, `%ag`, dotted `%.g`, `%.qg`, `%.ag`, underscore `%_g`, `%_qg`, `%_ag`, nested `%attr(...)` |
| `%nl(...)` | Renders its contents only if the current item is **not the last** | Static delimiter (e.g. `%nl(,)` or `%nl( )`), used inside `%attr(...)` |
| `%clr(...)` | Wraps inner content in ANSI color and mode escape sequences | See [ANSI Styling](#ansi-styling) below |

**Short Aliases & Escape Sequences**:
- `%t(...)`: Alias for `%time(...)`
- `%l(...)`: Alias for `%level(...)`
- `%m(...)`: Alias for `%msg(...)`
- `%s(...)`: Alias for `%source(...)`
- `%h(...)`: Alias for `%host(...)`
- `%%`: Escapes a literal percent sign (`%`)

### Generic Placeholders & Specifiers

- **Generic Placeholders (`%k`, `%qk`, `%ak`, `%v`, `%qv`, `%av`)**:
  These generic placeholders apply across field placeholders and structured attributes:
  - In `%time(...)`: `%k` represents `"time"`, and `%v` represents the formatted timestamp value.
  - In `%level(...)`: `%k` represents `"level"`, and `%v` represents the level name (e.g. `INFO`).
  - In `%source(...)`: `%k` represents `"source"`, and `%v` represents the caller source information.
  - In `%msg(...)`: `%k` represents `"msg"`, and `%v` represents the log message.
  - In `%host(...)`: `%k` represents `"host"`, and `%v` represents the system hostname.
  - In `%attr(...)`: `%k` represents the attribute key, and `%v` represents the attribute value.
  - **Quoting variations**:
    - `%k` and `%v`: Raw, unquoted values as-is.
    - `%qk`: Quoted counterpart for keys, always enclosed in double quotes (e.g. `"key"`), escaped for JSON or quoted logfmt.
    - `%qv`: Quoted counterpart for values. Quoting depends on attribute type: strings, durations, and timestamp strings are enclosed in double quotes (`"value"`), while numbers (integers, floats) and booleans remain unquoted, `nil` renders as `null`, and structured values (maps, structs, slices) render as JSON objects or arrays.
    - `%ak` and `%av`: Automatic quoted counterparts. Values and keys are printed unquoted by default, but automatically quoted with double quotes if they contain spaces, equals signs (`=`), double quotes (`"`), control characters, or if empty. For `%av`, precision truncation truncates the underlying value before automatic quoting, and width modifiers apply padding outside the quoted string.

- **Attribute Delimiter (`%nl`)**:
  `%nl(...)` applies **only inside `%attr(...)`**. It outputs its enclosed content only if the current attribute is **not the last** item in the iteration, making it ideal for delimiters (such as `%nl(,)` or `%nl( )`).

- **Group Specifiers (`%g`, `%qg`, `%ag`)**:
  Apply **inside `%grp(...)`** to render the group name:
  - `%g`: Raw unquoted group name.
  - `%qg`: Always quoted group name.
  - `%ag`: Automatic quoted group name (quoted only if containing spaces, equals signs, quotes, or control characters).
  Attributes within the group are iterated using a nested `%attr(...)`.

- **Dotted Group Key Specifiers (`%.g`, `%.qg`, `%.ag`)**:
  Apply inside nested `%attr(...)` within `%grp(...)` to format the combined dotted group and attribute key (`group.key`):
  - `%.g`: Raw unquoted dotted key (e.g. `metric.cpu`).
  - `%.qg`: Always quoted dotted key (e.g. `"metric.cpu"`).
  - `%.ag`: Automatically quoted dotted key (e.g. `metric.cpu` remains unquoted, but `"system metrics.cpu usage"` is quoted as a single combined string if either part contains spaces or equals signs).

- **Prefixed Key & Group Specifiers**:
  - `%_k`, `%_qk`, `%_ak`: Underscore-prefixed attribute keys (`_key`, `"_key"`), used in formats like GELF.
  - `%_g`, `%_qg`, `%_ag`: Underscore-prefixed group and attribute keys (`_group_key`, `"_group_key"`), prefixing the group and separating it from the key with an underscore.
  - `%.k`, `%.qk`, `%.ak`: Dot-prefixed attribute keys (`.key`, `".key"`).
  - Custom prefixes: A custom prefix can be specified in double quotes before the specifier, e.g. `%"custom"k` (yielding `customkey`) or `%"custom"g` (yielding `customgroup`).
  - Custom prefix combinations: Custom prefixes can be combined with dot or underscore notation:
    - `%"custom".k`: Key prefixed with a dot (`custom.key`).
    - `%"custom"_k`: Key prefixed with an underscore (`custom_key`).
    - `%"custom".g`: Dotted group key with custom prefix (`custom.group.key`).
    - `%"custom"_g`: Underscore group key with custom prefix (`custom_group_key`).

### Padding and Truncation Modifiers

You can format placeholders with minimum widths, maximum widths, and alignment:

- `%-5v`: Left-align with minimum width 5 (pads with trailing spaces, e.g. `INFO `)
- `%5v`: Right-align with minimum width 5 (pads with leading spaces, e.g. ` INFO`)
- `%.10v`: Truncate to maximum 10 characters from the start (keeps trailing characters)
- `%.-10v`: Truncate to maximum 10 characters from the end (keeps leading characters)
- `%-5.10v`: Combined left-align minimum 5 characters, maximum 10 characters

---

## Level Formatting & Syslog Severity

The `%level(...)` placeholder formats the log level name by default (`INFO`, `WARN`, `ERROR`, `FATAL`, `TRACE`). It supports padding and alignment modifiers (e.g. `%-5v`, `%5v`), generic keys (`%k`), unquoted and quoted values (`%v`, `%qv`, `%av`), and the dedicated `%"syslog"v` layout:

- `%"syslog"v`: Formats the level as its numeric RFC 5424 syslog severity code:
  - `FATAL`: `2` (Critical)
  - `ERROR`: `3` (Error)
  - `WARN`: `4` (Warning)
  - `INFO`: `6` (Informational)
  - `DEBUG` / `TRACE`: `7` (Debug)

---

## Source Formatting

When `source: true` is configured, `%source(...)` extracts runtime caller information:

- **Unquoted (`%v` or `%av`)**: Renders human-readable caller location with reduced package names, formatted as `function(file:line)` (e.g. `main.run(server.go:42)`).
- **Quoted (`%qv`)**: Emits the human-readable caller location as a quoted string (e.g. `"main.run(server.go:42)"`).
- **Custom JSON Layout (`%"json"v`)**: Emits a structured JSON object containing the file name, line number, and function name, bypassing `ReplaceAttr`:
  ```json
  {"file":{"name":"server.go","line":42},"function":"main.run"}
  ```

---

## Time Formatting

The `%time(...)` placeholder supports custom formatting layouts specified in double quotes before the target specifier:

```
%time(%"<layout>"v)
%time(%"<layout>"qv)
%time(%"<layout>"av)
```

When no custom layout is specified (such as `%time(%v)`), the default timestamp format used by Rays is `"2006/01/02 15:04:05.000"` (unquoted with `%v`, automatically quoted with `%av`, and always quoted with `%qv`).

### Predefined Epoch & Timestamp Layouts

Rays adds dedicated Unix epoch time layouts for high-precision and Unix-compatible timestamps:

| Layout Name | Description | Output Example |
| :--- | :--- | :--- |
| `"Unix"` | Epoch time in seconds | `1700000000` |
| `"UnixMilli"` | Epoch time with millisecond decimal precision (seconds.milliseconds) | `1700000000.123` |
| `"UnixMicro"` | Epoch time with microsecond decimal precision (seconds.microseconds) | `1700000000.123456` |
| `"UnixNano"` | Epoch time with nanosecond decimal precision (seconds.nanoseconds) | `1700000000.123456789` |

In addition to the Unix layouts above, all standard Go time format constants (such as `RFC3339`, `RFC3339Nano`, `DateTime`, `DateOnly`, `TimeOnly`, `ANSIC`, `UnixDate`, `RubyDate`, `RFC822`, `RFC822Z`, `RFC850`, `RFC1123`, `RFC1123Z`, `Kitchen`, `Stamp`, `StampMilli`, `StampMicro`, `StampNano`), ISO 8601 aliases (`iso8601`, `iso_8601`, `iso-8601`), and snake_case variants (e.g. `unix_milli`, `rfc3339_nano`, `date_time`, `date_only`, `time_only`, `unix_date`, `ruby_date`, `stamp_milli`, etc.) are supported case-insensitively by name. Refer to the [Go time package constants table](https://pkg.go.dev/time#pkg-constants) for the complete list of standard format names.

### Custom Go Time Layouts

Any standard Go reference time layout (`2006-01-02 15:04:05`) can be passed directly as the layout:

```yaml
rays:
  logging:
    handler: CUSTOM
    format: '[%time(%"2006/01/02 - 15:04:05.000"v)] %level(%-5v) %msg(%v)'
```

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
| `blinky` / `blink` | `5` | Blinking text |

### Supported Colors

| Color Name | Foreground Code | Color Name | Foreground Code |
| :--- | :--- | :--- | :--- |
| `black` | `30` | `bright_black` | `90` |
| `red` | `31` | `bright_red` | `91` |
| `green` | `32` | `bright_green` | `92` |
| `yellow` | `33` | `bright_yellow` | `93` |
| `blue` | `34` | `bright_blue` | `94` |
| `magenta` | `35` | `bright_magenta` | `95` |
| `cyan` | `36` | `bright_cyan` | `96` |
| `white` | `37` | `bright_white` | `97` |
| `default` | `39` | | |

> [!NOTE]
> Bright color names can be specified with either underscores or hyphens (e.g. `bright_black` or `bright-black`, `bright_cyan` or `bright-cyan`).

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
