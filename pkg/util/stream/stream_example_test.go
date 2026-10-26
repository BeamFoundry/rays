//nolint:errcheck // Not printing error handling
package stream_test

import (
	"fmt"

	. "github.com/BeamFoundry/rays/pkg/util/stream"
)

// Example demonstrates filtering, mapping, and collecting a stream of integers.
func Example() {
	// A pipeline filtering even numbers, doubling them, and collecting to a slice.
	result := Stream[int]{}.From(1, 2, 3, 4, 5, 6).
		Filter(func(n int) bool { return n%2 == 0 }).
		Map(func(n int) int { return n * 2 }).
		ToSlice()

	fmt.Println(result)
	// Output:
	// [4 8 12]
}

// ExampleStream demonstrates filtering, sorting, and joining language names.
func ExampleStream() {
	// Process a sequence of names: filter, sort by length, and join.
	result := Stream[string]{}.From("Go", "Python", "Rust", "TypeScript").
		Filter(func(s string) bool { return len(s) > 2 }).
		SortedBy(func(s string) int { return len(s) }).
		Collect(Collectors{}.Joining(", "))

	fmt.Println(result)
	// Output:
	// Rust, Python, TypeScript
}

// ExampleStream_FromEmpty demonstrates creating an empty stream with zero elements.
func ExampleStream_FromEmpty() {
	empty := Stream[int]{}.FromEmpty()
	fmt.Println(empty.Count())
	// Output:
	// 0
}

// ExampleStream_From demonstrates creating and consuming a stream of explicit values.
func ExampleStream_From() {
	Stream[string]{}.From("Go", "Java", "Rust").
		ForEach(func(lang string) {
			fmt.Println(lang)
		})
	// Output:
	// Go
	// Java
	// Rust
}

// ExampleStream_FromSlice demonstrates counting elements from a slice.
func ExampleStream_FromSlice() {
	words := []string{"hello", "world"}
	count := Stream[string]{}.FromSlice(words).Count()
	fmt.Println(count)
	// Output:
	// 2
}

// ExampleStream_FromCollection demonstrates filtering elements from a collection.
func ExampleStream_FromCollection() {
	list := Stream[string]{}.From("apple", "banana", "cherry").
		Collect(Collectors{}.ToList[string]())
	count := Stream[string]{}.FromCollection(list).
		Filter(func(s string) bool { return len(s) > 5 }).
		Count()
	fmt.Println(count)
	// Output:
	// 2
}

// ExampleStream_FromRange demonstrates filtering a range with an exclusive upper bound.
func ExampleStream_FromRange() {
	evens := Stream[int]{}.FromRange(0, 6).
		Filter(func(n int) bool { return n%2 == 0 }).
		ToSlice()
	fmt.Println(evens)
	// Output:
	// [0 2 4]
}

// ExampleStream_FromRangeClosed demonstrates summing a range with an inclusive upper bound.
func ExampleStream_FromRangeClosed() {
	sum := Stream[int]{}.FromRangeClosed(1, 5).
		ReduceWithIdentity(0, func(a, b int) int { return a + b })
	fmt.Println(sum)
	// Output:
	// 15
}

// ExampleStream_FromConcat demonstrates concatenating two streams in encounter order.
func ExampleStream_FromConcat() {
	s1 := Stream[int]{}.From(1, 2)
	s2 := Stream[int]{}.From(3, 4)
	combined := Stream[int]{}.FromConcat(s1, s2).ToSlice()
	fmt.Println(combined)
	// Output:
	// [1 2 3 4]
}

// ExampleStream_Filter demonstrates retaining words that match a length predicate.
func ExampleStream_Filter() {
	words := Stream[string]{}.From("cat", "elephant", "dog", "bear").
		Filter(func(w string) bool { return len(w) > 3 }).
		ToSlice()
	fmt.Println(words)
	// Output:
	// [elephant bear]
}

// ExampleStream_Map demonstrates mapping strings to their lengths.
func ExampleStream_Map() {
	lengths := Stream[string]{}.From("one", "three", "six").
		Map(func(s string) int { return len(s) }).
		ToSlice()
	fmt.Println(lengths)
	// Output:
	// [3 5 3]
}

// ExampleStream_FlatMap demonstrates flattening character streams from strings.
func ExampleStream_FlatMap() {
	pairs := Stream[string]{}.From("ab", "cd").
		FlatMap(func(s string) Stream[rune] {
			return Stream[rune]{}.FromSlice([]rune(s))
		}).
		Map(func(r rune) string { return string(r) }).
		ToSlice()
	fmt.Println(pairs)
	// Output:
	// [a b c d]
}

// ExampleStream_Distinct demonstrates removing duplicate integers.
func ExampleStream_Distinct() {
	items := Stream[int]{}.From(1, 2, 2, 3, 1, 4).
		Distinct().
		ToSlice()
	fmt.Println(items)
	// Output:
	// [1 2 3 4]
}

// ExampleStream_DistinctBy demonstrates retaining the first person for each unique ID.
func ExampleStream_DistinctBy() {
	type Person struct {
		Name string
		ID   int
	}
	people := Stream[Person]{}.From(
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

// ExampleStream_Sorted demonstrates sorting integers in natural order.
func ExampleStream_Sorted() {
	sorted := Stream[int]{}.From(4, 1, 3, 2).
		Sorted().
		ToSlice()
	fmt.Println(sorted)
	// Output:
	// [1 2 3 4]
}

// ExampleStream_SortedBy demonstrates sorting products by ascending price.
func ExampleStream_SortedBy() {
	type Product struct {
		Name  string
		Price int
	}
	sorted := Stream[Product]{}.From(
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

// ExampleStream_Limit demonstrates retaining only the first three elements.
func ExampleStream_Limit() {
	firstThree := Stream[int]{}.FromRange(1, 100).
		Limit(3).
		ToSlice()
	fmt.Println(firstThree)
	// Output:
	// [1 2 3]
}

// ExampleStream_Skip demonstrates discarding the first two elements.
func ExampleStream_Skip() {
	afterTwo := Stream[int]{}.From(10, 20, 30, 40).
		Skip(2).
		ToSlice()
	fmt.Println(afterTwo)
	// Output:
	// [30 40]
}

// ExampleStream_TakeWhile demonstrates taking elements until a predicate first fails.
func ExampleStream_TakeWhile() {
	taken := Stream[int]{}.From(1, 2, 3, 10, 4, 5).
		TakeWhile(func(n int) bool { return n < 5 }).
		ToSlice()
	fmt.Println(taken)
	// Output:
	// [1 2 3]
}

// ExampleStream_DropWhile demonstrates skipping elements until a predicate first fails.
func ExampleStream_DropWhile() {
	dropped := Stream[int]{}.From(1, 2, 3, 10, 4, 5).
		DropWhile(func(n int) bool { return n < 5 }).
		ToSlice()
	fmt.Println(dropped)
	// Output:
	// [10 4 5]
}

// ExampleStream_Peek demonstrates observing elements as a pipeline consumes them.
func ExampleStream_Peek() {
	var logged []int
	res := Stream[int]{}.From(1, 2, 3).
		Peek(func(n int) { logged = append(logged, n) }).
		Map(func(n int) int { return n * 10 }).
		ToSlice()

	fmt.Println("Result:", res)
	fmt.Println("Logged:", logged)
	// Output:
	// Result: [10 20 30]
	// Logged: [1 2 3]
}

// ExampleStream_ToSlice demonstrates collecting stream elements into a slice.
func ExampleStream_ToSlice() {
	slice := Stream[string]{}.From("a", "b", "c").ToSlice()
	fmt.Println(slice)
	// Output:
	// [a b c]
}

// ExampleStream_FindFirst demonstrates finding the first word that matches a filter.
func ExampleStream_FindFirst() {
	match := Stream[string]{}.From("cat", "elephant", "dog").
		Filter(func(s string) bool { return len(s) > 3 }).
		FindFirst()
	if val, ok := match.Get(); ok {
		fmt.Println(val)
	}
	// Output:
	// elephant
}

// ExampleStream_AnyMatch demonstrates testing whether a stream contains an even number.
func ExampleStream_AnyMatch() {
	hasEven := Stream[int]{}.From(1, 3, 5, 6, 7).
		AnyMatch(func(n int) bool { return n%2 == 0 })
	fmt.Println(hasEven)
	// Output:
	// true
}

// ExampleStream_AllMatch demonstrates testing whether every number is positive.
func ExampleStream_AllMatch() {
	allPositive := Stream[int]{}.From(1, 2, 3).
		AllMatch(func(n int) bool { return n > 0 })
	fmt.Println(allPositive)
	// Output:
	// true
}

// ExampleStream_NoneMatch demonstrates testing whether no numbers are negative.
func ExampleStream_NoneMatch() {
	noneNegative := Stream[int]{}.From(1, 2, 3).
		NoneMatch(func(n int) bool { return n < 0 })
	fmt.Println(noneNegative)
	// Output:
	// true
}

// ExampleStream_ReduceWithIdentity demonstrates multiplying numbers with an identity of one.
func ExampleStream_ReduceWithIdentity() {
	product := Stream[int]{}.From(2, 3, 4).
		ReduceWithIdentity(1, func(a, b int) int { return a * b })
	fmt.Println(product)
	// Output:
	// 24
}

// ExampleStream_ReduceWithInitial demonstrates reducing integers into a string result.
func ExampleStream_ReduceWithInitial() {
	res := Stream[int]{}.From(1, 2, 3).
		ReduceWithInitial("Numbers:", func(acc string, n int) string {
			return fmt.Sprintf("%s %d", acc, n)
		})
	fmt.Println(res)
	// Output:
	// Numbers: 1 2 3
}

// ExampleStream_Min demonstrates finding the smallest integer in a stream.
func ExampleStream_Min() {
	minVal := Stream[int]{}.From(5, 2, 8, 1, 9).Min()
	if val, ok := minVal.Get(); ok {
		fmt.Println(val)
	}
	// Output:
	// 1
}

// ExampleStream_Max demonstrates finding the largest integer in a stream.
func ExampleStream_Max() {
	maxVal := Stream[int]{}.From(5, 2, 8, 1, 9).Max()
	if val, ok := maxVal.Get(); ok {
		fmt.Println(val)
	}
	// Output:
	// 9
}

// ExampleStream_Parallel demonstrates filtering and counting a stream marked for parallel execution.
func ExampleStream_Parallel() {
	count := Stream[int]{}.FromRangeClosed(1, 100).
		Parallel().
		Filter(func(n int) bool { return n%2 == 0 }).
		Count()
	fmt.Println(count)
	// Output:
	// 50
}

// ExampleStream_OnClose demonstrates running a close handler after stream consumption.
func ExampleStream_OnClose() {
	stream := Stream[int]{}.From(1, 2, 3).OnClose(func() {
		fmt.Println("stream closed")
	})
	_ = stream.ToSlice()
	// Output:
	// stream closed
}
