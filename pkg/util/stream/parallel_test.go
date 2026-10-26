package stream_test

import (
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/BeamFoundry/rays/pkg/core"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/BeamFoundry/rays/pkg/util/stream"
)

var _ = Describe("Parallel Streams", func() {
	It("executes ForEach in parallel across goroutines", func() {
		goroutines := sync.Map{}
		var count atomic.Int64

		streamRangeClosed(1, 1000).
			Parallel().
			ForEach(func(x int) {
				count.Add(1)
				goroutines.Store(x%16, struct{}{})
			})

		Expect(count.Load()).To(Equal(int64(1000)))
	})

	It("collects to list in parallel preserving all elements", func() {
		res := streamRangeClosed(1, 500).
			Parallel().
			Collect(Collectors{}.ToList[int]())

		Expect(res.Size()).To(Equal(500))
		Expect(res.Contains(1)).To(BeTrue())
		Expect(res.Contains(250)).To(BeTrue())
		Expect(res.Contains(500)).To(BeTrue())
	})

	It("preserves encounter order across independently accumulated batches", func() {
		previous := runtime.GOMAXPROCS(2)
		DeferCleanup(func() { runtime.GOMAXPROCS(previous) })
		collector := Collectors{}.ToList[int]()
		accumulate := collector.Accumulator
		secondBatchStarted := make(chan struct{})
		thirdBatchStarted := make(chan struct{})
		collector.Accumulator = func(acc core.List[int], item int) core.List[int] {
			switch item {
			case 1:
				<-secondBatchStarted
			case 65:
				close(secondBatchStarted)
				<-thirdBatchStarted
			case 129:
				close(thirdBatchStarted)
			default:
			}
			return accumulate(acc, item)
		}
		expected := streamRangeClosed(1, 1025).ToSlice()
		result := streamOfSlice(expected).Parallel().Collect(collector)
		Expect(streamOfCollection(result).ToSlice()).To(Equal(expected))
	})

	It("preserves Joining order across full and partial batches", func() {
		previous := runtime.GOMAXPROCS(2)
		DeferCleanup(func() { runtime.GOMAXPROCS(previous) })
		items := streamRangeClosed(1, 1025).Map(strconv.Itoa).ToSlice()
		Expect(streamOfSlice(items).Parallel().Collect(Collectors{}.Joining(",", "[", "]"))).To(Equal("[" + strings.Join(items, ",") + "]"))
		Expect(streamEmpty[string]().Parallel().Collect(Collectors{}.Joining(",", "[", "]"))).To(Equal("[]"))
	})

	DescribeTable("recovers worker panics, drains input, and closes the stream",
		func(operation func(Stream[int])) {
			previous := runtime.GOMAXPROCS(2)
			DeferCleanup(func() { runtime.GOMAXPROCS(previous) })
			closed := false
			stream := streamRangeClosed(1, 4096).Parallel().OnClose(func() { closed = true })
			Expect(func() { operation(stream) }).To(PanicWith("worker-panic"))
			Expect(closed).To(BeTrue())
		},
		Entry("ForEach", func(s Stream[int]) { s.ForEach(func(int) { panic("worker-panic") }) }),
		Entry("AnyMatch", func(s Stream[int]) { s.AnyMatch(func(int) bool { panic("worker-panic") }) }),
		Entry("AllMatch", func(s Stream[int]) { s.AllMatch(func(int) bool { panic("worker-panic") }) }),
		Entry("Collect accumulator", func(s Stream[int]) {
			s.Collect(Collectors{}.SummingInt(func(int) int { panic("worker-panic") }))
		}),
		Entry("Collect supplier", func(s Stream[int]) {
			collector := Collectors{}.ToSlice[int]()
			collector.Supplier = func() []int { panic("worker-panic") }
			s.Collect(collector)
		}),
	)

	DescribeTable("closes channels and waits when a match producer panics",
		func(operation func(Stream[int])) {
			stream := streamOfSeq(func(yield func(int) bool) {
				yield(1)
				panic("seq-panic")
			}).Parallel()
			Expect(func() { operation(stream) }).To(PanicWith("seq-panic"))
		},
		Entry("AnyMatch", func(s Stream[int]) { s.AnyMatch(func(int) bool { return false }) }),
		Entry("AllMatch", func(s Stream[int]) { s.AllMatch(func(int) bool { return true }) }),
	)

	It("preserves encounter order across batches for ordered collectors", func() {
		previous := runtime.GOMAXPROCS(2)
		DeferCleanup(runtime.GOMAXPROCS, previous)

		for _, size := range []int{0, 1, 63, 64, 65, 1025} {
			expected := streamRange(0, size).ToSlice()
			Expect(streamRange(0, size).Parallel().Collect(Collectors{}.ToSlice[int]())).To(Equal(expected))
			Expect(streamRange(0, size).Parallel().Collect(Collectors{}.ToList[int]()).ToSlice()).To(Equal(expected))
			words := streamOfSlice(expected).Map(strconv.Itoa).ToSlice()
			Expect(streamOfSlice(words).Parallel().Collect(Collectors{}.Joining(",", "[", "]"))).
				To(Equal("[" + strings.Join(words, ",") + "]"))
		}
	})

	It("preserves encounter order when a worker processes later batches first", func() {
		previous := runtime.GOMAXPROCS(2)
		DeferCleanup(runtime.GOMAXPROCS, previous)

		laterBatch := make(chan struct{})
		collector := Collectors{}.Mapping(func(x int) int {
			if x == 0 {
				<-laterBatch
			}
			if x == 128 {
				close(laterBatch)
			}
			return x
		}, Collectors{}.ToSlice[int]())
		Expect(streamRange(0, 1025).Parallel().Collect(collector)).
			To(Equal(streamRange(0, 1025).ToSlice()))
	})

	DescribeTable("recovers worker panics and drains input before closing the stream",
		func(operation func(Stream[int])) {
			previous := runtime.GOMAXPROCS(2)
			DeferCleanup(runtime.GOMAXPROCS, previous)

			closes := 0
			seq := func(yield func(int) bool) {
				for i := range 10000 {
					if !yield(i) {
						return
					}
				}
			}
			s := streamOfSeq(seq).Parallel().OnClose(func() { closes++ })
			Expect(func() { operation(s) }).To(PanicWith("worker-panic"))
			Expect(closes).To(Equal(1))
		},
		Entry("ForEach", func(s Stream[int]) {
			s.ForEach(func(int) { panic("worker-panic") })
		}),
		Entry("AnyMatch", func(s Stream[int]) {
			s.AnyMatch(func(int) bool { panic("worker-panic") })
		}),
		Entry("AllMatch", func(s Stream[int]) {
			s.AllMatch(func(int) bool { panic("worker-panic") })
		}),
		Entry("Collect accumulator", func(s Stream[int]) {
			collector := Collectors{}.ToSlice[int]()
			collector.Accumulator = func([]int, int) []int { panic("worker-panic") }
			s.Collect(collector)
		}),
		Entry("Collect supplier", func(s Stream[int]) {
			collector := Collectors{}.ToSlice[int]()
			collector.Supplier = func() []int { panic("worker-panic") }
			s.Collect(collector)
		}),
	)

	DescribeTable("closes input and waits for workers when the producer panics",
		func(operation func(Stream[int], *atomic.Int64)) {
			var processed atomic.Int64
			closes := 0
			seq := func(yield func(int) bool) {
				for i := range 128 {
					yield(i)
				}
				panic("producer-panic")
			}
			s := streamOfSeq(seq).Parallel().OnClose(func() { closes++ })
			Expect(func() { operation(s, &processed) }).To(PanicWith("producer-panic"))
			Expect(processed.Load()).To(Equal(int64(128)))
			Expect(closes).To(Equal(1))
		},
		Entry("AnyMatch", func(s Stream[int], processed *atomic.Int64) {
			s.AnyMatch(func(int) bool { processed.Add(1); return false })
		}),
		Entry("AllMatch", func(s Stream[int], processed *atomic.Int64) {
			s.AllMatch(func(int) bool { processed.Add(1); return true })
		}),
		Entry("Collect", func(s Stream[int], processed *atomic.Int64) {
			s.Collect(Collectors{}.Mapping(func(x int) int { processed.Add(1); return x }, Collectors{}.ToSlice[int]()))
		}),
	)

	It("sums in parallel using SummingInt", func() {
		sum := streamRangeClosed(1, 100).
			Parallel().
			Collect(Collectors{}.SummingInt(func(x int) int { return x }))

		// 100 * 101 / 2 = 5050
		Expect(sum).To(Equal(int64(5050)))
	})

	It("summarizes in parallel using SummarizingInt", func() {
		stats := streamRangeClosed(1, 100).
			Parallel().
			Collect(Collectors{}.SummarizingInt(func(x int) int { return x }))

		Expect(stats.Count).To(Equal(int64(100)))
		Expect(stats.Sum).To(Equal(int64(5050)))
		Expect(stats.Min).To(Equal(1))
		Expect(stats.Max).To(Equal(100))
		Expect(stats.Average()).To(Equal(50.5))
	})

	It("groups in parallel using GroupingBy", func() {
		grouped := streamRangeClosed(1, 100).
			Parallel().
			Collect(Collectors{}.GroupingBy(
				func(x int) string {
					if x%2 == 0 {
						return "even"
					}
					return "odd"
				},
				Collectors{}.Counting[int](),
			))

		Expect(grouped["even"]).To(Equal(int64(50)))
		Expect(grouped["odd"]).To(Equal(int64(50)))
	})

	It("evaluates AnyMatch, AllMatch, and NoneMatch in parallel", func() {
		anyEven := streamRangeClosed(1, 1000).
			Parallel().
			AnyMatch(func(x int) bool { return x == 777 })
		Expect(anyEven).To(BeTrue())

		anyNotFound := streamRangeClosed(1, 1000).
			Parallel().
			AnyMatch(func(x int) bool { return x == 9999 })
		Expect(anyNotFound).To(BeFalse())

		allPositive := streamRangeClosed(1, 1000).
			Parallel().
			AllMatch(func(x int) bool { return x > 0 })
		Expect(allPositive).To(BeTrue())

		allEven := streamRangeClosed(1, 1000).
			Parallel().
			AllMatch(func(x int) bool { return x%2 == 0 })
		Expect(allEven).To(BeFalse())

		noneNegative := streamRangeClosed(1, 1000).
			Parallel().
			NoneMatch(func(x int) bool { return x < 0 })
		Expect(noneNegative).To(BeTrue())
	})

	It("finds any element in parallel", func() {
		found := streamRangeClosed(1, 100).
			Parallel().
			FindAny()

		Expect(found.IsPresent()).To(BeTrue())
		val, _ := found.Get()
		Expect(val).To(BeNumerically(">=", 1))
		Expect(val).To(BeNumerically("<=", 100))
	})

	It("captures callback panics in parallel ForEach and rethrows in the calling goroutine", func() {
		Expect(func() {
			streamRangeClosed(1, 100).
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
			streamOfSeq(panickingSeq).
				Parallel().
				ForEach(func(_ int) {})
		}).To(PanicWith("seq-panic"))
	})

	It("closes batch channel when sequence panics in parallel Collect", func() {
		panickingSeq := func(yield func(int) bool) {
			yield(1)
			yield(2)
			panic("seq-panic-collect")
		}
		Expect(func() {
			streamOfSeq(panickingSeq).
				Parallel().
				Collect(Collectors{}.ToList[int]())
		}).To(PanicWith("seq-panic-collect"))
	})

	It("stops producing items when upstream sequence ignores early stop", func() {
		sloppySeq := func(yield func(int) bool) {
			for i := range 1000 {
				yield(i)
				if i%20 == 0 {
					time.Sleep(20 * time.Microsecond)
				}
			}
		}

		// ForEach with worker panic
		var panicOnce atomic.Bool
		Expect(func() {
			streamOfSeq(sloppySeq).Parallel().ForEach(func(_ int) {
				if panicOnce.CompareAndSwap(false, true) {
					panic("foreach-worker-panic")
				}
			})
		}).To(Panic())

		// AnyMatch with early match
		matched := streamOfSeq(sloppySeq).Parallel().AnyMatch(func(n int) bool {
			return n == 0
		})
		Expect(matched).To(BeTrue())

		// AllMatch with early mismatch
		all := streamOfSeq(sloppySeq).Parallel().AllMatch(func(n int) bool {
			return n > 0
		})
		Expect(all).To(BeFalse())

		// FindAny with found item
		anyFound := streamOfSeq(sloppySeq).Parallel().FindAny()
		Expect(anyFound.IsPresent()).To(BeTrue())

		// Collect with worker panic
		var collectPanicOnce atomic.Bool
		collectCollector := Collectors{}.ToList[int]()
		collectCollector.Accumulator = func(acc core.List[int], _ int) core.List[int] {
			if collectPanicOnce.CompareAndSwap(false, true) {
				panic("collect-producer-stop-panic")
			}
			return acc
		}
		Expect(func() {
			streamOfSeq(sloppySeq).Parallel().Collect(collectCollector)
		}).To(Panic())
	})

	It("skips processing queued batches after a worker panic in parallel Collect", func() {
		previous := runtime.GOMAXPROCS(128)
		DeferCleanup(func() { runtime.GOMAXPROCS(previous) })
		var panicOnce atomic.Bool
		collector := Collectors{}.ToList[int]()
		collector.Accumulator = func(acc core.List[int], item int) core.List[int] {
			if item > 1000 && panicOnce.CompareAndSwap(false, true) {
				panic("collect-worker-panic")
			}
			return acc
		}
		Expect(func() {
			streamRangeClosed(0, 50000).Parallel().Collect(collector)
		}).To(PanicWith("collect-worker-panic"))
	})

	It("skips processing queued items after a worker panic in parallel ForEach", func() {
		previous := runtime.GOMAXPROCS(128)
		DeferCleanup(func() { runtime.GOMAXPROCS(previous) })
		var panicOnce atomic.Bool
		Expect(func() {
			streamRangeClosed(0, 50000).Parallel().ForEach(func(n int) {
				if n > 1000 && panicOnce.CompareAndSwap(false, true) {
					panic("foreach-worker-panic")
				}
			})
		}).To(PanicWith("foreach-worker-panic"))
	})
})
