package core

type IgnoreScanning interface {
	_IgnoreScanning()
}

// Stereotype base marker interface
//
// Stereotype is the kind of interface the container search in the runtime types available for
// Beam to register.
//
// It's a base type, not to be used. You should use [Component], [Service], [Configuration]... instead.
//
// You should note that the container does not make any difference between those [Stereotype],
// except when it has to call the [BeamPostConstruct.PostConstruct] function.
//
// It will be done in this order :
//   - [ConfigurationProperties]
//   - [Configuration]
//   - [Component]
//   - [Repository]
//   - [Service]
//   - [Controller]
//   - [Application]
//   - any other [Beam] implementing [Stereotype]
//   - any other [Beam]
type Stereotype interface {
	_Stereotype()
}

// Component [Stereotype] marker interface.
type Component interface {
	Stereotype
	_Component()
}

// Service [Stereotype] marker interface.
type Service interface {
	Stereotype
	_Service()
}

// Repository [Stereotype] marker interface.
type Repository interface {
	Stereotype
	_Repository()
}

// Controller [Stereotype] markker interface.
type Controller interface {
	Stereotype
	_Controller()
}

// Application [Stereotype] marker interface
type Application interface {
	Stereotype
	_Application()
}

// Configuration is a [Component] derived [Stereotype] marker interface.
//
// The container will scan components and define a [Beam] defining this struct, and any struct pointer or interface
// returned from public methods attached to this struct.
//
// It is mainly used to define [Beam] from struct not implementing a [Stereotype].
//
//	import (
//	    "database/sql"
//	    . "github.com/BeamFoundry/rays/pkg/core"
//	)
//
//	type DatabaseConfig struct {
//	    Configuration
//	}
//
//	// Forces the compiler to retain DatabaseConfig in the binary
//	// and verifies it implements the Stereotype interface.
//	var _, _ = any(&DatabaseConfig{}).(Stereotype)
//
//	// The container calls this method and registers the returned pointer as a Beam.
//	func (c *DatabaseConfig) ProvidePostgres() *sql.DB {
//	    db, err := sql.Open("postgres", "...")
//	    if err != nil {
//	        return nil
//	    }
//	    return db
//	}
//
// As with all [Stereotype] [Beam] it will be subject to [BeamConditional] loading.
//
// Each public method that will be considered must return a struct pointer or interface, and optionally an error
// or an [github.com/BeamFoundry/rays/pkg/lang.Error].
//
// If it only returns one element (struct pointer or interface), it will be checked for nil and not register a Beam in that case.
//
// If it returns two elements (struct pointer or interface, followed by an error or Error), the presence of the second element will be checked first
// invalidating the Beam registration and resulting in a container initialization error.
//
// [Conditional] and [BeamAttribute] (like [Qualifier]) may be defined in an anonymous struct defined as first argument to the method.
//
// For example, changing the [Qualifier], adding a [Condition] and returning optionally an error :
//
//	func (c *DatabaseConfig) ProvidePostgres(
//		_ struct {
//			Qualifier `Name:"DbDrive"` // change the Beam qualifier (it would have been "providePostgres")
//			ConditionalOnMissingBeam
//		},
//	) (*sql.DB, error) {
//		return sql.Open("postgres", "...")
//	}
//
// If no such anonymous struct defining a [Qualifier], the [Beam]'s [Qualifier] will be derived from the method's name (lowering the first letter).
//
// Note on [BeamConditional] inheritance on [Beam]:
//
// If there is no anonymous struct, [BeamConditional], if any, will be inherited from the struct definition.
//   - A [Configuration] Beam with [ConditionalOnMissingBeam] will always be unique and it will not apply in this case.
//     But a method not having an anonymous struct as first argument will inherit this [Condition] and registration may depends.
//   - A [Configuration] Beam with [ConditionalOnProfile] will be evaluated on this [Configuration] Beam and likewise will be the same for all the [Beam]
//     it tries to register.
type Configuration interface {
	Component
	_Configuration()
}

// ConfigurationProperties is a [Component] derived [Stereotype] marker interface. It is used to load configuration into a struct.
//
// During its initialization, the container will load its configuration, as defined from calls to the [github.com/BeamFoundry/rays.ApplicationContext],
// into struct defining this [Stereotype] interface.
//
// [ConfigurationProperties] members are to be defined as private (lowercase) as the container does not implements tag matching.
//
// The configuration key path to apply to this struct must be indicated, in a dot-separated format, in a "Name" struct tag to this marker interface.
//
// After that struct has to be injected into another [Stereotype] component like [Component] or [Service], in the same package.
//
// # Valid member types
//
//   - All primitive types like string, int32, float64 (including time.Duration), bool ... and string pointer (to check for nil)
//   - Struct or struct pointer with private (lowercase) members
//   - Struct implementing [TextUnmarshaler] (converting a string value to a struct)
//   - Slice of all primitive types and struct (see above)
//   - Map of string or map of TextUnmarshaler to all the above types and of those map type.
//
// # Validation
//
// Although [ConfigurationProperties] supports only private fields and the [github.com/go-playground/validator/v10.Validation] library only supports public fields,
// validation is supported.
//
// Using reflection a shadow struct will be constructed using public fields and be validated. If you use Cross-Field validation, remember to
// use fields with a uppercase first letter instead but errors will have the fields path/name with the "correct", private fields name.
//
// # Example
//
// Considering this YAML configuration loaded into the container:
//
//	app:
//	  kafka:
//	    brokers: ["localhost:9092"]
//	    timeout: "5000ms"
//	    retry: true
//
// A corresponding [ConfigurationProperties] beam and a [Service] beam loading the "app.kafka" configuration key path whould look like this:
//
//	// myrepo/myapp/services/kafka.go
//	package services
//
//	import (
//	    "fmt"
//	    "time"
//	    . "github.com/BeamFoundry/rays/pkg/core"
//	)
//
//	// 1. Define the Configuration
//	type KafkaConfig struct {
//	    ConfigurationProperties `Name:"app.kafka"`
//
//	    // Exact same-case matching is required.
//	    //
//	    // Ensure it's not empty and has valid hostname/port string values.
//	    brokers []string       `validate:"gt=0,dive,hostname_port"`
//	    // Parsing "5000ms" and validating it a minimum value
//	    timeout time.Duration  `validate:"min:100ms"`
//	    retry   bool
//	}
//
//	// 2. Inject it into a Service
//	type KafkaServer struct {
//	    Service
//	    config *KafkaConfig `@:"Inject"`
//	}
//
//	// 3. Initialize the service using the injected configuration
//	func (s *KafkaServer) PostConstruct() Error {
//	    fmt.Printf("Connecting to %d brokers with a timeout of %v\n", len(s.config.brokers), s.config.timeout)
//	    return nil
//	}
//
//	// Ensure detection by the compiler
//	var _, _ = any(&KafkaConfig{}).(Stereotype)
//	var _, _ = any(&KafkaServer{}).(Stereotype)
type ConfigurationProperties interface {
	Component
	_Property() string
}

// EventFilter is a base marker interface representing a filter for an event.
type EventFilter interface {
	_EventFilter()
}

// PreEventFilter [EventFilter] marker interface.
type PreEventFilter interface {
	EventFilter
	_PreEventFilter()
}

// PostEventFilter [EventFilter] marker interface.
//
// Used in [ConditionalOnMissingBeam], to indicate a [Conditional] that should be checked later, after most of all other.
type PostEventFilter interface {
	EventFilter
	_PostEventFilter()
}

// Primary is a [BeamAttribute] used to mark a Beam as primary.
//
// When more than one [Beam] can be detected to be injected on a simple field (not a slice or map), which would result in an error,
// if one of them has this attribute it will be selected and no error would be reported.
type Primary interface {
	BeamAttribute
	_Primary()
}

// Qualifier is a [BeamAttribute] used to indiquates a specific name for a Beam.
//
// May be used in [Configuration] method [Beam], anonymous struct first argument, as an alternate way to indicate
// the [Beam] Qualifier attribute, instead of relying on the method's name.
type Qualifier interface {
	BeamAttribute
	_Qualifier()
}

// Server is a [BeamAttribute] used to indiquates for which "server" (in a multi server environment), this [Beam] belongs to.
//
// It should defaults to "primary".
//
// Think of it as if you have two web server in you application one public named "primary", and one for "management" on another port.
//
// This let's you define [Controller]s for both servers. [Controller]s not havong this attributes should be considered for the "primary" server.
type Server interface {
	BeamAttribute
	_Server()
}
