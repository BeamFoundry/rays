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
		StreamOf(1, 2, 3).ForEach(func(x int) {
			collected = append(collected, x)
		})
		Expect(collected).To(Equal([]int{1, 2, 3}))

		var ordered []int
		StreamOf(1, 2, 3).ForEachOrdered(func(x int) {
			ordered = append(ordered, x)
		})
		Expect(ordered).To(Equal([]int{1, 2, 3}))
	})

	It("converts stream to Go iter.Seq for range loops", func() {
		s := StreamOf(10, 20, 30)
		var items []int
		for v := range s.Seq() {
			items = append(items, v)
		}
		Expect(items).To(Equal([]int{10, 20, 30}))
	})

	It("counts elements in a stream", func() {
		Expect(StreamEmpty[int]().Count()).To(Equal(int64(0)))
		Expect(StreamOf(1, 2, 3, 4, 5).Count()).To(Equal(int64(5)))
	})

	It("evaluates AnyMatch, AllMatch, and NoneMatch with short-circuiting", func() {
		s := StreamOf(1, 2, 3, 4, 5)
		Expect(s.AnyMatch(func(x int) bool { return x == 3 })).To(BeTrue())

		s2 := StreamOf(1, 2, 3, 4, 5)
		Expect(s2.AnyMatch(func(x int) bool { return x == 10 })).To(BeFalse())

		s3 := StreamOf(2, 4, 6, 8)
		Expect(s3.AllMatch(func(x int) bool { return x%2 == 0 })).To(BeTrue())

		s4 := StreamOf(2, 4, 5, 8)
		Expect(s4.AllMatch(func(x int) bool { return x%2 == 0 })).To(BeFalse())

		s5 := StreamOf(1, 3, 5)
		Expect(s5.NoneMatch(func(x int) bool { return x%2 == 0 })).To(BeTrue())

		s6 := StreamOf(1, 2, 5)
		Expect(s6.NoneMatch(func(x int) bool { return x%2 == 0 })).To(BeFalse())

		// Empty streams
		Expect(StreamEmpty[int]().AnyMatch(func(_ int) bool { return true })).To(BeFalse())
		Expect(StreamEmpty[int]().AllMatch(func(_ int) bool { return false })).To(BeTrue())
		Expect(StreamEmpty[int]().NoneMatch(func(_ int) bool { return true })).To(BeTrue())
	})

	It("finds first and any elements as Optionals", func() {
		first := StreamOf("apple", "banana", "cherry").FindFirst()
		Expect(first.IsPresent()).To(BeTrue())
		val, _ := first.Get()
		Expect(val).To(Equal("apple"))

		emptyFirst := StreamEmpty[string]().FindFirst()
		Expect(emptyFirst.IsEmpty()).To(BeTrue())

		anyElem := StreamOf(42, 43, 44).FindAny()
		Expect(anyElem.IsPresent()).To(BeTrue())
		valInt, _ := anyElem.Get()
		Expect(valInt).To(Equal(42))

		emptyAny := StreamEmpty[int]().FindAny()
		Expect(emptyAny.IsEmpty()).To(BeTrue())
	})

	It("reduces elements with accumulator only", func() {
		sumOpt := StreamOf(1, 2, 3, 4).Reduce(func(a, b int) int {
			return a + b
		})
		Expect(sumOpt.IsPresent()).To(BeTrue())
		val, _ := sumOpt.Get()
		Expect(val).To(Equal(10))

		emptyOpt := StreamEmpty[int]().Reduce(func(a, b int) int {
			return a + b
		})
		Expect(emptyOpt.IsEmpty()).To(BeTrue())
	})

	It("reduces elements with identity", func() {
		sum := StreamOf(1, 2, 3, 4).ReduceWithIdentity(100, func(a, b int) int {
			return a + b
		})
		Expect(sum).To(Equal(110))

		emptySum := StreamEmpty[int]().ReduceWithIdentity(100, func(a, b int) int {
			return a + b
		})
		Expect(emptySum).To(Equal(100))
	})

	It("reduces elements into a different type with ReduceWithInitial using method type parameters", func() {
		// Reduce ints into a formatted string
		str := StreamOf(1, 2, 3).ReduceWithInitial("values:", func(acc string, v int) string {
			return fmt.Sprintf("%s %d", acc, v)
		})
		Expect(str).To(Equal("values: 1 2 3"))

		// Reduce strings into a map of lengths
		counts := StreamOf("a", "bb", "ccc").ReduceWithInitial(make(map[int]int), func(m map[int]int, s string) map[int]int {
			m[len(s)]++
			return m
		})
		Expect(counts).To(Equal(map[int]int{1: 1, 2: 1, 3: 1}))
	})

	It("finds min and max elements with natural ordering and custom comparators", func() {
		minOpt := StreamOf(5, 2, 8, 1, 9).Min()
		Expect(minOpt.IsPresent()).To(BeTrue())
		minVal, _ := minOpt.Get()
		Expect(minVal).To(Equal(1))

		maxOpt := StreamOf(5, 2, 8, 1, 9).Max()
		Expect(maxOpt.IsPresent()).To(BeTrue())
		maxVal, _ := maxOpt.Get()
		Expect(maxVal).To(Equal(9))

		// Custom comparator
		minStr := StreamOf("banana", "apple", "pie").Min(func(a, b string) int {
			return len(a) - len(b)
		})
		Expect(minStr.IsPresent()).To(BeTrue())
		valStr, _ := minStr.Get()
		Expect(valStr).To(Equal("pie"))

		// Empty streams
		Expect(StreamEmpty[int]().Min().IsEmpty()).To(BeTrue())
		Expect(StreamEmpty[int]().Max().IsEmpty()).To(BeTrue())
	})

	It("finds min and max using MinBy and MaxBy with method type parameters", func() {
		people := []person{
			{name: "Alice", age: 25},
			{name: "Bob", age: 40},
			{name: "Charlie", age: 18},
		}

		youngest := StreamOfSlice(people).MinBy(func(p person) int { return p.age })
		Expect(youngest.IsPresent()).To(BeTrue())
		y, _ := youngest.Get()
		Expect(y.name).To(Equal("Charlie"))

		oldest := StreamOfSlice(people).MaxBy(func(p person) int { return p.age })
		Expect(oldest.IsPresent()).To(BeTrue())
		o, _ := oldest.Get()
		Expect(o.name).To(Equal("Bob"))
	})
})
