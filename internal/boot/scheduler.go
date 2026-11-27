package boot

import (
	"github.com/BeamFoundry/rays/internal/boot/scheduler"
	. "github.com/BeamFoundry/rays/pkg/core"
	. "github.com/BeamFoundry/rays/pkg/lang"
)

// ScheduleTypes is an initialized ScheduleTypeEnum variable.
var ScheduleTypes = ScheduleType{}.New()

// ScheduleType is an Enum alias for [ScheduleTypeEnum].
type ScheduleType = Enum[ScheduleTypeEnum]

// ScheduleTypeEnum defines the supported task scheduling modes.
type ScheduleTypeEnum struct {
	Enum[ScheduleTypeEnum]
	CRON,
	FIXED_RATE,
	FIXED_DELAY ScheduleType
}

// SchedulerProperties holds configuration properties under the "rays.scheduler" namespace.
type SchedulerProperties struct {
	ConfigurationProperties `Name:"rays.scheduler"`
	// Maximum number of tasks that can run concurrently in the scheduler pool.
	// Defaults to scheduler.DefaultPoolSize.
	pool int `validate:"gte=1"`
}

// Pool returns the configured maximum concurrent task pool size.
func (this *SchedulerProperties) Pool() int {
	return this.pool
}

// TaskScheduler proxies [scheduler.Scheduler] using the [ScheduleType] enum and Rays [Error].
type TaskScheduler struct {
	scheduler *scheduler.Scheduler
}

// NewTaskScheduler returns a new TaskScheduler proxying the provided scheduler instance.
func NewTaskScheduler(s *scheduler.Scheduler) *TaskScheduler {
	return &TaskScheduler{scheduler: s}
}

// PoolSize returns the fixed maximum number of tasks that can execute concurrently.
func (this *TaskScheduler) PoolSize() int {
	return this.scheduler.PoolSize()
}

// HasTasks reports whether at least one task has been scheduled or launched on this scheduler.
func (this *TaskScheduler) HasTasks() bool {
	return this.scheduler.HasTasks()
}

// Start launches all queued tasks and blocks until Stop is called and all active tasks finish.
func (this *TaskScheduler) Start() {
	this.scheduler.Start()
}

// Stop signals all scheduled and pending tasks to stop and waits for active tasks to finish.
func (this *TaskScheduler) Stop() {
	this.scheduler.Stop()
}

// ScheduleTask schedules task according to scheduleType and its parameter string:
//   - CRON: a 6-field cron expression starting with seconds.
//   - FIXED_RATE: a [time.ParseDuration] string expressing the delay between two starts of the task.
//   - FIXED_DELAY: a [time.ParseDuration] string expressing the delay between the end of a task and the start of the next one.
func (this *TaskScheduler) ScheduleTask(task func(), scheduleType ScheduleType, param string) Error {
	return this.scheduleTask(task, scheduleType, param)
}

// ScheduleTaskWithInitialDelay schedules task like [TaskScheduler.ScheduleTask], waiting for
// initialDelay (a [time.ParseDuration] string) before evaluating the task schedule.
func (this *TaskScheduler) ScheduleTaskWithInitialDelay(task func(), scheduleType ScheduleType, param, initialDelay string) Error {
	return this.scheduleTask(task, scheduleType, param, initialDelay)
}

// LaunchTask launches task once inside the scheduler's fixed worker pool.
func (this *TaskScheduler) LaunchTask(task func()) Error {
	return NewErrorFrom(this.scheduler.LaunchTask(task))
}

// LaunchTaskWithInitialDelay waits for initialDelay (a [time.ParseDuration] string)
// before launching task once inside the scheduler's fixed worker pool.
func (this *TaskScheduler) LaunchTaskWithInitialDelay(task func(), initialDelay string) Error {
	return NewErrorFrom(this.scheduler.LaunchTask(task, initialDelay))
}

func (this *TaskScheduler) scheduleTask(task func(), scheduleType ScheduleType, param string, initialDelay ...string) Error {
	switch scheduleType {
	case ScheduleTypes.CRON:
		return NewErrorFrom(this.scheduler.ScheduleCronTask(task, param, initialDelay...))
	case ScheduleTypes.FIXED_RATE:
		return NewErrorFrom(this.scheduler.ScheduleFixedRateTask(task, param, initialDelay...))
	case ScheduleTypes.FIXED_DELAY:
		return NewErrorFrom(this.scheduler.ScheduleFixedDelayTask(task, param, initialDelay...))
	default:
		return NewError("IllegalArgumentError", "Unsupported schedule type", "scheduleType", scheduleType.String())
	}
}
