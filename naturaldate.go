// Package naturaldate provides high-performance, zero-allocation natural language
// date and time parsing for Go. It supports absolute dates, relative expressions,
// recurring schedules, and embedded date extraction from free-form text.
package naturaldate

import (
	"fmt"
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
	Time       time.Time
	Recur      Recurrence // valid when HasRecur is true
	HasRecur   bool
	Direction  Direction
	Truncated  Unit // finest unit mentioned (e.g. Hour for "10am")
	Start      int  // byte offset of the matched expression
	End        int  // byte offset immediately after the matched expression
	Text       string
	Confidence float64 // 1.0 for deterministic parses; lower for fuzzy/casual matches
}

// Unit represents a calendar/time granularity.
type Unit int8

const (
	UnitNone    Unit = iota
	UnitSecond       // "10:05:22pm"
	UnitMinute       // "10:05pm"
	UnitHour         // "10am"
	UnitDay          // "today", "tomorrow"
	UnitWeek         // "next week"
	UnitMonth        // "last month"
	UnitYear         // "next year"
	UnitQuarter      // "next quarter"
)

// DateOrder controls ambiguous numeric dates.
type DateOrder int8

const (
	DateOrderDefault DateOrder = iota
	DateOrderMDY
	DateOrderDMY
	DateOrderYMD
)

// ParseMode controls how much non-date language Parse tolerates.
type ParseMode int8

const (
	ModeStrict ParseMode = iota
	ModeCasual
	ModeFuzzy
)

// ParseErrorKind classifies parse failures returned by ParseWithError.
type ParseErrorKind int8

const (
	ErrEmptyInput ParseErrorKind = iota + 1
	ErrInvalidExpression
	ErrTrailingInput
)

// ParseError describes why parsing failed.
type ParseError struct {
	Input string
	Kind  ParseErrorKind
	Pos   int
}

func (e *ParseError) Error() string {
	if e == nil {
		return "naturaldate: parse error"
	}
	switch e.Kind {
	case ErrEmptyInput:
		return "naturaldate: empty input"
	case ErrTrailingInput:
		return fmt.Sprintf("naturaldate: trailing input at byte %d", e.Pos)
	default:
		return "naturaldate: invalid date expression"
	}
}

// Recurrence describes a repeating schedule.
type Recurrence struct {
	Every     Unit // UnitSecond, UnitMinute, etc.
	Interval  int  // every N <units>
	At        Clock
	HasAt     bool
	OnDay     int8 // 0=any, 1=Mon … 7=Sun (ISO)
	OnDate    int8 // day-of-month anchor (0=none, -1=last day)
	OnMonth   int8 // month anchor (0=none)
	OnOrdinal int8 // nth occurrence: 1=first, 2=second, ..., -1=last
	OnWeekday bool // every weekday
	AlsoOnDay int8 // optional second weekday for "monday and wednesday"
	ExceptDay int8 // optional weekday exclusion for "every weekday except friday"
	Until     time.Time
	Count     int
}

// Clock is an optional time-of-day value stored without heap allocation.
type Clock struct {
	Hour, Min, Sec int
}

// Options configures parsing behaviour.
type Options struct {
	// Now returns the default reference time when Reference is zero.
	// Defaults to time.Now.
	Now func() time.Time

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
	// Each date is compared by calendar day in the parser location.
	Holidays []time.Time

	// WeekendDays configures non-working weekdays for business-day expressions.
	// Defaults to Saturday and Sunday.
	WeekendDays []time.Weekday

	// BusinessDayEnd is the clock used for COB/EOD expressions.
	// Defaults to 17:00.
	BusinessDayEnd Clock

	// FiscalYearStartMonth configures fiscal quarter/year parsing.
	// Defaults to January.
	FiscalYearStartMonth time.Month

	// DateOrder controls ambiguous numeric dates. Default keeps the historic
	// behavior: slash prefers MM/DD, dash and dot prefer DD/MM.
	DateOrder DateOrder

	// Mode controls tolerance for filler words. Default is ModeStrict.
	Mode ParseMode
}

// DateRange is an inclusive start and exclusive end calendar interval.
type DateRange struct {
	Start      time.Time
	End        time.Time
	Truncated  Unit
	Direction  Direction
	MatchStart int
	MatchEnd   int
	Text       string
}

// CalendarDuration stores a duration that may include calendar units.
type CalendarDuration struct {
	Years, Months, Weeks, Days int
	Duration                   time.Duration
}

// AddTo applies d to t using calendar-safe arithmetic for years, months, weeks,
// and days, then adds the clock duration.
func (d CalendarDuration) AddTo(t time.Time) time.Time {
	return t.AddDate(d.Years, d.Months, d.Weeks*7+d.Days).Add(d.Duration)
}

// Parse parses a natural-language date/time expression.
// It returns the Result and true on success, or a zero Result and false
// on failure. No allocations are made during parsing.
func Parse(s string, opts ...Options) (Result, bool) {
	options := normalizeOptions(firstOptions(opts))
	r, ok, _ := parseInternal(s, options)
	return r, ok
}

// ParseWithError is like Parse but returns a structured error on failure.
func ParseWithError(s string, opts ...Options) (Result, error) {
	options := normalizeOptions(firstOptions(opts))
	r, ok, parsedInput := parseInternal(s, options)
	if ok {
		return r, nil
	}
	trimmed := trimSpaceASCII(parsedInput)
	if trimmed == "" {
		return Result{}, &ParseError{Input: s, Kind: ErrEmptyInput}
	}
	if pos, ok := firstTrailingPos(parsedInput, options); ok {
		return Result{}, &ParseError{Input: s, Kind: ErrTrailingInput, Pos: pos}
	}
	return Result{}, &ParseError{Input: s, Kind: ErrInvalidExpression}
}

// ParseAllWithError extracts every date/time expression found in s and returns
// a structured error when none can be found.
func ParseAllWithError(s string, opts ...Options) ([]Result, error) {
	results := ParseAll(s, opts...)
	if len(results) > 0 {
		return results, nil
	}
	options := normalizeOptions(firstOptions(opts))
	trimmed := trimSpaceASCII(s)
	if trimmed == "" {
		return nil, &ParseError{Input: s, Kind: ErrEmptyInput}
	}
	if pos, ok := firstTrailingPos(s, options); ok {
		return nil, &ParseError{Input: s, Kind: ErrTrailingInput, Pos: pos}
	}
	return nil, &ParseError{Input: s, Kind: ErrInvalidExpression}
}

func parseInternal(s string, options Options) (Result, bool, string) {
	if body, loc, ok := stripTimezoneSuffix(s); ok {
		s = body
		options.Location = loc
		options.Reference = options.Reference.In(loc)
	}

	if r, ok := fastParse(s, options); ok {
		return withMatch(r, s, 0, len(s)), true, s
	}
	p := newParser(s, options)
	if r, ok := p.parse(options.AllowEmbedded || options.Mode != ModeStrict); ok {
		return r, true, s
	}
	return Result{}, false, s
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
	p := newParser(s, options)
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

// Match returns the matched source substring.
func (r Result) Match() string {
	return r.Text
}

func withMatch(r Result, src string, start, end int) Result {
	if start < 0 {
		start = 0
	}
	if end < start {
		end = start
	}
	if end > len(src) {
		end = len(src)
	}
	r.Start = start
	r.End = end
	r.Text = src[start:end]
	if r.Confidence == 0 {
		r.Confidence = 1
	}
	return r
}

func firstTrailingPos(s string, opts Options) (int, bool) {
	p := newParser(s, opts)
	p.lex.init(s)
	if r, ok := p.parseExpr(false); ok {
		_ = r
		if !p.lex.atEnd() && p.lex.pos < p.lex.n {
			return p.lex.tokens[p.lex.pos].lo, true
		}
	}
	return 0, false
}
