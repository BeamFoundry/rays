package internal

import (
	"iter"
	"reflect"
	"slices"

	. "github.com/BeamFoundry/rays/pkg/core"
	. "github.com/BeamFoundry/rays/pkg/lang"
)

// AbstractList provides a skeletal implementation of the List interface,
// backed by a slice.
type AbstractList[T any] struct {
	items []T
}

var _ List[int] = (*AbstractList[int])(nil)

// NewAbstractList creates a new AbstractList with the given elements.
func NewAbstractList[T any](elements ...T) *AbstractList[T] {
	items := make([]T, len(elements))
	copy(items, elements)
	return &AbstractList[T]{items: items}
}

// NewAbstractListWithCapacity creates a new empty AbstractList with the specified capacity.
func NewAbstractListWithCapacity[T any](capacity int) *AbstractList[T] {
	if capacity < 0 {
		capacity = 0
	}
	return &AbstractList[T]{items: make([]T, 0, capacity)}
}

// Size returns the number of elements in this list.
func (this *AbstractList[T]) Size() int {
	return len(this.items)
}

// IsEmpty returns true if this list contains no elements.
func (this *AbstractList[T]) IsEmpty() bool {
	return len(this.items) == 0
}

// Get returns the element at the specified position in this list.
func (this *AbstractList[T]) Get(index int) T {
	if index < 0 || index >= len(this.items) {
		panic(NewIndexOutOfBoundsError("list index out of range"))
	}
	return this.items[index]
}

// Set replaces the element at the specified position in this list with the specified element.
func (this *AbstractList[T]) Set(index int, element T) T {
	if index < 0 || index >= len(this.items) {
		panic(NewIndexOutOfBoundsError("list index out of range"))
	}
	old := this.items[index]
	this.items[index] = element
	return old
}

// Add appends the specified elements to the end of this list.
func (this *AbstractList[T]) Add(elements ...T) bool {
	if len(elements) == 0 {
		return false
	}
	this.items = append(this.items, elements...)
	return true
}

// AddAt inserts the specified element at the specified position in this list.
func (this *AbstractList[T]) AddAt(index int, element T) {
	if index < 0 || index > len(this.items) {
		panic(NewIndexOutOfBoundsError("list index out of range"))
	}
	if index == len(this.items) {
		this.items = append(this.items, element)
		return
	}
	this.items = append(this.items[:index+1], this.items[index:]...)
	this.items[index] = element
}

// RemoveAt removes the element at the specified position in this list.
func (this *AbstractList[T]) RemoveAt(index int) T {
	if index < 0 || index >= len(this.items) {
		panic(NewIndexOutOfBoundsError("list index out of range"))
	}
	removed := this.items[index]
	copy(this.items[index:], this.items[index+1:])
	var zero T
	this.items[len(this.items)-1] = zero
	this.items = this.items[:len(this.items)-1]
	return removed
}

// IndexOf returns the index of the first occurrence of the specified element in this list,
// or -1 if this list does not contain the element.
func (this *AbstractList[T]) IndexOf(element T) int {
	for i, item := range this.items {
		if reflect.DeepEqual(item, element) {
			return i
		}
	}
	return -1
}

// LastIndexOf returns the index of the last occurrence of the specified element in this list,
// or -1 if this list does not contain the element.
func (this *AbstractList[T]) LastIndexOf(element T) int {
	for i, v := range slices.Backward(this.items) {
		if reflect.DeepEqual(v, element) {
			return i
		}
	}
	return -1
}

// Contains returns true if this list contains the specified element.
func (this *AbstractList[T]) Contains(element T) bool {
	return this.IndexOf(element) != -1
}

// ContainsAll returns true if this list contains all of the elements in the specified collection.
func (this *AbstractList[T]) ContainsAll(c Collection[T]) bool {
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

// AddAll adds all of the elements in the specified collection to this list.
func (this *AbstractList[T]) AddAll(c Collection[T]) bool {
	if c == nil {
		return false
	}
	modified := false
	for item := range c.Seq() {
		this.items = append(this.items, item)
		modified = true
	}
	return modified
}

// Remove removes the first occurrence of the specified element from this list, if it is present.
func (this *AbstractList[T]) Remove(element T) bool {
	idx := this.IndexOf(element)
	if idx == -1 {
		return false
	}
	this.RemoveAt(idx)
	return true
}

// RemoveAll removes all of this list's elements that are also contained in the specified collection.
func (this *AbstractList[T]) RemoveAll(c Collection[T]) bool {
	if c == nil {
		return false
	}
	newItems := make([]T, 0, len(this.items))
	modified := false
	for _, item := range this.items {
		if c.Contains(item) {
			modified = true
		} else {
			newItems = append(newItems, item)
		}
	}
	if modified {
		this.items = newItems
	}
	return modified
}

// RetainAll retains only the elements in this list that are contained in the specified collection.
func (this *AbstractList[T]) RetainAll(c Collection[T]) bool {
	if c == nil {
		return false
	}
	newItems := make([]T, 0, len(this.items))
	modified := false
	for _, item := range this.items {
		if c.Contains(item) {
			newItems = append(newItems, item)
		} else {
			modified = true
		}
	}
	if modified {
		this.items = newItems
	}
	return modified
}

// Clear removes all elements from this list.
func (this *AbstractList[T]) Clear() {
	clear(this.items)
	this.items = this.items[:0]
}

// ToSlice returns a copy of the underlying slice.
func (this *AbstractList[T]) ToSlice() []T {
	result := make([]T, len(this.items))
	copy(result, this.items)
	return result
}

// Seq returns an iterator sequence over the elements in this list.
func (this *AbstractList[T]) Seq() iter.Seq[T] {
	return func(yield func(T) bool) {
		for _, item := range this.items {
			if !yield(item) {
				return
			}
		}
	}
}

// SubList returns an independent copy of the portion of this list between fromIndex, inclusive,
// and toIndex, exclusive. Changes to the returned list do not affect the original list.
func (this *AbstractList[T]) SubList(fromIndex, toIndex int) List[T] {
	if fromIndex < 0 || toIndex > len(this.items) || fromIndex > toIndex {
		panic(NewIndexOutOfBoundsError("sublist indices out of range"))
	}
	return NewAbstractList(this.items[fromIndex:toIndex]...)
}

// Equals returns true if the specified collection is equal to this list.
func (this *AbstractList[T]) Equals(c Collection[T]) bool {
	if c == nil || this.Size() != c.Size() {
		return false
	}
	i := 0
	for item := range c.Seq() {
		if !reflect.DeepEqual(this.items[i], item) {
			return false
		}
		i++
	}
	return true
}
