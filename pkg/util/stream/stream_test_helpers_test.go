package stream_test

import (
	"iter"

	. "github.com/BeamFoundry/rays/pkg/core"
	. "github.com/BeamFoundry/rays/pkg/lang"
	. "github.com/BeamFoundry/rays/pkg/util/stream"
)

// streamOf is a test helper constructing a Stream using Stream[T].
func streamOf[T any](elements ...T) Stream[T] {
	return Stream[T]{}.From(elements...)
}

// streamOfSlice is a test helper constructing a Stream using Stream[T].
func streamOfSlice[T any](slice []T) Stream[T] {
	return Stream[T]{}.FromSlice(slice)
}

// streamOfSeq is a test helper constructing a Stream using Stream[T].
func streamOfSeq[T any](seq iter.Seq[T]) Stream[T] {
	return Stream[T]{}.FromSeq(seq)
}

// streamOfCollection is a test helper constructing a Stream using Stream[T].
func streamOfCollection[T any](c Collection[T]) Stream[T] {
	return Stream[T]{}.FromCollection(c)
}

// streamEmpty is a test helper constructing an empty Stream using Stream[T].
func streamEmpty[T any]() Stream[T] {
	return Stream[T]{}.FromEmpty()
}

// streamRange is a test helper constructing a range Stream using Stream[T].
func streamRange[T IntegerConstraint](startInclusive, endExclusive T) Stream[T] {
	return Stream[T]{}.FromRange(startInclusive, endExclusive)
}

// streamRangeClosed is a test helper constructing a closed range Stream using Stream[T].
func streamRangeClosed[T IntegerConstraint](startInclusive, endInclusive T) Stream[T] {
	return Stream[T]{}.FromRangeClosed(startInclusive, endInclusive)
}

// streamIterate is a test helper constructing an iterate Stream using Stream[T].
func streamIterate[T any](seed T, f func(T) T) Stream[T] {
	return Stream[T]{}.FromIterate(seed, f)
}
