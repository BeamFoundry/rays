package i18n_test

import (
	"errors"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/BeamFoundry/rays/pkg/core"
	. "github.com/BeamFoundry/rays/pkg/i18n"
	. "github.com/BeamFoundry/rays/pkg/i18n/locales"
	. "github.com/BeamFoundry/rays/pkg/lang"
	. "github.com/BeamFoundry/rays/pkg/testing"

	"github.com/go-playground/validator/v10"
)

func TestWeb(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Pkg/I18n Suite")
}

var defaultLocale = Locales.EN

type LocalizationProperties struct {
	ConfigurationProperties `Name:"localization"`
	messages                map[string]LocalizedMessages
}

type LocalizationConfig struct {
	Configuration
	localizations *LocalizationProperties `@:"Inject"`
}

func (this *LocalizationConfig) MessageSource() (MessageSource, Error) {
	messageSource := DefaultMessageSource{}.New(defaultLocale)

	bundle := messageSource.DefaultMessageBundle()

	if err := bundle.LoadValidationTranslation(ValidationTranslations.EN); err != nil {
		return nil, NewErrorFrom(err)
	}
	bundle.LoadLocalizedMessages(this.localizations.messages["en"])

	bundle = messageSource.NewMessageBundle(Locales.FR)

	if err := bundle.LoadValidationTranslation(ValidationTranslations.FR); err != nil {
		return nil, NewErrorFrom(err)
	}
	bundle.LoadLocalizedMessages(this.localizations.messages["fr"])
	return messageSource, nil
}

func (this *LocalizationConfig) LocaleContextHolder(
	_ struct {
		ConditionalOnMissingBeam
	},
) LocaleContextHolder {
	return DefaultLocaleContextHolder{}.New(defaultLocale)
}

func (this *LocalizationConfig) MessageSourceValidator(
	that struct {
		localeContextHolder LocaleContextHolder `@:"Inject"`
		messageSource       MessageSource       `@:"Inject"`
	},
) MessageSourceValidator {
	return DefaultMessageSourceValidator{}.New(that.localeContextHolder, that.messageSource)
}

//nolint:gosec // False positive
var translations = map[string]any{
	"en": map[string]any{
		"translation": map[string]any{
			"welcome":                  "Welcome",
			"welcome_name":             "Welcome, {0} {1}",
			"fields.login.username":    "Username",
			"fields.login.password":    "Password",
			"fields.login.confirm":     "Confirm",
			"fields.login.info.email":  "Email",
			"fields.login.info.emails": "Emails",
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
			"welcome":                  "Bienvenue",
			"welcome_name":             "Bienvenue, {0} {1}",
			"fields.login.username":    "Nom d'utilisateur",
			"fields.login.password":    "Mot de passe",
			"fields.login.confirm":     "Confirmation",
			"fields.login.info.email":  "Email",
			"fields.login.info.emails": "Emails",
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

var _ = Describe("LocaleContextHolder", func() {
	BeforeEach(func() {
		AddBeamFor(&LocalizationConfig{})
		AddBeamFor(&LocalizationProperties{})
		Expect(GetContainer().LoadConfigAt(translations, "localization.messages")).NotTo(HaveOccurred())
	})
	Context("Default", func() {
		It("can use a LocalContextHolder", func() {
			Expect(GetContainer().Initialize()).NotTo(HaveOccurred())
			localeContext := GetBeamDefaultInstance[LocaleContextHolder]()
			locale := localeContext.Get()
			Expect(locale).To(Equal(Locales.EN))
			localeContext.Set(Locales.FR)
			localeContext.Clear()
			messageSource := GetBeamDefaultInstance[MessageSource]()
			bundle := messageSource.FindMessageBundle(Locales.EN_US)

			Expect(ValErr(bundle.Translate("welcome"))).To(Equal("Welcome"))
			Expect(ValErr(bundle.Translate("welcome_name", "Peter", "Parker"))).To(Equal("Welcome, Peter Parker"))

			Expect(messageSource.NewMessageBundle(Locales.EN)).NotTo(BeNil())
			Expect(messageSource.FindTranslator(Locales.EN_US, Locales.EN)).NotTo(BeNil())
			Expect(messageSource.FindTranslator(Locales.EN_US, Locales.EN_GB)).NotTo(BeNil())
			Expect(messageSource.UniversalTranslator()).NotTo(BeNil())
		})
	})
	Context("ThreadLocal", func() {
		It("can use a LocalContextHolder", func() {
			AddBeamFor(ThreadLocaleContextHolder{}.New(defaultLocale))
			Expect(GetContainer().Initialize()).NotTo(HaveOccurred())
			localeContext := GetBeamDefaultInstance[LocaleContextHolder]()
			locale := localeContext.Get()
			Expect(locale).To(Equal(Locales.EN))
			localeContext.Set(Locales.FR)
			localeContext.Clear()
			messageSource := GetBeamDefaultInstance[MessageSource]()
			bundle := messageSource.FindMessageBundle(Locales.EN_US)

			Expect(ValErr(bundle.Translate("welcome"))).To(Equal("Welcome"))
			Expect(ValErr(bundle.Translate("welcome_name", "Peter", "Parker"))).To(Equal("Welcome, Peter Parker"))
		})
	})
})

var _ = Describe("Locales", func() {
	BeforeEach(func() {
		AddBeamFor(&LocalizationConfig{})
		AddBeamFor(&LocalizationProperties{})
		Expect(GetContainer().LoadConfigAt(translations, "localization.messages")).NotTo(HaveOccurred())
		Expect(GetContainer().Initialize()).NotTo(HaveOccurred())
	})
	It("can use Translate", func() {
		messageSource := GetBeamDefaultInstance[MessageSource]()
		bundle := messageSource.FindMessageBundle(Locales.FR_FR, Locales.FR)

		Expect(ValErr(bundle.Translate("welcome"))).To(Equal("Bienvenue"))
		Expect(ValErr(bundle.Translate("welcome_name", "Peter", "Parker"))).To(Equal("Bienvenue, Peter Parker"))
	})
	It("can use Cardinal", func() {
		messageSource := GetBeamDefaultInstance[MessageSource]()
		bundle := messageSource.FindMessageBundle(Locales.FR)

		Expect(ValErr(bundle.Cardinal("days.left", 1))).To(Equal("1 jour restant"))
		Expect(ValErr(bundle.CardinalWithPrecision("days.left", 1, 1))).To(Equal("1,0 jour restant"))
		Expect(ValErr(bundle.Cardinal("days.left", 2))).To(Equal("2 jours restant"))
		Expect(ValErr(bundle.CardinalWithPrecision("days.left", 2, 1))).To(Equal("2,0 jours restant"))
	})
	It("can use Ordinal", func() {
		messageSource := GetBeamDefaultInstance[MessageSource]()
		bundle := messageSource.FindMessageBundle(Locales.EN)

		Expect(ValErr(bundle.Ordinal("price", 1))).To(Equal("1st price"))
		Expect(ValErr(bundle.Ordinal("price", 2))).To(Equal("2nd price"))
		Expect(ValErr(bundle.OrdinalWithPrecision("price", 2, 1))).To(Equal("2.0nd price"))
	})
	It("can use Range", func() {
		messageSource := GetBeamDefaultInstance[MessageSource]()
		bundle := messageSource.FindMessageBundle(Locales.FR)

		Expect(ValErr(bundle.Range("days.range", 0, 1))).To(Equal("0-1 jour"))
		Expect(ValErr(bundle.RangeWithPrecision("days.range", 0, 1, 1))).To(Equal("0,0-1,0 jour"))
		Expect(ValErr(bundle.Range("days.range", 1, 2))).To(Equal("1-2 jours"))
		Expect(ValErr(bundle.RangeWithPrecision("days.range", 1, 2, 1))).To(Equal("1,0-2,0 jours"))
	})
	It("can validate and translate an anonymous struct", func() {
		localeContext := GetBeamDefaultInstance[LocaleContextHolder]()
		localeContext.Set(Locales.EN)
		validation := GetBeamDefaultInstance[MessageSourceValidator]()

		email := struct {
			Email string `validate:"required,email"`
		}{
			Email: "foo@bar.com",
		}
		Expect(validation.Validate(&email)).NotTo(HaveOccurred())

		form := struct {
			Username string  `validate:"eqcsfield=Info.Email"`
			Password *string `validate:"required,min=8"`
			Confirm  string  `validate:"required,eqfield=Password"`
			Info     struct {
				Email  string   `validate:"required,email"`
				Emails []string `validate:"min=1,dive,email" field:"emails"`
			} `field:"info"`
		}{
			Username: "foo@bar.com",
			Password: new("bar"),
			Confirm:  "foo",
			Info: struct {
				Email  string   `validate:"required,email"`
				Emails []string `validate:"min=1,dive,email" field:"emails"`
			}{
				Email:  "foo",
				Emails: []string{},
			},
		}

		err := validation.Validate(&form)
		Expect(err).To(HaveOccurred())
		errs, ok := errors.AsType[validator.ValidationErrors](err)
		Expect(ok).To(BeTrue())
		msg := validation.TranslateValidationErrors(errs, "fields.login")
		Expect(msg["username"]).To(Equal("'Username' must be equal to 'Email'"))
		Expect(msg["confirm"]).To(Equal("'Confirm' must be equal to 'Password'"))
		Expect(msg["password"]).To(Equal("'Password' must be at least 8 characters in length"))
		Expect(msg["email"]).To(Equal("'Email' must be a valid email address"))
		Expect(msg["emails"]).To(Equal("'Emails' must contain at least 1 item"))
		Expect(msg).To(HaveLen(5))
	})
	It("can validate and translate a registered struct", func() {
		localeContext := GetBeamDefaultInstance[LocaleContextHolder]()
		localeContext.Set(Locales.FR)
		validation := GetBeamDefaultInstance[MessageSourceValidator]()

		type Info struct {
			Email  string   `validate:"required,email"`
			Emails []string `validate:"min=1,dive,email"`
		}

		email := Info{
			Email: "foo@bar.com",
			Emails: []string{
				"foo@bar.com",
			},
		}
		Expect(validation.Validate(&email)).NotTo(HaveOccurred())

		type Form struct {
			Username string  `validate:"eqcsfield=Info.Email"`
			Password *string `validate:"required,min=8"`
			Confirm  string  `validate:"required,eqfield=Password"`
			Info     Info    `field:"info"`
		}
		form := Form{
			Username: "foo@bar.com",
			Password: new("bar"),
			Confirm:  "foo",
			Info: Info{
				Email:  "foo",
				Emails: []string{},
			},
		}

		Expect(validation.Validate(form)).To(HaveOccurred())
		err := validation.Validate(&form)
		Expect(err).To(HaveOccurred())
		errs, ok := errors.AsType[validator.ValidationErrors](err)
		Expect(ok).To(BeTrue())
		msg := validation.TranslateValidationErrors(errs, "fields.login")
		Expect(msg["username"]).To(Equal("'Nom d'utilisateur' doit être égal à 'Email'"))
		Expect(msg["confirm"]).To(Equal("'Confirmation' doit être égal à 'Mot de passe'"))
		Expect(msg["password"]).To(Equal("'Mot de passe' doit faire une taille minimum de 8 caractères"))
		Expect(msg["email"]).To(Equal("'Email' doit être une adresse email valide"))
		Expect(msg["emails"]).To(Equal("'Emails' doit contenir au moins 1 element"))
		Expect(msg).To(HaveLen(5))
	})
})
