---
title: "7. Internationalization"
weight: 7
bookCollapseSection: false
---

# Internationalization & Localization

Rays provides an enterprise-ready internationalization (i18n) and localization (L10n) framework designed for modular Go applications and web services. Built on top of the established [Go-Playground Locales](https://github.com/go-playground/locales) and [Go-Playground Validator](https://github.com/go-playground/validator) libraries, Rays unifies message translation, CLDR-compliant pluralization, request-scoped locale resolution, and multilingual form validation into a cohesive, declarative developer experience.

## Key Features

- **Declarative Message Loading**: Bind multilingual message catalogs directly from YAML configuration files using `ConfigurationProperties`.
- **Request-Scoped Context**: Manage active locales per HTTP request or goroutine using `ThreadLocaleContextHolder` without threading context parameters manually through every function.
- **Pluralization & Ranges**: Full support for CLDR cardinal plurals, ordinal plurals, and numeric ranges with localized number formatting and precision control.
- **Predefined Catalogs**: Immediate access to over 200 standard locales and 22+ built-in validator translation sets via `pkg/i18n/locales`.
- **Localized Struct Validation**: Automatically validate structs and translate field-level validation errors into user-friendly localized messages with namespace and cross-field support.
- **HTML Template Integration**: Seamlessly expose `translate`, `cardinal`, `ordinal`, and `ranges` functions to Go's standard `html/template` engine.

---

## Architecture & Data Flow

The Rays i18n subsystem is composed of five core components working together:

```
                  +--------------------------------+
                  |      YAML Configuration        |
                  |     (application.yaml)         |
                  +---------------+----------------+
                                  |
                                  v
+--------------------+   +--------------------+
|  HTTP Middleware   |   |   MessageSource    |
| (Cookie, Header,   |   |  (Registry & Cache)|
|  Query Parameter)  |   +----+---------------+
+---------+----------+        |
          |                   | creates / retrieves
          v                   v
+--------------------+   +--------------------+
|LocaleContextHolder |   |LocaleMessageBundle |
| (Thread/Request)   |   | (Translations per  |
+---------+----------+   |      locale)       |
          |              +----+---------------+
          +---------+---------+
                    |
          +---------+---------+
          |                   |
          v                   v
+--------------------+ +--------------------+
|MessageSource       | |   LocaleFuncMap    |
|Validator           | | (HTML Templates)   |
| (Form Validation)  | +--------------------+
+--------------------+
```

### Component Breakdown

| Component | Interface / Type | Package | Description |
| :--- | :--- | :--- | :--- |
| **Locale Context** | `LocaleContextHolder` | `pkg/i18n` | Stores and retrieves the active locale. `ThreadLocaleContextHolder` provides thread-local isolation for HTTP requests; `DefaultLocaleContextHolder` provides global locking for simple or CLI apps. |
| **Message Source** | `MessageSource` (`DefaultMessageSource`) | `pkg/i18n` | Central registry holding the `UniversalTranslator`, `validator.Validate`, and cached `LocaleMessageBundle` instances. |
| **Message Bundle** | `LocaleMessageBundle` (`DefaultLocaleMessageBundle`) | `pkg/i18n` | Manages translation keys, cardinal/ordinal plurals, and numeric ranges for a specific locale. |
| **Validation Engine** | `MessageSourceValidator` (`DefaultMessageSourceValidator`) | `pkg/i18n` | Validates struct pointers and translates validation error messages according to the active locale and field namespaces. |
| **Template Helpers** | `LocaleFuncMap` | `pkg/web` | Implements `FuncMapper` to supply `translate`, `cardinal`, `ordinal`, and `ranges` helpers to `html/template`. |

---

## Guide Topics

Explore the sections below to learn how to configure and use Rays internationalization:

- **[1. Message Catalogs & Pluralization](1-message-catalogs/)**: Learn how to define messages in YAML, understand the 4 message types (`translation`, `cardinal`, `ordinal`, `ranges`), use `PluralRules`, and translate strings programmatically.
- **[2. Container Configuration](2-configuration/)**: Wire the i18n infrastructure into the Rays container as a `Configuration` Beam with `PostConstruct()`.
- **[3. Web & Template Integration](3-web-and-templates/)**: Set request-scoped locales in HTTP middleware and integrate localization helpers into standard Go `html/template` templates.
- **[4. Localized Struct Validation](4-validation/)**: Validate struct inputs and translate error messages into user-friendly localized text with namespaces and cross-field rules.
- **[5. Predefined Catalogs Reference](5-catalogs-reference/)**: Access pre-built singletons for 200+ locales and 22+ built-in validator translation sets from `pkg/i18n/locales`.

---

## Quick Reference

| Task | Method / Code Snippet |
| :--- | :--- |
| **Set Request Locale** | `holder.Set(locales.Locales.FR)` |
| **Reset Request Locale** | `holder.Clear()` |
| **Get Current Locale** | `locale := holder.Get()` |
| **Simple Translation** | `bundle.Translate("key", "param0", "param1")` |
| **Cardinal Plural** | `bundle.Cardinal("key", count)` |
| **Ordinal Plural** | `bundle.Ordinal("key", position)` |
| **Number Range** | `bundle.Range("key", start, end)` |
| **Validate Struct** | `validator.Validate(&formStruct)` |
| **Translate Errors** | `validator.TranslateValidationErrors(errs, "namespace")` |
| **Template Helpers** | `{{ translate "key" }}`, `{{ cardinal "key" .Count }}` |
