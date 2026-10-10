package i18n

import (
	"reflect"
	"strings"

	. "github.com/BeamFoundry/rays/pkg/core"
	. "github.com/BeamFoundry/rays/pkg/lang"

	loc "github.com/go-playground/locales"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
)

// Locale is a local alias for [github.com/go-playground/locales.Translator]
type Locale = loc.Translator

// Translator is a local alias for [github.com/go-playground/universal-translator.Translator]
type Translator = ut.Translator

// ValidationTranslationFunction is a type of function used to register a [Translator] in the [validator.Validate] object in a [LocaleMessageBundle].
type ValidationTranslationFunction func(*validator.Validate, Translator) (err error)

// MessageSource is an interface used to define and retrieve [LocaleMessageBundle].
//
// It is the central component to this localization and validation engine, as it contains
// the central [validator.Validate] and [ut.UniversalTranslator] pointers.
//
// Implementations should register a default [Locale] to create a default [LocaleMessageBundle].
// Further call to [MessageSource.NewMessageBundle] should be used the configure other [LocaleMessageBundle]
type MessageSource interface {
	// DefaultMessageBundle() lets you retrieve the default configured [LocaleMessageBundle]
	DefaultMessageBundle() LocaleMessageBundle

	// NewMessageBundle() add a new [LocalMessageBundle] for the desired [Locale]
	NewMessageBundle(locale Locale) LocaleMessageBundle

	// FindMessageBundle() let's you retrieve the first available [LocaleMessageBundle] from the locale/locales parameters,
	// or returns the "defaultLocale" [LocaleMessageBundle] if none match.
	FindMessageBundle(locale Locale, locales ...Locale) LocaleMessageBundle

	// FindTranslator() let's you retrieve the first available [Translator] from the locale/locales parameters,
	// or returns the "defaultLocale" [Translator] if none match.
	FindTranslator(locale Locale, locales ...Locale) Translator

	// UniversalTranslator() returns the [github.com/go-playground/universal-translator.UniversalTranslator].
	UniversalTranslator() *ut.UniversalTranslator

	// Validator() returns the [github.com/go-playground/validator/v10.Validator].
	Validator() *validator.Validate
}

// DefaultMessageSource is the main object used to create / retrieve individual [LocalMessageBundle].
type DefaultMessageSource struct {
	defaultLocale       Locale
	bundles             map[string]*DefaultLocaleMessageBundle
	universalTranslator *ut.UniversalTranslator
	validation          *validator.Validate
}

var _ MessageSource = &DefaultMessageSource{}

// New() lets you create a new [DefaultMessageSource] with a "defaultLocale".
func (this DefaultMessageSource) New(defaultLocale Locale) *DefaultMessageSource {
	this.universalTranslator = ut.New(defaultLocale, defaultLocale)
	this.defaultLocale = defaultLocale
	this.validation = validator.New()
	this.validation.RegisterTagNameFunc(func(field reflect.StructField) string {
		if tag := GetFieldFirstTag(&field, []string{"json", "form", "xml", "field"}); tag != "" {
			return tag
		}
		return strings.ToLower(field.Name)
	})
	bundle := DefaultLocaleMessageBundle{}.New(defaultLocale, &this)
	this.bundles = map[string]*DefaultLocaleMessageBundle{defaultLocale.Locale(): bundle}
	return &this
}

// DefaultMessageBundle() implements [MessageSource.DefaultMessageBundle].
func (this *DefaultMessageSource) DefaultMessageBundle() LocaleMessageBundle {
	return this.bundles[this.defaultLocale.Locale()]
}

// NewMessageBundle() implements [MessageSource.NewMessageBundle].
func (this *DefaultMessageSource) NewMessageBundle(locale Locale) LocaleMessageBundle {
	if bundle, ok := this.bundles[locale.Locale()]; ok {
		return bundle
	}
	err := Try(func() {
		RaiseNilErr(this.universalTranslator.AddTranslator(locale, true))
	})
	err.LogError()
	this.bundles[locale.Locale()] = DefaultLocaleMessageBundle{}.New(locale, this)
	return this.bundles[locale.Locale()]
}

// FindMessageBundle() implements [MessageSource.FindMessageBundle].
func (this *DefaultMessageSource) FindMessageBundle(locale Locale, locales ...Locale) LocaleMessageBundle {
	if bundle, ok := this.bundles[locale.Locale()]; ok {
		return bundle
	} else {
		for _, locale := range locales {
			if bundle, ok := this.bundles[locale.Locale()]; ok {
				return bundle
			}
		}
	}
	return this.bundles[this.defaultLocale.Locale()]
}

// FindTranslator() implements [MessageSource.FindTranslator].
func (this *DefaultMessageSource) FindTranslator(locale Locale, locales ...Locale) Translator {
	if translator, ok := this.universalTranslator.GetTranslator(locale.Locale()); ok {
		return translator
	}
	for _, locale := range locales {
		if translator, ok := this.universalTranslator.GetTranslator(locale.Locale()); ok {
			return translator
		}
	}
	translator, _ := this.universalTranslator.GetTranslator(this.defaultLocale.Locale())
	return translator
}

// Validator() implements [MessageSource.Validator].
func (this *DefaultMessageSource) Validator() *validator.Validate {
	return this.validation
}

// UniversalTranslator() implements [MessageSource.UniversalTranslator].
func (this *DefaultMessageSource) UniversalTranslator() *ut.UniversalTranslator {
	return this.universalTranslator
}
