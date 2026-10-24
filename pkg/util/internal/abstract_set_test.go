package internal

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AbstractSet", func() {
	It("creates an empty or populated AbstractSet", func() {
		s1 := NewAbstractSet[string]()
		Expect(s1.Size()).To(Equal(0))
		Expect(s1.IsEmpty()).To(BeTrue())

		s2 := NewAbstractSet("a", "b", "c", "a")
		Expect(s2.Size()).To(Equal(3))
		Expect(s2.IsEmpty()).To(BeFalse())
		Expect(s2.Contains("a")).To(BeTrue())
		Expect(s2.Contains("d")).To(BeFalse())
	})

	It("supports NewAbstractSetWithCapacity and negative capacity clamping", func() {
		s1 := NewAbstractSetWithCapacity[int](10)
		Expect(s1.Size()).To(Equal(0))

		s2 := NewAbstractSetWithCapacity[int](-5)
		Expect(s2.Size()).To(Equal(0))
	})

	It("supports Add, AddAll, Remove, RemoveAll, RetainAll, and Clear", func() {
		s := NewAbstractSet(1, 2)
		Expect(s.Add(3)).To(BeTrue())
		Expect(s.Add(3)).To(BeFalse())

		Expect(s.ContainsAll(nil)).To(BeFalse())
		Expect(s.AddAll(nil)).To(BeFalse())
		Expect(s.RemoveAll(nil)).To(BeFalse())
		Expect(s.RetainAll(nil)).To(BeFalse())

		other := NewAbstractSet(3, 4, 5)
		Expect(s.AddAll(other)).To(BeTrue())
		Expect(s.AddAll(other)).To(BeFalse())
		Expect(s.ContainsAll(other)).To(BeTrue())
		Expect(s.ContainsAll(NewAbstractSet(3, 99))).To(BeFalse())

		Expect(s.Remove(5)).To(BeTrue())
		Expect(s.Remove(99)).To(BeFalse())

		toRemove := NewAbstractSet(1, 4)
		Expect(s.RemoveAll(toRemove)).To(BeTrue())
		Expect(s.Contains(1)).To(BeFalse())
		Expect(s.Contains(4)).To(BeFalse())

		toRetain := NewAbstractSet(2, 99)
		Expect(s.RetainAll(toRetain)).To(BeTrue())
		Expect(s.Size()).To(Equal(1))
		Expect(s.Contains(2)).To(BeTrue())

		s.Clear()
		Expect(s.Size()).To(Equal(0))
		Expect(s.IsEmpty()).To(BeTrue())
	})

	It("supports ToSlice and Seq with early exit", func() {
		s := NewAbstractSet(10, 20)
		slice := s.ToSlice()
		Expect(slice).To(ConsistOf(10, 20))

		var collected []int
		for item := range s.Seq() {
			collected = append(collected, item)
		}
		Expect(collected).To(ConsistOf(10, 20))

		var partial []int
		for item := range s.Seq() {
			partial = append(partial, item)
			break
		}
		Expect(partial).To(HaveLen(1))
	})

	It("supports Equals", func() {
		s1 := NewAbstractSet(1, 2, 3)
		s2 := NewAbstractSet(1, 2, 3)
		s3 := NewAbstractSet(1, 2, 4)
		sDiffSize := NewAbstractSet(1, 2)

		Expect(s1.Equals(nil)).To(BeFalse())
		Expect(s1.Equals(sDiffSize)).To(BeFalse())
		Expect(s1.Equals(s3)).To(BeFalse())
		Expect(s1.Equals(s2)).To(BeTrue())
	})
})
