package lang_test

import (
	"go/constant"
	"unicode"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/BeamFoundry/rays/pkg/lang"
)

var _ = Describe("Strings", func() {
	Context(constant.String.String(), func() {
		Expect(Strings("a").String()).To(Equal("a"))
		Expect(Strings("a").ToString()).To(Equal(String("a")))
		Expect(ToStrings("a").String()).To(Equal("a"))
	})

	Context("CompareTo", func() {
		Expect(Strings("a").Compare("b")).To(BeNumerically("<", 0))
		Expect(Strings("a").Compare("a")).To(BeNumerically("==", 0))
		Expect(Strings("b").Compare("a")).To(BeNumerically(">", 0))
	})

	Context("Contains", func() {
		It("can test if it contains a string", func() {
			Expect(Strings("test").Contains("st")).To(BeTrue())
		})
	})

	Context("ContainsAny", func() {
		It("can test if it contains a string", func() {
			Expect(Strings("test").ContainsAny("et")).To(BeTrue())
		})
	})

	Context("ContainsFunc", func() {
		It("can test if it contains a string", func() {
			f := func(r rune) bool {
				return r == 'a' || r == 'e' || r == 'i' || r == 'o' || r == 'u'
			}
			Expect(Strings("test").ContainsFunc(f)).To(BeTrue())
		})
	})

	Context("ContainsRune", func() {
		It("can test if it contains a rune", func() {
			Expect(Strings("test").ContainsRune('e')).To(BeTrue())
		})
	})

	Context("Count", func() {
		Expect(Strings("test").Count("t")).To(BeNumerically("==", 2))
	})

	Context("Cut", func() {
		b, a, ok := Strings("test").Cut("e")
		Expect(ok).To(BeTrue())
		Expect(b).To(Equal("t"))
		Expect(a).To(Equal("st"))
	})

	Context("CutPrefix", func() {
		a, ok := Strings("test").CutPrefix("te")
		Expect(ok).To(BeTrue())
		Expect(a).To(Equal("st"))
	})

	Context("CutSuffix", func() {
		b, ok := Strings("test").CutSuffix("st")
		Expect(ok).To(BeTrue())
		Expect(b).To(Equal("te"))
	})

	Context("EqualFold", func() {
		Expect(Strings("test").EqualFold("TEST")).To(BeTrue())
	})

	Context("Fields", func() {
		Expect(Strings("aaa bbb").Fields()).To(Equal([]string{"aaa", "bbb"}))
	})

	Context("FieldsFunc", func() {
		f := func(c rune) bool {
			return c == ' '
		}
		Expect(Strings("aaa bbb").FieldsFunc(f)).To(Equal([]string{"aaa", "bbb"}))
	})

	Context("HasPrefix", func() {
		Expect(Strings("test").HasPrefix("te")).To(BeTrue())
	})

	Context("HasSuffix", func() {
		Expect(Strings("test").HasSuffix("st")).To(BeTrue())
	})

	Context("Index", func() {
		Expect(Strings("test").Index("st")).To(Equal(2))
	})

	Context("IndexAny", func() {
		Expect(Strings("test").IndexAny("xs")).To(Equal(2))
	})

	Context("IndexByte", func() {
		Expect(Strings("test").IndexByte(byte('s'))).To(Equal(2))
	})

	Context("IndexFunc", func() {
		Expect(Strings("test").IndexFunc(func(r rune) bool { return r == 's' })).To(Equal(2))
	})

	Context("IndexRune", func() {
		Expect(Strings("test").IndexRune('s')).To(Equal(2))
	})

	Context("LastIndex", func() {
		Expect(Strings("test").LastIndex("st")).To(Equal(2))
	})

	Context("LastIndexAny", func() {
		Expect(Strings("test").LastIndexAny("xs")).To(Equal(2))
	})

	Context("LastIndexByte", func() {
		Expect(Strings("test").LastIndexByte(byte('s'))).To(Equal(2))
	})

	Context("LastIndexFunc", func() {
		Expect(Strings("test").LastIndexFunc(func(r rune) bool { return r == 's' })).To(Equal(2))
	})

	Context("Map", func() {
		f := func(r rune) rune {
			if r == 'e' {
				return ' '
			}
			return r
		}
		Expect(Strings("test").Map(f)).To(Equal("t st"))
	})

	Context("Repeat", func() {
		Expect(Strings("x").Repeat(3)).To(Equal("xxx"))
	})

	Context("ReplaceAll", func() {
		Expect(Strings("test").ReplaceAll("t", "T")).To(Equal("TesT"))
	})

	Context("Split", func() {
		Expect(Strings("a,b,c").Split(",")).To(Equal([]string{"a", "b", "c"}))
	})

	Context("SplitAfter", func() {
		Expect(Strings("a,b,c").SplitAfter(",")).To(Equal([]string{"a,", "b,", "c"}))
	})

	Context("SplitAfterN", func() {
		Expect(Strings("a,b,c").SplitAfterN(",", 2)).To(Equal([]string{"a,", "b,c"}))
	})

	Context("SplitN", func() {
		Expect(Strings("a,b,c").SplitN(",", 2)).To(Equal([]string{"a", "b,c"}))
	})

	Context("ToLowerSpecial", func() {
		Expect(Strings("Önnek İş").ToLowerSpecial(unicode.TurkishCase)).To(Equal("önnek iş"))
	})

	Context("ToUpperSpecial", func() {
		Expect(Strings("örnek iş").ToUpperSpecial(unicode.TurkishCase)).To(Equal("ÖRNEK İŞ"))
	})

	Context("ToValidUTF8", func() {
		Expect(Strings("a\xffb\xC0\xAFc\xff").ToValidUTF8("")).To(Equal("abc"))
	})

	Context("Trim", func() {
		Expect(Strings("¡¡¡Hello, Gophers!!!").Trim("!¡")).To(Equal("Hello, Gophers"))
	})

	Context("TrimFunc", func() {
		f := func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsNumber(r)
		}
		Expect(Strings("¡¡¡Hello, Gophers!!!").TrimFunc(f)).To(Equal("Hello, Gophers"))
	})

	Context("TrimLeft", func() {
		Expect(Strings("¡¡¡Hello, Gophers!!!").TrimLeft("!¡")).To(Equal("Hello, Gophers!!!"))
	})

	Context("TrimPrefix", func() {
		Expect(Strings("¡¡¡Hello, Gophers!!!").TrimPrefix("¡¡¡Hello, ")).To(Equal("Gophers!!!"))
	})

	Context("TrimLeftFunc", func() {
		f := func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsNumber(r)
		}
		Expect(Strings("¡¡¡Hello, Gophers!!!").TrimLeftFunc(f)).To(Equal("Hello, Gophers!!!"))
	})

	Context("TrimRight", func() {
		Expect(Strings("¡¡¡Hello, Gophers!!!").TrimRight("!¡")).To(Equal("¡¡¡Hello, Gophers"))
	})

	Context("TrimRightFunc", func() {
		f := func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsNumber(r)
		}
		Expect(Strings("¡¡¡Hello, Gophers!!!").TrimRightFunc(f)).To(Equal("¡¡¡Hello, Gophers"))
	})

	Context("TrimSpace", func() {
		Expect(Strings(" \t\n Hello, Gophers \n\t\r\n").TrimSpace()).To(Equal("Hello, Gophers"))
	})

	Context("TrimSuffix", func() {
		Expect(Strings("¡¡¡Hello, Gophers!!!").TrimSuffix(", Gophers!!!")).To(Equal("¡¡¡Hello"))
	})
})
