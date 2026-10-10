package util

import (
	"iter"

	. "github.com/BeamFoundry/rays/pkg/lang"
)

// Map is an interface for an object oriented map.
//
// TODO: Add Clone and JSON Marshaler/Unmarshaler ???
type Map[K comparable, V any] interface {
	// Map() returns a snapshot of the underlying map as a Go native map.
	//
	// This should be a shallow copy of the Map.
	Map() map[K]V

	// Clear() empty the map
	Clear()

	// ContainsKey() returns whether the map contains that key
	ContainsKey(key K) bool

	// Merge() will merge the given map
	//
	// To the difference of PutAll(), this will try to merge a map recusively.
	Merge(m Map[K, V])

	// Iterator() returns an iterator over the underlying map.
	//
	// This should be done over a snapshot (shallow copy) of the Map.
	Iterator() iter.Seq2[K, V]

	// Equals() returns whether a Map equals to this one.
	Equals(m Map[K, V]) bool

	// Get() returns optionally the requested key
	Get(key K) Optional[V]

	// Remove() returns optionally the requested key after removing it from the map
	Remove(key K) Optional[V]

	// Delete() apply a filter to remove key/values from the map
	Delete(f func(key K, val V) bool)

	// IsEmpty() returns whether the Map is empty or not
	IsEmpty() bool

	// Put() add (or replace) a key/value in the Map.
	Put(key K, val V) Map[K, V]

	// PutAll() merge the given Map and returns it.
	//
	// To the difference of Merge() this will replace existing key without merging Map entry of the same type.
	PutAll(m Map[K, V]) Map[K, V]

	// Size() returns the size of the Map
	Size() int
}
