package lang_test

import (
	"reflect"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/BeamFoundry/rays/pkg/lang"
)

var _ = Describe("Optional", func() {
	type testStruct struct {
		test string
	}

	mapper := func(v *testStruct) *testStruct {
		v.test = "foo"
		return v
	}

	//nolint:gocritic // unlambda error
	flatMapper := func(v *testStruct) Optional[*testStruct] {
		return OptionalOf(v)
	}
	It("can created from Empty", func() {
		opt := OptionalEmpty[string]()
		Expect(opt.IsPresent()).To(BeFalse())
		Expect(opt.IsEmpty()).To(BeTrue())
		Expect(opt.Type()).To(Equal(reflect.TypeFor[string]()))
		Expect(opt.SetValue(0)).To(HaveOccurred())
	})

	It("can be created from a value with OptionalOfNullable, even empty (but not nil)", func() {
		v := OptionalOfNullable[string]("")
		Expect(v.OrElse("test")).To(Equal(""))
	})

	It("can not be created from a value with OptionalOfNullable if nil", func() {
		opt := OptionalOfNullable[*testStruct](nil)
		// Expect(err).To(HaveOccurred())
		opt.IfPresentOrElse(func(_ *testStruct) {
			Expect(true).To(BeFalse())
		}, func() {
			Expect(true).To(BeTrue())
		})
	})

	It("can be created from a value with OptionalOf", func() {
		test := &testStruct{"test"}
		opt := OptionalOf(test)
		Expect(opt.IsPresent()).To(BeTrue())
		Expect(opt.OrElseGet(func() *testStruct { return &testStruct{"foo"} }).test).To(Equal("test"))
		opt.IfPresent(func(v *testStruct) { Expect(v.test).To(Equal("test")) })
		opt.IfPresentOrElse(func(v *testStruct) {
			Expect(v.test).To(Equal("test"))
		}, func() {
			Expect(nil).To(BeNil())
		})
	})
	It("can be created from a nil value with OptionalOf", func() {
		var test *testStruct = nil
		opt := OptionalOf(test)
		Expect(opt.IsPresent()).To(BeFalse())
		Expect(opt.OrElse(&testStruct{"test"}).test).To(Equal("test"))
		Expect(opt.OrElseGet(func() *testStruct { return &testStruct{"test"} }).test).To(Equal("test"))
		Expect(opt.Map(func(v *testStruct) *testStruct { return v }).IsPresent()).To(BeFalse())
		Expect(opt.FlatMap(flatMapper).IsPresent()).To(BeFalse())
	})

	It("can be mapped", func() {
		test := &testStruct{"test"}
		opt := OptionalOfNullable[*testStruct](test)
		// Expect(err).NotTo(HaveOccurred())

		v, ok := opt.Map(mapper).Get()
		Expect(ok).To(BeTrue())
		Expect(v.test).To(Equal("foo"))

		Expect(opt.FlatMap(flatMapper).IsPresent()).To(BeTrue())
	})
})
