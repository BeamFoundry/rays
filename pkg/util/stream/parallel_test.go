package stream_test

import (
	"sync"
	"sync/atomic"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/BeamFoundry/rays/pkg/util/stream"
)

var _ = Describe("Parallel Streams", func() {
	It("executes ForEach in parallel across goroutines", func() {
		goroutines := sync.Map{}
		var count atomic.Int64

		StreamRangeClosed(1, 1000).
			Parallel().
			ForEach(func(x int) {
				count.Add(1)
				goroutines.Store(x%16, struct{}{})
			})

		Expect(count.Load()).To(Equal(int64(1000)))
	})

	It("collects to list in parallel preserving all elements", func() {
		res := StreamRangeClosed(1, 500).
			Parallel().
			Collect(ToList[int]())

		Expect(res.Size()).To(Equal(500))
		Expect(res.Contains(1)).To(BeTrue())
		Expect(res.Contains(250)).To(BeTrue())
		Expect(res.Contains(500)).To(BeTrue())
	})

	It("sums in parallel using SummingInt", func() {
		sum := StreamRangeClosed(1, 100).
			Parallel().
			Collect(SummingInt(func(x int) int { return x }))

		// 100 * 101 / 2 = 5050
		Expect(sum).To(Equal(int64(5050)))
	})

	It("summarizes in parallel using SummarizingInt", func() {
		stats := StreamRangeClosed(1, 100).
			Parallel().
			Collect(SummarizingInt(func(x int) int { return x }))

		Expect(stats.Count).To(Equal(int64(100)))
		Expect(stats.Sum).To(Equal(int64(5050)))
		Expect(stats.Min).To(Equal(1))
		Expect(stats.Max).To(Equal(100))
		Expect(stats.Average()).To(Equal(50.5))
	})

	It("groups in parallel using GroupingBy", func() {
		grouped := StreamRangeClosed(1, 100).
			Parallel().
			Collect(GroupingBy(
				func(x int) string {
					if x%2 == 0 {
						return "even"
					}
					return "odd"
				},
				Counting[int](),
			))

		Expect(grouped["even"]).To(Equal(int64(50)))
		Expect(grouped["odd"]).To(Equal(int64(50)))
	})

	It("evaluates AnyMatch, AllMatch, and NoneMatch in parallel", func() {
		anyEven := StreamRangeClosed(1, 1000).
			Parallel().
			AnyMatch(func(x int) bool { return x == 777 })
		Expect(anyEven).To(BeTrue())

		anyNotFound := StreamRangeClosed(1, 1000).
			Parallel().
			AnyMatch(func(x int) bool { return x == 9999 })
		Expect(anyNotFound).To(BeFalse())

		allPositive := StreamRangeClosed(1, 1000).
			Parallel().
			AllMatch(func(x int) bool { return x > 0 })
		Expect(allPositive).To(BeTrue())

		allEven := StreamRangeClosed(1, 1000).
			Parallel().
			AllMatch(func(x int) bool { return x%2 == 0 })
		Expect(allEven).To(BeFalse())

		noneNegative := StreamRangeClosed(1, 1000).
			Parallel().
			NoneMatch(func(x int) bool { return x < 0 })
		Expect(noneNegative).To(BeTrue())
	})

	It("finds any element in parallel", func() {
		found := StreamRangeClosed(1, 100).
			Parallel().
			FindAny()

		Expect(found.IsPresent()).To(BeTrue())
		val, _ := found.Get()
		Expect(val).To(BeNumerically(">=", 1))
		Expect(val).To(BeNumerically("<=", 100))
	})

	It("captures callback panics in parallel ForEach and rethrows in the calling goroutine", func() {
		Expect(func() {
			StreamRangeClosed(1, 100).
				Parallel().
				ForEach(func(x int) {
					if x == 50 {
						panic("worker-panic")
					}
				})
		}).To(PanicWith("worker-panic"))
	})

	It("closes producer channel when sequence panics in parallel ForEach", func() {
		panickingSeq := func(yield func(int) bool) {
			yield(1)
			yield(2)
			panic("seq-panic")
		}
		Expect(func() {
			StreamOfSeq(panickingSeq).
				Parallel().
				ForEach(func(x int) {})
		}).To(PanicWith("seq-panic"))
	})

	It("closes batch channel when sequence panics in parallel Collect", func() {
		panickingSeq := func(yield func(int) bool) {
			yield(1)
			yield(2)
			panic("seq-panic-collect")
		}
		Expect(func() {
			StreamOfSeq(panickingSeq).
				Parallel().
				Collect(ToList[int]())
		}).To(PanicWith("seq-panic-collect"))
	})
})
