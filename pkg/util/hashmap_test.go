package util_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/BeamFoundry/rays/pkg/util"
)

var _ = Describe("Maps", func() {
	It("can create a new Map", func() {
		m := NewHashMap[string, int]()
		m.Put("test", 1)

		Expect(m.Map()["test"]).To(Equal(1))
	})

	It("can create a Map for a map", func() {
		m1 := map[string]int{"test": 1}

		m2 := NewHashMapFor(m1)
		Expect(m1["test"]).To(Equal(m2.Map()["test"]))

		m2.Put("test", 2)
		Expect(m1["test"]).To(Equal(m2.Map()["test"]))
	})

	It("can create a Map from a map", func() {
		m1 := map[string]int{"test": 1}

		m2 := NewHashMapFrom(m1)
		Expect(m1["test"]).To(Equal(m2.Map()["test"]))

		m2.Put("test", 2)
		Expect(m1["test"]).NotTo(Equal(m2.Map()["test"]))
	})

	It("can remove all the mappings", func() {
		m1 := map[string]int{"test": 1}

		m2 := NewHashMapFor(m1)
		Expect(m1["test"]).To(Equal(1))
		Expect(m1["test"]).To(Equal(m2.Map()["test"]))

		m2.Clear()
		Expect(m1["test"]).To(Equal(0))
		Expect(m1["test"]).To(Equal(m2.Map()["test"]))
	})

	It("can merge DeepMerge a map", func() {
		m1 := map[string]any{"a": 1, "c": map[string]any{"x": 1, "y": 1}}
		m2 := map[string]any{"a": 2, "b": 1, "c": map[string]any{"y": 2, "z": 1}}
		m3 := map[string]any{"b": 2, "c": map[string]any{"y": 3, "z": 2}}

		expected := map[string]any{"a": 2, "b": 2, "c": map[string]any{"x": 1, "y": 3, "z": 2}}

		m := NewHashMap[string, any]()
		m.Merge(NewHashMapFor(m1))
		ma := NewHashMapFor(m2)
		mb := NewHashMapFor(m3)
		m.Merge(ma)
		m.Merge(mb)

		Expect(m.Equals(NewHashMapFor(expected))).To(BeTrue())
		Expect(ma.Map()["b"]).To(Equal(1))
		Expect(mb.Map()["b"]).To(Equal(2))
		Expect(ma.Map()["c"].(map[string]any)["y"]).To(Equal(2))
		Expect(mb.Map()["c"].(map[string]any)["y"]).To(Equal(3))
	})

	It("can DeepMerge and replace key type", func() {
		m1 := map[string]any{"a": 1, "b": map[string]any{"x": 1, "y": 1}}
		m2 := map[string]any{"b": map[string]any{"y": map[string]any{"z": 1}}}

		expected := map[string]any{"a": 1, "b": map[string]any{"x": 1, "y": map[string]any{"z": 1}}}
		m := NewHashMapFor(m1)
		m.Merge(NewHashMapFor(m2))

		Expect(m.Equals(NewHashMapFor(expected))).To(BeTrue())
	})

	It("can clone (shallow) a Map", func() {
		m1 := NewHashMapFrom(map[string]int{"test": 1})
		m2 := m1.Clone()
		m2.Put("test", 2)
		Expect(m1.Map()["test"]).To(Equal(1))
		Expect(m2.Map()["test"]).To(Equal(2))
	})

	It("can assert that two Maps are equals", func() {
		m1 := map[string]int{"test": 1}

		m2 := NewHashMapFrom(m1)
		Expect(m2.Equals(NewHashMapFor(m1))).To(BeTrue())

		m2.Put("test", 2)
		Expect(m2.Equals(NewHashMapFor(m1))).To(BeFalse())
	})

	It("can calculate its size and if it's empty", func() {
		m := NewHashMap[string, int]()

		Expect(m.IsEmpty()).To(BeTrue())

		m.Put("test", 1)

		Expect(m.IsEmpty()).To(BeFalse())
		Expect(m.Size()).To(Equal(1))
	})

	It("can assert if a key exist", func() {
		m := NewHashMap[string, int]()
		m.Put("test", 1)

		Expect(m.Map()["test"]).To(Equal(1))
		Expect(m.ContainsKey("test")).To(BeTrue())
		Expect(m.ContainsKey("testing")).To(BeFalse())
	})

	It("can set a key", func() {
		m := NewHashMap[string, int]()
		m.Put("test", 1)
		Expect(m.Map()["test"]).To(Equal(1))
		Expect(m.ContainsKey("test")).To(BeTrue())
	})
	It("can set multiple keys", func() {
		m := NewHashMap[string, int]()
		m.PutAll(NewHashMapFor(map[string]int{"test": 1}))
		Expect(m.Map()["test"]).To(Equal(1))
		Expect(m.ContainsKey("test")).To(BeTrue())
	})
	It("can retrieve a key", func() {
		m := NewHashMap[string, int]()
		m.Put("test", 1)
		Expect(m.ContainsKey("test")).To(BeTrue())
		v, ok := m.Get("test").Get()
		Expect(ok).To(BeTrue())
		Expect(v).To(Equal(1))
		v, ok = m.Get("testing").Get()
		Expect(ok).To(BeFalse())
		Expect(v).To(BeZero())
	})
	It("can remove a key", func() {
		m := NewHashMap[string, int]()
		m.Put("test", 1)
		Expect(m.ContainsKey("test")).To(BeTrue())
		r, ok := m.Remove("test").Get()
		Expect(ok).To(BeTrue())
		Expect(r).To(Equal(1))
		Expect(m.Size()).To(Equal(0))
		Expect(m.Remove("test").IsEmpty()).To(BeTrue())
	})
	It("can use a function to delete", func() {
		m := NewHashMap[string, int]()
		m.Put("test", 1)
		Expect(m.ContainsKey("test")).To(BeTrue())
		m.Delete(func(key string, val int) bool {
			if key == "test" && val == 1 {
				return true
			}
			return false
		})
		Expect(m.Size()).To(Equal(0))
	})
	It("can Marshal/Unmarshal to/from JSON", func() {
		m := NewHashMap[string, int]()
		m.Put("test", 1)
		Expect(m.ContainsKey("test")).To(BeTrue())
		b, err := m.MarshalJSON()
		Expect(err).NotTo(HaveOccurred())
		r, ok := m.Remove("test").Get()
		Expect(ok).To(BeTrue())
		Expect(r).To(Equal(1))
		Expect(m.Size()).To(Equal(0))
		Expect(m.UnmarshalJSON(b)).NotTo(HaveOccurred())
		Expect(m.ContainsKey("test")).To(BeTrue())
		Expect(m.Size()).To(Equal(1))
	})
})
