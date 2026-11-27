package container_test

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/BeamFoundry/rays/internal/container"
	. "github.com/BeamFoundry/rays/pkg/core"
	. "github.com/BeamFoundry/rays/pkg/lang"
	. "github.com/BeamFoundry/rays/pkg/testing"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type preDestroyBeam struct {
	Component
	IgnoreScanning
	recorded  *[]string
	err       Error
	destroyed bool
}

func (this *preDestroyBeam) PreDestroy() Error {
	this.destroyed = true
	if this.recorded != nil && this.err != nil {
		*this.recorded = append(*this.recorded, this.err.Message())
	}
	return this.err
}

type preDestroyProtoBeam struct {
	Component
	Prototype
	IgnoreScanning
	destroyed bool
}

func (this *preDestroyProtoBeam) PreDestroy() Error {
	this.destroyed = true
	return nil
}

type runnerWithPreDestroy struct {
	Component
	IgnoreScanning
	ran       bool
	destroyed bool
	shutDown  bool
}

func (this *runnerWithPreDestroy) Run(wg *sync.WaitGroup) {
	wg.Go(func() {
		this.ran = true
	})
}

func (this *runnerWithPreDestroy) PreDestroy() Error {
	this.destroyed = true
	return nil
}

func (this *runnerWithPreDestroy) Shutdown() Error {
	this.shutDown = true
	return nil
}

type blockingRunnerBeam struct {
	Component
	IgnoreScanning
	started  chan struct{}
	released chan struct{}
}

func (this *blockingRunnerBeam) Run(wg *sync.WaitGroup) {
	wg.Go(func() {
		close(this.started)
		<-this.released
	})
}

func (this *blockingRunnerBeam) Shutdown() Error {
	close(this.released)
	return nil
}

type orderedLifecycleBeam struct {
	Component
	IgnoreScanning
	mu             *sync.Mutex
	events         *[]string
	preDestroyGate chan struct{}
	preDestroyWait chan struct{}
	shutdownErr    Error
}

func (this *orderedLifecycleBeam) PreDestroy() Error {
	if this.preDestroyGate != nil {
		close(this.preDestroyGate)
	}
	if this.preDestroyWait != nil {
		<-this.preDestroyWait
	}
	this.mu.Lock()
	*this.events = append(*this.events, "PreDestroy")
	this.mu.Unlock()
	return nil
}

func (this *orderedLifecycleBeam) Shutdown() Error {
	this.mu.Lock()
	*this.events = append(*this.events, "Shutdown")
	this.mu.Unlock()
	return this.shutdownErr
}

type shutdownErrBeam struct {
	Component
	IgnoreScanning
	err    Error
	called bool
}

func (this *shutdownErrBeam) Shutdown() Error {
	this.called = true
	return this.err
}

type ifaceOnly interface {
	DoWork()
}

type concreteWithPreDestroy struct {
	destroyed bool
}

func (this *concreteWithPreDestroy) DoWork() {}

func (this *concreteWithPreDestroy) PreDestroy() Error {
	this.destroyed = true
	return nil
}

type ifaceConfig struct {
	Configuration
	IgnoreScanning
}

func (this *ifaceConfig) ProvideIface() ifaceOnly {
	return &concreteWithPreDestroy{}
}

var _ = Describe("Container Shutdown", func() {
	It("invokes PreDestroy and Shutdown on initialized singletons, skips prototypes and uninitialized beams, and returns first recorded error", func() {
		var recorded []string
		first := &preDestroyBeam{err: NewError("ShutdownError", "first failure"), recorded: &recorded}
		second := &preDestroyBeam{err: NewError("ShutdownError", "second failure"), recorded: &recorded}
		sdBeam := &shutdownErrBeam{err: NewError("ShutdownError", "shutdown failure")}
		proto := &preDestroyProtoBeam{}
		uninit := &preDestroyBeam{}

		_, err := GetContainer().AddBeamFor(first)
		Expect(err).NotTo(HaveOccurred())
		_, err = GetContainer().AddBeamFor(second)
		Expect(err).NotTo(HaveOccurred())
		_, err = GetContainer().AddBeamFor(sdBeam)
		Expect(err).NotTo(HaveOccurred())
		_, err = GetContainer().AddBeamFor(proto)
		Expect(err).NotTo(HaveOccurred())

		Expect(GetContainer().Initialize()).NotTo(HaveOccurred())

		// Register an uninitialized beam after Initialize to verify uninitialized instances are skipped.
		uninitBeam, err := container.GetContainer().Registry.AddBeamFor(uninit)
		Expect(err).NotTo(HaveOccurred())
		Expect(uninitBeam).NotTo(BeNil())

		shutdownErr := GetContainer().Shutdown()
		Expect(shutdownErr).To(HaveOccurred())
		Expect(first.destroyed).To(BeTrue())
		Expect(second.destroyed).To(BeTrue())
		Expect(sdBeam.called).To(BeTrue())
		Expect(recorded).To(HaveLen(2))
		Expect(shutdownErr.Message()).To(Equal(recorded[0]))
		Expect(proto.destroyed).To(BeFalse())
		Expect(uninit.destroyed).To(BeFalse())
	})

	It("records BeamShutdown error when PreDestroy succeeds", func() {
		sdBeam := &shutdownErrBeam{err: NewError("ShutdownError", "only shutdown failed")}
		_, err := GetContainer().AddBeamFor(sdBeam)
		Expect(err).NotTo(HaveOccurred())
		Expect(GetContainer().Initialize()).NotTo(HaveOccurred())

		shutdownErr := GetContainer().Shutdown()
		Expect(shutdownErr).To(HaveOccurred())
		Expect(shutdownErr.Message()).To(Equal("only shutdown failed"))
		Expect(sdBeam.called).To(BeTrue())
	})

	It("invokes PreDestroy when declared Beam.Type() is an interface not embedding BeamPreDestroy", func() {
		cfg := &ifaceConfig{}
		_, err := GetContainer().AddBeamFor(cfg)
		Expect(err).NotTo(HaveOccurred())
		Expect(GetContainer().Initialize()).NotTo(HaveOccurred())

		provided := GetBeamDefaultInstance[ifaceOnly]()
		concrete, ok := provided.(*concreteWithPreDestroy)
		Expect(ok).To(BeTrue())

		Expect(GetContainer().Shutdown()).NotTo(HaveOccurred())
		Expect(concrete.destroyed).To(BeTrue())
	})

	It("invokes cleanup automatically after Run completes", func() {
		runner := &runnerWithPreDestroy{}
		_, err := GetContainer().AddBeamFor(runner)
		Expect(err).NotTo(HaveOccurred())
		Expect(GetContainer().Initialize()).NotTo(HaveOccurred())

		GetContainer().Run()
		Expect(runner.ran).To(BeTrue())
		Expect(runner.destroyed).To(BeTrue())
		Expect(runner.shutDown).To(BeTrue())
	})

	It("unblocks Run on OS signal and runs PreDestroy before Shutdown", func() {
		sigCtx, cancelSig := context.WithCancel(context.Background())
		restore := container.SetNotifySignalContext(func(_ context.Context, _ ...os.Signal) (context.Context, context.CancelFunc) {
			return sigCtx, cancelSig
		})
		defer restore()

		var mu sync.Mutex
		var events []string
		ordered := &orderedLifecycleBeam{mu: &mu, events: &events}
		runner := &blockingRunnerBeam{
			started:  make(chan struct{}),
			released: make(chan struct{}),
		}

		_, err := GetContainer().AddBeamFor(ordered)
		Expect(err).NotTo(HaveOccurred())
		_, err = GetContainer().AddBeamFor(runner)
		Expect(err).NotTo(HaveOccurred())
		Expect(GetContainer().Initialize()).NotTo(HaveOccurred())

		runDone := make(chan struct{})
		go func() {
			GetContainer().Run()
			close(runDone)
		}()

		Eventually(runner.started, time.Second).Should(BeClosed())
		cancelSig()
		Eventually(runDone, time.Second).Should(BeClosed())

		mu.Lock()
		Expect(events).To(Equal([]string{"PreDestroy", "Shutdown"}))
		mu.Unlock()
	})

	It("unblocks Run via concurrent Shutdown calls and stops Scheduler before PreDestroy and Shutdown", func() {
		var mu sync.Mutex
		var events []string
		var taskRunning atomic.Bool
		taskStarted := make(chan struct{})

		gate := make(chan struct{})
		wait := make(chan struct{})
		ordered := &orderedLifecycleBeam{
			mu:             &mu,
			events:         &events,
			preDestroyGate: gate,
			preDestroyWait: wait,
			shutdownErr:    NewError("ShutdownError", "final shutdown error"),
		}
		_, err := GetContainer().AddBeamFor(ordered)
		Expect(err).NotTo(HaveOccurred())
		Expect(GetContainer().Initialize()).NotTo(HaveOccurred())

		// Queue a task on the scheduler that sleeps briefly so we can verify Scheduler.Stop waits for it
		Expect(container.GetContainer().Scheduler().LaunchTask(func() {
			taskRunning.Store(true)
			close(taskStarted)
			time.Sleep(30 * time.Millisecond)
			mu.Lock()
			events = append(events, "TaskDone")
			mu.Unlock()
			taskRunning.Store(false)
		})).NotTo(HaveOccurred())

		runDone := make(chan struct{})
		go func() {
			GetContainer().Run()
			close(runDone)
		}()

		Eventually(taskStarted, time.Second).Should(BeClosed())

		// First Shutdown unblocks Run() and enters cleanup -> blocks in PreDestroy at gate/wait
		errCh1 := make(chan Error, 1)
		go func() {
			errCh1 <- GetContainer().Shutdown()
		}()

		// Wait until PreDestroy is active, then issue a second concurrent Shutdown()
		Eventually(gate, time.Second).Should(BeClosed())
		Expect(taskRunning.Load()).To(BeFalse())

		errCh2 := make(chan Error, 1)
		go func() {
			errCh2 <- GetContainer().Shutdown()
		}()

		time.Sleep(15 * time.Millisecond)
		close(wait)

		Eventually(runDone, time.Second).Should(BeClosed())
		var err1, err2 Error
		Eventually(errCh1, time.Second).Should(Receive(&err1))
		Eventually(errCh2, time.Second).Should(Receive(&err2))
		Expect(err1).To(HaveOccurred())
		Expect(err1.Message()).To(Equal("final shutdown error"))
		Expect(err2).To(Equal(err1))

		mu.Lock()
		Expect(events).To(Equal([]string{"TaskDone", "PreDestroy", "Shutdown"}))
		mu.Unlock()
	})

	It("aborts Run immediately when the container has not been initialized", func() {
		runner := &runnerWithPreDestroy{}
		_, err := GetContainer().AddBeamFor(runner)
		Expect(err).NotTo(HaveOccurred())

		GetContainer().Run()
		Expect(runner.ran).To(BeFalse())
		Expect(runner.destroyed).To(BeFalse())
		Expect(runner.shutDown).To(BeFalse())
	})

	It("times out cleanly when a BeamRunner does not exit during Shutdown", func() {
		restoreTimeout := container.SetShutdownTimeout(20 * time.Millisecond)
		defer restoreTimeout()

		started := make(chan struct{})
		release := make(chan struct{})
		defer close(release)

		runner := &stuckRunnerBeam{started: started, release: release}
		_, err := GetContainer().AddBeamFor(runner)
		Expect(err).NotTo(HaveOccurred())
		Expect(GetContainer().Initialize()).NotTo(HaveOccurred())

		runDone := make(chan struct{})
		go func() {
			GetContainer().Run()
			close(runDone)
		}()

		Eventually(started, time.Second).Should(BeClosed())
		Expect(GetContainer().Shutdown()).NotTo(HaveOccurred())
		Eventually(runDone, time.Second).Should(BeClosed())
	})

	It("allows WrapLoggingHandler during PostConstruct and ignores calls after Initialize completes", func() {
		var postConstructCalls, postInitCalls atomic.Int32
		beam := &wrapInPostConstructBeam{calls: &postConstructCalls}
		_, err := GetContainer().AddBeamFor(beam)
		Expect(err).NotTo(HaveOccurred())
		Expect(GetContainer().Initialize()).NotTo(HaveOccurred())

		GetContainer().WrapLoggingHandler(func(next slog.Handler) slog.Handler {
			postInitCalls.Add(1)
			return next
		})

		GetContainer().Logger().Info("probe wrapper")
		Expect(postConstructCalls.Load()).To(BeNumerically(">=", int32(1)))
		Expect(postInitCalls.Load()).To(Equal(int32(0)))
	})

	It("allows a scheduled callback to request Shutdown without deadlocking Scheduler.Stop", func() {
		var mu sync.Mutex
		var events []string
		beam := &scheduledShutdownBeam{mu: &mu, events: &events}
		_, err := GetContainer().AddBeamFor(beam)
		Expect(err).NotTo(HaveOccurred())
		Expect(GetContainer().Initialize()).NotTo(HaveOccurred())

		runDone := make(chan struct{})
		go func() {
			GetContainer().Run()
			close(runDone)
		}()

		Eventually(runDone, time.Second).Should(BeClosed())
		Expect(beam.shutdownErr).NotTo(HaveOccurred())

		mu.Lock()
		Expect(events).To(Equal([]string{"CallbackBeforeReturn", "PreDestroy", "Shutdown"}))
		mu.Unlock()
	})

	It("marks scheduler started even with no scheduled tasks, rejecting Schedule* and running LaunchTask during Run", func() {
		runner := &runtimeSchedulerRunnerBeam{}
		_, err := GetContainer().AddBeamFor(runner)
		Expect(err).NotTo(HaveOccurred())
		Expect(GetContainer().Initialize()).NotTo(HaveOccurred())

		GetContainer().Run()
		Expect(runner.scheduleErr).To(HaveOccurred())
		Expect(runner.launchErr).NotTo(HaveOccurred())
		Expect(runner.launchedRan.Load()).To(BeTrue())
	})
})

type scheduledShutdownBeam struct {
	Service
	IgnoreScanning
	mu          *sync.Mutex
	events      *[]string
	shutdownErr Error
}

func (this *scheduledShutdownBeam) TriggerShutdown(
	_ struct {
		Scheduled
	},
) {
	this.shutdownErr = GetContainer().Shutdown()
	time.Sleep(20 * time.Millisecond)
	this.mu.Lock()
	*this.events = append(*this.events, "CallbackBeforeReturn")
	this.mu.Unlock()
}

func (this *scheduledShutdownBeam) PreDestroy() Error {
	this.mu.Lock()
	*this.events = append(*this.events, "PreDestroy")
	this.mu.Unlock()
	return nil
}

func (this *scheduledShutdownBeam) Shutdown() Error {
	this.mu.Lock()
	*this.events = append(*this.events, "Shutdown")
	this.mu.Unlock()
	return nil
}

type runtimeSchedulerRunnerBeam struct {
	Component
	IgnoreScanning
	scheduleErr error
	launchErr   error
	launchedRan atomic.Bool
}

func (this *runtimeSchedulerRunnerBeam) Run(wg *sync.WaitGroup) {
	wg.Go(func() {
		sched := container.GetContainer().Scheduler()
		this.scheduleErr = sched.ScheduleTask(func() {}, container.ScheduleType{}.New().FIXED_RATE, "10ms")
		done := make(chan struct{})
		this.launchErr = sched.LaunchTask(func() {
			this.launchedRan.Store(true)
			close(done)
		})
		<-done
	})
}

type wrapInPostConstructBeam struct {
	Component
	IgnoreScanning
	calls *atomic.Int32
}

func (this *wrapInPostConstructBeam) PostConstruct() Error {
	GetContainer().WrapLoggingHandler(func(next slog.Handler) slog.Handler {
		this.calls.Add(1)
		return next
	})
	return nil
}

type stuckRunnerBeam struct {
	Component
	IgnoreScanning
	started chan struct{}
	release chan struct{}
}

func (this *stuckRunnerBeam) Run(wg *sync.WaitGroup) {
	wg.Go(func() {
		close(this.started)
		<-this.release
	})
}
