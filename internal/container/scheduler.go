package container

import (
	"reflect"
	"strings"

	"github.com/BeamFoundry/rays/internal/boot"
	"github.com/BeamFoundry/rays/internal/services"
	. "github.com/BeamFoundry/rays/pkg/core"
	. "github.com/BeamFoundry/rays/pkg/lang"
)

type (
	ScheduleType        = boot.ScheduleType
	SchedulerProperties = boot.SchedulerProperties
	TaskScheduler       = boot.TaskScheduler
)

type scheduledSpec struct {
	param        string
	initialDelay string
	scheduleType boot.ScheduleType
	scheduled    bool
}

type pendingScheduledTask struct {
	beamType   reflect.Type
	task       func()
	methodName string
	spec       scheduledSpec
}

func (this *Container) registerScheduledTasks() Error {
	singleton := BeamScope{}.New().SINGLETON
	var tasks []pendingScheduledTask

	for _, beam := range this.GetAllBeams() {
		t := beam.Type()
		if this.IsChildBeam(beam) || beam.Scope() != singleton ||
			(t.Kind() != reflect.Struct && (t.Kind() != reflect.Pointer || t.Elem().Kind() != reflect.Struct)) ||
			FindFieldForInterface[Stereotype](t) == nil {
			continue
		}

		for mt := range t.Methods() {
			if mt.Type.NumIn() != 2 || mt.Type.NumOut() != 0 {
				continue
			}
			aType, aValue := this.GetMethodAnnotedType(&mt)
			if !aValue.IsValid() {
				continue
			}
			field := FindFieldWithAnnotation[Scheduled](aType)
			if field == nil {
				continue
			}

			spec, err := parseScheduledTag(field.Tag)
			if err != nil {
				return err
			}

			instance := this.GetBeamDefaultInstance(beam).Get()
			if err := this.InjectMethodAnnotedValue(aValue, instance); err != nil {
				return err
			}

			methodVal := reflect.ValueOf(instance).Method(mt.Index)
			args := []reflect.Value{aValue}
			tasks = append(tasks, pendingScheduledTask{
				beamType:   beam.Type(),
				methodName: mt.Name,
				spec:       spec,
				task: func() {
					methodVal.Call(args)
				},
			})
		}
	}

	sched := this.Scheduler()
	for _, entry := range tasks {
		var err Error
		if entry.spec.scheduled {
			services.Get().Logger.Debug("Registering scheduled task", "beam", entry.beamType, "method", entry.methodName, "type", entry.spec.scheduleType.String(), "param", entry.spec.param, "initialDelay", entry.spec.initialDelay)
			if entry.spec.initialDelay != "" {
				err = sched.ScheduleTaskWithInitialDelay(entry.task, entry.spec.scheduleType, entry.spec.param, entry.spec.initialDelay)
			} else {
				err = sched.ScheduleTask(entry.task, entry.spec.scheduleType, entry.spec.param)
			}
		} else {
			services.Get().Logger.Debug("Registering launch task", "beam", entry.beamType, "method", entry.methodName, "initialDelay", entry.spec.initialDelay)
			if entry.spec.initialDelay != "" {
				err = sched.LaunchTaskWithInitialDelay(entry.task, entry.spec.initialDelay)
			} else {
				err = sched.LaunchTask(entry.task)
			}
		}
		if err != nil {
			sched.Stop()
			return err
		}
	}

	return nil
}

func parseScheduledTag(tag reflect.StructTag) (scheduledSpec, Error) {
	var (
		spec    scheduledSpec
		matched int
	)

	remaining := strings.TrimSpace(string(tag))
	for remaining != "" {
		colonIdx := strings.IndexByte(remaining, ':')
		if colonIdx <= 0 || colonIdx+1 >= len(remaining) || remaining[colonIdx+1] != '"' {
			return scheduledSpec{}, NewError("InitializationError", "Invalid Scheduled struct tag syntax", "tag", string(tag))
		}
		key := remaining[:colonIdx]
		rest := remaining[colonIdx+2:]
		endQuote := strings.IndexByte(rest, '"')
		if endQuote < 0 {
			return scheduledSpec{}, NewError("InitializationError", "Invalid Scheduled struct tag syntax", "tag", string(tag))
		}
		val := rest[:endQuote]
		remaining = strings.TrimSpace(rest[endQuote+1:])

		switch key {
		case "cron":
			spec.scheduleType = boot.ScheduleTypes.CRON
			spec.param = val
			spec.scheduled = true
			matched++
		case "fixedRate":
			spec.scheduleType = boot.ScheduleTypes.FIXED_RATE
			spec.param = val
			spec.scheduled = true
			matched++
		case "fixedDelay":
			spec.scheduleType = boot.ScheduleTypes.FIXED_DELAY
			spec.param = val
			spec.scheduled = true
			matched++
		case "initialDelay":
			spec.initialDelay = val
		default:
			return scheduledSpec{}, NewError("InitializationError", "Unsupported struct tag on Scheduled annotation", "key", key)
		}
	}

	if matched > 1 {
		return scheduledSpec{}, NewError("InitializationError", "Scheduled annotation must specify at most one of cron, fixedRate, or fixedDelay")
	}

	return spec, nil
}
