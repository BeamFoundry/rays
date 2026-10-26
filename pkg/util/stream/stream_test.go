package stream_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/BeamFoundry/rays/pkg/util/internal"
	. "github.com/BeamFoundry/rays/pkg/util/stream"
)

var _ = Describe("Stream Lifecycle & Constructors", func() {
	var s Stream[int]
	var sStr Stream[string]

	It("creates an empty stream with Stream.FromEmpty", func() {
		Expect(s.FromEmpty().ToSlice()).To(BeEmpty())
		Expect(s.FromEmpty().Count()).To(Equal(int64(0)))
		Expect(sStr.FromEmpty().ToSlice()).To(BeEmpty())
		Expect(sStr.FromEmpty().Count()).To(Equal(int64(0)))
	})

	It("creates a stream from Stream.From elements", func() {
		st := s.From(1, 2, 3)
		Expect(st.ToSlice()).To(Equal([]int{1, 2, 3}))
	})

	It("creates a stream from Stream.FromSlice", func() {
		input := []string{"hello", "world"}
		st := sStr.FromSlice(input)
		Expect(st.ToSlice()).To(Equal(input))
	})

	It("creates a stream from Stream.FromChan", func() {
		ch := make(chan int, 3)
		ch <- 10
		ch <- 20
		ch <- 30
		close(ch)

		st := s.FromChan(ch)
		Expect(st.ToSlice()).To(Equal([]int{10, 20, 30}))

		ch2 := make(chan int, 3)
		ch2 <- 1
		ch2 <- 2
		ch2 <- 3
		close(ch2)
		Expect(s.FromChan(ch2).Limit(1).ToSlice()).To(Equal([]int{1}))
	})

	It("creates a stream from Stream.FromSeq", func() {
		seq := func(yield func(int) bool) {
			yield(100)
			yield(200)
		}
		st := s.FromSeq(seq)
		Expect(st.ToSlice()).To(Equal([]int{100, 200}))
	})

	It("creates a stream from Stream.FromCollection", func() {
		list := s.From(1, 2, 3).Collect(Collectors{}.ToList[int]())
		st := s.FromCollection(list)
		Expect(st.ToSlice()).To(Equal([]int{1, 2, 3}))

		nilStream := s.FromCollection(nil)
		Expect(nilStream.Count()).To(Equal(int64(0)))

		var typedNilList *internal.AbstractList[int]
		Expect(s.FromCollection(typedNilList).Count()).To(Equal(int64(0)))

		var typedNilSet *internal.AbstractSet[int]
		Expect(s.FromCollection(typedNilSet).Count()).To(Equal(int64(0)))
	})

	It("creates a stream from Stream.FromGenerate", func() {
		i := 0
		st := s.FromGenerate(func() int {
			i++
			return i
		}).Limit(5)

		Expect(st.ToSlice()).To(Equal([]int{1, 2, 3, 4, 5}))
	})

	It("creates a stream from Stream.FromIterate", func() {
		st := s.FromIterate(1, func(x int) int {
			return x * 2
		}).Limit(5)

		Expect(st.ToSlice()).To(Equal([]int{1, 2, 4, 8, 16}))
	})

	It("creates a stream from Stream.FromIterateWithPredicate", func() {
		st := s.FromIterateWithPredicate(1, func(x int) bool {
			return x <= 16
		}, func(x int) int {
			return x * 2
		})

		Expect(st.ToSlice()).To(Equal([]int{1, 2, 4, 8, 16}))

		// When seed fails predicate
		Expect(s.FromIterateWithPredicate(100, func(x int) bool { return x < 50 }, func(x int) int { return x + 1 }).ToSlice()).To(BeEmpty())
		// Early break
		Expect(s.FromIterateWithPredicate(1, func(x int) bool { return x < 100 }, func(x int) int { return x + 1 }).Limit(2).ToSlice()).To(Equal([]int{1, 2}))
	})

	It("creates a stream from Stream.FromRange and FromRangeClosed", func() {
		Expect(s.FromRange(1, 5).ToSlice()).To(Equal([]int{1, 2, 3, 4}))
		Expect(s.FromRange(5, 5).ToSlice()).To(BeEmpty())
		Expect(s.FromRange(6, 5).ToSlice()).To(BeEmpty())

		Expect(s.FromRangeClosed(1, 5).ToSlice()).To(Equal([]int{1, 2, 3, 4, 5}))
		Expect(s.FromRangeClosed(5, 5).ToSlice()).To(Equal([]int{5}))
		Expect(s.FromRangeClosed(6, 5).ToSlice()).To(BeEmpty())
	})

	It("concatenates two streams with Stream.FromConcat", func() {
		s1 := s.From(1, 2)
		s2 := s.From(3, 4, 5)
		st := s.FromConcat(s1, s2)
		Expect(st.ToSlice()).To(Equal([]int{1, 2, 3, 4, 5}))

		// Early exit during first stream
		cEarly1 := s.FromConcat(s.From(1, 2), s.From(3, 4)).Limit(1).ToSlice()
		Expect(cEarly1).To(Equal([]int{1}))

		// Early exit during second stream
		cEarly2 := s.FromConcat(s.From(1, 2), s.From(3, 4)).Limit(3).ToSlice()
		Expect(cEarly2).To(Equal([]int{1, 2, 3}))

		// Both stream close handlers execute
		var closedA, closedB bool
		sA := s.From(1).OnClose(func() { closedA = true })
		sB := s.From(2).OnClose(func() { closedB = true })
		concat := s.FromConcat(sA, sB)
		concat.Close()
		Expect(closedA).To(BeTrue())
		Expect(closedB).To(BeTrue())
	})

	It("handles parallel and sequential flags", func() {
		st := Stream[int]{}.From(1, 2, 3)
		Expect(st.IsParallel()).To(BeFalse())

		st = st.Parallel()
		Expect(st.IsParallel()).To(BeTrue())

		st = st.Sequential()
		Expect(st.IsParallel()).To(BeFalse())
	})

	It("executes OnClose handlers when closed or consumed", func() {
		closed1 := false
		closed2 := false

		st := Stream[int]{}.From(1, 2, 3).
			OnClose(func() { closed1 = true }).
			OnClose(func() { closed2 = true })

		Expect(st.ToSlice()).To(Equal([]int{1, 2, 3}))
		Expect(closed1).To(BeTrue())
		Expect(closed2).To(BeTrue())
	})

	It("calling Close() multiple times is idempotent", func() {
		count := 0
		st := Stream[int]{}.From(1, 2).OnClose(func() { count++ })
		st.Close()
		st.Close()
		Expect(count).To(Equal(1))
	})

	It("executes shared OnClose handlers once when closing both source and derived streams", func() {
		count := 0
		source := Stream[int]{}.From(1, 2, 3).OnClose(func() { count++ })
		derived := source.Filter(func(x int) bool { return x > 1 })
		source.Close()
		derived.Close()
		Expect(count).To(Equal(1))
	})

	It("panics when operated upon more than once (single-use semantics)", func() {
		st := Stream[int]{}.From(1, 2, 3)
		Expect(st.ToSlice()).To(Equal([]int{1, 2, 3}))

		Expect(func() {
			st.ToSlice()
		}).To(Panic())

		st2 := Stream[int]{}.From(1, 2, 3)
		_ = st2.Filter(func(x int) bool { return x > 1 })
		Expect(func() {
			st2.Filter(func(x int) bool { return x > 2 })
		}).To(Panic())

		st3 := Stream[int]{}.From(1, 2, 3)
		st3.Close()
		Expect(func() {
			st3.ToSlice()
		}).To(Panic())
	})
})
