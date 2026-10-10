package container

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"reflect"
	"slices"
	"sync"
	"sync/atomic"

	. "github.com/BeamFoundry/rays/internal/config"
	"github.com/BeamFoundry/rays/internal/logger"
	. "github.com/BeamFoundry/rays/pkg/core"
	. "github.com/BeamFoundry/rays/pkg/lang"
)

type Container struct {
	*Registry
	*Config
	loggingProperties *LoggingProperties
	scanfor           reflect.Type
	handler           func(io.Writer, *slog.HandlerOptions) slog.Handler
	loggers           []string
	lock              sync.Mutex
	initialized       int32
	scanning          bool
	// filterScan  func(pkgName string) bool
}

var _ ContainerConfiguration = &Container{}

var _container = &Container{
	Registry: NewRegistry(),
	Config:   NewConfig(),
	scanfor:  reflect.TypeFor[Stereotype](),
	handler:  nil,
}

func GetContainer() *Container {
	return _container
}

func (this *Container) EnableScanning() {
	this.scanning = true
}

func (this *Container) AddBeamFor(ptr any) (Beam, Error) {
	if this.isInitialized() {
		return nil, NewError("InitializationError", "Container is already initialized")
	}
	return this.Registry.AddBeamFor(ptr)
}

func (this *Container) ConfigValue(key string) (string, bool) {
	v, err := this.GetConfigValue(key)
	if err != nil {
		return "", false
	}
	return v, true
}

func (this *Container) ConfigUnmarshall(path string, object any) Error {
	if sub, err := this.GetSubConfig(path); err != nil {
		err.LogError()
		return err
	} else if err := sub.Unmarshal(object); err != nil {
		return err
	}
	return nil
}

func (this *Container) SetLoggingHandler(handler func(io.Writer, *slog.HandlerOptions) slog.Handler) {
	this.handler = handler
}

func (this *Container) Logger() Logging {
	return &logging{logger.GetLogger()}
}

func (this *Container) LoggerForClass(class any) Logging {
	rtype := reflect.TypeOf(class)
	return this.LoggerForType(rtype)
}

func (this *Container) LoggerForType(rtype reflect.Type) Logging {
	if this.loggingProperties == nil || rtype.Kind() != reflect.Pointer || rtype.Elem().Kind() != reflect.Struct {
		return this.Logger()
	}

	pkg := rtype.Elem().PkgPath()
	name := pkg + "/" + rtype.Elem().Name()
	for _, level := range this.loggers {
		if pkg != level && name != level {
			continue
		}
		logger.GetLogger().Trace("Creating specific new logger", "pkg", pkg, "name", name)
		lg := logger.GetLogger().Clone()
		logLevel := this.loggingProperties.MapLevels()[level]
		lg.SetLogLevel(logLevel.Level())
		return &logging{lg}
	}
	return this.Logger()
}

func (this *Container) isInitialized() bool {
	return atomic.CompareAndSwapInt32(&this.initialized, 1, 1)
}

func (this *Container) SetScanFor(rtype reflect.Type) {
	this.lock.Lock()
	defer this.lock.Unlock()
	this.scanfor = rtype
}

func (this *Container) GetMethodAnnotedType(method *reflect.Method) (reflect.Type, reflect.Value) {
	if method.Type.NumIn() <= 1 {
		return nil, reflect.Value{}
	}
	t := method.Type.In(1)
	if t.Kind() == reflect.Struct {
		t = reflect.PointerTo(t)
	}
	if t.Kind() != reflect.Pointer || t.Elem().Kind() != reflect.Struct {
		return nil, reflect.Value{}
	}

	// Not an anonymous struct
	if t.Elem().Name() != "" || t.Elem().PkgPath() != "" {
		return nil, reflect.Value{}
	}

	value := reflect.New(t.Elem())
	annotatedType := value.Type()

	if method.Type.In(1).Kind() == reflect.Struct {
		value = value.Elem()
	}
	return annotatedType, value
}

// InjectMethodAnnotedValue() will inject a reflect.Value, as created by [GetMethodAnnotedType]
func (this *Container) InjectMethodAnnotedValue(value reflect.Value, parent any) Error {
	if value.Type().Kind() == reflect.Struct {
		if !value.CanAddr() {
			return NewError("IllegalArgumentError", "Unable to address struct")
		}
		value = value.Addr()
	}
	var beam Beam
	if beamBuilder, err := NewBeamBuilder(value.Interface()); err != nil {
		return err
	} else {
		beam = beamBuilder.Build()
	}
	err := Try(func() {
		instance := NewInstance(beam)
		RaiseErr(instance.Create(context.Background()))
		RaiseErr(instance.Inject())
		RaiseErr(instance.InitializeWithParent(parent))
	})
	return err
}

func (this *Container) Initialize() Error {
	this.lock.Lock()
	defer this.lock.Unlock()
	if this.isInitialized() {
		return NewError("InitializationError", "Container is already initialized")
	}

	// 1. Initializing configuration & logging
	// Initialize configuration
	//nolint:errcheck // It should not happen
	this.InitConfig()

	// Initializing logging
	if cfg, err := this.GetSubConfig("rays.logging"); err == nil {
		this.loggingProperties = &LoggingProperties{}
		err := cfg.Unmarshal(this.loggingProperties)
		if err != nil {
			return err
		}
		level := this.loggingProperties.LogLevel()
		logger.GetLogger().SetLogLevel(level.Level())
		if this.loggingProperties.AddSource() {
			logger.GetLogger().AddSource()
		}
		if this.handler != nil {
			logger.GetLogger().SetHandler(this.handler)
		} else if handler := this.loggingProperties.LogHandler(); handler.Handler() != nil {
			logger.GetLogger().SetHandler(handler.Handler())
		}
		this.loggers = slices.Sorted(maps.Keys(this.loggingProperties.MapLevels()))
		slices.Reverse(this.loggers)
	}
	logger.GetLogger().SetDefault()

	logger.GetLogger().Debug("Config & Logging initialized")

	// 2. Scan for Stereotype
	if this.scanning || flag.Lookup("test.v") == nil {
		bp := reflect.TypeFor[BeamProvider]()
		filterScan := func(t reflect.Type) bool {
			if t.Kind() == reflect.Struct && FindFieldWithAnnotation[IgnoreScanning](t) != nil {
				return true
			}
			if String(t.PkgPath()).Contains("test") {
				return true
			}
			return false
		}
		for _, t := range FindTypesImplementingInterface(this.scanfor) {
			if t.Implements(bp) {
				logger.GetLogger().Debug("Beam scanning skipping provider beam", "type", t.String())
				continue
			} else if filterScan(t.Elem()) && !filterScan(this.scanfor) {
				logger.GetLogger().Trace("Beam scanning skipping filtered beam", "type", t.String())
				continue
			}
			logger.GetLogger().Debug("Beam scanning found beam", "type", t.String())
			if _, err := this.AddBeamFor(t); err != nil {
				err.LogError()
			}
		}
	}

	// 3. Filter Conditional Beams
	this.filterConditionalBeams()

	// 4. Run Beam Initialization loop
	creatableBeams := this.GetAllCreateableBeams()
	creatable := len(creatableBeams)
	for creatable > 0 {
		var created int
		for _, beam := range creatableBeams {
			if err := this.CreateBeam(beam); err != nil {
				if err.Name() == "InvalidBeamError" {
					logger.GetLogger().Trace("Beam not yet createable", "message", err.Message(), "qualifier", beam.Attributes()["Qualifier"])
				} else {
					err.LogError()
				}
			} else {
				created++
			}
		}

		for _, beam := range this.GetAllInjectableBeams() {
			// If parent beam, initialize it now.
			if this.IsParentBeam(beam) {
				if err := this.InjectBeam(beam); err == nil {
					if err := this.InitializeBeam(beam); err != nil {
						err.LogError()
					} else {
						logger.GetLogger().Debug("Initializing parent beam", "type", beam.Type())
					}
				}
			}
		}

		creatableBeams = this.GetAllCreateableBeams()
		if created == 0 && len(creatableBeams) > 0 && len(creatableBeams) == creatable {
			err := NewError("BeamCreationError", fmt.Sprintf("Unable to create more beams, %d remaining", len(creatableBeams)))
			err.LogError()
			for _, beam := range creatableBeams {
				logger.GetLogger().Error("BeamCreationError: Beam not created", "qualifier", beam.Attributes()["Qualifier"], "type", beam.Type())
			}
			return err
		}
		creatable = len(creatableBeams)
	}

	//nolint:gocritic // May be needed to reactive.
	/*
		for _, beam := range this.GetAllBeams() {
			instance := this.GetBeamDefaultInstance(beam)
			if instance.Status() < Created {
				err := NewError("BeamCreationError", fmt.Sprintf("Beam not created : %s, %v", beam.Attributes()["Qualifier"], beam.Type()))
				err.LogError()
				return err
			}
		}
	*/

	logger.GetLogger().Info("All beam created")

	injectableBeams := this.GetAllInjectableBeams()
	injectable := len(injectableBeams)

	for injectable > 0 {
		var injected int

		for _, beam := range injectableBeams {
			if err := this.InjectBeam(beam); err == nil {
				injected++
			}
		}

		injectableBeams = this.GetAllInjectableBeams()
		if injected == 0 && len(injectableBeams) > 0 && len(injectableBeams) == injectable {
			err := NewError("BeamInjectionError", fmt.Sprintf("Unable to inject more beams, %d remaining", len(injectableBeams)))
			err.LogError()
			for _, beam := range injectableBeams {
				NewError("BeamInjectionError", "Unable to inject beam", "type", beam.Type()).LogError()
			}
			return err
		}
		injectable = len(injectableBeams)
	}

	//nolint:gocritic // May be needed to reactive.
	/*
		for _, beam := range this.GetAllBeams() {
			instance := this.GetBeamDefaultInstance(beam)
			if instance.Status() < Injected {
				err := NewError("BeamInjectError", fmt.Sprintf("Beam not injected : %s, %v", beam.Attributes()["Qualifier"], beam.Type()))
				err.LogError()
				return err
			}
		}
	*/
	logger.GetLogger().Info("All beam injected")

	this.initializeBeams(func(beam Beam) bool {
		if beam.Type().Kind() != reflect.Interface && FindFieldWithAnnotation[ConfigurationProperties](beam.Type()) != nil {
			logger.GetLogger().Debug("Initializing ConfigurationProperties beam", "type", beam.Type())
			return true
		}
		return false
	})
	this.initializeBeams(func(beam Beam) bool {
		init := false
		if this.IsChildBeam(beam) && FindFieldWithAnnotation[Configuration](this.GetBeamParent(beam).Type()) != nil {
			if beam.Type().Kind() == reflect.Interface {
				init = true
			} else if beam.Type().Kind() == reflect.Pointer && beam.Type().Elem().Kind() == reflect.Struct {
				if FindFieldWithAnnotation[Stereotype](beam.Type().Elem()) == nil {
					init = true
				}
			}
		}
		if init {
			logger.GetLogger().Debug("Initializing Configuration beam", "type", beam.AnnotatedType())
			return true
		}
		return false
	})
	this.initializeBeams(func(beam Beam) bool {
		if FindFieldForInterface[Component](beam.Type()) != nil {
			logger.GetLogger().Debug("Initializing Component beam", "type", beam.Type())
			return true
		}
		return false
	})
	this.initializeBeams(func(beam Beam) bool {
		if FindFieldForInterface[Repository](beam.Type()) != nil {
			logger.GetLogger().Debug("Initializing Repository beam", "type", beam.Type())
			return true
		}
		return false
	})
	this.initializeBeams(func(beam Beam) bool {
		if FindFieldForInterface[Service](beam.Type()) != nil {
			logger.GetLogger().Debug("Initializing Service beam", "type", beam.Type())
			return true
		}
		return false
	})
	this.initializeBeams(func(beam Beam) bool {
		if FindFieldForInterface[Controller](beam.Type()) != nil {
			logger.GetLogger().Debug("Initializing Controller beam", "type", beam.Type())
			return true
		}
		return false
	})
	this.initializeBeams(func(beam Beam) bool {
		if FindFieldForInterface[Stereotype](beam.Type()) != nil {
			logger.GetLogger().Debug("Initializing Stereotype beam", "type", beam.Type())
			return true
		}
		return false
	})

	this.initializeBeams(func(beam Beam) bool {
		logger.GetLogger().Trace("Initializing remaining beam", "type", beam.Type())
		return true
	})

	for _, beam := range this.GetAllBeams() {
		instance := this.GetBeamDefaultInstance(beam)
		if instance.Status().Compare(InstanceStatus{}.New().INITIALIZED) != 0 {
			return NewError("InitializationError", fmt.Sprintf("Beam not initialized : %s, %v", beam.Attributes()["Qualifier"], beam.Type()))
		}
	}

	logger.GetLogger().Info("All beam initialized")
	atomic.StoreInt32(&this.initialized, 1)
	return nil
}

func (this *Container) filterConditionalBeams() {
	filterConditionalBeams := func(includePostEvent bool) {
		// Remove "parent" beam if it does not evaluate his condition
		for _, beam := range this.GetAllBeams() {
			if this.IsChildBeam(beam) {
				continue
			}
			if condition := this.GetBeamCondition(beam, includePostEvent); condition != nil && !condition.EvaluateBeamCondition(beam, this) {
				logger.GetLogger().Debug("Removing (parent) beam has its condition evaluates to false", "type", beam.Type(), "qualifier", beam.Attributes()["Qualifier"])
				this.UnregisterBeam(beam)
				for _, child := range this.GetBeamChildren(beam) {
					logger.GetLogger().Debug("Removing (child) beam has its condition evaluates to false", "type", child.Type(), "qualifier", child.Attributes()["Qualifier"])
					this.UnregisterBeam(child)
				}
			}
		}

		// Remove remaining "child" beam if it does not evaluate his condition
		for _, beam := range this.GetAllBeams() {
			if !this.IsChildBeam(beam) {
				continue
			}
			if condition := this.GetBeamCondition(beam, includePostEvent); condition != nil && !condition.EvaluateBeamCondition(beam, this) {
				logger.GetLogger().Debug("Removing child beam has its condition evaluates to false", "type", beam.Type(), "qualifier", beam.Attributes()["Qualifier"])
				this.UnregisterBeam(beam)
			}
		}
	}
	filterConditionalBeams(false)
	filterConditionalBeams(true)
}

func (this *Container) initializeBeams(filter func(Beam) bool) {
	for _, beam := range this.GetAllInitializableBeams() {
		if filter(beam) {
			instance := this.GetBeamDefaultInstance(beam)
			if err := instance.Initialize(); err != nil {
				err.LogError()
			}
		}
	}
}

func (this *Container) Reset() {
	this.lock.Lock()
	defer this.lock.Unlock()
	this.Registry.Reset()
	this.Config.Reset()
	this.initialized = 0
	this.scanning = false
	this.scanfor = reflect.TypeFor[Stereotype]()
	this.handler = nil
	this.loggingProperties = nil
	this.loggers = []string{}
	logger.GetLogger().Reset()
}

func (this *Container) Run() {
	var wg sync.WaitGroup
	runner := reflect.TypeFor[BeamRunner]()

	for _, beam := range this.FindBeams(runner) {
		instance := this.GetBeamDefaultInstance(beam)
		if run, ok := instance.Get().(BeamRunner); ok {
			run.Run(&wg)
		}
	}

	wg.Wait()
}
