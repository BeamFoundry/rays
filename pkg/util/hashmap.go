package util

import (
	"encoding/json"
	"iter"
	"maps"
	"reflect"

	. "github.com/BeamFoundry/rays/pkg/lang"
)

var _ Map[string, any] = &HashMap[string, any]{}

// HashMap is default [Map] implementation
type HashMap[K comparable, V any] struct {
	m map[K]V
}

// NewHashMap() creates a new, empty [HashMap]
func NewHashMap[K comparable, V any]() HashMap[K, V] {
	return HashMap[K, V]{m: map[K]V{}}
}

// NewHashMapFor() encapsulate an existing map as a [HashMap]
func NewHashMapFor[K comparable, V any](m map[K]V) HashMap[K, V] {
	return HashMap[K, V]{m: m}
}

// NewHashMapFrom() encapsulate an existing map as a [HashMap] and clones it.
func NewHashMapFrom[K comparable, V any](m map[K]V) HashMap[K, V] {
	return NewHashMapFor(m).Clone()
}

// Map() implements [Map.Map]
func (this HashMap[K, V]) Map() map[K]V {
	return NewHashMapFrom(this.m).m
}

// Clear() implements [Map.Clear]
func (this HashMap[K, V]) Clear() {
	clear(this.m)
}

/*
func (this HashMap[K, V]) Clone() HashMap[K, V] {
	var clone func(m map[K]V) map[K]V
	clone = func(m map[K]V) map[K]V {
		cl := maps.Clone(m)
		value := reflect.ValueOf(cl)
		for _, k := range value.MapKeys() {
			if value.MapIndex(k).Type() == value.Type() {
				value.MapIndex(k).Set(reflect.ValueOf(clone(value.MapIndex(k).Interface().(map[K]V))))
			}
		}
		return cl
	}
	return HashMap[K, V]{m: clone(this.m)}
}
*/

// Clone() clones the current map
func (this HashMap[K, V]) Clone() HashMap[K, V] {
	return HashMap[K, V]{m: maps.Clone(this.m)}
}

// ContainsKey() implements [Map.ContainsKey]
func (this HashMap[K, V]) ContainsKey(key K) bool {
	_, ok := this.m[key]
	return ok
}

// Merge() implements [Map.Merge].
func (this HashMap[K, V]) Merge(m Map[K, V]) {
	for k, src := range m.Iterator() {
		srcType := reflect.TypeOf(src)
		if dst, ok := this.m[k]; !ok {
			if srcType != nil && srcType.Kind() == reflect.Map {
				n := map[K]V{}
				HashMap[K, V]{m: n}.Merge(HashMap[K, V]{m: any(src).(map[K]V)})
				this.m[k] = any(n).(V)
			} else {
				this.m[k] = src
			}
		} else {
			dstType := reflect.TypeOf(dst)

			if srcType == dstType && srcType != nil && srcType.Kind() == reflect.Map {
				HashMap[K, V]{m: any(dst).(map[K]V)}.Merge(HashMap[K, V]{m: any(src).(map[K]V)})
			} else {
				this.m[k] = src
			}
		}
	}
}

// Iterator() implements [Map.Iterator]
func (this HashMap[K, V]) Iterator() iter.Seq2[K, V] {
	return maps.All(maps.Clone(this.m))
}

// Equals() implements [Map.Equals].
//
// It does it using [reflect.DeepEqual]
func (this HashMap[K, V]) Equals(m Map[K, V]) bool {
	return reflect.DeepEqual(this.m, m.Map())
}

// Get() implements [Map.Get]
func (this HashMap[K, V]) Get(key K) Optional[V] {
	v, ok := this.m[key]
	if !ok {
		return OptionalEmpty[V]()
	}
	return OptionalOf(v)
}

// Remove() implements [Map.Remove]
func (this HashMap[K, V]) Remove(key K) Optional[V] {
	v, ok := this.m[key]
	if !ok {
		return OptionalEmpty[V]()
	}
	optional := OptionalOf(v)
	delete(this.m, key)
	return optional
}

// Delete() implements [Map.Delete]
func (this HashMap[K, V]) Delete(f func(key K, val V) bool) {
	maps.DeleteFunc(this.m, f)
}

// IsEmpty() implements [Map.IsEmpty]
func (this HashMap[K, V]) IsEmpty() bool {
	return len(this.m) == 0
}

// Put() implements [Map.Put]
func (this HashMap[K, V]) Put(key K, val V) Map[K, V] {
	this.m[key] = val
	return this
}

// PutAll() implements [Map.PutAll]
func (this HashMap[K, V]) PutAll(m Map[K, V]) Map[K, V] {
	maps.Copy(this.m, m.Map())
	return this
}

// Size() implements [Map.Size]
func (this HashMap[K, V]) Size() int {
	return len(this.m)
}

// MashalJSON() implements [json.Marshaler]
func (this HashMap[K, V]) MarshalJSON() ([]byte, error) {
	//nolint:wrapcheck // Will return the error as is.
	return json.Marshal(this.m)
}

// UnmarshalJSON() implements [json.Unmarshal]
func (this HashMap[K, V]) UnmarshalJSON(data []byte) error {
	//nolint:wrapcheck // Will return the error as is.
	return json.Unmarshal(data, &this.m)
}
