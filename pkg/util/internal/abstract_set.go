package internal

import (
	"iter"

	. "github.com/BeamFoundry/rays/pkg/core"
)

// AbstractSet provides a skeletal implementation of the Set interface,
// backed by a Go map.
type AbstractSet[T comparable] struct {
	elements map[T]struct{}
}

var _ Set[int] = (*AbstractSet[int])(nil)

// NewAbstractSet creates a new AbstractSet with the given elements.
func NewAbstractSet[T comparable](elements ...T) *AbstractSet[T] {
	m := make(map[T]struct{}, len(elements))
	for _, el := range elements {
		m[el] = struct{}{}
	}
	return &AbstractSet[T]{elements: m}
}

// NewAbstractSetWithCapacity creates a new empty AbstractSet with the specified capacity.
func NewAbstractSetWithCapacity[T comparable](capacity int) *AbstractSet[T] {
	if capacity < 0 {
		capacity = 0
	}
	return &AbstractSet[T]{elements: make(map[T]struct{}, capacity)}
}

// Size returns the number of elements in this set.
func (this *AbstractSet[T]) Size() int {
	return len(this.elements)
}

// IsEmpty returns true if this set contains no elements.
func (this *AbstractSet[T]) IsEmpty() bool {
	return len(this.elements) == 0
}

// Contains returns true if this set contains the specified element.
func (this *AbstractSet[T]) Contains(element T) bool {
	_, ok := this.elements[element]
	return ok
}

// ContainsAll returns true if this set contains all of the elements in the specified collection.
func (this *AbstractSet[T]) ContainsAll(c Collection[T]) bool {
	if c == nil {
		return false
	}
	for item := range c.Seq() {
		if !this.Contains(item) {
			return false
		}
	}
	return true
}

// Add adds the specified elements to this set.
func (this *AbstractSet[T]) Add(elements ...T) bool {
	modified := false
	for _, el := range elements {
		if _, exists := this.elements[el]; !exists {
			this.elements[el] = struct{}{}
			modified = true
		}
	}
	return modified
}

// AddAll adds all of the elements in the specified collection to this set.
func (this *AbstractSet[T]) AddAll(c Collection[T]) bool {
	if c == nil {
		return false
	}
	modified := false
	for item := range c.Seq() {
		if _, exists := this.elements[item]; !exists {
			this.elements[item] = struct{}{}
			modified = true
		}
	}
	return modified
}

// Remove removes the specified element from this set, if it is present.
func (this *AbstractSet[T]) Remove(element T) bool {
	if _, exists := this.elements[element]; exists {
		delete(this.elements, element)
		return true
	}
	return false
}

// RemoveAll removes all of this set's elements that are also contained in the specified collection.
func (this *AbstractSet[T]) RemoveAll(c Collection[T]) bool {
	if c == nil {
		return false
	}
	modified := false
	for item := range c.Seq() {
		if this.Remove(item) {
			modified = true
		}
	}
	return modified
}

// RetainAll retains only the elements in this set that are contained in the specified collection.
func (this *AbstractSet[T]) RetainAll(c Collection[T]) bool {
	if c == nil {
		return false
	}
	modified := false
	for item := range this.elements {
		if !c.Contains(item) {
			delete(this.elements, item)
			modified = true
		}
	}
	return modified
}

// Clear removes all elements from this set.
func (this *AbstractSet[T]) Clear() {
	clear(this.elements)
}

// ToSlice returns a slice containing all elements in this set.
func (this *AbstractSet[T]) ToSlice() []T {
	result := make([]T, 0, len(this.elements))
	for item := range this.elements {
		result = append(result, item)
	}
	return result
}

// Seq returns an iterator sequence over the elements in this set.
func (this *AbstractSet[T]) Seq() iter.Seq[T] {
	return func(yield func(T) bool) {
		for item := range this.elements {
			if !yield(item) {
				return
			}
		}
	}
}

// Equals returns true if the specified collection is equal to this set.
func (this *AbstractSet[T]) Equals(c Collection[T]) bool {
	if c == nil || this.Size() != c.Size() {
		return false
	}
	return this.ContainsAll(c)
}
