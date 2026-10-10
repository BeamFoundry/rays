package locales

import (
	. "github.com/BeamFoundry/rays/pkg/i18n"
	"github.com/go-playground/validator/v10/translations/ar"
	"github.com/go-playground/validator/v10/translations/de"
	"github.com/go-playground/validator/v10/translations/en"
	"github.com/go-playground/validator/v10/translations/es"
	"github.com/go-playground/validator/v10/translations/fa"
	"github.com/go-playground/validator/v10/translations/fr"
	"github.com/go-playground/validator/v10/translations/id"
	"github.com/go-playground/validator/v10/translations/it"
	"github.com/go-playground/validator/v10/translations/ja"
	"github.com/go-playground/validator/v10/translations/ko"
	"github.com/go-playground/validator/v10/translations/lv"
	"github.com/go-playground/validator/v10/translations/nl"
	"github.com/go-playground/validator/v10/translations/pl"
	"github.com/go-playground/validator/v10/translations/pt"
	"github.com/go-playground/validator/v10/translations/pt_BR"
	"github.com/go-playground/validator/v10/translations/ru"
	"github.com/go-playground/validator/v10/translations/th"
	"github.com/go-playground/validator/v10/translations/tr"
	"github.com/go-playground/validator/v10/translations/uk"
	"github.com/go-playground/validator/v10/translations/vi"
	"github.com/go-playground/validator/v10/translations/zh"
	"github.com/go-playground/validator/v10/translations/zh_tw"
)

var ValidationTranslations = struct {
	AR,
	DE,
	EN,
	ES,
	FA,
	FR,
	ID,
	IT,
	JA,
	KO,
	LV,
	NL,
	PL,
	PT,
	PT_BR,
	RU,
	TH,
	TR,
	UK,
	VI,
	ZH,
	ZH_TW ValidationTranslationFunction
}{
	AR:    ar.RegisterDefaultTranslations,
	DE:    de.RegisterDefaultTranslations,
	EN:    en.RegisterDefaultTranslations,
	ES:    es.RegisterDefaultTranslations,
	FA:    fa.RegisterDefaultTranslations,
	FR:    fr.RegisterDefaultTranslations,
	ID:    id.RegisterDefaultTranslations,
	IT:    it.RegisterDefaultTranslations,
	JA:    ja.RegisterDefaultTranslations,
	KO:    ko.RegisterDefaultTranslations,
	LV:    lv.RegisterDefaultTranslations,
	NL:    nl.RegisterDefaultTranslations,
	PL:    pl.RegisterDefaultTranslations,
	PT:    pt.RegisterDefaultTranslations,
	PT_BR: pt_BR.RegisterDefaultTranslations,
	RU:    ru.RegisterDefaultTranslations,
	TH:    th.RegisterDefaultTranslations,
	TR:    tr.RegisterDefaultTranslations,
	UK:    uk.RegisterDefaultTranslations,
	VI:    vi.RegisterDefaultTranslations,
	ZH:    zh.RegisterDefaultTranslations,
	ZH_TW: zh_tw.RegisterDefaultTranslations,
}
