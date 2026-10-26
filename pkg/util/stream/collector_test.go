package stream_test

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/BeamFoundry/rays/pkg/lang"
	. "github.com/BeamFoundry/rays/pkg/util/stream"
)

var _ = Describe("Collectors", func() {
	var c Collectors

	It("collects to list with ToList", func() {
		res := streamOf(1, 2, 3).Collect(c.ToList[int]())
		Expect(res.Size()).To(Equal(3))
		Expect(res.ToSlice()).To(Equal([]int{1, 2, 3}))
	})

	It("collects to slice with ToSlice", func() {
		res := streamOf("a", "b").Collect(c.ToSlice[string]())
		Expect(res).To(Equal([]string{"a", "b"}))
	})

	It("collects to set with ToSet", func() {
		set := streamOf(1, 2, 2, 3, 1).Collect(c.ToSet[int]())
		Expect(set.Size()).To(Equal(3))
		Expect(set.Contains(1)).To(BeTrue())
		Expect(set.Contains(2)).To(BeTrue())
		Expect(set.Contains(3)).To(BeTrue())
	})

	It("collects to deduplicated slice with ToSetSlice", func() {
		slice := streamOf(1, 2, 2, 3, 1).Collect(c.ToSetSlice[int]())
		Expect(slice).To(ConsistOf(1, 2, 3))
	})

	It("joins strings with Joining", func() {
		res := streamOf("apple", "banana", "cherry").Collect(c.Joining(", "))
		Expect(res).To(Equal("apple, banana, cherry"))

		resWithPrefixSuffix := streamOf("apple", "banana").Collect(c.Joining(", ", "[", "]"))
		Expect(resWithPrefixSuffix).To(Equal("[apple, banana]"))

		emptyJoin := streamEmpty[string]().Collect(c.Joining(", ", "[", "]"))
		Expect(emptyJoin).To(Equal("[]"))
	})

	It("counts elements with Counting", func() {
		count := streamOf(10, 20, 30).Collect(c.Counting[int]())
		Expect(count).To(Equal(int64(3)))
	})

	It("sums integers and floats with SummingInt and SummingFloat64", func() {
		words := streamOf("apple", "banana", "pie")
		sum := words.Collect(c.SummingInt(func(s string) int { return len(s) }))
		Expect(sum).To(Equal(int64(5 + 6 + 3)))

		floats := streamOf(1.5, 2.5, 3.0)
		floatSum := floats.Collect(c.SummingFloat64(func(x float64) float64 { return x }))
		Expect(floatSum).To(Equal(7.0))
	})

	It("averages values with AveragingInt and AveragingFloat64", func() {
		words := streamOf("a", "bb", "ccc")
		avg := words.Collect(c.AveragingInt(func(s string) int { return len(s) }))
		Expect(avg).To(Equal(2.0))

		floats := streamOf(10.0, 20.0, 30.0)
		floatAvg := floats.Collect(c.AveragingFloat64(func(x float64) float64 { return x }))
		Expect(floatAvg).To(Equal(20.0))

		emptyAvg := streamEmpty[int]().Collect(c.AveragingInt(func(x int) int { return x }))
		Expect(emptyAvg).To(Equal(0.0))
	})

	It("summarizes values with SummarizingInt and SummarizingFloat64", func() {
		stats := streamOf(1, 2, 3, 4, 5).Collect(c.SummarizingInt(func(x int) int { return x }))
		Expect(stats.Count).To(Equal(int64(5)))
		Expect(stats.Sum).To(Equal(int64(15)))
		Expect(stats.Min).To(Equal(1))
		Expect(stats.Max).To(Equal(5))
		Expect(stats.Average()).To(Equal(3.0))
		Expect(stats.String()).To(ContainSubstring("IntSummaryStatistics"))

		fStats := streamOf(1.0, 2.0, 3.0).Collect(c.SummarizingFloat64(func(x float64) float64 { return x }))
		Expect(fStats.Count).To(Equal(int64(3)))
		Expect(fStats.Sum).To(Equal(6.0))
		Expect(fStats.Min).To(Equal(1.0))
		Expect(fStats.Max).To(Equal(3.0))
		Expect(fStats.Average()).To(Equal(2.0))
		Expect(fStats.String()).To(ContainSubstring("Float64SummaryStatistics"))
	})

	It("transforms with Mapping, Filtering, and FlatMapping downstream collectors", func() {
		// Mapping: extract lengths and sum them
		sumLens := streamOf("hi", "hello", "world").
			Collect(c.Mapping(
				func(s string) int { return len(s) },
				c.SummingInt(func(l int) int { return l }),
			))
		Expect(sumLens).To(Equal(int64(2 + 5 + 5)))

		// Filtering: only keep words of length > 2 and join them
		joined := streamOf("a", "big", "apple").
			Collect(c.Filtering(
				func(s string) bool { return len(s) > 2 },
				c.Joining(", "),
			))
		Expect(joined).To(Equal("big, apple"))

		// FlatMapping: split words and collect all characters into set
		chars := streamOf("hi", "bye").
			Collect(c.FlatMapping(
				func(s string) []string { return strings.Split(s, "") },
				c.ToSet[string](),
			))
		Expect(chars.Size()).To(Equal(5)) // h, i, b, y, e
	})

	It("groups elements with GroupingBy and GroupingBySlice", func() {
		words := streamOf("apple", "banana", "apricot", "blueberry", "cherry")

		grouped := words.Collect(c.GroupingBy(
			func(s string) string { return string(s[0]) },
			c.ToList[string](),
		))

		Expect(grouped["a"].ToSlice()).To(Equal([]string{"apple", "apricot"}))
		Expect(grouped["b"].ToSlice()).To(Equal([]string{"banana", "blueberry"}))
		Expect(grouped["c"].ToSlice()).To(Equal([]string{"cherry"}))

		// GroupingBySlice
		groupedSlice := streamOf("one", "two", "three", "four").
			Collect(c.GroupingBySlice(func(s string) int { return len(s) }))

		Expect(groupedSlice[3]).To(Equal([]string{"one", "two"}))
		Expect(groupedSlice[4]).To(Equal([]string{"four"}))
		Expect(groupedSlice[5]).To(Equal([]string{"three"}))
	})

	It("partitions elements with PartitioningBy and PartitioningBySlice", func() {
		partitioned := streamOf(1, 2, 3, 4, 5, 6).
			Collect(c.PartitioningBySlice(func(x int) bool { return x%2 == 0 }))
		Expect(partitioned[true]).To(Equal([]int{2, 4, 6}))
		Expect(partitioned[false]).To(Equal([]int{1, 3, 5}))

		// With downstream collector (Counting)
		partitionCounts := streamOf(1, 2, 3, 4, 5, 6).
			Collect(c.PartitioningBy(
				func(x int) bool { return x%2 == 0 },
				c.Counting[int](),
			))

		Expect(partitionCounts[true]).To(Equal(int64(3)))
		Expect(partitionCounts[false]).To(Equal(int64(3)))
	})

	It("collects into map with ToMap", func() {
		m := streamOf("a", "bb", "ccc").Collect(
			c.ToMap(
				func(s string) string { return s },
				func(s string) int { return len(s) },
			),
		)
		Expect(m).To(Equal(map[string]int{"a": 1, "bb": 2, "ccc": 3}))

		// With merge rule for duplicate keys using ToMap
		merged := streamOf("apple", "apricot", "banana").Collect(
			c.ToMap(
				func(s string) string { return string(s[0]) },
				func(s string) string { return s },
				func(existing, newVal string) string { return existing + "/" + newVal },
			),
		)
		Expect(merged["a"]).To(Equal("apple/apricot"))
		Expect(merged["b"]).To(Equal("banana"))

		// Duplicate keys without merge rule panics with IllegalStateError
		Expect(func() {
			streamOf("apple", "apricot").Collect(
				c.ToMap(
					func(s string) string { return string(s[0]) },
					func(s string) string { return s },
				),
			)
		}).To(Panic())
	})

	It("reduces using Reducing collector", func() {
		res := streamOf("a", "b", "c").Collect(
			c.Reducing("", func(a, b string) string { return a + b }),
		)
		Expect(res).To(Equal("abc"))
	})

	It("finds min and max using MinBy and MaxBy collectors", func() {
		minOpt := streamOf(4, 2, 9, 1, 7).Collect(
			c.MinBy(func(a, b int) int { return a - b }),
		)
		Expect(minOpt.IsPresent()).To(BeTrue())
		valMin, _ := minOpt.Get()
		Expect(valMin).To(Equal(1))

		maxOpt := streamOf(4, 2, 9, 1, 7).Collect(
			c.MaxBy(func(a, b int) int { return a - b }),
		)
		Expect(maxOpt.IsPresent()).To(BeTrue())
		valMax, _ := maxOpt.Get()
		Expect(valMax).To(Equal(9))
	})

	It("collects using CollectWith", func() {
		res := streamOf(1, 2, 3, 4).CollectWith(
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
		res := streamOf(1, 2, 3).Collect(c)
		Expect(res).To(Equal(3))
	})

	It("combines IntSummaryStatistics and Float64SummaryStatistics directly", func() {
		s1 := NewIntSummaryStatistics()
		s1.Accept(10)
		s1.Accept(20)
		s1.Accept(2)

		s2 := NewIntSummaryStatistics()
		s2.Accept(5)
		s2.Accept(40)

		s1.Combine(s2)
		Expect(s1.Count).To(Equal(int64(5)))
		Expect(s1.Sum).To(Equal(int64(77)))
		Expect(s1.Min).To(Equal(2))
		Expect(s1.Max).To(Equal(40))

		// Combine with empty
		s1.Combine(nil)
		s1.Combine(NewIntSummaryStatistics())
		Expect(s1.Count).To(Equal(int64(5)))

		// Combine into empty
		emptyStats := NewIntSummaryStatistics()
		emptyStats.Combine(s1)
		Expect(emptyStats.Count).To(Equal(int64(5)))

		// Float64
		f1 := NewFloat64SummaryStatistics()
		f1.Accept(1.5)
		f1.Accept(3.5)
		f1.Accept(0.2)

		f2 := NewFloat64SummaryStatistics()
		f2.Accept(0.5)
		f2.Accept(4.5)

		f1.Combine(f2)
		Expect(f1.Count).To(Equal(int64(5)))
		Expect(f1.Sum).To(Equal(10.2))
		Expect(f1.Min).To(Equal(0.2))
		Expect(f1.Max).To(Equal(4.5))

		// Combine with empty / nil
		f1.Combine(nil)
		f1.Combine(NewFloat64SummaryStatistics())
		Expect(f1.Count).To(Equal(int64(5)))

		emptyF := NewFloat64SummaryStatistics()
		emptyF.Combine(f1)
		Expect(emptyF.Count).To(Equal(int64(5)))

		// Combine where other has smaller Min and larger Max
		combIntTarget := NewIntSummaryStatistics()
		combIntTarget.Accept(20)
		combIntOther := NewIntSummaryStatistics()
		combIntOther.Accept(5)
		combIntOther.Accept(50)
		combIntTarget.Combine(combIntOther)
		Expect(combIntTarget.Min).To(Equal(5))
		Expect(combIntTarget.Max).To(Equal(50))

		combFloatTarget := NewFloat64SummaryStatistics()
		combFloatTarget.Accept(20.0)
		combFloatOther := NewFloat64SummaryStatistics()
		combFloatOther.Accept(5.0)
		combFloatOther.Accept(50.0)
		combFloatTarget.Combine(combFloatOther)
		Expect(combFloatTarget.Min).To(Equal(5.0))
		Expect(combFloatTarget.Max).To(Equal(50.0))

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

		grouped := c.GroupingBy(func(x int) string {
			if x%2 == 0 {
				return "even"
			}
			return "odd"
		}, noCombinerDownstream)
		Expect(grouped.Combiner).To(BeNil())

		// Parallel stream falls back to sequential collect when Combiner is nil
		resGrouped := streamOf(1, 2, 3, 4).Parallel().Collect(grouped)
		Expect(resGrouped["even"]).To(Equal([]int{2, 4}))
		Expect(resGrouped["odd"]).To(Equal([]int{1, 3}))

		partitioned := c.PartitioningBy(func(x int) bool { return x%2 == 0 }, noCombinerDownstream)
		Expect(partitioned.Combiner).To(BeNil())

		resPartitioned := streamOf(1, 2, 3, 4).Parallel().Collect(partitioned)
		Expect(resPartitioned[true]).To(Equal([]int{2, 4}))
		Expect(resPartitioned[false]).To(Equal([]int{1, 3}))
	})

	It("supports Stream() method on streamList and streamSet", func() {
		listRes := streamOf(10, 20, 30).Collect(c.ToList[int]())
		streamer, ok := listRes.(interface{ Stream() Stream[int] })
		Expect(ok).To(BeTrue())
		Expect(streamer.Stream().ToSlice()).To(Equal([]int{10, 20, 30}))

		setRes := streamOf("x", "y").Collect(c.ToSet[string]())
		setStreamer, ok := setRes.(interface{ Stream() Stream[string] })
		Expect(ok).To(BeTrue())
		Expect(setStreamer.Stream().Count()).To(Equal(int64(2)))
	})

	It("supports parallel execution of all collectors (exercising combiners)", func() {
		// ToList
		list := streamRangeClosed(1, 100).Parallel().Collect(c.ToList[int]())
		Expect(list.Size()).To(Equal(100))

		// ToSlice
		slice := streamRangeClosed(1, 100).Parallel().Collect(c.ToSlice[int]())
		Expect(slice).To(HaveLen(100))

		// ToSet
		set := streamRangeClosed(1, 100).Parallel().Collect(c.ToSet[int]())
		Expect(set.Size()).To(Equal(100))

		// ToSetSlice
		setSlice := streamRangeClosed(1, 100).Parallel().Collect(c.ToSetSlice[int]())
		Expect(setSlice).To(HaveLen(100))

		// Joining
		joined := streamOf("a", "b", "c", "d").Parallel().Collect(c.Joining("-", "[", "]"))
		Expect(joined).To(ContainSubstring("a"))
		Expect(joined).To(HavePrefix("["))
		Expect(joined).To(HaveSuffix("]"))

		// Joining combiner edge cases
		jCollector := c.Joining("-", "<", ">")
		acc1 := jCollector.Supplier()
		acc2 := jCollector.Supplier()
		// both empty
		Expect(jCollector.Finisher(jCollector.Combiner(acc1, acc2))).To(Equal("<>"))
		// a empty, b not empty
		jCollector.Accumulator(acc2, "B")
		Expect(jCollector.Finisher(jCollector.Combiner(acc1, acc2))).To(Equal("<B>"))
		// b empty, a not empty
		acc3 := jCollector.Supplier()
		jCollector.Accumulator(acc3, "A")
		acc4 := jCollector.Supplier()
		Expect(jCollector.Finisher(jCollector.Combiner(acc3, acc4))).To(Equal("<A>"))

		// Counting
		cnt := streamRangeClosed(1, 100).Parallel().Collect(c.Counting[int]())
		Expect(cnt).To(Equal(int64(100)))

		// SummingInt & SummingFloat64
		sumI := streamRangeClosed(1, 100).Parallel().Collect(c.SummingInt(func(x int) int { return x }))
		Expect(sumI).To(Equal(int64(5050)))
		sumF := streamOf(1.5, 2.5, 3.0).Parallel().Collect(c.SummingFloat64(func(x float64) float64 { return x }))
		Expect(sumF).To(Equal(7.0))

		// AveragingInt & AveragingFloat64
		avgI := streamRangeClosed(1, 100).Parallel().Collect(c.AveragingInt(func(x int) int { return x }))
		Expect(avgI).To(Equal(50.5))
		avgF := streamOf(2.0, 4.0, 6.0).Parallel().Collect(c.AveragingFloat64(func(x float64) float64 { return x }))
		Expect(avgF).To(Equal(4.0))

		// Averaging on empty streams
		Expect(streamEmpty[int]().Collect(c.AveragingInt(func(x int) int { return x }))).To(Equal(0.0))
		Expect(streamEmpty[float64]().Collect(c.AveragingFloat64(func(x float64) float64 { return x }))).To(Equal(0.0))

		// SummarizingInt & SummarizingFloat64
		statsI := streamRangeClosed(1, 100).Parallel().Collect(c.SummarizingInt(func(x int) int { return x }))
		Expect(statsI.Count).To(Equal(int64(100)))
		Expect(statsI.Sum).To(Equal(int64(5050)))

		statsF := streamOf(1.0, 2.0, 3.0).Parallel().Collect(c.SummarizingFloat64(func(x float64) float64 { return x }))
		Expect(statsF.Count).To(Equal(int64(3)))
		Expect(statsF.Sum).To(Equal(6.0))

		// SummaryStatistics on empty
		emptyIntStats := NewIntSummaryStatistics()
		Expect(emptyIntStats.Average()).To(Equal(0.0))
		emptyFloatStats := NewFloat64SummaryStatistics()
		Expect(emptyFloatStats.Average()).To(Equal(0.0))

		// GroupingBy & GroupingBySlice
		grouped := streamRangeClosed(1, 100).Parallel().Collect(c.GroupingBy(func(x int) int { return x % 2 }, c.ToList[int]()))
		Expect(grouped[0].Size()).To(Equal(50))
		Expect(grouped[1].Size()).To(Equal(50))

		groupedSlice := streamRangeClosed(1, 100).Parallel().Collect(c.GroupingBySlice(func(x int) int { return x % 2 }))
		Expect(groupedSlice[0]).To(HaveLen(50))

		// PartitioningBy & PartitioningBySlice
		part := streamRangeClosed(1, 100).Parallel().Collect(c.PartitioningBy(func(x int) bool { return x%2 == 0 }, c.ToList[int]()))
		Expect(part[true].Size()).To(Equal(50))

		partSlice := streamRangeClosed(1, 100).Parallel().Collect(c.PartitioningBySlice(func(x int) bool { return x%2 == 0 }))
		Expect(partSlice[true]).To(HaveLen(50))

		// ToMap in parallel with merger
		toMap := streamOf("apple", "apricot", "banana", "blueberry").Parallel().Collect(
			c.ToMap(
				func(s string) string { return string(s[0]) },
				func(s string) string { return s },
				func(a, b string) string { return a + "," + b },
			),
		)
		Expect(toMap["a"]).To(ContainSubstring("apple"))
		Expect(toMap["b"]).To(ContainSubstring("banana"))

		// ToMap combiner duplicate key panic without merger
		toMapColl := c.ToMap(func(s string) string { return s }, func(s string) string { return s })
		Expect(func() {
			toMapColl.Combiner(map[string]string{"k": "v1"}, map[string]string{"k": "v2"})
		}).To(Panic())

		// Reducing in parallel
		red := streamRangeClosed(1, 10).Parallel().Collect(c.Reducing(0, func(a, b int) int { return a + b }))
		Expect(red).To(Equal(55))

		// MaxBy & MinBy in parallel
		maxVal := streamRangeClosed(1, 100).Parallel().Collect(c.MaxBy(func(a, b int) int { return a - b }))
		Expect(maxVal.IsPresent()).To(BeTrue())
		v, _ := maxVal.Get()
		Expect(v).To(Equal(100))

		minVal := streamRangeClosed(1, 100).Parallel().Collect(c.MinBy(func(a, b int) int { return a - b }))
		Expect(minVal.IsPresent()).To(BeTrue())
		vMin, _ := minVal.Get()
		Expect(vMin).To(Equal(1))

		// MaxBy & MinBy on empty stream
		Expect(streamEmpty[int]().Collect(c.MaxBy(func(a, b int) int { return a - b })).IsEmpty()).To(BeTrue())
		Expect(streamEmpty[int]().Collect(c.MinBy(func(a, b int) int { return a - b })).IsEmpty()).To(BeTrue())

		// MaxBy & MinBy combiner branches
		maxColl := c.MaxBy(func(a, b int) int { return a - b })
		optEmp1 := maxColl.Supplier()
		optEmp2 := maxColl.Supplier()
		optVal1 := maxColl.Supplier()
		*optVal1 = OptionalOf(10)
		optVal2 := maxColl.Supplier()
		*optVal2 = OptionalOf(20)

		Expect(maxColl.Combiner(optEmp1, optEmp2).IsEmpty()).To(BeTrue())
		Expect(*maxColl.Combiner(optEmp1, optVal2)).To(Equal(OptionalOf(20)))
		Expect(*maxColl.Combiner(optVal1, optEmp2)).To(Equal(OptionalOf(10)))
		Expect(*maxColl.Combiner(optVal1, optVal2)).To(Equal(OptionalOf(20)))
		Expect(*maxColl.Combiner(optVal2, optVal1)).To(Equal(OptionalOf(20)))

		minColl := c.MinBy(func(a, b int) int { return a - b })
		mOptEmp1 := minColl.Supplier()
		mOptEmp2 := minColl.Supplier()
		mOptVal1 := minColl.Supplier()
		*mOptVal1 = OptionalOf(10)
		mOptVal2 := minColl.Supplier()
		*mOptVal2 = OptionalOf(20)

		Expect(minColl.Combiner(mOptEmp1, mOptEmp2).IsEmpty()).To(BeTrue())
		Expect(*minColl.Combiner(mOptEmp1, mOptVal2)).To(Equal(OptionalOf(20)))
		Expect(*minColl.Combiner(mOptVal1, mOptEmp2)).To(Equal(OptionalOf(10)))
		Expect(*minColl.Combiner(mOptVal1, mOptVal2)).To(Equal(OptionalOf(10)))
		Expect(*minColl.Combiner(mOptVal2, mOptVal1)).To(Equal(OptionalOf(10)))

		// NewSimpleCollector with combiner
		simpleWithComb := NewSimpleCollector(
			func() []int { return nil },
			func(acc []int, item int) []int { return append(acc, item) },
			func(a, b []int) []int { return append(a, b...) },
		)
		Expect(simpleWithComb.Combiner([]int{1}, []int{2})).To(Equal([]int{1, 2}))

		// Joining with prefix only
		Expect(streamOf("a", "b").Collect(c.Joining(",", "["))).To(Equal("[a,b"))

		// Direct combiner invocations for 100% coverage
		sumIntColl := c.SummingInt(func(x int) int { return x })
		aI, bI := sumIntColl.Supplier(), sumIntColl.Supplier()
		*aI = 10
		*bI = 20
		Expect(*sumIntColl.Combiner(aI, bI)).To(Equal(int64(30)))

		sumFloatColl := c.SummingFloat64(func(x float64) float64 { return x })
		aF, bF := sumFloatColl.Supplier(), sumFloatColl.Supplier()
		*aF = 1.5
		*bF = 2.5
		Expect(*sumFloatColl.Combiner(aF, bF)).To(Equal(4.0))

		avgIntColl := c.AveragingInt(func(x int) int { return x })
		aAvg, bAvg := avgIntColl.Supplier(), avgIntColl.Supplier()
		avgIntColl.Accumulator(aAvg, 10)
		avgIntColl.Accumulator(bAvg, 20)
		Expect(avgIntColl.Finisher(avgIntColl.Combiner(aAvg, bAvg))).To(Equal(15.0))

		avgFColl := c.AveragingFloat64(func(x float64) float64 { return x })
		aAvgF, bAvgF := avgFColl.Supplier(), avgFColl.Supplier()
		avgFColl.Accumulator(aAvgF, 5.0)
		avgFColl.Accumulator(bAvgF, 15.0)
		Expect(avgFColl.Finisher(avgFColl.Combiner(aAvgF, bAvgF))).To(Equal(10.0))

		statsIColl := c.SummarizingInt(func(x int) int { return x })
		aStatsI, bStatsI := statsIColl.Supplier(), statsIColl.Supplier()
		aStatsI.Accept(10)
		bStatsI.Accept(20)
		combStatsI := statsIColl.Combiner(aStatsI, bStatsI)
		Expect(combStatsI.Count).To(Equal(int64(2)))
		Expect(combStatsI.Sum).To(Equal(int64(30)))

		statsFColl := c.SummarizingFloat64(func(x float64) float64 { return x })
		aStatsF, bStatsF := statsFColl.Supplier(), statsFColl.Supplier()
		aStatsF.Accept(10.5)
		bStatsF.Accept(20.5)
		combStatsF := statsFColl.Combiner(aStatsF, bStatsF)
		Expect(combStatsF.Count).To(Equal(int64(2)))
		Expect(combStatsF.Sum).To(Equal(31.0))

		redColl := c.Reducing(0, func(a, b int) int { return a + b })
		Expect(*redColl.Combiner(redColl.Accumulator(redColl.Supplier(), 10), redColl.Accumulator(redColl.Supplier(), 20))).To(Equal(30))

		partColl := c.PartitioningBy(func(x int) bool { return x > 0 }, c.ToSlice[int]())
		pm1, pm2 := partColl.Supplier(), partColl.Supplier()
		pm1[true] = []int{1}
		pm2[true] = []int{2}
		pComb := partColl.Combiner(pm1, pm2)
		Expect(pComb[true]).To(Equal([]int{1, 2}))

		groupByColl := c.GroupingBy(func(_ int) string { return "k" }, c.ToSlice[int]())
		gm1 := map[string][]int{"k1": {1}}
		gm2 := map[string][]int{"k1": {2}, "k2": {3}}
		gcomb := groupByColl.Combiner(gm1, gm2)
		Expect(gcomb["k1"]).To(Equal([]int{1, 2}))
		Expect(gcomb["k2"]).To(Equal([]int{3}))

		toMapWithMerger := c.ToMap(func(s string) string { return s }, func(s string) string { return s }, func(a, b string) string { return a + b })
		tm1 := map[string]string{"k1": "a"}
		tm2 := map[string]string{"k1": "b", "k2": "c"}
		tmComb := toMapWithMerger.Combiner(tm1, tm2)
		Expect(tmComb["k1"]).To(Equal("ab"))
		Expect(tmComb["k2"]).To(Equal("c"))

		setSliceColl := c.ToSetSlice[int]()
		Expect(setSliceColl.Combiner(map[int]struct{}{1: {}}, map[int]struct{}{2: {}})).To(HaveKey(2))

		setColl := c.ToSet[int]()
		sA, sB := setColl.Supplier(), setColl.Supplier()
		sA.Add(1)
		sB.Add(2)
		Expect(setColl.Combiner(sA, sB).Contains(2)).To(BeTrue())

		cntColl := c.Counting[int]()
		cntA, cntB := cntColl.Supplier(), cntColl.Supplier()
		*cntA = 5
		*cntB = 7
		Expect(*cntColl.Combiner(cntA, cntB)).To(Equal(int64(12)))

		jColl := c.Joining(",")
		ja1 := jColl.Supplier()
		jColl.Accumulator(ja1, "x")
		ja2 := jColl.Supplier()
		jColl.Accumulator(ja2, "y")
		Expect(jColl.Finisher(jColl.Combiner(ja1, ja2))).To(Equal("x,y"))
	})

	It("provides all collector factory methods on Collectors", func() {
		var coll Collectors
		Expect(coll.Joining(",").Finisher(coll.Joining(",").Accumulator(coll.Joining(",").Supplier(), "a"))).To(Equal("a"))
		Expect(*coll.Counting[int]().Supplier()).To(Equal(int64(0)))
		Expect(*coll.SummingInt(func(x int) int { return x }).Supplier()).To(Equal(int64(0)))
		Expect(*coll.SummingFloat64(func(x float64) float64 { return x }).Supplier()).To(Equal(0.0))
		Expect(coll.AveragingInt(func(x int) int { return x }).Finisher(coll.AveragingInt(func(x int) int { return x }).Supplier())).To(Equal(0.0))
		Expect(coll.AveragingFloat64(func(x float64) float64 { return x }).Finisher(coll.AveragingFloat64(func(x float64) float64 { return x }).Supplier())).To(Equal(0.0))
		Expect(coll.SummarizingInt(func(x int) int { return x }).Finisher(coll.SummarizingInt(func(x int) int { return x }).Supplier()).Count).To(Equal(int64(0)))
		Expect(coll.SummarizingFloat64(func(x float64) float64 { return x }).Finisher(coll.SummarizingFloat64(func(x float64) float64 { return x }).Supplier()).Count).To(Equal(int64(0)))
		Expect(coll.Mapping(func(x int) int { return x * 2 }, coll.ToSlice[int]()).Supplier()).To(Equal([]int{}))
		Expect(coll.Filtering(func(x int) bool { return x > 0 }, coll.ToSlice[int]()).Supplier()).To(Equal([]int{}))
		Expect(coll.FlatMapping(func(x int) []int { return []int{x} }, coll.ToSlice[int]()).Supplier()).To(Equal([]int{}))
		Expect(coll.GroupingBy(func(x int) int { return x }, coll.ToSlice[int]()).Supplier()).NotTo(BeNil())
		Expect(coll.GroupingBySlice(func(x int) int { return x }).Supplier()).NotTo(BeNil())
		Expect(coll.PartitioningBy(func(x int) bool { return x > 0 }, coll.ToSlice[int]()).Supplier()).NotTo(BeNil())
		Expect(coll.PartitioningBySlice(func(x int) bool { return x > 0 }).Supplier()).NotTo(BeNil())
		Expect(coll.Reducing(0, func(a, b int) int { return a + b }).Finisher(coll.Reducing(0, func(a, b int) int { return a + b }).Supplier())).To(Equal(0))
		Expect(coll.MaxBy(func(a, b int) int { return a - b }).Supplier().IsEmpty()).To(BeTrue())
		Expect(coll.MinBy(func(a, b int) int { return a - b }).Supplier().IsEmpty()).To(BeTrue())
	})
})
