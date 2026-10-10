package container

import (
	. "github.com/BeamFoundry/rays/pkg/core"
)

type PrimaryEvaluator struct {
	Primary
}

var _, _ = any(&PrimaryEvaluator{}).(BeamAttributeEvaluator)

var _ BeamAttributeEvaluator = &PrimaryEvaluator{}

// EvaluateBeamAttribute implements [BeamAttributeEvaluator].
func (this *PrimaryEvaluator) EvaluateBeamAttribute(_ Beam) string {
	return "primary"
}

type QualifierEvaluator struct {
	Qualifier
}

var _, _ = any(&QualifierEvaluator{}).(BeamAttributeEvaluator)

var _ BeamAttributeEvaluator = &QualifierEvaluator{}

// EvaluateBeamAttribute implements [BeamAttributeEvaluator].
func (q *QualifierEvaluator) EvaluateBeamAttribute(beam Beam) string {
	field := FindFieldWithAnnotation[Qualifier](beam.AnnotatedType())
	return GetFieldFirstTag(field, []string{"_", "Name", "Qualifier"})
}

type ServerEvaluator struct {
	Server
}

var _, _ = any(&ServerEvaluator{}).(BeamAttributeEvaluator)

var _ BeamAttributeEvaluator = &ServerEvaluator{}

// EvaluateBeamAttribute implements [BeamAttributeEvaluator].
func (q *ServerEvaluator) EvaluateBeamAttribute(beam Beam) string {
	// TODO: Validate / Explain why use FindFieldForInterface instead of
	// FindFieldWithAnnotation
	field := FindFieldForInterface[Server](beam.AnnotatedType())
	return GetFieldFirstTag(field, []string{"_", "Name", "Server"})
}

// TODO: Create a generic implementation
