package util

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"iter"

	. "github.com/BeamFoundry/rays/pkg/core"
	. "github.com/BeamFoundry/rays/pkg/util/internal"
	. "github.com/BeamFoundry/rays/pkg/util/stream"
)

// HashSet is a hash table based implementation of the Set interface.
type HashSet[T comparable] struct {
	AbstractSet[T]
}

var (
	_ Set[int]         = (*HashSet[int])(nil)
	_ json.Marshaler   = (*HashSet[int])(nil)
	_ json.Unmarshaler = (*HashSet[int])(nil)
	_ driver.Valuer    = (*HashSet[int])(nil)
	_ sql.Scanner      = (*HashSet[int])(nil)

	errNilHashSet = errors.New("cannot operate on nil *HashSet")
)

// NewHashSet creates a new HashSet containing the given elements.
func NewHashSet[T comparable](elements ...T) *HashSet[T] {
	return &HashSet[T]{
		AbstractSet: *NewAbstractSet(elements...),
	}
}

// NewHashSetWithCapacity creates an empty HashSet with the specified initial capacity.
func NewHashSetWithCapacity[T comparable](capacity int) *HashSet[T] {
	return &HashSet[T]{
		AbstractSet: *NewAbstractSetWithCapacity[T](capacity),
	}
}

// Size returns the number of elements in this set.
func (this *HashSet[T]) Size() int {
	return this.AbstractSet.Size()
}

// IsEmpty returns true if this set contains no elements.
func (this *HashSet[T]) IsEmpty() bool {
	return this.AbstractSet.IsEmpty()
}

// Contains returns true if this set contains the specified element.
func (this *HashSet[T]) Contains(element T) bool {
	return this.AbstractSet.Contains(element)
}

// ContainsAll returns true if this set contains all of the elements in the specified collection.
func (this *HashSet[T]) ContainsAll(c Collection[T]) bool {
	return this.AbstractSet.ContainsAll(c)
}

// Add adds the specified elements to this set.
// Returns true if this set changed as a result of the call.
func (this *HashSet[T]) Add(elements ...T) bool {
	return this.AbstractSet.Add(elements...)
}

// AddAll adds all of the elements in the specified collection to this set.
// Returns true if this set changed as a result of the call.
func (this *HashSet[T]) AddAll(c Collection[T]) bool {
	return this.AbstractSet.AddAll(c)
}

// Remove removes the specified element from this set, if it is present.
// Returns true if an element was removed.
func (this *HashSet[T]) Remove(element T) bool {
	return this.AbstractSet.Remove(element)
}

// RemoveAll removes all of this set's elements that are also contained in the specified collection.
// Returns true if this set changed as a result of the call.
func (this *HashSet[T]) RemoveAll(c Collection[T]) bool {
	return this.AbstractSet.RemoveAll(c)
}

// RetainAll retains only the elements in this set that are contained in the specified collection.
// Returns true if this set changed as a result of the call.
func (this *HashSet[T]) RetainAll(c Collection[T]) bool {
	return this.AbstractSet.RetainAll(c)
}

// Clear removes all elements from this set.
func (this *HashSet[T]) Clear() {
	this.AbstractSet.Clear()
}

// ToSlice returns a slice containing all elements in this set.
func (this *HashSet[T]) ToSlice() []T {
	return this.AbstractSet.ToSlice()
}

// Seq returns an iterator sequence over the elements in this set.
func (this *HashSet[T]) Seq() iter.Seq[T] {
	return this.AbstractSet.Seq()
}

// Equals returns true if the specified collection is equal to this set.
func (this *HashSet[T]) Equals(c Collection[T]) bool {
	return this.AbstractSet.Equals(c)
}

// Stream returns a sequential Stream with this set as its source.
func (this *HashSet[T]) Stream() Stream[T] {
	return Stream[T]{}.FromSeq(this.Seq())
}

// MarshalJSON implements json.Marshaler.
func (this *HashSet[T]) MarshalJSON() ([]byte, error) {
	if this == nil {
		return []byte("[]"), nil
	}
	return this.AbstractSet.MarshalJSON()
}

// UnmarshalJSON implements json.Unmarshaler.
func (this *HashSet[T]) UnmarshalJSON(data []byte) error {
	if this == nil {
		return errNilHashSet
	}
	return this.AbstractSet.UnmarshalJSON(data)
}

// Value implements driver.Valuer.
//
//nolint:nilnil // driver.Valuer returns nil, nil to represent SQL NULL.
func (this *HashSet[T]) Value() (driver.Value, error) {
	if this == nil {
		return nil, nil
	}
	return this.AbstractSet.Value()
}

// Scan implements sql.Scanner.
func (this *HashSet[T]) Scan(src any) error {
	if this == nil {
		return errNilHashSet
	}
	return this.AbstractSet.Scan(src)
}
