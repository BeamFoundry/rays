//nolint:errcheck,wrapcheck //Removing error checking and wrapcheck
package i18n

import (
	"github.com/go-playground/locales"
)

// LocaleMessageBundle is an interface to an object that let you define [Locale] messages and apply them, depending on the context.
//
// LocaleMessageBundle should be constructed through a single [DefaultMessageSource] object.
//
// Each locale messages can be loaded using its methods or through the [LocalizedMessages.Load] method.
type LocaleMessageBundle interface {
	LoadLocalizedMessages(messages LocalizedMessages)

	// AddValidationTranslation() let's you register a validation function.
	//
	// To be used with [github.com/go-playground/validator/v10/translations] modules.
	//
	// Example :
	//
	//  import "github.com/go-playground/validator/v10/translations/en
	//
	//  ...
	//
	//  bundle.AddValidationTranslation(en.RegisterDefaultTranslations)
	LoadValidationTranslation(translation ValidationTranslationFunction) error

	// AddTranslation() adds a normal translation for this LocaleMessageBundle language/locale
	// Multiple string parameters may be references (like "{0}", "{1}", ...)
	//
	// Example :
	//
	//  bundle.AddTranslation("welcome": "Welcome back, {0}")
	AddTranslation(key, text string)

	// AddCardinal() adds a cardinal translation for this LocaleMessageBundle language/locale.
	// {0} is the only replacement type accepted and only one variable is accepted as
	// multiple cannot be used for a plural rule determination.
	//
	// Example :
	//
	//  bundle.AddCardinal("days.left", "{0} day left", PluralRules.ONE)
	//  bundle.AddCardinal("days.left", "{0} days left", PluralRules.OTHER)
	AddCardinal(key, text string, rule PluralRule)

	// AddOrdinal() adds an ordinal translation for this LocaleMessageBundle language/locale.
	// {0} is the only replacement type accepted and only one variable is accepted as
	// multiple cannot be used for a plural rule determination.
	//
	// Example :
	//
	// bundle.AddOrdinal("spring_days", "{0}st day of spring", PluralRules.ONE)
	// bundle.AddOrdinal("spring_days", "{0}nd day of spring", PluralRules.TWO)
	// bundle.AddOrdinal("spring_days", "{0}rd day of spring", PluralRules.FEW)
	// bundle.AddOrdinal("spring_days", "{0}th day of spring", PluralRules.OTHER)
	AddOrdinal(key, text string, rule PluralRule)

	// AddRange() adds a range translation for this LocaleMessageBundle language/locale.
	// {0} and {1} are the only replacement types accepted and only these are accepted.
	//
	// Example :
	//
	// bundle.AddRange('days_left', "{0}-{1} day left", PluralRules.ONE)
	// bundle.AddRange('days_left', "{0}-{1} days left", PluralRules.OTHER)
	AddRange(key, text string, rule PluralRule)

	// Translate() creates the translation for the locale given the 'key' and params passed in
	Translate(key string, params ...string) (string, error)

	// Cardinal() creates the cardinal translation for the locale given the 'key', 'num' and 'digit' arguments
	// and param passed in
	Cardinal(key string, num float64) (string, error)

	// CardinalWithPrecision() creates the cardinal translation for the locale given the 'key', 'num' and 'digit' arguments
	// and param passed in.
	CardinalWithPrecision(key string, num float64, precision uint64) (string, error)

	// Ordinal() creates the ordinal translation for the locale given the 'key', 'num' and 'digit' arguments
	// and param passed in.
	Ordinal(key string, num float64) (string, error)

	// OrdinalWithPrecision() creates the ordinal translation for the locale given the 'key', 'num' and 'digit' arguments
	// and param passed in.
	OrdinalWithPrecision(key string, num float64, precision uint64) (string, error)

	// Range() creates the range translation for the locale given the 'key', 'num1', 'digit1', 'num2' and
	// 'digit2' arguments and 'param1' and 'param2' passed in.
	Range(key string, num1, num2 float64) (string, error)

	// RangeWithPrecision() creates the range translation for the locale given the 'key', 'num1', 'digit1', 'num2' and
	// 'digit2' arguments and 'param1' and 'param2' passed in.
	RangeWithPrecision(key string, num1, num2 float64, precision uint64) (string, error)
}

// DefaultLocaleMessageBundle is the default implementation of [LocaleMessageBundle]
type DefaultLocaleMessageBundle struct {
	locale        Locale
	messageSource MessageSource
}

var _ LocaleMessageBundle = &DefaultLocaleMessageBundle{}

// New() returns a new DefaultLocaleMessageBundle
func (this DefaultLocaleMessageBundle) New(locale Locale, messageSource MessageSource) *DefaultLocaleMessageBundle {
	this.locale = locale
	this.messageSource = messageSource
	return &this
}

// LoadLocalizedMessages() implements [LocaleMessageBundle.LoadLocalizedMessages]
func (this *DefaultLocaleMessageBundle) LoadLocalizedMessages(messages LocalizedMessages) {
	for k, v := range messages.translation {
		this.AddTranslation(k, v)
	}

	for k, v := range messages.cardinal {
		for p, m := range v {
			this.AddCardinal(k, m, p)
		}
	}

	for k, v := range messages.ordinal {
		for p, m := range v {
			this.AddOrdinal(k, m, p)
		}
	}

	for k, v := range messages.ranges {
		for p, m := range v {
			this.AddRange(k, m, p)
		}
	}
}

// LoadValidationTranslation() implements [LocaleMessageBundle.LoadValidationTranslation]
func (this *DefaultLocaleMessageBundle) LoadValidationTranslation(translation ValidationTranslationFunction) error {
	return translation(this.messageSource.Validator(), this.translator())
}

// AddTranslation() implements [LocaleMessageBundle.AddTranslation]
func (this *DefaultLocaleMessageBundle) AddTranslation(key, text string) {
	this.translator().Add(key, text, true)
}

// AddCardinal() implements [LocaleMessageBundle.AddCardinal]
func (this *DefaultLocaleMessageBundle) AddCardinal(key, text string, rule PluralRule) {
	this.translator().AddCardinal(key, text, locales.PluralRule(rule.Ordinal()+1), true)
}

// AddOrdinal() implements [LocaleMessageBundle.AddOrdinal]
func (this *DefaultLocaleMessageBundle) AddOrdinal(key, text string, rule PluralRule) {
	this.translator().AddOrdinal(key, text, locales.PluralRule(rule.Ordinal()+1), true)
}

// AddRange() implements [LocaleMessageBundle.AddRange]
func (this *DefaultLocaleMessageBundle) AddRange(key, text string, rule PluralRule) {
	this.translator().AddRange(key, text, locales.PluralRule(rule.Ordinal()+1), true)
}

// Translate() implements [LocaleMessageBundle.Translate]
func (this *DefaultLocaleMessageBundle) Translate(key string, params ...string) (string, error) {
	return this.translator().T(key, params...)
}

// Cardinal() implements [LocaleMessageBundle.Cardinal]
func (this *DefaultLocaleMessageBundle) Cardinal(key string, num float64) (string, error) {
	return this.translator().C(key, num, 0, this.locale.FmtNumber(num, 0))
}

// CardinalWithPrecision() implements [LocaleMessageBundle.CardinalWithPrecision]
func (this *DefaultLocaleMessageBundle) CardinalWithPrecision(key string, num float64, precision uint64) (string, error) {
	return this.translator().C(key, num, precision, this.locale.FmtNumber(num, precision))
}

// Ordinal() implements [LocaleMessageBundle.Ordinal]
func (this *DefaultLocaleMessageBundle) Ordinal(key string, num float64) (string, error) {
	return this.translator().O(key, num, 0, this.locale.FmtNumber(num, 0))
}

// OrdinalWithPrecision() implements [LocaleMessageBundle.OrdinalWithPrecision]
func (this *DefaultLocaleMessageBundle) OrdinalWithPrecision(key string, num float64, precision uint64) (string, error) {
	return this.translator().O(key, num, precision, this.locale.FmtNumber(num, precision))
}

// Range() implements [LocaleMessageBundle.Range]
func (this *DefaultLocaleMessageBundle) Range(key string, num1, num2 float64) (string, error) {
	return this.translator().R(key, num1, 0, num2, 0, this.locale.FmtNumber(num1, 0), this.locale.FmtNumber(num2, 0))
}

// RangeWithPrecision() implements [LocaleMessageBundle.RangeWithPrecision]
func (this *DefaultLocaleMessageBundle) RangeWithPrecision(key string, num1, num2 float64, precision uint64) (string, error) {
	return this.translator().R(key, num1, precision, num2, precision, this.locale.FmtNumber(num1, precision), this.locale.FmtNumber(num2, precision))
}

func (this *DefaultLocaleMessageBundle) translator() Translator {
	return this.messageSource.FindTranslator(this.locale)
}
