package container_test

import (
	"reflect"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	// . "github.com/BeamFoundry/rays/internal/container"

	. "github.com/BeamFoundry/rays/pkg/core"
	. "github.com/BeamFoundry/rays/pkg/lang"
	. "github.com/BeamFoundry/rays/pkg/testing"
)

type BeamWithCondition struct {
	Configuration
	condition BeamConditionEvaluator
	beam      *BeamWithValue
}

func (this *BeamWithCondition) BeamWithValue() *BeamWithValue {
	return this.beam
}

func (this *BeamWithCondition) PostConstruct() Error {
	this.beam = (&BeamWithValue{}).WithValue("test")
	return nil
}

func (this *BeamWithCondition) Condition() BeamConditionEvaluator {
	return this.condition
}

type BeamWithProfileAnnotation struct {
	Component
	ConditionalOnProfile `Name:"testing"`
}

/*
func (this *BeamWithProfileAnnotation) Profile() string {
	return "testing"
}
*/

type BeamMissingBeamCondition struct {
	Configuration
	ConditionalOnMissingBeam
}

func (this *BeamMissingBeamCondition) Beam() *BeamWithValue {
	return (&BeamWithValue{}).WithValue("registered")
}

var _ = Describe("Conditional", func() {
	Describe("ProfileCondition", func() {
		type BeamProfileCondition struct {
			Component
			ConditionalOnProfile `Name:"accepted"`
		}

		It("can accept Beam with Conditional annotation", func() {
			AddBeamFor(&BeamProfileCondition{})
			Expect(GetContainer().SetActiveProfiles([]string{"accepted"})).NotTo(HaveOccurred())

			if err := GetContainer().Initialize(); true {
				Expect(err).NotTo(HaveOccurred())
			}
			beams := GetContainer().FindBeams(reflect.TypeFor[*BeamProfileCondition]())
			Expect(beams).To(HaveLen(1))
		})

		It("can reject Beam with Conditional annotation", func() {
			AddBeamFor(&BeamProfileCondition{})
			Expect(GetContainer().SetActiveProfiles([]string{"refused"})).NotTo(HaveOccurred())

			if err := GetContainer().Initialize(); true {
				Expect(err).NotTo(HaveOccurred())
			}
			beams := GetContainer().FindBeams(reflect.TypeFor[*BeamProfileCondition]())
			Expect(beams).To(BeEmpty())
		})

		It("can accept an empty profile", func() {
			AddBeamFor(&BeamEmptyProfileCondition{})
			err := GetContainer().Initialize()
			Expect(err).NotTo(HaveOccurred())
			beams := GetContainer().FindBeams(reflect.TypeFor[*BeamWithValue]())
			Expect(beams).To(HaveLen(1))
		})
	})

	/*
		type BeamWithPropertyCondition struct {
			Component
			ConditionalOnProperty `Name:"config" `
		}
	*/

	Describe("PropertyCondition", func() {
		config := map[string]any{
			"string": "string",
			"true":   "true",
			"false":  "false",
			"empty":  "",
		}
		Context("If property condition value is not empty and key exist", func() {
			It("can accept Beam PropertyCondition if key exists and has correct value", func() {
				type BeamPropertyCondition struct {
					Component
					ConditionalOnProperty `Name:"string" HavingValue:"string"`
				}
				Expect(GetContainer().LoadConfig(config)).NotTo(HaveOccurred())
				AddBeamFor(&BeamPropertyCondition{})
				if err := GetContainer().Initialize(); true {
					Expect(err).NotTo(HaveOccurred())
				}

				beams := GetContainer().FindBeams(reflect.TypeFor[*BeamPropertyCondition]())
				Expect(beams).To(HaveLen(1))
			})

			It("can reject Beam PropertyCondition if key exists but has incorrect value", func() {
				type BeamPropertyCondition struct {
					Component
					ConditionalOnProperty `Name:"string" HavingValue:"not"`
				}
				Expect(GetContainer().LoadConfig(config)).NotTo(HaveOccurred())
				AddBeamFor(&BeamPropertyCondition{})
				if err := GetContainer().Initialize(); true {
					Expect(err).NotTo(HaveOccurred())
				}

				beams := GetContainer().FindBeams(reflect.TypeFor[*BeamPropertyCondition]())
				Expect(beams).To(BeEmpty())
			})
		})

		Context("If property key does not exist", func() {
			It("can accept Beam PropertyInMissingCondition if key does not exists", func() {
				type BeamPropertyCondition struct {
					Component
					ConditionalOnProperty `Name:"test" MatchIfMissing:"true"`
				}
				Expect(GetContainer().LoadConfig(config)).NotTo(HaveOccurred())
				AddBeamFor(&BeamPropertyCondition{})
				if err := GetContainer().Initialize(); true {
					Expect(err).NotTo(HaveOccurred())
				}

				beams := GetContainer().FindBeams(reflect.TypeFor[*BeamPropertyCondition]())
				Expect(beams).To(HaveLen(1))
			})

			It("can reject Beam PropertyCondition if key does not exists", func() {
				type BeamPropertyCondition struct {
					Component
					ConditionalOnProperty `Name:"test" MatchIfMissing:"false"`
				}
				Expect(GetContainer().LoadConfig(config)).NotTo(HaveOccurred())
				AddBeamFor(&BeamPropertyCondition{})
				if err := GetContainer().Initialize(); true {
					Expect(err).NotTo(HaveOccurred())
				}

				beams := GetContainer().FindBeams(reflect.TypeFor[*BeamPropertyCondition]())
				Expect(beams).To(BeEmpty())
			})
		})

		Context("If property condition value is empty", func() {
			It("can accept Beam PropertyCondition if config value evaluate to true", func() {
				type BeamPropertyCondition struct {
					Component
					ConditionalOnProperty `Name:"true"`
				}
				Expect(GetContainer().LoadConfig(config)).NotTo(HaveOccurred())
				AddBeamFor(&BeamPropertyCondition{})
				if err := GetContainer().Initialize(); true {
					Expect(err).NotTo(HaveOccurred())
				}

				beams := GetContainer().FindBeams(reflect.TypeFor[*BeamPropertyCondition]())
				Expect(beams).To(HaveLen(1))
			})

			It("can accept Beam PropertyCondition if config value does not evaluate to boolean", func() {
				type BeamPropertyCondition struct {
					Component
					ConditionalOnProperty `Name:"string"`
				}
				Expect(GetContainer().LoadConfig(config)).NotTo(HaveOccurred())
				AddBeamFor(&BeamPropertyCondition{})
				if err := GetContainer().Initialize(); true {
					Expect(err).NotTo(HaveOccurred())
				}

				beams := GetContainer().FindBeams(reflect.TypeFor[*BeamPropertyCondition]())
				Expect(beams).To(HaveLen(1))
			})
			It("can reject Beam PropertyCondition if config value evaluate to false", func() {
				type BeamPropertyCondition struct {
					Component
					ConditionalOnProperty `Name:"false"`
				}
				Expect(GetContainer().LoadConfig(config)).NotTo(HaveOccurred())
				AddBeamFor(&BeamPropertyCondition{})
				if err := GetContainer().Initialize(); true {
					Expect(err).NotTo(HaveOccurred())
				}

				beams := GetContainer().FindBeams(reflect.TypeFor[*BeamPropertyCondition]())
				Expect(beams).To(BeEmpty())
			})
		})
	})

	Describe("MissingBeamCondition", func() {
		It("can accept Beam with MissingBeamCondition", func() {
			AddBeamFor(&BeamMissingBeamCondition{})
			if err := GetContainer().Initialize(); true {
				Expect(err).NotTo(HaveOccurred())
			}

			beams := GetContainer().FindBeams(reflect.TypeFor[*BeamWithValue]())
			Expect(GetBeamDefaultInstance[*BeamWithValue]().GetValue()).To(Equal("registered"))
			Expect(beams).To(HaveLen(1))
		})
		It("can reject Beam with MissingBeamCondition", func() {
			AddBeamFor(&BeamMissingBeamCondition{})
			AddBeamFor((&BeamWithValue{}).WithValue("testing"))
			if err := GetContainer().Initialize(); true {
				Expect(err).NotTo(HaveOccurred())
			}

			beams := GetContainer().FindBeams(reflect.TypeFor[*BeamWithValue]())
			Expect(beams).To(HaveLen(1))
			Expect(GetBeamDefaultInstance[*BeamWithValue]().GetValue()).To(Equal("testing"))
		})
	})
})

type BeamEmptyProfileCondition struct {
	Configuration
	ConditionalOnProfile
	ConditionalOnProperty
}

func (this *BeamEmptyProfileCondition) Beam() *BeamWithValue {
	return (&BeamWithValue{}).WithValue("registered")
}

type BeamEmptyPropertyCondition struct {
	Configuration
	ConditionalOnProperty `Name:"string"`
}

func (this *BeamEmptyPropertyCondition) Beam() *BeamWithValue {
	return (&BeamWithValue{}).WithValue("registered")
}

var _ = Describe("Empty Profile Conditional", func() {
	It("can accept an empty profile", func() {
		AddBeamFor(&BeamEmptyProfileCondition{})
		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())
		beams := GetContainer().FindBeams(reflect.TypeFor[*BeamWithValue]())
		Expect(beams).To(HaveLen(1))
	})
	It("can accept an empty property", func() {
		/*
			config := map[string]any{
				"string": "string",
			}
		*/
		AddBeamFor(&BeamEmptyPropertyCondition{})
		// GetContainer().LoadConfig(config)
		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())
		beams := GetContainer().FindBeams(reflect.TypeFor[*BeamWithValue]())
		Expect(beams).To(BeEmpty())
	})
})

type BeamConfigurationWithAnnotedType struct {
	Configuration
}

func (this *BeamConfigurationWithAnnotedType) Beam1(_ struct{ Primary }) *BeamWithValue {
	return (&BeamWithValue{}).WithValue("beam1")
}

func (this *BeamConfigurationWithAnnotedType) Beam2() *BeamWithValue {
	return (&BeamWithValue{}).WithValue("beam2")
}

var _ = Describe("Register", func() {
	It("can register beam with annoted type", func() {
		AddBeamFor(&BeamConfigurationWithAnnotedType{})
		AddBeamFor(&BeamWithInject{})
		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())
		instance := GetBeamDefaultInstance[*BeamWithInject]()
		Expect(instance.GetBeam().GetValue()).To(Equal("beam1"))
	})
})

type BeamWithConditionalAnnotationProperties struct {
	used bool
}

type BeamWithConditionalAnnotation struct {
	Configuration
	ConditionalOnProfile `Name:"activated"`
	beam                 *BeamWithValue
	properties           *BeamWithConditionalAnnotationProperties `@:"Inject"`
}

func (this *BeamWithConditionalAnnotation) BeamWithValue() *BeamWithValue {
	Expect(this.properties.used).To(BeTrue())
	return this.beam
}

func (this *BeamWithConditionalAnnotation) PostContruct() Error {
	this.beam = (&BeamWithValue{}).WithValue("registered")
	return nil
}

var _ = Describe("Register Conditional", func() {
	It("can register beam with conditional annotation", func() {
		AddBeamFor(&BeamWithConditionalAnnotationProperties{used: true})
		AddBeamFor(&BeamWithConditionalAnnotation{})
		Expect(GetContainer().SetActiveProfiles([]string{"activated"})).NotTo(HaveOccurred())
		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())
	})

	It("can refuse to register bean with conditional annotation", func() {
		AddBeamFor(&BeamWithConditionalAnnotationProperties{used: false})
		AddBeamFor(&BeamWithConditionalAnnotation{})
		Expect(GetContainer().SetActiveProfiles([]string{"disabled"})).NotTo(HaveOccurred())
		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())
		cb := GetContainer().FindBeams(reflect.TypeFor[*BeamWithValue]())
		Expect(cb).To(BeEmpty())
	})
})

type ConditionalEmpty interface {
	BeamConditional
	Empty() string
}

type BeamWithEmptyCondition struct {
	Component
	ConditionalEmpty
}

// TODO Test with interface !!!

type BeamWithNullReturningMethod struct {
	Configuration
	register bool
}

func (this *BeamWithNullReturningMethod) BeamWithValue() *BeamWithValue {
	if !this.register {
		return nil
	}
	return (&BeamWithValue{}).WithValue("registered")
}

var _ = Describe("Register Null", func() {
	It("will load a beam with not defined Conditional Evaluator", func() {
		AddBeamFor(&BeamWithEmptyCondition{})
		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())
	})
	It("can refuse to register bean with nil returned", func() {
		AddBeamFor(&BeamWithNullReturningMethod{register: false})
		AddBeamFor((&BeamWithValue{}).WithValue("test"))

		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())

		cb := GetContainer().FindBeams(reflect.TypeFor[*BeamWithValue]())
		Expect(cb).To(HaveLen(1))
		instance := GetBeamDefaultInstance[*BeamWithValue]()
		Expect(instance.GetValue()).To(Equal("test"))
	})
})

var _ = Describe("CGO", func() {
	type BeamOnCGO struct {
		Component
		ConditionalOnCGO
	}
	type BeamOnNoCGO struct {
		Component
		ConditionalOnNoCGO
	}
	It("will load a Beam on CGO/NoCGO", func() {
		AddBeamFor(&BeamOnCGO{})
		AddBeamFor(&BeamOnNoCGO{})
		Expect(GetContainer().Initialize()).NotTo(HaveOccurred())
		cgo := GetContainer().FindBeams(reflect.TypeFor[*BeamOnCGO]())
		Expect(cgo).To(HaveLen(1))
		nocgo := GetContainer().FindBeams(reflect.TypeFor[*BeamOnNoCGO]())
		Expect(nocgo).To(BeEmpty())
	})
})
