package core_test

import (
	"reflect"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/BeamFoundry/rays/pkg/core"
	. "github.com/BeamFoundry/rays/pkg/testing"
)

var _ = Describe("BeamBuilder", func() {
	It("can create beam for Interface", func() {
		type Foo interface {
			Foo() string
		}

		_, err := NewBeamBuilder(reflect.TypeFor[Foo]())
		Expect(err).NotTo(HaveOccurred())
	})
	It("can create beam for Pointer/Struct", func() {
		type Foo struct {
			Component
		}
		_, err := NewBeamBuilder(&Foo{})
		Expect(err).NotTo(HaveOccurred())
		_, err = NewBeamBuilder((*Foo)(nil))
		Expect(err).NotTo(HaveOccurred())
	})

	It("reject creating for other type other than Pointer/Struct", func() {
		_, err := NewBeamBuilder(reflect.TypeFor[BeamWithValue]())
		Expect(err).To(HaveOccurred())
	})

	It("can create beam with Prototype annotation", func() {
		type Foo struct {
			Prototype
		}

		AddBeamFor((*Foo)(nil))
		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())
	})

	It("can create beam with Scope annotation", func() {
		type Foo struct {
			BeamScopeAnnotation `Name:"Prototype"`
		}

		_, err := NewBeamBuilder((*Foo)(nil))
		Expect(err).NotTo(HaveOccurred())
	})

	It("can create Prototype beam using WithScope", func() {
		type Foo struct {
			Component
		}
		b, err := NewBeamBuilder(reflect.TypeFor[*Foo]())
		Expect(err).NotTo(HaveOccurred())
		err = GetContainer().Register(b.WithScope(BeamScope{}.New().PROTOTYPE).Build())
		Expect(err).NotTo(HaveOccurred())
		err = GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())
	})

	Context("can set beam qualifier", func() {
		type Foo struct {
			Component
		}
		type FooBar struct {
			Component
			Qualifier `Name:"bar"`
		}
		It("auto generated", func() {
			builder, err := NewBeamBuilder(reflect.TypeFor[*Foo]())
			Expect(err).NotTo(HaveOccurred())
			beam := builder.Build()
			Expect(beam.Attributes()["Qualifier"]).To(Equal("foo"))
		})
		It("using WithQualifier", func() {
			builder, err := NewBeamBuilder(reflect.TypeFor[*Foo]())
			Expect(err).NotTo(HaveOccurred())
			beam := builder.WithQualifier("bar").Build()
			Expect(beam.Attributes()["Qualifier"]).To(Equal("bar"))
		})
		It("using WithQualifier", func() {
			builder, err := NewBeamBuilder(reflect.TypeFor[*FooBar]())
			Expect(err).NotTo(HaveOccurred())
			beam := builder.Build()
			Expect(beam.Attributes()["Qualifier"]).To(Equal("bar"))
		})
	})
})

type BeamWithNew struct {
	Component
	slice []string
}

func (this BeamWithNew) New() *BeamWithNew {
	this.slice = []string{"foo"}
	return &this
}

func (this *BeamWithNew) Length() int {
	return len(this.slice)
}

var _ = Describe("BeamWithNew", func() {
	It("can be created from constructor", func() {
		AddBeamFor((*BeamWithNew)(nil))
		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())
		beam := GetBeamDefaultInstance[*BeamWithNew]()
		Expect(beam.Length()).To(Equal(1))
	})
})
