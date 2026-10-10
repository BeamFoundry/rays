package container_test

import (
	"context"
	"reflect"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/BeamFoundry/rays/pkg/core"
	. "github.com/BeamFoundry/rays/pkg/lang"
	. "github.com/BeamFoundry/rays/pkg/testing"
)

type BeamCircular1 struct {
	Component
	beam *BeamCircular2 `@:"Inject"`
	name string
}

type BeamCircular2 struct {
	Component
	beam *BeamCircular1 `@:"Inject"`
	name string
}

var _ = Describe("Inject Singleton Reference", func() {
	It("can register Beams", func() {
		AddBeamFor((&BeamWithValue{}).WithValue("test"))
		AddBeamFor(&BeamWithInject{})
		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())

		instance := GetBeamDefaultInstance[*BeamWithInject]()
		Expect(instance.GetBeam().GetValue()).To(Equal("test"))
	})

	It("can register cycling Beams", func() {
		AddBeamFor(&BeamCircular1{name: "beamA"})
		AddBeamFor(&BeamCircular2{name: "beamB"})
		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())

		instance1 := GetBeamDefaultInstance[*BeamCircular1]()
		Expect(instance1.beam.name).To(Equal("beamB"))
		instance2 := GetBeamDefaultInstance[*BeamCircular2]()
		Expect(instance2.beam.name).To(Equal("beamA"))
	})
})

type BeamCircularMethod struct {
	Configuration
}

func (this *BeamCircularMethod) Beam1() *BeamCircular1 {
	return &BeamCircular1{name: "beamA"}
}

func (this *BeamCircularMethod) Beam2() *BeamCircular2 {
	return &BeamCircular2{name: "beamB"}
}

var _ = Describe("Inject Singleton Method", func() {
	It("can register cycling beams", func() {
		AddBeamFor(&BeamCircularMethod{})
		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())

		instance1 := GetBeamDefaultInstance[*BeamCircular1]()
		Expect(instance1.beam.name).To(Equal("beamB"))
		instance2 := GetBeamDefaultInstance[*BeamCircular2]()
		Expect(instance2.beam.name).To(Equal("beamA"))
	})
})

type BeamWithPrimaryConfiguration struct {
	Configuration
	Primary
}

func (this *BeamWithPrimaryConfiguration) BeamWithPrimaryValue() *BeamWithValue {
	return (&BeamWithValue{}).WithValue("Primary")
}

type BeamWithSecondaryConfiguration struct {
	Configuration
}

func (this *BeamWithSecondaryConfiguration) BeamWithSecondaryValue() *BeamWithValue {
	return (&BeamWithValue{}).WithValue("Secondary")
}

var _ = Describe("Register Primary", func() {
	It("can inject primary beam", func() {
		AddBeamFor(&BeamWithPrimaryConfiguration{})
		AddBeamFor(&BeamWithSecondaryConfiguration{})
		AddBeamFor(&BeamWithInject{})

		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())
		instance := GetBeamDefaultInstance[*BeamWithInject]()
		Expect(instance.GetBeam().GetValue()).To(Equal("Primary"))
	})
})

type BeamWithScope struct {
	Component
	beam1 TestValuer `@:"Inject"`
	beam2 TestValuer `@:"Inject"`
}

func (this *BeamWithScope) PostConstruct() Error {
	this.beam1.SetValue("beam1")
	this.beam2.SetValue("beam2")
	return nil
}

var _ = Describe("Inject Prototype", func() {
	It("can inject Prototype beams (Scope)", func() {
		type BeamWithScopeValue struct {
			Component
			BeamScopeAnnotation `Name:"Prototype"`
			BeamWithValue
		}
		AddBeamFor((*BeamWithScopeValue)(nil))
		AddBeamFor(&BeamWithScope{})
		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())

		instance := GetBeamDefaultInstance[*BeamWithScope]()

		Expect(instance.beam1.GetValue()).To(Equal("beam1"))
		Expect(instance.beam2.GetValue()).To(Equal("beam2"))
	})

	It("can inject Prototype beams (Prototype)", func() {
		type BeamWithPrototypeValue struct {
			Component
			Prototype
			BeamWithValue
		}
		AddBeamFor((*BeamWithPrototypeValue)(nil))
		AddBeamFor(&BeamWithScope{})
		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())

		instance := GetBeamDefaultInstance[*BeamWithScope]()

		Expect(instance.beam1.GetValue()).To(Equal("beam1"))
		Expect(instance.beam2.GetValue()).To(Equal("beam2"))
	})
})

type BeamWithPrototypeConfiguration struct {
	Configuration
}

func (this *BeamWithPrototypeConfiguration) BeamWithValue(ctx context.Context) *BeamWithValue {
	GetLogger().Debug("BeamWithValue Prototype constructor called")
	if value, ok := ctx.Value(keyValue).(string); ok {
		ct := ctx.Value(PrototypeContextParentType)
		Expect(ct).NotTo(BeNil())
		return (&BeamWithValue{}).WithValue(value)
	}
	return (&BeamWithValue{}).WithValue("default")
}

type BeamWithPrototypeValue struct {
	Component
	Prototype
	value *BeamWithValue
}

func (this *BeamWithPrototypeValue) WithContext(ctx context.Context) Error {
	if value, ok := ctx.Value(keyValue).(string); ok {
		ct := ctx.Value(PrototypeContextParentType)
		Expect(ct).NotTo(BeNil())
		this.value = (&BeamWithValue{}).WithValue(value)
	}
	return nil
}

func (this *BeamWithPrototypeValue) Value() string {
	return this.value.GetValue()
}

var _ = Describe("Foundry", func() {
	It("can use Foundry", func() {
		AddBeamFor(&BeamWithPrototypeConfiguration{})
		AddBeamFor(&BeamWithFoundry[*BeamWithValue]{})

		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())

		instance := GetBeamDefaultInstance[*BeamWithFoundry[*BeamWithValue]]()

		beam, err := instance.GetFoundry().Forge(context.WithValue(context.Background(), keyValue, "beam1"))
		Expect(err).NotTo(HaveOccurred())
		Expect(beam.GetValue()).To(Equal("beam1"))
	})
	It("can use Foundry", func() {
		AddBeamFor(&BeamWithPrototypeValue{})
		AddBeamFor(&BeamWithFoundry[*BeamWithPrototypeValue]{})

		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())

		instance := GetBeamDefaultInstance[*BeamWithFoundry[*BeamWithPrototypeValue]]()

		beam, err := instance.GetFoundry().Forge(context.WithValue(context.Background(), keyValue, "beam1"))
		Expect(err).NotTo(HaveOccurred())
		Expect(beam.Value()).To(Equal("beam1"))
	})
	It("can use a Foundry even for Singleton", func() {
		AddBeamFor((&BeamWithValue{}).WithValue("value"))
		AddBeamFor(&BeamWithFoundry[*BeamWithValue]{})

		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())

		instance := GetBeamDefaultInstance[*BeamWithFoundry[*BeamWithValue]]()

		beam, err := instance.GetFoundry().Forge(context.WithValue(context.Background(), keyValue, "beam1"))
		Expect(err).NotTo(HaveOccurred())
		Expect(beam.GetValue()).To(Equal("value"))
	})
})

type BeamWithQualifier struct {
	Component
	beam1 *BeamWithValue `@:"Inject,Qualifier{beam1}"`
	beam2 *BeamWithValue `@:"Inject,Qualifier{beam2}"`
	beam3 *BeamWithValue `@:"Inject,Qualifier{beam3}"`
	beam4 *BeamWithValue `@:"Inject,Qualifier{beam4}"`
}

type BeamWithQualifierConfiguration struct {
	Configuration
}

func (this *BeamWithQualifierConfiguration) Beam1(
	_ struct {
		Qualifier `Name:"beam1"`
	},
) *BeamWithValue {
	return (&BeamWithValue{}).WithValue("beam1")
}

func (this *BeamWithQualifierConfiguration) Beam2(
	_ *struct {
		Qualifier `Name:"beam2"`
	},
) *BeamWithValue {
	return (&BeamWithValue{}).WithValue("beam2")
}

type BeamWithInjectConfiguration struct {
	Configuration
}

func (this *BeamWithInjectConfiguration) Beam3(
	s struct {
		Qualifier `Name:"beam3"`
		beam      *BeamWithValue `@:"Inject,Qualifier{beam4}"`
	},
) *BeamWithValue {
	if s.beam == nil || s.beam.GetValue() != "beam4" {
		return nil
	}
	return (&BeamWithValue{}).WithValue("beam3")
}

func (this *BeamWithInjectConfiguration) Beam4(
	s *struct {
		Qualifier `Name:"beam4"`
		beam      *BeamWithValue `@:"Inject,Qualifier{beam2}"`
	},
) *BeamWithValue {
	if s.beam == nil || s.beam.GetValue() != "beam2" {
		return nil
	}
	return (&BeamWithValue{}).WithValue("beam4")
}

var _ = Describe("Inject With Attributes", func() {
	It("Can inject beams by attributes", func() {
		AddBeamFor(&BeamWithQualifier{})
		AddBeamFor(&BeamWithQualifierConfiguration{})
		AddBeamFor(&BeamWithInjectConfiguration{})

		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())

		instance := GetBeamDefaultInstance[*BeamWithQualifier]()
		Expect(instance.beam1.GetValue()).To(Equal("beam1"))
		Expect(instance.beam2.GetValue()).To(Equal("beam2"))
		Expect(instance.beam3.GetValue()).To(Equal("beam3"))
		Expect(instance.beam4.GetValue()).To(Equal("beam4"))
	})
})

type BeamConfigurationWithInjectBeam struct {
	Configuration
}

func (this *BeamConfigurationWithInjectBeam) BeamWithInject() *BeamWithInject {
	return &BeamWithInject{}
}

var _ = Describe("Inject on beam declared via Configuration", func() {
	It("can inject beam in a beam declared via Configuration", func() {
		AddBeamFor((&BeamWithValue{}).WithValue("value"))
		AddBeamFor(&BeamConfigurationWithInjectBeam{})

		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())

		instance := GetBeamDefaultInstance[*BeamWithInject]()
		Expect(instance.GetBeam().GetValue()).To(Equal("value"))
	})
})

type BeamWithStereotypeInject struct {
	Service
	components []Component `@:"Inject"`
}

var _ = Describe("Inject by Stereotype", func() {
	It("Can inject beams by Stereotype", func() {
		AddBeamFor((&BeamWithValue{}).WithValue("value"))
		AddBeamFor(&BeamWithStereotypeInject{})

		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())
		instance := GetBeamDefaultInstance[*BeamWithStereotypeInject]()
		Expect(instance.components).NotTo(BeEmpty())
	})
})

type BeamWithSlice struct {
	Component
	beams []*BeamWithValue `@:"Inject"`
}

type BeamWithSliceInterface struct {
	Component
	beams []TestValuer `@:"Inject"`
}

var _ = Describe("Inject Slice", func() {
	It("can inject Slice", func() {
		AddBeamFor((&BeamWithValue{}).WithValue("beam1"))
		AddBeamFor((&BeamWithValue{}).WithValue("beam2"))

		AddBeamFor(&BeamWithSlice{})

		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())

		beams := GetContainer().FindBeams(reflect.TypeFor[*BeamWithValue]())
		Expect(beams).To(HaveLen(2))

		instance := GetBeamDefaultInstance[*BeamWithSlice]()
		Expect(instance.beams).To(HaveLen(2))
		for _, beam := range instance.beams {
			Expect(beam.GetValue()).To(BeElementOf([]string{"beam1", "beam2"}))
		}
	})
	It("can inject Slice by Interface", func() {
		AddBeamFor((&BeamWithValue{}).WithValue("beam1"))
		AddBeamFor((&BeamWithValue{}).WithValue("beam2"))

		AddBeamFor(&BeamWithSliceInterface{})

		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())

		beams := GetContainer().FindBeams(reflect.TypeFor[*BeamWithValue]())
		Expect(beams).To(HaveLen(2))

		instance := GetBeamDefaultInstance[*BeamWithSliceInterface]()
		Expect(instance.beams).To(HaveLen(2))
		for _, beam := range instance.beams {
			Expect(beam.GetValue()).To(BeElementOf([]string{"beam1", "beam2"}))
		}
	})
})

type BeamWithMapConfiguration struct {
	Configuration
}

func (this *BeamWithMapConfiguration) Beam1(
	struct {
		Server `_:"primary"`
	},
) *BeamWithValue {
	return (&BeamWithValue{}).WithValue("beam1")
}

func (this *BeamWithMapConfiguration) Beam2() *BeamWithValue {
	return (&BeamWithValue{}).WithValue("beam2")
}

type BeamWithMap struct {
	Component
	beams map[string]*BeamWithValue `@:"Inject"`
}

type BeamWithMapInterface struct {
	Component
	beams map[string]TestValuer `@:"Inject"`
}

type BeamWithMapGroupBy struct {
	Component
	beams map[string]*BeamWithValue `@:"Inject{GroupBy=Server}"`
}

type BeamWithMapGroupByWithDefault struct {
	Component
	beams map[string]*BeamWithValue `@:"Inject{GroupBy=Server,Default=primary}"`
}

type BeamWithMapGroupBySlice struct {
	Component
	beams map[string][]*BeamWithValue `@:"Inject{GroupBy=Server,Default=primary}"`
}

var _ = Describe("Inject Map", func() {
	It("can inject Map", func() {
		AddBeamFor(&BeamWithMapConfiguration{})
		AddBeamFor(&BeamWithMap{})

		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())

		beams := GetContainer().FindBeams(reflect.TypeFor[*BeamWithValue]())
		Expect(beams).To(HaveLen(2))

		instance := GetBeamDefaultInstance[*BeamWithMap]()
		Expect(instance.beams).To(HaveLen(2))
		Expect(instance.beams["beam1"].GetValue()).To(Equal("beam1"))
		Expect(instance.beams["beam2"].GetValue()).To(Equal("beam2"))
	})
	It("can inject Map by interface", func() {
		AddBeamFor(&BeamWithMapConfiguration{})
		AddBeamFor(&BeamWithMapInterface{})

		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())

		beams := GetContainer().FindBeams(reflect.TypeFor[*BeamWithValue]())
		Expect(beams).To(HaveLen(2))

		instance := GetBeamDefaultInstance[*BeamWithMapInterface]()
		Expect(instance.beams).To(HaveLen(2))
		Expect(instance.beams["beam1"].GetValue()).To(Equal("beam1"))
		Expect(instance.beams["beam2"].GetValue()).To(Equal("beam2"))
	})
	It("can inject map with groupby", func() {
		AddBeamFor(&BeamWithMapConfiguration{})
		AddBeamFor(&BeamWithMapGroupBy{})

		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())

		beams := GetContainer().FindBeams(reflect.TypeFor[*BeamWithValue]())
		Expect(beams).To(HaveLen(2))

		instance := GetBeamDefaultInstance[*BeamWithMapGroupBy]()
		Expect(instance.beams).To(HaveLen(1))
		Expect(instance.beams["primary"].GetValue()).To(Equal("beam1"))
	})
	It("can not inject map with same groupby value", func() {
		AddBeamFor(&BeamWithMapConfiguration{})
		AddBeamFor(&BeamWithMapGroupByWithDefault{})

		err := GetContainer().Initialize()
		Expect(err).To(HaveOccurred())
	})
	It("will not fail if no match", func() {
		AddBeamFor(&BeamWithMapGroupByWithDefault{})

		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())

		instance := GetBeamDefaultInstance[*BeamWithMapGroupByWithDefault]()
		Expect(instance.beams).NotTo(HaveKey("primary"))
		Expect(instance.beams).To(BeEmpty())
	})
	It("can inject map slice with same groupby value", func() {
		AddBeamFor(&BeamWithMapConfiguration{})
		AddBeamFor(&BeamWithMapGroupBySlice{})

		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())

		beams := GetContainer().FindBeams(reflect.TypeFor[*BeamWithValue]())
		Expect(beams).To(HaveLen(2))

		instance := GetBeamDefaultInstance[*BeamWithMapGroupBySlice]()
		Expect(instance.beams).To(HaveLen(1))
		Expect(instance.beams["primary"]).To(HaveLen(2))
	})
})
