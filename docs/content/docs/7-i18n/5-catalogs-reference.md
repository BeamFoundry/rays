---
title: "Predefined Catalogs Reference"
weight: 5
---

# Predefined Catalogs Reference

The `github.com/BeamFoundry/rays/pkg/i18n/locales` package eliminates boilerplate by providing pre-instantiated singletons for standard locales and pre-registered validation translation functions.

---

## Predefined Locales

The `Locales` struct contains pre-instantiated [Go-Playground Locales](https://github.com/go-playground/locales) singletons for over 200 standard language and territory combinations.

### Common Locales

```go
import "github.com/BeamFoundry/rays/pkg/i18n/locales"

// English
locales.Locales.EN      // English (base)
locales.Locales.EN_US   // English (United States)
locales.Locales.EN_GB   // English (United Kingdom)
locales.Locales.EN_CA   // English (Canada)
locales.Locales.EN_AU   // English (Australia)

// French
locales.Locales.FR      // French (base)
locales.Locales.FR_FR   // French (France)
locales.Locales.FR_CA   // French (Canada)
locales.Locales.FR_BE   // French (Belgium)
locales.Locales.FR_CH   // French (Switzerland)

// German
locales.Locales.DE      // German (base)
locales.Locales.DE_DE   // German (Germany)
locales.Locales.DE_AT   // German (Austria)
locales.Locales.DE_CH   // German (Switzerland)

// Spanish
locales.Locales.ES      // Spanish (base)
locales.Locales.ES_ES   // Spanish (Spain)
locales.Locales.ES_MX   // Spanish (Mexico)

// Other European Languages
locales.Locales.IT      // Italian
locales.Locales.PT      // Portuguese
locales.Locales.PT_BR   // Portuguese (Brazil)
locales.Locales.NL      // Dutch
locales.Locales.PL      // Polish
locales.Locales.RU      // Russian
locales.Locales.UK      // Ukrainian

// Asian Languages
locales.Locales.JA      // Japanese
locales.Locales.KO      // Korean
locales.Locales.ZH      // Chinese (Simplified)
locales.Locales.ZH_TW   // Chinese (Traditional)
locales.Locales.VI      // Vietnamese
locales.Locales.TH      // Thai
locales.Locales.ID      // Indonesian

// Middle Eastern Languages
locales.Locales.AR      // Arabic
locales.Locales.FA      // Persian / Farsi
locales.Locales.TR      // Turkish
```

### Usage with Message Bundles

Pass any `Locales` singleton to `messageSource.NewMessageBundle()` or `messageSource.FindMessageBundle()`:

```go
bundle := messageSource.NewMessageBundle(locales.Locales.DE)
```

---

## Predefined Validation Translations

The `ValidationTranslations` struct provides ready-to-use registration functions that load built-in translation catalogs for Go-Playground Validator into a `LocaleMessageBundle`.

### Loading Validation Translations

```go
import "github.com/BeamFoundry/rays/pkg/i18n/locales"

// Load English validator translations
if err := bundleEN.LoadValidationTranslation(locales.ValidationTranslations.EN); err != nil {
    return err
}

// Load French validator translations
if err := bundleFR.LoadValidationTranslation(locales.ValidationTranslations.FR); err != nil {
    return err
}
```

### Supported Languages (22 Translations)

| Code | Language | Registration Constant |
| :--- | :--- | :--- |
| `AR` | Arabic | `locales.ValidationTranslations.AR` |
| `DE` | German | `locales.ValidationTranslations.DE` |
| `EN` | English | `locales.ValidationTranslations.EN` |
| `ES` | Spanish | `locales.ValidationTranslations.ES` |
| `FA` | Persian (Farsi) | `locales.ValidationTranslations.FA` |
| `FR` | French | `locales.ValidationTranslations.FR` |
| `ID` | Indonesian | `locales.ValidationTranslations.ID` |
| `IT` | Italian | `locales.ValidationTranslations.IT` |
| `JA` | Japanese | `locales.ValidationTranslations.JA` |
| `KO` | Korean | `locales.ValidationTranslations.KO` |
| `LV` | Latvian | `locales.ValidationTranslations.LV` |
| `NL` | Dutch | `locales.ValidationTranslations.NL` |
| `PL` | Polish | `locales.ValidationTranslations.PL` |
| `PT` | Portuguese | `locales.ValidationTranslations.PT` |
| `PT_BR` | Portuguese (Brazil) | `locales.ValidationTranslations.PT_BR` |
| `RU` | Russian | `locales.ValidationTranslations.RU` |
| `TH` | Thai | `locales.ValidationTranslations.TH` |
| `TR` | Turkish | `locales.ValidationTranslations.TR` |
| `UK` | Ukrainian | `locales.ValidationTranslations.UK` |
| `VI` | Vietnamese | `locales.ValidationTranslations.VI` |
| `ZH` | Chinese (Simplified) | `locales.ValidationTranslations.ZH` |
| `ZH_TW` | Chinese (Traditional) | `locales.ValidationTranslations.ZH_TW` |
