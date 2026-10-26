package web

import (
	"html/template"
	"reflect"

	. "github.com/BeamFoundry/rays/pkg/core"
	. "github.com/BeamFoundry/rays/pkg/i18n"
	. "github.com/BeamFoundry/rays/pkg/lang"
	. "github.com/BeamFoundry/rays/pkg/util"
)

// FuncMapper is an interface defining an object that returns a [template.FuncMap]
type FuncMapper interface {
	FuncMap() template.FuncMap
}

// TemplateFunctionMap is a container object for [template.FuncMap]
type TemplateFunctionMap struct {
	fnmap Map[string, any]
}

// New() creates a new TemplateFunctionMap
func (this TemplateFunctionMap) New() *TemplateFunctionMap {
	this.fnmap = ConcurrentHashMap[string, any]{}.New()
	return &this
}

// Add() adds a function "fn" with name "name" to the this TemplateFunctionMap
func (this *TemplateFunctionMap) Add(name string, fn any) Error {
	t := reflect.TypeOf(fn)
	if t.Kind() != reflect.Func {
		return NewError("InvalidTypeError", "f argument must be a function")
	}
	this.fnmap.Put(name, fn)
	return nil
}

// FuncMap() returns all the template function loaded as a [template.FuncMap]
func (this *TemplateFunctionMap) FuncMap() template.FuncMap {
	return this.fnmap.Map()
}

// Load() will load an already defined [template.FuncMap] into itself.
func (this *TemplateFunctionMap) Load(fnmap template.FuncMap) Error {
	for name, fn := range fnmap {
		t := reflect.TypeOf(fn)
		if t.Kind() != reflect.Func {
			return NewError("InvalidTypeError", "fn is not a function.", "name", name)
		}
	}
	this.fnmap.Merge(NewHashMapFrom(fnmap))
	return nil
}

// LocaleFuncMap implements [FuncMapper] and deliver template functions to retrieve translation and localized message.
//
// Functions :
//
//   - "translate": [LocaleMessageBundle.Translate]
//   - "cardinal": [LocaleMessageBundle.Cardinal]
//   - "ordinal": [LocaleMessageBundle.Ordinal]
//   - "ranges": [LocaleMessageBundle.Range]
type LocaleFuncMap struct {
	localeContextHolder LocaleContextHolder
	messageSource       MessageSource
}

// New() create a new [LocaleFuncMap] using a [ThreadLocaleContextHolder] and [MessageSource]
func (this LocaleFuncMap) New(contextHolder LocaleContextHolder, messageSource MessageSource) *LocaleFuncMap {
	this.localeContextHolder = contextHolder
	this.messageSource = messageSource
	return &this
}

// FuncMap() implements [FuncMapper]
func (this *LocaleFuncMap) FuncMap() template.FuncMap {
	fnmap := TemplateFunctionMap{}.New()

	err := Try(func() {
		fn := template.FuncMap{
			"translate": func(key string, params ...string) (string, error) {
				return this.getMessageBundle().Translate(key, params...)
			},
			"cardinal": func(key string, num float64) (string, error) {
				return this.getMessageBundle().Cardinal(key, num)
			},
			"ordinal": func(key string, num float64) (string, error) {
				return this.getMessageBundle().Ordinal(key, num)
			},
			"ranges": func(key string, num1, num2 float64) (string, error) {
				return this.getMessageBundle().Range(key, num1, num2)
			},
		}
		RaiseNilErr(fnmap.Load(fn))
	})
	err.LogError()
	return fnmap.FuncMap()
}

func (this *LocaleFuncMap) getMessageBundle() LocaleMessageBundle {
	locale := this.localeContextHolder.Get()
	return this.messageSource.FindMessageBundle(locale)
}
