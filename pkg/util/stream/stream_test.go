package stream_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/BeamFoundry/rays/pkg/util/stream"
)

var _ = Describe("Stream Lifecycle & Constructors", func() {
	It("creates an empty stream with StreamEmpty", func() {
		Expect(StreamEmpty[int]().ToSlice()).To(BeEmpty())
		Expect(StreamEmpty[int]().Count()).To(Equal(int64(0)))
		Expect(StreamEmpty[string]().ToSlice()).To(BeEmpty())
		Expect(StreamEmpty[string]().Count()).To(Equal(int64(0)))
	})

	It("creates a stream from StreamOf elements", func() {
		s := StreamOf(1, 2, 3)
		Expect(s.ToSlice()).To(Equal([]int{1, 2, 3}))
	})

	It("creates a stream from StreamOfSlice", func() {
		input := []string{"hello", "world"}
		s := StreamOfSlice(input)
		Expect(s.ToSlice()).To(Equal(input))
	})

	It("creates a stream from StreamOfChan", func() {
		ch := make(chan int, 3)
		ch <- 10
		ch <- 20
		ch <- 30
		close(ch)

		s := StreamOfChan(ch)
		Expect(s.ToSlice()).To(Equal([]int{10, 20, 30}))
	})

	It("creates a stream from StreamOfSeq", func() {
		seq := func(yield func(int) bool) {
			yield(100)
			yield(200)
		}
		s := StreamOfSeq(seq)
		Expect(s.ToSlice()).To(Equal([]int{100, 200}))
	})

	It("creates a stream from StreamOfCollection", func() {
		list := StreamOf(1, 2, 3).Collect(ToList[int]())
		s := StreamOfCollection(list)
		Expect(s.ToSlice()).To(Equal([]int{1, 2, 3}))

		nilStream := StreamOfCollection[int](nil)
		Expect(nilStream.Count()).To(Equal(int64(0)))
	})

	It("creates a stream from StreamGenerate", func() {
		i := 0
		s := StreamGenerate(func() int {
			i++
			return i
		}).Limit(5)

		Expect(s.ToSlice()).To(Equal([]int{1, 2, 3, 4, 5}))
	})

	It("creates a stream from StreamIterate", func() {
		s := StreamIterate(1, func(x int) int {
			return x * 2
		}).Limit(5)

		Expect(s.ToSlice()).To(Equal([]int{1, 2, 4, 8, 16}))
	})

	It("creates a stream from StreamIterateWithPredicate", func() {
		s := StreamIterateWithPredicate(1, func(x int) bool {
			return x <= 16
		}, func(x int) int {
			return x * 2
		})

		Expect(s.ToSlice()).To(Equal([]int{1, 2, 4, 8, 16}))
	})

	It("creates a stream from StreamRange and StreamRangeClosed", func() {
		Expect(StreamRange(1, 5).ToSlice()).To(Equal([]int{1, 2, 3, 4}))
		Expect(StreamRange(5, 5).ToSlice()).To(BeEmpty())
		Expect(StreamRange(6, 5).ToSlice()).To(BeEmpty())

		Expect(StreamRangeClosed(1, 5).ToSlice()).To(Equal([]int{1, 2, 3, 4, 5}))
		Expect(StreamRangeClosed(5, 5).ToSlice()).To(Equal([]int{5}))
		Expect(StreamRangeClosed(6, 5).ToSlice()).To(BeEmpty())
	})

	It("concatenates two streams with StreamConcat", func() {
		s1 := StreamOf(1, 2)
		s2 := StreamOf(3, 4, 5)
		s := StreamConcat(s1, s2)
		Expect(s.ToSlice()).To(Equal([]int{1, 2, 3, 4, 5}))
	})

	It("handles parallel and sequential flags", func() {
		s := StreamOf(1, 2, 3)
		Expect(s.IsParallel()).To(BeFalse())

		s = s.Parallel()
		Expect(s.IsParallel()).To(BeTrue())

		s = s.Sequential()
		Expect(s.IsParallel()).To(BeFalse())
	})

	It("executes OnClose handlers when closed or consumed", func() {
		closed1 := false
		closed2 := false

		s := StreamOf(1, 2, 3).
			OnClose(func() { closed1 = true }).
			OnClose(func() { closed2 = true })

		Expect(s.ToSlice()).To(Equal([]int{1, 2, 3}))
		Expect(closed1).To(BeTrue())
		Expect(closed2).To(BeTrue())
	})

	It("calling Close() multiple times is idempotent", func() {
		count := 0
		s := StreamOf(1, 2).OnClose(func() { count++ })
		s.Close()
		s.Close()
		Expect(count).To(Equal(1))
	})

	It("panics when operated upon more than once (single-use semantics)", func() {
		s := StreamOf(1, 2, 3)
		Expect(s.ToSlice()).To(Equal([]int{1, 2, 3}))

		Expect(func() {
			s.ToSlice()
		}).To(Panic())

		s2 := StreamOf(1, 2, 3)
		_ = s2.Filter(func(x int) bool { return x > 1 })
		Expect(func() {
			s2.Filter(func(x int) bool { return x > 2 })
		}).To(Panic())

		s3 := StreamOf(1, 2, 3)
		s3.Close()
		Expect(func() {
			s3.ToSlice()
		}).To(Panic())
	})
})
