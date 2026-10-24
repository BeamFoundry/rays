package util

import (
	. "github.com/BeamFoundry/rays/pkg/core"
	. "github.com/BeamFoundry/rays/pkg/util/internal"
	. "github.com/BeamFoundry/rays/pkg/util/stream"
)

// HashSet is a hash table based implementation of the Set interface.
type HashSet[T comparable] struct {
	*AbstractSet[T]
}

var _ Set[int] = (*HashSet[int])(nil)

// NewHashSet creates a new HashSet containing the given elements.
func NewHashSet[T comparable](elements ...T) *HashSet[T] {
	return &HashSet[T]{
		AbstractSet: NewAbstractSet(elements...),
	}
}

// NewHashSetWithCapacity creates an empty HashSet with the specified initial capacity.
func NewHashSetWithCapacity[T comparable](capacity int) *HashSet[T] {
	return &HashSet[T]{
		AbstractSet: NewAbstractSetWithCapacity[T](capacity),
	}
}

// Stream returns a sequential Stream with this set as its source.
func (this *HashSet[T]) Stream() Stream[T] {
	return StreamOfSeq(this.Seq())
}
