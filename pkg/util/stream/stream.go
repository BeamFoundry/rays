// Package stream provides a lazy, thread-safe, expressive stream processing framework
// inspired by Java 8 streams and backed by Go iterators (iter.Seq).
//
// The package is designed to be cleanly dot-imported:
//
//	import . "github.com/BeamFoundry/rays/pkg/util/stream"
package stream

import (
	"iter"
	"sync"
	"sync/atomic"

	. "github.com/BeamFoundry/rays/pkg/core"
	. "github.com/BeamFoundry/rays/pkg/lang"
)

// Stream represents a lazy, potentially parallel sequence of elements supporting aggregate operations.
// Stream operations are divided into intermediate operations (which return a new Stream) and terminal
// operations (which consume the stream and produce a result or a side-effect).
//
// Streams are single-use: once a terminal operation has been executed, or an intermediate operation
// has linked this stream into a new pipeline stage, the stream is considered consumed and cannot be reused.
type Stream[T any] struct {
	seq           iter.Seq[T]
	consumed      *atomic.Bool
	closed        *atomic.Bool
	mu            *sync.Mutex
	closeHandlers []func()
	parallel      bool
}

// newStream creates a new unconsumed Stream with default state.
func newStream[T any](seq iter.Seq[T]) Stream[T] {
	return Stream[T]{
		seq:           seq,
		closeHandlers: make([]func(), 0),
		consumed:      &atomic.Bool{},
		closed:        &atomic.Bool{},
		mu:            &sync.Mutex{},
		parallel:      false,
	}
}

// markConsumed validates that the stream has not already been operated upon or closed,
// and marks it as consumed. Panics with an IllegalStateError if already consumed.
func (this Stream[T]) markConsumed() {
	if this.consumed != nil && !this.consumed.CompareAndSwap(false, true) {
		panic(NewError("IllegalStateError", "stream has already been operated upon or closed"))
	}
}

// newIntermediate creates a downstream Stream stage inheriting the parallel mode and close handlers
// from the parent stream, while marking the parent stream as consumed.
func newIntermediate[T, R any](parent Stream[T], seq iter.Seq[R]) Stream[R] {
	parent.markConsumed()

	parent.mu.Lock()
	handlers := make([]func(), len(parent.closeHandlers))
	copy(handlers, parent.closeHandlers)
	parent.mu.Unlock()

	return Stream[R]{
		seq:           seq,
		closeHandlers: handlers,
		parallel:      parent.parallel,
		consumed:      &atomic.Bool{},
		closed:        &atomic.Bool{},
		mu:            &sync.Mutex{},
	}
}

// StreamEmpty returns an empty sequential Stream.
func StreamEmpty[T any]() Stream[T] {
	return newStream(func(_ func(T) bool) {})
}

// StreamOf returns a sequential Stream containing the specified elements.
func StreamOf[T any](elements ...T) Stream[T] {
	return StreamOfSlice(elements)
}

// StreamOfSlice returns a sequential Stream whose source is the specified slice.
func StreamOfSlice[T any](slice []T) Stream[T] {
	return newStream(func(yield func(T) bool) {
		for _, item := range slice {
			if !yield(item) {
				return
			}
		}
	})
}

// StreamOfSeq returns a sequential Stream whose source is the given Go iterator sequence (iter.Seq[T]).
func StreamOfSeq[T any](seq iter.Seq[T]) Stream[T] {
	return newStream(seq)
}

// StreamOfCollection returns a sequential Stream whose source is the specified collection.
func StreamOfCollection[T any](c Collection[T]) Stream[T] {
	if c == nil {
		return StreamEmpty[T]()
	}
	return StreamOfSeq(c.Seq())
}

// StreamOfChan returns a sequential Stream whose elements are read from the specified channel
// until the channel is closed.
func StreamOfChan[T any](ch <-chan T) Stream[T] {
	return newStream(func(yield func(T) bool) {
		for item := range ch {
			if !yield(item) {
				return
			}
		}
	})
}

// StreamGenerate returns an infinite sequential unordered Stream where each element is generated
// by the provided supplier function.
func StreamGenerate[T any](supplier func() T) Stream[T] {
	return newStream(func(yield func(T) bool) {
		for {
			if !yield(supplier()) {
				return
			}
		}
	})
}

// StreamIterate returns an infinite sequential ordered Stream produced by iterative application of a function
// to an initial element seed: seed, f(seed), f(f(seed)), ...
func StreamIterate[T any](seed T, f func(T) T) Stream[T] {
	return newStream(func(yield func(T) bool) {
		current := seed
		for {
			if !yield(current) {
				return
			}
			current = f(current)
		}
	})
}

// StreamIterateWithPredicate returns a sequential ordered Stream produced by iterative application of a function
// to an initial element seed, conditioned on satisfying the specified hasNext predicate.
func StreamIterateWithPredicate[T any](seed T, hasNext func(T) bool, f func(T) T) Stream[T] {
	return newStream(func(yield func(T) bool) {
		for current := seed; hasNext(current); current = f(current) {
			if !yield(current) {
				return
			}
		}
	})
}

// StreamRange returns a sequential ordered Stream from startInclusive (inclusive) to endExclusive (exclusive) by an incremental step of 1.
func StreamRange[T IntegerConstraint](startInclusive, endExclusive T) Stream[T] {
	if startInclusive >= endExclusive {
		return StreamEmpty[T]()
	}
	return newStream(func(yield func(T) bool) {
		for i := startInclusive; i < endExclusive; i++ {
			if !yield(i) {
				return
			}
		}
	})
}

// StreamRangeClosed returns a sequential ordered Stream from startInclusive (inclusive) to endInclusive (inclusive) by an incremental step of 1.
func StreamRangeClosed[T IntegerConstraint](startInclusive, endInclusive T) Stream[T] {
	if startInclusive > endInclusive {
		return StreamEmpty[T]()
	}
	return newStream(func(yield func(T) bool) {
		for i := startInclusive; i <= endInclusive; i++ {
			if !yield(i) {
				return
			}
			if i == endInclusive {
				break
			}
		}
	})
}

// StreamConcat creates a lazily concatenated Stream whose elements are all the elements of the first stream
// followed by all the elements of the second stream.
func StreamConcat[T any](a, b Stream[T]) Stream[T] {
	a.markConsumed()
	b.markConsumed()

	isParallel := a.parallel || b.parallel

	var closeHandlers []func()
	if a.mu != nil {
		a.mu.Lock()
		closeHandlers = append(closeHandlers, a.closeHandlers...)
		a.mu.Unlock()
	}
	if b.mu != nil {
		b.mu.Lock()
		closeHandlers = append(closeHandlers, b.closeHandlers...)
		b.mu.Unlock()
	}

	res := newStream(func(yield func(T) bool) {
		shouldContinue := true
		a.seq(func(item T) bool {
			if !yield(item) {
				shouldContinue = false
				return false
			}
			return true
		})
		if !shouldContinue {
			return
		}
		b.seq(func(item T) bool {
			return yield(item)
		})
	})
	res.parallel = isParallel
	res.closeHandlers = closeHandlers
	return res
}

// Parallel returns an equivalent stream that is parallel.
func (this Stream[T]) Parallel() Stream[T] {
	this.parallel = true
	return this
}

// Sequential returns an equivalent stream that is sequential.
func (this Stream[T]) Sequential() Stream[T] {
	this.parallel = false
	return this
}

// IsParallel returns whether this stream would execute terminal operations in parallel.
func (this Stream[T]) IsParallel() bool {
	return this.parallel
}

// OnClose returns an equivalent stream with an additional close handler.
// Close handlers are run when the Close() method is called on the stream.
func (this Stream[T]) OnClose(handler func()) Stream[T] {
	if handler != nil && this.mu != nil {
		this.mu.Lock()
		this.closeHandlers = append(this.closeHandlers, handler)
		this.mu.Unlock()
	}
	return this
}

// Close closes this stream, executing all registered close handlers.
// Calling Close is idempotent: handlers are executed at most once.
func (this Stream[T]) Close() {
	if this.consumed != nil {
		this.consumed.Store(true)
	}
	if this.closed != nil && this.closed.CompareAndSwap(false, true) {
		if this.mu != nil {
			this.mu.Lock()
			handlers := this.closeHandlers
			this.mu.Unlock()
			for _, h := range handlers {
				if h != nil {
					h()
				}
			}
		}
	}
}
