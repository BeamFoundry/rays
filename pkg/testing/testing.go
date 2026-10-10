package testing

import (
	"reflect"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/BeamFoundry/rays/internal/container"
	"github.com/BeamFoundry/rays/internal/logger"
	. "github.com/BeamFoundry/rays/pkg/core"
	. "github.com/BeamFoundry/rays/pkg/lang"
)

/*
var _ = BeforeEach(func() {
	GetLogger().SetOutput(GinkgoWriter)
	GetLogger().SetLogLevel(slog.LevelDebug)
	GetLogger().AddSource()
})
*/

var _ = AfterEach(func() {
	GetContainer().Reset()
})

var _ = BeforeSuite(func() {
	GetLogger().SetOutput(GinkgoWriter)
	GetLogger().SetLogLevel(logger.LevelDebug)
	GetLogger().SetDefault()
	cfg := map[string]any{"handler": "DEFAULT", "level": "DEBUG", "source": "true"}
	Expect(GetContainer().LoadDefaultConfigAt(cfg, "rays.logging")).NotTo(HaveOccurred())
})

type TestingComponent interface {
	Stereotype
	_TestingComponent()
}

type TestingContext interface {
	// ContainerContext
	InitializingContext
	ContainerConfiguration
	// AddBeamFor() adds a Beam for either a struct pointer or a reflect.Type of a struct pointer.
	AddBeamFor(any) (Beam, Error)
	// LoadConfig() will load a configuration map for the current context
	LoadConfig(map[string]any) Error
	// LoadConfigAt() will load a configuration map for the current context at the specified path
	LoadConfigAt(map[string]any, string) Error
	// LoadConfigForProfile() will load a configuration map for a specified profile
	LoadConfigForProfile(map[string]any, string) Error
	// LoadConfigForProfileAt() will load a configuration map for a specified profile at the specified path.
	LoadConfigForProfileAt(map[string]any, string, string) Error
	// Reset() will reset the container. You don't have to do it if you use Gomega.
	Reset()
	// Set the specific type to scan, only for special internal testing cases.
	SetScanFor(rtype reflect.Type)
	GetBeamDefaultInstance(Beam) *container.Instance
	Register(beam Beam) Error
	// EnableScanning() must be called in testing to activate component scanning.
	EnableScanning()
}

func GetContainer() TestingContext {
	return container.GetContainer()
}

func GetLogger() *logger.Logger {
	return logger.GetLogger()
}

// GetBeam() retrieve a Beam of type T (T must be a struct pointer or interface)
func GetBeam[T any]() Beam {
	t := reflect.TypeFor[T]()
	beams := GetContainer().FindBeams(t)
	if len(beams) > 1 {
		logger.GetLogger().Error("Too many Beam found", "type", t.String(), "len", len(beams))
		return nil
		// } else if len(beams) == 0 {
		// 	logger.GetLogger().Error("Not one Beam found", "type", t.String(), "len", len(beams))
	}
	Expect(beams).To(HaveLen(1), t.String())
	return beams[0]
}

// GetBeamDefaultInstance() retrieve a Beam instance of type T (T must be a struct pointer or interface)
func GetBeamDefaultInstance[T any]() T {
	beam := GetContainer().GetBeamDefaultInstance(GetBeam[T]())
	instance, ok := beam.Get().(T)
	Expect(ok).To(BeTrue())
	return instance
}

func ValErr[T any](val T, err error) T {
	Expect(err).NotTo(HaveOccurred())
	return val
}

// TODO: Remove
func GetBeamScopedInstance[T any]() T {
	//nolint:gocritic //To be removed
	/*
		bean := GetContainer().GetRegistry().GetBeamScopedInstance(GetBeam[T]())
		instance, ok := bean.Get().(T)
		Expect(ok).To(BeTrue())
		return instance
	*/
	return GetBeamDefaultInstance[T]()
}

// Creates and register a Beam via a struct pointer or reflect.Type of a struct pointer
func AddBeamFor(ptr any) Beam {
	beam, err := GetContainer().AddBeamFor(ptr)
	Expect(err).NotTo(HaveOccurred())
	return beam
}

// Creates a Beam via a struct pointer or reflect.Type of a struct pointer
func NewBeamFor(ptr any) Beam {
	builder, err := NewBeamBuilder(ptr)
	Expect(err).ToNot(HaveOccurred())
	return builder.Build()
}

type TestValuer interface {
	GetValue() string
	SetValue(string)
}

type BeamWithValue struct {
	Component
	value string
}

func (this *BeamWithValue) GetValue() string {
	return this.value
}

func (this *BeamWithValue) SetValue(value string) {
	this.value = value
}

func (this *BeamWithValue) WithValue(value string) *BeamWithValue {
	this.value = value
	return this
}

type BeamWithInject struct {
	Component
	beam *BeamWithValue `@:"Inject"`
}

func (this *BeamWithInject) GetBeam() *BeamWithValue {
	return this.beam
}

func (this *BeamWithInject) SetBeam(beam *BeamWithValue) {
	this.beam = beam
}

func (this *BeamWithInject) WithBeam(beam *BeamWithValue) *BeamWithInject {
	this.beam = beam
	return this
}

type BeamWithFoundry[T any] struct {
	Component
	foundry Foundry[T] `@:"Inject"`
}

func (this *BeamWithFoundry[T]) GetFoundry() *Foundry[T] {
	return &this.foundry
}
