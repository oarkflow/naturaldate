// Package naturaldate provides high-performance, zero-allocation natural language
// date and time parsing for Go. It supports absolute dates, relative expressions,
// recurring schedules, and embedded date extraction from free-form text.
package naturaldate

import (
	"time"
)

// Direction indicates whether a relative reference points forward or backward in time.
type Direction int8

const (
	Past    Direction = -1
	Present Direction = 0
	Future  Direction = 1
)

// Result holds a parsed date/time along with metadata.
type Result struct {
	Time      time.Time
	Recur     Recurrence // valid when HasRecur is true
	HasRecur  bool
	Direction Direction
	Truncated Unit // finest unit mentioned (e.g. Hour for "10am")
}

// Unit represents a calendar/time granularity.
type Unit int8

const (
	UnitNone   Unit = iota
	UnitSecond      // "10:05:22pm"
	UnitMinute      // "10:05pm"
	UnitHour        // "10am"
	UnitDay         // "today", "tomorrow"
	UnitWeek        // "next week"
	UnitMonth       // "last month"
	UnitYear        // "next year"
)

// Recurrence describes a repeating schedule.
type Recurrence struct {
	Every    Unit
	Interval int   // every N <units>
	At       Clock // optional time-of-day anchor (valid when HasAt is true)
	HasAt    bool
	OnDay    int8 // 0=any, 1=Mon … 7=Sun (ISO)
	OnDate   int8 // day-of-month anchor (0=none)
	OnMonth  int8 // month anchor (0=none)
}

// Clock is an optional time-of-day value stored without heap allocation.
type Clock struct {
	Hour, Min, Sec int
}

// Options configures parsing behaviour.
type Options struct {
	// Reference is the "now" moment used for relative expressions.
	// Defaults to time.Now() when zero.
	Reference time.Time

	// Weekday preference: when a bare weekday name appears, prefer the
	// nearest Past or Future occurrence. Default: Past.
	WeekdayDir Direction

	// AllowEmbedded: when true, scan the whole input string for a date
	// expression rather than requiring the whole string to be a date.
	AllowEmbedded bool
}

// Parse parses a natural-language date/time expression.
// It returns the Result and true on success, or a zero Result and false
// on failure. No allocations are made during parsing.
func Parse(s string, opts ...Options) (Result, bool) {
	var options Options
	if len(opts) > 0 {
		options = opts[0]
	}
	if options.Reference.IsZero() {
		options.Reference = time.Now()
	}
	if r, ok := fastParse(s, options); ok {
		return r, true
	}
	p := parser{src: s, ref: options.Reference, wdir: options.WeekdayDir}
	if options.WeekdayDir == 0 {
		p.wdir = Past
	}
	return p.parse(options.AllowEmbedded)
}

// MustParse is like Parse but panics on failure.
func MustParse(s string, opts ...Options) Result {
	var options Options
	if len(opts) > 0 {
		options = opts[0]
	}
	r, ok := Parse(s, options)
	if !ok {
		panic("naturaldate: cannot parse " + s)
	}
	return r
}
