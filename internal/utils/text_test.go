package utils_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/BeamFoundry/rays/internal/utils"
)

var _ = Describe("Text", func() {
	Describe("Uncapitalize", func() {
		It("uncapitalizes empty strings and single characters", func() {
			Expect(Uncapitalize("")).To(Equal(""))
			Expect(Uncapitalize("A")).To(Equal("a"))
			Expect(Uncapitalize("a")).To(Equal("a"))
		})

		It("uncapitalizes words while preserving remaining characters", func() {
			Expect(Uncapitalize("HelloWorld")).To(Equal("helloWorld"))
			Expect(Uncapitalize("Test")).To(Equal("test"))
			Expect(Uncapitalize("test")).To(Equal("test"))
			Expect(Uncapitalize("ALL")).To(Equal("aLL"))
		})

		It("handles strings beginning with non-letter characters", func() {
			Expect(Uncapitalize("123ABC")).To(Equal("123ABC"))
			Expect(Uncapitalize("_private")).To(Equal("_private"))
			Expect(Uncapitalize(" space")).To(Equal(" space"))
		})

		It("handles multi-byte unicode characters", func() {
			Expect(Uncapitalize("Éléphant")).To(Equal("éléphant"))
			Expect(Uncapitalize("Ångström")).To(Equal("ångström"))
			Expect(Uncapitalize("Σigma")).To(Equal("σigma"))
			Expect(Uncapitalize("日本語")).To(Equal("日本語"))
		})

		It("handles invalid UTF-8 sequences", func() {
			invalid := string([]byte{0xff, 0xfe})
			Expect(Uncapitalize(invalid)).To(Equal(invalid))
		})
	})
})
