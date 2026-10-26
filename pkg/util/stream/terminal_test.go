package stream_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/BeamFoundry/rays/pkg/util/stream"
)

var _ = Describe("Terminal Operations", func() {
	It("executes ForEach and ForEachOrdered", func() {
		var collected []int
		streamOf(1, 2, 3).ForEach(func(x int) {
			collected = append(collected, x)
		})
		Expect(collected).To(Equal([]int{1, 2, 3}))

		var ordered []int
		streamOf(1, 2, 3).ForEachOrdered(func(x int) {
			ordered = append(ordered, x)
		})
		Expect(ordered).To(Equal([]int{1, 2, 3}))
	})

	It("converts stream to Go iter.Seq for range loops", func() {
		s := streamOf(10, 20, 30)
		var items []int
		for v := range s.Seq() {
			items = append(items, v)
		}
		Expect(items).To(Equal([]int{10, 20, 30}))
	})

	It("counts elements in a stream", func() {
		Expect(Stream[int]{}.FromEmpty().Count()).To(Equal(int64(0)))
		Expect(streamOf(1, 2, 3, 4, 5).Count()).To(Equal(int64(5)))
	})

	It("evaluates AnyMatch, AllMatch, and NoneMatch with short-circuiting", func() {
		s := streamOf(1, 2, 3, 4, 5)
		Expect(s.AnyMatch(func(x int) bool { return x == 3 })).To(BeTrue())

		s2 := streamOf(1, 2, 3, 4, 5)
		Expect(s2.AnyMatch(func(x int) bool { return x == 10 })).To(BeFalse())

		s3 := streamOf(2, 4, 6, 8)
		Expect(s3.AllMatch(func(x int) bool { return x%2 == 0 })).To(BeTrue())

		s4 := streamOf(2, 4, 5, 8)
		Expect(s4.AllMatch(func(x int) bool { return x%2 == 0 })).To(BeFalse())

		s5 := streamOf(1, 3, 5)
		Expect(s5.NoneMatch(func(x int) bool { return x%2 == 0 })).To(BeTrue())

		s6 := streamOf(1, 2, 5)
		Expect(s6.NoneMatch(func(x int) bool { return x%2 == 0 })).To(BeFalse())

		// Empty streams
		Expect(streamEmpty[int]().AnyMatch(func(_ int) bool { return true })).To(BeFalse())
		Expect(streamEmpty[int]().AllMatch(func(_ int) bool { return false })).To(BeTrue())
		Expect(streamEmpty[int]().NoneMatch(func(_ int) bool { return true })).To(BeTrue())

		// Parallel match operations
		Expect(streamRangeClosed(1, 1000).Parallel().AnyMatch(func(x int) bool { return x == 1 })).To(BeTrue())
		Expect(streamRangeClosed(1, 1000).Parallel().AllMatch(func(x int) bool { return x < 5 })).To(BeFalse())
		Expect(streamRangeClosed(1, 1000).Parallel().FindAny().IsPresent()).To(BeTrue())
	})

	It("finds first and any elements as Optionals", func() {
		first := streamOf("apple", "banana", "cherry").FindFirst()
		Expect(first.IsPresent()).To(BeTrue())
		val, _ := first.Get()
		Expect(val).To(Equal("apple"))

		emptyFirst := streamEmpty[string]().FindFirst()
		Expect(emptyFirst.IsEmpty()).To(BeTrue())

		anyElem := streamOf(42, 43, 44).FindAny()
		Expect(anyElem.IsPresent()).To(BeTrue())
		valInt, _ := anyElem.Get()
		Expect(valInt).To(Equal(42))

		emptyAny := streamEmpty[int]().FindAny()
		Expect(emptyAny.IsEmpty()).To(BeTrue())
	})

	It("reduces elements with accumulator only", func() {
		sumOpt := streamOf(1, 2, 3, 4).Reduce(func(a, b int) int {
			return a + b
		})
		Expect(sumOpt.IsPresent()).To(BeTrue())
		val, _ := sumOpt.Get()
		Expect(val).To(Equal(10))

		emptyOpt := streamEmpty[int]().Reduce(func(a, b int) int {
			return a + b
		})
		Expect(emptyOpt.IsEmpty()).To(BeTrue())
	})

	It("reduces elements with identity", func() {
		sum := streamOf(1, 2, 3, 4).ReduceWithIdentity(100, func(a, b int) int {
			return a + b
		})
		Expect(sum).To(Equal(110))

		emptySum := streamEmpty[int]().ReduceWithIdentity(100, func(a, b int) int {
			return a + b
		})
		Expect(emptySum).To(Equal(100))
	})

	It("reduces elements into a different type with ReduceWithInitial using method type parameters", func() {
		// Reduce ints into a formatted string
		str := streamOf(1, 2, 3).ReduceWithInitial("values:", func(acc string, v int) string {
			return fmt.Sprintf("%s %d", acc, v)
		})
		Expect(str).To(Equal("values: 1 2 3"))

		// Reduce strings into a map of lengths
		counts := streamOf("a", "bb", "ccc").ReduceWithInitial(make(map[int]int), func(m map[int]int, s string) map[int]int {
			m[len(s)]++
			return m
		})
		Expect(counts).To(Equal(map[int]int{1: 1, 2: 1, 3: 1}))
	})

	It("finds min and max elements with natural ordering and custom comparators", func() {
		minOpt := streamOf(5, 2, 8, 1, 9).Min()
		Expect(minOpt.IsPresent()).To(BeTrue())
		minVal, _ := minOpt.Get()
		Expect(minVal).To(Equal(1))

		maxOpt := streamOf(5, 2, 8, 1, 9).Max()
		Expect(maxOpt.IsPresent()).To(BeTrue())
		maxVal, _ := maxOpt.Get()
		Expect(maxVal).To(Equal(9))

		// Custom comparator
		minStr := streamOf("banana", "apple", "pie").Min(func(a, b string) int {
			return len(a) - len(b)
		})
		Expect(minStr.IsPresent()).To(BeTrue())
		valStr, _ := minStr.Get()
		Expect(valStr).To(Equal("pie"))

		maxStr := streamOf("banana", "apple", "pie").Max(func(a, b string) int {
			return len(a) - len(b)
		})
		Expect(maxStr.IsPresent()).To(BeTrue())
		valMaxStr, _ := maxStr.Get()
		Expect(valMaxStr).To(Equal("banana"))

		minStr2 := streamOf("pie", "apple", "banana").Min(func(a, b string) int {
			return len(a) - len(b)
		})
		Expect(minStr2.IsPresent()).To(BeTrue())
		v2, _ := minStr2.Get()
		Expect(v2).To(Equal("pie"))

		maxStr2 := streamOf("apple", "banana").Max(func(a, b string) int {
			return len(a) - len(b)
		})
		Expect(maxStr2.IsPresent()).To(BeTrue())
		vMax2, _ := maxStr2.Get()
		Expect(vMax2).To(Equal("banana"))

		// Empty streams
		Expect(streamEmpty[int]().Min().IsEmpty()).To(BeTrue())
		Expect(streamEmpty[int]().Max().IsEmpty()).To(BeTrue())
		Expect(streamEmpty[int]().Min(func(a, b int) int { return a - b }).IsEmpty()).To(BeTrue())
		Expect(streamEmpty[int]().Max(func(a, b int) int { return a - b }).IsEmpty()).To(BeTrue())

		// Unorderable types panic on natural Min/Max
		Expect(func() { streamOf([]int{1}, []int{2}).Min() }).To(Panic())
		Expect(func() { streamOf([]int{1}, []int{2}).Max() }).To(Panic())

		// Parallel FindAny on empty stream
		Expect(streamEmpty[int]().Parallel().FindAny().IsEmpty()).To(BeTrue())
	})

	It("finds min and max using MinBy and MaxBy with method type parameters", func() {
		people := []person{
			{name: "Alice", age: 25},
			{name: "Bob", age: 40},
			{name: "Charlie", age: 18},
		}

		youngest := streamOfSlice(people).MinBy(func(p person) int { return p.age })
		Expect(youngest.IsPresent()).To(BeTrue())
		y, _ := youngest.Get()
		Expect(y.name).To(Equal("Charlie"))

		oldest := streamOfSlice(people).MaxBy(func(p person) int { return p.age })
		Expect(oldest.IsPresent()).To(BeTrue())
		o, _ := oldest.Get()
		Expect(o.name).To(Equal("Bob"))
	})
})
