package util_test

import (
	"testing"

	. "github.com/BeamFoundry/rays/pkg/util"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestUtil(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Pkg/Util Suite")
}

var _ = Describe("CaseUtils", func() {
	It("can capitalize a string", func() {
		Expect(CaseUtils("test").Capitalize().String()).To(Equal("Test"))
		Expect(CaseUtils("").Capitalize().String()).To(Equal(""))
	})
	It("can uncapitalize a string", func() {
		Expect(CaseUtils("Test").Uncapitalize().String()).To(Equal("test"))
		Expect(CaseUtils("").Uncapitalize().String()).To(Equal(""))
	})
	It("can UPPER a string", func() {
		Expect(CaseUtils("test").UpperCase().String()).To(Equal("TEST"))
	})
	It("can lower a string", func() {
		Expect(CaseUtils("TEST").LowerCase().String()).To(Equal("test"))
	})
	It("can snake_case aString", func() {
		Expect(CaseUtils("aString").SnakeCase().String()).To(Equal("a_string"))
		Expect(CaseUtils("JSONDate").SnakeCase().String()).To(Equal("json_date"))
		Expect(CaseUtils("JSON_Date").SnakeCase().String()).To(Equal("json_date"))
		Expect(CaseUtils("JSON Date").SnakeCase().String()).To(Equal("json_date"))
	})
})

var _ = Describe("AntMatcher", func() {
	It("can match a ? as any character", func() {
		p, err := AntMatcher{}.New("pkg/t?st.go")
		Expect(err).NotTo(HaveOccurred())
		Expect(p.Match([]byte("pkg/test.go"))).To(BeTrue())
		Expect(p.MatchString("pkg/test.go")).To(BeTrue())
		Expect(p.MatchString("pkg/st.go")).To(BeFalse())
		Expect(p.MatchString("pkg/testago")).To(BeFalse())
	})
	It("can match a * any character expect /", func() {
		p, err := AntMatcher{}.New("pkg/*.go")
		Expect(err).NotTo(HaveOccurred())
		Expect(p.String()).To(Equal("pkg/*.go"))
		Expect(p.MatchString("pkg/test.go")).To(BeTrue())
		Expect(p.MatchString("pkg/test/test.go")).To(BeFalse())
	})
	It("can match a ** any character", func() {
		p, err := AntMatcher{}.New("pkg/**.go")
		Expect(err).NotTo(HaveOccurred())
		Expect(p.MatchString("pkg/test.go")).To(BeTrue())
		Expect(p.MatchString("pkg/test/test.go")).To(BeTrue())
	})
	It("can match a ** as zero or more directory", func() {
		p, err := AntMatcher{}.New("pkg/**/test.go")
		Expect(err).NotTo(HaveOccurred())
		Expect(p.MatchString("pkg/test.go")).To(BeTrue())
		Expect(p.MatchString("pkg/pkg/test.go")).To(BeTrue())
	})
	It("can match special characters", func() {
		p, err := AntMatcher{}.New("pkg/^*$.go")
		Expect(err).NotTo(HaveOccurred())
		Expect(p.MatchString("pkg/^test$.go")).To(BeTrue())
	})
	It("will return an error on pattern matching error", func() {
		_, err := AntMatcher{}.New("pkg/\\?[a")
		Expect(err).To(HaveOccurred())
	})
})
