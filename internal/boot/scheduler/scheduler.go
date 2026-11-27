// Package scheduler provides a fixed-pool concurrent task scheduler supporting
// one-shot, delayed, cron (with seconds), fixed-rate, and fixed-delay execution.
package scheduler

import (
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/BeamFoundry/rays/internal/boot/logger"
	"github.com/timandy/routine"
)

const (
	// DefaultPoolSize is the default maximum number of tasks that can run concurrently.
	DefaultPoolSize = 10

	errTaskNil = "task function cannot be nil"
)

// Scheduler manages concurrent task execution within a fixed-size worker pool,
// supporting one-shot, delayed, cron, fixed-rate, and fixed-delay tasks.
// Recurring tasks (CRON, FIXED_RATE, FIXED_DELAY) must be registered before
// [Scheduler.Start] or [Scheduler.StartAsync] is called; one-shot [Scheduler.LaunchTask]
// calls are queued before start and executed immediately once started.
type Scheduler struct {
	workerLocal routine.ThreadLocal[bool]
	sem         chan struct{}
	stopCh      chan struct{}
	now         func() time.Time
	logger      atomic.Pointer[logger.Logger]
	pending     []func()
	wg          sync.WaitGroup
	mu          sync.Mutex
	poolSize    int
	started     bool
	stopped     bool
	hasTasks    bool
}

// NewScheduler creates a new Scheduler with a fixed concurrency pool of poolSize.
// If poolSize is less than or equal to zero, [DefaultPoolSize] is used.
func NewScheduler(poolSize int) *Scheduler {
	if poolSize <= 0 {
		poolSize = DefaultPoolSize
	}
	return &Scheduler{
		workerLocal: routine.NewThreadLocal[bool](),
		sem:         make(chan struct{}, poolSize),
		stopCh:      make(chan struct{}),
		now:         time.Now,
		poolSize:    poolSize,
	}
}

// SetLogger sets the logger used to report panics recovered from scheduled tasks.
// If lg is nil, the scheduler falls back to [slog.Default].
func (this *Scheduler) SetLogger(lg *logger.Logger) {
	this.logger.Store(lg)
}

// PoolSize returns the fixed maximum number of tasks that can execute concurrently.
func (this *Scheduler) PoolSize() int {
	return this.poolSize
}

// HasTasks reports whether at least one task has been scheduled or launched on this scheduler.
func (this *Scheduler) HasTasks() bool {
	this.mu.Lock()
	defer this.mu.Unlock()
	return this.hasTasks
}

// InWorker reports whether the calling goroutine is currently executing a task in this scheduler.
func (this *Scheduler) InWorker() bool {
	return this.workerLocal.Get()
}

// StartAsync transitions the scheduler to the started state and launches all queued tasks
// without blocking. Returns false if the scheduler has already been stopped.
func (this *Scheduler) StartAsync() bool {
	this.mu.Lock()
	defer this.mu.Unlock()
	if this.stopped {
		return false
	}
	if !this.started {
		this.started = true
		pending := this.pending
		this.pending = nil
		for _, fn := range pending {
			this.wg.Add(1)
			go func(taskFn func()) {
				defer this.wg.Done()
				taskFn()
			}(fn)
		}
	}
	return true
}

// Wait blocks until [Scheduler.Stop] is called and all active tasks have finished.
func (this *Scheduler) Wait() {
	<-this.stopCh
	this.wg.Wait()
}

// Start launches all queued tasks and blocks until [Scheduler.Stop] is called
// and all active tasks have finished. Calling Start on an already started scheduler
// waits until Stop completes. Calling Start after Stop returns immediately.
func (this *Scheduler) Start() {
	if !this.StartAsync() {
		return
	}
	this.Wait()
}

// Stop signals all scheduled and pending tasks to stop and waits for active tasks to finish
// without starting any new tasks. Calling Stop multiple times is safe and idempotent.
func (this *Scheduler) Stop() {
	this.mu.Lock()
	if !this.stopped {
		this.stopped = true
		this.pending = nil
		close(this.stopCh)
	}
	this.mu.Unlock()
	this.wg.Wait()
}

// ScheduleCronTask schedules task according to a 6-field cron expression starting with seconds.
// An optional initialDelay ([time.ParseDuration] string) may be provided to wait before evaluating the schedule.
// Must be called before [Scheduler.Start] or [Scheduler.StartAsync].
func (this *Scheduler) ScheduleCronTask(task func(), expr string, initialDelay ...string) error {
	if task == nil {
		return errors.New(errTaskNil)
	}
	delay, err := parseOptionalInitialDelay(initialDelay...)
	if err != nil {
		return err
	}
	parsed, err := ParseCron(expr)
	if err != nil {
		return err
	}
	return this.enqueueScheduled(func() {
		this.runCron(task, parsed, delay)
	})
}

// ScheduleFixedRateTask schedules task to run repeatedly with rate ([time.ParseDuration] string)
// between two starts of the task. An optional initialDelay may be provided.
// Must be called before [Scheduler.Start] or [Scheduler.StartAsync].
func (this *Scheduler) ScheduleFixedRateTask(task func(), rate string, initialDelay ...string) error {
	if task == nil {
		return errors.New(errTaskNil)
	}
	delay, err := parseOptionalInitialDelay(initialDelay...)
	if err != nil {
		return err
	}
	interval, err := parsePositiveDuration(rate)
	if err != nil {
		return err
	}
	return this.enqueueScheduled(func() {
		this.runFixedRate(task, interval, delay)
	})
}

// ScheduleFixedDelayTask schedules task to run repeatedly with delay ([time.ParseDuration] string)
// between the end of one execution and the start of the next. An optional initialDelay may be provided.
// Must be called before [Scheduler.Start] or [Scheduler.StartAsync].
func (this *Scheduler) ScheduleFixedDelayTask(task func(), delay string, initialDelay ...string) error {
	if task == nil {
		return errors.New(errTaskNil)
	}
	initDelay, err := parseOptionalInitialDelay(initialDelay...)
	if err != nil {
		return err
	}
	waitDelay, err := parsePositiveDuration(delay)
	if err != nil {
		return err
	}
	return this.enqueueScheduled(func() {
		this.runFixedDelay(task, waitDelay, initDelay)
	})
}

// LaunchTask launches task once inside the scheduler's fixed worker pool.
// An optional initialDelay ([time.ParseDuration] string) may be provided to wait before launching the task.
func (this *Scheduler) LaunchTask(task func(), initialDelay ...string) error {
	if task == nil {
		return errors.New(errTaskNil)
	}
	delay, err := parseOptionalInitialDelay(initialDelay...)
	if err != nil {
		return err
	}
	return this.enqueueLaunch(func() {
		if !this.wait(delay) {
			return
		}
		this.runInPool(task)
	})
}

func (this *Scheduler) enqueueScheduled(fn func()) error {
	this.mu.Lock()
	defer this.mu.Unlock()
	if this.stopped {
		return errors.New("scheduler is stopped")
	}
	if this.started {
		return errors.New("scheduler is already started")
	}
	this.hasTasks = true
	this.pending = append(this.pending, fn)
	return nil
}

func (this *Scheduler) enqueueLaunch(fn func()) error {
	this.mu.Lock()
	if this.stopped {
		this.mu.Unlock()
		return errors.New("scheduler is stopped")
	}
	this.hasTasks = true
	if !this.started {
		this.pending = append(this.pending, fn)
		this.mu.Unlock()
		return nil
	}
	this.wg.Add(1)
	this.mu.Unlock()

	go func() {
		defer this.wg.Done()
		fn()
	}()
	return nil
}

func (this *Scheduler) runCron(task func(), expr *CronExpression, initialDelay time.Duration) {
	if !this.wait(initialDelay) {
		return
	}
	now := this.now()
	next := expr.Next(now)
	for {
		if !this.wait(next.Sub(now)) {
			return
		}
		if !this.runInPool(task) {
			return
		}
		now = this.now()
		after := now
		if next.After(after) {
			after = next
		}
		next = expr.Next(after)
	}
}

func (this *Scheduler) runFixedRate(task func(), interval, initialDelay time.Duration) {
	if !this.wait(initialDelay) {
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		if !this.runInPool(task) {
			return
		}
		select {
		case <-this.stopCh:
			return
		case <-ticker.C:
		}
	}
}

func (this *Scheduler) runFixedDelay(task func(), delay, initialDelay time.Duration) {
	if !this.wait(initialDelay) {
		return
	}
	for {
		if !this.runInPool(task) {
			return
		}
		if !this.wait(delay) {
			return
		}
	}
}

func (this *Scheduler) isStopped() bool {
	select {
	case <-this.stopCh:
		return true
	default:
		return false
	}
}

func (this *Scheduler) runInPool(task func()) bool {
	select {
	case <-this.stopCh:
		return false
	case this.sem <- struct{}{}:
	}
	defer func() { <-this.sem }()

	if this.isStopped() {
		return false
	}

	this.executeTask(task)
	return true
}

func (this *Scheduler) executeTask(task func()) {
	this.workerLocal.Set(true)
	defer this.workerLocal.Remove()
	defer func() {
		if r := recover(); r != nil {
			stack := string(debug.Stack())
			if lg := this.logger.Load(); lg != nil {
				lg.Error("Scheduled task panicked", "panic", r, "stack", stack)
				return
			}
			slog.Default().Error("Scheduled task panicked", "panic", r, "stack", stack)
		}
	}()
	task()
}

func (this *Scheduler) wait(d time.Duration) bool {
	if this.isStopped() {
		return false
	}
	if d <= 0 {
		return true
	}
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-this.stopCh:
		return false
	case <-timer.C:
		return !this.isStopped()
	}
}

func parseOptionalInitialDelay(initialDelay ...string) (time.Duration, error) {
	if len(initialDelay) == 0 || initialDelay[0] == "" {
		return 0, nil
	}
	d, err := time.ParseDuration(initialDelay[0])
	if err != nil || d < 0 {
		return 0, fmt.Errorf("invalid initial delay duration: %s", initialDelay[0])
	}
	return d, nil
}

func parsePositiveDuration(raw string) (time.Duration, error) {
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		return 0, fmt.Errorf("invalid positive duration: %s", raw)
	}
	return d, nil
}
