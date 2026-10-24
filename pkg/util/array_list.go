package util

import (
	. "github.com/BeamFoundry/rays/pkg/core"
	. "github.com/BeamFoundry/rays/pkg/util/internal"
	. "github.com/BeamFoundry/rays/pkg/util/stream"
)

// ArrayList is a resizable-array implementation of the List interface.
type ArrayList[T any] struct {
	*AbstractList[T]
}

var _ List[int] = (*ArrayList[int])(nil)

// NewArrayList creates a new ArrayList containing the given elements.
func NewArrayList[T any](elements ...T) *ArrayList[T] {
	return &ArrayList[T]{
		AbstractList: NewAbstractList(elements...),
	}
}

// NewArrayListWithCapacity creates an empty ArrayList with the specified initial capacity.
func NewArrayListWithCapacity[T any](capacity int) *ArrayList[T] {
	return &ArrayList[T]{
		AbstractList: NewAbstractListWithCapacity[T](capacity),
	}
}

// Stream returns a sequential Stream with this list as its source.
func (this *ArrayList[T]) Stream() Stream[T] {
	return StreamOfSeq(this.Seq())
}
