package scheduler

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	cronFieldCount = 6

	cronSecondMin = 0
	cronSecondMax = 59
	cronMinuteMin = 0
	cronMinuteMax = 59
	cronHourMin   = 0
	cronHourMax   = 23
	cronDOMMin    = 1
	cronDOMMax    = 31
	cronMonthMin  = 1
	cronMonthMax  = 12
	cronDOWMin    = 0
	cronDOWMax    = 6
	cronDOWSunday = 7

	cronWildcard = "*"
	cronQuestion = "?"
)

var (
	cronMonthNames = map[string]int{
		"JAN": 1, "FEB": 2, "MAR": 3, "APR": 4,
		"MAY": 5, "JUN": 6, "JUL": 7, "AUG": 8,
		"SEP": 9, "OCT": 10, "NOV": 11, "DEC": 12,
	}

	cronDOWNames = map[string]int{
		"SUN": 0, "MON": 1, "TUE": 2, "WED": 3,
		"THU": 4, "FRI": 5, "SAT": 6,
	}

	cronMaxDaysInMonth = [13]int{0, 31, 29, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}

	cronSecondSpec = cronFieldSpec{minVal: cronSecondMin, maxVal: cronSecondMax}
	cronMinuteSpec = cronFieldSpec{minVal: cronMinuteMin, maxVal: cronMinuteMax}
	cronHourSpec   = cronFieldSpec{minVal: cronHourMin, maxVal: cronHourMax}
	cronDOMSpec    = cronFieldSpec{minVal: cronDOMMin, maxVal: cronDOMMax, allowQuestion: true}
	cronMonthSpec  = cronFieldSpec{names: cronMonthNames, minVal: cronMonthMin, maxVal: cronMonthMax}
	cronDOWSpec    = cronFieldSpec{names: cronDOWNames, minVal: cronDOWMin, maxVal: cronDOWMax, allowQuestion: true, allowSevenAsZero: true}
)

type cronFieldSpec struct {
	names            map[string]int
	minVal           int
	maxVal           int
	allowQuestion    bool
	allowSevenAsZero bool
}

// CronExpression represents a parsed 6-field cron schedule with a leading seconds field:
// `<second> <minute> <hour> <day-of-month> <month> <day-of-week>`.
type CronExpression struct {
	seconds     uint64
	minutes     uint64
	hours       uint64
	daysOfMonth uint64
	months      uint64
	daysOfWeek  uint64
	domWildcard bool
	dowWildcard bool
}

// ParseCron parses a 6-field cron expression (seconds, minutes, hours, day-of-month, month, day-of-week).
func ParseCron(expr string) (*CronExpression, error) {
	fields := strings.Fields(expr)
	if len(fields) != cronFieldCount {
		return nil, errors.New("cron expression must have 6 fields (seconds minutes hours day-of-month month day-of-week)")
	}

	seconds, _, err := parseCronField(fields[0], cronSecondSpec)
	if err != nil {
		return nil, err
	}
	minutes, _, err := parseCronField(fields[1], cronMinuteSpec)
	if err != nil {
		return nil, err
	}
	hours, _, err := parseCronField(fields[2], cronHourSpec)
	if err != nil {
		return nil, err
	}
	dom, domWildcard, err := parseCronField(fields[3], cronDOMSpec)
	if err != nil {
		return nil, err
	}
	months, _, err := parseCronField(fields[4], cronMonthSpec)
	if err != nil {
		return nil, err
	}
	dow, dowWildcard, err := parseCronField(fields[5], cronDOWSpec)
	if err != nil {
		return nil, err
	}

	parsed := &CronExpression{
		seconds:     seconds,
		minutes:     minutes,
		hours:       hours,
		daysOfMonth: dom,
		months:      months,
		daysOfWeek:  dow,
		domWildcard: domWildcard,
		dowWildcard: dowWildcard,
	}

	if !parsed.hasReachableDate() {
		return nil, errors.New("cron expression matches no valid calendar date")
	}

	return parsed, nil
}

// Next returns the earliest time strictly after from that satisfies the cron expression.
func (this *CronExpression) Next(from time.Time) time.Time {
	t := from.Truncate(time.Second).Add(time.Second)
	for {
		if this.months&(uint64(1)<<t.Month()) == 0 {
			t = time.Date(t.Year(), t.Month()+1, 1, 0, 0, 0, 0, t.Location())
			continue
		}
		if !this.matchesDay(t) {
			t = time.Date(t.Year(), t.Month(), t.Day()+1, 0, 0, 0, 0, t.Location())
			continue
		}
		if this.hours&(uint64(1)<<t.Hour()) == 0 {
			t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour()+1, 0, 0, 0, t.Location())
			continue
		}
		if this.minutes&(uint64(1)<<t.Minute()) == 0 {
			t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute()+1, 0, 0, t.Location())
			continue
		}
		if this.seconds&(uint64(1)<<t.Second()) == 0 {
			t = t.Add(time.Second)
			continue
		}
		return t
	}
}

func (this *CronExpression) matchesDay(t time.Time) bool {
	domMatch := this.daysOfMonth&(uint64(1)<<t.Day()) != 0
	dowMatch := this.daysOfWeek&(uint64(1)<<t.Weekday()) != 0
	if !this.domWildcard && !this.dowWildcard {
		return domMatch || dowMatch
	}
	return domMatch && dowMatch
}

func (this *CronExpression) hasReachableDate() bool {
	if this.domWildcard || !this.dowWildcard {
		return true
	}
	for m := cronMonthMin; m <= cronMonthMax; m++ {
		if this.months&(uint64(1)<<m) == 0 {
			continue
		}
		maxDay := cronMaxDaysInMonth[m]
		for d := cronDOMMin; d <= maxDay; d++ {
			if this.daysOfMonth&(uint64(1)<<d) != 0 {
				return true
			}
		}
	}
	return false
}

func parseCronField(field string, spec cronFieldSpec) (mask uint64, wildcard bool, err error) {
	if field == cronWildcard || (spec.allowQuestion && field == cronQuestion) {
		return bitRange(spec.minVal, spec.maxVal, 1), true, nil
	}

	for part := range strings.SplitSeq(field, ",") {
		partMask, partErr := parseCronPart(part, spec)
		if partErr != nil {
			return 0, false, partErr
		}
		mask |= partMask
	}

	if spec.allowSevenAsZero && mask&(uint64(1)<<cronDOWSunday) != 0 {
		mask = (mask &^ (uint64(1) << cronDOWSunday)) | 1
	}

	return mask, false, nil
}

func parseCronPart(part string, spec cronFieldSpec) (uint64, error) {
	if part == "" {
		return 0, errors.New("empty cron field segment")
	}

	step := 1
	base, stepStr, hasStep := strings.Cut(part, "/")
	if hasStep {
		parsedStep, err := strconv.Atoi(stepStr)
		if err != nil || parsedStep <= 0 {
			return 0, fmt.Errorf("invalid cron step value: %s", stepStr)
		}
		step = parsedStep
	}

	if base == cronWildcard {
		return bitRange(spec.minVal, spec.maxVal, step), nil
	}

	if startStr, endStr, isRange := strings.Cut(base, "-"); isRange {
		start, err := parseCronAtom(startStr, spec)
		if err != nil {
			return 0, err
		}
		end, err := parseCronAtom(endStr, spec)
		if err != nil {
			return 0, err
		}
		if start > end {
			return 0, fmt.Errorf("cron range start exceeds end: %s", base)
		}
		return bitRange(start, end, step), nil
	}

	val, err := parseCronAtom(base, spec)
	if err != nil {
		return 0, err
	}
	if hasStep {
		if spec.allowSevenAsZero && val == cronDOWSunday {
			val = cronDOWMin
		}
		return bitRange(val, spec.maxVal, step), nil
	}
	return uint64(1) << val, nil
}

func parseCronAtom(raw string, spec cronFieldSpec) (int, error) {
	if spec.names != nil {
		if v, ok := spec.names[strings.ToUpper(raw)]; ok {
			return v, nil
		}
	}
	val, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid cron value: %s", raw)
	}
	maxAllowed := spec.maxVal
	if spec.allowSevenAsZero {
		maxAllowed = cronDOWSunday
	}
	if val < spec.minVal || val > maxAllowed {
		return 0, fmt.Errorf("cron value out of range: %s", raw)
	}
	return val, nil
}

func bitRange(start, end, step int) uint64 {
	var mask uint64
	for v := start; v <= end; v += step {
		mask |= uint64(1) << v
	}
	return mask
}
