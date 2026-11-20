package container_test

import (
	"context"
	"errors"
	"reflect"

	"github.com/BeamFoundry/rays/internal/container"
	. "github.com/BeamFoundry/rays/pkg/core"
	. "github.com/BeamFoundry/rays/pkg/lang"
	. "github.com/BeamFoundry/rays/pkg/testing"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type BeamConfigurationError struct {
	Component
	Error   bool
	Err     bool
	PostErr bool
}

type BeamConfigurationErrors struct {
	Configuration
	beamError *BeamConfigurationError `@:"Inject"`
}

func (this *BeamConfigurationErrors) BeamWithValueError() (*BeamWithValue, Error) {
	if this.beamError.Error {
		return nil, NewError("TestError", "Error")
	}
	return (&BeamWithValue{}).WithValue("value"), nil
}

func (this *BeamConfigurationErrors) BeamWithValueErr() (*BeamWithValue, error) {
	if this.beamError.Err {
		return nil, errors.New("Test Error")
	}
	return (&BeamWithValue{}).WithValue("value"), nil
}

func (this *BeamConfigurationErrors) BeamWithWrongSignature(_ string) *BeamWithValue {
	Expect(false).To(BeTrue())
	return &BeamWithValue{}
}

func (this *BeamConfigurationErrors) BeamWithNonAnonymous(_ *BeamWithValue) *BeamWithValue {
	Expect(false).To(BeTrue())
	return &BeamWithValue{}
}

func (this *BeamConfigurationErrors) NoReturn() {}

func (this *BeamConfigurationErrors) PostConstruct() Error {
	if this.beamError.PostErr {
		return NewError("TestError", "Error")
	}
	return nil
}

type BeamConfigurationWithPrototypeError struct {
	Configuration
}

type keyValueType string

var keyValue keyValueType = "value"

func (this *BeamConfigurationWithPrototypeError) BeamWithPrototypeError(ctx context.Context) (*BeamWithValue, Error) {
	if _, ok := ctx.Value(keyValue).(string); ok {
		return nil, NewError("TestError", "Error")
	}
	return (&BeamWithValue{}).WithValue("value"), nil
}

type BeamPrototypeWithContextError struct {
	Component
	Prototype
}

func (this *BeamPrototypeWithContextError) WithContext(_ context.Context) Error {
	return NewError("TestError", "Error")
}

type BeamWithPrototypeContextErrorFoundry struct {
	Component
	foundry Foundry[*BeamPrototypeWithContextError] `@:"Inject"`
}

func (this *BeamWithPrototypeContextErrorFoundry) Validate() {
	_, err := this.foundry.Forge(context.Background())
	Expect(err).To(HaveOccurred())
}

//nolint:unused // Should be so.
type BeamWithPointerOfPrototypeWithContextError struct {
	Component
	beam *BeamPrototypeWithContextError `@:"Inject"`
}

//nolint:unused // Injected via reflection in error test
type EmbeddedPrototypeWithContextError struct {
	beam *BeamPrototypeWithContextError `@:"Inject"`
}

type BeamWithEmbeddedPrototypeWithContextError struct {
	Component
	EmbeddedPrototypeWithContextError
}

type BeamWithSliceOfPrototypeWithContextError struct {
	Component
	beams []*BeamPrototypeWithContextError `@:"Inject"`
}

type BeamWithMapOfPrototypeWithContextError struct {
	Component
	beams map[string]*BeamPrototypeWithContextError `@:"Inject"`
}

type BeamWithOptionalOfPrototypeWithContextError struct {
	Component
	beam Optional[*BeamPrototypeWithContextError] `@:"Inject"`
}

type PrototypeErrorTracker struct {
	calls []string
}

type PrototypeInnerErrorHook struct{}

func (this *PrototypeInnerErrorHook) PostConstruct(parent any) Error {
	if p, ok := parent.(*BeamPrototypeStageError); ok {
		p.recordCall("InnerPostConstruct")
		if p.failInner {
			return NewError("InnerError", "inner failed")
		}
	}
	return nil
}

type BeamPrototypeStageError struct {
	Component
	Prototype
	PrototypeInnerErrorHook
	tracker   *PrototypeErrorTracker
	failCtx   bool
	failInner bool
	failPost  bool
}

func (this *BeamPrototypeStageError) recordCall(name string) {
	if this.tracker != nil {
		this.tracker.calls = append(this.tracker.calls, name)
	}
}

func (this *BeamPrototypeStageError) WithContext(ctx context.Context) Error {
	if tracker, ok := ctx.Value(trackerKey).(*PrototypeErrorTracker); ok {
		this.tracker = tracker
	}
	if mode, ok := ctx.Value(keyValue).(string); ok {
		switch mode {
		case "ctx":
			this.failCtx = true
		case "inner":
			this.failInner = true
		case "post":
			this.failPost = true
		default:
		}
	}
	this.recordCall("WithContext")
	if this.failCtx {
		return NewError("ContextError", "context failed")
	}
	return nil
}

func (this *BeamPrototypeStageError) PostConstruct() Error {
	this.recordCall("PostConstruct")
	if this.failPost {
		return NewError("PostConstructError", "postconstruct failed")
	}
	return nil
}

type trackerKeyType string

var trackerKey trackerKeyType = "tracker"

type BeamPrototypeConfigurationError struct {
	Configuration
	Prototype
}

func (this *BeamPrototypeConfigurationError) Value() *BeamWithValue {
	return (&BeamWithValue{}).WithValue("invalid")
}

type BeamRefusedBeam struct {
	Configuration
	ConditionalOnMissingBeam
}

func (this *BeamRefusedBeam) BeamWithValueError() *BeamWithValue {
	return (&BeamWithValue{}).WithValue("test")
}

type BeamMissingBeam struct {
	Component
	beam *BeamWithValue `@:"Inject"`
}

func (this *BeamMissingBeam) BeamMissing() *BeamWithValue {
	return this.beam
}

type BeamWithStruct struct {
	Component
	beam BeamWithValue `@:"Inject"`
}

func (this *BeamWithStruct) GetValuer() TestValuer {
	return &this.beam
}

type GetValuer interface {
	GetValuer() TestValuer
}

type BeamConfigurationWithBeamWithStruct struct {
	Configuration
}

func (this *BeamConfigurationWithBeamWithStruct) BeamWithStruct() *BeamWithStruct {
	return &BeamWithStruct{}
}

//nolint:unused // Test only
type BeamConfigurationWithInjectStruct struct {
	Configuration
	beam BeamWithValue `@:"Inject"`
}

type BeamMultiplePrimary struct {
	Configuration
	Primary
}

func (this *BeamMultiplePrimary) Beam1() *BeamWithValue {
	return (&BeamWithValue{}).WithValue("beam1")
}

func (this *BeamMultiplePrimary) Beam2() *BeamWithValue {
	return (&BeamWithValue{}).WithValue("beam2")
}

type BeamWithNotInjectableConfiguration struct {
	Configuration
}

func (this *BeamWithNotInjectableConfiguration) Beam(
	_ struct {
		beam *BeamWithValue `@:"Inject,Qualifier{beam1}"`
	},
) *BeamWithValue {
	return (&BeamWithValue{}).WithValue("beam")
}

//nolint:unused // testing
type BeamWithStringInject struct {
	Component
	foo string `@:"Inject"`
}

var _ = Describe("Errors", func() {
	It("It can reject registering a non pointer type", func() {
		_, err := GetContainer().AddBeamFor(BeamConfiguration{})
		Expect(err).To(HaveOccurred())
		_, err = GetContainer().AddBeamFor(BeamWithValue{})
		Expect(err).To(HaveOccurred())
	})
	It("can error on recreating beam instance", func() {
		beam := NewBeamFor((&BeamWithValue{}).WithValue("test"))
		Expect(GetContainer().Register(beam)).NotTo(HaveOccurred())
		AddBeamFor(&BeamWithInject{})
		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())
		Expect(container.GetContainer().Registry.CreateBeam(beam)).To(HaveOccurred())
		Expect(container.GetContainer().Registry.InjectBeam(beam)).To(HaveOccurred())
		Expect(container.GetContainer().Registry.InitializeBeam(beam)).To(HaveOccurred())
		instance := GetContainer().GetBeamDefaultInstance(beam)
		Expect(instance.Create(context.Background())).To(HaveOccurred())
		Expect(instance.Inject()).To(HaveOccurred())
		Expect(instance.Initialize()).To(HaveOccurred())
		Expect(instance.InitializeWithContext(context.Background())).To(HaveOccurred())
	})
	It("can error on injecting or initializing a non created instance", func() {
		beam := NewBeamFor((&BeamWithValue{}).WithValue("test"))
		Expect(GetContainer().GetBeamDefaultInstance(beam)).To(BeNil())
		Expect(GetContainer().Register(beam)).NotTo(HaveOccurred())
		instance := GetContainer().GetBeamDefaultInstance(beam)
		Expect(instance.Inject()).To(HaveOccurred())
		Expect(instance.Initialize()).To(HaveOccurred())
		Expect(instance.InitializeWithContext(context.Background())).To(HaveOccurred())
	})
	It("can error on injecting an instance with missing candidate dependencies", func() {
		beam := NewBeamFor(&BeamMissingBeam{})
		Expect(GetContainer().Register(beam)).NotTo(HaveOccurred())
		instance := GetContainer().GetBeamDefaultInstance(beam)
		Expect(instance.Create(context.Background())).NotTo(HaveOccurred())
		Expect(instance.Inject()).To(HaveOccurred())
	})
	It("can error on creating beam after initialize", func() {
		AddBeamFor((&BeamWithValue{}).WithValue("test"))
		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())
		_, err = GetContainer().AddBeamFor(&BeamWithInject{})
		Expect(err).To(HaveOccurred())
	})

	It("can analyze Beam Constructor Error returned", func() {
		AddBeamFor(&BeamConfigurationError{Error: true})
		AddBeamFor(&BeamConfigurationErrors{})
		err := GetContainer().Initialize()
		Expect(err).To(HaveOccurred())
	})
	It("can analyze Beam Constructor error returned", func() {
		AddBeamFor(&BeamConfigurationError{Err: true})
		AddBeamFor(&BeamConfigurationErrors{})
		err := GetContainer().Initialize()
		Expect(err).To(HaveOccurred())
	})

	// TODO: Container should still initialize
	It("can log error on initialize beam error", func() {
		AddBeamFor(&BeamConfigurationError{PostErr: true})
		AddBeamFor(&BeamConfigurationErrors{})
		err := GetContainer().Initialize()
		Expect(err).To(HaveOccurred())
	})
	It("can report error on prototype beam creation ", func() {
		AddBeamFor(&BeamConfigurationWithPrototypeError{})
		AddBeamFor(&BeamWithFoundry[*BeamWithValue]{})
		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())

		instance := GetBeamDefaultInstance[*BeamWithFoundry[*BeamWithValue]]()

		// TODO: Add key

		_, err = instance.GetFoundry().Forge(context.WithValue(context.Background(), keyValue, "beam"))
		Expect(err).To(HaveOccurred())
	})
	It("can report error on prototype beam creation ", func() {
		AddBeamFor(&BeamPrototypeWithContextError{})
		AddBeamFor(&BeamWithPrototypeContextErrorFoundry{})
		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())

		instance := GetBeamDefaultInstance[*BeamWithPrototypeContextErrorFoundry]()

		instance.Validate()
	})
	It("can report error on prototype beam creation", func() {
		AddBeamFor(&BeamPrototypeWithContextError{})
		ownerBeam := AddBeamFor(&BeamWithPointerOfPrototypeWithContextError{})
		err := GetContainer().Initialize()
		Expect(err).To(HaveOccurred())
		Expect(GetContainer().GetBeamDefaultInstance(ownerBeam).Status()).To(Equal(container.InstanceStatus{}.New().CREATED))
	})
	It("fails owner injection for single instance, but does not block slice, map, or Optional fields on error", func() {
		// Embedded struct path (single instance injection fails owner)
		AddBeamFor(&BeamPrototypeWithContextError{})
		embeddedOwner := AddBeamFor(&BeamWithEmbeddedPrototypeWithContextError{})
		Expect(GetContainer().Initialize()).To(HaveOccurred())
		Expect(GetContainer().GetBeamDefaultInstance(embeddedOwner).Status()).To(Equal(container.InstanceStatus{}.New().CREATED))
		GetContainer().Reset()

		// Slice path (non-blocking, logs Error.LogError, empty slice)
		AddBeamFor(&BeamPrototypeWithContextError{})
		sliceOwner := AddBeamFor(&BeamWithSliceOfPrototypeWithContextError{})
		Expect(GetContainer().Initialize()).NotTo(HaveOccurred())
		Expect(GetContainer().GetBeamDefaultInstance(sliceOwner).Status()).To(Equal(container.InstanceStatus{}.New().INITIALIZED))
		Expect(GetBeamDefaultInstance[*BeamWithSliceOfPrototypeWithContextError]().beams).To(BeEmpty())
		GetContainer().Reset()

		// Map path (non-blocking, logs Error.LogError, empty map)
		AddBeamFor(&BeamPrototypeWithContextError{})
		mapOwner := AddBeamFor(&BeamWithMapOfPrototypeWithContextError{})
		Expect(GetContainer().Initialize()).NotTo(HaveOccurred())
		Expect(GetContainer().GetBeamDefaultInstance(mapOwner).Status()).To(Equal(container.InstanceStatus{}.New().INITIALIZED))
		Expect(GetBeamDefaultInstance[*BeamWithMapOfPrototypeWithContextError]().beams).To(BeEmpty())
		GetContainer().Reset()

		// Optional[T] path (non-blocking, logs Error.LogError, OptionalEmpty)
		AddBeamFor(&BeamPrototypeWithContextError{})
		optOwner := AddBeamFor(&BeamWithOptionalOfPrototypeWithContextError{})
		Expect(GetContainer().Initialize()).NotTo(HaveOccurred())
		Expect(GetContainer().GetBeamDefaultInstance(optOwner).Status()).To(Equal(container.InstanceStatus{}.New().INITIALIZED))
		Expect(GetBeamDefaultInstance[*BeamWithOptionalOfPrototypeWithContextError]().beam.IsPresent()).To(BeFalse())
	})
	It("stops prototype lifecycle on error at each stage", func() {
		AddBeamFor(&BeamPrototypeStageError{})
		AddBeamFor(&BeamWithFoundry[*BeamPrototypeStageError]{})
		Expect(GetContainer().Initialize()).NotTo(HaveOccurred())

		foundry := GetBeamDefaultInstance[*BeamWithFoundry[*BeamPrototypeStageError]]().GetFoundry()

		trackerCtx := &PrototypeErrorTracker{}
		ctx := context.WithValue(context.WithValue(context.Background(), keyValue, "ctx"), trackerKey, trackerCtx)
		_, err := foundry.Forge(ctx)
		Expect(err).To(HaveOccurred())
		Expect(err.Name()).To(Equal("ContextError"))
		Expect(trackerCtx.calls).To(Equal([]string{"WithContext"}))

		trackerInner := &PrototypeErrorTracker{}
		ctx = context.WithValue(context.WithValue(context.Background(), keyValue, "inner"), trackerKey, trackerInner)
		_, err = foundry.Forge(ctx)
		Expect(err).To(HaveOccurred())
		Expect(err.Name()).To(Equal("InnerError"))
		Expect(trackerInner.calls).To(Equal([]string{"WithContext", "InnerPostConstruct"}))

		trackerPost := &PrototypeErrorTracker{}
		ctx = context.WithValue(context.WithValue(context.Background(), keyValue, "post"), trackerKey, trackerPost)
		_, err = foundry.Forge(ctx)
		Expect(err).To(HaveOccurred())
		Expect(err.Name()).To(Equal("PostConstructError"))
		Expect(trackerPost.calls).To(Equal([]string{"WithContext", "InnerPostConstruct", "PostConstruct"}))
	})
	It("rejects Prototype Configuration parent at registration", func() {
		_, err := GetContainer().AddBeamFor(&BeamPrototypeConfigurationError{})
		Expect(err).To(HaveOccurred())
		Expect(err.Name()).To(Equal("InvalidScope"))
	})
	It("can log error if set profile after initialize", func() {
		Expect(GetContainer().Initialize()).NotTo(HaveOccurred())
		Expect(GetContainer().SetActiveProfiles([]string{"test"})).To(HaveOccurred())
	})
	It("can refuse to re-initialize", func() {
		Expect(GetContainer().Initialize()).NotTo(HaveOccurred())
		err := GetContainer().Initialize()
		Expect(err).To(HaveOccurred())
	})
	It("can remove beam on condition", func() {
		AddBeamFor(&BeamRefusedBeam{})
		AddBeamFor((&BeamWithValue{}).WithValue("test"))
		Expect(GetContainer().Initialize()).NotTo(HaveOccurred())
	})
	It("will fail to initialize if beam can not be created", func() {
		AddBeamFor(&BeamMissingBeam{})
		err := GetContainer().Initialize()
		Expect(err).To(HaveOccurred())
	})
	It("will fail adding a beam with a wrong inject field", func() {
		_, err := GetContainer().AddBeamFor(&BeamWithStruct{})
		Expect(err).To(HaveOccurred())
	})
	It("will fail adding a beam with a wrong inject field", func() {
		_, err := GetContainer().AddBeamFor(&BeamWithStringInject{})
		Expect(err).To(HaveOccurred())
	})
	It("will fail creating a Configuration beam with failed inject annotation", func() {
		AddBeamFor(&BeamWithNotInjectableConfiguration{})
		err := GetContainer().Initialize()
		Expect(err).To(HaveOccurred())
	})
	It("will log an error initializing a configuration beam with an inject struct", func() {
		_, err := GetContainer().AddBeamFor(&BeamConfigurationWithInjectStruct{})
		Expect(err).To(HaveOccurred())
		Expect(GetContainer().Initialize()).NotTo(HaveOccurred())
		beams := GetContainer().FindBeams(reflect.TypeFor[*BeamWithStruct]())
		Expect(beams).To(BeEmpty())
	})
	It("will log an error initializing a configuration beam with a beam with struct", func() {
		_, err := GetContainer().AddBeamFor(&BeamConfigurationWithBeamWithStruct{})
		Expect(err).To(HaveOccurred())
		Expect(GetContainer().Initialize()).NotTo(HaveOccurred())
		beams := GetContainer().FindBeams(reflect.TypeFor[*BeamWithStruct]())
		Expect(beams).To(BeEmpty())
	})
	It("will log an error if a scanned beam contains a struct to be injected", func() {
		GetContainer().SetScanFor(reflect.TypeFor[GetValuer]())
		GetContainer().EnableScanning()
		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())
		beams := GetContainer().FindBeams(reflect.TypeFor[*BeamWithStruct]())
		Expect(beams).To(BeEmpty())
	})
	It("will error on multiple primary beam", func() {
		AddBeamFor(&BeamMultiplePrimary{})
		AddBeamFor(&BeamWithInject{})
		err := GetContainer().Initialize()
		Expect(err).To(HaveOccurred())
	})
	It("will error on multiple beam with no primary", func() {
		AddBeamFor((&BeamWithValue{}).WithValue("beam1"))
		AddBeamFor((&BeamWithValue{}).WithValue("beam2"))
		AddBeamFor(&BeamWithInject{})
		err := GetContainer().Initialize()
		Expect(err).To(HaveOccurred())
	})
	It("will fail to find beams with not the right attributes", func() {
		AddBeamFor((&BeamWithValue{}).WithValue("beam1"))
		err := GetContainer().Initialize()
		Expect(err).NotTo(HaveOccurred())
		beams := GetContainer().FindBeamsByAttributes(reflect.TypeFor[*BeamWithValue](), map[string]string{"foo": "bar"})
		Expect(beams).To(BeEmpty())
	})
	It("will error on injecting a method annotated value if it can not be addressable", func() {
		parent := &BeamWithValue{}
		err := GetContainer().InjectMethodAnnotedValue(reflect.ValueOf(BeamWithValue{}), parent)
		Expect(err).To(HaveOccurred())
	})
	It("will error on injecting a method annotated value if not a struct pointer", func() {
		parent := &BeamWithValue{}
		err := GetContainer().InjectMethodAnnotedValue(reflect.ValueOf(new(1)), parent)
		Expect(err).To(HaveOccurred())
	})
	It("will error on loading bad logging parameters", func() {
		Expect(GetContainer().LoadConfigAt(map[string]any{"level": "LEVEL"}, "rays.logging")).NotTo(HaveOccurred())
		err := GetContainer().Initialize()
		Expect(err).To(HaveOccurred())
	})
	It("will error on injecting string", func() {
		_, err := GetContainer().AddBeamFor(&BeamWithStringInject{})
		Expect(err).To(HaveOccurred())
	})
})
