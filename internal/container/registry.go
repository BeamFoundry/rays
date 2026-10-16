package container

import (
	"context"
	"reflect"
	"sync"

	"github.com/BeamFoundry/rays/internal/logger"
	. "github.com/BeamFoundry/rays/pkg/core"
	. "github.com/BeamFoundry/rays/pkg/lang"
)

type Registry struct {
	beams     map[Beam]*Instance
	parent    map[Beam]Beam
	providers map[reflect.Type]reflect.Type
	lock      sync.RWMutex
}

func NewRegistry() *Registry {
	reg := &Registry{}
	reg.beams = make(map[Beam]*Instance)
	reg.parent = make(map[Beam]Beam)
	reg.providers = make(map[reflect.Type]reflect.Type)
	return reg
}

func (this *Registry) AddBeamFor(ptr any) (Beam, Error) {
	this.lock.Lock()
	defer this.lock.Unlock()
	var reflectType reflect.Type
	if rtype, ok := ptr.(reflect.Type); ok {
		reflectType = rtype
	} else {
		reflectType = reflect.TypeOf(ptr)
	}

	// TODO Verify beams are Singleton for parent if children are there
	if field := FindFieldForInterface[Stereotype](reflectType); field != nil {
		bpi := reflect.TypeFor[BeamProvider]()
		if bp, ok := this.providers[field.Type]; ok {
			logger.GetLogger().Trace("Cached provider", "stereotype", field.Type.Name(), "provider", bp)
			if provider, ok := reflect.New(bp.Elem()).Interface().(BeamProvider); ok {
				return this.registerProvider(reflectType, provider)
			}
		}
		if providers := FindTypesImplementingInterface(bpi, field.Type); len(providers) == 1 {
			if provider, ok := reflect.New(providers[0].Elem()).Interface().(BeamProvider); ok {
				logger.GetLogger().Trace("Found a provider", "stereotype", field.Type.Name(), "provider", providers[0])
				this.providers[field.Type] = providers[0]
				return this.registerProvider(reflectType, provider)
			}
		} else {
			logger.GetLogger().Trace("Provider not found", "type", field.Type.Name(), "reflectType", reflectType, "len", len(providers))
		}
	} else {
		logger.GetLogger().Trace("No stereotype found", "reflectType", reflectType)
	}
	builder, err := NewBeamBuilder(ptr)
	if err != nil {
		err.LogError()
		return nil, err
	}
	beam := builder.Build()
	if err := this.register(beam); err != nil {
		err.LogError()
		return nil, err
	}
	return beam, nil
}

func (this *Registry) CreateBeam(beam Beam) Error {
	this.lock.RLock()
	defer this.lock.RUnlock()
	if !this.isBeamCreatable(beam) {
		return NewError("InvalidBeamError", "Beam is already created")
	}

	if parent := this.GetBeamParent(beam); parent != nil {
		if this.getBeamDefaultInstance(parent).Status().Compare(InstanceStatus{}.New().INITIALIZED) < 0 {
			return NewError("InvalidBeamError", "Child beam and parent not yet initialized")
		} else if beam.Type() != beam.AnnotatedType() {
			for field := range beam.AnnotatedType().Elem().Fields() {
				logger.GetLogger().Trace("Annotated: Evaluating field to inject", "field.Name", field.Name, "beam", beam.Attributes()[QualifierAttribute])
				// TODO: Add parent type to call (instead of recurse)
				if !this.isFieldInjectable(&field, true) {
					return NewError("InvalidBeamError", "Annotated Beam not injectable", "field.Name", field.Name, "beam", beam.Attributes()[QualifierAttribute])
				}
			}
		}
	}
	instance := this.getBeamDefaultInstance(beam)
	if err := instance.Create(context.Background()); err != nil {
		if err.Name() != "NilPointer" {
			return err
		}

		logger.GetLogger().Debug("Beam instance creation returned a nil pointer, removing beam", "qualifier", beam.Attributes()[QualifierAttribute])
		this.unregisterBeam(beam)
	}
	return nil
}

func (this *Registry) FindBeams(beamType reflect.Type) []Beam {
	this.lock.RLock()
	defer this.lock.RUnlock()
	var beams []Beam
	for _, beam := range this.GetAllBeams() {
		if beam.Type().AssignableTo(beamType) {
			beams = append(beams, beam)
		}
	}
	return beams
}

func (this *Registry) FindBeamsByAttributes(beanType reflect.Type, attributes map[string]string) []Beam {
	var beams []Beam
	for _, beam := range this.FindBeams(beanType) {
		valid := true
		for k, v := range attributes {
			if val, ok := beam.Attributes()[k]; !ok {
				valid = false
			} else if val != v {
				valid = false
			}
		}
		if valid {
			beams = append(beams, beam)
		}
	}
	return beams
}

func (this *Registry) FindBeamsForStructField(field *reflect.StructField) ([]Beam, Error) {
	var attributes map[string]string
	var options map[string]string
	if af := NewAnnotationField(field); af != nil {
		attributes = af.Attributes()
		options = af.Options()
	} else {
		return nil, NewError("InvalidParameter", "Field is not injectable")
	}

	var multi bool
	var optional bool
	var many bool
	reflectType := field.Type

	switch field.Type.Kind() {
	case reflect.Struct:
		if Strings(field.Type.String()).HasPrefix("lang.Optional[") {
			if f, ok := reflectType.FieldByName("value"); ok {
				reflectType = f.Type
				optional = true
			}
		} else if Strings(field.Type.String()).HasPrefix("core.Foundry[") {
			if f, ok := reflectType.FieldByName("beamType"); ok {
				reflectType = f.Type
			}
		}
	case reflect.Slice:
		multi = true
		optional = true
		reflectType = field.Type.Elem()
	case reflect.Map:
		multi = true
		optional = true
		reflectType = field.Type.Elem()
		if reflectType.Kind() == reflect.Slice {
			many = true
			reflectType = reflectType.Elem()
		}
	default:
	}

	beams := this.FindBeamsByAttributes(reflectType, attributes)

	switch {
	case len(beams) == 0 && !optional:
		return beams, NewError("InjectionError", "Not enough candidates", "type", reflectType)
	case len(beams) > 1 && !multi:
		var primary Beam
		for _, beam := range beams {
			if this.IsPrimaryBeam(beam) {
				if primary != nil {
					return nil, NewError("InjectionError", "Only one beam of same type must be marked as 'primary'.", "type", reflectType)
				}
				primary = beam
			}
		}
		if primary != nil {
			beams = []Beam{primary}
		}
		if len(beams) > 1 {
			return nil, NewError("InjectionError", "More than one beam qualifies, you should mark one as primary.", "type", reflectType)
		}
	case len(beams) > 1 && multi && field.Type.Kind() == reflect.Map && !many:
		groupBy := QualifierAttribute
		if options["GroupBy"] != "" {
			groupBy = options["GroupBy"]
		}
		beamMap := map[string][]Beam{}

		for _, beam := range beams {
			attribute := beam.Attributes()[groupBy]
			if attribute == "" {
				def := options["Default"]
				if def == "" {
					continue
				}
				attribute = def
			}
			if _, ok := beamMap[attribute]; !ok {
				beamMap[attribute] = []Beam{beam}
				continue
			}
			return nil, NewError("InjectionError", "More thant one value for a map field. Make map element a slice or restrict on more attributes", "type", reflectType)
		}
	default:
	}

	return beams, nil
}

func (this *Registry) GetAllBeams() []Beam {
	this.lock.RLock()
	defer this.lock.RUnlock()
	var beams []Beam
	for beam := range this.beams {
		beams = append(beams, beam)
	}
	return beams
}

func (this *Registry) GetAllCreateableBeams() []Beam {
	var beams []Beam
	for _, beam := range this.GetAllBeams() {
		if this.isBeamCreatable(beam) {
			beams = append(beams, beam)
		}
	}
	return beams
}

func (this *Registry) GetAllInjectableBeams() []Beam {
	var beams []Beam
	for _, beam := range this.GetAllBeams() {
		if this.isBeamInjectable(beam) {
			beams = append(beams, beam)
		}
	}
	return beams
}

func (this *Registry) GetAllInitializableBeams() []Beam {
	var beams []Beam
	for _, beam := range this.GetAllBeams() {
		if this.isBeamInitializable(beam) {
			beams = append(beams, beam)
		}
	}
	return beams
}

func (this *Registry) GetBeamChildren(beam Beam) []Beam {
	beams := []Beam{}
	for child, parent := range this.parent {
		if parent == beam {
			beams = append(beams, child)
		}
	}
	return beams
}

//nolint:revive // ignore flag parameter
func (this *Registry) GetBeamCondition(beam Beam, includePost bool) BeamConditionEvaluator {
	reflectType := beam.AnnotatedType()

	conditional := reflect.TypeFor[BeamConditional]()
	evaluator := reflect.TypeFor[BeamConditionEvaluator]()
	conditions := []BeamConditionEvaluator{}
	postevent := reflect.TypeFor[PostEventFilter]()
	for field := range reflectType.Elem().Fields() {
		if field.Type.Implements(conditional) {
			if !includePost && field.Type.Implements(postevent) {
				return nil
			}
			if evaluators := FindTypesImplementingInterface(evaluator, field.Type); len(evaluators) == 1 {
				if eval, ok := reflect.New(evaluators[0].Elem()).Interface().(BeamConditionEvaluator); ok {
					conditions = append(conditions, eval)
				}
			} else {
				err := NewError("InvalidType", "Unable to find a Condition Evaluator", "evaluators", len(evaluators), "type", field.Type)
				err.LogError()
			}
		}
	}

	if len(conditions) > 0 {
		return AllCondition(conditions)
	}
	return nil
}

func (this *Registry) GetBeamDefaultInstance(beam Beam) *Instance {
	this.lock.RLock()
	defer this.lock.RUnlock()
	return this.getBeamDefaultInstance(beam)
}

func (this *Registry) GetBeamParent(beam Beam) Beam {
	if parent, ok := this.parent[beam]; ok {
		return parent
	}
	return nil
}

func (this *Registry) GetBeamScopedInstance(ctx context.Context, beam Beam, parent *Instance) (*Instance, Error) {
	this.lock.RLock()
	defer this.lock.RUnlock()
	if beam.Scope() == (BeamScope{}.New().SINGLETON) {
		return this.getBeamDefaultInstance(beam), nil
	}
	ctx = context.WithValue(ctx, PrototypeContextParentType, parent.beam.Type())
	ctx = context.WithValue(ctx, PrototypeContextParentInstance, parent.instance)
	beamInstance := NewInstance(beam)
	//nolint:contextcheck // False
	err := Try(func() {
		RaiseErr(beamInstance.Create(ctx))
		RaiseErr(beamInstance.Inject())
		RaiseErr(beamInstance.Initialize())
	})
	if err != nil {
		return nil, err
	}
	if this.isChildBeam(beam) {
		return beamInstance, nil
	}
	instance := beamInstance.Get()
	if post, ok := instance.(BeamPostConstructWithContext); ok {
		if err := post.WithContext(ctx); err != nil {
			return nil, err
		}
	}
	return beamInstance, nil
}

func (this *Registry) GetInstance(beam Beam) any {
	return this.GetBeamDefaultInstance(beam).Get()
}

func (this *Registry) InitializeBeam(beam Beam) Error {
	this.lock.RLock()
	defer this.lock.RUnlock()
	if !this.isBeamInitializable(beam) {
		return NewError("InvalidBeamError", "Beam is not yet initializable")
	}
	instance := this.getBeamDefaultInstance(beam)
	return instance.Initialize()
}

func (this *Registry) InjectBeam(beam Beam) Error {
	this.lock.RLock()
	defer this.lock.RUnlock()
	if !this.isBeamInjectable(beam) {
		return NewError("InvalidBeamError", "Beam is not yet injectable")
	}
	if beam.Type().Kind() == reflect.Pointer && beam.Type().Elem().Kind() == reflect.Struct {
		for field := range beam.Type().Elem().Fields() {
			logger.GetLogger().Trace("Evaluating field to inject", "field.Name", field.Name, "beam", beam.Attributes()[QualifierAttribute])
			// TODO: Add parent type to call (instead of recurse)
			if !this.isFieldInjectable(&field, true) {
				logger.GetLogger().Trace("Beam not injectable", "field.Name", field.Name, "beam", beam.Attributes()[QualifierAttribute])
				return NewError("InvalidBeamError", "Beam not injectable", "field.Name", field.Name, "beam", beam.Attributes()[QualifierAttribute])
			}
		}
	}

	instance := this.getBeamDefaultInstance(beam)
	return instance.Inject()
}

func (this *Registry) IsChildBeam(beam Beam) bool {
	this.lock.RLock()
	defer this.lock.RUnlock()
	return this.isChildBeam(beam)
}

func (this *Registry) IsParentBeam(beam Beam) bool {
	this.lock.RLock()
	defer this.lock.RUnlock()
	return this.isParentBeam(beam)
}

func (this *Registry) IsPrimaryBeam(beam Beam) bool {
	_, primary := beam.Attributes()["Primary"]
	return primary
}

func (this *Registry) Register(beam Beam) Error {
	this.lock.Lock()
	defer this.lock.Unlock()
	return this.register(beam)
}

func (this *Registry) Reset() {
	this.lock.Lock()
	defer this.lock.Unlock()
	clear(this.beams)
	clear(this.parent)
	clear(this.providers)
}

func (this *Registry) UnregisterBeam(beam Beam) {
	this.lock.Lock()
	defer this.lock.Unlock()
	this.unregisterBeam(beam)
}

func (this *Registry) getBeamDefaultInstance(beam Beam) *Instance {
	if instance, ok := this.beams[beam]; ok {
		return instance
	}
	return nil
}

func (this *Registry) isBeamCreatable(beam Beam) bool {
	instance := this.getBeamDefaultInstance(beam)
	return instance.Status().Compare(InstanceStatus{}.New().CREATED) < 0
}

func (this *Registry) isBeamInjectable(beam Beam) bool {
	instance := this.getBeamDefaultInstance(beam)
	return instance.Status().Compare(InstanceStatus{}.New().CREATED) == 0
}

func (this *Registry) isBeamInitializable(beam Beam) bool {
	instance := this.getBeamDefaultInstance(beam)
	return instance.Status().Compare(InstanceStatus{}.New().INJECTED) == 0
}

func (this *Registry) isChildBeam(beam Beam) bool {
	_, ok := this.parent[beam]
	return ok
}

//nolint:revive // flag-parameter
func (this *Registry) isFieldInjectable(field *reflect.StructField, recurse bool) bool {
	if recurse && field.Anonymous && field.Type.Kind() == reflect.Struct /* && field.Tag == "" */ {
		for f := range field.Type.Fields() {
			if !this.isFieldInjectable(&f, false) {
				return false
			}
		}
		return true
	}
	if beams, err := this.FindBeamsForStructField(field); err != nil {
		if err.Name() == "InvalidParameter" {
			logger.GetLogger().Trace("Field is not meant to be injected")
			return true
		}
		err.LogError()
		return false
	} else {
		for _, beam := range beams {
			if this.GetBeamDefaultInstance(beam).Status().Compare(InstanceStatus{}.New().CREATED) < 0 {
				logger.GetLogger().Trace("Found beam but it is not created", "beam", beam.Attributes()[QualifierAttribute])
				return false
			}
		}
	}
	return true
}

func (this *Registry) isParentBeam(beam Beam) bool {
	for _, parent := range this.parent {
		if parent == beam {
			return true
		}
	}
	return false
}

func (this *Registry) register(beam Beam) Error {
	if err := this.validateBeam(beam); err != nil {
		err.LogError()
		return err
	}
	if _, ok := this.beams[beam]; !ok {
		this.beams[beam] = NewInstance(beam)
	}

	return nil
}

func (this *Registry) registerProvider(reflectType reflect.Type, provider BeamProvider) (Beam, Error) {
	parent, beams := provider.ProvideBeam(reflectType, GetContainer())

	if parent == nil {
		return nil, NewError("InvalidType", "Bean must be a pointer to a struct")
	}

	if err := this.validateBeam(parent); err != nil {
		return nil, err
	}
	if _, ok := this.beams[parent]; !ok {
		this.beams[parent] = NewInstance(parent)
	}
	this.parent[Beam(nil)] = parent
	for _, beam := range beams {
		this.parent[beam] = parent
		if err := this.validateBeam(beam); err != nil {
			delete(this.parent, beam)
			return nil, err
		}
		if _, ok := this.beams[beam]; !ok {
			this.beams[beam] = NewInstance(beam)
		}
	}
	delete(this.parent, Beam(nil))
	return parent, nil
}

func (this *Registry) unregisterBeam(beam Beam) {
	delete(this.beams, beam)
	delete(this.parent, beam)
}

func (this *Registry) validateBeam(beam Beam) Error {
	if beam.Type().Kind() == reflect.Interface && this.isChildBeam(beam) {
		return nil
	}
	for field := range beam.Type().Elem().Fields() {
		if af := NewAnnotationField(&field); af == nil || !af.Inject() {
			continue
		}
		switch field.Type.Kind() {
		case reflect.Interface:
			continue
		case reflect.Pointer:
			if t := field.Type.Elem().Kind(); t == reflect.Struct {
				continue
			}
		case reflect.Slice:
			if t := field.Type.Elem().Kind(); t == reflect.Interface {
				continue
			} else if t == reflect.Pointer && field.Type.Elem().Elem().Kind() == reflect.Struct {
				continue
			}
		case reflect.Map:
			if field.Type.Key().Kind() == reflect.String {
				if field.Type.Elem().Kind() == reflect.Interface ||
					(field.Type.Elem().Kind() == reflect.Pointer &&
						field.Type.Elem().Elem().Kind() == reflect.Struct) ||
					(field.Type.Elem().Kind() == reflect.Slice &&
						(field.Type.Elem().Elem().Kind() == reflect.Interface ||
							(field.Type.Elem().Elem().Kind() == reflect.Pointer &&
								field.Type.Elem().Elem().Elem().Kind() == reflect.Struct))) {
					continue
				}
			}
		case reflect.Struct:
			if Strings(field.Type.String()).HasPrefix("lang.Optional[") {
				continue
			}
			if Strings(field.Type.String()).HasPrefix("core.Foundry[") {
				continue
			}
		default:
		}
		return NewError("InvalidFieldError", "Field type not valid", "qualifier", beam.Attributes()[QualifierAttribute], "field.Name", field.Name, "field.type", field.Type)
	}
	return nil
}

/*
func (this *Registry) IsBeamInitialized(beam *DefaultBeam) bool {
	instance := beam.Instance()
	return instance.Status() >= Initialized
}
*/

/*
func (this *Registry) SetBeamParent(beam, parent Beam) {
	this.parent[beam] = parent
}
*/
