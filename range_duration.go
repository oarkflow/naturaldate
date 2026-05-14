package naturaldate

import (
	"strings"
	"time"
)

// ParseRange parses natural date ranges such as "last 7 days", "this month",
// "next quarter", "Q1 2026", "from monday to friday", and
// "between march 1 and march 5".
func ParseRange(s string, opts ...Options) (DateRange, bool) {
	options := normalizeOptions(firstOptions(opts))
	src := strings.TrimSpace(s)
	if src == "" {
		return DateRange{}, false
	}
	lower := strings.ToLower(src)

	if start, end, ok := parseExplicitRange(src, lower, options); ok {
		return DateRange{Start: start.Time, End: rangeEndForResult(end), Truncated: coarserUnit(start.Truncated, end.Truncated), Direction: directionFromCompare(start.Time, options.Reference), MatchStart: 0, MatchEnd: len(s), Text: s}, true
	}

	if r, ok := parseRelativeRange(src, lower, options); ok {
		r.MatchStart = 0
		r.MatchEnd = len(s)
		r.Text = s
		return r, true
	}

	if strings.HasPrefix(lower, "q") {
		if point, ok := Parse(src, options); ok && point.Truncated == UnitQuarter {
			return DateRange{Start: point.Time, End: point.Time.AddDate(0, 3, 0), Truncated: UnitQuarter, Direction: point.Direction, MatchStart: 0, MatchEnd: len(s), Text: s}, true
		}
	}

	if point, ok := Parse(src, options); ok {
		return DateRange{Start: point.Time, End: rangeEndForResult(point), Truncated: point.Truncated, Direction: point.Direction, MatchStart: point.Start, MatchEnd: point.End, Text: point.Text}, true
	}
	return DateRange{}, false
}

func parseExplicitRange(src, lower string, options Options) (Result, Result, bool) {
	body := ""
	sep := ""
	if strings.HasPrefix(lower, "from ") {
		body = strings.TrimSpace(src[5:])
		sep = " to "
	} else if strings.HasPrefix(lower, "between ") {
		body = strings.TrimSpace(src[8:])
		sep = " and "
	}
	if body == "" {
		return Result{}, Result{}, false
	}
	idx := strings.Index(strings.ToLower(body), sep)
	if idx < 0 {
		return Result{}, Result{}, false
	}
	left := strings.TrimSpace(body[:idx])
	right := strings.TrimSpace(body[idx+len(sep):])
	if left == "" || right == "" {
		return Result{}, Result{}, false
	}
	start, ok := Parse(left, options)
	if !ok {
		return Result{}, Result{}, false
	}
	endOpts := options
	endOpts.Reference = start.Time
	endOpts.WeekdayDir = Future
	end, ok := Parse(right, endOpts)
	if !ok {
		return Result{}, Result{}, false
	}
	return start, end, true
}

func parseRelativeRange(src, lower string, options Options) (DateRange, bool) {
	parts := strings.Fields(lower)
	if len(parts) == 2 {
		dir, ok := rangeDirection(parts[0])
		if !ok {
			return DateRange{}, false
		}
		unit, ok := wordToUnit(parts[1])
		if !ok || unit == UnitSecond || unit == UnitMinute || unit == UnitHour {
			return DateRange{}, false
		}
		return periodRange(options.Reference, unit, dir), true
	}
	if len(parts) == 3 && (parts[0] == "last" || parts[0] == "past") {
		n, ok := wordToInt(parts[1])
		if !ok {
			n = atoi(parts[1])
		}
		unit, uok := wordToUnit(parts[2])
		if n < 1 || !uok {
			return DateRange{}, false
		}
		start := shiftByUnitN(options.Reference, unit, -n)
		return DateRange{Start: start, End: options.Reference, Truncated: unit, Direction: Past}, true
	}
	_ = src
	return DateRange{}, false
}

func rangeDirection(s string) (Direction, bool) {
	switch s {
	case "this":
		return Present, true
	case "next":
		return Future, true
	case "last", "past":
		return Past, true
	}
	return Present, false
}

func periodRange(ref time.Time, unit Unit, dir Direction) DateRange {
	base := ref
	if dir == Past {
		base = shiftByUnit(ref, unit, Past)
	} else if dir == Future {
		base = shiftByUnit(ref, unit, Future)
	}
	var start, end time.Time
	switch unit {
	case UnitDay:
		start = startOfDay(base)
		end = start.AddDate(0, 0, 1)
	case UnitWeek:
		start = startOfWeek(base)
		end = start.AddDate(0, 0, 7)
	case UnitMonth:
		start = startOfMonth(base)
		end = start.AddDate(0, 1, 0)
	case UnitQuarter:
		start = startOfQuarter(base)
		end = start.AddDate(0, 3, 0)
	case UnitYear:
		start = startOfYear(base)
		end = start.AddDate(1, 0, 0)
	default:
		start = base
		end = rangeEndForResult(Result{Time: base, Truncated: unit})
	}
	return DateRange{Start: start, End: end, Truncated: unit, Direction: dir}
}

func rangeEndForResult(r Result) time.Time {
	switch r.Truncated {
	case UnitSecond:
		return r.Time.Add(time.Second)
	case UnitMinute:
		return r.Time.Add(time.Minute)
	case UnitHour:
		return r.Time.Add(time.Hour)
	case UnitDay:
		return startOfDay(r.Time).AddDate(0, 0, 1)
	case UnitWeek:
		return startOfWeek(r.Time).AddDate(0, 0, 7)
	case UnitMonth:
		return startOfMonth(r.Time).AddDate(0, 1, 0)
	case UnitQuarter:
		return startOfQuarter(r.Time).AddDate(0, 3, 0)
	case UnitYear:
		return startOfYear(r.Time).AddDate(1, 0, 0)
	default:
		return r.Time
	}
}

func coarserUnit(a, b Unit) Unit {
	if a > b {
		return a
	}
	return b
}

// ParseDuration parses clock-safe durations. Calendar units larger than weeks
// are rejected because time.Duration cannot represent them exactly.
func ParseDuration(s string) (time.Duration, bool) {
	cd, ok := ParseCalendarDuration(s)
	if !ok || cd.Years != 0 || cd.Months != 0 {
		return 0, false
	}
	return time.Duration(cd.Weeks*7+cd.Days)*24*time.Hour + cd.Duration, true
}

// ParseCalendarDuration parses mixed calendar and clock durations such as
// "one week and three days", "2h 30m", or "1 month 2 days".
func ParseCalendarDuration(s string) (CalendarDuration, bool) {
	var p parser
	p.src = s
	p.lex.init(s)
	var out CalendarDuration
	seen := false
	for p.lex.remaining() > 0 {
		if p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "and") {
			p.lex.next()
			continue
		}
		n, unit, ok := p.parseQuantity()
		if !ok || n < 0 {
			return CalendarDuration{}, false
		}
		seen = true
		switch unit {
		case UnitSecond:
			out.Duration += time.Duration(n) * time.Second
		case UnitMinute:
			out.Duration += time.Duration(n) * time.Minute
		case UnitHour:
			out.Duration += time.Duration(n) * time.Hour
		case UnitDay:
			out.Days += n
		case UnitWeek:
			out.Weeks += n
		case UnitMonth:
			out.Months += n
		case UnitQuarter:
			out.Months += n * 3
		case UnitYear:
			out.Years += n
		default:
			return CalendarDuration{}, false
		}
	}
	return out, seen
}
