package stream

import (
	"fmt"
	"iter"
	"reflect"
	"slices"

	. "github.com/BeamFoundry/rays/pkg/lang"
)

// Filter returns a stream consisting of the elements of this stream that match the given predicate.
func (this Stream[T]) Filter(predicate func(T) bool) Stream[T] {
	return this.newIntermediate(func(yield func(T) bool) {
		this.seq(func(item T) bool {
			if predicate(item) {
				return yield(item)
			}
			return true
		})
	})
}

// Map returns a stream consisting of the results of applying the given function to the elements of this stream.
// Note: This method leverages Go 1.27's type parameters on struct methods.
func (this Stream[T]) Map[R any](mapper func(T) R) Stream[R] {
	return this.newIntermediate(func(yield func(R) bool) {
		this.seq(func(item T) bool {
			return yield(mapper(item))
		})
	})
}

// FlatMap returns a stream consisting of the results of replacing each element of this stream with the contents of a mapped stream.
// Each mapped stream is automatically closed after its contents have been placed into this stream.
// Note: This method leverages Go 1.27's type parameters on struct methods.
func (this Stream[T]) FlatMap[R any](mapper func(T) Stream[R]) Stream[R] {
	return this.newIntermediate(func(yield func(R) bool) {
		shouldContinue := true
		this.seq(func(item T) bool {
			subStream := mapper(item)
			defer subStream.Close()

			subStream.seq(func(subItem R) bool {
				if !yield(subItem) {
					shouldContinue = false
					return false
				}
				return true
			})
			return shouldContinue
		})
	})
}

// FlatMapSlice returns a stream consisting of the results of replacing each element of this stream with the contents of a mapped slice.
// Note: This method leverages Go 1.27's type parameters on struct methods.
func (this Stream[T]) FlatMapSlice[R any](mapper func(T) []R) Stream[R] {
	return this.newIntermediate(func(yield func(R) bool) {
		shouldContinue := true
		this.seq(func(item T) bool {
			for _, subItem := range mapper(item) {
				if !yield(subItem) {
					shouldContinue = false
					return false
				}
			}
			return shouldContinue
		})
	})
}

// FlatMapSeq returns a stream consisting of the results of replacing each element of this stream with the contents of a mapped Go iterator sequence.
// Note: This method leverages Go 1.27's type parameters on struct methods.
func (this Stream[T]) FlatMapSeq[R any](mapper func(T) iter.Seq[R]) Stream[R] {
	return this.newIntermediate(func(yield func(R) bool) {
		shouldContinue := true
		this.seq(func(item T) bool {
			subSeq := mapper(item)
			subSeq(func(subItem R) bool {
				if !yield(subItem) {
					shouldContinue = false
					return false
				}
				return true
			})
			return shouldContinue
		})
	})
}

// Distinct returns a stream consisting of the distinct elements of this stream.
// For comparable types, deduplication uses hash set lookup. For non-comparable types (e.g. slices),
// it safely falls back to deep equality checks.
func (this Stream[T]) Distinct() Stream[T] {
	return this.newIntermediate(func(yield func(T) bool) {
		seen := make(map[any]struct{})
		var nonComparableSeen []T

		this.seq(func(item T) bool {
			v := any(item)
			if v == nil {
				if _, ok := seen[nil]; !ok {
					seen[nil] = struct{}{}
					return yield(item)
				}
				return true
			}

			if reflect.ValueOf(v).Comparable() {
				if _, ok := seen[v]; !ok {
					seen[v] = struct{}{}
					return yield(item)
				}
				return true
			}

			// Non-comparable fallback
			for _, prev := range nonComparableSeen {
				if reflect.DeepEqual(prev, item) {
					return true
				}
			}
			nonComparableSeen = append(nonComparableSeen, item)
			return yield(item)
		})
	})
}

// DistinctBy returns a stream consisting of distinct elements according to the key returned by keySelector.
// Note: This method leverages Go 1.27's type parameters on struct methods.
func (this Stream[T]) DistinctBy[K comparable](keySelector func(T) K) Stream[T] {
	return this.newIntermediate(func(yield func(T) bool) {
		seen := make(map[K]struct{})
		this.seq(func(item T) bool {
			key := keySelector(item)
			if _, ok := seen[key]; !ok {
				seen[key] = struct{}{}
				return yield(item)
			}
			return true
		})
	})
}

// Peek returns a stream consisting of the elements of this stream, additionally performing the provided action
// on each element as elements are consumed from the resulting stream.
func (this Stream[T]) Peek(action func(T)) Stream[T] {
	return this.newIntermediate(func(yield func(T) bool) {
		this.seq(func(item T) bool {
			action(item)
			return yield(item)
		})
	})
}

// Limit returns a stream consisting of the elements of this stream, truncated to be no longer than maxSize in length.
func (this Stream[T]) Limit(maxSize int64) Stream[T] {
	if maxSize <= 0 {
		return this.newIntermediate(func(_ func(T) bool) {})
	}
	return this.newIntermediate(func(yield func(T) bool) {
		var count int64
		this.seq(func(item T) bool {
			if count < maxSize {
				count++
				return yield(item) && count < maxSize
			}
			return false
		})
	})
}

// Skip returns a stream consisting of the remaining elements of this stream after discarding the first n elements.
func (this Stream[T]) Skip(n int64) Stream[T] {
	if n <= 0 {
		return this
	}
	return this.newIntermediate(func(yield func(T) bool) {
		var count int64
		this.seq(func(item T) bool {
			if count < n {
				count++
				return true
			}
			return yield(item)
		})
	})
}

// TakeWhile returns a stream consisting of the longest prefix of elements of this stream that match the given predicate.
func (this Stream[T]) TakeWhile(predicate func(T) bool) Stream[T] {
	return this.newIntermediate(func(yield func(T) bool) {
		taking := true
		this.seq(func(item T) bool {
			if taking && predicate(item) {
				return yield(item)
			}
			taking = false
			return false
		})
	})
}

// DropWhile returns a stream consisting of the remaining elements of this stream after dropping the longest prefix
// of elements that match the given predicate.
func (this Stream[T]) DropWhile(predicate func(T) bool) Stream[T] {
	return this.newIntermediate(func(yield func(T) bool) {
		dropping := true
		this.seq(func(item T) bool {
			if dropping {
				if predicate(item) {
					return true
				}
				dropping = false
			}
			return yield(item)
		})
	})
}

// Unordered returns an equivalent stream that is unordered.
func (this Stream[T]) Unordered() Stream[T] {
	return this
}

// Sorted returns a stream consisting of the elements of this stream, sorted according to natural order
// or the provided comparator function (which returns negative if a < b, 0 if a == b, and positive if a > b).
//
// If no comparator is supplied and the elements are not naturally ordered, an IllegalArgumentError is raised.
func (this Stream[T]) Sorted(comparator ...func(a, b T) int) Stream[T] {
	if len(comparator) > 0 {
		comp := comparator[0]
		return this.SortedFunc(func(a, b T) bool {
			return comp(a, b) < 0
		})
	}

	return this.newIntermediate(func(yield func(T) bool) {
		var items []T
		this.seq(func(item T) bool {
			items = append(items, item)
			return true
		})

		if len(items) <= 1 {
			for _, item := range items {
				if !yield(item) {
					return
				}
			}
			return
		}

		// Try natural ordering
		less, err := getNaturalLess(items[0])
		if err != nil {
			panic(err)
		}

		slices.SortStableFunc(items, func(a, b T) int {
			if less(a, b) {
				return -1
			}
			if less(b, a) {
				return 1
			}
			return 0
		})

		for _, item := range items {
			if !yield(item) {
				return
			}
		}
	})
}

// SortedFunc returns a stream consisting of the elements of this stream, sorted according to the provided less function.
func (this Stream[T]) SortedFunc(less func(a, b T) bool) Stream[T] {
	return this.newIntermediate(func(yield func(T) bool) {
		var items []T
		this.seq(func(item T) bool {
			items = append(items, item)
			return true
		})

		if len(items) > 1 {
			slices.SortStableFunc(items, func(a, b T) int {
				if less(a, b) {
					return -1
				}
				if less(b, a) {
					return 1
				}
				return 0
			})
		}

		for _, item := range items {
			if !yield(item) {
				return
			}
		}
	})
}

// SortedBy returns a stream consisting of the elements of this stream, sorted in ascending order
// by the key returned from the keySelector function.
// Note: This method leverages Go 1.27's type parameters on struct methods.
func (this Stream[T]) SortedBy[K OrderedConstraint](keySelector func(T) K) Stream[T] {
	return this.SortedFunc(func(a, b T) bool {
		return keySelector(a) < keySelector(b)
	})
}

// SortedByDesc returns a stream consisting of the elements of this stream, sorted in descending order
// by the key returned from the keySelector function.
// Note: This method leverages Go 1.27's type parameters on struct methods.
func (this Stream[T]) SortedByDesc[K OrderedConstraint](keySelector func(T) K) Stream[T] {
	return this.SortedFunc(func(a, b T) bool {
		return keySelector(a) > keySelector(b)
	})
}

// getNaturalLess determines an ordering function for type T based on standard ordered types
// or Comparable interfaces.
func getNaturalLess[T any](sample T) (func(a, b T) bool, error) {
	switch any(sample).(type) {
	case int:
		return func(a, b T) bool { return any(a).(int) < any(b).(int) }, nil
	case int8:
		return func(a, b T) bool { return any(a).(int8) < any(b).(int8) }, nil
	case int16:
		return func(a, b T) bool { return any(a).(int16) < any(b).(int16) }, nil
	case int32:
		return func(a, b T) bool { return any(a).(int32) < any(b).(int32) }, nil
	case int64:
		return func(a, b T) bool { return any(a).(int64) < any(b).(int64) }, nil
	case uint:
		return func(a, b T) bool { return any(a).(uint) < any(b).(uint) }, nil
	case uint8:
		return func(a, b T) bool { return any(a).(uint8) < any(b).(uint8) }, nil
	case uint16:
		return func(a, b T) bool { return any(a).(uint16) < any(b).(uint16) }, nil
	case uint32:
		return func(a, b T) bool { return any(a).(uint32) < any(b).(uint32) }, nil
	case uint64:
		return func(a, b T) bool { return any(a).(uint64) < any(b).(uint64) }, nil
	case uintptr:
		return func(a, b T) bool { return any(a).(uintptr) < any(b).(uintptr) }, nil
	case float32:
		return func(a, b T) bool { return any(a).(float32) < any(b).(float32) }, nil
	case float64:
		return func(a, b T) bool { return any(a).(float64) < any(b).(float64) }, nil
	case string:
		return func(a, b T) bool { return any(a).(string) < any(b).(string) }, nil
	}

	// Check for CompareTo
	if comp, ok := any(sample).(interface{ CompareTo(T) int }); ok {
		_ = comp
		return func(a, b T) bool {
			return any(a).(interface{ CompareTo(T) int }).CompareTo(b) < 0
		}, nil
	}

	// Check for Less
	if l, ok := any(sample).(interface{ Less(T) bool }); ok {
		_ = l
		return func(a, b T) bool {
			return any(a).(interface{ Less(T) bool }).Less(b)
		}, nil
	}

	// Fall back to the underlying kind for named ordered types.
	//nolint:exhaustive // other kinds are not naturally ordered and return the error below
	switch reflect.ValueOf(sample).Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return func(a, b T) bool { return reflect.ValueOf(a).Int() < reflect.ValueOf(b).Int() }, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return func(a, b T) bool { return reflect.ValueOf(a).Uint() < reflect.ValueOf(b).Uint() }, nil
	case reflect.Float32, reflect.Float64:
		return func(a, b T) bool { return reflect.ValueOf(a).Float() < reflect.ValueOf(b).Float() }, nil
	case reflect.String:
		return func(a, b T) bool { return reflect.ValueOf(a).String() < reflect.ValueOf(b).String() }, nil
	}

	return nil, NewError("IllegalArgumentError", fmt.Sprintf("type %T is not naturally ordered; provide an explicit comparator or less function", sample))
}
