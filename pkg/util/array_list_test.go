package util_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/BeamFoundry/rays/pkg/core"
	. "github.com/BeamFoundry/rays/pkg/util"
)

var _ = Describe("ArrayList", func() {
	It("creates an empty or populated ArrayList", func() {
		l1 := NewArrayList[string]()
		Expect(l1.Size()).To(Equal(0))
		Expect(l1.IsEmpty()).To(BeTrue())

		l2 := NewArrayList("a", "b", "c")
		Expect(l2.Size()).To(Equal(3))
		Expect(l2.IsEmpty()).To(BeFalse())

		var iface List[string] = l2
		Expect(iface.Size()).To(Equal(3))
	})

	It("supports element access and modification by index", func() {
		l := NewArrayList("alpha", "beta", "gamma")

		Expect(l.Get(0)).To(Equal("alpha"))
		Expect(l.Get(1)).To(Equal("beta"))

		old := l.Set(1, "BETA")
		Expect(old).To(Equal("beta"))
		Expect(l.Get(1)).To(Equal("BETA"))

		Expect(func() { l.Get(-1) }).To(Panic())
		Expect(func() { l.Get(10) }).To(Panic())
		Expect(func() { l.Set(-1, "x") }).To(Panic())
	})

	It("supports Add and AddAt", func() {
		l := NewArrayList(1, 2)
		Expect(l.Add(3, 4)).To(BeTrue())
		Expect(l.ToSlice()).To(Equal([]int{1, 2, 3, 4}))

		l.AddAt(0, 0)
		Expect(l.ToSlice()).To(Equal([]int{0, 1, 2, 3, 4}))

		l.AddAt(5, 5)
		Expect(l.ToSlice()).To(Equal([]int{0, 1, 2, 3, 4, 5}))

		Expect(func() { l.AddAt(-1, 99) }).To(Panic())
		Expect(func() { l.AddAt(100, 99) }).To(Panic())
	})

	It("supports RemoveAt and Remove", func() {
		l := NewArrayList("x", "y", "z")

		removed := l.RemoveAt(1)
		Expect(removed).To(Equal("y"))
		Expect(l.ToSlice()).To(Equal([]string{"x", "z"}))

		Expect(l.Remove("z")).To(BeTrue())
		Expect(l.Remove("not-found")).To(BeFalse())
		Expect(l.ToSlice()).To(Equal([]string{"x"}))

		Expect(func() { l.RemoveAt(5) }).To(Panic())
	})

	It("supports IndexOf and LastIndexOf", func() {
		l := NewArrayList("a", "b", "c", "b", "a")
		Expect(l.IndexOf("b")).To(Equal(1))
		Expect(l.LastIndexOf("b")).To(Equal(3))
		Expect(l.IndexOf("missing")).To(Equal(-1))
		Expect(l.LastIndexOf("missing")).To(Equal(-1))
	})

	It("supports Contains and ContainsAll", func() {
		l := NewArrayList("one", "two", "three")
		Expect(l.Contains("two")).To(BeTrue())
		Expect(l.Contains("four")).To(BeFalse())

		other := NewArrayList("one", "three")
		Expect(l.ContainsAll(other)).To(BeTrue())

		notAll := NewArrayList("one", "four")
		Expect(l.ContainsAll(notAll)).To(BeFalse())
	})

	It("supports AddAll, RemoveAll, RetainAll, and Clear", func() {
		l := NewArrayList(1, 2, 3)
		extra := NewArrayList(4, 5)
		Expect(l.AddAll(extra)).To(BeTrue())
		Expect(l.ToSlice()).To(Equal([]int{1, 2, 3, 4, 5}))

		toRemove := NewArrayList(2, 4)
		Expect(l.RemoveAll(toRemove)).To(BeTrue())
		Expect(l.ToSlice()).To(Equal([]int{1, 3, 5}))

		toRetain := NewArrayList(1, 5, 9)
		Expect(l.RetainAll(toRetain)).To(BeTrue())
		Expect(l.ToSlice()).To(Equal([]int{1, 5}))

		l.Clear()
		Expect(l.Size()).To(Equal(0))
		Expect(l.IsEmpty()).To(BeTrue())
	})

	It("supports SubList", func() {
		l := NewArrayList("a", "b", "c", "d", "e")
		sub := l.SubList(1, 4)
		Expect(sub.ToSlice()).To(Equal([]string{"b", "c", "d"}))

		Expect(func() { l.SubList(-1, 2) }).To(Panic())
		Expect(func() { l.SubList(3, 2) }).To(Panic())
		Expect(func() { l.SubList(0, 10) }).To(Panic())
	})

	It("supports Seq iterator and Equals", func() {
		l1 := NewArrayList(10, 20, 30)
		l2 := NewArrayList(10, 20, 30)
		l3 := NewArrayList(10, 20, 40)

		Expect(l1.Equals(l2)).To(BeTrue())
		Expect(l1.Equals(l3)).To(BeFalse())

		var collected []int
		for item := range l1.Seq() {
			collected = append(collected, item)
		}
		Expect(collected).To(Equal([]int{10, 20, 30}))
	})

	It("supports Stream() method", func() {
		l := NewArrayList(1, 2, 3, 4, 5, 6)
		evensSquared := l.Stream().
			Filter(func(n int) bool { return n%2 == 0 }).
			Map(func(n int) int { return n * n }).
			ToSlice()

		Expect(evensSquared).To(Equal([]int{4, 16, 36}))
	})
})
