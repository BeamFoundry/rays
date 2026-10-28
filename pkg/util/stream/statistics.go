package stream

import (
	"fmt"
	"math"
)

// IntSummaryStatistics is a state object for collecting statistics such as count, min, max, sum, and average.
type IntSummaryStatistics struct {
	Count int64
	Sum   int64
	Min   int
	Max   int
}

// NewIntSummaryStatistics constructs an empty instance with zero count, zero sum,
// math.MaxInt min, and math.MinInt max.
func NewIntSummaryStatistics() *IntSummaryStatistics {
	return &IntSummaryStatistics{
		Count: 0,
		Sum:   0,
		Min:   math.MaxInt,
		Max:   math.MinInt,
	}
}

// Accept records a new int value into the summary information.
func (this *IntSummaryStatistics) Accept(value int) {
	if this.Count == 0 {
		this.Min = value
		this.Max = value
	}
	this.Count++
	this.Sum += int64(value)
	if value < this.Min {
		this.Min = value
	}
	if value > this.Max {
		this.Max = value
	}
}

// Combine combines the state of another IntSummaryStatistics into this one.
func (this *IntSummaryStatistics) Combine(other *IntSummaryStatistics) {
	if other == nil || other.Count == 0 {
		return
	}
	if this.Count == 0 {
		this.Count = other.Count
		this.Sum = other.Sum
		this.Min = other.Min
		this.Max = other.Max
		return
	}
	this.Count += other.Count
	this.Sum += other.Sum
	if other.Min < this.Min {
		this.Min = other.Min
	}
	if other.Max > this.Max {
		this.Max = other.Max
	}
}

// Average returns the arithmetic mean of values recorded, or zero if no values have been recorded.
func (this *IntSummaryStatistics) Average() float64 {
	if this.Count == 0 {
		return 0
	}
	return float64(this.Sum) / float64(this.Count)
}

// String returns a string representation of the statistics.
func (this *IntSummaryStatistics) String() string {
	return fmt.Sprintf("IntSummaryStatistics{count=%d, sum=%d, min=%d, average=%f, max=%d}",
		this.Count, this.Sum, this.Min, this.Average(), this.Max)
}

// Float64SummaryStatistics is a state object for collecting statistics such as count, min, max, sum, and average.
type Float64SummaryStatistics struct {
	Count int64
	Sum   float64
	Min   float64
	Max   float64
}

// NewFloat64SummaryStatistics constructs an empty instance with zero count, zero sum,
// math.MaxFloat64 min, and -math.MaxFloat64 max.
func NewFloat64SummaryStatistics() *Float64SummaryStatistics {
	return &Float64SummaryStatistics{
		Count: 0,
		Sum:   0,
		Min:   math.MaxFloat64,
		Max:   -math.MaxFloat64,
	}
}

// Accept records a new float64 value into the summary information.
func (this *Float64SummaryStatistics) Accept(value float64) {
	if this.Count == 0 {
		this.Min = value
		this.Max = value
	}
	this.Count++
	this.Sum += value
	if value < this.Min {
		this.Min = value
	}
	if value > this.Max {
		this.Max = value
	}
}

// Combine combines the state of another Float64SummaryStatistics into this one.
func (this *Float64SummaryStatistics) Combine(other *Float64SummaryStatistics) {
	if other == nil || other.Count == 0 {
		return
	}
	if this.Count == 0 {
		this.Count = other.Count
		this.Sum = other.Sum
		this.Min = other.Min
		this.Max = other.Max
		return
	}
	this.Count += other.Count
	this.Sum += other.Sum
	if other.Min < this.Min {
		this.Min = other.Min
	}
	if other.Max > this.Max {
		this.Max = other.Max
	}
}

// Average returns the arithmetic mean of values recorded, or zero if no values have been recorded.
func (this *Float64SummaryStatistics) Average() float64 {
	if this.Count == 0 {
		return 0
	}
	return this.Sum / float64(this.Count)
}

// String returns a string representation of the statistics.
func (this *Float64SummaryStatistics) String() string {
	return fmt.Sprintf("Float64SummaryStatistics{count=%d, sum=%f, min=%f, average=%f, max=%f}",
		this.Count, this.Sum, this.Min, this.Average(), this.Max)
}
