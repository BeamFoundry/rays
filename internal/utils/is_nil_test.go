package utils_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/BeamFoundry/rays/internal/utils"
)

var _ = Describe("IsNil", func() {
	It("correctly identifies nil and typed-nil values", func() {
		Expect(IsNil(nil)).To(BeTrue())

		var ptr *int
		Expect(IsNil(ptr)).To(BeTrue())

		var m map[string]int
		Expect(IsNil(m)).To(BeTrue())

		var s []string
		Expect(IsNil(s)).To(BeTrue())

		var ch chan int
		Expect(IsNil(ch)).To(BeFalse())

		nonNilInt := 42
		Expect(IsNil(&nonNilInt)).To(BeFalse())

		nonNilMap := make(map[string]int)
		Expect(IsNil(nonNilMap)).To(BeFalse())

		nonNilSlice := make([]string, 0)
		Expect(IsNil(nonNilSlice)).To(BeFalse())

		Expect(IsNil(123)).To(BeFalse())
		Expect(IsNil("hello")).To(BeFalse())
	})
})
