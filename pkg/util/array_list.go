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

// ArrayList is a resizable-array implementation of the List interface.
type ArrayList[T any] struct {
	AbstractList[T]
}

var (
	_ List[int]        = (*ArrayList[int])(nil)
	_ json.Marshaler   = (*ArrayList[int])(nil)
	_ json.Unmarshaler = (*ArrayList[int])(nil)
	_ driver.Valuer    = (*ArrayList[int])(nil)
	_ sql.Scanner      = (*ArrayList[int])(nil)

	errNilArrayList = errors.New("cannot operate on nil *ArrayList")
)

// NewArrayList creates a new ArrayList containing the given elements.
func NewArrayList[T any](elements ...T) *ArrayList[T] {
	return &ArrayList[T]{
		AbstractList: *NewAbstractList(elements...),
	}
}

// NewArrayListWithCapacity creates an empty ArrayList with the specified initial capacity.
func NewArrayListWithCapacity[T any](capacity int) *ArrayList[T] {
	return &ArrayList[T]{
		AbstractList: *NewAbstractListWithCapacity[T](capacity),
	}
}

// Size returns the number of elements in this list.
func (this *ArrayList[T]) Size() int {
	return this.AbstractList.Size()
}

// IsEmpty returns true if this list contains no elements.
func (this *ArrayList[T]) IsEmpty() bool {
	return this.AbstractList.IsEmpty()
}

// Get returns the element at the specified position in this list.
// Panics with IndexOutOfBoundsError if the index is out of range.
func (this *ArrayList[T]) Get(index int) T {
	return this.AbstractList.Get(index)
}

// Set replaces the element at the specified position in this list with the specified element.
// Returns the element previously at the specified position.
// Panics with IndexOutOfBoundsError if the index is out of range.
func (this *ArrayList[T]) Set(index int, element T) T {
	return this.AbstractList.Set(index, element)
}

// Add appends the specified elements to the end of this list.
// Returns true if this list changed as a result of the call.
func (this *ArrayList[T]) Add(elements ...T) bool {
	return this.AbstractList.Add(elements...)
}

// AddAt inserts the specified element at the specified position in this list.
// Shifts the element currently at that position (if any) and any subsequent elements to the right.
// Panics with IndexOutOfBoundsError if the index is out of range.
func (this *ArrayList[T]) AddAt(index int, element T) {
	this.AbstractList.AddAt(index, element)
}

// IndexOf returns the index of the first occurrence of the specified element in this list,
// or -1 if this list does not contain the element.
func (this *ArrayList[T]) IndexOf(element T) int {
	return this.AbstractList.IndexOf(element)
}

// LastIndexOf returns the index of the last occurrence of the specified element in this list,
// or -1 if this list does not contain the element.
func (this *ArrayList[T]) LastIndexOf(element T) int {
	return this.AbstractList.LastIndexOf(element)
}

// Contains returns true if this list contains the specified element.
func (this *ArrayList[T]) Contains(element T) bool {
	return this.AbstractList.Contains(element)
}

// ContainsAll returns true if this list contains all of the elements in the specified collection.
func (this *ArrayList[T]) ContainsAll(c Collection[T]) bool {
	return this.AbstractList.ContainsAll(c)
}

// AddAll appends all of the elements in the specified collection to the end of this list.
// Returns true if this list changed as a result of the call.
func (this *ArrayList[T]) AddAll(c Collection[T]) bool {
	return this.AbstractList.AddAll(c)
}

// Remove removes the first occurrence of the specified element from this list, if it is present.
// Returns true if an element was removed.
func (this *ArrayList[T]) Remove(element T) bool {
	return this.AbstractList.Remove(element)
}

// RemoveAt removes the element at the specified position in this list.
// Shifts any subsequent elements to the left.
// Returns the element that was removed from the list.
// Panics with IndexOutOfBoundsError if the index is out of range.
func (this *ArrayList[T]) RemoveAt(index int) T {
	return this.AbstractList.RemoveAt(index)
}

// RemoveAll removes all of this list's elements that are also contained in the specified collection.
// Returns true if this list changed as a result of the call.
func (this *ArrayList[T]) RemoveAll(c Collection[T]) bool {
	return this.AbstractList.RemoveAll(c)
}

// RetainAll retains only the elements in this list that are contained in the specified collection.
// Returns true if this list changed as a result of the call.
func (this *ArrayList[T]) RetainAll(c Collection[T]) bool {
	return this.AbstractList.RetainAll(c)
}

// Clear removes all elements from this list.
func (this *ArrayList[T]) Clear() {
	this.AbstractList.Clear()
}

// ToSlice returns a copy of the underlying slice.
func (this *ArrayList[T]) ToSlice() []T {
	return this.AbstractList.ToSlice()
}

// Seq returns an iterator sequence over the elements in this list.
func (this *ArrayList[T]) Seq() iter.Seq[T] {
	return this.AbstractList.Seq()
}

// SubList returns an independent copy of the portion of this list between fromIndex, inclusive,
// and toIndex, exclusive. Changes to the returned list do not affect the original list.
// Panics with IndexOutOfBoundsError if the indices are out of range.
func (this *ArrayList[T]) SubList(fromIndex, toIndex int) List[T] {
	return this.AbstractList.SubList(fromIndex, toIndex)
}

// Equals returns true if the specified collection is equal to this list.
func (this *ArrayList[T]) Equals(c Collection[T]) bool {
	return this.AbstractList.Equals(c)
}

// Stream returns a sequential Stream with this list as its source.
func (this *ArrayList[T]) Stream() Stream[T] {
	return Stream[T]{}.FromSeq(this.Seq())
}

// MarshalJSON implements json.Marshaler.
func (this *ArrayList[T]) MarshalJSON() ([]byte, error) {
	if this == nil {
		return []byte("[]"), nil
	}
	return this.AbstractList.MarshalJSON()
}

// UnmarshalJSON implements json.Unmarshaler.
func (this *ArrayList[T]) UnmarshalJSON(data []byte) error {
	if this == nil {
		return errNilArrayList
	}
	return this.AbstractList.UnmarshalJSON(data)
}

// Value implements driver.Valuer.
//
//nolint:nilnil // driver.Valuer returns nil, nil to represent SQL NULL.
func (this *ArrayList[T]) Value() (driver.Value, error) {
	if this == nil {
		return nil, nil
	}
	return this.AbstractList.Value()
}

// Scan implements sql.Scanner.
func (this *ArrayList[T]) Scan(src any) error {
	if this == nil {
		return errNilArrayList
	}
	return this.AbstractList.Scan(src)
}
