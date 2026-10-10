package core

// ConditionalOnProfile [BeamConditional] marker interface.
//
// This marker interface indicates if a [Beam] should be eligible for registration based on [InitializingContext] ActiveProfiles().
//
// Profile(s) should be specified as a comma-separated StructTag using either the tag "Name", "Profile" or "Profile".
//
//	// ConditionService will be instantiated if either "profile1" or "profile2" profiles are activated
//	type ConditionService struct {
//	  Component
//	  ConditionalOnProfile `Name:"profile1,profile2"`
//	}
type ConditionalOnProfile interface {
	BeamConditional
	_ConditionalOnProfile()
	_Profile() string // Alias for rag: Name or Profiles
}

// ConditionalOnMissingBeam [BeamConditional] marker interface.
//
// This marker interface indicates a [Beam] should be eligible for registration if no other [Beam] of the same type is actually registered.
//
// This works by delaying this condition to when all others conditional [Beam] have been evaluated and acts as a failback [Beam] registration.
//
// As this condition is evaluated on the [Beam] type (same struct type or same interface), it can be best used inside a [Configuration] Beam.
//
//	// DatabaseConfiguration will configure an `DBDriver` interface.
//	type DatabaseConfiguration struct {
//	  Configuration
//	}
//
//	// InMemoryDatabaseDriver will register a `*InMemoryDatabaseDriver` as a `DBDriver` interface,
//	// if the `Profile` "development" has been actived.
//	func (this *DatabaseConfiguration) InMemoryDatabaseDriver(
//	  _ struct {
//	    ConditionalOnProfile `Name:"developpement`
//	  },
//	) DBDriver {
//	  return &InMemorySQLDriver{}
//	}
//
//	// Fallback `&SQLDriver{}` as a `DBDriver` interface, if no other `DBDriver` has been registered.
//	func (this *DatabaseConfiguration) DatabaseDriver(
//	  _ struct {
//	    ConditionalOnMissingBeam
//	  },
//	) DBDriver {
//	  return &SQLDriver{}
//	}
type ConditionalOnMissingBeam interface {
	BeamConditional
	PostEventFilter
	_ConditionalOnMissingBeam()
	_MissingBeam() any
}

// ConditionaOnProperty [BeamConditional] marker interface.
//
// This marker interface indicates if a [Beam] should be eligible for registration based on a configuration property.
//
// This interface must have a StructTag "Name" indicating this property. By default it must be defined and not evaluate to "false" (using [strconv.ParseBool] rules).
//
// Instead of evaluating it as a "bool", a StructTag 'HavingValue" can be specified to indicate the value it should have.
//
// Furthermore if the property is not defined, and a StructTag 'MatchIfMissing' is provided with a boolean string expression (see [strconv.ParseBool), it will be checked to know
// if this condition matches.
//
// Given the following configuration :
//
//	app:
//	  audit:
//	    enabled: true
//
// And this code :
//
//	// Tracer interface
//	type Tracer interface {
//	  Trace(message string)
//	}
//
//	// AppTracerConfig will be registered only if "app.audit.enabled" evaluates to "true".
//	// The Beam it defines will not be evaluated and registered if not so.
//	type AppTracerConfig struct {
//	  Configuration
//	  ConditionalOnProperty `Name:"app.audit.enabled" MatchIfMissing:"false"`
//	}
//
//	// LogTracer will only be called if "app.audit.mode" exists
//	// and has the value "logging" or if does not exists at all.
//	func (a *AppTracerConfig) LogTracer(
//	  _ struct {
//	    ConditionalOnProperty `Name:"app.audit.mode" HavingValue="logging" MatchIfMissing:"true"`
//	  },
//	) Tracer {
//	  return &LogTracer{}
//	}
//
//	// KafkaTracer will only be called if "app.audit.mode" exists and equals to "kafka.
//	func (a *AppTracerConfig) KafkaTracer(
//	  _ struct {
//	    ConditionalOnProperty `Name:"app.audit.mode" HavingValue="kafka"`
//	  },
//	) Tracer {
//	  return &KafkaTracer{}
//	}
//
//	var _, _ = any(&AppTracerConfig{{}).(Stereotype)
//
//	// AppAuditor will be registered only if "app.audit.enabled" evaluates to "true".
//	// It will need a "Tracer" which will be provided by "AppTracerConfig"
//	type AppAuditor struct {
//	  Component
//	  ConditionalOnProperty `Name:"app.audit.enabled"`
//	  tracer Tracer `@:"Inject"`
//	}
//
//	func (a *AppTracer) Audit(message string) {
//	  a.tracer.Trace(message)
//	}
//
//	var _, _ = any(&AppAuditor{}).(Stereotype)
//
//	// A Service with an Optional AppAuditor
//	type AppService struct {
//	  Service
//	  auditor  Optional[*AppAuditor] `@:"Inject"`
//	}
//
//	func (a *AppService) PostConstruct() Error {
//	  if auditor, ok := a.auditor.Get(); ok {
//	    auditor.Audit("AppService PostConstruct")
//	  }
//	}
//
//	var _, _ = any(&AppService{]).(Stereotype)
//
// How this works :
//
//  1. AppTracerConfig will be registered, having its ConditionalOnProperty "app.audit.enabled" evaluating to true.
//
//     - It will register a "Tracer" Beam using LogTracer() because the property "app.audit.mode" is not defined.
//
//     - It will have done the same if "app.audit.mode" was defined with the value "logging".
//
//     - It will not define a "Tracer" Beam using KafkaTracer() as "app.audit.mode" is not defined and with a value of "kafka".
//
//  2. AppAuditor will be registered, as its ConditionalOnProperty "app.audit.enabled" is "true", and can be injected.
//
//  3. AppService will br injected with a "*AppAuditor".
type ConditionalOnProperty interface {
	BeamConditional
	_ConditionalOnProperty()
	_Name() string
	_MatchIfMissing() bool
	_HavingValue() string
}

// ConditionalOnNoCGO BeamConditional marker interface.
//
// This marker interface indicates if a [Beam] should eligible for registration based on the value of environment
// variable CGO_ENABLED. If this variable is not set to "1", this condition will apply.
type ConditionalOnNoCGO interface {
	BeamConditional
	_ConditionalOnNoCGO()
}

// ConditionalOnCGO BeamConditional marker interface.
//
// This marker interface indicates if a [Beam] should eligible for registration based on the value of environment
// variable CGO_ENABLED. If this variable is set to "1", this condition will apply.
type ConditionalOnCGO interface {
	BeamConditional
	_ConditionalOnCGO()
}
