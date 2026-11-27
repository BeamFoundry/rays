package scheduler_test

import (
	"bytes"
	"sync"
	"sync/atomic"
	"time"

	"github.com/BeamFoundry/rays/internal/boot/logger"
	"github.com/BeamFoundry/rays/internal/boot/scheduler"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func newStartedScheduler(poolSize int) (s *scheduler.Scheduler, stop func()) {
	sched := scheduler.NewScheduler(poolSize)
	Expect(sched.StartAsync()).To(BeTrue())
	done := make(chan struct{})
	go func() {
		sched.Wait()
		close(done)
	}()
	return sched, func() {
		sched.Stop()
		Eventually(done, time.Second).Should(BeClosed())
	}
}

func startScheduler(sched *scheduler.Scheduler) func() {
	done := make(chan struct{})
	go func() {
		sched.Start()
		close(done)
	}()
	return func() {
		sched.Stop()
		Eventually(done, time.Second).Should(BeClosed())
	}
}

var _ = Describe("Scheduler", func() {
	Context("construction, Start/Stop lifecycle, and pool configuration", func() {
		It("uses the provided positive pool size", func() {
			s := scheduler.NewScheduler(4)
			defer s.Stop()
			Expect(s.PoolSize()).To(Equal(4))
			Expect(s.HasTasks()).To(BeFalse())
			Expect(s.InWorker()).To(BeFalse())
		})

		It("falls back to DefaultPoolSize when poolSize is zero or negative", func() {
			sZero := scheduler.NewScheduler(0)
			defer sZero.Stop()
			Expect(sZero.PoolSize()).To(Equal(scheduler.DefaultPoolSize))

			sNeg := scheduler.NewScheduler(-5)
			defer sNeg.Stop()
			Expect(sNeg.PoolSize()).To(Equal(scheduler.DefaultPoolSize))
		})

		It("queues tasks until Start is called, blocks Start until Stop, reports InWorker, and rejects Schedule* after Start", func() {
			s := scheduler.NewScheduler(2)
			Expect(s.HasTasks()).To(BeFalse())

			var ran atomic.Bool
			var observedInWorker atomic.Bool
			Expect(s.LaunchTask(func() {
				observedInWorker.Store(s.InWorker())
				ran.Store(true)
			})).NotTo(HaveOccurred())
			Expect(s.HasTasks()).To(BeTrue())

			// Task must not execute before Start()
			Consistently(ran.Load, 30*time.Millisecond, 5*time.Millisecond).Should(BeFalse())

			startDone1 := make(chan struct{})
			startDone2 := make(chan struct{})
			go func() {
				s.Start()
				close(startDone1)
			}()
			Eventually(ran.Load, time.Second, 5*time.Millisecond).Should(BeTrue())
			Expect(observedInWorker.Load()).To(BeTrue())
			Expect(s.InWorker()).To(BeFalse())

			// Recurring Schedule* calls are rejected once started
			Expect(s.ScheduleCronTask(func() {}, "* * * * * *")).To(HaveOccurred())
			Expect(s.ScheduleFixedRateTask(func() {}, "10ms")).To(HaveOccurred())
			Expect(s.ScheduleFixedDelayTask(func() {}, "10ms")).To(HaveOccurred())

			// Calling Start a second time while already running also blocks until Stop()
			go func() {
				s.Start()
				close(startDone2)
			}()
			Consistently(startDone1, 20*time.Millisecond).ShouldNot(BeClosed())
			Consistently(startDone2, 20*time.Millisecond).ShouldNot(BeClosed())

			s.Stop()
			Eventually(startDone1, time.Second).Should(BeClosed())
			Eventually(startDone2, time.Second).Should(BeClosed())

			// Start / StartAsync after Stop return immediately
			Expect(s.StartAsync()).To(BeFalse())
			s.Start()
		})

		It("drops pending tasks if Stop is called before Start", func() {
			s := scheduler.NewScheduler(1)
			var ran atomic.Bool
			Expect(s.LaunchTask(func() {
				ran.Store(true)
			})).NotTo(HaveOccurred())

			s.Stop()
			s.Start()
			Expect(ran.Load()).To(BeFalse())
			for range 64 {
				Expect(scheduler.RunInPool(s, func() { ran.Store(true) })).To(BeFalse())
			}
			Expect(scheduler.Wait(s, 0)).To(BeFalse())
			Expect(ran.Load()).To(BeFalse())
		})

		It("enforces the fixed concurrency pool limit", func() {
			const poolSize = 2
			const totalTasks = 8

			s, cleanup := newStartedScheduler(poolSize)
			defer cleanup()

			var active atomic.Int32
			var maxObserved atomic.Int32
			var completed atomic.Int32

			for range totalTasks {
				err := s.LaunchTask(func() {
					cur := active.Add(1)
					for {
						prev := maxObserved.Load()
						if cur <= prev || maxObserved.CompareAndSwap(prev, cur) {
							break
						}
					}
					time.Sleep(20 * time.Millisecond)
					active.Add(-1)
					completed.Add(1)
				})
				Expect(err).NotTo(HaveOccurred())
			}

			Eventually(completed.Load, time.Second, 5*time.Millisecond).Should(Equal(int32(totalTasks)))
			Expect(maxObserved.Load()).To(BeNumerically("<=", int32(poolSize)))
			Expect(maxObserved.Load()).To(Equal(int32(poolSize)))
		})

		It("recovers from panics using both default slog and custom logger without leaking pool slots", func() {
			// Default slog path (no custom logger set)
			sDefault, cleanupDefault := newStartedScheduler(1)
			defer cleanupDefault()

			defaultStarted := make(chan struct{})
			Expect(sDefault.LaunchTask(func() {
				close(defaultStarted)
				panic("default boom")
			})).NotTo(HaveOccurred())
			Eventually(defaultStarted, time.Second).Should(BeClosed())

			doneDefault := make(chan struct{})
			Expect(sDefault.LaunchTask(func() {
				close(doneDefault)
			})).NotTo(HaveOccurred())
			Eventually(doneDefault, time.Second).Should(BeClosed())

			// Custom logger path via SetLogger
			sCustom, cleanupCustom := newStartedScheduler(1)

			var buf bytes.Buffer
			customLogger := logger.NewLogger()
			customLogger.SetOutput(&buf)
			sCustom.SetLogger(customLogger)

			customStarted := make(chan struct{})
			Expect(sCustom.LaunchTask(func() {
				close(customStarted)
				panic("custom boom")
			})).NotTo(HaveOccurred())
			Eventually(customStarted, time.Second).Should(BeClosed())

			doneCustom := make(chan struct{})
			Expect(sCustom.LaunchTask(func() {
				close(doneCustom)
			})).NotTo(HaveOccurred())
			Eventually(doneCustom, time.Second).Should(BeClosed())
			cleanupCustom()
			Expect(buf.String()).To(ContainSubstring("custom boom"))
		})
	})

	Context("LaunchTask with optional initialDelay", func() {
		It("rejects nil task functions and invalid or negative initial delays", func() {
			s := scheduler.NewScheduler(2)
			defer s.Stop()

			Expect(s.LaunchTask(nil)).To(HaveOccurred())
			Expect(s.LaunchTask(nil, "10ms")).To(HaveOccurred())
			Expect(s.LaunchTask(func() {}, "invalid")).To(HaveOccurred())
			Expect(s.LaunchTask(func() {}, "-10ms")).To(HaveOccurred())
		})

		It("treats empty string initialDelay as no delay and ignores extra variadic arguments", func() {
			s, cleanup := newStartedScheduler(2)
			defer cleanup()

			doneEmpty := make(chan struct{})
			Expect(s.LaunchTask(func() {
				close(doneEmpty)
			}, "")).NotTo(HaveOccurred())
			Eventually(doneEmpty, time.Second).Should(BeClosed())

			start := time.Now()
			executedAt := make(chan time.Time, 1)
			Expect(s.LaunchTask(func() {
				executedAt <- time.Now()
			}, "35ms", "ignored-second-arg")).NotTo(HaveOccurred())

			var ts time.Time
			Eventually(executedAt, time.Second).Should(Receive(&ts))
			Expect(ts.Sub(start)).To(BeNumerically(">=", 30*time.Millisecond))
		})

		It("aborts LaunchTask if stopped during initial delay", func() {
			s, cleanup := newStartedScheduler(2)

			var ran atomic.Bool
			Expect(s.LaunchTask(func() {
				ran.Store(true)
			}, "5s")).NotTo(HaveOccurred())

			cleanup()
			Expect(ran.Load()).To(BeFalse())
		})

		It("aborts queued LaunchTask if stopped while waiting for a pool slot", func() {
			s, cleanup := newStartedScheduler(1)

			started := make(chan struct{})
			release := make(chan struct{})
			Expect(s.LaunchTask(func() {
				close(started)
				<-release
			})).NotTo(HaveOccurred())
			Eventually(started, time.Second).Should(BeClosed())

			var secondRan atomic.Bool
			Expect(s.LaunchTask(func() {
				secondRan.Store(true)
			})).NotTo(HaveOccurred())

			time.Sleep(15 * time.Millisecond)

			stopDone := make(chan struct{})
			go func() {
				cleanup()
				close(stopDone)
			}()

			time.Sleep(15 * time.Millisecond)
			close(release)
			Eventually(stopDone, time.Second).Should(BeClosed())
			Expect(secondRan.Load()).To(BeFalse())
		})
	})

	Context("Scheduling validation and lifecycle", func() {
		It("rejects nil tasks, invalid expressions/durations, and invalid initial delays", func() {
			s := scheduler.NewScheduler(2)
			defer s.Stop()

			Expect(s.ScheduleCronTask(nil, "* * * * * *")).To(HaveOccurred())
			Expect(s.ScheduleCronTask(func() {}, "invalid")).To(HaveOccurred())
			Expect(s.ScheduleCronTask(func() {}, "* * * * * *", "bad")).To(HaveOccurred())
			Expect(s.ScheduleCronTask(func() {}, "* * * * * *", "-1ms")).To(HaveOccurred())

			Expect(s.ScheduleFixedRateTask(nil, "10ms")).To(HaveOccurred())
			Expect(s.ScheduleFixedRateTask(func() {}, "bad")).To(HaveOccurred())
			Expect(s.ScheduleFixedRateTask(func() {}, "0s")).To(HaveOccurred())
			Expect(s.ScheduleFixedRateTask(func() {}, "-5ms")).To(HaveOccurred())
			Expect(s.ScheduleFixedRateTask(func() {}, "10ms", "bad")).To(HaveOccurred())

			Expect(s.ScheduleFixedDelayTask(nil, "10ms")).To(HaveOccurred())
			Expect(s.ScheduleFixedDelayTask(func() {}, "bad")).To(HaveOccurred())
			Expect(s.ScheduleFixedDelayTask(func() {}, "0s")).To(HaveOccurred())
			Expect(s.ScheduleFixedDelayTask(func() {}, "10ms", "bad")).To(HaveOccurred())
		})

		It("rejects tasks after the scheduler has been stopped and allows idempotent Stop calls", func() {
			s := scheduler.NewScheduler(2)
			s.Stop()
			s.Stop()

			Expect(s.LaunchTask(func() {})).To(HaveOccurred())
			Expect(s.ScheduleCronTask(func() {}, "* * * * * *")).To(HaveOccurred())
			Expect(s.ScheduleFixedRateTask(func() {}, "10ms")).To(HaveOccurred())
			Expect(s.ScheduleFixedDelayTask(func() {}, "10ms")).To(HaveOccurred())
		})
	})

	Context("ScheduleFixedRateTask", func() {
		It("runs tasks repeatedly at a fixed rate between starts", func() {
			s := scheduler.NewScheduler(2)

			var mu sync.Mutex
			var starts []time.Time

			Expect(s.ScheduleFixedRateTask(func() {
				mu.Lock()
				starts = append(starts, time.Now())
				mu.Unlock()
				time.Sleep(15 * time.Millisecond)
			}, "40ms")).NotTo(HaveOccurred())

			cleanup := startScheduler(s)
			defer cleanup()

			Eventually(func() int {
				mu.Lock()
				defer mu.Unlock()
				return len(starts)
			}, time.Second, 5*time.Millisecond).Should(BeNumerically(">=", 3))

			cleanup()

			mu.Lock()
			firstGap := starts[1].Sub(starts[0])
			secondGap := starts[2].Sub(starts[1])
			mu.Unlock()

			Expect(firstGap).To(BeNumerically(">=", 30*time.Millisecond))
			Expect(secondGap).To(BeNumerically(">=", 30*time.Millisecond))
		})

		It("honors initial delay and stops cleanly if stopped during initial delay or while waiting for pool", func() {
			s := scheduler.NewScheduler(2)
			firstRun := make(chan time.Time, 1)

			Expect(s.ScheduleFixedRateTask(func() {
				select {
				case firstRun <- time.Now():
				default:
				}
			}, "30ms", "40ms")).NotTo(HaveOccurred())

			start := time.Now()
			cleanup := startScheduler(s)

			var ts time.Time
			Eventually(firstRun, time.Second).Should(Receive(&ts))
			Expect(ts.Sub(start)).To(BeNumerically(">=", 35*time.Millisecond))
			cleanup()

			// Stopped during initial delay
			s2 := scheduler.NewScheduler(1)
			var ran atomic.Bool
			Expect(s2.ScheduleFixedRateTask(func() {
				ran.Store(true)
			}, "10ms", "5s")).NotTo(HaveOccurred())
			cleanup2 := startScheduler(s2)
			time.Sleep(15 * time.Millisecond)
			cleanup2()
			Expect(ran.Load()).To(BeFalse())

			// Stopped while blocked on pool slot
			s3 := scheduler.NewScheduler(1)
			started := make(chan struct{})
			release := make(chan struct{})
			Expect(s3.LaunchTask(func() {
				close(started)
				<-release
			})).NotTo(HaveOccurred())

			var rateRan atomic.Bool
			Expect(s3.ScheduleFixedRateTask(func() {
				rateRan.Store(true)
			}, "10ms", "15ms")).NotTo(HaveOccurred())

			cleanup3 := startScheduler(s3)
			Eventually(started, time.Second).Should(BeClosed())
			time.Sleep(30 * time.Millisecond)

			stopDone := make(chan struct{})
			go func() {
				cleanup3()
				close(stopDone)
			}()
			time.Sleep(15 * time.Millisecond)
			close(release)
			Eventually(stopDone, time.Second).Should(BeClosed())
			Expect(rateRan.Load()).To(BeFalse())
		})
	})

	Context("ScheduleFixedDelayTask", func() {
		It("waits the fixed delay after each task finishes before starting the next", func() {
			s := scheduler.NewScheduler(2)

			type runSpan struct {
				start time.Time
				end   time.Time
			}
			var mu sync.Mutex
			var spans []runSpan

			Expect(s.ScheduleFixedDelayTask(func() {
				st := time.Now()
				time.Sleep(25 * time.Millisecond)
				en := time.Now()
				mu.Lock()
				spans = append(spans, runSpan{start: st, end: en})
				mu.Unlock()
			}, "35ms")).NotTo(HaveOccurred())

			cleanup := startScheduler(s)
			defer cleanup()

			Eventually(func() int {
				mu.Lock()
				defer mu.Unlock()
				return len(spans)
			}, time.Second, 5*time.Millisecond).Should(BeNumerically(">=", 3))

			cleanup()

			mu.Lock()
			gap1 := spans[1].start.Sub(spans[0].end)
			gap2 := spans[2].start.Sub(spans[1].end)
			mu.Unlock()

			Expect(gap1).To(BeNumerically(">=", 30*time.Millisecond))
			Expect(gap2).To(BeNumerically(">=", 30*time.Millisecond))
		})

		It("honors initial delay and stops cleanly during initial delay or pool wait", func() {
			s := scheduler.NewScheduler(2)
			firstRun := make(chan time.Time, 1)

			Expect(s.ScheduleFixedDelayTask(func() {
				select {
				case firstRun <- time.Now():
				default:
				}
			}, "30ms", "40ms")).NotTo(HaveOccurred())

			start := time.Now()
			cleanup := startScheduler(s)

			var ts time.Time
			Eventually(firstRun, time.Second).Should(Receive(&ts))
			Expect(ts.Sub(start)).To(BeNumerically(">=", 35*time.Millisecond))
			cleanup()

			// Stopped during initial delay
			s2 := scheduler.NewScheduler(1)
			var ran atomic.Bool
			Expect(s2.ScheduleFixedDelayTask(func() {
				ran.Store(true)
			}, "10ms", "5s")).NotTo(HaveOccurred())
			cleanup2 := startScheduler(s2)
			time.Sleep(15 * time.Millisecond)
			cleanup2()
			Expect(ran.Load()).To(BeFalse())

			// Stopped while blocked on pool slot
			s3 := scheduler.NewScheduler(1)
			started := make(chan struct{})
			release := make(chan struct{})
			Expect(s3.LaunchTask(func() {
				close(started)
				<-release
			})).NotTo(HaveOccurred())

			var delayRan atomic.Bool
			Expect(s3.ScheduleFixedDelayTask(func() {
				delayRan.Store(true)
			}, "10ms", "15ms")).NotTo(HaveOccurred())

			cleanup3 := startScheduler(s3)
			Eventually(started, time.Second).Should(BeClosed())
			time.Sleep(30 * time.Millisecond)

			stopDone := make(chan struct{})
			go func() {
				cleanup3()
				close(stopDone)
			}()
			time.Sleep(15 * time.Millisecond)
			close(release)
			Eventually(stopDone, time.Second).Should(BeClosed())
			Expect(delayRan.Load()).To(BeFalse())
		})
	})

	Context("ScheduleCronTask", func() {
		It("executes tasks according to a 6-field cron schedule with seconds", func() {
			s := scheduler.NewScheduler(2)

			base := time.Date(2026, time.March, 15, 10, 0, 0, 985000000, time.UTC)
			var callCount atomic.Int32
			scheduler.SetNowFunc(s, func() time.Time {
				switch callCount.Add(1) {
				case 1:
					return base
				case 2:
					return base.Add(10 * time.Millisecond)
				default:
					return time.Date(2026, time.March, 15, 10, 0, 2, 0, time.UTC)
				}
			})

			var runs atomic.Int32
			Expect(s.ScheduleCronTask(func() {
				runs.Add(1)
			}, "* * * * * *")).NotTo(HaveOccurred())

			cleanup := startScheduler(s)
			defer cleanup()

			Eventually(runs.Load, 2*time.Second, 5*time.Millisecond).Should(BeNumerically(">=", 2))
			cleanup()
		})

		It("honors initial delay and stops cleanly during initial delay, cron wait, or pool wait", func() {
			// 1. Stopped during initial delay
			s1 := scheduler.NewScheduler(1)
			var ran1 atomic.Bool
			Expect(s1.ScheduleCronTask(func() {
				ran1.Store(true)
			}, "* * * * * *", "5s")).NotTo(HaveOccurred())
			cleanup1 := startScheduler(s1)
			time.Sleep(15 * time.Millisecond)
			cleanup1()
			Expect(ran1.Load()).To(BeFalse())

			// 2. Stopped during cron wait after initial delay
			s2 := scheduler.NewScheduler(1)
			var ran2 atomic.Bool
			Expect(s2.ScheduleCronTask(func() {
				ran2.Store(true)
			}, "0 0 0 1 1 *", "10ms")).NotTo(HaveOccurred())
			cleanup2 := startScheduler(s2)
			time.Sleep(25 * time.Millisecond)
			cleanup2()
			Expect(ran2.Load()).To(BeFalse())

			// 3. Stopped while blocked waiting for a pool slot on a cron trigger
			s3 := scheduler.NewScheduler(1)
			scheduler.SetNowFunc(s3, func() time.Time {
				return time.Date(2026, time.March, 15, 10, 0, 0, 990000000, time.UTC)
			})
			started := make(chan struct{})
			release := make(chan struct{})
			Expect(s3.LaunchTask(func() {
				close(started)
				<-release
			})).NotTo(HaveOccurred())

			var cronRan atomic.Bool
			Expect(s3.ScheduleCronTask(func() {
				cronRan.Store(true)
			}, "* * * * * *", "10ms")).NotTo(HaveOccurred())

			cleanup3 := startScheduler(s3)
			Eventually(started, time.Second).Should(BeClosed())
			time.Sleep(35 * time.Millisecond)

			stopDone := make(chan struct{})
			go func() {
				cleanup3()
				close(stopDone)
			}()
			time.Sleep(15 * time.Millisecond)
			close(release)
			Eventually(stopDone, time.Second).Should(BeClosed())
			Expect(cronRan.Load()).To(BeFalse())
		})
	})
})
