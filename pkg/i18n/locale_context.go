package i18n

import (
	"sync"

	"github.com/timandy/routine"
)

type LocaleContextHolder interface {
	// Clear() reset the current value for Locale, reverting to its default value.
	Clear()

	// Set() set the current Locale to use after.
	Set(locale Locale)

	// Get() returns the current Locale or the default one if none is defined.
	Get() Locale
}

type DefaultLocaleContextHolder struct {
	defaultLocale Locale
	locale        Locale
	lock          *sync.RWMutex
}

var _ LocaleContextHolder = &DefaultLocaleContextHolder{}

func (this DefaultLocaleContextHolder) New(defaultLococale Locale) *DefaultLocaleContextHolder {
	this.defaultLocale = defaultLococale
	this.locale = defaultLococale
	this.lock = &sync.RWMutex{}
	return &this
}

// Clear() implements [LocaleContextHolder.Clear].
func (this *DefaultLocaleContextHolder) Clear() {
	this.lock.Lock()
	defer this.lock.Unlock()
	this.locale = this.defaultLocale
}

// Get() implements [LocaleContextHolder.Get].
func (this *DefaultLocaleContextHolder) Get() Locale {
	this.lock.RLock()
	defer this.lock.RUnlock()
	return this.locale
}

// Set() implements [LocaleContextHolder.Set].
func (this *DefaultLocaleContextHolder) Set(locale Locale) {
	this.lock.Lock()
	defer this.lock.Unlock()
	this.locale = locale
}

// ThreadLocaleContextHolder is an implementation of [LocaleContextHolder] that can be used in
// a http middleware and in an http request handler function to set the current [Locale] to use
// (based on cookie, query parameter) ...) and retrieve it.
//
// It is to be registred in a [Configuration] object.
//
// It's based on [routine.ThreadLocal].
type ThreadLocaleContextHolder struct {
	threadLocal routine.ThreadLocal[Locale]
}

var _ LocaleContextHolder = &ThreadLocaleContextHolder{}

// New() creates a new LocaleContextHolder
func (this ThreadLocaleContextHolder) New(defautLocale Locale) *ThreadLocaleContextHolder {
	this.threadLocal = routine.NewThreadLocalWithInitial(func() Locale { return defautLocale })
	return &this
}

// Clear() implements [LocaleContextHolder.Clear].
//
// It should be called in a http middleware first before calling Set() or Get(),
// as it might returns a previously used thread value.
func (this *ThreadLocaleContextHolder) Clear() {
	this.threadLocal.Set(nil)
}

// Set() implements [LocaleContextHolder.Set].
func (this *ThreadLocaleContextHolder) Set(locale Locale) {
	this.threadLocal.Set(locale)
}

// Get() implements [LocaleContextHolder.Get].
func (this *ThreadLocaleContextHolder) Get() Locale {
	return this.threadLocal.Get()
}
