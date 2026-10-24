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

func (p person) CompareTo(other person) int {
	if p.age < other.age {
		return -1
	}
	if p.age > other.age {
		return 1
	}
	return strings.Compare(p.name, other.name)
}

var _ = Describe("Intermediate Operations", func() {
	It("filters elements", func() {
		res := StreamRangeClosed(1, 10).
			Filter(func(x int) bool { return x%2 == 0 }).
			ToSlice()

		Expect(res).To(Equal([]int{2, 4, 6, 8, 10}))
	})

	It("maps elements using method type parameters", func() {
		// Map int to string
		res := StreamOf(1, 2, 3).
			Map(func(x int) string {
				return fmt.Sprintf("val:%d", x)
			}).
			ToSlice()

		Expect(res).To(Equal([]string{"val:1", "val:2", "val:3"}))

		// Map string to struct
		people := StreamOf("Alice", "Bob").
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
		res := StreamOf(1, 2, 3).
			FlatMap(func(x int) Stream[int] {
				return StreamRangeClosed(1, x)
			}).
			ToSlice()

		Expect(res).To(Equal([]int{1, 1, 2, 1, 2, 3}))
	})

	It("flat-maps slices using method type parameters", func() {
		res := StreamOf("foo bar", "hello world").
			FlatMapSlice(func(s string) []string {
				return strings.Split(s, " ")
			}).
			ToSlice()

		Expect(res).To(Equal([]string{"foo", "bar", "hello", "world"}))
	})

	It("flat-maps iterator sequences using method type parameters", func() {
		res := StreamOf(1, 2).
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
		res := StreamOf(1, 2, 2, 3, 1, 4, 3, 5).
			Distinct().
			ToSlice()

		Expect(res).To(Equal([]int{1, 2, 3, 4, 5}))

		// Non-comparable types (slices)
		slicesList := StreamOf([]int{1, 2}, []int{3}, []int{1, 2}, []int{4}).
			Distinct().
			ToSlice()

		Expect(slicesList).To(Equal([][]int{{1, 2}, {3}, {4}}))

		// Struct with dynamically uncomparable field (interface holding a slice)
		type dynamicContainer struct {
			data any
		}
		containers := StreamOf(
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
	})

	It("deduplicates elements with DistinctBy using method type parameters", func() {
		people := StreamOf(
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
		res := StreamOf(1, 2, 3).
			Peek(func(x int) { peeked = append(peeked, x) }).
			Map(func(x int) int { return x * 10 }).
			ToSlice()

		Expect(peeked).To(Equal([]int{1, 2, 3}))
		Expect(res).To(Equal([]int{10, 20, 30}))
	})

	It("limits elements lazily", func() {
		// Verify Limit short-circuits an infinite stream
		res := StreamIterate(1, func(x int) int { return x + 1 }).
			Filter(func(x int) bool { return x%2 == 0 }).
			Limit(3).
			ToSlice()

		Expect(res).To(Equal([]int{2, 4, 6}))

		Expect(StreamOf(1, 2, 3).Limit(0).ToSlice()).To(BeEmpty())
		Expect(StreamOf(1, 2, 3).Limit(-5).ToSlice()).To(BeEmpty())
	})

	It("skips elements", func() {
		res := StreamRangeClosed(1, 5).
			Skip(2).
			ToSlice()

		Expect(res).To(Equal([]int{3, 4, 5}))
		Expect(StreamOf(1, 2).Skip(10).ToSlice()).To(BeEmpty())
		Expect(StreamOf(1, 2).Skip(0).ToSlice()).To(Equal([]int{1, 2}))
	})

	It("takes elements while predicate is satisfied", func() {
		res := StreamOf(2, 4, 6, 7, 8, 10).
			TakeWhile(func(x int) bool { return x%2 == 0 }).
			ToSlice()

		Expect(res).To(Equal([]int{2, 4, 6}))
	})

	It("drops elements while predicate is satisfied", func() {
		res := StreamOf(2, 4, 6, 7, 8, 10).
			DropWhile(func(x int) bool { return x%2 == 0 }).
			ToSlice()

		Expect(res).To(Equal([]int{7, 8, 10}))
	})

	It("sorts elements with natural order", func() {
		Expect(StreamOf(5, 3, 1, 4, 2).Sorted().ToSlice()).To(Equal([]int{1, 2, 3, 4, 5}))
		Expect(StreamOf("banana", "apple", "cherry").Sorted().ToSlice()).To(Equal([]string{"apple", "banana", "cherry"}))

		// Struct implementing Comparable
		people := StreamOf(
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

	It("sorts elements with custom comparator or less function", func() {
		res := StreamOf(1, 5, 2, 4, 3).
			Sorted(func(a, b int) int { return b - a }).
			ToSlice()

		Expect(res).To(Equal([]int{5, 4, 3, 2, 1}))

		resFunc := StreamOf(1, 5, 2, 4, 3).
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

		byAge := StreamOfSlice(people).
			SortedBy(func(p person) int { return p.age }).
			ToSlice()

		Expect(byAge).To(Equal([]person{
			{name: "Alice", age: 25},
			{name: "Bob", age: 30},
			{name: "Charlie", age: 35},
		}))

		byAgeDesc := StreamOfSlice(people).
			SortedByDesc(func(p person) int { return p.age }).
			ToSlice()

		Expect(byAgeDesc).To(Equal([]person{
			{name: "Charlie", age: 35},
			{name: "Bob", age: 30},
			{name: "Alice", age: 25},
		}))
	})

	It("supports Unordered", func() {
		s := StreamOf(1, 2, 3).Unordered()
		Expect(s.ToSlice()).To(Equal([]int{1, 2, 3}))
	})

	It("sorts various primitive numeric types naturally", func() {
		Expect(StreamOf(int8(3), int8(1)).Sorted().ToSlice()).To(Equal([]int8{1, 3}))
		Expect(StreamOf(int16(3), int16(1)).Sorted().ToSlice()).To(Equal([]int16{1, 3}))
		Expect(StreamOf(int32(3), int32(1)).Sorted().ToSlice()).To(Equal([]int32{1, 3}))
		Expect(StreamOf(int64(3), int64(1)).Sorted().ToSlice()).To(Equal([]int64{1, 3}))
		Expect(StreamOf(uint(3), uint(1)).Sorted().ToSlice()).To(Equal([]uint{1, 3}))
		Expect(StreamOf(uint8(3), uint8(1)).Sorted().ToSlice()).To(Equal([]uint8{1, 3}))
		Expect(StreamOf(uint16(3), uint16(1)).Sorted().ToSlice()).To(Equal([]uint16{1, 3}))
		Expect(StreamOf(uint32(3), uint32(1)).Sorted().ToSlice()).To(Equal([]uint32{1, 3}))
		Expect(StreamOf(uint64(3), uint64(1)).Sorted().ToSlice()).To(Equal([]uint64{1, 3}))
		Expect(StreamOf(uintptr(3), uintptr(1)).Sorted().ToSlice()).To(Equal([]uintptr{1, 3}))
		Expect(StreamOf(float32(3.5), float32(1.2)).Sorted().ToSlice()).To(Equal([]float32{1.2, 3.5}))
		Expect(StreamOf(float64(3.5), float64(1.2)).Sorted().ToSlice()).To(Equal([]float64{1.2, 3.5}))
	})

	It("panics when sorting non-comparable type without comparator", func() {
		type custom struct {
			_ map[string]int
		}
		Expect(func() {
			StreamOf(custom{}, custom{}).Sorted().ToSlice()
		}).To(Panic())
	})
})
