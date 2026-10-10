package container

import (
	"reflect"
	"runtime/debug"
	"slices"
	"strconv"
	"strings"

	"github.com/BeamFoundry/rays/internal/logger"
	. "github.com/BeamFoundry/rays/pkg/core"
)

//---

type ConditionalOnProfileEvaluator struct {
	ConditionalOnProfile
	reflectType reflect.Type
}

var _, _ = any(&ConditionalOnProfileEvaluator{}).(BeamConditionEvaluator)

var _ BeamConditionEvaluator = &ConditionalOnProfileEvaluator{}

func (this *ConditionalOnProfileEvaluator) Profile() string {
	field := FindFieldWithAnnotation[ConditionalOnProfile](this.reflectType)
	return GetFieldFirstTag(field, []string{"Name", "Profile", "Profiles"})
}

func (this *ConditionalOnProfileEvaluator) EvaluateBeamCondition(beam Beam, environment InitializingContext) bool {
	this.reflectType = beam.AnnotatedType()
	if this.Profile() == "" {
		logger.GetLogger().Trace("Empty profile name, assuming true", "type", this.reflectType.Elem().Name())
		return true
	}
	for profile := range strings.SplitSeq(this.Profile(), ",") {
		if slices.Contains(environment.ActiveProfiles(), profile) {
			logger.GetLogger().Trace("Profile found", "type", this.reflectType.Name(), "profile", profile)
			return true
		}
		logger.GetLogger().Trace("Profile not found", "type", this.reflectType.Name(), "profile", profile)
	}
	return false
}

//---

type ConditionalOnPropertyEvaluator struct {
	ConditionalOnProperty
	reflectType reflect.Type
	log         *logger.Logger
}

var _, _ = any(&ConditionalOnPropertyEvaluator{}).(BeamConditionEvaluator)

var _ BeamConditionEvaluator = &ConditionalOnPropertyEvaluator{}

func (this *ConditionalOnPropertyEvaluator) Name() string {
	if field := FindFieldWithAnnotation[ConditionalOnProperty](this.reflectType); field != nil {
		if name, ok := field.Tag.Lookup("Name"); ok {
			return name
		}
	}
	return ""
}

func (this *ConditionalOnPropertyEvaluator) HavingValue() string {
	if field := FindFieldWithAnnotation[ConditionalOnProperty](this.reflectType); field != nil {
		if having, ok := field.Tag.Lookup("HavingValue"); ok {
			return having
		}
	}
	return ""
}

func (this *ConditionalOnPropertyEvaluator) MatchIfMissing() bool {
	if field := FindFieldWithAnnotation[ConditionalOnProperty](this.reflectType); field != nil {
		if matchIfMissing, ok := field.Tag.Lookup("MatchIfMissing"); ok {
			if boolValue, err := strconv.ParseBool(matchIfMissing); err == nil {
				this.log.Trace("Evaluating MatchIfMissing", "matchIfMissing", matchIfMissing)
				return boolValue
			}
		}
	}
	this.log.Trace("Evaluating MatchIfMissing tag not found")
	return false
}

func (this *ConditionalOnPropertyEvaluator) EvaluateBeamCondition(beam Beam, container InitializingContext) bool {
	this.reflectType = beam.AnnotatedType()
	name := this.Name()

	this.log = logger.GetLogger().With(
		"beam", beam.Type(),
		"tag", FindFieldWithAnnotation[ConditionalOnProperty](this.reflectType).Tag,
	)
	if name == "" {
		this.log.Trace("Name not found")
		return true
	}

	if value, ok := container.ConfigValue(name); ok {
		this.log.Trace("Config value found", "beam", beam.Type())
		if having := this.HavingValue(); having != "" {
			this.log.Trace("Evaluating having value", "having", having, "value", value)
			return having == value
		}

		boolValue, err := strconv.ParseBool(value)
		if err != nil {
			this.log.Trace("ParseBool failed, returning true", "value", value)
			return true
		}
		this.log.Trace("Returning ParseBool", "boolValue", boolValue)
		return boolValue
	}

	matchIfMissing := this.MatchIfMissing()
	this.log.Trace("Returning MatchIfMissing", "matchIfMissing", matchIfMissing)
	return matchIfMissing
}

//---

type ConditionalOnMissingBeamEvaluator struct {
	ConditionalOnMissingBeam
	reflectType reflect.Type
}

var _, _ = any(&ConditionalOnMissingBeamEvaluator{}).(BeamConditionEvaluator)

var _ BeamConditionEvaluator = &ConditionalOnMissingBeamEvaluator{}

func (this *ConditionalOnMissingBeamEvaluator) MissingBeam() any {
	return this.reflectType
}

func (this *ConditionalOnMissingBeamEvaluator) EvaluateBeamCondition(beam Beam, container InitializingContext) bool {
	this.reflectType = beam.Type()
	beams := container.FindBeams(this.MissingBeam().(reflect.Type))
	logger.GetLogger().Trace("Evaluating beam", "type", this.reflectType, "quantity", len(beams))

	if len(beams) > 1 {
		logger.GetLogger().Trace("Found different beam, returning false", "type", this.reflectType)
		return false
	}
	logger.GetLogger().Trace("Returning true", "type", this.reflectType)
	return true
}

//---

type CGODetector struct{}

func (this CGODetector) IsCGOEnabled() bool {
	var cgoEnabled bool
	if bi, ok := debug.ReadBuildInfo(); ok {
		for _, bs := range bi.Settings {
			if bs.Key == "CGO_ENABLED" && bs.Value == "1" {
				cgoEnabled = true
			}
		}
	}
	return cgoEnabled
}

type ConditionalOnNoCGOEvaluator struct {
	CGODetector
	ConditionalOnNoCGO
}

var _, _ = any(&ConditionalOnNoCGOEvaluator{}).(BeamConditionEvaluator)

var _ BeamConditionEvaluator = &ConditionalOnNoCGOEvaluator{}

func (this *ConditionalOnNoCGOEvaluator) EvaluateBeamCondition(_ Beam, _ InitializingContext) bool {
	return !this.IsCGOEnabled()
}

type ConditionalOnCGOEvaluator struct {
	CGODetector
	ConditionalOnCGO
}

var _, _ = any(&ConditionalOnCGOEvaluator{}).(BeamConditionEvaluator)

var _ BeamConditionEvaluator = &ConditionalOnCGOEvaluator{}

func (this *ConditionalOnCGOEvaluator) EvaluateBeamCondition(_ Beam, _ InitializingContext) bool {
	return this.IsCGOEnabled()
}

// ---
type ConditionOnAllEvaluator struct {
	BeamConditional
	conditions []BeamConditionEvaluator
}

func AllCondition(conditions []BeamConditionEvaluator) BeamConditionEvaluator {
	return &ConditionOnAllEvaluator{conditions: conditions}
}

func (this *ConditionOnAllEvaluator) EvaluateBeamCondition(beam Beam, environment InitializingContext) bool {
	for _, condition := range this.conditions {
		if !condition.EvaluateBeamCondition(beam, environment) {
			return false
		}
	}

	return true
}

//---

/*
type ConditionalOnAnyEvaluator struct {
	Conditional
	conditions []Condition
}

func AnyCondition(conditions []Condition) Condition {
	return &ConditionalOnAnyEvaluator{conditions: conditions}
}

func (this *ConditionalOnAnyEvaluator) Evaluate(reflectType reflect.Type, environment Environment) bool {
	for _, condition := range this.conditions {
		if condition.Evaluate(reflectType, environment) {
			return true
		}
	}

	return false
}
*/
