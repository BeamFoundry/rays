package util_test

import (
	"encoding/json"

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

		sCap := NewHashSetWithCapacity[int](10)
		Expect(sCap.Size()).To(Equal(0))
		Expect(sCap.IsEmpty()).To(BeTrue())
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

	It("implements json.Marshaler and json.Unmarshaler", func() {
		s := NewHashSet("apple", "banana")
		data, err := json.Marshal(s)
		Expect(err).NotTo(HaveOccurred())
		var unmarshaledSlice []string
		Expect(json.Unmarshal(data, &unmarshaledSlice)).To(Succeed())
		Expect(unmarshaledSlice).To(ConsistOf("apple", "banana"))

		empty := NewHashSet[int]()
		dataEmpty, err := json.Marshal(empty)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(dataEmpty)).To(Equal(`[]`))

		var nilSet *HashSet[int]
		dataNil, err := nilSet.MarshalJSON()
		Expect(err).NotTo(HaveOccurred())
		Expect(string(dataNil)).To(Equal(`[]`))

		var unmarshaled HashSet[int]
		Expect(json.Unmarshal([]byte(`[1, 2, 2, 3]`), &unmarshaled)).To(Succeed())
		Expect(unmarshaled.Size()).To(Equal(3))
		Expect(unmarshaled.Contains(1)).To(BeTrue())
		Expect(unmarshaled.Contains(2)).To(BeTrue())
		Expect(unmarshaled.Contains(3)).To(BeTrue())

		Expect(json.Unmarshal([]byte(`null`), &unmarshaled)).To(Succeed())
		Expect(unmarshaled.Size()).To(Equal(0))

		Expect(nilSet.UnmarshalJSON([]byte(`[]`))).NotTo(Succeed())
		Expect(unmarshaled.UnmarshalJSON([]byte(`invalid`))).NotTo(Succeed())
	})

	It("implements driver.Valuer and sql.Scanner", func() {
		s := NewHashSet(10, 20)
		val, err := s.Value()
		Expect(err).NotTo(HaveOccurred())
		var valSlice []int
		Expect(json.Unmarshal([]byte(val.(string)), &valSlice)).To(Succeed())
		Expect(valSlice).To(ConsistOf(10, 20))

		var nilSet *HashSet[int]
		nilVal, err := nilSet.Value()
		Expect(err).NotTo(HaveOccurred())
		Expect(nilVal).To(BeNil())

		scanned := NewHashSet[int]()
		Expect(scanned.Scan(`[1,2,2,3]`)).To(Succeed())
		Expect(scanned.Size()).To(Equal(3))
		Expect(scanned.Contains(1)).To(BeTrue())
		Expect(scanned.Contains(2)).To(BeTrue())
		Expect(scanned.Contains(3)).To(BeTrue())

		Expect(scanned.Scan([]byte(`[4,5]`))).To(Succeed())
		Expect(scanned.Size()).To(Equal(2))
		Expect(scanned.Contains(4)).To(BeTrue())
		Expect(scanned.Contains(5)).To(BeTrue())

		Expect(scanned.Scan(nil)).To(Succeed())
		Expect(scanned.Size()).To(Equal(0))

		Expect(scanned.Scan("")).To(Succeed())
		Expect(scanned.Size()).To(Equal(0))

		Expect(scanned.Scan(12345)).NotTo(Succeed())
		Expect(nilSet.Scan(`[]`)).NotTo(Succeed())

		var zeroSet HashSet[string]
		Expect(zeroSet.Scan(`["hello", "world"]`)).To(Succeed())
		Expect(zeroSet.Size()).To(Equal(2))
		Expect(zeroSet.Contains("hello")).To(BeTrue())
		Expect(zeroSet.Contains("world")).To(BeTrue())
	})

	It("works as an empty set from its zero value", func() {
		var s HashSet[int]
		Expect(s.Size()).To(Equal(0))
		Expect(s.IsEmpty()).To(BeTrue())
		Expect(s.Contains(1)).To(BeFalse())
		Expect(s.ToSlice()).To(BeEmpty())

		var collected []int
		for item := range s.Seq() {
			collected = append(collected, item)
		}
		Expect(collected).To(BeEmpty())

		Expect(s.Add(1, 2)).To(BeTrue())
		Expect(s.Size()).To(Equal(2))
		Expect(s.Contains(1)).To(BeTrue())

		var s2 HashSet[int]
		Expect(s2.AddAll(NewHashSet(10, 20))).To(BeTrue())
		Expect(s2.Size()).To(Equal(2))
		Expect(s2.Contains(10)).To(BeTrue())
	})
})
