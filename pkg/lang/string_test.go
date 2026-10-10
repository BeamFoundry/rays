package lang_test

import (
	"go/constant"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/BeamFoundry/rays/pkg/lang"
)

var _ = Describe(constant.String.String(), func() {
	Context("Basename", func() {
		It("can return the basename", func() {
			Expect(String("/home/base/test").Basename().String()).To(Equal("test"))
		})
	})

	Context("CharAt", func() {
		It("can not get a char at a negative index", func() {
			_, err := String("test").CharAt(-1)
			Expect(err).To(HaveOccurred())
		})
		It("can not get a char at an index greater than string length", func() {
			_, err := String("test").CharAt(4)
			Expect(err).To(HaveOccurred())
		})
		It("can get a char at an index", func() {
			c, err := String("test").CharAt(3)
			Expect(err).ToNot(HaveOccurred())
			Expect(c).To(Equal(byte('t')))
		})
	})

	Context("CodePointAt", func() {
		It("can not get a char at a negative index", func() {
			_, err := String("test").CodePointAt(-1)
			Expect(err).To(HaveOccurred())
		})
		It("can not get a char at an index greater than string length", func() {
			_, err := String("test").CodePointAt(4)
			Expect(err).To(HaveOccurred())
		})
		It("can get a char at an index", func() {
			c, err := String("test").CodePointAt(3)
			Expect(err).ToNot(HaveOccurred())
			Expect(c).To(Equal(rune('t')))
		})
	})

	Context("CompareTo", func() {
		It("can compare String", func() {
			Expect(String("a").CompareTo("b")).To(BeNumerically("<", 0))
			Expect(String("a").CompareTo("a")).To(BeNumerically("==", 0))
			Expect(String("b").CompareTo("a")).To(BeNumerically(">", 0))
		})
	})

	Context("Concat", func() {
		It("can concaterate a string", func() {
			Expect(String("test").Concat("test").String()).To(Equal("testtest"))
		})
		It("can concaterate using operator", func() {
			h := String("Hello")
			w := String("World")
			m := h + " " + w
			Expect(m.Equals("Hello World")).To(BeTrue())
		})
	})

	Context("Contains", func() {
		It("can test if it contains a string", func() {
			Expect(String("test").Contains("st")).To(BeTrue())
		})
	})

	Context("Format", func() {
		It("can be used to format values", func() {
			Expect(String("%.2f").Format(3.14159).String()).To(Equal("3.14"))
		})
	})

	Context("EndsWith", func() {
		It("can test if a string has a suffix", func() {
			Expect(String("test").EndsWith("st")).To(BeTrue())
		})
	})

	Context("Equals", func() {
		It("can test if a String is equal to another String", func() {
			Expect(String("test").Equals("test")).To(BeTrue())
		})
	})

	Context("EqualsToIgnoreCase", func() {
		It("can test if a String is equal to another String regardless of its case", func() {
			Expect(String("Test").EqualsToIgnoreCase("test")).To(BeTrue())
		})
	})

	Context("GetBytes", func() {
		It("can get bytes", func() {
			Expect(String("test").GetBytes()).To(Equal([]byte("test")))
		})
	})

	Context("IndexOf", func() {
		It("can return the fist position of a char in a string", func() {
			Expect(String("test").IndexOf('s')).To(Equal(2))
		})
	})

	Context("IsBlank", func() {
		It("can test if a string is empty of full of whitespace", func() {
			Expect(String("").IsBlank()).To(BeTrue())
			Expect(String(" ").IsBlank()).To(BeTrue())
			Expect(String("a").IsBlank()).To(BeFalse())
		})
	})

	Context("Length", func() {
		It("can get the length of a String", func() {
			Expect(String("test").Length()).To(Equal(4))
		})
	})

	Context("Matches", func() {
		It("can match itself against a regexp", func() {
			Expect(String("test").Matches("^t")).To(BeTrue())
			Expect(String("test").Matches("^a")).To(BeFalse())
			_, err := String("Test").Matches("[a")
			Expect(err).To(HaveOccurred())
		})
	})

	Context("RegionMatches", func() {
		It("can compare a string region to another string region", func() {
			Expect(String("abcxyz").RegionMatches(false, 3, "xyz", 0, 3)).To(BeTrue())
			Expect(String("abcXYZ").RegionMatches(true, 3, "xyz", 0, 3)).To(BeTrue())
		})
	})

	Context("Replace", func() {
		It("can replace all char inside itself", func() {
			Expect(String("test").Replace("t", "T").String()).To(Equal("TesT"))
		})
	})
	Context("ReplaceRune", func() {
		It("can replace all char inside itself", func() {
			Expect(String("test").ReplaceRune('t', 'T').String()).To(Equal("TesT"))
		})
	})

	Context("ReplaceAll", func() {
		It("can replace all string inside itself", func() {
			_, err := String("test").ReplaceAll("[t", "T")
			Expect(err).To(HaveOccurred())
			v, err := String("test").ReplaceAll("te", "Te")
			Expect(err).NotTo(HaveOccurred())
			Expect(v.String()).To(Equal("Test"))
		})
	})

	Context("ReplaceFirst", func() {
		It("can replace the first string inside itself", func() {
			_, err := String("test").ReplaceFirst("[t", "T")
			Expect(err).To(HaveOccurred())
			v, err := String("test").ReplaceFirst("[t]", "T")
			Expect(err).NotTo(HaveOccurred())
			Expect(v.String()).To(Equal("Test"))
			v, err = String("test").ReplaceFirst("[x]", "T")
			Expect(err).NotTo(HaveOccurred())
			Expect(v.String()).To(Equal("test"))
		})
	})

	Context("StartsWith", func() {
		It("can test if a string has a prefix", func() {
			Expect(String("test").StartsWith("te")).To(BeTrue())
		})
	})

	Context("Split", func() {
		It("can split a string", func() {
			_, err := String("test").Split("[e")
			Expect(err).To(HaveOccurred())
			Expect(String("test").Split("e")).To(Equal([]string{"t", "st"}))
		})
	})

	Context("SplitN", func() {
		It("can split a string with a limit", func() {
			_, err := String("test").SplitN("[e", 2)
			Expect(err).To(HaveOccurred())
			Expect(String("a.b.c").SplitN("\\.", 2)).To(Equal([]string{"a", "b.c"}))
			Expect(String("a.b.c").SplitN("\\.", 3)).To(Equal([]string{"a", "b", "c"}))
			Expect(String("a.b.c").SplitN("\\.", -1)).To(Equal([]string{"a", "b", "c"}))
			Expect(String("a.b.c").SplitN("\\.", 0)).To(Equal([]string{"a", "b"}))
		})
	})

	Context("Substring", func() {
		It("can return a substring", func() {
			Expect(String("abcxyz").Substring(3).String()).To(Equal("xyz"))
		})
	})

	Context("SubstringOffset", func() {
		It("can return a substring with offset", func() {
			Expect(String("abcxyz").SubstringOffset(3, 4).String()).To(Equal("x"))
			Expect(String("").SubstringOffset(1, 2).String()).To(Equal(""))
			Expect(String("abcxyz").SubstringOffset(-3, 4).String()).To(Equal("x"))
			Expect(String("abcxyz").SubstringOffset(-3, -2).String()).To(Equal("x"))
			Expect(String("abcxyz").SubstringOffset(3, 8).String()).To(Equal("xyz"))
			Expect(String("abcxyz").SubstringOffset(3, 2).String()).To(Equal(""))
			Expect(String("abcxyz").SubstringOffset(-10, 3).String()).To(Equal("abc"))
			Expect(String("abcxyz").SubstringOffset(-20, -10).String()).To(Equal(""))
		})
	})

	Context("ToCharArray", func() {
		It("can return an array of rune", func() {
			Expect(String("abc").ToCharArray()).To(Equal([]rune("abc")))
		})
	})

	Context("ToLowerCase", func() {
		It("can lower a string", func() {
			Expect(String("Test").ToLowerCase().String()).To(Equal("test"))
		})
	})

	Context("ToUpperCase", func() {
		It("can upper a string", func() {
			Expect(String("Test").ToUpperCase().String()).To(Equal("TEST"))
		})
	})

	Context("Trim", func() {
		It("can trim a string", func() {
			Expect(String("   test   ").Trim()).To(Equal(String("test")))
		})
	})
})
