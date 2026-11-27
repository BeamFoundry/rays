package container

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"reflect"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/BeamFoundry/rays/internal/boot"
	"github.com/BeamFoundry/rays/internal/services"
	"github.com/timandy/routine"

	. "github.com/BeamFoundry/rays/pkg/core"
	. "github.com/BeamFoundry/rays/pkg/lang"
)

const defaultShutdownTimeout = 30 * time.Second

type Container struct {
	*boot.BootLoader
	*Registry
	scanfor     reflect.Type
	shutdownErr Error
	runLocal    routine.ThreadLocal[bool]
	shutdownCh  chan struct{}
	cleanupDone chan struct{}
	lock        sync.Mutex
	runLock     sync.Mutex
	initialized atomic.Int32
	scanning    bool
	// filterScan  func(pkgName string) bool
}

var (
	_                   ContainerConfiguration = &Container{}
	notifySignalContext                        = signal.NotifyContext
	shutdownTimeout                            = defaultShutdownTimeout
)

var _container = &Container{
	BootLoader: boot.NewBootLoader(),
	Registry:   NewRegistry(),
	scanfor:    reflect.TypeFor[Stereotype](),
	runLocal:   routine.NewThreadLocal[bool](),
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

func (this *Container) isInitialized() bool {
	return this.initialized.CompareAndSwap(1, 1)
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

	// 1. Initializing configuration & logging via BootLoader
	if err := this.Boot(); err != nil {
		return err
	}

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
				services.Get().Logger.Debug("Beam scanning skipping provider beam", "type", t.String())
				continue
			} else if filterScan(t.Elem()) && !filterScan(this.scanfor) {
				services.Get().Logger.Trace("Beam scanning skipping filtered beam", "type", t.String())
				continue
			}
			services.Get().Logger.Debug("Beam scanning found beam", "type", t.String())
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
					services.Get().Logger.Trace("Beam not yet createable", "message", err.Message(), "qualifier", beam.Attributes()[QualifierAttribute])
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
						services.Get().Logger.Debug("Initializing parent beam", "type", beam.Type())
					}
				}
			}
		}

		creatableBeams = this.GetAllCreateableBeams()
		if created == 0 && len(creatableBeams) > 0 && len(creatableBeams) == creatable {
			err := NewError("BeamCreationError", fmt.Sprintf("Unable to create more beams, %d remaining", len(creatableBeams)))
			err.LogError()
			for _, beam := range creatableBeams {
				services.Get().Logger.Error("BeamCreationError: Beam not created", "qualifier", beam.Attributes()[QualifierAttribute], "type", beam.Type())
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
				err := NewError("BeamCreationError", fmt.Sprintf("Beam not created : %s, %v", beam.Attributes()[QualifierAttribute], beam.Type()))
				err.LogError()
				return err
			}
		}
	*/

	services.Get().Logger.Info("All beam created")

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
				err := NewError("BeamInjectError", fmt.Sprintf("Beam not injected : %s, %v", beam.Attributes()[QualifierAttribute], beam.Type()))
				err.LogError()
				return err
			}
		}
	*/
	services.Get().Logger.Info("All beam injected")

	this.initializeBeams(func(beam Beam) bool {
		if beam.Type().Kind() != reflect.Interface && FindFieldWithAnnotation[ConfigurationProperties](beam.Type()) != nil {
			services.Get().Logger.Debug("Initializing ConfigurationProperties beam", "type", beam.Type())
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
			services.Get().Logger.Debug("Initializing Configuration beam", "type", beam.AnnotatedType())
			return true
		}
		return false
	})
	this.initializeBeams(func(beam Beam) bool {
		if FindFieldForInterface[Component](beam.Type()) != nil {
			services.Get().Logger.Debug("Initializing Component beam", "type", beam.Type())
			return true
		}
		return false
	})
	this.initializeBeams(func(beam Beam) bool {
		if FindFieldForInterface[Repository](beam.Type()) != nil {
			services.Get().Logger.Debug("Initializing Repository beam", "type", beam.Type())
			return true
		}
		return false
	})
	this.initializeBeams(func(beam Beam) bool {
		if FindFieldForInterface[Service](beam.Type()) != nil {
			services.Get().Logger.Debug("Initializing Service beam", "type", beam.Type())
			return true
		}
		return false
	})
	this.initializeBeams(func(beam Beam) bool {
		if FindFieldForInterface[Controller](beam.Type()) != nil {
			services.Get().Logger.Debug("Initializing Controller beam", "type", beam.Type())
			return true
		}
		return false
	})
	this.initializeBeams(func(beam Beam) bool {
		if FindFieldForInterface[Stereotype](beam.Type()) != nil {
			services.Get().Logger.Debug("Initializing Stereotype beam", "type", beam.Type())
			return true
		}
		return false
	})

	this.initializeBeams(func(beam Beam) bool {
		services.Get().Logger.Trace("Initializing remaining beam", "type", beam.Type())
		return true
	})

	for _, beam := range this.GetAllBeams() {
		instance := this.GetBeamDefaultInstance(beam)
		if instance.Status().Compare(InstanceStatus{}.New().INITIALIZED) != 0 {
			return NewError("InitializationError", fmt.Sprintf("Beam not initialized : %s, %v", beam.Attributes()[QualifierAttribute], beam.Type()))
		}
	}

	services.Get().Logger.Info("All beam initialized")
	// Look for Scheduled tasks.
	if err := this.registerScheduledTasks(); err != nil {
		return err
	}

	// Set Initialized
	this.initialized.Store(1)
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
				services.Get().Logger.Debug("Removing (parent) beam has its condition evaluates to false", "type", beam.Type(), "qualifier", beam.Attributes()[QualifierAttribute])
				this.UnregisterBeam(beam)
				for _, child := range this.GetBeamChildren(beam) {
					services.Get().Logger.Debug("Removing (child) beam has its condition evaluates to false", "type", child.Type(), "qualifier", child.Attributes()[QualifierAttribute])
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
				services.Get().Logger.Debug("Removing child beam has its condition evaluates to false", "type", beam.Type(), "qualifier", beam.Attributes()[QualifierAttribute])
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

// WrapLoggingHandler registers a wrapper function to decorate the active slog.Handler
// before initialization completes. Calls after the container is initialized are ignored.
func (this *Container) WrapLoggingHandler(wrapper func(slog.Handler) slog.Handler) {
	if this.isInitialized() {
		return
	}
	this.BootLoader.WrapLoggingHandler(wrapper)
}

func (this *Container) Reset() {
	// Stop running tasks before acquiring Container.lock or resetting Registry so
	// active tasks cannot deadlock against Reset or observe half-reset state.
	this.Scheduler().Stop()

	this.lock.Lock()
	defer this.lock.Unlock()
	this.Registry.Reset()
	this.BootLoader.Reset()
	this.runLock.Lock()
	this.shutdownCh = nil
	this.cleanupDone = nil
	this.shutdownErr = nil
	this.runLock.Unlock()
	this.initialized.Store(0)
	this.scanning = false
	this.scanfor = reflect.TypeFor[Stereotype]()
}

func (this *Container) Run() {
	if !this.isInitialized() {
		services.Get().Logger.Error("Container is not initialized, aborting Run()")
		return
	}

	this.runLocal.Set(true)
	defer this.runLocal.Remove()

	ctx, stopSignals := notifySignalContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()

	shutdownCh := make(chan struct{})
	cleanupDone := make(chan struct{})

	this.runLock.Lock()
	this.shutdownCh = shutdownCh
	this.cleanupDone = cleanupDone
	this.shutdownErr = nil
	this.runLock.Unlock()

	// Start the scheduler synchronously before launching BeamRunners so recurring
	// Schedule*Task calls are rejected and dynamic LaunchTask calls execute immediately.
	// Only add Wait to wg when tasks were registered prior to Run().
	sched := this.Scheduler()
	hasTasks := sched.HasTasks()
	sched.StartAsync()

	var wg sync.WaitGroup
	runner := reflect.TypeFor[BeamRunner]()

runLoop:
	for _, beam := range this.FindBeams(runner) {
		instance := this.GetBeamDefaultInstance(beam)
		if run, ok := instance.Get().(BeamRunner); ok {
			run.Run(&wg)
			select {
			case <-shutdownCh:
				break runLoop
			default:
			}
		}
	}

	if hasTasks {
		wg.Go(sched.Wait)
	}

	// Waiting for all runners in a goroutine so OS signals or Shutdown() can unblock Run()
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
	case <-shutdownCh:
	}

	err := this.cleanup()

	waitCtx, cancelWait := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancelWait()
	select {
	case <-done:
	case <-waitCtx.Done():
		services.Get().Logger.Warn("Container shutdown timed out waiting for runners to exit")
	}

	this.runLock.Lock()
	this.shutdownErr = err
	this.shutdownCh = nil
	this.runLock.Unlock()
	close(cleanupDone)
}

// Shutdown triggers and waits for container teardown (stopping the scheduler and invoking
// [BeamPreDestroy.PreDestroy] followed by [BeamShutdown.Shutdown] on initialized singleton Beams).
//
// Teardown executes at most once per initialization cycle (until [Container.Reset] is called):
//   - If [Container.Run] is active, Shutdown unblocks Run and waits for its teardown to complete
//     (unless invoked from inside a scheduler worker or synchronously from Run, in which case it
//     signals teardown and returns nil immediately without deadlocking on its own completion).
//   - If [Container.Run] has already completed or Shutdown was already called, Shutdown returns
//     the recorded teardown error without rerunning singleton hooks.
//   - If [Container.Run] was never called, Shutdown runs teardown directly once and records the result.
func (this *Container) Shutdown() Error {
	this.runLock.Lock()
	done := this.cleanupDone
	if done != nil {
		if this.shutdownCh != nil {
			close(this.shutdownCh)
			this.shutdownCh = nil
		}
		this.runLock.Unlock()
		if this.Scheduler().InWorker() || this.runLocal.Get() {
			return nil
		}
		<-done
		this.runLock.Lock()
		err := this.shutdownErr
		this.runLock.Unlock()
		return err
	}

	cleanupDone := make(chan struct{})
	this.cleanupDone = cleanupDone
	this.runLock.Unlock()

	err := this.cleanup()

	this.runLock.Lock()
	this.shutdownErr = err
	this.runLock.Unlock()
	close(cleanupDone)
	return err
}

// cleanup stops the scheduler and then invokes [BeamPreDestroy.PreDestroy] followed by
// [BeamShutdown.Shutdown] on all initialized singleton Beams.
func (this *Container) cleanup() Error {
	this.Scheduler().Stop()

	var firstErr Error
	singleton := BeamScope{}.New().SINGLETON
	initialized := InstanceStatus{}.New().INITIALIZED

	var initializedSingletons []any
	for _, beam := range this.GetAllBeams() {
		if beam.Scope() != singleton {
			continue
		}
		instance := this.GetBeamDefaultInstance(beam)
		if instance == nil || instance.Status().Compare(initialized) < 0 {
			continue
		}
		initializedSingletons = append(initializedSingletons, instance.Get())
	}

	for _, obj := range initializedSingletons {
		if pd, ok := obj.(BeamPreDestroy); ok {
			if err := pd.PreDestroy(); err != nil {
				err.LogError()
				if firstErr == nil {
					firstErr = err
				}
			}
		}
	}

	for _, obj := range initializedSingletons {
		if sd, ok := obj.(BeamShutdown); ok {
			if err := sd.Shutdown(); err != nil {
				err.LogError()
				if firstErr == nil {
					firstErr = err
				}
			}
		}
	}

	return firstErr
}
