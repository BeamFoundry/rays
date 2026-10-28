package internal

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"iter"

	. "github.com/BeamFoundry/rays/internal/utils"
	. "github.com/BeamFoundry/rays/pkg/core"
)

// AbstractSet provides a skeletal implementation of the Set interface,
// backed by a Go map.
type AbstractSet[T comparable] struct {
	elements map[T]struct{}
}

var (
	_ Set[int]         = (*AbstractSet[int])(nil)
	_ json.Marshaler   = (*AbstractSet[int])(nil)
	_ json.Unmarshaler = (*AbstractSet[int])(nil)
	_ driver.Valuer    = (*AbstractSet[int])(nil)
	_ sql.Scanner      = (*AbstractSet[int])(nil)
)

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
	if IsNil(c) {
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
	if this.elements == nil {
		this.elements = make(map[T]struct{})
	}
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
	if IsNil(c) {
		return false
	}
	if this.elements == nil {
		this.elements = make(map[T]struct{})
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
	if IsNil(c) {
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
	if IsNil(c) {
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
	if IsNil(c) {
		return false
	}
	other, ok := c.(Set[T])
	if !ok || this.Size() != other.Size() {
		return false
	}
	if _, isList := c.(List[T]); isList {
		return false
	}
	for item := range this.elements {
		if !other.Contains(item) {
			return false
		}
	}
	return true
}

// MarshalJSON implements json.Marshaler.
func (this *AbstractSet[T]) MarshalJSON() ([]byte, error) {
	if this == nil || len(this.elements) == 0 {
		return []byte("[]"), nil
	}
	//nolint:wrapcheck // Will return the error as is.
	return json.Marshal(this.ToSlice())
}

// UnmarshalJSON implements json.Unmarshaler.
func (this *AbstractSet[T]) UnmarshalJSON(data []byte) error {
	if this.elements == nil {
		this.elements = make(map[T]struct{})
	}
	if string(data) == "null" {
		clear(this.elements)
		return nil
	}
	var items []T
	//nolint:wrapcheck // Will return the error as is.
	if err := json.Unmarshal(data, &items); err != nil {
		return err
	}
	this.elements = make(map[T]struct{}, len(items))
	for _, item := range items {
		this.elements[item] = struct{}{}
	}
	return nil
}

// Value implements driver.Valuer.
//
//nolint:nilnil // driver.Valuer returns nil, nil to represent SQL NULL.
func (this *AbstractSet[T]) Value() (driver.Value, error) {
	if this == nil {
		return nil, nil
	}
	b, err := this.MarshalJSON()
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

// Scan implements sql.Scanner.
func (this *AbstractSet[T]) Scan(src any) error {
	if this.elements == nil {
		this.elements = make(map[T]struct{})
	}
	if src == nil {
		clear(this.elements)
		return nil
	}
	var data []byte
	switch v := src.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("%w: cannot scan %T into AbstractSet: expected []byte or string", ErrUnsupportedScanType, src)
	}
	if len(data) == 0 {
		clear(this.elements)
		return nil
	}
	return this.UnmarshalJSON(data)
}
