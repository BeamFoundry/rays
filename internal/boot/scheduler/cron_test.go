package scheduler_test

import (
	"fmt"
	"math"
	"time"

	"github.com/BeamFoundry/rays/internal/boot/scheduler"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CronExpression", func() {
	Context("parsing validation", func() {
		It("rejects expressions without exactly 6 fields", func() {
			_, err := scheduler.ParseCron("* * * * *")
			Expect(err).To(HaveOccurred())

			_, err = scheduler.ParseCron("* * * * * * *")
			Expect(err).To(HaveOccurred())

			_, err = scheduler.ParseCron("")
			Expect(err).To(HaveOccurred())
		})

		It("rejects invalid values in each of the 6 fields", func() {
			invalidExprs := []string{
				"60 * * * * *",    // second out of range
				"-1 * * * * *",    // negative second
				"abc * * * * *",   // non-numeric second
				"0 60 * * * *",    // minute out of range
				"0 0 24 * * *",    // hour out of range
				"0 0 0 0 * *",     // day-of-month below 1
				"0 0 0 32 * *",    // day-of-month above 31
				"0 0 0 * 0 *",     // month below 1
				"0 0 0 * 13 *",    // month above 12
				"0 0 0 * * 8",     // day-of-week above 7
				"1,,2 * * * * *",  // empty list segment
				"*/0 * * * * *",   // zero step
				"*/-2 * * * * *",  // negative step
				"*/abc * * * * *", // non-numeric step
				"1/2/3 * * * * *", // multiple slashes
				"20-10 * * * * *", // inverted range
				"abc-5 * * * * *", // invalid range start
				"1-abc * * * * *", // invalid range end
				"? * * * * *",     // question mark disallowed in seconds
				"0 0 0 ?/2 * *",   // question mark with step
				"0 0 0 31 2 *",    // impossible calendar date (Feb 31)
				"0 0 0 31 4,6 *",  // impossible calendar date (Apr/Jun 31)
				"0 0 0 * * 7-3",   // inverted Sunday-7 range
			}

			for _, expr := range invalidExprs {
				parsed, err := scheduler.ParseCron(expr)
				Expect(err).To(HaveOccurred(), "expected error for expr %q", expr)
				Expect(parsed).To(BeNil())
			}
		})
	})

	Context("next execution calculation", func() {
		It("computes next second for wildcard expressions", func() {
			expr, err := scheduler.ParseCron("* * * * * *")
			Expect(err).NotTo(HaveOccurred())

			from := time.Date(2026, time.March, 15, 10, 20, 30, 500000000, time.UTC)
			Expect(expr.Next(from)).To(Equal(time.Date(2026, time.March, 15, 10, 20, 31, 0, time.UTC)))
		})

		It("handles seconds steps, ranges, and lists", func() {
			expr, err := scheduler.ParseCron("*/15 * * * * *")
			Expect(err).NotTo(HaveOccurred())

			from := time.Date(2026, time.March, 15, 10, 20, 30, 0, time.UTC)
			Expect(expr.Next(from)).To(Equal(time.Date(2026, time.March, 15, 10, 20, 45, 0, time.UTC)))
			Expect(expr.Next(time.Date(2026, time.March, 15, 10, 20, 45, 0, time.UTC))).
				To(Equal(time.Date(2026, time.March, 15, 10, 21, 0, 0, time.UTC)))

			exprRange, err := scheduler.ParseCron("5,10-20/5,45/10 * * * * *")
			Expect(err).NotTo(HaveOccurred())
			Expect(exprRange.Next(time.Date(2026, time.March, 15, 10, 0, 0, 0, time.UTC))).
				To(Equal(time.Date(2026, time.March, 15, 10, 0, 5, 0, time.UTC)))
			Expect(exprRange.Next(time.Date(2026, time.March, 15, 10, 0, 5, 0, time.UTC))).
				To(Equal(time.Date(2026, time.March, 15, 10, 0, 10, 0, time.UTC)))
			Expect(exprRange.Next(time.Date(2026, time.March, 15, 10, 0, 10, 0, time.UTC))).
				To(Equal(time.Date(2026, time.March, 15, 10, 0, 15, 0, time.UTC)))
			Expect(exprRange.Next(time.Date(2026, time.March, 15, 10, 0, 20, 0, time.UTC))).
				To(Equal(time.Date(2026, time.March, 15, 10, 0, 45, 0, time.UTC)))
			Expect(exprRange.Next(time.Date(2026, time.March, 15, 10, 0, 45, 0, time.UTC))).
				To(Equal(time.Date(2026, time.March, 15, 10, 0, 55, 0, time.UTC)))
		})

		It("rolls over across minutes, hours, days, months, and years", func() {
			expr, err := scheduler.ParseCron("30 15 14 20 6 *")
			Expect(err).NotTo(HaveOccurred())

			from := time.Date(2026, time.June, 20, 14, 15, 30, 0, time.UTC)
			Expect(expr.Next(from)).To(Equal(time.Date(2027, time.June, 20, 14, 15, 30, 0, time.UTC)))

			fromSameDay := time.Date(2026, time.June, 20, 10, 0, 0, 0, time.UTC)
			Expect(expr.Next(fromSameDay)).To(Equal(time.Date(2026, time.June, 20, 14, 15, 30, 0, time.UTC)))
		})

		It("handles leap year February 29", func() {
			expr, err := scheduler.ParseCron("0 0 12 29 FEB *")
			Expect(err).NotTo(HaveOccurred())

			from := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
			Expect(expr.Next(from)).To(Equal(time.Date(2028, time.February, 29, 12, 0, 0, 0, time.UTC)))
		})

		It("supports named months, named days of week, question mark, and Sunday as 7", func() {
			// Named month + named weekday + ?
			expr, err := scheduler.ParseCron("0 30 9 ? MAR MON-FRI")
			Expect(err).NotTo(HaveOccurred())
			// 2026-03-14 is Saturday -> next weekday is Monday 2026-03-16
			from := time.Date(2026, time.March, 14, 10, 0, 0, 0, time.UTC)
			Expect(expr.Next(from)).To(Equal(time.Date(2026, time.March, 16, 9, 30, 0, 0, time.UTC)))

			// Sunday as 7 (standalone, range, and stepped)
			exprSun, err := scheduler.ParseCron("0 0 8 * * 7")
			Expect(err).NotTo(HaveOccurred())
			// 2026-03-14 is Saturday -> next Sunday is 2026-03-15
			Expect(exprSun.Next(from)).To(Equal(time.Date(2026, time.March, 15, 8, 0, 0, 0, time.UTC)))

			exprFriSun, err := scheduler.ParseCron("0 0 8 * * 5-7")
			Expect(err).NotTo(HaveOccurred())
			Expect(exprFriSun.Next(time.Date(2026, time.March, 12, 9, 0, 0, 0, time.UTC))).
				To(Equal(time.Date(2026, time.March, 13, 8, 0, 0, 0, time.UTC))) // Friday
			Expect(exprFriSun.Next(time.Date(2026, time.March, 14, 9, 0, 0, 0, time.UTC))).
				To(Equal(time.Date(2026, time.March, 15, 8, 0, 0, 0, time.UTC))) // Sunday

			exprSunStep, err := scheduler.ParseCron("0 0 8 * * 7/2")
			Expect(err).NotTo(HaveOccurred())
			// 7/2 => Sunday(0), Tuesday(2), Thursday(4), Saturday(6)
			Expect(exprSunStep.Next(time.Date(2026, time.March, 15, 9, 0, 0, 0, time.UTC))).
				To(Equal(time.Date(2026, time.March, 17, 8, 0, 0, 0, time.UTC))) // Tuesday
		})

		It("uses union semantics when both day-of-month and day-of-week are restricted", func() {
			// Match either the 15th of the month OR any Monday
			expr, err := scheduler.ParseCron("0 0 12 15 * MON")
			Expect(err).NotTo(HaveOccurred())

			// 2026-03-14 is Saturday; 2026-03-15 is Sunday (matches DOM=15), 2026-03-16 is Monday (matches DOW=MON)
			sat := time.Date(2026, time.March, 14, 12, 0, 0, 0, time.UTC)
			sun := expr.Next(sat)
			Expect(sun).To(Equal(time.Date(2026, time.March, 15, 12, 0, 0, 0, time.UTC)))

			mon := expr.Next(sun)
			Expect(mon).To(Equal(time.Date(2026, time.March, 16, 12, 0, 0, 0, time.UTC)))
		})

		It("handles large steps without integer overflow", func() {
			expr, err := scheduler.ParseCron(fmt.Sprintf("1/%d * * * * *", math.MaxInt))
			Expect(err).NotTo(HaveOccurred())

			from := time.Date(2026, time.March, 15, 10, 0, 0, 0, time.UTC)
			Expect(expr.Next(from)).To(Equal(time.Date(2026, time.March, 15, 10, 0, 1, 0, time.UTC)))
			Expect(expr.Next(time.Date(2026, time.March, 15, 10, 0, 1, 0, time.UTC))).
				To(Equal(time.Date(2026, time.March, 15, 10, 1, 1, 0, time.UTC)))
		})

		It("advances past nonexistent hours on America/New_York DST spring-forward date", func() {
			nyLoc, err := time.LoadLocation("America/New_York")
			Expect(err).NotTo(HaveOccurred())

			// 2026-03-08 is the US spring-forward date (01:59:59 EST -> 03:00:00 EDT).
			from := time.Date(2026, time.March, 8, 1, 30, 0, 0, nyLoc)

			exprAfterGap, parseErr := scheduler.ParseCron("0 0 3 8 3 *")
			Expect(parseErr).NotTo(HaveOccurred())
			Expect(exprAfterGap.Next(from)).To(Equal(time.Date(2026, time.March, 8, 3, 0, 0, 0, nyLoc)))

			exprInGap, parseErr := scheduler.ParseCron("0 15 2 8 3 *")
			Expect(parseErr).NotTo(HaveOccurred())
			Expect(exprInGap.Next(from)).To(Equal(time.Date(2027, time.March, 8, 2, 15, 0, 0, nyLoc)))
		})

		It("returns both matching occurrences during an America/New_York DST fall-back repeated hour", func() {
			nyLoc, err := time.LoadLocation("America/New_York")
			Expect(err).NotTo(HaveOccurred())

			// 2026-11-01 is the US fall-back date (01:59:59 EDT -0400 -> 01:00:00 EST -0500).
			expr, parseErr := scheduler.ParseCron("0 30 1 1 11 *")
			Expect(parseErr).NotTo(HaveOccurred())

			beforeFirst := time.Date(2026, time.November, 1, 1, 0, 0, 0, nyLoc)
			first := expr.Next(beforeFirst)
			Expect(first.Format(time.RFC3339)).To(Equal("2026-11-01T01:30:00-04:00"))

			second := expr.Next(first)
			Expect(second.Format(time.RFC3339)).To(Equal("2026-11-01T01:30:00-05:00"))
			Expect(second.After(first)).To(BeTrue())

			// Starting inside the second repeated hour must advance forward, never jump back to EDT.
			inSecondHour := second.Add(-10 * time.Minute)
			Expect(expr.Next(inSecondHour)).To(Equal(second))
		})
	})
})
