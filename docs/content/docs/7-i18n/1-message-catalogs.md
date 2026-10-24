---
title: "Message Catalogs & Pluralization"
weight: 1
---

# Message Catalogs & Pluralization

Rays simplifies message localization by letting you define translation catalogs directly in your YAML configuration
files. Messages are automatically loaded into `LocaleMessageBundle` instances, with full support for positional
parameters, Unicode CLDR pluralization rules, and numeric ranges.

---

## Defining Messages in YAML

Message catalogs are organized by locale under the `localization.messages` property in your `application.yaml` file.
Rays recognizes four distinct categories of localized messages:

| Category | Description | Placeholders |
| :--- | :--- | :--- |
| **`translation`** | Simple string translations. Positional parameters are denoted by `{0}`, `{1}`, `{2}`, etc. | `{0}`, `{1}`, `{2}`, ... |
| **`cardinal`** | Plural rules determined by item quantities (e.g., "1 day" vs. "5 days"). | `{0}` (the formatted number) |
| **`ordinal`** | Plural rules determined by position or rank (e.g., "1st", "2nd", "3rd"). | `{0}` (the formatted number) |
| **`ranges`** | Plural rules for intervals between two numbers (e.g., "1-3 days"). | `{0}`, `{1}` (the start and end numbers) |

### Complete YAML Example

```yaml
localization:
  messages:
    en:
      translation:
        welcome: "Welcome"
        welcome_user: "Welcome, {0} {1}!"
        fields.registration.username: "Username"
        fields.registration.password: "Password"
        fields.registration.confirm: "Confirm Password"
      cardinal:
        days.left:
          ONE: "{0} day remaining"
          OTHER: "{0} days remaining"
      ordinal:
        competition.place:
          ONE: "{0}st place"
          TWO: "{0}nd place"
          FEW: "{0}rd place"
          OTHER: "{0}th place"
      ranges:
        days.range:
          OTHER: "{0}-{1} days"

    fr:
      translation:
        welcome: "Bienvenue"
        welcome_user: "Bienvenue, {0} {1} !"
        fields.registration.username: "Nom d'utilisateur"
        fields.registration.password: "Mot de passe"
        fields.registration.confirm: "Confirmation du mot de passe"
      cardinal:
        days.left:
          ONE: "{0} jour restant"
          OTHER: "{0} jours restants"
      ordinal:
        competition.place:
          ONE: "{0}er prix"
          OTHER: "{0}e prix"
      ranges:
        days.range:
          ONE: "{0}-{1} jour"
          OTHER: "{0}-{1} jours"
```

---

## CLDR Plural Rules

Different languages follow different rules for pluralization. In English, cardinal numbers only distinguish between
`ONE` (1) and `OTHER` (0, 2, 3, ...). Other languages like Polish, Russian, or Arabic use additional categories such as
`FEW` or `MANY`.

Rays provides a type-safe `PluralRule` enum corresponding to standard Unicode CLDR plural categories:

```go
package i18n

type PluralRuleEnum struct {
    ZERO,
    ONE,
    TWO,
    FEW,
    MANY,
    OTHER PluralRule
}

type PluralRule = Enum[PluralRuleEnum]

var PluralRules = PluralRule{}.New()
```

When defining cardinal, ordinal, or range entries in YAML, specify rule names matching these enum values (e.g., `ONE`,
`FEW`, `OTHER`).

---

## Programmatic Translation API

You can look up any `LocaleMessageBundle` from a `MessageSource` to format and translate messages in your business
logic.

### 1. Simple Translations

Use `Translate()` to look up keys and substitute positional parameters:

```go
bundle := messageSource.FindMessageBundle(locales.Locales.EN)

// Without parameters
msg, err := bundle.Translate("welcome")
// Output: "Welcome"

// With positional parameters
msg, err = bundle.Translate("welcome_user", "John", "Doe")
// Output: "Welcome, John Doe!"
```

### 2. Cardinal Pluralization

Use `Cardinal()` to format quantities according to the language's plural rules. Rays automatically formats the number
according to the locale's number convention:

```go
// English: 1 -> ONE, 5 -> OTHER
msg1, _ := bundleEN.Cardinal("days.left", 1) // "1 day remaining"
msg5, _ := bundleEN.Cardinal("days.left", 5) // "5 days remaining"

// French: 1 -> ONE, 5 -> OTHER
msgFR1, _ := bundleFR.Cardinal("days.left", 1) // "1 jour restant"
msgFR5, _ := bundleFR.Cardinal("days.left", 5) // "5 jours restants"
```

#### Precision Control

Use `CardinalWithPrecision()` when working with floating-point numbers:

```go
// Formats with 1 decimal place: "2,0 jours restants" (French decimal comma)
msg, _ := bundleFR.CardinalWithPrecision("days.left", 2.0, 1)
```

### 3. Ordinal Pluralization

Use `Ordinal()` for ranks, positions, or numbered items:

```go
msg1, _ := bundleEN.Ordinal("competition.place", 1) // "1st place"
msg2, _ := bundleEN.Ordinal("competition.place", 2) // "2nd place"
msg3, _ := bundleEN.Ordinal("competition.place", 3) // "3rd place"
msg4, _ := bundleEN.Ordinal("competition.place", 4) // "4th place"
```

### 4. Numeric Ranges

Use `Range()` to express intervals between two numbers:

```go
// English: "1-5 days"
rangeEN, _ := bundleEN.Range("days.range", 1, 5)

// French: "0-1 jour" (ONE) vs. "1-5 jours" (OTHER)
rangeFR1, _ := bundleFR.Range("days.range", 0, 1) // "0-1 jour"
rangeFR5, _ := bundleFR.Range("days.range", 1, 5) // "1-5 jours"
```

Use `RangeWithPrecision()` to enforce specific decimal formatting:

```go
msg, _ := bundleFR.RangeWithPrecision("days.range", 1.0, 5.0, 1)
// Output: "1,0-5,0 jours"
```

---

## Adding Messages Programmatically

In addition to YAML configuration, you can register messages dynamically at runtime using the `LocaleMessageBundle` API:

```go
bundle := messageSource.NewMessageBundle(locales.Locales.EN)

// Add simple translation
bundle.AddTranslation("alert.timeout", "Request timed out after {0} seconds")

// Add cardinal plural rule
bundle.AddCardinal("items.count", "{0} item", PluralRules.ONE)
bundle.AddCardinal("items.count", "{0} items", PluralRules.OTHER)

// Add ordinal plural rule
bundle.AddOrdinal("rank", "{0}st", PluralRules.ONE)
bundle.AddOrdinal("rank", "{0}nd", PluralRules.TWO)
bundle.AddOrdinal("rank", "{0}rd", PluralRules.FEW)
bundle.AddOrdinal("rank", "{0}th", PluralRules.OTHER)

// Add range rule
bundle.AddRange("delivery.days", "{0}-{1} business days", PluralRules.OTHER)
```
