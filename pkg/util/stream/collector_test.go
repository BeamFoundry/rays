package stream_test

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/BeamFoundry/rays/pkg/util/stream"
)

var _ = Describe("Collectors", func() {
	It("collects to list with ToList and slice with ToSlice", func() {
		res := StreamOf(1, 2, 3).Collect(ToList[int]())
		Expect(res.Size()).To(Equal(3))
		Expect(res.ToSlice()).To(Equal([]int{1, 2, 3}))

		res2 := StreamOf("a", "b").Collect(ToSlice[string]())
		Expect(res2).To(Equal([]string{"a", "b"}))
	})

	It("collects to set with ToSet and ToSetSlice", func() {
		set := StreamOf(1, 2, 2, 3, 1).Collect(ToSet[int]())
		Expect(set.Size()).To(Equal(3))
		Expect(set.Contains(1)).To(BeTrue())
		Expect(set.Contains(2)).To(BeTrue())
		Expect(set.Contains(3)).To(BeTrue())

		slice := StreamOf(1, 2, 2, 3, 1).Collect(ToSetSlice[int]())
		Expect(slice).To(ConsistOf(1, 2, 3))
	})

	It("joins strings with Joining", func() {
		res := StreamOf("apple", "banana", "cherry").Collect(Joining(", "))
		Expect(res).To(Equal("apple, banana, cherry"))

		resWithPrefixSuffix := StreamOf("apple", "banana").Collect(Joining(", ", "[", "]"))
		Expect(resWithPrefixSuffix).To(Equal("[apple, banana]"))

		emptyJoin := StreamEmpty[string]().Collect(Joining(", ", "[", "]"))
		Expect(emptyJoin).To(Equal("[]"))
	})

	It("counts elements with Counting", func() {
		count := StreamOf(10, 20, 30).Collect(Counting[int]())
		Expect(count).To(Equal(int64(3)))
	})

	It("sums integers and floats with SummingInt and SummingFloat64", func() {
		words := StreamOf("apple", "banana", "pie")
		sum := words.Collect(SummingInt(func(s string) int { return len(s) }))
		Expect(sum).To(Equal(int64(5 + 6 + 3)))

		floats := StreamOf(1.5, 2.5, 3.0)
		floatSum := floats.Collect(SummingFloat64(func(x float64) float64 { return x }))
		Expect(floatSum).To(Equal(7.0))
	})

	It("averages values with AveragingInt and AveragingFloat64", func() {
		words := StreamOf("a", "bb", "ccc")
		avg := words.Collect(AveragingInt(func(s string) int { return len(s) }))
		Expect(avg).To(Equal(2.0))

		floats := StreamOf(10.0, 20.0, 30.0)
		floatAvg := floats.Collect(AveragingFloat64(func(x float64) float64 { return x }))
		Expect(floatAvg).To(Equal(20.0))

		emptyAvg := StreamEmpty[int]().Collect(AveragingInt(func(x int) int { return x }))
		Expect(emptyAvg).To(Equal(0.0))
	})

	It("summarizes values with SummarizingInt and SummarizingFloat64", func() {
		stats := StreamOf(1, 2, 3, 4, 5).Collect(SummarizingInt(func(x int) int { return x }))
		Expect(stats.Count).To(Equal(int64(5)))
		Expect(stats.Sum).To(Equal(int64(15)))
		Expect(stats.Min).To(Equal(1))
		Expect(stats.Max).To(Equal(5))
		Expect(stats.Average()).To(Equal(3.0))
		Expect(stats.String()).To(ContainSubstring("IntSummaryStatistics"))

		fStats := StreamOf(1.0, 2.0, 3.0).Collect(SummarizingFloat64(func(x float64) float64 { return x }))
		Expect(fStats.Count).To(Equal(int64(3)))
		Expect(fStats.Sum).To(Equal(6.0))
		Expect(fStats.Min).To(Equal(1.0))
		Expect(fStats.Max).To(Equal(3.0))
		Expect(fStats.Average()).To(Equal(2.0))
		Expect(fStats.String()).To(ContainSubstring("Float64SummaryStatistics"))
	})

	It("transforms with Mapping, Filtering, and FlatMapping downstream collectors", func() {
		// Mapping: extract lengths and sum them
		sumLens := StreamOf("hi", "hello", "world").
			Collect(Mapping(
				func(s string) int { return len(s) },
				SummingInt(func(l int) int { return l }),
			))
		Expect(sumLens).To(Equal(int64(2 + 5 + 5)))

		// Filtering: only keep words of length > 2 and join them
		joined := StreamOf("a", "big", "apple").
			Collect(Filtering(
				func(s string) bool { return len(s) > 2 },
				Joining(", "),
			))
		Expect(joined).To(Equal("big, apple"))

		// FlatMapping: split words and collect all characters into set
		chars := StreamOf("hi", "bye").
			Collect(FlatMapping(
				func(s string) []string { return strings.Split(s, "") },
				ToSet[string](),
			))
		Expect(chars.Size()).To(Equal(5)) // h, i, b, y, e
	})

	It("groups elements with GroupingBy and GroupingBySlice", func() {
		words := StreamOf("apple", "banana", "apricot", "blueberry", "cherry")

		grouped := words.Collect(GroupingBy(
			func(s string) string { return string(s[0]) },
			ToList[string](),
		))

		Expect(grouped["a"].ToSlice()).To(Equal([]string{"apple", "apricot"}))
		Expect(grouped["b"].ToSlice()).To(Equal([]string{"banana", "blueberry"}))
		Expect(grouped["c"].ToSlice()).To(Equal([]string{"cherry"}))

		// GroupingBySlice
		groupedSlice := StreamOf("one", "two", "three", "four").
			Collect(GroupingBySlice(func(s string) int { return len(s) }))

		Expect(groupedSlice[3]).To(Equal([]string{"one", "two"}))
		Expect(groupedSlice[4]).To(Equal([]string{"four"}))
		Expect(groupedSlice[5]).To(Equal([]string{"three"}))
	})

	It("partitions elements with PartitioningBy and PartitioningBySlice", func() {
		partitioned := StreamOf(1, 2, 3, 4, 5, 6).
			Collect(PartitioningBySlice(func(x int) bool { return x%2 == 0 }))

		Expect(partitioned[true]).To(Equal([]int{2, 4, 6}))
		Expect(partitioned[false]).To(Equal([]int{1, 3, 5}))

		// With downstream collector (Counting)
		partitionCounts := StreamOf(1, 2, 3, 4, 5, 6).
			Collect(PartitioningBy(
				func(x int) bool { return x%2 == 0 },
				Counting[int](),
			))

		Expect(partitionCounts[true]).To(Equal(int64(3)))
		Expect(partitionCounts[false]).To(Equal(int64(3)))
	})

	It("collects into map with ToMap", func() {
		m := StreamOf("a", "bb", "ccc").Collect(
			ToMap(
				func(s string) string { return s },
				func(s string) int { return len(s) },
			),
		)
		Expect(m).To(Equal(map[string]int{"a": 1, "bb": 2, "ccc": 3}))

		// With merge rule for duplicate keys
		merged := StreamOf("apple", "apricot", "banana").Collect(
			ToMap(
				func(s string) string { return string(s[0]) },
				func(s string) string { return s },
				func(existing, newVal string) string { return existing + "/" + newVal },
			),
		)
		Expect(merged["a"]).To(Equal("apple/apricot"))
		Expect(merged["b"]).To(Equal("banana"))

		// Duplicate keys without merge rule panics with IllegalStateError
		Expect(func() {
			StreamOf("apple", "apricot").Collect(
				ToMap(
					func(s string) string { return string(s[0]) },
					func(s string) string { return s },
				),
			)
		}).To(Panic())
	})

	It("reduces using Reducing collector", func() {
		res := StreamOf("a", "b", "c").Collect(
			Reducing("", func(a, b string) string { return a + b }),
		)
		Expect(res).To(Equal("abc"))
	})

	It("finds min and max using MinBy and MaxBy collectors", func() {
		minOpt := StreamOf(4, 2, 9, 1, 7).Collect(
			MinBy(func(a, b int) int { return a - b }),
		)
		Expect(minOpt.IsPresent()).To(BeTrue())
		valMin, _ := minOpt.Get()
		Expect(valMin).To(Equal(1))

		maxOpt := StreamOf(4, 2, 9, 1, 7).Collect(
			MaxBy(func(a, b int) int { return a - b }),
		)
		Expect(maxOpt.IsPresent()).To(BeTrue())
		valMax, _ := maxOpt.Get()
		Expect(valMax).To(Equal(9))
	})

	It("collects using CollectWith", func() {
		res := StreamOf(1, 2, 3, 4).CollectWith(
			func() []int { return []int{} },
			func(acc []int, item int) []int { return append(acc, item*2) },
		)
		Expect(res).To(Equal([]int{2, 4, 6, 8}))
	})

	It("supports NewCollector directly", func() {
		c := NewCollector(
			func() []string { return []string{} },
			func(acc []string, item int) []string { return append(acc, strings.Repeat("x", item)) },
			func(a, b []string) []string { return append(a, b...) },
			func(acc []string) int { return len(acc) },
		)
		res := StreamOf(1, 2, 3).Collect(c)
		Expect(res).To(Equal(3))
	})

	It("combines IntSummaryStatistics and Float64SummaryStatistics directly", func() {
		s1 := NewIntSummaryStatistics()
		s1.Accept(10)
		s1.Accept(20)

		s2 := NewIntSummaryStatistics()
		s2.Accept(5)
		s2.Accept(40)

		s1.Combine(s2)
		Expect(s1.Count).To(Equal(int64(4)))
		Expect(s1.Sum).To(Equal(int64(75)))
		Expect(s1.Min).To(Equal(5))
		Expect(s1.Max).To(Equal(40))

		// Combine with empty
		s1.Combine(nil)
		s1.Combine(NewIntSummaryStatistics())
		Expect(s1.Count).To(Equal(int64(4)))

		// Combine into empty
		emptyStats := NewIntSummaryStatistics()
		emptyStats.Combine(s1)
		Expect(emptyStats.Count).To(Equal(int64(4)))

		// Float64
		f1 := NewFloat64SummaryStatistics()
		f1.Accept(1.5)
		f1.Accept(3.5)

		f2 := NewFloat64SummaryStatistics()
		f2.Accept(0.5)
		f2.Accept(4.5)

		f1.Combine(f2)
		Expect(f1.Count).To(Equal(int64(4)))
		Expect(f1.Sum).To(Equal(10.0))
		Expect(f1.Min).To(Equal(0.5))
		Expect(f1.Max).To(Equal(4.5))

		// Combine with empty / nil
		f1.Combine(nil)
		f1.Combine(NewFloat64SummaryStatistics())
		Expect(f1.Count).To(Equal(int64(4)))

		emptyF := NewFloat64SummaryStatistics()
		emptyF.Combine(f1)
		Expect(emptyF.Count).To(Equal(int64(4)))

		// Test zero-initialized IntSummaryStatistics
		var zeroIntStats IntSummaryStatistics
		zeroIntStats.Accept(42)
		Expect(zeroIntStats.Count).To(Equal(int64(1)))
		Expect(zeroIntStats.Sum).To(Equal(int64(42)))
		Expect(zeroIntStats.Min).To(Equal(42))
		Expect(zeroIntStats.Max).To(Equal(42))

		// Test zero-initialized Float64SummaryStatistics
		var zeroFloatStats Float64SummaryStatistics
		zeroFloatStats.Accept(42.5)
		Expect(zeroFloatStats.Count).To(Equal(int64(1)))
		Expect(zeroFloatStats.Sum).To(Equal(42.5))
		Expect(zeroFloatStats.Min).To(Equal(42.5))
		Expect(zeroFloatStats.Max).To(Equal(42.5))
	})

	It("propagates nil Combiner in GroupingBy and PartitioningBy when downstream has no Combiner", func() {
		noCombinerDownstream := NewSimpleCollector(
			func() []int { return nil },
			func(acc []int, item int) []int { return append(acc, item) },
		)
		Expect(noCombinerDownstream.Combiner).To(BeNil())

		grouped := GroupingBy(func(x int) string {
			if x%2 == 0 {
				return "even"
			}
			return "odd"
		}, noCombinerDownstream)
		Expect(grouped.Combiner).To(BeNil())

		// Parallel stream falls back to sequential collect when Combiner is nil
		resGrouped := StreamOf(1, 2, 3, 4).Parallel().Collect(grouped)
		Expect(resGrouped["even"]).To(Equal([]int{2, 4}))
		Expect(resGrouped["odd"]).To(Equal([]int{1, 3}))

		partitioned := PartitioningBy(func(x int) bool { return x%2 == 0 }, noCombinerDownstream)
		Expect(partitioned.Combiner).To(BeNil())

		resPartitioned := StreamOf(1, 2, 3, 4).Parallel().Collect(partitioned)
		Expect(resPartitioned[true]).To(Equal([]int{2, 4}))
		Expect(resPartitioned[false]).To(Equal([]int{1, 3}))
	})
})
