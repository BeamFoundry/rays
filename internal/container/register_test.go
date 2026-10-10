package container_test

import (
	"reflect"
	"sync"

	. "github.com/BeamFoundry/rays/pkg/core"
	. "github.com/BeamFoundry/rays/pkg/lang"
	. "github.com/BeamFoundry/rays/pkg/testing"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type BeamConfiguration struct {
	Configuration
	Primary
	beam BeamWithValue
}

func (this *BeamConfiguration) BeamWithValue() *BeamWithValue {
	return &this.beam
}

// Will be excluded as it does not return a Pointer or Interface
func (this *BeamConfiguration) Value() string {
	return this.beam.GetValue()
}

// This will also be exluded as it's not a pointer to a struct
func (this *BeamConfiguration) ValuePointer() *string {
	return new(this.beam.GetValue())
}

func (this *BeamConfiguration) PostConstruct() Error {
	this.beam.SetValue("test")
	return nil
}

var _ = Describe("Register", func() {
	It("can register beam from a function", func() {
		AddBeamFor(&BeamConfiguration{})
		GetContainer().EnableScanning()
		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())
		beam := GetContainer().FindBeamsByAttributes(
			reflect.TypeFor[*BeamWithValue](),
			map[string]string{"Primary": "primary", "Qualifier": "beamWithValue"})
		Expect(beam).To(HaveLen(1))
		instance := GetBeamDefaultInstance[*BeamWithValue]()
		Expect(instance.GetValue()).To(Equal("test"))
		instance = GetBeamScopedInstance[*BeamWithValue]()
		Expect(instance.GetValue()).To(Equal("test"))
	})
})

type BeanConfigurationWithInject struct {
	Configuration
	beam    *BeamWithValue `@:"Inject"`
	beamRef *BeamWithInject
}

func (this BeanConfigurationWithInject) New() *BeanConfigurationWithInject {
	return &this
}

func (this *BeanConfigurationWithInject) BeamRef() *BeamWithInject {
	Expect(this.beamRef).NotTo(BeNil())
	Expect(this.beam.GetValue()).To(Equal("registered"))
	return this.beamRef
}

func (this *BeanConfigurationWithInject) PostConstruct() Error {
	Expect(this.beam.GetValue()).To(Equal("test"))
	this.beamRef = (&BeamWithInject{}).WithBeam(this.beam)
	this.beam.SetValue("registered")
	return nil
}

type BeanWithFuncRef struct {
	Component
	beamRef *BeamWithInject `@:"Inject"`
}

func (this *BeanWithFuncRef) PostConstruct() Error {
	Expect(this.beamRef.GetBeam().GetValue()).To(Equal("registered"))
	return nil
}

var _ = Describe("Register", func() {
	It("can register bean with functions and parameters", func() {
		AddBeamFor(&BeanConfigurationWithInject{})
		AddBeamFor((&BeamWithValue{}).WithValue("test"))
		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())

		instance := GetBeamDefaultInstance[*BeamWithInject]()
		Expect(instance.GetBeam().GetValue()).To(Equal("registered"))
	})

	It("can register bean with functions and parameters and inject it", func() {
		AddBeamFor(&BeanConfigurationWithInject{})
		AddBeamFor((&BeamWithValue{}).WithValue("test"))
		AddBeamFor(&BeanWithFuncRef{})
		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())

		instance := GetBeamDefaultInstance[*BeanWithFuncRef]()
		Expect(instance.beamRef.GetBeam().GetValue()).To(Equal("registered"))
	})
})

type BeamInterfaceConfiguration struct {
	Configuration
}

func (this *BeamInterfaceConfiguration) Valuer(_ struct{ Primary }) TestValuer {
	return (&BeamWithValue{}).WithValue("test")
}

func (this *BeamInterfaceConfiguration) ValuerNil() TestValuer {
	return nil
}

var _ = Describe("Register with interface", func() {
	It("can register a struct as an interface", func() {
		AddBeamFor(&BeamInterfaceConfiguration{})
		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())
		instance := GetBeamDefaultInstance[TestValuer]()
		Expect(instance.GetValue()).To(Equal("test"))
	})
})

type BeamScannedValue interface {
	ValueForScanned(value string)
}

type BeamScanned struct {
	Component
}

func (this *BeamScanned) ValueForScanned(_ string) {}

type BeamProviderScanned struct {
	TestingComponent
}

func (this *BeamProviderScanned) ValueForScanned(_ string) {}

func (this *BeamProviderScanned) ParentBeam(reflect.Type, InitializingContext) Beam {
	return nil
}

func (this *BeamProviderScanned) AdditionalBeams(Beam, InitializingContext) []Beam {
	return []Beam{}
}

type BeamIgnoreScan struct {
	TestingComponent
	IgnoreScanning
}

func (this *BeamIgnoreScan) ValueForScanned(_ string) {}

type BeamRepository struct {
	Repository
}

func (this *BeamRepository) ValueForScanned(_ string) {}

type BemController struct {
	Controller
}

func (this *BemController) ValueForScanned(_ string) {}

var _ = Describe("Scanning", func() {
	It("can register beam by scanning", func() {
		_, _ = any(&BeamProviderScanned{}).(Stereotype)
		GetContainer().SetScanFor(reflect.TypeFor[BeamScannedValue]())
		GetContainer().EnableScanning()
		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())
		GetBeamDefaultInstance[*BeamScanned]()
		GetBeamDefaultInstance[*BeamRepository]()
		GetBeamDefaultInstance[*BemController]()
	})
	It("can ignore beam", func() {
		GetContainer().SetScanFor(reflect.TypeFor[TestingComponent]())
		GetContainer().EnableScanning()
		beams := GetContainer().FindBeams(reflect.TypeFor[*BeamIgnoreScan]())
		Expect(beams).To(BeEmpty())
	})
})

type BeamCompositeInject struct {
	beam *BeamWithValue `@:"Inject"`
}

func (this *BeamCompositeInject) PostConstruct(parent any) Error {
	switch p := parent.(type) {
	case *BeamComposite:
		p.value = "parent"
		return nil
	case *BeamCompositeError:
		return NewError("TestError", "TestError")
	case *BeamWithAnnotedCompositeError:
		return NewError("TestError", "TestError")
	default:
		return nil
	}
}

type BeamCompositeRegister struct{}

func (this *BeamCompositeRegister) BeamWithValue() *BeamWithValue {
	return (&BeamWithValue{}).WithValue("beam")
}

type BeamComposite struct {
	Component
	BeamCompositeInject
	value string
}

type BeamCompositeError struct {
	Component
	BeamCompositeInject
}

//nolint:govet //Respecting formatting
type BeamCompositeConfiguration struct {
	Configuration
	BeamCompositeRegister
}

type BeamWithAnnotedComposite struct {
	Configuration
}

func (this *BeamWithAnnotedComposite) BeamWithValue() *BeamWithValue {
	return (&BeamWithValue{}).WithValue("beam")
}

func (this *BeamWithAnnotedComposite) TestValuer(
	_ struct {
		BeamCompositeInject
	},
) TestValuer {
	return (&BeamWithValue{}).WithValue("beam")
}

type BeamWithAnnotedCompositeError struct {
	Configuration
}

func (this *BeamWithAnnotedCompositeError) BeamWithValue() *BeamWithValue {
	return (&BeamWithValue{}).WithValue("beam")
}

func (this *BeamWithAnnotedCompositeError) TestValuer(
	_ struct {
		BeamCompositeInject
	},
) TestValuer {
	return (&BeamWithValue{}).WithValue("beam")
}

var _ = Describe("Composite Beam", func() {
	It("can refuse to handle an embedded struct if dependencies does not satisfy", func() {
		AddBeamFor(&BeamComposite{})
		err := GetContainer().Initialize()
		Expect(err).To(HaveOccurred())
	})
	It("can handle an embedded struct", func() {
		AddBeamFor(&BeamComposite{})
		AddBeamFor((&BeamWithValue{}).WithValue("beam"))
		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())
		beam := GetBeamDefaultInstance[*BeamComposite]()
		Expect(beam.beam.GetValue()).To(Equal("beam"))
		Expect(beam.value).To(Equal("parent"))
	})
	It("can handle an embedded struct in a Configuration", func() {
		AddBeamFor(&BeamCompositeConfiguration{})
		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())
		beam := GetBeamDefaultInstance[*BeamWithValue]()
		Expect(beam.GetValue()).To(Equal("beam"))
	})
	It("can handle an embedded struct in a Configuration annoted method", func() {
		AddBeamFor(&BeamWithAnnotedComposite{})
		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())
		beam := GetBeamDefaultInstance[*BeamWithValue]()
		Expect(beam.GetValue()).To(Equal("beam"))
	})
	It("can handle an embedded struct in a Configuration annoted method error", func() {
		AddBeamFor(&BeamWithAnnotedCompositeError{})
		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())
		beam := GetBeamDefaultInstance[*BeamWithValue]()
		Expect(beam.GetValue()).To(Equal("beam"))
	})
	It("can handle an embedded struct error", func() {
		AddBeamFor(&BeamCompositeError{})
		AddBeamFor((&BeamWithValue{}).WithValue("beam"))
		err := GetContainer().Initialize()
		Expect(err).To(HaveOccurred())
	})
})

type BeamWithRunner struct {
	Component
}

func (this *BeamWithRunner) Run(*sync.WaitGroup) {}

var _ = Describe("Beam With Runner", func() {
	It("can run", func() {
		AddBeamFor(&BeamWithRunner{})
		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())
		GetContainer().Run()
	})
})
