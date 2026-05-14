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
	Every     Unit // UnitSecond, UnitMinute, etc.
	Interval  int  // every N <units>
	At        Clock
	HasAt     bool
	OnDay     int8 // 0=any, 1=Mon … 7=Sun (ISO)
	OnDate    int8 // day-of-month anchor (0=none)
	OnMonth   int8 // month anchor (0=none)
	OnOrdinal int8 // nth occurrence: 1=first, 2=second, ..., -1=last
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

	// Location is the timezone used for parsing calendar expressions.
	// When set, Reference is converted into this location before parsing.
	Location *time.Location

	// Weekday preference: when a bare weekday name appears, prefer the
	// nearest Past or Future occurrence. Default: Past.
	WeekdayDir Direction

	// AllowEmbedded: when true, scan the whole input string for a date
	// expression rather than requiring the whole string to be a date.
	AllowEmbedded bool

	// Holidays is a list of dates to skip when calculating business days.
	// Each date should be in the format "YYYY-MM-DD" or as time.Time values.
	Holidays []time.Time
}

// Parse parses a natural-language date/time expression.
// It returns the Result and true on success, or a zero Result and false
// on failure. No allocations are made during parsing.
func Parse(s string, opts ...Options) (Result, bool) {
	options := normalizeOptions(firstOptions(opts))
	if body, loc, ok := stripTimezoneSuffix(s); ok {
		s = body
		options.Location = loc
		options.Reference = options.Reference.In(loc)
	}

	if r, ok := fastParse(s, options); ok {
		return r, true
	}
	p := parser{src: s, ref: options.Reference, wdir: options.WeekdayDir, holidays: options.Holidays}
	return p.parse(options.AllowEmbedded)
}

// ParseAll extracts every date/time expression found in s.
// It always scans embedded text; use AppendAll to reuse caller-owned storage.
func ParseAll(s string, opts ...Options) []Result {
	return AppendAll(nil, s, opts...)
}

// AppendAll appends every date/time expression found in s to dst.
// It is useful in hot paths where the caller wants to reuse result storage.
func AppendAll(dst []Result, s string, opts ...Options) []Result {
	options := normalizeOptions(firstOptions(opts))
	p := parser{src: s, ref: options.Reference, wdir: options.WeekdayDir, holidays: options.Holidays}
	return p.parseAll(dst)
}

// MustParse is like Parse but panics on failure.
func MustParse(s string, opts ...Options) Result {
	r, ok := Parse(s, opts...)
	if !ok {
		panic("naturaldate: cannot parse " + s)
	}
	return r
}

// Next returns the first occurrence of r after after.
func (r Recurrence) Next(after time.Time) (time.Time, bool) {
	if r.Interval < 1 {
		return time.Time{}, false
	}
	next := nextOccurrenceFrom(after, &r)
	if !next.After(after) {
		return time.Time{}, false
	}
	return next, true
}

// Next returns the next occurrence for recurring results.
func (r Result) Next(after time.Time) (time.Time, bool) {
	if !r.HasRecur {
		return time.Time{}, false
	}
	return r.Recur.Next(after)
}
