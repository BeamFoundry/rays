package stream

import (
	"iter"
	"runtime"
	"sync"
	"sync/atomic"

	. "github.com/BeamFoundry/rays/pkg/lang"
)

// ForEach performs an action for each element of this stream.
// If the stream is parallel, the action is executed concurrently across goroutines;
// the order of execution is not guaranteed.
func (this Stream[T]) ForEach(action func(T)) {
	this.markConsumed()
	defer this.Close()

	if this.parallel {
		numWorkers := max(runtime.GOMAXPROCS(0), 2)

		var panicked atomic.Bool
		ch := make(chan T, numWorkers*16)
		var wg sync.WaitGroup
		var panicOnce sync.Once
		var firstPanic any

		for range numWorkers {
			wg.Go(func() {
				defer func() {
					if r := recover(); r != nil {
						panicked.Store(true)
						panicOnce.Do(func() {
							firstPanic = r
						})
						for range ch {
						}
					}
				}()
				for item := range ch {
					if panicked.Load() {
						continue
					}
					action(item)
				}
			})
		}

		func() {
			defer wg.Wait()
			defer close(ch)
			this.seq(func(item T) bool {
				if panicked.Load() {
					return false
				}
				ch <- item
				return !panicked.Load()
			})
		}()

		if firstPanic != nil {
			panic(firstPanic)
		}
		return
	}

	this.seq(func(item T) bool {
		action(item)
		return true
	})
}

// ForEachOrdered performs an action for each element of this stream, in the encounter order of the stream.
func (this Stream[T]) ForEachOrdered(action func(T)) {
	this.markConsumed()
	defer this.Close()

	this.seq(func(item T) bool {
		action(item)
		return true
	})
}

// ToSlice returns a slice containing the elements of this stream in encounter order.
func (this Stream[T]) ToSlice() []T {
	this.markConsumed()
	defer this.Close()

	var result []T
	this.seq(func(item T) bool {
		result = append(result, item)
		return true
	})
	if result == nil {
		return make([]T, 0)
	}
	return result
}

// Seq returns an iter.Seq[T] iterator over the elements of this stream.
// The stream is marked as consumed, and closing is automatically handled when iteration finishes.
func (this Stream[T]) Seq() iter.Seq[T] {
	this.markConsumed()
	return func(yield func(T) bool) {
		defer this.Close()
		this.seq(yield)
	}
}

// Count returns the count of elements in this stream.
func (this Stream[T]) Count() int64 {
	this.markConsumed()
	defer this.Close()

	var count int64
	this.seq(func(_ T) bool {
		count++
		return true
	})
	return count
}

// AnyMatch returns whether any elements of this stream match the provided predicate.
// May not evaluate the predicate on all elements if not necessary for determining the result.
func (this Stream[T]) AnyMatch(predicate func(T) bool) bool {
	this.markConsumed()
	defer this.Close()

	if this.parallel {
		numWorkers := max(runtime.GOMAXPROCS(0), 2)

		var matched atomic.Bool
		var panicked atomic.Bool
		ch := make(chan T, numWorkers*16)
		var wg sync.WaitGroup
		var panicOnce sync.Once
		var firstPanic any
		for range numWorkers {
			wg.Go(func() {
				defer func() {
					if r := recover(); r != nil {
						panicked.Store(true)
						panicOnce.Do(func() {
							firstPanic = r
						})
						for range ch {
						}
					}
				}()
				for item := range ch {
					if matched.Load() || panicked.Load() {
						continue
					}
					if predicate(item) {
						matched.Store(true)
					}
				}
			})
		}

		func() {
			defer wg.Wait()
			defer close(ch)
			this.seq(func(item T) bool {
				if matched.Load() || panicked.Load() {
					return false
				}
				ch <- item
				return !matched.Load() && !panicked.Load()
			})
		}()
		if firstPanic != nil {
			panic(firstPanic)
		}
		return matched.Load()
	}

	found := false
	this.seq(func(item T) bool {
		if predicate(item) {
			found = true
			return false
		}
		return true
	})
	return found
}

// AllMatch returns whether all elements of this stream match the provided predicate.
// May not evaluate the predicate on all elements if not necessary for determining the result.
// If the stream is empty, true is returned and the predicate is not evaluated.
func (this Stream[T]) AllMatch(predicate func(T) bool) bool {
	this.markConsumed()
	defer this.Close()

	if this.parallel {
		numWorkers := max(runtime.GOMAXPROCS(0), 2)

		var failed atomic.Bool
		var panicked atomic.Bool
		ch := make(chan T, numWorkers*16)
		var wg sync.WaitGroup
		var panicOnce sync.Once
		var firstPanic any
		for range numWorkers {
			wg.Go(func() {
				defer func() {
					if r := recover(); r != nil {
						panicked.Store(true)
						panicOnce.Do(func() {
							firstPanic = r
						})
						for range ch {
						}
					}
				}()
				for item := range ch {
					if failed.Load() || panicked.Load() {
						continue
					}
					if !predicate(item) {
						failed.Store(true)
					}
				}
			})
		}

		func() {
			defer wg.Wait()
			defer close(ch)
			this.seq(func(item T) bool {
				if failed.Load() || panicked.Load() {
					return false
				}
				ch <- item
				return !failed.Load() && !panicked.Load()
			})
		}()
		if firstPanic != nil {
			panic(firstPanic)
		}
		return !failed.Load()
	}

	allMatch := true
	this.seq(func(item T) bool {
		if !predicate(item) {
			allMatch = false
			return false
		}
		return true
	})
	return allMatch
}

// NoneMatch returns whether no elements of this stream match the provided predicate.
// May not evaluate the predicate on all elements if not necessary for determining the result.
// If the stream is empty, true is returned and the predicate is not evaluated.
func (this Stream[T]) NoneMatch(predicate func(T) bool) bool {
	return !this.AnyMatch(predicate)
}

// FindFirst returns an Optional describing the first element of this stream in encounter order,
// or an empty Optional if the stream is empty.
func (this Stream[T]) FindFirst() Optional[T] {
	this.markConsumed()
	defer this.Close()

	var first T
	found := false

	this.seq(func(item T) bool {
		first = item
		found = true
		return false
	})

	if found {
		return OptionalOf(first)
	}
	return OptionalEmpty[T]()
}

// FindAny returns an Optional describing some element of the stream, or an empty Optional if the stream is empty.
// In parallel streams, returns whichever element is evaluated first by any goroutine.
func (this Stream[T]) FindAny() Optional[T] {
	if !this.parallel {
		return this.FindFirst()
	}

	this.markConsumed()
	defer this.Close()

	numWorkers := max(runtime.GOMAXPROCS(0), 2)

	var found atomic.Bool
	var valLock sync.Mutex
	var foundVal T

	ch := make(chan T, numWorkers*16)
	var wg sync.WaitGroup
	for range numWorkers {
		wg.Go(func() {
			for item := range ch {
				if found.CompareAndSwap(false, true) {
					valLock.Lock()
					foundVal = item
					valLock.Unlock()
				}
			}
		})
	}

	func() {
		defer wg.Wait()
		defer close(ch)
		this.seq(func(item T) bool {
			if found.Load() {
				return false
			}
			ch <- item
			return !found.Load()
		})
	}()

	if found.Load() {
		valLock.Lock()
		res := foundVal
		valLock.Unlock()
		return OptionalOf(res)
	}
	return OptionalEmpty[T]()
}

// Reduce performs a reduction on the elements of this stream, using an associative accumulation function,
// and returns an Optional describing the reduced value, if any.
func (this Stream[T]) Reduce(accumulator func(a, b T) T) Optional[T] {
	this.markConsumed()
	defer this.Close()

	var result T
	found := false

	this.seq(func(item T) bool {
		if !found {
			result = item
			found = true
		} else {
			result = accumulator(result, item)
		}
		return true
	})

	if found {
		return OptionalOf(result)
	}
	return OptionalEmpty[T]()
}

// ReduceWithIdentity performs a reduction on the elements of this stream, using the provided identity value
// and an associative accumulation function, and returns the reduced value.
func (this Stream[T]) ReduceWithIdentity(identity T, accumulator func(a, b T) T) T {
	this.markConsumed()
	defer this.Close()

	result := identity
	this.seq(func(item T) bool {
		result = accumulator(result, item)
		return true
	})
	return result
}

// ReduceWithInitial performs a reduction on the elements of this stream, starting from an initial value
// of type R and accumulating elements of type T into R.
// Note: This method leverages Go 1.27's type parameters on struct methods.
func (this Stream[T]) ReduceWithInitial[R any](identity R, accumulator func(R, T) R) R {
	this.markConsumed()
	defer this.Close()

	result := identity
	this.seq(func(item T) bool {
		result = accumulator(result, item)
		return true
	})
	return result
}

// Collect performs a mutable reduction operation on the elements of this stream using a Collector.
// If the stream is parallel and the collector has a Combiner, collection will be performed concurrently across goroutines.
// Note: This method leverages Go 1.27's type parameters on struct methods.
func (this Stream[T]) Collect[R, A any](collector Collector[T, A, R]) R {
	this.markConsumed()
	defer this.Close()

	if this.parallel && collector.Combiner != nil {
		numWorkers := max(runtime.GOMAXPROCS(0), 2)

		type batch struct {
			items []T
			index int
		}
		batchChan := make(chan batch, numWorkers*2)

		results := make(map[int]A)
		var resultsMu sync.Mutex
		var panicked atomic.Bool
		var wg sync.WaitGroup
		var panicOnce sync.Once
		var firstPanic any
		for range numWorkers {
			wg.Go(func() {
				defer func() {
					if r := recover(); r != nil {
						panicked.Store(true)
						panicOnce.Do(func() {
							firstPanic = r
						})
						for range batchChan {
						}
					}
				}()
				for b := range batchChan {
					if panicked.Load() {
						continue
					}
					acc := collector.Supplier()
					for _, item := range b.items {
						acc = collector.Accumulator(acc, item)
					}
					resultsMu.Lock()
					results[b.index] = acc
					resultsMu.Unlock()
				}
			})
		}

		batchCount := 0
		func() {
			defer wg.Wait()
			defer close(batchChan)
			currBatch := make([]T, 0, 64)
			this.seq(func(item T) bool {
				if panicked.Load() {
					return false
				}
				currBatch = append(currBatch, item)
				if len(currBatch) == cap(currBatch) {
					batchChan <- batch{index: batchCount, items: currBatch}
					batchCount++
					currBatch = make([]T, 0, 64)
				}
				return !panicked.Load()
			})
			if !panicked.Load() && len(currBatch) > 0 {
				batchChan <- batch{index: batchCount, items: currBatch}
				batchCount++
			}
		}()
		if firstPanic != nil {
			panic(firstPanic)
		}

		finalAcc := collector.Supplier()
		for i := range batchCount {
			if i == 0 {
				finalAcc = results[i]
			} else {
				finalAcc = collector.Combiner(finalAcc, results[i])
			}
		}
		return collector.Finisher(finalAcc)
	}

	acc := collector.Supplier()
	this.seq(func(item T) bool {
		acc = collector.Accumulator(acc, item)
		return true
	})
	return collector.Finisher(acc)
}

// CollectWith is a convenience method that accumulates elements into a result container using the provided supplier,
// accumulator, and optional combiner functions.
// Note: This method leverages Go 1.27's type parameters on struct methods.
func (this Stream[T]) CollectWith[R any](supplier func() R, accumulator func(R, T) R, combiner ...func(R, R) R) R {
	return this.Collect(NewSimpleCollector(supplier, accumulator, combiner...))
}

// Min returns the minimum element of this stream according to the provided comparator
// (or natural ordering if no comparator is passed).
func (this Stream[T]) Min(comparator ...func(a, b T) int) Optional[T] {
	if len(comparator) > 0 {
		comp := comparator[0]
		return this.Reduce(func(a, b T) T {
			if comp(a, b) <= 0 {
				return a
			}
			return b
		})
	}

	// Natural ordering
	var minVal T
	found := false
	var less func(a, b T) bool

	this.markConsumed()
	defer this.Close()

	this.seq(func(item T) bool {
		if !found {
			var err error
			less, err = getNaturalLess(item)
			if err != nil {
				panic(err)
			}
			minVal = item
			found = true
		} else if less(item, minVal) {
			minVal = item
		}
		return true
	})

	if found {
		return OptionalOf(minVal)
	}
	return OptionalEmpty[T]()
}

// Max returns the maximum element of this stream according to the provided comparator
// (or natural ordering if no comparator is passed).
func (this Stream[T]) Max(comparator ...func(a, b T) int) Optional[T] {
	if len(comparator) > 0 {
		comp := comparator[0]
		return this.Reduce(func(a, b T) T {
			if comp(a, b) >= 0 {
				return a
			}
			return b
		})
	}

	// Natural ordering
	var maxVal T
	found := false
	var less func(a, b T) bool

	this.markConsumed()
	defer this.Close()

	this.seq(func(item T) bool {
		if !found {
			var err error
			less, err = getNaturalLess(item)
			if err != nil {
				panic(err)
			}
			maxVal = item
			found = true
		} else if less(maxVal, item) {
			maxVal = item
		}
		return true
	})

	if found {
		return OptionalOf(maxVal)
	}
	return OptionalEmpty[T]()
}

// MinBy returns the minimum element of this stream according to the key returned by keySelector.
// Note: This method leverages Go 1.27's type parameters on struct methods.
func (this Stream[T]) MinBy[K OrderedConstraint](keySelector func(T) K) Optional[T] {
	return this.Reduce(func(a, b T) T {
		if keySelector(a) <= keySelector(b) {
			return a
		}
		return b
	})
}

// MaxBy returns the maximum element of this stream according to the key returned by keySelector.
// Note: This method leverages Go 1.27's type parameters on struct methods.
func (this Stream[T]) MaxBy[K OrderedConstraint](keySelector func(T) K) Optional[T] {
	return this.Reduce(func(a, b T) T {
		if keySelector(a) >= keySelector(b) {
			return a
		}
		return b
	})
}
