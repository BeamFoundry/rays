package i18n

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

// MessageSourceValidator is an interface representing an object that can be used to validate a struct and
// translate validation errors using the current [Locale] as defined in the [ThreadLocaleContextHolder].
//
// [MessageSourceValidator.Validate] if used with a predefined struct (ie. not an anonymous one),
// it will recreate one using reflection to insure the error messages are standardized.
//
// NOTE: It will be changed in a future release.
//
// [MessageSourceValidator.TranslateValidationErrors] may optionally use a "namespace" prefix to search
// field name translation for the current Locale.
// It will be used to combine it, with the fields name (which can be overridden using the struct tag "field"),
// to return the validation error message with the localized field's name.
//
// This is specially designed to return to web users form validation errors using the apparent field's displayed label,
// not the HTML form input name.
//
// For examole for an english Locale, with the following locales defined is configuration properties :
//
//	localization:
//	  messages:
//	    en:
//	      translation:
//	        registration.firstname: First Name
//	        registration.lastname: Last Name
//
// For the following struct where you have binded the current HTTP Post request to the following struct:
//
//	struct {
//	  Firstname string `form:"firstname" field:"firstname" validate:"required"`
//	  Lastname string `form:"lastname" field:"lastname" validate:"required"`
//	}
//
// If any of the fields does not validate, a call to this function using a namespace of "registration"
// will return the localized english message validation using either 'First Name' or 'Last Name', instead of the
// form fields name ("Firstname"/"firstname" or "Lastname"/"lastname").
type MessageSourceValidator interface {
	// Validate() implements the validation function.
	Validate(data any) error

	// TranslateValidationErrors() will help ou translate any validation errors,
	// using the current locale obtain from the LocaleContextHolder.
	TranslateValidationErrors(validationErrors validator.ValidationErrors, namespace ...string) validator.ValidationErrorsTranslations
}

// DefaultMessageSourceValidator is the default implementation of [MessageSourceValidator],
// an object that can be used to validate a struct using the provided LocaleContextHolder and MessageSource.
//
// It also implements [FuncMapper] to offer access to localization function in HTML templates.
type DefaultMessageSourceValidator struct {
	localeContextHolder LocaleContextHolder
	messageSource       MessageSource
}

var _ MessageSourceValidator = &DefaultMessageSourceValidator{}

// New() construct a new MessageSourceValidator
func (this DefaultMessageSourceValidator) New(contextHolder LocaleContextHolder, messageSource MessageSource) *DefaultMessageSourceValidator {
	this.localeContextHolder = contextHolder
	this.messageSource = messageSource

	return &this
}

// Validate() implements [MessageSourceValidator.Validate].
func (this *DefaultMessageSourceValidator) Validate(data any) error {
	t := reflect.TypeOf(data)
	if t.Kind() != reflect.Pointer || t.Elem().Kind() != reflect.Struct {
		return errors.New("data must be a struct pointer")
	}
	t = t.Elem()
	if t.Name() == "" && t.PkgPath() == "" {
		if err := this.messageSource.Validator().Struct(data); err != nil {
			return fmt.Errorf("%w", err)
		}
		return nil
	}
	origVal := reflect.ValueOf(data).Elem()
	var fields []reflect.StructField
	for field := range t.Fields() {
		fields = append(fields, reflect.StructField{
			Name: field.Name,
			Type: field.Type,
			Tag:  field.Tag,
		})
	}
	anonType := reflect.StructOf(fields)
	anonPtr := reflect.New(anonType)
	anonVal := anonPtr.Elem()
	for i := 0; i < origVal.NumField(); i++ {
		anonVal.Field(i).Set(origVal.Field(i))
	}
	if err := this.messageSource.Validator().Struct(anonPtr.Interface()); err != nil {
		return fmt.Errorf("%w", err)
	}
	return nil
}

// TranslateValidationErrors() implements [MessageSourceValidator.TranslateValidationErrors].
func (this *DefaultMessageSourceValidator) TranslateValidationErrors(validationErrors validator.ValidationErrors, namespace ...string) validator.ValidationErrorsTranslations {
	bundle := this.getMessageBundle()

	translateFieldError := func(fieldError validator.FieldError, namespace ...string) string {
		prefix := "fields"
		if len(namespace) >= 1 {
			prefix = namespace[0]
		}

		field := fieldError.Field()
		tag := fieldError.ActualTag()
		param := fieldError.Param()

		if translated, err := bundle.Translate(prefix + "." + strings.ToLower(fieldError.Namespace())); err == nil {
			field = "'" + translated + "'"
		}

		feField := fieldError.Field()
		fefield := field

		if strings.HasSuffix(tag, "field") {
			if strings.HasSuffix(tag, "csfield") {
				if translated, err := bundle.Translate(prefix + "." + strings.ToLower(param)); err == nil {
					param = "'" + translated + "'"
				}
			} else {
				ns := fieldError.Namespace()
				ns = strings.TrimSuffix(ns, fieldError.Field())
				if translated, err := bundle.Translate(prefix + "." + strings.ToLower(ns+fieldError.Param())); err == nil {
					param = "'" + translated + "'"
				}
			}
			feField = fieldError.Param()
			fefield = param
			if msg, err := bundle.Translate(tag, field, param); err == nil {
				return msg
			}
		}

		message := fieldError.Translate(this.getTranslator())
		message = strings.Replace(message, feField, fefield, 1)
		return message
	}

	m := map[string]string{}
	for _, fieldError := range validationErrors {
		msg := translateFieldError(fieldError, namespace...)
		m[fieldError.Field()] = msg
	}
	return m
}

func (this *DefaultMessageSourceValidator) getMessageBundle() LocaleMessageBundle {
	locale := this.localeContextHolder.Get()
	return this.messageSource.FindMessageBundle(locale)
}

func (this *DefaultMessageSourceValidator) getTranslator() Translator {
	locale := this.localeContextHolder.Get()
	return this.messageSource.FindTranslator(locale)
}
