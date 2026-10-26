package stream_test

import (
	"fmt"
	"iter"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/BeamFoundry/rays/pkg/util/stream"
)

type person struct {
	name string
	age  int
}

// CompareTo orders people by ascending age, breaking ties by name.
func (p person) CompareTo(other person) int {
	if p.age < other.age {
		return -1
	}
	if p.age > other.age {
		return 1
	}
	return strings.Compare(p.name, other.name)
}

type lessItem int

// Less orders lessItem values in ascending numeric order.
func (l lessItem) Less(other lessItem) bool {
	return l < other
}

type reverseComparable int

// CompareTo orders reverseComparable values in descending numeric order.
func (v reverseComparable) CompareTo(other reverseComparable) int {
	if v > other {
		return -1
	}
	if v < other {
		return 1
	}
	return 0
}

type reverseCompare int

// CompareTo supplies a descending comparison for the small integers used in tests.
func (v reverseCompare) CompareTo(other reverseCompare) int {
	return int(other - v)
}

type reverseLess string

// Less orders reverseLess values in descending lexicographic order.
func (v reverseLess) Less(other reverseLess) bool {
	return v > other
}

var _ = Describe("Intermediate Operations", func() {
	It("filters elements", func() {
		res := streamRangeClosed(1, 10).
			Filter(func(x int) bool { return x%2 == 0 }).
			ToSlice()

		Expect(res).To(Equal([]int{2, 4, 6, 8, 10}))
	})

	It("maps elements using method type parameters", func() {
		// Map int to string
		res := streamOf(1, 2, 3).
			Map(func(x int) string {
				return fmt.Sprintf("val:%d", x)
			}).
			ToSlice()

		Expect(res).To(Equal([]string{"val:1", "val:2", "val:3"}))

		// Map string to struct
		people := streamOf("Alice", "Bob").
			Map(func(name string) person {
				return person{name: name, age: len(name) * 10}
			}).
			ToSlice()

		Expect(people).To(Equal([]person{
			{name: "Alice", age: 50},
			{name: "Bob", age: 30},
		}))
	})

	It("flat-maps streams using method type parameters", func() {
		res := streamOf(1, 2, 3).
			FlatMap(func(x int) Stream[int] {
				return streamRangeClosed(1, x)
			}).
			ToSlice()

		Expect(res).To(Equal([]int{1, 1, 2, 1, 2, 3}))
	})

	It("flat-maps slices using method type parameters", func() {
		res := streamOf("foo bar", "hello world").
			FlatMapSlice(func(s string) []string {
				return strings.Split(s, " ")
			}).
			ToSlice()

		Expect(res).To(Equal([]string{"foo", "bar", "hello", "world"}))
	})

	It("flat-maps iterator sequences using method type parameters", func() {
		res := streamOf(1, 2).
			FlatMapSeq(func(x int) iter.Seq[string] {
				return func(yield func(string) bool) {
					yield(fmt.Sprintf("%d-A", x))
					yield(fmt.Sprintf("%d-B", x))
				}
			}).
			ToSlice()

		Expect(res).To(Equal([]string{"1-A", "1-B", "2-A", "2-B"}))
	})

	It("deduplicates elements with Distinct", func() {
		res := streamOf(1, 2, 2, 3, 1, 4, 3, 5).
			Distinct().
			ToSlice()

		Expect(res).To(Equal([]int{1, 2, 3, 4, 5}))

		// Non-comparable types (slices)
		slicesList := streamOf([]int{1, 2}, []int{3}, []int{1, 2}, []int{4}).
			Distinct().
			ToSlice()

		Expect(slicesList).To(Equal([][]int{{1, 2}, {3}, {4}}))

		// Struct with dynamically uncomparable field (interface holding a slice)
		type dynamicContainer struct {
			data any
		}
		containers := streamOf(
			dynamicContainer{data: []int{1, 2}},
			dynamicContainer{data: []int{3}},
			dynamicContainer{data: []int{1, 2}},
			dynamicContainer{data: "comparable"},
			dynamicContainer{data: "comparable"},
		).
			Distinct().
			ToSlice()

		Expect(containers).To(Equal([]dynamicContainer{
			{data: []int{1, 2}},
			{data: []int{3}},
			{data: "comparable"},
		}))

		// Distinct with nil elements
		val := 1
		ptrs := streamOf(nil, &val, nil).Distinct().ToSlice()
		Expect(ptrs).To(Equal([]*int{nil, &val}))
	})

	It("deduplicates elements with DistinctBy using method type parameters", func() {
		people := streamOf(
			person{name: "Alice", age: 25},
			person{name: "Bob", age: 30},
			person{name: "Alice", age: 35},
		).
			DistinctBy(func(p person) string { return p.name }).
			ToSlice()

		Expect(people).To(Equal([]person{
			{name: "Alice", age: 25},
			{name: "Bob", age: 30},
		}))
	})

	It("peeks elements during consumption", func() {
		var peeked []int
		res := streamOf(1, 2, 3).
			Peek(func(x int) { peeked = append(peeked, x) }).
			Map(func(x int) int { return x * 10 }).
			ToSlice()

		Expect(peeked).To(Equal([]int{1, 2, 3}))
		Expect(res).To(Equal([]int{10, 20, 30}))
	})

	It("limits elements lazily", func() {
		// Verify Limit short-circuits an infinite stream
		res := streamIterate(1, func(x int) int { return x + 1 }).
			Filter(func(x int) bool { return x%2 == 0 }).
			Limit(3).
			ToSlice()

		Expect(res).To(Equal([]int{2, 4, 6}))

		Expect(streamOf(1, 2, 3).Limit(0).ToSlice()).To(BeEmpty())
		Expect(streamOf(1, 2, 3).Limit(-5).ToSlice()).To(BeEmpty())
	})

	It("skips elements", func() {
		res := streamRangeClosed(1, 5).
			Skip(2).
			ToSlice()

		Expect(res).To(Equal([]int{3, 4, 5}))
		Expect(streamOf(1, 2).Skip(10).ToSlice()).To(BeEmpty())
		Expect(streamOf(1, 2).Skip(0).ToSlice()).To(Equal([]int{1, 2}))
	})

	It("takes elements while predicate is satisfied", func() {
		res := streamOf(2, 4, 6, 7, 8, 10).
			TakeWhile(func(x int) bool { return x%2 == 0 }).
			ToSlice()

		Expect(res).To(Equal([]int{2, 4, 6}))
	})

	It("drops elements while predicate is satisfied", func() {
		res := streamOf(2, 4, 6, 7, 8, 10).
			DropWhile(func(x int) bool { return x%2 == 0 }).
			ToSlice()

		Expect(res).To(Equal([]int{7, 8, 10}))
	})

	It("sorts elements with natural order", func() {
		Expect(streamOf(5, 3, 1, 4, 2).Sorted().ToSlice()).To(Equal([]int{1, 2, 3, 4, 5}))
		Expect(streamOf("banana", "apple", "cherry").Sorted().ToSlice()).To(Equal([]string{"apple", "banana", "cherry"}))

		// Struct implementing Comparable
		people := streamOf(
			person{name: "Bob", age: 30},
			person{name: "Alice", age: 25},
			person{name: "Charlie", age: 25},
		).Sorted().ToSlice()

		Expect(people).To(Equal([]person{
			{name: "Alice", age: 25},
			{name: "Charlie", age: 25},
			{name: "Bob", age: 30},
		}))
	})

	It("naturally orders named numeric and string types", func() {
		type namedInt int
		type namedInt8 int8
		type namedInt16 int16
		type namedInt32 int32
		type namedInt64 int64
		type namedUint uint
		type namedUint8 uint8
		type namedUint16 uint16
		type namedUint32 uint32
		type namedUint64 uint64
		type namedUintptr uintptr
		type namedFloat32 float32
		type namedFloat64 float64
		type namedString string

		for _, pair := range [][]any{
			{namedInt(3), namedInt(-1)},
			{namedInt8(3), namedInt8(-1)},
			{namedInt16(3), namedInt16(-1)},
			{namedInt32(3), namedInt32(-1)},
			{namedInt64(1 << 62), namedInt64((1 << 62) - 1)},
			{namedUint(3), namedUint(1)},
			{namedUint8(3), namedUint8(1)},
			{namedUint16(3), namedUint16(1)},
			{namedUint32(3), namedUint32(1)},
			{namedUint64(1 << 63), namedUint64((1 << 63) - 1)},
			{namedUintptr(3), namedUintptr(1)},
			{namedFloat32(3.5), namedFloat32(1.2)},
			{namedFloat64(3.5), namedFloat64(1.2)},
			{namedString("banana"), namedString("apple")},
		} {
			Expect(streamOfSlice(pair).Sorted().ToSlice()).To(Equal([]any{pair[1], pair[0]}))
		}
		Expect(streamOf(namedInt(3), namedInt(-1)).Min().OrElse(0)).To(Equal(namedInt(-1)))
		Expect(streamOf(namedString("banana"), namedString("apple")).Max().OrElse("")).To(Equal(namedString("banana")))
	})

	It("prefers CompareTo and Less over the underlying kind", func() {
		Expect(streamOf(reverseCompare(1), reverseCompare(3)).Sorted().ToSlice()).To(Equal([]reverseCompare{3, 1}))
		Expect(streamOf(reverseLess("apple"), reverseLess("banana")).Sorted().ToSlice()).To(Equal([]reverseLess{"banana", "apple"}))
	})

	It("sorts elements with custom comparator or less function", func() {
		res := streamOf(1, 5, 2, 4, 3).
			Sorted(func(a, b int) int { return b - a }).
			ToSlice()

		Expect(res).To(Equal([]int{5, 4, 3, 2, 1}))

		resFunc := streamOf(1, 5, 2, 4, 3).
			SortedFunc(func(a, b int) bool { return a > b }).
			ToSlice()

		Expect(resFunc).To(Equal([]int{5, 4, 3, 2, 1}))
	})

	It("sorts elements using SortedBy and SortedByDesc with method type parameters", func() {
		people := []person{
			{name: "Charlie", age: 35},
			{name: "Alice", age: 25},
			{name: "Bob", age: 30},
		}

		byAge := streamOfSlice(people).
			SortedBy(func(p person) int { return p.age }).
			ToSlice()

		Expect(byAge).To(Equal([]person{
			{name: "Alice", age: 25},
			{name: "Bob", age: 30},
			{name: "Charlie", age: 35},
		}))

		byAgeDesc := streamOfSlice(people).
			SortedByDesc(func(p person) int { return p.age }).
			ToSlice()

		Expect(byAgeDesc).To(Equal([]person{
			{name: "Charlie", age: 35},
			{name: "Bob", age: 30},
			{name: "Alice", age: 25},
		}))
	})

	It("supports Unordered", func() {
		s := streamOf(1, 2, 3).Unordered()
		Expect(s.ToSlice()).To(Equal([]int{1, 2, 3}))
	})

	It("sorts various primitive numeric types naturally", func() {
		Expect(streamOf(int8(3), int8(1)).Sorted().ToSlice()).To(Equal([]int8{1, 3}))
		Expect(streamOf(int16(3), int16(1)).Sorted().ToSlice()).To(Equal([]int16{1, 3}))
		Expect(streamOf(int32(3), int32(1)).Sorted().ToSlice()).To(Equal([]int32{1, 3}))
		Expect(streamOf(int64(3), int64(1)).Sorted().ToSlice()).To(Equal([]int64{1, 3}))
		Expect(streamOf(uint(3), uint(1)).Sorted().ToSlice()).To(Equal([]uint{1, 3}))
		Expect(streamOf(uint8(3), uint8(1)).Sorted().ToSlice()).To(Equal([]uint8{1, 3}))
		Expect(streamOf(uint16(3), uint16(1)).Sorted().ToSlice()).To(Equal([]uint16{1, 3}))
		Expect(streamOf(uint32(3), uint32(1)).Sorted().ToSlice()).To(Equal([]uint32{1, 3}))
		Expect(streamOf(uint64(3), uint64(1)).Sorted().ToSlice()).To(Equal([]uint64{1, 3}))
		Expect(streamOf(uintptr(3), uintptr(1)).Sorted().ToSlice()).To(Equal([]uintptr{1, 3}))
		Expect(streamOf(float32(3.5), float32(1.2)).Sorted().ToSlice()).To(Equal([]float32{1.2, 3.5}))
		Expect(streamOf(float64(3.5), float64(1.2)).Sorted().ToSlice()).To(Equal([]float64{1.2, 3.5}))
	})

	It("naturally orders named numeric and string types in Sorted, Min, and Max", func() {
		type namedInt int
		type namedInt8 int8
		type namedInt16 int16
		type namedInt32 int32
		type namedInt64 int64
		type namedUint uint
		type namedUint8 uint8
		type namedUint16 uint16
		type namedUint32 uint32
		type namedUint64 uint64
		type namedUintptr uintptr
		type namedFloat32 float32
		type namedFloat64 float64
		type namedString string

		cases := []struct{ low, high any }{
			{namedInt(-3), namedInt(2)},
			{namedInt8(-3), namedInt8(2)},
			{namedInt16(-3), namedInt16(2)},
			{namedInt32(-3), namedInt32(2)},
			{namedInt64(-1 << 63), namedInt64(1<<63 - 1)},
			{namedUint(1), namedUint(3)},
			{namedUint8(1), namedUint8(3)},
			{namedUint16(1), namedUint16(3)},
			{namedUint32(1), namedUint32(3)},
			{namedUint64(1), namedUint64(1<<64 - 1)},
			{namedUintptr(1), namedUintptr(3)},
			{namedFloat32(-1.5), namedFloat32(2.5)},
			{namedFloat64(-1.5), namedFloat64(2.5)},
			{namedString("apple"), namedString("banana")},
		}
		for _, tc := range cases {
			By(fmt.Sprintf("ordering %T", tc.low))
			Expect(streamOf(tc.high, tc.low).Sorted().ToSlice()).To(Equal([]any{tc.low, tc.high}))
			minVal, _ := streamOf(tc.high, tc.low).Min().Get()
			maxVal, _ := streamOf(tc.low, tc.high).Max().Get()
			Expect(minVal).To(Equal(tc.low))
			Expect(maxVal).To(Equal(tc.high))
		}
		Expect(streamOf(namedInt(2), namedInt(-3)).Sorted().ToSlice()).To(Equal([]namedInt{-3, 2}))
	})

	It("uses CompareTo and Less before the underlying kind of named types", func() {
		Expect(streamOf(reverseComparable(1), reverseComparable(3), reverseComparable(2)).Sorted().ToSlice()).
			To(Equal([]reverseComparable{3, 2, 1}))
		Expect(streamOf(reverseLess("a"), reverseLess("c"), reverseLess("b")).Sorted().ToSlice()).
			To(Equal([]reverseLess{"c", "b", "a"}))
	})

	It("panics when sorting non-comparable type without comparator", func() {
		type custom struct {
			_ map[string]int
		}
		Expect(func() {
			streamOf(custom{}, custom{}).Sorted().ToSlice()
		}).To(Panic())
	})

	It("handles edge cases in Limit, Skip, and Sorted", func() {
		Expect(streamOf(1, 2, 3).Limit(0).ToSlice()).To(BeEmpty())
		Expect(streamOf(1, 2, 3).Limit(-1).ToSlice()).To(BeEmpty())

		Expect(streamOf(1, 2).Skip(0).ToSlice()).To(Equal([]int{1, 2}))
		Expect(streamOf(1, 2).Skip(-1).ToSlice()).To(Equal([]int{1, 2}))

		Expect(streamEmpty[int]().Sorted().ToSlice()).To(BeEmpty())
		Expect(streamOf(42).Sorted().ToSlice()).To(Equal([]int{42}))

		sortedPeople := streamOf(person{"Bob", 30}, person{"Alice", 25}).Sorted().ToSlice()
		Expect(sortedPeople).To(Equal([]person{{"Alice", 25}, {"Bob", 30}}))

		lessSlice := streamOf(lessItem(5), lessItem(2), lessItem(8)).Sorted().ToSlice()
		Expect(lessSlice).To(Equal([]lessItem{2, 5, 8}))
	})

	It("supports early termination in FlatMap, FlatMapSlice, and FlatMapSeq", func() {
		fm := streamOf(1, 2, 3).FlatMap(func(x int) Stream[int] {
			return streamOf(x, x*10)
		}).Limit(1).ToSlice()
		Expect(fm).To(Equal([]int{1}))

		fms := streamOf(1, 2, 3).FlatMapSlice(func(x int) []int {
			return []int{x, x * 10}
		}).Limit(1).ToSlice()
		Expect(fms).To(Equal([]int{1}))

		fmseq := streamOf(1, 2, 3).FlatMapSeq(func(x int) iter.Seq[int] {
			return func(yield func(int) bool) {
				if !yield(x) {
					return
				}
				yield(x * 10)
			}
		}).Limit(1).ToSlice()
		Expect(fmseq).To(Equal([]int{1}))
	})

	It("supports early termination in Distinct, Sorted, and SortedFunc", func() {
		d := streamOf(1, 2, 3).Distinct().Limit(1).ToSlice()
		Expect(d).To(Equal([]int{1}))

		dNil := streamOf[any](nil, nil).Distinct().Limit(1).ToSlice()
		Expect(dNil).To(Equal([]any{nil}))

		dNonComp := streamOf([]int{1}, []int{2}).Distinct().Limit(1).ToSlice()
		Expect(dNonComp).To(Equal([][]int{{1}}))

		sMulti := streamOf(3, 1, 2).Sorted().Limit(1).ToSlice()
		Expect(sMulti).To(Equal([]int{1}))

		sSingle := streamOf(1).Sorted().FindFirst()
		val, present := sSingle.Get()
		Expect(present).To(BeTrue())
		Expect(val).To(Equal(1))

		sFunc := streamOf(3, 1, 2).SortedFunc(func(a, b int) bool { return a < b }).Limit(1).ToSlice()
		Expect(sFunc).To(Equal([]int{1}))

		// Distinct with multiple nil elements
		dNilAll := streamOf[any](nil, nil).Distinct().ToSlice()
		Expect(dNilAll).To(Equal([]any{nil}))

		// Limit with sequence that continues after yield returns false
		sloppySeq := func(yield func(int) bool) {
			for i := range 5 {
				yield(i)
			}
		}
		Expect(streamOfSeq(sloppySeq).Limit(2).ToSlice()).To(Equal([]int{0, 1}))

		// Sorted with duplicate elements exercising equal comparison branch
		sDup := streamOf(2, 2, 1).Sorted().ToSlice()
		Expect(sDup).To(Equal([]int{1, 2, 2}))

		sFuncDup := streamOf(2, 2, 1).SortedFunc(func(a, b int) bool { return a < b }).ToSlice()
		Expect(sFuncDup).To(Equal([]int{1, 2, 2}))
	})
})
