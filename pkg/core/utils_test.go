package core_test

import (
	"reflect"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/BeamFoundry/rays/pkg/core"
)

//nolint:unused // Testing
type Foo struct {
	Service
	Primary
	Qualifier            `Name:"foo"` // Anonymous Annotation : @ alias for Name or _{Type}
	BeamScopeAnnotation  `Name:"Singleton"`
	ConditionalOnProfile `Name:"Test"`
	foo                  any `@:"Autowired,Qualifier{foo}"`
	bar                  any `@:"Autowired,Qualifier{bar}"`
}

var _ = Describe("Annotation", func() {
	It("can parse Annotations", func() {
		foo := reflect.TypeFor[*Foo]()

		Expect(FindFieldWithAnnotation[Service](foo)).NotTo(BeNil())
		Expect(FindFieldForInterface[Stereotype](foo)).NotTo(BeNil())
		Expect(FindFieldForInterface[ConditionalOnProperty](foo)).To(BeNil())

		field, ok := foo.Elem().FieldByName("Qualifier")
		Expect(ok).To(BeTrue())
		Expect(GetFieldFirstTag(&field, []string{"Name"})).NotTo(BeEmpty())
		Expect(GetFieldFirstTag(&field, []string{"Value"})).To(BeEmpty())
	})
})

type ComponentDefinition struct {
	Component
}

//nolint:unused // For testing
type ServiceDefinition struct {
	Service
	Primary
	Qualifier `Name:"serviceDefinition"`
	component *ComponentDefinition
}

var _, _ = any(&ServiceDefinition{}).(Stereotype)

var _ = Describe("Component scanning", func() {
	It("can find components", func() {
		stereo := reflect.TypeFor[Stereotype]()
		types := FindTypesImplementingInterface(stereo)
		Expect(len(types)).To(BeNumerically(">", 1))
		configuration := reflect.TypeFor[Configuration]()
		provider := reflect.TypeFor[BeamProvider]()
		types = FindTypesImplementingInterface(configuration, provider)
		Expect(types).To(HaveLen(1))
		types = FindTypesImplementingInterface(stereo, provider)
		Expect(len(types)).To(BeNumerically(">", 1))
	})
})
