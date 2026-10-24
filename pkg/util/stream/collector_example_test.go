//nolint:errcheck // Not printing error handling
package stream_test

import (
	"fmt"

	. "github.com/BeamFoundry/rays/pkg/util/stream"
)

func ExampleStream_Collect() {
	list := StreamOf(1, 2, 3).
		Collect(ToList[int]())
	fmt.Println(list.ToSlice())
	// Output:
	// [1 2 3]
}

func ExampleToList() {
	evens := StreamOf(1, 2, 3, 4).
		Filter(func(n int) bool { return n%2 == 0 }).
		Collect(ToList[int]())
	fmt.Println(evens.ToSlice())
	// Output:
	// [2 4]
}

func ExampleToSet() {
	set := StreamOf("apple", "banana", "apple", "cherry").
		Collect(ToSet[string]())

	fmt.Println(set.Contains("apple"))
	fmt.Println(set.Contains("banana"))
	fmt.Println(set.Size())
	// Output:
	// true
	// true
	// 3
}

func ExampleJoining() {
	joined := StreamOf("Alice", "Bob", "Charlie").
		Collect(Joining(", ", "[", "]"))
	fmt.Println(joined)
	// Output:
	// [Alice, Bob, Charlie]
}

func ExampleGroupingBySlice() {
	type Employee struct {
		Name string
		Dept string
	}
	employees := []Employee{
		{Name: "Alice", Dept: "Engineering"},
		{Name: "Bob", Dept: "Sales"},
		{Name: "Charlie", Dept: "Engineering"},
	}
	byDept := StreamOfSlice(employees).Collect(
		GroupingBySlice(func(e Employee) string { return e.Dept }),
	)
	fmt.Println(len(byDept["Engineering"]))
	fmt.Println(len(byDept["Sales"]))
	// Output:
	// 2
	// 1
}

func ExampleGroupingBy() {
	words := []string{"apple", "apricot", "banana", "cherry", "cranberry"}
	counts := StreamOfSlice(words).Collect(
		GroupingBy(
			func(w string) string { return string(w[0]) },
			Counting[string](),
		),
	)
	fmt.Println(counts["a"])
	fmt.Println(counts["b"])
	fmt.Println(counts["c"])
	// Output:
	// 2
	// 1
	// 2
}

func ExamplePartitioningBySlice() {
	partitioned := StreamOf(1, 2, 3, 4, 5, 6).Collect(
		PartitioningBySlice(func(n int) bool { return n%2 == 0 }),
	)
	fmt.Println(partitioned[true])
	fmt.Println(partitioned[false])
	// Output:
	// [2 4 6]
	// [1 3 5]
}

func ExamplePartitioningBy() {
	grades := []int{45, 78, 92, 55, 88}
	counts := StreamOfSlice(grades).Collect(
		PartitioningBy(
			func(grade int) bool { return grade >= 60 },
			Counting[int](),
		),
	)
	fmt.Println("Passed:", counts[true])
	fmt.Println("Failed:", counts[false])
	// Output:
	// Passed: 3
	// Failed: 2
}

func ExampleSummarizingInt() {
	stats := StreamOf(10, 20, 30, 40).Collect(
		SummarizingInt(func(n int) int { return n }),
	)
	fmt.Printf("Count=%d, Sum=%d, Min=%d, Max=%d, Avg=%.1f\n",
		stats.Count, stats.Sum, stats.Min, stats.Max, stats.Average())
	// Output:
	// Count=4, Sum=100, Min=10, Max=40, Avg=25.0
}

func ExampleToMap() {
	type City struct {
		Code string
		Name string
	}
	cities := []City{
		{Code: "NYC", Name: "New York City"},
		{Code: "PAR", Name: "Paris"},
		{Code: "LON", Name: "London"},
	}
	cityMap := StreamOfSlice(cities).Collect(
		ToMap(
			func(c City) string { return c.Code },
			func(c City) string { return c.Name },
		),
	)
	fmt.Println(cityMap["PAR"])
	fmt.Println(cityMap["NYC"])
	// Output:
	// Paris
	// New York City
}

func ExampleCounting() {
	count := StreamOf("one", "two", "three").Collect(
		Counting[string](),
	)
	fmt.Println(count)
	// Output:
	// 3
}
