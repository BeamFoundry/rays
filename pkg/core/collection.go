package core

import (
	"iter"
)

// Collection represents the root interface in the collection hierarchy.
// A collection represents a group of objects, known as its elements.
type Collection[T any] interface {
	// Size returns the number of elements in this collection.
	Size() int

	// IsEmpty returns true if this collection contains no elements.
	IsEmpty() bool

	// Contains returns true if this collection contains the specified element.
	Contains(element T) bool

	// ContainsAll returns true if this collection contains all of the elements in the specified collection.
	ContainsAll(c Collection[T]) bool

	// Add ensures that this collection contains the specified elements.
	// Returns true if this collection changed as a result of the call.
	Add(elements ...T) bool

	// AddAll adds all of the elements in the specified collection to this collection.
	// Returns true if this collection changed as a result of the call.
	AddAll(c Collection[T]) bool

	// Remove removes a single instance of the specified element from this collection, if it is present.
	// Returns true if an element was removed.
	Remove(element T) bool

	// RemoveAll removes all of this collection's elements that are also contained in the specified collection.
	// Returns true if this collection changed as a result of the call.
	RemoveAll(c Collection[T]) bool

	// RetainAll retains only the elements in this collection that are contained in the specified collection.
	// Returns true if this collection changed as a result of the call.
	RetainAll(c Collection[T]) bool

	// Clear removes all of the elements from this collection.
	Clear()

	// ToSlice returns a slice containing all of the elements in this collection.
	ToSlice() []T

	// Seq returns an iterator sequence (iter.Seq[T]) over the elements in this collection,
	// enabling direct Go 1.23+ range loops: for item := range c.Seq()
	Seq() iter.Seq[T]

	// Equals returns true if the specified collection is equal to this collection.
	Equals(c Collection[T]) bool
}

// List represents an ordered collection (also known as a sequence).
// The user can access elements by their integer index and search for elements in the list.
type List[T any] interface {
	Collection[T]

	// Get returns the element at the specified position in this list.
	// Panics with IndexOutOfBoundsError if the index is out of range (index < 0 || index >= Size()).
	Get(index int) T

	// Set replaces the element at the specified position in this list with the specified element.
	// Returns the element previously at the specified position.
	// Panics with IndexOutOfBoundsError if the index is out of range.
	Set(index int, element T) T

	// AddAt inserts the specified element at the specified position in this list.
	// Shifts the element currently at that position (if any) and any subsequent elements to the right.
	// Panics with IndexOutOfBoundsError if the index is out of range (index < 0 || index > Size()).
	AddAt(index int, element T)

	// RemoveAt removes the element at the specified position in this list.
	// Shifts any subsequent elements to the left.
	// Returns the element that was removed from the list.
	// Panics with IndexOutOfBoundsError if the index is out of range.
	RemoveAt(index int) T

	// IndexOf returns the index of the first occurrence of the specified element in this list,
	// or -1 if this list does not contain the element.
	IndexOf(element T) int

	// LastIndexOf returns the index of the last occurrence of the specified element in this list,
	// or -1 if this list does not contain the element.
	LastIndexOf(element T) int

	// SubList returns an independent copy of the portion of this list between the specified fromIndex, inclusive,
	// and toIndex, exclusive. Changes to the returned list do not affect the original list.
	SubList(fromIndex, toIndex int) List[T]
}

// Set represents a collection that contains no duplicate elements.
// As elements are compared for equality, element type T must satisfy comparable.
type Set[T comparable] interface {
	Collection[T]
}
