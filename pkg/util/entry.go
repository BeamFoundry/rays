package util

import (
	"reflect"
)

// MapEntry is default [MapEntry] implementation.
type MapEntry[K comparable, V any] struct {
	key   K
	value V
}

// KeyType() implements [MapEntry.KeyType].
//
// It returns the [reflect.Type] of K.
func (this *MapEntry[K, V]) KeyType() reflect.Type {
	return reflect.TypeFor[K]()
}

// ValueType() implements [MapEntry.ValueType].
//
// It returns the [reflect.Type] of V.
func (this *MapEntry[K, V]) ValueType() reflect.Type {
	return reflect.TypeFor[V]()
}

// Key() implements [MapEntry.Key].
//
// It returns the key corresponding to this entry.
func (this *MapEntry[K, V]) Key() K {
	return this.key
}

// Value() implements [MapEntry.Value].
//
// It returns the value corresponding to this entry.
func (this *MapEntry[K, V]) Value() V {
	return this.value
}

// SetValue() implements [MapEntry.SetValue].
//
// It replaces the value corresponding to this entry with the specified value.
func (this *MapEntry[K, V]) SetValue(value V) {
	this.value = value
}

// Equals() implements [MapEntry.Equals].
//
// It compares the specified object with this entry for equality using [reflect.DeepEqual].
func (this *MapEntry[K, V]) Equals(value V) bool {
	return reflect.DeepEqual(this.value, value)
}
