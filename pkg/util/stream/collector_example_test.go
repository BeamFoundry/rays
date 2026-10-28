//nolint:errcheck // Not printing error handling
package stream_test

import (
	"fmt"

	. "github.com/BeamFoundry/rays/pkg/util/stream"
)

// ExampleStream_Collect demonstrates collecting stream elements into a list.
func ExampleStream_Collect() {
	list := streamOf(1, 2, 3).
		Collect(Collectors{}.ToList[int]())
	fmt.Println(list.ToSlice())
	// Output:
	// [1 2 3]
}

// ExampleCollectors_ToList demonstrates collecting filtered even numbers into a list.
func ExampleCollectors_ToList() {
	evens := streamOf(1, 2, 3, 4).
		Filter(func(n int) bool { return n%2 == 0 }).
		Collect(Collectors{}.ToList[int]())
	fmt.Println(evens.ToSlice())
	// Output:
	// [2 4]
}

// ExampleCollectors_ToSet demonstrates collecting unique strings into a set.
func ExampleCollectors_ToSet() {
	set := streamOf("apple", "banana", "apple", "cherry").
		Collect(Collectors{}.ToSet[string]())

	fmt.Println(set.Contains("apple"))
	fmt.Println(set.Contains("banana"))
	fmt.Println(set.Size())
	// Output:
	// true
	// true
	// 3
}

// ExampleCollectors_Joining demonstrates joining names with a delimiter, prefix, and suffix.
func ExampleCollectors_Joining() {
	joined := streamOf("Alice", "Bob", "Charlie").
		Collect(Collectors{}.Joining(", ", "[", "]"))
	fmt.Println(joined)
	// Output:
	// [Alice, Bob, Charlie]
}

// ExampleCollectors_GroupingBySlice demonstrates grouping employees into slices by department.
func ExampleCollectors_GroupingBySlice() {
	type Employee struct {
		Name string
		Dept string
	}
	employees := []Employee{
		{Name: "Alice", Dept: "Engineering"},
		{Name: "Bob", Dept: "Sales"},
		{Name: "Charlie", Dept: "Engineering"},
	}
	byDept := streamOfSlice(employees).Collect(
		Collectors{}.GroupingBySlice(func(e Employee) string { return e.Dept }),
	)
	fmt.Println(len(byDept["Engineering"]))
	fmt.Println(len(byDept["Sales"]))
	// Output:
	// 2
	// 1
}

// ExampleCollectors_GroupingBy demonstrates counting words grouped by their first letter.
func ExampleCollectors_GroupingBy() {
	words := []string{"apple", "apricot", "banana", "cherry", "cranberry"}
	counts := streamOfSlice(words).Collect(
		Collectors{}.GroupingBy(
			func(w string) string { return string(w[0]) },
			Collectors{}.Counting[string](),
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

// ExampleCollectors_PartitioningBySlice demonstrates partitioning numbers into even and odd slices.
func ExampleCollectors_PartitioningBySlice() {
	partitioned := streamOf(1, 2, 3, 4, 5, 6).Collect(
		Collectors{}.PartitioningBySlice(func(n int) bool { return n%2 == 0 }),
	)
	fmt.Println(partitioned[true])
	fmt.Println(partitioned[false])
	// Output:
	// [2 4 6]
	// [1 3 5]
}

// ExampleCollectors_PartitioningBy demonstrates counting passing and failing grades.
func ExampleCollectors_PartitioningBy() {
	grades := []int{45, 78, 92, 55, 88}
	counts := streamOfSlice(grades).Collect(
		Collectors{}.PartitioningBy(
			func(grade int) bool { return grade >= 60 },
			Collectors{}.Counting[int](),
		),
	)
	fmt.Println("Passed:", counts[true])
	fmt.Println("Failed:", counts[false])
	// Output:
	// Passed: 3
	// Failed: 2
}

// ExampleCollectors_SummarizingInt demonstrates collecting count, sum, extrema, and average.
func ExampleCollectors_SummarizingInt() {
	stats := streamOf(10, 20, 30, 40).Collect(
		Collectors{}.SummarizingInt(func(n int) int { return n }),
	)
	fmt.Printf("Count=%d, Sum=%d, Min=%d, Max=%d, Avg=%.1f\n",
		stats.Count, stats.Sum, stats.Min, stats.Max, stats.Average())
	// Output:
	// Count=4, Sum=100, Min=10, Max=40, Avg=25.0
}

// ExampleCollectors_ToMap demonstrates collecting city names into a map keyed by city code.
func ExampleCollectors_ToMap() {
	type City struct {
		Code string
		Name string
	}
	cities := []City{
		{Code: "NYC", Name: "New York City"},
		{Code: "PAR", Name: "Paris"},
		{Code: "LON", Name: "London"},
	}
	cityMap := streamOfSlice(cities).Collect(
		Collectors{}.ToMap(
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

// ExampleCollectors_Counting demonstrates counting elements with a collector.
func ExampleCollectors_Counting() {
	count := streamOf("one", "two", "three").Collect(
		Collectors{}.Counting[string](),
	)
	fmt.Println(count)
	// Output:
	// 3
}
