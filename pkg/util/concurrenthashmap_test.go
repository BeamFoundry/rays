package util_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/BeamFoundry/rays/pkg/util"
)

//nolint:errcheck // not testing
var _ = Describe("ConcurrentHashMap", func() {
	It("can create a new Map", func() {
		m := ConcurrentHashMap[string, int]{}.New()
		Expect(m).NotTo(BeNil())
		m.Put("one", 1)
		Expect(m.ContainsKey("one")).To(BeTrue())
		Expect(m.IsEmpty()).To(BeFalse())
		Expect(m.Size()).To(Equal(1))
		one, ok := m.Get("one").Get()
		Expect(ok).To(BeTrue())
		Expect(one).To(Equal(1))
		m.Merge(NewHashMapFor(map[string]int{"two": 2}))
		m.PutAll(NewHashMapFor(map[string]int{"three": 3}))
		m.Remove("two")
		m.Delete(func(_ string, _ int) bool { return true })
		b, _ := m.MarshalJSON()
		n := ConcurrentHashMap[string, int]{}.New()
		_ = n.UnmarshalJSON(b)
		m.Iterator()
		m.Equals(ConcurrentHashMap[string, int]{}.New())
		m.Clear()
	})
})
