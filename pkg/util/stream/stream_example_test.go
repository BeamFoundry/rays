//nolint:errcheck // Not printing error handling
package stream_test

import (
	"fmt"

	. "github.com/BeamFoundry/rays/pkg/util/stream"
)

func Example() {
	// A pipeline filtering even numbers, doubling them, and collecting to a slice.
	result := StreamOf(1, 2, 3, 4, 5, 6).
		Filter(func(n int) bool { return n%2 == 0 }).
		Map(func(n int) int { return n * 2 }).
		ToSlice()

	fmt.Println(result)
	// Output:
	// [4 8 12]
}

func ExampleStream() {
	// Process a sequence of names: filter, sort by length, and join.
	result := StreamOf("Go", "Python", "Rust", "TypeScript").
		Filter(func(s string) bool { return len(s) > 2 }).
		SortedBy(func(s string) int { return len(s) }).
		Collect(Joining(", "))

	fmt.Println(result)
	// Output:
	// Rust, Python, TypeScript
}

func ExampleStreamEmpty() {
	empty := StreamEmpty[int]()
	fmt.Println(empty.Count())
	// Output:
	// 0
}

func ExampleStreamOf() {
	StreamOf("Go", "Java", "Rust").
		ForEach(func(lang string) {
			fmt.Println(lang)
		})
	// Output:
	// Go
	// Java
	// Rust
}

func ExampleStreamOfSlice() {
	words := []string{"hello", "world"}
	count := StreamOfSlice(words).Count()
	fmt.Println(count)
	// Output:
	// 2
}

func ExampleStreamOfCollection() {
	list := StreamOf("apple", "banana", "cherry").
		Collect(ToList[string]())
	count := StreamOfCollection(list).
		Filter(func(s string) bool { return len(s) > 5 }).
		Count()
	fmt.Println(count)
	// Output:
	// 2
}

func ExampleStreamRange() {
	evens := StreamRange(0, 6).
		Filter(func(n int) bool { return n%2 == 0 }).
		ToSlice()
	fmt.Println(evens)
	// Output:
	// [0 2 4]
}

func ExampleStreamRangeClosed() {
	sum := StreamRangeClosed(1, 5).
		ReduceWithIdentity(0, func(a, b int) int { return a + b })
	fmt.Println(sum)
	// Output:
	// 15
}

func ExampleStreamConcat() {
	s1 := StreamOf(1, 2)
	s2 := StreamOf(3, 4)
	combined := StreamConcat(s1, s2).ToSlice()
	fmt.Println(combined)
	// Output:
	// [1 2 3 4]
}

func ExampleStream_Filter() {
	words := StreamOf("cat", "elephant", "dog", "bear").
		Filter(func(w string) bool { return len(w) > 3 }).
		ToSlice()
	fmt.Println(words)
	// Output:
	// [elephant bear]
}

func ExampleStream_Map() {
	lengths := StreamOf("one", "three", "six").
		Map(func(s string) int { return len(s) }).
		ToSlice()
	fmt.Println(lengths)
	// Output:
	// [3 5 3]
}

func ExampleStream_FlatMap() {
	pairs := StreamOf("ab", "cd").
		FlatMap(func(s string) Stream[rune] {
			return StreamOfSlice([]rune(s))
		}).
		Map(func(r rune) string { return string(r) }).
		ToSlice()
	fmt.Println(pairs)
	// Output:
	// [a b c d]
}

func ExampleStream_Distinct() {
	items := StreamOf(1, 2, 2, 3, 1, 4).
		Distinct().
		ToSlice()
	fmt.Println(items)
	// Output:
	// [1 2 3 4]
}

func ExampleStream_DistinctBy() {
	type Person struct {
		Name string
		ID   int
	}
	people := StreamOf(
		Person{Name: "Alice", ID: 1},
		Person{Name: "Bob", ID: 2},
		Person{Name: "Alice Duplicate", ID: 1},
	).DistinctBy(func(p Person) int { return p.ID }).
		Map(func(p Person) string { return p.Name }).
		ToSlice()
	fmt.Println(people)
	// Output:
	// [Alice Bob]
}

func ExampleStream_Sorted() {
	sorted := StreamOf(4, 1, 3, 2).
		Sorted().
		ToSlice()
	fmt.Println(sorted)
	// Output:
	// [1 2 3 4]
}

func ExampleStream_SortedBy() {
	type Product struct {
		Name  string
		Price int
	}
	sorted := StreamOf(
		Product{Name: "Book", Price: 15},
		Product{Name: "Pen", Price: 3},
		Product{Name: "Bag", Price: 40},
	).SortedBy(func(p Product) int { return p.Price }).
		Map(func(p Product) string { return p.Name }).
		ToSlice()
	fmt.Println(sorted)
	// Output:
	// [Pen Book Bag]
}

func ExampleStream_Limit() {
	firstThree := StreamRange(1, 100).
		Limit(3).
		ToSlice()
	fmt.Println(firstThree)
	// Output:
	// [1 2 3]
}

func ExampleStream_Skip() {
	afterTwo := StreamOf(10, 20, 30, 40).
		Skip(2).
		ToSlice()
	fmt.Println(afterTwo)
	// Output:
	// [30 40]
}

func ExampleStream_TakeWhile() {
	taken := StreamOf(1, 2, 3, 10, 4, 5).
		TakeWhile(func(n int) bool { return n < 5 }).
		ToSlice()
	fmt.Println(taken)
	// Output:
	// [1 2 3]
}

func ExampleStream_DropWhile() {
	dropped := StreamOf(1, 2, 3, 10, 4, 5).
		DropWhile(func(n int) bool { return n < 5 }).
		ToSlice()
	fmt.Println(dropped)
	// Output:
	// [10 4 5]
}

func ExampleStream_Peek() {
	var logged []int
	res := StreamOf(1, 2, 3).
		Peek(func(n int) { logged = append(logged, n) }).
		Map(func(n int) int { return n * 10 }).
		ToSlice()

	fmt.Println("Result:", res)
	fmt.Println("Logged:", logged)
	// Output:
	// Result: [10 20 30]
	// Logged: [1 2 3]
}

func ExampleStream_ToSlice() {
	slice := StreamOf("a", "b", "c").ToSlice()
	fmt.Println(slice)
	// Output:
	// [a b c]
}

func ExampleStream_FindFirst() {
	match := StreamOf("cat", "elephant", "dog").
		Filter(func(s string) bool { return len(s) > 3 }).
		FindFirst()
	if val, ok := match.Get(); ok {
		fmt.Println(val)
	}
	// Output:
	// elephant
}

func ExampleStream_AnyMatch() {
	hasEven := StreamOf(1, 3, 5, 6, 7).
		AnyMatch(func(n int) bool { return n%2 == 0 })
	fmt.Println(hasEven)
	// Output:
	// true
}

func ExampleStream_AllMatch() {
	allPositive := StreamOf(1, 2, 3).
		AllMatch(func(n int) bool { return n > 0 })
	fmt.Println(allPositive)
	// Output:
	// true
}

func ExampleStream_NoneMatch() {
	noneNegative := StreamOf(1, 2, 3).
		NoneMatch(func(n int) bool { return n < 0 })
	fmt.Println(noneNegative)
	// Output:
	// true
}

func ExampleStream_ReduceWithIdentity() {
	product := StreamOf(2, 3, 4).
		ReduceWithIdentity(1, func(a, b int) int { return a * b })
	fmt.Println(product)
	// Output:
	// 24
}

func ExampleStream_ReduceWithInitial() {
	res := StreamOf(1, 2, 3).
		ReduceWithInitial("Numbers:", func(acc string, n int) string {
			return fmt.Sprintf("%s %d", acc, n)
		})
	fmt.Println(res)
	// Output:
	// Numbers: 1 2 3
}

func ExampleStream_Min() {
	minVal := StreamOf(5, 2, 8, 1, 9).Min()
	if val, ok := minVal.Get(); ok {
		fmt.Println(val)
	}
	// Output:
	// 1
}

func ExampleStream_Max() {
	maxVal := StreamOf(5, 2, 8, 1, 9).Max()
	if val, ok := maxVal.Get(); ok {
		fmt.Println(val)
	}
	// Output:
	// 9
}

func ExampleStream_Parallel() {
	count := StreamRangeClosed(1, 100).
		Parallel().
		Filter(func(n int) bool { return n%2 == 0 }).
		Count()
	fmt.Println(count)
	// Output:
	// 50
}

func ExampleStream_OnClose() {
	stream := StreamOf(1, 2, 3).OnClose(func() {
		fmt.Println("stream closed")
	})
	_ = stream.ToSlice()
	// Output:
	// stream closed
}
