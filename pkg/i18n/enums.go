package i18n

import (
	. "github.com/BeamFoundry/rays/pkg/lang"
)

// PluralRuleEnum is an Enum struct based on [locales.PluralRule] possible values.
//
// See [github.com/BeamFoundry/rays/pkg/lang.Enum] on how to use this type.
type PluralRuleEnum struct {
	ZERO,
	ONE,
	TWO,
	FEW,
	MANY,
	OTHER PluralRule
}

// PluralRule is a Enum alias for [PluralRuleEnum].
//
// See [github.com/BeamFoundry/rays/pkg/lang.Enum] on how to use this type.
type PluralRule = Enum[PluralRuleEnum]

// PluralRules is an initialized PluralRuleEnum variable.
var PluralRules = PluralRule{}.New()

// LocalizedMessages is a type that can be used in a [github.com/BeamFoundry/rays/pkg/core.ConfigurationProperties] to load locale messages
//
// For example :
//
//	type LocalizationProperties struct {
//		ConfigurationProperties `Name:"localization"`
//		messages                map[string]LocalizedMessages
//	}
//
// This can load the following YAML example :
//
//	localization:
//	  messages:
//	    en:
//	      translation: # map[string]string
//	        welcome: Welcome
//	      cardinal: # map[string]map[PluralRule]string
//	        days.left: # map[PluralRule]string
//	          ONE: {0} day remaining
//	          OTHER: {0} days remaining
//	      ordinal: # map[string]map[PluralRule]string
//	        price: # map[PluralRule]string
//	          ONE: {0}st price
//	          TWO: {0}nd price
//	          FEW: {0}rd price
//	          OTHER: {0}th price
//	      ranges: # map[string]map[PluralRule]string
//	        days.range: # map[PluralRule]string
//	          OTHER: {0}-{1} days
//
// It is to simplify loading messages into a [LocaleMessageBundle].
type LocalizedMessages struct {
	translation map[string]string
	cardinal    map[string]map[PluralRule]string
	ordinal     map[string]map[PluralRule]string
	ranges      map[string]map[PluralRule]string
}
