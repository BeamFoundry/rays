package web_test

import (
	"html/template"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/BeamFoundry/rays/pkg/core"
	. "github.com/BeamFoundry/rays/pkg/i18n"
	. "github.com/BeamFoundry/rays/pkg/i18n/locales"
	. "github.com/BeamFoundry/rays/pkg/lang"
	. "github.com/BeamFoundry/rays/pkg/testing"
	. "github.com/BeamFoundry/rays/pkg/web"
)

func TestWeb(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Pkg/Web Suite")
}

type LocalizationProperties struct {
	ConfigurationProperties `Name:"localization"`
	messages                map[string]LocalizedMessages
}

type LocalizationConfig struct {
	Configuration
	localizations       *LocalizationProperties `@:"Inject"`
	localeContextHolder *DefaultLocaleContextHolder
	messageSource       *DefaultMessageSource
}

func (this *LocalizationConfig) PostConstruct() Error {
	defaultLocale := Locales.EN

	this.localeContextHolder = DefaultLocaleContextHolder{}.New(defaultLocale)
	this.messageSource = DefaultMessageSource{}.New(defaultLocale)

	bundle := this.messageSource.DefaultMessageBundle()

	if err := bundle.LoadValidationTranslation(ValidationTranslations.EN); err != nil {
		return NewErrorFrom(err)
	}
	bundle.LoadLocalizedMessages(this.localizations.messages["en"])

	bundle = this.messageSource.NewMessageBundle(Locales.FR)

	if err := bundle.LoadValidationTranslation(ValidationTranslations.FR); err != nil {
		return NewErrorFrom(err)
	}
	bundle.LoadLocalizedMessages(this.localizations.messages["fr"])

	return nil
}

func (this *LocalizationConfig) LocaleContextHolder() LocaleContextHolder {
	return this.localeContextHolder
}

func (this *LocalizationConfig) MessageSource() MessageSource {
	return this.messageSource
}

func (this *LocalizationConfig) LocaleFuncMap() *LocaleFuncMap {
	return LocaleFuncMap{}.New(this.localeContextHolder, this.messageSource)
}

var _ = Describe("Locales", func() {
	translations := map[string]any{
		"en": map[string]any{
			"translation": map[string]any{
				"welcome":      "Welcome",
				"welcome_name": "Welcome, {0} {1}",
			},
			"cardinal": map[string]any{
				"days.left": map[string]any{
					"ONE":   "{0} day remaining",
					"OTHER": "{0} days remaining",
				},
			},
			"ordinal": map[string]any{
				"price": map[string]any{
					"ONE":   "{0}st price",
					"TWO":   "{0}nd price",
					"FEW":   "{0}rd price",
					"OTHER": "{0}th price",
				},
			},
			"ranges": map[string]any{
				"days.range": map[string]any{
					"OTHER": "{0}-{1} days",
				},
			},
		},
		"fr": map[string]any{
			"translation": map[string]any{
				"welcome":      "Bienvenue",
				"welcome_name": "Bienvenue, {0} {1}",
			},
			"cardinal": map[string]any{
				"days.left": map[string]any{
					"ONE":   "{0} jour restant",
					"OTHER": "{0} jours restant",
				},
			},
			"ordinal": map[string]any{
				"price": map[string]any{
					"ONE":   "{0}er prix",
					"OTHER": "{0}eme prix",
				},
			},
			"ranges": map[string]any{
				"days.range": map[string]any{
					"ONE":   "{0}-{1} jour",
					"OTHER": "{0}-{1} jours",
				},
			},
		},
	}
	BeforeEach(func() {
		AddBeamFor(&LocalizationConfig{})
		AddBeamFor(&LocalizationProperties{})
		Expect(GetContainer().LoadConfigAt(translations, "localization.messages")).NotTo(HaveOccurred())
		Expect(GetContainer().Initialize()).NotTo(HaveOccurred())
	})
	It("can use TemplateFuncMap", func() {
		fnmap := TemplateFunctionMap{}.New()
		Expect(fnmap.Add("add", func(a, b int) int { return a + b })).NotTo(HaveOccurred())
		Expect(fnmap.Add("string", "string")).To(HaveOccurred())
		fnOK := template.FuncMap{
			"add": func(a, b int) int {
				return a + b
			},
		}
		Expect(fnmap.Load(fnOK)).NotTo(HaveOccurred())
		fnBAD := template.FuncMap{
			"func": "string",
		}
		Expect(fnmap.Load(fnBAD)).To(HaveOccurred())
	})
	It("can use LocaleFuncMap", func() {
		localeContext := GetBeamDefaultInstance[LocaleContextHolder]()
		localeContext.Set(Locales.FR)
		localeFuncMap := GetBeamDefaultInstance[*LocaleFuncMap]()

		fnMap := localeFuncMap.FuncMap()
		Expect(fnMap).NotTo(BeEmpty())

		if translate, ok := (fnMap["translate"]).(func(key string, params ...string) (string, error)); ok {
			Expect(ValErr(translate("welcome"))).To(Equal("Bienvenue"))
		}
		if cardinal, ok := (fnMap["cardinal"]).(func(key string, num float64) (string, error)); ok {
			Expect(ValErr(cardinal("days.left", 2))).To(Equal("2 jours restant"))
		}
		if ordinal, ok := (fnMap["ordinal"]).(func(key string, num float64) (string, error)); ok {
			Expect(ValErr(ordinal("price", 2))).To(Equal("2eme prix"))
		}
		if ranges, ok := (fnMap["ranges"]).(func(key string, num1, num2 float64) (string, error)); ok {
			Expect(ValErr(ranges("days.range", 1, 2))).To(Equal("1-2 jours"))
		}
	})
})
