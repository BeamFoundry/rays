package container

import (
	"context"
	"reflect"

	"github.com/BeamFoundry/rays/internal/logger"
	. "github.com/BeamFoundry/rays/pkg/core"
	. "github.com/BeamFoundry/rays/pkg/lang"
)

type InstanceStatuses struct {
	READY,
	CREATED,
	INJECTED,
	INITIALIZED Enum[InstanceStatuses]
}

type InstanceStatus = Enum[InstanceStatuses]

type Instance struct {
	beam     Beam
	instance any
	logger   *logger.Logger
	status   InstanceStatus
}

func NewInstance(beam Beam) *Instance {
	log := logger.GetLogger().WithGroup("beam")
	instance := &Instance{beam: beam, logger: log}
	return instance
}

func (this *Instance) Status() InstanceStatus {
	return this.status
}

//nolint:contextcheck //False positive
func (this *Instance) Create(ctx context.Context) Error {
	if this.status.Compare(InstanceStatus{}.New().CREATED) >= 0 {
		return NewError("InstanceError", "Instance already created")
	}
	this.logger.Trace("Creating", "qualifier", this.beam.Attributes()[QualifierAttribute], "type", this.beam.Type())
	instance, err := this.beam.Foundry()(ctx)
	if err != nil {
		return err
	}
	if (this.beam.Type().Kind() == reflect.Interface && instance == nil) ||
		reflect.ValueOf(instance).IsNil() {
		return NewError("NilPointer", "Instance created is nil")
	}
	this.instance = instance
	this.status = InstanceStatus{}.New().CREATED
	this.logger.Trace("Created", "qualifier", this.beam.Attributes()[QualifierAttribute], "type", this.beam.Type())
	return nil
}

func (this *Instance) Inject() Error {
	if this.status.Compare(InstanceStatus{}.New().INJECTED) >= 0 {
		return NewError("InstanceError", "Instance already injected")
	} else if this.status.Compare(InstanceStatus{}.New().CREATED) < 0 {
		return NewError("InstanceError", "Instance not created")
	}
	if err := this.injectBeams(); err != nil {
		return err
	}
	this.status = InstanceStatus{}.New().INJECTED
	this.logger.Trace("Injected", "qualifier", this.beam.Attributes()[QualifierAttribute], "type", this.beam.Type())
	return nil
}

func (this *Instance) injectBeams() Error {
	beamType := this.beam.Type()
	if beamType.Kind() == reflect.Interface && GetContainer().IsChildBeam(this.beam) {
		// return
		beamType = reflect.ValueOf(this.instance).Type()
	}
	for i := range beamType.Elem().NumField() {
		field := beamType.Elem().Field(i)
		valueOf := reflect.ValueOf(this.instance).Elem().Field(i)
		if err := this.injectField(&field, valueOf, true); err != nil {
			return err
		}
	}
	return nil
}

//nolint:errcheck,revive // It should not occurs
func (this *Instance) injectField(field *reflect.StructField, valueOf reflect.Value, recurse bool) Error {
	if recurse && field.Anonymous && field.Type.Kind() == reflect.Struct /* && field.Tag == "" */ {
		for i := range field.Type.NumField() {
			f := field.Type.Field(i)
			v := valueOf.Field(i)
			this.injectField(&f, v, false)
		}
		return nil
	}
	beams, err := GetContainer().FindBeamsForStructField(field)
	if err != nil && err.Name() == "InvalidParameter" {
		return nil
	}

	this.logger.Trace("Evaluating field", "field.Name", field.Name, "qualifier", this.beam.Attributes()[QualifierAttribute])

	//nolint:exhaustive // all valid cases whould be tested in FindBeamsForStructField
	switch field.Type.Kind() {
	case reflect.Pointer, reflect.Interface:
		// TODO: Verify this is still needed
		if !valueOf.IsNil() {
			return nil
		}
		beam := beams[0]
		instance, err := GetContainer().GetBeamScopedInstance(context.Background(), beam, this)
		// Can be trigged by a panic in a construct function.
		// Returning an error with Prototype constructor should do it also....
		if err != nil {
			err.LogErrorWithStackTrace()
			return err
		}
		value := UnprotectValue(valueOf)
		value.Set(reflect.ValueOf(instance.Get()))
		this.logger.Trace("Injected", "field.Name", field.Name, "type", this.beam.Type())
	case reflect.Slice:
		instances := []reflect.Value{}
		for _, beam := range beams {
			if instance, err := GetContainer().GetBeamScopedInstance(context.Background(), beam, this); err == nil {
				instances = append(instances, reflect.ValueOf(instance.Get()))
			}
		}
		value := UnprotectValue(valueOf)
		value.Set(reflect.MakeSlice(field.Type, len(instances), len(instances)))
		for i, instance := range instances {
			value.Index(i).Set(instance)
		}
	case reflect.Map:
		value := UnprotectValue(valueOf)
		m := reflect.MakeMap(field.Type)
		af := NewAnnotationField(field)
		groupBy := QualifierAttribute
		if af.Options()["GroupBy"] != "" {
			groupBy = af.Options()["GroupBy"]
		}
		instanceMap := map[string][]reflect.Value{}

		for _, beam := range beams {
			attribute := beam.Attributes()[groupBy]
			if attribute == "" {
				def := af.Options()["Default"]
				if def == "" {
					continue
				}
				attribute = def
			}
			if instance, err := GetContainer().GetBeamScopedInstance(context.Background(), beam, this); err == nil {
				if _, ok := instanceMap[attribute]; !ok {
					instanceMap[attribute] = []reflect.Value{reflect.ValueOf(instance.Get())}
				} else {
					instanceMap[attribute] = append(instanceMap[attribute], reflect.ValueOf(instance.Get()))
				}
			}
		}
		for key, instances := range instanceMap {
			switch field.Type.Elem().Kind() {
			case reflect.Slice:
				sl := reflect.MakeSlice(field.Type.Elem(), len(instances), len(instances))
				for i, instance := range instances {
					sl.Index(i).Set(instance)
				}
				m.SetMapIndex(reflect.ValueOf(key), sl)
			default:
				m.SetMapIndex(reflect.ValueOf(key), instances[0])
			}
		}
		value.Set(m)
	case reflect.Struct:
		if Strings(field.Type.String()).HasPrefix("lang.Optional[") && len(beams) == 1 {
			beam := beams[0]
			if instance, err := GetContainer().GetBeamScopedInstance(context.Background(), beam, this); err == nil {
				value := UnprotectValue(valueOf.FieldByName("value"))
				value.Set(reflect.ValueOf(instance.Get()))
				present := UnprotectValue(valueOf.FieldByName("present"))
				present.Set(reflect.ValueOf(true))
			}
		} else if Strings(field.Type.String()).HasPrefix("core.Foundry[") && len(beams) == 1 {
			beam := beams[0]
			furnace := UnprotectValue(valueOf.FieldByName("furnace"))

			factory := func(ctx context.Context) (any, Error) {
				instance, err := GetContainer().GetBeamScopedInstance(ctx, beam, this)
				if err != nil {
					return nil, err
				}
				return instance.Get(), nil
			}
			furnace.Set(reflect.ValueOf(factory))
		}
	}
	return nil
}

func (this *Instance) Initialize() Error {
	if this.status.Compare(InstanceStatus{}.New().INITIALIZED) >= 0 {
		return NewError("InstanceError", "Instance already initialized")
	} else if this.status.Compare(InstanceStatus{}.New().INJECTED) < 0 {
		return NewError("InstanceError", "Instance not injected")
	}

	if this.beam.Type().Kind() == reflect.Interface && GetContainer().IsChildBeam(this.beam) {
		this.status = InstanceStatus{}.New().INITIALIZED
		this.logger.Trace("Type is interface, returning.", "qualifier", this.beam.Attributes()[QualifierAttribute], "type", this.beam.Type())
		return nil
	}
	for i := range this.beam.Type().Elem().NumField() {
		field := this.beam.Type().Elem().Field(i)
		if field.Anonymous && field.Type.Kind() == reflect.Struct {
			if inner, ok := reflect.ValueOf(this.instance).Elem().Field(i).Addr().Interface().(InnerBeamPostConstruct); ok {
				if err := inner.PostConstruct(this.instance); err != nil {
					return err
				}
				this.logger.Trace("Inner field initialized", "qualifier", this.beam.Attributes()[QualifierAttribute], "type", this.beam.Type(), "field", field.Name)
			}
		}
	}

	if post, ok := this.instance.(BeamPostConstruct); ok {
		if err := post.PostConstruct(); err != nil {
			return err
		}
		this.logger.Debug("Initialized", "qualifier", this.beam.Attributes()[QualifierAttribute], "type", this.beam.Type())
	}

	this.status = InstanceStatus{}.New().INITIALIZED
	return nil
}

func (this *Instance) InitializeWithParent(parent any) Error {
	//nolint:gocritic // keeping that for now.
	/*
			if this.status.Compare(InstanceStatus{}.New().INITIALIZED) >= 0 {
				return NewError("InstanceError", "Instance already initialized")
			} else if this.status.Compare(InstanceStatus{}.New().INJECTED) < 0 {
				return NewError("InstanceError", "Instance not injected")
			}

		if this.beam.Type().Kind() == reflect.Interface && GetContainer().IsChildBeam(this.beam) {
			this.status = InstanceStatus{}.New().INITIALIZED
			this.logger.Trace("Type is interface, returning.", "qualifier", this.beam.Attributes()[QualifierAttribute], "type", this.beam.Type())
			return nil
		}
	*/
	for i := range this.beam.Type().Elem().NumField() {
		field := this.beam.Type().Elem().Field(i)
		if field.Anonymous && field.Type.Kind() == reflect.Struct {
			if inner, ok := reflect.ValueOf(this.instance).Elem().Field(i).Addr().Interface().(InnerBeamPostConstruct); ok {
				if err := inner.PostConstruct(parent); err != nil {
					return err
				}
				this.logger.Trace("Inner field initialized", "qualifier", this.beam.Attributes()[QualifierAttribute], "type", this.beam.Type(), "field", field.Name)
			}
		}
	}

	this.status = InstanceStatus{}.New().INITIALIZED
	return nil
}

func (this *Instance) Get() any {
	//nolint:gocritic // Leaving comment for now
	/*
		if this.status < InstanceInitialized {
		return nil, NewError("InstanceError", "Instance not initialized")
		}
	*/

	return this.instance
}
