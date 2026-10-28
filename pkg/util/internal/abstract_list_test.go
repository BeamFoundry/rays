package internal

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AbstractList", func() {
	It("creates empty or populated lists and supports capacity constructor", func() {
		l1 := NewAbstractList[int]()
		Expect(l1.Size()).To(Equal(0))
		Expect(l1.IsEmpty()).To(BeTrue())

		l2 := NewAbstractList(1, 2, 3)
		Expect(l2.Size()).To(Equal(3))
		Expect(l2.IsEmpty()).To(BeFalse())

		lCap := NewAbstractListWithCapacity[string](10)
		Expect(lCap.Size()).To(Equal(0))
		Expect(lCap.IsEmpty()).To(BeTrue())

		lNegCap := NewAbstractListWithCapacity[string](-5)
		Expect(lNegCap.Size()).To(Equal(0))
	})

	It("supports element access and modification with bounds checking", func() {
		l := NewAbstractList("alpha", "beta", "gamma")

		Expect(l.Get(0)).To(Equal("alpha"))
		Expect(l.Get(1)).To(Equal("beta"))
		Expect(l.Get(2)).To(Equal("gamma"))

		old := l.Set(1, "BETA")
		Expect(old).To(Equal("beta"))
		Expect(l.Get(1)).To(Equal("BETA"))

		Expect(func() { l.Get(-1) }).To(Panic())
		Expect(func() { l.Get(3) }).To(Panic())
		Expect(func() { l.Set(-1, "x") }).To(Panic())
		Expect(func() { l.Set(3, "x") }).To(Panic())
	})

	It("supports Add and AddAt with bounds checking", func() {
		l := NewAbstractList[int]()
		Expect(l.Add()).To(BeFalse())
		Expect(l.Add(1, 2)).To(BeTrue())
		Expect(l.ToSlice()).To(Equal([]int{1, 2}))

		// AddAt at end
		l.AddAt(2, 3)
		Expect(l.ToSlice()).To(Equal([]int{1, 2, 3}))

		// AddAt at beginning
		l.AddAt(0, 0)
		Expect(l.ToSlice()).To(Equal([]int{0, 1, 2, 3}))

		// AddAt in middle
		l.AddAt(2, 99)
		Expect(l.ToSlice()).To(Equal([]int{0, 1, 99, 2, 3}))

		Expect(func() { l.AddAt(-1, 5) }).To(Panic())
		Expect(func() { l.AddAt(10, 5) }).To(Panic())
	})

	It("supports RemoveAt and Remove with tail slot zeroing", func() {
		type item struct {
			name string
		}

		i1 := &item{"first"}
		i2 := &item{"second"}
		i3 := &item{"third"}

		list := NewAbstractList(i1, i2, i3)
		removed := list.RemoveAt(1)

		Expect(removed).To(Equal(i2))
		Expect(list.Size()).To(Equal(2))
		Expect(list.Get(0)).To(Equal(i1))
		Expect(list.Get(1)).To(Equal(i3))

		// The vacated tail slot within capacity should be zeroed (nil pointer)
		backing := list.items[:3]
		Expect(backing[2]).To(BeNil())

		// Remove by element
		Expect(list.Remove(i1)).To(BeTrue())
		Expect(list.Size()).To(Equal(1))
		Expect(list.Remove(&item{"not-present"})).To(BeFalse())

		Expect(func() { list.RemoveAt(-1) }).To(Panic())
		Expect(func() { list.RemoveAt(5) }).To(Panic())
	})

	It("supports IndexOf, LastIndexOf, and Contains", func() {
		l := NewAbstractList("a", "b", "c", "b", "a")

		Expect(l.IndexOf("b")).To(Equal(1))
		Expect(l.IndexOf("missing")).To(Equal(-1))

		Expect(l.LastIndexOf("b")).To(Equal(3))
		Expect(l.LastIndexOf("missing")).To(Equal(-1))

		Expect(l.Contains("a")).To(BeTrue())
		Expect(l.Contains("missing")).To(BeFalse())
	})

	It("supports ContainsAll and AddAll", func() {
		l := NewAbstractList(1, 2, 3)

		Expect(l.ContainsAll(nil)).To(BeFalse())
		Expect(l.ContainsAll(NewAbstractList(1, 3))).To(BeTrue())
		Expect(l.ContainsAll(NewAbstractList(1, 99))).To(BeFalse())

		Expect(l.AddAll(nil)).To(BeFalse())
		Expect(l.AddAll(NewAbstractList(4, 5))).To(BeTrue())
		Expect(l.ToSlice()).To(Equal([]int{1, 2, 3, 4, 5}))
	})

	It("supports RemoveAll and RetainAll", func() {
		l := NewAbstractList(1, 2, 3, 4, 5)

		Expect(l.RemoveAll(nil)).To(BeFalse())
		Expect(l.RemoveAll(NewAbstractList(99, 100))).To(BeFalse())
		Expect(l.RemoveAll(NewAbstractList(2, 4))).To(BeTrue())
		Expect(l.ToSlice()).To(Equal([]int{1, 3, 5}))

		Expect(l.RetainAll(nil)).To(BeFalse())
		Expect(l.RetainAll(NewAbstractList(1, 3, 5))).To(BeFalse())
		Expect(l.RetainAll(NewAbstractList(1, 5, 99))).To(BeTrue())
		Expect(l.ToSlice()).To(Equal([]int{1, 5}))
	})

	It("supports Clear with element zeroing", func() {
		type item struct {
			name string
		}

		i1 := &item{"first"}
		i2 := &item{"second"}

		list := NewAbstractList(i1, i2)
		list.Clear()

		Expect(list.Size()).To(Equal(0))
		Expect(list.IsEmpty()).To(BeTrue())

		// All elements up to previous length within capacity should be zeroed
		backing := list.items[:2]
		Expect(backing[0]).To(BeNil())
		Expect(backing[1]).To(BeNil())
	})

	It("supports ToSlice and Seq iteration", func() {
		l := NewAbstractList("x", "y", "z")

		slice := l.ToSlice()
		Expect(slice).To(Equal([]string{"x", "y", "z"}))
		// Mutating returned slice should not mutate internal list
		slice[0] = "mutated"
		Expect(l.Get(0)).To(Equal("x"))

		var full []string
		for item := range l.Seq() {
			full = append(full, item)
		}
		Expect(full).To(Equal([]string{"x", "y", "z"}))

		var partial []string
		for item := range l.Seq() {
			partial = append(partial, item)
			if item == "y" {
				break
			}
		}
		Expect(partial).To(Equal([]string{"x", "y"}))
	})

	It("supports SubList with bounds checking and independent copy semantics", func() {
		l := NewAbstractList("a", "b", "c", "d", "e")

		sub := l.SubList(1, 4)
		Expect(sub.ToSlice()).To(Equal([]string{"b", "c", "d"}))

		// Modifications to sublist do not affect original
		sub.Set(0, "MODIFIED")
		Expect(l.Get(1)).To(Equal("b"))

		Expect(func() { l.SubList(-1, 2) }).To(Panic())
		Expect(func() { l.SubList(0, 10) }).To(Panic())
		Expect(func() { l.SubList(3, 2) }).To(Panic())
	})

	It("supports Equals", func() {
		l1 := NewAbstractList(10, 20, 30)
		l2 := NewAbstractList(10, 20, 30)
		l3 := NewAbstractList(10, 20, 40)
		lDiffSize := NewAbstractList(10, 20)

		Expect(l1.Equals(nil)).To(BeFalse())
		Expect(l1.Equals(lDiffSize)).To(BeFalse())
		Expect(l1.Equals(l3)).To(BeFalse())
		Expect(l1.Equals(l2)).To(BeTrue())

		set := NewAbstractSet(10, 20, 30)
		Expect(l1.Equals(set)).To(BeFalse())
	})

	It("handles typed-nil collection operands across Equals, ContainsAll, AddAll, RemoveAll, and RetainAll", func() {
		l := NewAbstractList(1, 2, 3)
		var typedNil *AbstractList[int]

		Expect(l.ContainsAll(typedNil)).To(BeFalse())
		Expect(l.AddAll(typedNil)).To(BeFalse())
		Expect(l.RemoveAll(typedNil)).To(BeFalse())
		Expect(l.RetainAll(typedNil)).To(BeFalse())
		Expect(l.Equals(typedNil)).To(BeFalse())
	})

	It("implements json.Marshaler and json.Unmarshaler", func() {
		l := NewAbstractList("alpha", "beta")
		data, err := json.Marshal(l)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(data)).To(Equal(`["alpha","beta"]`))

		empty := NewAbstractList[int]()
		dataEmpty, err := json.Marshal(empty)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(dataEmpty)).To(Equal(`[]`))

		var nilList *AbstractList[int]
		dataNil, err := nilList.MarshalJSON()
		Expect(err).NotTo(HaveOccurred())
		Expect(string(dataNil)).To(Equal(`[]`))

		var unmarshaled AbstractList[string]
		Expect(json.Unmarshal([]byte(`["one","two"]`), &unmarshaled)).To(Succeed())
		Expect(unmarshaled.ToSlice()).To(Equal([]string{"one", "two"}))

		Expect(json.Unmarshal([]byte(`null`), &unmarshaled)).To(Succeed())
		Expect(unmarshaled.Size()).To(Equal(0))

		Expect(json.Unmarshal([]byte(`invalid`), &unmarshaled)).NotTo(Succeed())
		Expect(unmarshaled.UnmarshalJSON([]byte(`invalid`))).NotTo(Succeed())
		Expect(unmarshaled.UnmarshalJSON([]byte(`  null  `))).To(Succeed())
		Expect(unmarshaled.Size()).To(Equal(0))
	})

	It("implements driver.Valuer and sql.Scanner", func() {
		l := NewAbstractList(10, 20)
		val, err := l.Value()
		Expect(err).NotTo(HaveOccurred())
		Expect(val).To(Equal(`[10,20]`))

		var nilList *AbstractList[int]
		nilVal, err := nilList.Value()
		Expect(err).NotTo(HaveOccurred())
		Expect(nilVal).To(BeNil())

		unmarshalableList := NewAbstractList(func() {})
		valErr, err := unmarshalableList.Value()
		Expect(err).To(HaveOccurred())
		Expect(valErr).To(BeNil())

		scanned := NewAbstractList[int]()
		Expect(scanned.Scan(`[1,2,3]`)).To(Succeed())
		Expect(scanned.ToSlice()).To(Equal([]int{1, 2, 3}))

		Expect(scanned.Scan([]byte(`[4,5]`))).To(Succeed())
		Expect(scanned.ToSlice()).To(Equal([]int{4, 5}))

		Expect(scanned.Scan(nil)).To(Succeed())
		Expect(scanned.Size()).To(Equal(0))

		Expect(scanned.Scan("")).To(Succeed())
		Expect(scanned.Size()).To(Equal(0))

		Expect(scanned.Scan(12345)).NotTo(Succeed())
	})
})
