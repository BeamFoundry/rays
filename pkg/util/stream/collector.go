package stream

import (
	"fmt"
	"strings"

	. "github.com/BeamFoundry/rays/pkg/core"
	. "github.com/BeamFoundry/rays/pkg/lang"
	. "github.com/BeamFoundry/rays/pkg/util/internal"
)

// streamList is a private implementation of List for stream collection.
type streamList[T any] struct {
	*AbstractList[T]
}

func newStreamList[T any](elements ...T) *streamList[T] {
	return &streamList[T]{
		AbstractList: NewAbstractList(elements...),
	}
}

// Stream returns a sequential Stream with this list as its source.
func (this *streamList[T]) Stream() Stream[T] {
	return StreamOfSeq(this.Seq())
}

var _ List[int] = (*streamList[int])(nil)

// streamSet is a private implementation of Set for stream collection.
type streamSet[T comparable] struct {
	*AbstractSet[T]
}

func newStreamSet[T comparable](elements ...T) *streamSet[T] {
	return &streamSet[T]{
		AbstractSet: NewAbstractSet(elements...),
	}
}

// Stream returns a sequential Stream with this set as its source.
func (this *streamSet[T]) Stream() Stream[T] {
	return StreamOfSeq(this.Seq())
}

var _ Set[int] = (*streamSet[int])(nil)

// Collector is a mutable reduction operation that accumulates input elements into a mutable
// result container, optionally transforming the accumulated result into a final representation
// after all input elements have been processed.
//
// T is the type of input elements to the reduction.
// A is the mutable accumulation type of the reduction.
// R is the result type of the reduction.
type Collector[T, A, R any] struct {
	Supplier    func() A
	Accumulator func(A, T) A
	Combiner    func(A, A) A
	Finisher    func(A) R
}

// NewCollector constructs a new Collector with the specified supplier, accumulator, combiner, and finisher.
func NewCollector[T, A, R any](supplier func() A, accumulator func(A, T) A, combiner func(A, A) A, finisher func(A) R) Collector[T, A, R] {
	return Collector[T, A, R]{
		Supplier:    supplier,
		Accumulator: accumulator,
		Combiner:    combiner,
		Finisher:    finisher,
	}
}

// NewSimpleCollector constructs a Collector where the intermediate accumulation type and final result type are identical,
// using an identity finisher.
func NewSimpleCollector[T, R any](supplier func() R, accumulator func(R, T) R, combiner ...func(R, R) R) Collector[T, R, R] {
	var comb func(R, R) R
	if len(combiner) > 0 {
		comb = combiner[0]
	}
	return Collector[T, R, R]{
		Supplier:    supplier,
		Accumulator: accumulator,
		Combiner:    comb,
		Finisher:    func(r R) R { return r },
	}
}

// ToList returns a Collector that accumulates input elements into a List[T].
func ToList[T any]() Collector[T, List[T], List[T]] {
	return Collector[T, List[T], List[T]]{
		Supplier: func() List[T] {
			return newStreamList[T]()
		},
		Accumulator: func(l List[T], item T) List[T] {
			l.Add(item)
			return l
		},
		Combiner: func(a, b List[T]) List[T] {
			a.AddAll(b)
			return a
		},
		Finisher: func(l List[T]) List[T] {
			return l
		},
	}
}

// ToSlice returns a Collector that accumulates input elements into a new slice.
func ToSlice[T any]() Collector[T, []T, []T] {
	return Collector[T, []T, []T]{
		Supplier: func() []T {
			return make([]T, 0)
		},
		Accumulator: func(acc []T, item T) []T {
			return append(acc, item)
		},
		Combiner: func(a, b []T) []T {
			return append(a, b...)
		},
		Finisher: func(acc []T) []T {
			return acc
		},
	}
}

// ToSet returns a Collector that accumulates unique input elements into a Set[T].
func ToSet[T comparable]() Collector[T, Set[T], Set[T]] {
	return Collector[T, Set[T], Set[T]]{
		Supplier: func() Set[T] {
			return newStreamSet[T]()
		},
		Accumulator: func(set Set[T], item T) Set[T] {
			set.Add(item)
			return set
		},
		Combiner: func(a, b Set[T]) Set[T] {
			a.AddAll(b)
			return a
		},
		Finisher: func(set Set[T]) Set[T] {
			return set
		},
	}
}

// ToSetSlice returns a Collector that accumulates unique elements into a slice.
func ToSetSlice[T comparable]() Collector[T, map[T]struct{}, []T] {
	return Collector[T, map[T]struct{}, []T]{
		Supplier: func() map[T]struct{} {
			return make(map[T]struct{})
		},
		Accumulator: func(set map[T]struct{}, item T) map[T]struct{} {
			set[item] = struct{}{}
			return set
		},
		Combiner: func(a, b map[T]struct{}) map[T]struct{} {
			for k := range b {
				a[k] = struct{}{}
			}
			return a
		},
		Finisher: func(set map[T]struct{}) []T {
			res := make([]T, 0, len(set))
			for k := range set {
				res = append(res, k)
			}
			return res
		},
	}
}

type joinAcc struct {
	builder strings.Builder
	first   bool
}

// Joining returns a Collector that concatenates input strings, separated by the specified delimiter,
// with an optional prefix and suffix.
func Joining(delimiter string, prefixAndSuffix ...string) Collector[string, *joinAcc, string] {
	prefix := ""
	suffix := ""
	if len(prefixAndSuffix) > 0 {
		prefix = prefixAndSuffix[0]
	}
	if len(prefixAndSuffix) > 1 {
		suffix = prefixAndSuffix[1]
	}

	return Collector[string, *joinAcc, string]{
		Supplier: func() *joinAcc {
			return &joinAcc{first: true}
		},
		Accumulator: func(acc *joinAcc, s string) *joinAcc {
			if !acc.first {
				//nolint:errcheck // strings.Builder WriteString never returns error
				acc.builder.WriteString(delimiter)
			} else {
				acc.first = false
			}
			//nolint:errcheck // strings.Builder WriteString never returns error
			acc.builder.WriteString(s)
			return acc
		},
		Combiner: func(a, b *joinAcc) *joinAcc {
			if b.first {
				return a
			}
			if a.first {
				return b
			}
			//nolint:errcheck // strings.Builder WriteString never returns error
			a.builder.WriteString(delimiter)
			//nolint:errcheck // strings.Builder WriteString never returns error
			a.builder.WriteString(b.builder.String())
			return a
		},
		Finisher: func(acc *joinAcc) string {
			if acc.first {
				return prefix + suffix
			}
			return prefix + acc.builder.String() + suffix
		},
	}
}

// Counting returns a Collector that counts the number of input elements.
func Counting[T any]() Collector[T, *int64, int64] {
	return Collector[T, *int64, int64]{
		Supplier: func() *int64 {
			var n int64
			return &n
		},
		Accumulator: func(acc *int64, _ T) *int64 {
			*acc++
			return acc
		},
		Combiner: func(a, b *int64) *int64 {
			*a += *b
			return a
		},
		Finisher: func(acc *int64) int64 {
			return *acc
		},
	}
}

// SummingInt returns a Collector that produces the sum of an integer-valued function applied to input elements.
func SummingInt[T any](mapper func(T) int) Collector[T, *int64, int64] {
	return Collector[T, *int64, int64]{
		Supplier: func() *int64 {
			var s int64
			return &s
		},
		Accumulator: func(acc *int64, item T) *int64 {
			*acc += int64(mapper(item))
			return acc
		},
		Combiner: func(a, b *int64) *int64 {
			*a += *b
			return a
		},
		Finisher: func(acc *int64) int64 {
			return *acc
		},
	}
}

// SummingFloat64 returns a Collector that produces the sum of a float64-valued function applied to input elements.
func SummingFloat64[T any](mapper func(T) float64) Collector[T, *float64, float64] {
	return Collector[T, *float64, float64]{
		Supplier: func() *float64 {
			var s float64
			return &s
		},
		Accumulator: func(acc *float64, item T) *float64 {
			*acc += mapper(item)
			return acc
		},
		Combiner: func(a, b *float64) *float64 {
			*a += *b
			return a
		},
		Finisher: func(acc *float64) float64 {
			return *acc
		},
	}
}

type avgAcc struct {
	count int64
	sum   float64
}

// AveragingInt returns a Collector that produces the arithmetic mean of an integer-valued function applied to input elements.
func AveragingInt[T any](mapper func(T) int) Collector[T, *avgAcc, float64] {
	return Collector[T, *avgAcc, float64]{
		Supplier: func() *avgAcc {
			return &avgAcc{}
		},
		Accumulator: func(acc *avgAcc, item T) *avgAcc {
			acc.count++
			acc.sum += float64(mapper(item))
			return acc
		},
		Combiner: func(a, b *avgAcc) *avgAcc {
			a.count += b.count
			a.sum += b.sum
			return a
		},
		Finisher: func(acc *avgAcc) float64 {
			if acc.count == 0 {
				return 0
			}
			return acc.sum / float64(acc.count)
		},
	}
}

// AveragingFloat64 returns a Collector that produces the arithmetic mean of a float64-valued function applied to input elements.
func AveragingFloat64[T any](mapper func(T) float64) Collector[T, *avgAcc, float64] {
	return Collector[T, *avgAcc, float64]{
		Supplier: func() *avgAcc {
			return &avgAcc{}
		},
		Accumulator: func(acc *avgAcc, item T) *avgAcc {
			acc.count++
			acc.sum += mapper(item)
			return acc
		},
		Combiner: func(a, b *avgAcc) *avgAcc {
			a.count += b.count
			a.sum += b.sum
			return a
		},
		Finisher: func(acc *avgAcc) float64 {
			if acc.count == 0 {
				return 0
			}
			return acc.sum / float64(acc.count)
		},
	}
}

// SummarizingInt returns a Collector that computes summary statistics on an integer-valued function applied to input elements.
func SummarizingInt[T any](mapper func(T) int) Collector[T, *IntSummaryStatistics, IntSummaryStatistics] {
	return Collector[T, *IntSummaryStatistics, IntSummaryStatistics]{
		Supplier: NewIntSummaryStatistics,
		Accumulator: func(stats *IntSummaryStatistics, item T) *IntSummaryStatistics {
			stats.Accept(mapper(item))
			return stats
		},
		Combiner: func(a, b *IntSummaryStatistics) *IntSummaryStatistics {
			a.Combine(b)
			return a
		},
		Finisher: func(stats *IntSummaryStatistics) IntSummaryStatistics {
			return *stats
		},
	}
}

// SummarizingFloat64 returns a Collector that computes summary statistics on a float64-valued function applied to input elements.
func SummarizingFloat64[T any](mapper func(T) float64) Collector[T, *Float64SummaryStatistics, Float64SummaryStatistics] {
	return Collector[T, *Float64SummaryStatistics, Float64SummaryStatistics]{
		Supplier: NewFloat64SummaryStatistics,
		Accumulator: func(stats *Float64SummaryStatistics, item T) *Float64SummaryStatistics {
			stats.Accept(mapper(item))
			return stats
		},
		Combiner: func(a, b *Float64SummaryStatistics) *Float64SummaryStatistics {
			a.Combine(b)
			return a
		},
		Finisher: func(stats *Float64SummaryStatistics) Float64SummaryStatistics {
			return *stats
		},
	}
}

// Mapping adapts a Collector accepting elements of type U to one accepting elements of type T
// by applying a mapping function to each input element before accumulation.
func Mapping[T, U, A, R any](mapper func(T) U, downstream Collector[U, A, R]) Collector[T, A, R] {
	return Collector[T, A, R]{
		Supplier: downstream.Supplier,
		Accumulator: func(acc A, item T) A {
			return downstream.Accumulator(acc, mapper(item))
		},
		Combiner: downstream.Combiner,
		Finisher: downstream.Finisher,
	}
}

// Filtering adapts a Collector to one accepting elements of the same type T by applying the predicate
// and accumulating only elements that match.
func Filtering[T, A, R any](predicate func(T) bool, downstream Collector[T, A, R]) Collector[T, A, R] {
	return Collector[T, A, R]{
		Supplier: downstream.Supplier,
		Accumulator: func(acc A, item T) A {
			if predicate(item) {
				return downstream.Accumulator(acc, item)
			}
			return acc
		},
		Combiner: downstream.Combiner,
		Finisher: downstream.Finisher,
	}
}

// FlatMapping adapts a Collector accepting elements of type U to one accepting elements of type T
// by applying a flat-mapping function that maps each input element to a slice of U before accumulation.
func FlatMapping[T, U, A, R any](mapper func(T) []U, downstream Collector[U, A, R]) Collector[T, A, R] {
	return Collector[T, A, R]{
		Supplier: downstream.Supplier,
		Accumulator: func(acc A, item T) A {
			for _, sub := range mapper(item) {
				acc = downstream.Accumulator(acc, sub)
			}
			return acc
		},
		Combiner: downstream.Combiner,
		Finisher: downstream.Finisher,
	}
}

// GroupingBy returns a Collector implementing a "group by" operation on input elements of type T,
// grouping elements according to a classification function, and performing a downstream reduction
// on the elements associated with a given key.
func GroupingBy[T any, K comparable, A, D any](classifier func(T) K, downstream Collector[T, A, D]) Collector[T, map[K]A, map[K]D] {
	var combiner func(m1, m2 map[K]A) map[K]A
	if downstream.Combiner != nil {
		combiner = func(m1, m2 map[K]A) map[K]A {
			for k, v2 := range m2 {
				if v1, exists := m1[k]; exists {
					m1[k] = downstream.Combiner(v1, v2)
				} else {
					m1[k] = v2
				}
			}
			return m1
		}
	}

	return Collector[T, map[K]A, map[K]D]{
		Supplier: func() map[K]A {
			return make(map[K]A)
		},
		Accumulator: func(m map[K]A, item T) map[K]A {
			key := classifier(item)
			acc, exists := m[key]
			if !exists {
				acc = downstream.Supplier()
			}
			m[key] = downstream.Accumulator(acc, item)
			return m
		},
		Combiner: combiner,
		Finisher: func(m map[K]A) map[K]D {
			res := make(map[K]D, len(m))
			for k, v := range m {
				res[k] = downstream.Finisher(v)
			}
			return res
		},
	}
}

// GroupingBySlice returns a Collector that groups elements by key into slices of elements.
func GroupingBySlice[T any, K comparable](classifier func(T) K) Collector[T, map[K][]T, map[K][]T] {
	return GroupingBy(classifier, ToSlice[T]())
}

// PartitioningBy returns a Collector which partitions the input elements according to a Predicate,
// and organizes them into a map[bool]D via a downstream collector.
func PartitioningBy[T, A, D any](predicate func(T) bool, downstream Collector[T, A, D]) Collector[T, map[bool]A, map[bool]D] {
	var combiner func(m1, m2 map[bool]A) map[bool]A
	if downstream.Combiner != nil {
		combiner = func(m1, m2 map[bool]A) map[bool]A {
			m1[false] = downstream.Combiner(m1[false], m2[false])
			m1[true] = downstream.Combiner(m1[true], m2[true])
			return m1
		}
	}

	return Collector[T, map[bool]A, map[bool]D]{
		Supplier: func() map[bool]A {
			m := make(map[bool]A, 2)
			m[false] = downstream.Supplier()
			m[true] = downstream.Supplier()
			return m
		},
		Accumulator: func(m map[bool]A, item T) map[bool]A {
			part := predicate(item)
			m[part] = downstream.Accumulator(m[part], item)
			return m
		},
		Combiner: combiner,
		Finisher: func(m map[bool]A) map[bool]D {
			return map[bool]D{
				false: downstream.Finisher(m[false]),
				true:  downstream.Finisher(m[true]),
			}
		},
	}
}

// PartitioningBySlice returns a Collector which partitions the input elements according to a Predicate
// into two slices: map[true][]T and map[false][]T.
func PartitioningBySlice[T any](predicate func(T) bool) Collector[T, map[bool][]T, map[bool][]T] {
	return PartitioningBy(predicate, ToSlice[T]())
}

// ToMap returns a Collector that accumulates elements into a map whose keys and values are the result
// of applying the provided mapping functions to the input elements.
//
// If duplicate keys are encountered, mergeRule (if provided) is invoked to resolve the collision.
// If no mergeRule is provided, duplicate keys trigger an IllegalStateError.
func ToMap[T any, K comparable, V any](keyMapper func(T) K, valueMapper func(T) V, mergeRule ...func(existing, newVal V) V) Collector[T, map[K]V, map[K]V] {
	var merger func(existing, newVal V) V
	if len(mergeRule) > 0 {
		merger = mergeRule[0]
	}

	return Collector[T, map[K]V, map[K]V]{
		Supplier: func() map[K]V {
			return make(map[K]V)
		},
		Accumulator: func(m map[K]V, item T) map[K]V {
			k := keyMapper(item)
			v := valueMapper(item)
			if existing, exists := m[k]; exists {
				if merger == nil {
					panic(NewError("IllegalStateError", fmt.Sprintf("duplicate key %v", k)))
				}
				m[k] = merger(existing, v)
			} else {
				m[k] = v
			}
			return m
		},
		Combiner: func(m1, m2 map[K]V) map[K]V {
			for k, v2 := range m2 {
				if v1, exists := m1[k]; exists {
					if merger == nil {
						panic(NewError("IllegalStateError", fmt.Sprintf("duplicate key %v", k)))
					}
					m1[k] = merger(v1, v2)
				} else {
					m1[k] = v2
				}
			}
			return m1
		},
		Finisher: func(m map[K]V) map[K]V {
			return m
		},
	}
}

// Reducing returns a Collector which performs a reduction of its input elements under a specified identity
// and associative BinaryOperator.
func Reducing[T any](identity T, op func(T, T) T) Collector[T, *T, T] {
	return Collector[T, *T, T]{
		Supplier: func() *T {
			id := identity
			return &id
		},
		Accumulator: func(acc *T, item T) *T {
			*acc = op(*acc, item)
			return acc
		},
		Combiner: func(a, b *T) *T {
			*a = op(*a, *b)
			return a
		},
		Finisher: func(acc *T) T {
			return *acc
		},
	}
}

// MaxBy returns a Collector that produces the maximal element according to a given comparator,
// described as an Optional[T].
func MaxBy[T any](comparator func(a, b T) int) Collector[T, *Optional[T], Optional[T]] {
	return Collector[T, *Optional[T], Optional[T]]{
		Supplier: func() *Optional[T] {
			opt := OptionalEmpty[T]()
			return &opt
		},
		Accumulator: func(acc *Optional[T], item T) *Optional[T] {
			if acc.IsEmpty() {
				*acc = OptionalOf(item)
			} else {
				curr, _ := acc.Get()
				if comparator(item, curr) > 0 {
					*acc = OptionalOf(item)
				}
			}
			return acc
		},
		Combiner: func(a, b *Optional[T]) *Optional[T] {
			if b.IsEmpty() {
				return a
			}
			if a.IsEmpty() {
				return b
			}
			vA, _ := a.Get()
			vB, _ := b.Get()
			if comparator(vB, vA) > 0 {
				return b
			}
			return a
		},
		Finisher: func(acc *Optional[T]) Optional[T] {
			return *acc
		},
	}
}

// MinBy returns a Collector that produces the minimal element according to a given comparator,
// described as an Optional[T].
func MinBy[T any](comparator func(a, b T) int) Collector[T, *Optional[T], Optional[T]] {
	return Collector[T, *Optional[T], Optional[T]]{
		Supplier: func() *Optional[T] {
			opt := OptionalEmpty[T]()
			return &opt
		},
		Accumulator: func(acc *Optional[T], item T) *Optional[T] {
			if acc.IsEmpty() {
				*acc = OptionalOf(item)
			} else {
				curr, _ := acc.Get()
				if comparator(item, curr) < 0 {
					*acc = OptionalOf(item)
				}
			}
			return acc
		},
		Combiner: func(a, b *Optional[T]) *Optional[T] {
			if b.IsEmpty() {
				return a
			}
			if a.IsEmpty() {
				return b
			}
			vA, _ := a.Get()
			vB, _ := b.Get()
			if comparator(vB, vA) < 0 {
				return b
			}
			return a
		},
		Finisher: func(acc *Optional[T]) Optional[T] {
			return *acc
		},
	}
}
