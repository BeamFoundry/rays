package util_test

import (
	"reflect"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/BeamFoundry/rays/pkg/util"
)

var _ = Describe("MapEntry", func() {
	It("returns correct KeyType and ValueType", func() {
		entry := &MapEntry[string, int]{}
		Expect(entry.KeyType()).To(Equal(reflect.TypeFor[string]()))
		Expect(entry.ValueType()).To(Equal(reflect.TypeFor[int]()))
	})

	It("can get Key and get/set Value", func() {
		entry := &MapEntry[string, string]{}
		Expect(entry.Key()).To(Equal(""))
		Expect(entry.Value()).To(Equal(""))

		entry.SetValue("hello")
		Expect(entry.Value()).To(Equal("hello"))
	})

	Context("Equals", func() {
		It("compares primitive values correctly", func() {
			entry := &MapEntry[string, int]{}
			entry.SetValue(42)

			Expect(entry.Equals(42)).To(BeTrue())
			Expect(entry.Equals(0)).To(BeFalse())
			Expect(entry.Equals(100)).To(BeFalse())
		})

		It("compares string values correctly", func() {
			entry := &MapEntry[int, string]{}
			entry.SetValue("test")

			Expect(entry.Equals("test")).To(BeTrue())
			Expect(entry.Equals("other")).To(BeFalse())
		})

		It("compares complex and non-comparable values like slices and maps", func() {
			entrySlice := &MapEntry[string, []int]{}
			entrySlice.SetValue([]int{1, 2, 3})

			Expect(entrySlice.Equals([]int{1, 2, 3})).To(BeTrue())
			Expect(entrySlice.Equals([]int{1, 2})).To(BeFalse())
			Expect(entrySlice.Equals(nil)).To(BeFalse())

			entryMap := &MapEntry[string, map[string]int]{}
			entryMap.SetValue(map[string]int{"a": 1, "b": 2})

			Expect(entryMap.Equals(map[string]int{"a": 1, "b": 2})).To(BeTrue())
			Expect(entryMap.Equals(map[string]int{"a": 1})).To(BeFalse())
			Expect(entryMap.Equals(nil)).To(BeFalse())
		})

		It("compares any/interface values correctly", func() {
			entryAny := &MapEntry[string, any]{}
			entryAny.SetValue("dynamic")

			Expect(entryAny.Equals("dynamic")).To(BeTrue())
			Expect(entryAny.Equals("other")).To(BeFalse())
			Expect(entryAny.Equals(123)).To(BeFalse())
			Expect(entryAny.Equals(nil)).To(BeFalse())

			entryAny.SetValue(nil)
			Expect(entryAny.Equals(nil)).To(BeTrue())
			Expect(entryAny.Equals("not nil")).To(BeFalse())
		})

		It("compares struct and pointer values correctly", func() {
			type sample struct {
				Name string
				ID   int
			}

			s1 := sample{ID: 1, Name: "alpha"}
			s2 := sample{ID: 1, Name: "alpha"}
			s3 := sample{ID: 2, Name: "beta"}

			entryStruct := &MapEntry[string, sample]{}
			entryStruct.SetValue(s1)
			Expect(entryStruct.Equals(s2)).To(BeTrue())
			Expect(entryStruct.Equals(s3)).To(BeFalse())

			entryPtr := &MapEntry[string, *sample]{}
			entryPtr.SetValue(&s1)
			Expect(entryPtr.Equals(&s2)).To(BeTrue())
			Expect(entryPtr.Equals(&s3)).To(BeFalse())
			Expect(entryPtr.Equals(nil)).To(BeFalse())
		})
	})
})
