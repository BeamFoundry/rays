package lang

import (
	"fmt"
	"reflect"
)

// Aliasing Exception to Error. Can be used to indicate this Error should be handled with Try.
type Exception = Error

// ErrorHandler is an interface used by the Try() function as each clauses.
type ErrorHandler interface {
	// Supports() returns whether can support handling err through Catch().
	//
	// Implementations should check err through [IsNilErr] and return false if that's the case.
	Supports(err Error) bool
	// Catch() will handle err. It may return nil, a new Error or the same, after for example using [Error.LogError].
	Catch(err Error) Error
}

type typedCatch[E Error] struct {
	catch func(E) Error
}

func (this typedCatch[E]) New(action func(E) Error) ErrorHandler {
	this.catch = action
	return &this
}

func (this *typedCatch[E]) Supports(err Error) bool {
	if IsNilError(err) {
		return false
	}

	t := reflect.TypeOf(err)
	tE := reflect.TypeFor[E]()
	if tE.Kind() == reflect.Pointer {
		return t == tE
	} else if tE.Kind() == reflect.Interface {
		return t.Implements(tE)
	}
	return false
}

func (this *typedCatch[E]) Catch(err Error) Error {
	ret := reflect.ValueOf(this.catch).Call([]reflect.Value{reflect.ValueOf(err)})
	if e, ok := ret[0].Interface().(Error); ok {
		return e
	}
	return nil
}

type namedCatch struct {
	catch func(Error) Error
	name  string
}

func (this namedCatch) New(name string, action func(Error) Error) ErrorHandler {
	this.name = name
	this.catch = action
	return &this
}

func (this *namedCatch) Supports(err Error) bool {
	if IsNilError(err) {
		return false
	}

	return err.Name() == this.name
}

func (this *namedCatch) Catch(err Error) Error {
	return this.catch(err)
}

type finallyCatch struct {
	finally func()
}

func (this finallyCatch) New(action func()) ErrorHandler { return &finallyCatch{action} }

func (*finallyCatch) Supports(err Error) bool {
	return IsNilError(err)
}

func (this *finallyCatch) Catch(Error) Error {
	this.finally()
	return nil
}

// Catch() returns an ErrorHandler with an action for a typed Error E which will return an error or nil back as the Try return.
func Catch[E Error](action func(E) Error) ErrorHandler {
	return typedCatch[E]{}.New(action)
}

// CatchName() returns an ErrorHandler with anction for a named Error which will return an error or nil back as the Try return.
func CatchName(name string, action func(Error) Error) ErrorHandler {
	return namedCatch{}.New(name, action)
}

// Finally() returns an ErrorHandler which executes after, whatever the outcome.
//
// Internally, its [ErrorHandler.Supports] method will return true only if its argument applied to [IsNilError] returns true,
// and its [ErrorHandler.Catch] method will run action, without regarding its argument.
//
// That's the reason an [ErrorHandler.Supports] method must check for [IsNilError].
func Finally(action func()) ErrorHandler {
	return finallyCatch{}.New(action)
}

// RaiseVal() helps panic a Try() action.
// To be used around a function returning an object and optionally an error.
// If the error is not nil it will use it to panic() otherwise it returns the object T.
func RaiseVal[T any](t T, e error) T {
	RaiseErr(e)
	return t
}

// RaiseErr() helps panic a Try() action.
// To be used around a function returning optionally an error.
// If the error is not nil it will use it to panic() otherwise it returns false.
func RaiseErr(err error) bool {
	if err != nil {
		panic(err)
	}
	return false
}

// RaiseNilErr() helps panic a Try() action.
// To be used around a function returning an object and optionally an error.
// If the error is not nil it will use it to panic() otherwise it will panic() with an NilError.
//
// It only usage is on untestable situation where an error may occur. It will be returned as an Error,
// in any cases, letting you can [Error.LogError] on it and not doing anything if it's a [NilError].
func RaiseNilErr(err error) bool {
	if err != nil {
		panic(err)
	}
	panic(&NilError{})
}

// Try() will execute a function try which may panic, resolving the different ErrorHandler defined and returning an Error if it occurred.
//
// Note on Go Idioms: While using Try/Catch and panics for standard error handling is considered an anti-pattern
// in idiomatic Go (which heavily favors explicit `if err != nil` returns), this helper has its perks.
// It simplifies code that must catch unexpected runtime panics triggered underneath, without forcing
// the developer to rely on `recover()` directly, securely catching everything into a managed Error.
func Try(try func(), catches ...ErrorHandler) Error {
	var err Error
	func() {
		// Recovery
		defer func() {
			if e := recover(); e != nil {
				if reflect.TypeOf(e).Implements(reflect.TypeFor[Error]()) {
					err = e.(Error)
				} else {
					switch v := e.(type) {
					case error:
						err = NewPanicError(v.Error())
					default:
						err = NewPanicError(fmt.Sprintf("%v", v))
					}
				}
			}
		}()
		// Calling try()
		try()
	}()

	// Finding the first ErrorHandler supporting "nil"
	var finally ErrorHandler
	for _, catch := range catches {
		if f := catch.Supports(nil); f {
			finally = catch
			break
		}
	}

	// Defering finally
	if finally != nil {
		//nolint:errcheck // it will always be nil
		defer finally.Catch(nil)
	}

	// Finding a suitable Catch.
	if err != nil {
		for _, catch := range catches {
			if catch.Supports(err) {
				return catch.Catch(err)
			}
		}
		return err
	}

	return nil
}
