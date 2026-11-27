package scheduler

import (
	"time"
)

// SetNowFunc overrides the clock function on s for deterministic unit tests.
func SetNowFunc(s *Scheduler, fn func() time.Time) {
	s.now = fn
}

// RunInPool exposes runInPool for deterministic unit tests.
func RunInPool(s *Scheduler, task func()) bool {
	return s.runInPool(task)
}

// Wait exposes wait for deterministic unit tests.
func Wait(s *Scheduler, d time.Duration) bool {
	return s.wait(d)
}
