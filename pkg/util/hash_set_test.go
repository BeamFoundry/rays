package util_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/BeamFoundry/rays/pkg/core"
	. "github.com/BeamFoundry/rays/pkg/util"
)

var _ = Describe("HashSet", func() {
	It("creates an empty or populated HashSet", func() {
		s1 := NewHashSet[string]()
		Expect(s1.Size()).To(Equal(0))
		Expect(s1.IsEmpty()).To(BeTrue())

		s2 := NewHashSet("a", "b", "c", "a")
		Expect(s2.Size()).To(Equal(3))
		Expect(s2.IsEmpty()).To(BeFalse())

		var iface Set[string] = s2
		Expect(iface.Size()).To(Equal(3))
	})

	It("supports Add, Contains, and Remove", func() {
		s := NewHashSet[int]()
		Expect(s.Add(1)).To(BeTrue())
		Expect(s.Add(1)).To(BeFalse())
		Expect(s.Add(2, 3)).To(BeTrue())

		Expect(s.Contains(1)).To(BeTrue())
		Expect(s.Contains(2)).To(BeTrue())
		Expect(s.Contains(99)).To(BeFalse())

		Expect(s.Remove(2)).To(BeTrue())
		Expect(s.Remove(2)).To(BeFalse())
		Expect(s.Contains(2)).To(BeFalse())
		Expect(s.Size()).To(Equal(2))
	})

	It("supports ContainsAll, AddAll, RemoveAll, RetainAll, and Clear", func() {
		s := NewHashSet(1, 2, 3)
		other := NewHashSet(2, 3)
		Expect(s.ContainsAll(other)).To(BeTrue())

		notContained := NewHashSet(3, 4)
		Expect(s.ContainsAll(notContained)).To(BeFalse())

		Expect(s.AddAll(notContained)).To(BeTrue())
		Expect(s.Size()).To(Equal(4))

		Expect(s.RemoveAll(NewHashSet(1, 4))).To(BeTrue())
		Expect(s.Size()).To(Equal(2))
		Expect(s.Contains(1)).To(BeFalse())
		Expect(s.Contains(4)).To(BeFalse())

		Expect(s.RetainAll(NewHashSet(2))).To(BeTrue())
		Expect(s.Size()).To(Equal(1))
		Expect(s.Contains(2)).To(BeTrue())

		s.Clear()
		Expect(s.Size()).To(Equal(0))
		Expect(s.IsEmpty()).To(BeTrue())
	})

	It("supports ToSlice, Seq, and Equals", func() {
		s1 := NewHashSet("x", "y")
		s2 := NewHashSet("y", "x")
		s3 := NewHashSet("x", "z")

		Expect(s1.Equals(s2)).To(BeTrue())
		Expect(s1.Equals(s3)).To(BeFalse())

		slice := s1.ToSlice()
		Expect(slice).To(ConsistOf("x", "y"))

		var seqItems []string
		for item := range s1.Seq() {
			seqItems = append(seqItems, item)
		}
		Expect(seqItems).To(ConsistOf("x", "y"))
	})

	It("supports Stream() method", func() {
		s := NewHashSet(10, 20, 30, 40)
		count := s.Stream().
			Filter(func(n int) bool { return n > 15 }).
			Count()

		Expect(count).To(Equal(int64(3)))
	})
})
