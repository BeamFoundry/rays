package core

import (
	"context"
	"io"
	"io/fs"
	"log/slog"
	"reflect"
	"sync"

	. "github.com/BeamFoundry/rays/pkg/lang"
)

// Beam represent a beam definition, which is what the "container" instantiates and injects.
//
// There is a default [Beam] struct implementation [DefaultBeam], which can be built using the [BeamBuilder] type
// and the [NewBeamBuilder] function.
//
// There is no need to have your own implementation, but it's open.
type Beam interface {
	// Attributes() returns a map of string/string containing this beam attributes.
	//
	// Attributes are a way to mark Beam with values like "Qualifier" or "Primary".
	Attributes() map[string]string

	// Type() returns the struct type (as a pointer)
	Type() reflect.Type

	// AnnotatedType() may returns a different type, to look for annotations.
	//
	// This is mainly used by [Configuration] structs where each function may have as first argument an anonymous
	// struct defining other annotations than the parent [Configuration] struct.
	AnnotatedType() reflect.Type

	// Scope() represent the [BeamScope] (Singleton or Prototype)
	Scope() BeamScope

	// Foundry() represent the function used to create another instance of this [Beam].
	//
	// Implementation for a Singleton Beam should always return the same pointer to the struct, regardless of [context.Context].
	//
	// Implementations for a Prototype Beam will get from [context.Context] the following key/values :
	//
	//  - [PrototypeContextParentType] : [reflect.Type] of the parent object it is inject to.
	//
	//  - [PrototypeContextParentInstance] : (any) current instance of the parent object.
	//
	// NOTE: Default Foundry created by [BeamBuilder] do not care about those.
	// Only [Configuration] method [Beam] will receive them, if it has a [context.Context] argument.
	// Typical stereotyped Beam should use [BeamPostConstructWithContext.WithContext].
	Foundry() func(context.Context) (any, Error)
}

// BeamAttribute represents a base marker interface for a [Beam] attribute. Like for example [Qualifier].
//
// Those attributes may be used with an @:"Inject" tag to limit injection on a specific [Beam].
type BeamAttribute interface {
	_BeamAttribute()
}

// BeamAttributeEvaluator represents the interface a struct must implements to define an evaluator of a particular
// [BeamAttribute].
//
// That struct must contain that [BeamAttribute] marker interface. For example the [Primary] [BeamAttribute]
// and its [BeamAttributeEvaluator]:
//
//	type Primary interface {
//		BeamAttribute // Implements BeamAttribute
//		_Primary() // Marker interface method
//	}
//
//	type PrimaryEvaluator struct {
//		Primary
//	}
//
//	func (this *PrimaryEvaluator) EvaluateBeamAttribute(_ Beam) string {
//		return "primary"
//	}
type BeamAttributeEvaluator interface {
	// EvaluateBeamAttribute() must return a value for that specific attribute, mostly taken from a struct tag.
	//
	// The returned string must not be empty, as it will indicate that Beam does not have that attribute.
	EvaluateBeamAttribute(beam Beam) string
}

// BeamConditional represents a base marker interface for a [Beam] conditional creation, like for example [ConditionalOnProfile]
type BeamConditional interface {
	_BeamConditional()
}

// BeamConditionEvaluator represents the interface a struct must implements to define an evaluator of
// a particular [BeamConditional].
//
// That struct must contain that [BeamConditional] marker interface.
type BeamConditionEvaluator interface {
	// EvaluateBeamCondition() must return true if a [Beam] satisfy that condition.
	EvaluateBeamCondition(beam Beam, container InitializingContext) bool
}

// BeamPostConstruct represent an interface with a PostConstruct function for a Singleton [Beam].
type BeamPostConstruct interface {
	// PostConstruct() is the function a Stereotyped Singleton struct must implement and the
	// container will call it once the Beam has been injected successfully.
	//
	// If an Error is returned, the container initialization will fail.
	PostConstruct() Error
}

// BeamPostConstructWithContext represent an interface with a PostConstruct function for a Prototype [Beam].
//
// This function will be called after [BeamPostConstruct.PostConstruct].
type BeamPostConstructWithContext interface {
	// WithContext() is the function a Stereotyped Prototype struct must implement and the
	// container will call it once the [Beam] has been injected successfully.
	//
	// If an Error is returned, the container initialization will fail.
	//
	// The [context.Context] will contain at least the following key/values :
	//
	//  - [PrototypeContextParentType] : [reflect.Type] of the parent object it is inject to.
	//
	//  - [PrototypeContextParentInstance] : (any) current instance of the parent object.
	//
	// Users can also populate the context with predefined key/values when used with a [Foundry].
	WithContext(context.Context) Error
}

// InnerBeamPostConstruct represent an interface with a PostConstruct function for a Composite struct.
type InnerBeamPostConstruct interface {
	// Stereotyped (parent) struct can also contain embedded (composite) struct with their own injected types.
	//
	// Those inner structs will be initialized first with a pointer to their parent struct using this function.
	PostConstruct(parent any) Error
}

// BeamRunner define an interface Beams may implement to have it called by the container function Run() after Initialize().
type BeamRunner interface {
	// The container will first need to be [ContainerConfiguration.Initialize] without errors and when
	// [ContainerConfiguration.Run] is called, this function will be called.
	Run(wg *sync.WaitGroup)
}

// BeamProvider is an interface representing a compile-in type which can provide Beams using a predefined stereotype.
//
// There is four main [Stereotype] : [Service], [Component], [Repository] and [Controller].
//
// There is two predefined [Stereotype] derived from [Component] : [Configuration] and [ConfigurationProperties].
//
// Those two marker interfaces are used by a struct implements [BeamProvider] and those stereotypes to have specialized [Beam].
type BeamProvider interface {
	ProvideBeam(reflect.Type, InitializingContext) (Beam, []Beam)
	// ParentBeam must return a [Beam] for the struct implementing that Stereotype.
	//
	// For example for [ConfigurationProperties], this function create a [Beam] with a specific foundry,
	// which call [InitializingContext.ConfigUnmarshall] on that object.
	// ParentBeam(reflect.Type, InitializingContext) Beam
	// AdditionalBeams may return additional [Beam] to be registered.
	//
	// For example for [Configuration], this function returns all the [Beam] defined by struct functions.
	// AdditionalBeams(Beam, InitializingContext) []Beam
}

// ConfigurationParser represent the interface an object parsing a filesystems for configuration files need to implement.
type ConfigurationParser interface {
	ParseConfigurationFilesystem(fsys fs.FS, loader func(m map[string]any, profile string))
}

// InitializingContext represent a limited set of container functions avaiiable during a container initializing.
// Mainly used by [BeamProvider] implementations.
type InitializingContext interface {
	// Add [ContainerContext] methods
	Environment

	// LoggerForClass() lets you retrieve a logger given a struct pointer.
	LoggerForClass(class any) Logging

	// LoggerForType() lets you retrieve a logger given a struct pointer reflect.Type;
	LoggerForType(rtype reflect.Type) Logging

	// FindBeams() find all the beams with a defined reflect.Type
	FindBeams(reflect.Type) []Beam

	// FindBeamsByAttributes() find all the beams by attributes, like "Qualifier" or "Primary".
	FindBeamsByAttributes(reflect.Type, map[string]string) []Beam

	// GetInstance() let you get a [Beam] created , injected and initialized instance.
	//
	// Mostly used inside [BeamProvider] to retrieve a parent Beam instance to call methods on it,
	// letting you create child [Beam].
	GetInstance(beam Beam) any
	// GetMethodAnnotedType() will analyze a reflect.Method to check for an anonymous struct in its first argument,
	// and return a reflect.Type (always of a struct pointer), and a reflect.Value (either struct or pointer of struct).
	//
	// The [reflect.Value] can be used directly as a [Beam.AnnotatedType], while the [reflect.Value] can be used as-is
	// as an argument to [reflect.Method.Call].
	//
	// This [reflect.Value] will not have it's field injected. To do that you have to call
	// [InitializingContext.InjectMethodAnnotedValue], from a the Beam foundry function.
	//
	// Mainly used by [BeamProvider] implementing struct.
	GetMethodAnnotedType(method *reflect.Method) (reflect.Type, reflect.Value)

	// InjectMethodAnnotedValue() will inject fields on a [reflect.Value].
	// For this to work, it should be done from the Beam foundry method.
	//
	// Mainly used by [BeamProvider] implementing struct.
	InjectMethodAnnotedValue(value reflect.Value, parent any) Error
}

// Environment let you access the container parameters, once the container is initialized.
//
// This interface may be injected.
type Environment interface {
	// ActiveProfiles() returns the list of active profiles
	ActiveProfiles() []string

	// ConfigValue() retrieve a single configuration value as a string, bool indicates if the value was found.
	ConfigValue(path string) (string, bool)

	// ConfigUnmarshall()
	ConfigUnmarshall(path string, object any) Error
}

// ContainerConfiguration represent the interface exposed by the container to configure and initialize it.
//
// It's mainly used from the [rays.ApplicationContext] function.
//
// From a main() application function it will :
//
//   - Lets you specify what profiles need to be activated
//
//   - Add [fs.FS] to be parsed
//
//   - Declare an override of a configuration key.
//
//   - Enable scanning (disabled by default)
//
//   - Set you own logging handler
//
//   - Initialize the container engine
//
//   - Finally Run all [BeamRunner].
//
// For a module developper it will let you define default configurations (not be erased after a test is runned).
type ContainerConfiguration interface {
	// Logger() lets you retrieve thez default Logging. Useful from main().
	Logger() Logging

	// SetLoggingHandler() lets define a different log handler, override the current log mecanism.
	SetLoggingHandler(handler func(io.Writer, *slog.HandlerOptions) slog.Handler)

	// SetActiveProfiles() set the list of active profiles
	SetActiveProfiles([]string) Error

	// AddOverride() will defined a configuration to override, useful for command-line defined overrides or environment variables.
	AddOverride(key, value string) Error

	// AddConfigFS() will add a FS to scan for configuration
	AddConfigFS(fs.FS) Error

	// AddDefaultConfigFS() loads default config in the provided FS.
	AddDefaultConfigFS(fs.FS) Error
	//
	// NOTE: Those will not be erased on container resets in testing.

	// LoadDefaultConfig() loads default config.
	//
	// NOTE: Those will not be erased on container resets in testing.
	//
	// WARNING: You should not deep down any map type other that map[string]any.
	LoadDefaultConfig(map[string]any) Error

	// LoadDefaultConfigAt() loads default config at a given hierarchy.
	//
	// WARNING: You should not deep down any map type other that map[string]any.
	LoadDefaultConfigAt(map[string]any, string) Error
	//
	// NOTE: Those will not be erased on container resets in testing.

	// CopyDefaultConfig() lets you duplicate a "default" configuration key.
	CopyDefaultConfig(from, to string) Error

	// CopyDefaultConfigForProfile() lets you duplicate a "default" configuration key for a profile only.
	CopyDefaultConfigForProfile(from, to, profile string) Error

	// Initialize() will initialize the container.
	Initialize() Error

	// Run will call all the BeamRunner Beams.
	Run()
}

// Transactional represent a transaction provider for a <T> type.
type Transactional[T any] interface {
	Start(context.Context) *T
	Transactional(ctx context.Context, action func(*T) Error)
	Get() *T
	End()
}

// Manager is a marker interface to differentiate a provider of a component type and
// a manager which proxy for many component of that type.
type Manager interface {
	_Manager()
}
