---
title: "Container Configuration"
weight: 2
---

# Container Configuration

Rays integrates internationalization cleanly into its Inversion of Control (IoC) container. By declaring an i18n `Configuration` Beam, you can bind YAML configuration properties, configure default and auxiliary locales, load validator translation packs, and publish i18n services for injection throughout your application.

---

## 1. Defining Configuration Properties

First, define a struct embedding `ConfigurationProperties` to bind the `localization` section from your `application.yaml` file:

```go
package config

import (
    . "github.com/BeamFoundry/rays/pkg/core"
    . "github.com/BeamFoundry/rays/pkg/i18n"
)

// LocalizationProperties binds messages from application.yaml under "localization".
type LocalizationProperties struct {
    ConfigurationProperties `Name:"localization"`
    messages                map[string]LocalizedMessages
}
```

Rays automatically maps the YAML properties to the internal `LocalizedMessages` struct containing `translation`, `cardinal`, `ordinal`, and `ranges` maps.

---

## 2. The Localization Configuration Beam

Next, define your `LocalizationConfig` struct embedding `Configuration`. During the container's `PostConstruct()` lifecycle hook, it initializes the `ThreadLocaleContextHolder`, creates the central `DefaultMessageSource`, and registers each supported locale:

```go
package config

import (
    . "github.com/BeamFoundry/rays/pkg/core"
    . "github.com/BeamFoundry/rays/pkg/i18n"
    . "github.com/BeamFoundry/rays/pkg/i18n/locales"
    . "github.com/BeamFoundry/rays/pkg/lang"
    . "github.com/BeamFoundry/rays/pkg/web"
)

// LocalizationConfig configures and publishes all i18n Beams.
type LocalizationConfig struct {
    Configuration
    localizations       *LocalizationProperties `@:"Inject"`
    localeContextHolder *ThreadLocaleContextHolder
    messageSource       *DefaultMessageSource
}

// Compile-time check ensuring Stereotype compliance
var _, _ = any(&LocalizationConfig{}).(Stereotype)

// PostConstruct initializes message bundles and validators after injection.
func (this *LocalizationConfig) PostConstruct() Error {
    defaultLocale := Locales.EN

    // 1. Initialize thread-scoped locale holder and central message source
    this.localeContextHolder = ThreadLocaleContextHolder{}.New(defaultLocale)
    this.messageSource = DefaultMessageSource{}.New(defaultLocale)

    // 2. Configure default bundle (English)
    bundleEN := this.messageSource.DefaultMessageBundle()
    if err := bundleEN.LoadValidationTranslation(ValidationTranslations.EN); err != nil {
        return NewErrorFrom(err)
    }
    if this.localizations != nil && this.localizations.messages != nil {
        bundleEN.LoadLocalizedMessages(this.localizations.messages["en"])
    }

    // 3. Configure additional bundle (French)
    bundleFR := this.messageSource.NewMessageBundle(Locales.FR)
    if err := bundleFR.LoadValidationTranslation(ValidationTranslations.FR); err != nil {
        return NewErrorFrom(err)
    }
    if this.localizations != nil && this.localizations.messages != nil {
        bundleFR.LoadLocalizedMessages(this.localizations.messages["fr"])
    }

    return nil
}

// LocaleContextHolder exposes the request-scoped locale holder.
func (this *LocalizationConfig) LocaleContextHolder() LocaleContextHolder {
    return this.localeContextHolder
}

// MessageSource exposes the central message source registry.
func (this *LocalizationConfig) MessageSource() MessageSource {
    return this.messageSource
}

// MessageSourceValidator exposes the struct validation engine.
func (this *LocalizationConfig) MessageSourceValidator() MessageSourceValidator {
    return DefaultMessageSourceValidator{}.New(this.localeContextHolder, this.messageSource)
}

// LocaleFuncMap exposes the template function map for html/template.
func (this *LocalizationConfig) LocaleFuncMap() *LocaleFuncMap {
    return LocaleFuncMap{}.New(this.localeContextHolder, this.messageSource)
}
```

---

## 3. Published Beams & Their Purposes

By exposing factory methods on `LocalizationConfig`, the container makes the following Beams available for injection:

| Beam Type | Interface | Typical Use Case |
| :--- | :--- | :--- |
| `LocaleContextHolder` | Interface | HTTP middlewares to set the active locale per request; services needing to query the active locale. |
| `MessageSource` | Interface | Services needing direct programmatic access to locale message bundles. |
| `MessageSourceValidator` | Interface | API controllers and form handlers validating incoming requests and translating error messages. |
| `*LocaleFuncMap` | Pointer to struct | Template rendering pipelines needing i18n functions (`translate`, `cardinal`, etc.). |

---

## 4. Injecting into Application Services

Once configured, any application service can inject the i18n Beams using standard Rays `@:"Inject"` tags:

```go
package services

import (
    . "github.com/BeamFoundry/rays/pkg/core"
    . "github.com/BeamFoundry/rays/pkg/i18n"
    . "github.com/BeamFoundry/rays/pkg/lang"
)

type NotificationService struct {
    Service
    messageSource MessageSource       `@:"Inject"`
    contextHolder LocaleContextHolder `@:"Inject"`
    logger        Logging             `@:"Inject"`
}

func (s *NotificationService) SendWelcomeEmail(userName, email string) Error {
    // Look up the bundle matching the current user's request locale
    currentLocale := s.contextHolder.Get()
    bundle := s.messageSource.FindMessageBundle(currentLocale)

    welcomeText, err := bundle.Translate("welcome_user", userName)
    if err != nil {
        return NewErrorFrom(err)
    }

    s.logger.Info("Sending localized email", "to", email, "subject", welcomeText)
    return nil
}
```
