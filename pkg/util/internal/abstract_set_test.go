package internal

import (
	"encoding/json"
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type failJSONKey struct {
	A int
}

func (f failJSONKey) MarshalJSON() ([]byte, error) {
	return nil, errors.New("cannot marshal failJSONKey")
}

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

		list := NewAbstractList(1, 2, 3)
		Expect(s1.Equals(list)).To(BeFalse())
	})

	It("handles typed-nil collection operands across Equals, ContainsAll, AddAll, RemoveAll, and RetainAll", func() {
		s := NewAbstractSet(1, 2, 3)
		var typedNil *AbstractSet[int]

		Expect(s.ContainsAll(typedNil)).To(BeFalse())
		Expect(s.AddAll(typedNil)).To(BeFalse())
		Expect(s.RemoveAll(typedNil)).To(BeFalse())
		Expect(s.RetainAll(typedNil)).To(BeFalse())
		Expect(s.Equals(typedNil)).To(BeFalse())
	})

	It("implements json.Marshaler and json.Unmarshaler", func() {
		s := NewAbstractSet("apple", "banana")
		data, err := json.Marshal(s)
		Expect(err).NotTo(HaveOccurred())
		var unmarshaledSlice []string
		Expect(json.Unmarshal(data, &unmarshaledSlice)).To(Succeed())
		Expect(unmarshaledSlice).To(ConsistOf("apple", "banana"))

		empty := NewAbstractSet[int]()
		dataEmpty, err := json.Marshal(empty)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(dataEmpty)).To(Equal(`[]`))

		var nilSet *AbstractSet[int]
		dataNil, err := nilSet.MarshalJSON()
		Expect(err).NotTo(HaveOccurred())
		Expect(string(dataNil)).To(Equal(`[]`))

		var unmarshaled AbstractSet[int]
		Expect(json.Unmarshal([]byte(`[1, 2, 2, 3]`), &unmarshaled)).To(Succeed())
		Expect(unmarshaled.Size()).To(Equal(3))
		Expect(unmarshaled.Contains(1)).To(BeTrue())
		Expect(unmarshaled.Contains(2)).To(BeTrue())
		Expect(unmarshaled.Contains(3)).To(BeTrue())

		Expect(json.Unmarshal([]byte(`null`), &unmarshaled)).To(Succeed())
		Expect(unmarshaled.Size()).To(Equal(0))

		Expect(json.Unmarshal([]byte(`invalid`), &unmarshaled)).NotTo(Succeed())
		Expect(unmarshaled.UnmarshalJSON([]byte(`invalid`))).NotTo(Succeed())
	})

	It("implements driver.Valuer and sql.Scanner", func() {
		s := NewAbstractSet(10, 20)
		val, err := s.Value()
		Expect(err).NotTo(HaveOccurred())
		var valSlice []int
		Expect(json.Unmarshal([]byte(val.(string)), &valSlice)).To(Succeed())
		Expect(valSlice).To(ConsistOf(10, 20))

		var nilSet *AbstractSet[int]
		nilVal, err := nilSet.Value()
		Expect(err).NotTo(HaveOccurred())
		Expect(nilVal).To(BeNil())

		unmarshalableSet := NewAbstractSet(failJSONKey{A: 1})
		valErr, err := unmarshalableSet.Value()
		Expect(err).To(HaveOccurred())
		Expect(valErr).To(BeNil())

		scanned := NewAbstractSet[int]()

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
	})

	It("supports lazily initializing elements on zero-value AbstractSet", func() {
		var s AbstractSet[int]
		Expect(s.Add(1, 2)).To(BeTrue())
		Expect(s.Contains(1)).To(BeTrue())

		var s2 AbstractSet[int]
		Expect(s2.AddAll(NewAbstractSet(10, 20))).To(BeTrue())
		Expect(s2.Contains(10)).To(BeTrue())

		var s3 AbstractSet[int]
		Expect(s3.Scan(`[1, 2]`)).To(Succeed())
		Expect(s3.Contains(1)).To(BeTrue())

		var s4 AbstractSet[int]
		Expect(s4.UnmarshalJSON([]byte(`[5, 6]`))).To(Succeed())
		Expect(s4.Contains(5)).To(BeTrue())
	})
})
