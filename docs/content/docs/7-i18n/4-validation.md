---
title: "Localized Struct Validation"
weight: 4
---

# Localized Struct Validation

Web frameworks and API routers typically bind request payloads to structs and validate them using [Go-Playground
Validator](https://github.com/go-playground/validator). However, default validation outputs suffer from two major
problems:
1. Error descriptions are in English only.
2. Field names reflect raw Go struct fields (e.g., `"ConfirmPassword"` or `"user_email"`) rather than user-friendly UI
   labels.

Rays' **`MessageSourceValidator`** solves both problems by translating validation rules into the active request locale
and replacing field names with localized labels defined in your message catalog.

---

## How It Works

`MessageSourceValidator` wraps the validation engine with localization awareness:

1. **Validation**: It invokes `Validate(&struct)` to execute standard Go-Playground validation tags (`required`,
   `min=8`, `email`, `eqfield`, etc.).
2. **Field Name Resolution**: Field names are determined by the first matching tag among `field`, `form`, `json`, and
   `xml`. If none are present, the struct field name is lowercased.
3. **Namespace Translation**: When calling `TranslateValidationErrors(errs, namespace)`, Rays looks up the localized
   field name in the active locale's bundle under `<namespace>.<field>` (e.g., `fields.registration.username`).
4. **Cross-Field Translation**: For cross-field rules like `eqfield=Password` or `eqcsfield=Parent.Field`, Rays
   translates both the source field *and* the target comparison field.

---

## Step-by-Step Implementation

### 1. Define Your Form Struct

Annotate your struct with validation rules and field identifier tags:

```go
package forms

type UserRegistrationForm struct {
    Username string `json:"username" form:"username" field:"username" validate:"required"`
    Password string `json:"password" form:"password" field:"password" validate:"required,min=8"`
    Confirm  string `json:"confirm"  form:"confirm"  field:"confirm"  validate:"required,eqfield=Password"`
}
```

### 2. Configure Field Labels in YAML

Define the friendly labels for each field under a dedicated namespace (e.g., `fields.registration`) in your message
catalogs:

```yaml
localization:
  messages:
    en:
      translation:
        fields.registration.username: "Username"
        fields.registration.password: "Password"
        fields.registration.confirm: "Confirm Password"
    fr:
      translation:
        fields.registration.username: "Nom d'utilisateur"
        fields.registration.password: "Mot de passe"
        fields.registration.confirm: "Confirmation du mot de passe"
```

### 3. Validate and Translate in Controllers

Inject `MessageSourceValidator` into your controller or service. When validation fails, extract
`validator.ValidationErrors` and call `TranslateValidationErrors()`:

```go
package controllers

import (
    "errors"
    "net/http"

    "github.com/BeamFoundry/rays/pkg/i18n"
    "github.com/go-playground/validator/v10"
)

type RegistrationController struct {
    validator i18n.MessageSourceValidator `@:"Inject"`
}

func (c *RegistrationController) Register(form *UserRegistrationForm) map[string]string {
    // 1. Validate the form pointer
    err := c.validator.Validate(form)
    if err == nil {
        return nil // Validation succeeded
    }

    // 2. Extract validation errors
    var valErrs validator.ValidationErrors
    if errors.As(err, &valErrs) {
        // 3. Translate using the "fields.registration" namespace
        return c.validator.TranslateValidationErrors(valErrs, "fields.registration")
    }

    return map[string]string{"error": err.Error()}
}
```

---

## Validation Error Outputs

Assume a user submits a registration form with `Password: "short"` (less than 8 characters) and `Confirm: "mismatch"`:

### English (`Locales.EN`)

``` password: 'Password' must be at least 8 characters in length confirm:  'Confirm Password' must be equal to
'Password' ```

### French (`Locales.FR`)

``` password: 'Mot de passe' doit faire une taille minimum de 8 caractères confirm:  'Confirmation du mot de passe' doit
être égal à 'Mot de passe' ```

### Key Highlights

- **Quotes and Labels**: Field names are wrapped in quotes (`'Password'`, `'Mot de passe'`) using your translated terms
  instead of raw struct identifiers.
- **Cross-Field Equality**: Notice that the `confirm` field's error in French correctly localized both fields:
  `'Confirmation du mot de passe' doit être égal à 'Mot de passe'`.
- **Returned Data Type**: `TranslateValidationErrors` returns `validator.ValidationErrorsTranslations`
  (`map[string]string`), keyed by the field name (`"password"`, `"confirm"`), making it straightforward to serialize
  directly to JSON or render alongside form inputs in HTML templates.
