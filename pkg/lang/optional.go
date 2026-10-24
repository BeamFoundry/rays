package lang

import (
	"reflect"
)

type Optional[T any] struct {
	value   T
	present bool
}

// OptionalEmpty() returns an empty Optional instance.
func OptionalEmpty[T any]() Optional[T] {
	return Optional[T]{}
}

// OptionalOf() returns an Optional describing the specified value, if non-null, otherwise returns an empty Optional.
//
// Contraty to [OptionalOfNullable], this function will infere [T] from the argument "value".
//
// As such this will not be accepted and produce a compile error :
//
//	value := OptionalOf[string](nil)
//
// Passing a nil pointer will produce a OptionalEmpty.
//
//	var value *MyStruct = nil
//	opt := OptionalOf(value)
//	opt.IsPresent() // returns false
func OptionalOf[T any](value T) Optional[T] {
	opt := Optional[T]{}
	opt.SetValue(value)
	return opt
}

// OptionalOfNullable() returns an Optional[T] with the specified present non-nil value or an empty Optional.
//
// Contrary to [OptionalOf], this function needs the type to be specified, as the value argument is of type "any".
//
// For example, this will be accepted and produce an OptionalEmpty[string]().
//
//	opt := OptionalOfNullable[string](nil)
//	opt.IsPresent() // returns false
//
// Passing a nil pointer will produce a OptionalEmpty.
//
//	var value *MyStruct = nil
//	opt := OptionalOfNullable[*MyStruct](value)
//	opt.IsPresent() // returns false
func OptionalOfNullable[T any](value any) Optional[T] {
	opt := Optional[T]{}
	if err := opt.SetAnyValue(value); err != nil {
		return OptionalEmpty[T]()
	}
	return opt
}

// ValueType() returns a [reflect.ValueType] of T
func (this Optional[T]) ValueType() reflect.Type {
	return reflect.TypeFor[T]()
}

// SetValue() let's you set the value of T, if the value passed is not nil.
func (this *Optional[T]) SetValue(value T) {
	if this.isValid(value) {
		this.value = value
		this.present = true
	}
}

// SetAnyValue() let's you set the value of T, if the value passed is correct.
func (this *Optional[T]) SetAnyValue(value any) Error {
	if !this.isValid(value) {
		return NewError("InvalidValueError", "Provided value is nil")
	}
	if val, ok := value.(T); ok {
		this.value = val
		this.present = true
		return nil
	}
	return NewError("WrongTypeError", "Generic wrong type")
}

func (this *Optional[T]) isValid(value any) bool {
	var invalid bool
	func() {
		defer func() {
			//nolint:errcheck // recover
			recover()
		}()
		v := reflect.ValueOf(value)
		if !v.IsValid() || v.IsNil() {
			invalid = true
		}
	}()
	return !invalid
}

// Get() returns the value (or its zero value) and if it is present
func (this Optional[T]) Get() (T, bool) {
	return this.value, this.present
}

// IsEmpty() if a value is not present, returns true, otherwise false.
func (this Optional[T]) IsEmpty() bool {
	return !this.present
}

// IsPresent() if a value is present, returns true, otherwise false.
func (this Optional[T]) IsPresent() bool {
	return this.present
}

// IfPresent() if a value is present, performs the given action with the value, otherwise does nothing.
func (this Optional[T]) IfPresent(action func(T)) {
	if this.present {
		action(this.value)
	}
}

// IfPresentOrElse() if a value is present, performs the given action with the value, otherwise performs the given empty-based action.
func (this Optional[T]) IfPresentOrElse(action func(T), empty func()) {
	if this.present {
		action(this.value)
	} else {
		empty()
	}
}

// OrElse() if a value is present, returns the value, otherwise returns other.
func (this Optional[T]) OrElse(other T) T {
	if this.present {
		return this.value
	}
	return other
}

// OrElseGet() if a value is present returns the value, otherwise returns the value from the supplier function.
func (this Optional[T]) OrElseGet(supplier func() T) T {
	if this.present {
		return this.value
	}
	return supplier()
}

// Map() if a value is present, returns an Optional describing (as if by OptionalOfNullable the result of applying the given mapping function to the value, otherwise returns an empty Optional.
func (this Optional[T]) Map(mapper func(T) T) Optional[T] {
	if this.present {
		return OptionalOf(mapper(this.value))
	}
	return OptionalEmpty[T]()
}

// FlatMap() if a value is present, returns the result of applying the given Optional-bearing mapping function to the value, otherwise returns an empty Optional.
func (this Optional[T]) FlatMap(mapper func(T) Optional[T]) Optional[T] {
	if this.present {
		return mapper(this.value)
	}
	return OptionalEmpty[T]()
}
