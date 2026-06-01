package naturaldate

import "time"

// parser holds the mutable state for one parse attempt.
// All state is on the stack or in the fixed-size arrays of lexer/token.
type parser struct {
	src       string
	ref       time.Time
	wdir      Direction
	holidays  []time.Time
	weekends  []time.Weekday
	bizEnd    Clock
	fiscal    time.Month
	dateOrder DateOrder
	mode      ParseMode
	lex       lexer
}

func newParser(s string, options Options) parser {
	return parser{
		src:       s,
		ref:       options.Reference,
		wdir:      options.WeekdayDir,
		holidays:  options.Holidays,
		weekends:  options.WeekendDays,
		bizEnd:    options.BusinessDayEnd,
		fiscal:    options.FiscalYearStartMonth,
		dateOrder: options.DateOrder,
		mode:      options.Mode,
	}
}

func (p *parser) parse(embedded bool) (Result, bool) {
	p.lex.init(p.src)
	if embedded {
		p.skipCasualWords()
	}
	start := p.lex.pos
	if r, ok := p.parseExpr(embedded); ok && (embedded || p.lex.atEnd()) {
		return p.withTokenMatch(r, start), true
	}
	if embedded {
		for start := 1; start < p.lex.n; start++ {
			p.lex.reset(start)
			p.skipCasualWords()
			exprStart := p.lex.pos
			if r, ok := p.parseExpr(true); ok {
				return p.withTokenMatch(r, exprStart), true
			}
		}
	}
	return Result{}, false
}

func (p *parser) parseAll(dst []Result) []Result {
	p.lex.init(p.src)
	for start := 0; start < p.lex.n; {
		p.lex.reset(start)
		p.skipCasualWords()
		exprStart := p.lex.pos
		r, ok := p.parseExpr(false)
		if !ok {
			start++
			continue
		}
		r = p.withTokenMatch(r, exprStart)
		dst = append(dst, r)
		if p.lex.pos > start {
			start = p.lex.pos
		} else {
			start++
		}
	}
	return dst
}

func (p *parser) parseExpr(embedded bool) (Result, bool) {
	mark := p.lex.mark()

	if r, ok := p.parseRecurring(); ok {
		return r, true
	}
	p.lex.reset(mark)

	if r, ok := p.parseRelative(embedded); ok {
		return r, true
	}
	p.lex.reset(mark)

	if r, ok := p.parseSpecialRelative(); ok {
		return r, true
	}
	p.lex.reset(mark)

	if r, ok := p.parseAnchor(); ok {
		return r, true
	}
	p.lex.reset(mark)

	if r, ok := p.parseQuarterDate(); ok {
		return r, true
	}
	p.lex.reset(mark)

	if r, ok := p.parseLastNext(); ok {
		return r, true
	}
	p.lex.reset(mark)

	if r, ok := p.parseLastNextMonth(); ok {
		return r, true
	}
	p.lex.reset(mark)

	if r, ok := p.parseBareWeekday(); ok {
		return r, true
	}
	p.lex.reset(mark)

	if r, ok := p.parseMonthDay(); ok {
		return r, true
	}
	p.lex.reset(mark)

	if r, ok := p.parseBareMonth(); ok {
		return r, true
	}
	p.lex.reset(mark)

	if r, ok := p.parseOrdinalOfMonth(); ok {
		return r, true
	}
	p.lex.reset(mark)

	if r, ok := p.parseNumericDate(); ok {
		return r, true
	}
	p.lex.reset(mark)

	if r, ok := p.parseBareTime(); ok {
		return r, true
	}
	p.lex.reset(mark)

	return Result{}, false
}

// ── anchor words ───────────────────────────────────────────────────────────

func (p *parser) parseAnchor() (Result, bool) {
	t := p.lex.peek()
	if t.kind != tokWord {
		return Result{}, false
	}
	var base time.Time
	unit := UnitDay
	dir := Present
	switch {
	case p.lex.wordEq(t, "now"):
		p.lex.next()
		return Result{Time: p.ref, Truncated: UnitSecond, Direction: Present}, true
	case p.lex.wordEq(t, "today"):
		p.lex.next()
		base = startOfDay(p.ref)
	case p.lex.wordEq(t, "yesterday"):
		p.lex.next()
		base = startOfDay(p.ref).AddDate(0, 0, -1)
		dir = Past
	case p.lex.wordEq(t, "tomorrow"):
		p.lex.next()
		base = startOfDay(p.ref).AddDate(0, 0, 1)
		dir = Future
		if p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "morning") {
			p.lex.next()
			base = setClock(base, clockResult{9, 0, 0, UnitHour})
			unit = UnitHour
		} else if clk, ok := p.parsePartOfDayClock(); ok {
			base = setClock(base, clk)
			unit = clk.unit
		}
	case p.lex.wordEq(t, "tonight"):
		p.lex.next()
		base = setClock(startOfDay(p.ref), clockResult{20, 0, 0, UnitHour})
		if !base.After(p.ref) {
			base = base.AddDate(0, 0, 1)
		}
		unit = UnitHour
		dir = Future
	case p.lex.wordEq(t, "eod") || p.lex.wordEq(t, "cob"):
		p.lex.next()
		t := applyClock(startOfDay(p.ref), p.bizEnd, true)
		if !t.After(p.ref) {
			t = t.AddDate(0, 0, 1)
		}
		return Result{Time: t, Truncated: UnitHour, Direction: Future}, true
	case p.lex.wordEq(t, "midnight"):
		p.lex.next()
		t := startOfDay(p.ref)
		return Result{Time: t, Truncated: UnitHour, Direction: directionFromCompare(t, p.ref)}, true
	case p.lex.wordEq(t, "noon"):
		p.lex.next()
		t := startOfDay(p.ref).Add(12 * time.Hour)
		return Result{Time: t, Truncated: UnitHour, Direction: directionFromCompare(t, p.ref)}, true
	case p.lex.wordEq(t, "start"):
		p.lex.next()
		if result, ok := p.parseStartOf(); ok {
			return result, true
		}
		return Result{}, false
	case p.lex.wordEq(t, "beginning"):
		p.lex.next()
		if result, ok := p.parseStartOf(); ok {
			return result, true
		}
		return Result{}, false
	case p.lex.wordEq(t, "end"):
		p.lex.next()
		if p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "of") && p.lex.peekAt(1).kind == tokWord && p.lex.wordEq(p.lex.peekAt(1), "business") {
			p.lex.next()
			p.lex.next()
			if p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "day") {
				p.lex.next()
			}
			t := applyClock(startOfDay(p.ref), p.bizEnd, true)
			if !t.After(p.ref) {
				t = applyClock(startOfDay(addBusinessDays(p.ref, 1, p.holidays, p.weekends)), p.bizEnd, true)
			}
			return Result{Time: t, Truncated: UnitHour, Direction: Future}, true
		}
		if result, ok := p.parseEndOf(); ok {
			return result, true
		}
		return Result{}, false
	default:
		return Result{}, false
	}
	if clk, ok := p.parseAtTime(); ok {
		base = setClock(base, clk)
		unit = clk.unit
		if dir == Present {
			dir = directionFromCompare(base, p.ref)
		}
	} else if clk, ok := p.parsePartOfDayClock(); ok {
		base = setClock(base, clk)
		unit = clk.unit
		if dir == Present {
			dir = directionFromCompare(base, p.ref)
		}
	}
	return Result{Time: base, Truncated: unit, Direction: dir}, true
}

func (p *parser) parseStartOf() (Result, bool) {
	if p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "of") {
		p.lex.next()
	}
	if base, unit, ok := p.parseDirectedPeriodStart(); ok {
		return Result{Time: base, Truncated: unit, Direction: directionFromCompare(base, p.ref)}, true
	}
	switch {
	case p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "day"):
		p.lex.next()
		return Result{Time: startOfDay(p.ref), Truncated: UnitDay, Direction: Present}, true
	case p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "week"):
		p.lex.next()
		return Result{Time: startOfWeek(p.ref), Truncated: UnitWeek, Direction: Present}, true
	case p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "month"):
		p.lex.next()
		return Result{Time: startOfMonth(p.ref), Truncated: UnitMonth, Direction: Present}, true
	case p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "quarter"):
		p.lex.next()
		return Result{Time: startOfQuarter(p.ref), Truncated: UnitQuarter, Direction: Present}, true
	case p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "year"):
		p.lex.next()
		return Result{Time: startOfYear(p.ref), Truncated: UnitYear, Direction: Present}, true
	}
	return Result{}, false
}

func (p *parser) parseEndOf() (Result, bool) {
	if p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "of") {
		p.lex.next()
	}
	if base, unit, ok := p.parseDirectedPeriodStart(); ok {
		return Result{Time: periodEnd(base, unit), Truncated: unit, Direction: directionFromCompare(base, p.ref)}, true
	}
	switch {
	case p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "day"):
		p.lex.next()
		return Result{Time: endOfDay(p.ref), Truncated: UnitDay, Direction: Present}, true
	case p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "week"):
		p.lex.next()
		return Result{Time: endOfWeek(p.ref), Truncated: UnitWeek, Direction: Present}, true
	case p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "month"):
		p.lex.next()
		return Result{Time: endOfMonth(p.ref), Truncated: UnitMonth, Direction: Present}, true
	case p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "quarter"):
		p.lex.next()
		return Result{Time: endOfQuarter(p.ref), Truncated: UnitQuarter, Direction: Present}, true
	case p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "year"):
		p.lex.next()
		return Result{Time: endOfYear(p.ref), Truncated: UnitYear, Direction: Present}, true
	}
	return Result{}, false
}

func (p *parser) parseDirectedPeriodStart() (time.Time, Unit, bool) {
	t := p.lex.peek()
	if t.kind != tokWord {
		return time.Time{}, 0, false
	}
	dir := Present
	switch {
	case p.lex.wordEq(t, "this"):
		dir = Present
	case p.lex.wordEq(t, "next"):
		dir = Future
	case p.lex.wordEq(t, "last"):
		dir = Past
	default:
		return time.Time{}, 0, false
	}
	p.lex.next()
	if p.lex.peek().kind == tokWord && isBusinessDays(p.lex.val(p.lex.peek())) {
		p.lex.next()
		if unit, ok := p.peekUnit(); ok && unit == UnitDay {
			p.lex.next()
		}
		n := 1
		if dir == Past {
			n = -1
		}
		t := startOfDay(addBusinessDays(p.ref, n, p.holidays, p.weekends))
		return t, UnitDay, true
	}
	unit, ok := p.peekUnit()
	if !ok || unit == UnitSecond || unit == UnitMinute || unit == UnitHour {
		return time.Time{}, 0, false
	}
	p.lex.next()
	base := p.ref
	if dir != Present {
		base = shiftByUnit(p.ref, unit, dir)
	}
	return periodStart(base, unit), unit, true
}

// ── last / next ────────────────────────────────────────────────────────────

func (p *parser) parseLastNext() (Result, bool) {
	t := p.lex.peek()
	if t.kind != tokWord {
		return Result{}, false
	}
	var dir Direction
	switch {
	case p.lex.wordEq(t, "last"):
		dir = Past
	case p.lex.wordEq(t, "next") || p.lex.wordEq(t, "this"):
		dir = Future
	default:
		return Result{}, false
	}
	p.lex.next()

	if n, ok := p.parseNumber(); ok {
		if p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "weekdays") {
			p.lex.next()
			mul := 1
			if dir == Past {
				mul = -1
			}
			return Result{Time: addWeekdays(p.ref, mul*n), Truncated: UnitDay, Direction: dir}, true
		}
		if p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "business") {
			p.lex.next()
			if unit, ok := p.peekUnit(); ok && unit == UnitWeek {
				p.lex.next()
				mul := 1
				if dir == Past {
					mul = -1
				}
				return Result{Time: addBusinessWeeks(p.ref, mul*n, p.holidays, p.weekends), Truncated: UnitWeek, Direction: dir}, true
			}
		}
		return Result{}, false
	}

	t2 := p.lex.peek()
	if t2.kind != tokWord {
		return Result{}, false
	}
	v2 := p.lex.val(t2)

	if p.lex.wordEq(t2, "fiscal") {
		p.lex.next()
		if p.lex.peek().kind != tokWord {
			return Result{}, false
		}
		if p.lex.wordEq(p.lex.peek(), "quarter") {
			p.lex.next()
			base := fiscalPeriodStart(p.ref, UnitQuarter, dir, p.fiscal)
			return Result{Time: base, Truncated: UnitQuarter, Direction: dir}, true
		}
		if p.lex.wordEq(p.lex.peek(), "year") {
			p.lex.next()
			base := fiscalPeriodStart(p.ref, UnitYear, dir, p.fiscal)
			return Result{Time: base, Truncated: UnitYear, Direction: dir}, true
		}
		return Result{}, false
	}

	if isBusinessDays(v2) {
		p.lex.next()
		if unit, ok := p.peekUnit(); ok && unit == UnitDay {
			p.lex.next()
		}
		n := 1
		if dir == Past {
			n = -1
		}
		base := addBusinessDays(p.ref, n, p.holidays, p.weekends)
		return Result{Time: base, Truncated: UnitDay, Direction: dir}, true
	}

	if p.lex.wordEq(t2, "weekend") {
		p.lex.next()
		base := weekendStart(p.ref, dir)
		return Result{Time: base, Truncated: UnitDay, Direction: dir}, true
	}

	if wd, ok := wordToWeekday(v2); ok {
		p.lex.next()
		base := startOfDay(nearestWeekday(p.ref, wd, dir))
		if clk, ok := p.parseAtTime(); ok {
			base = setClock(base, clk)
		} else if clk, ok := p.parsePartOfDayClock(); ok {
			base = setClock(base, clk)
		}
		return Result{Time: base, Truncated: UnitDay, Direction: dir}, true
	}

	if unit, ok := wordToUnit(v2); ok {
		p.lex.next()
		base := shiftByUnit(p.ref, unit, dir)
		return Result{Time: base, Truncated: unit, Direction: dir}, true
	}
	return Result{}, false
}

func (p *parser) parseLastNextMonth() (Result, bool) {
	t := p.lex.peek()
	if t.kind != tokWord {
		return Result{}, false
	}
	var dir Direction
	switch {
	case p.lex.wordEq(t, "last"):
		dir = Past
	case p.lex.wordEq(t, "next"):
		dir = Future
	default:
		return Result{}, false
	}
	p.lex.next()

	t2 := p.lex.peek()
	if t2.kind != tokWord {
		return Result{}, false
	}
	mon, ok := wordToMonth(p.lex.val(t2))
	if !ok {
		return Result{}, false
	}
	p.lex.next()

	y := p.ref.Year()
	if dir == Future {
		if p.ref.Month() >= mon {
			y++
		}
	} else {
		if p.ref.Month() <= mon {
			y--
		}
	}
	base := time.Date(y, mon, 1, 0, 0, 0, 0, p.ref.Location())
	return Result{Time: base, Truncated: UnitMonth, Direction: dir}, true
}

// ── bare weekday ───────────────────────────────────────────────────────────

func (p *parser) parseBareWeekday() (Result, bool) {
	t := p.lex.peek()
	if t.kind != tokWord {
		return Result{}, false
	}
	wd, ok := wordToWeekday(p.lex.val(t))
	if !ok {
		return Result{}, false
	}
	p.lex.next()
	base := startOfDay(nearestWeekday(p.ref, wd, p.wdir))
	dir := p.wdir
	if clk, ok := p.parseAtTime(); ok {
		candidate := setClock(base, clk)
		if candidate.Before(p.ref) && dir == Past {
			base = startOfDay(nearestWeekday(p.ref, wd, Future))
			candidate = setClock(base, clk)
			dir = Future
		}
		return Result{Time: candidate, Truncated: clk.unit, Direction: dir}, true
	}
	if clk, ok := p.parsePartOfDayClock(); ok {
		candidate := setClock(base, clk)
		if candidate.Before(p.ref) && dir == Past {
			base = startOfDay(nearestWeekday(p.ref, wd, Future))
			candidate = setClock(base, clk)
			dir = Future
		}
		return Result{Time: candidate, Truncated: clk.unit, Direction: dir}, true
	}
	return Result{Time: base, Truncated: UnitDay, Direction: dir}, true
}

// ── relative ───────────────────────────────────────────────────────────────

func (p *parser) parseRelative(embedded bool) (Result, bool) {
	mark := p.lex.mark()
	if p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "in") {
		p.lex.next()
	}
	if p.lex.remaining() == 0 {
		p.lex.reset(mark)
		return Result{}, false
	}
	if r, ok := p.parseRelativeHere(); ok {
		return r, true
	}
	p.lex.reset(mark)
	return Result{}, false
}

func (p *parser) parseRelativeHere() (Result, bool) {
	mark := p.lex.mark()
	if p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "half") {
		p.lex.next()
		if p.lex.peek().kind == tokWord && (p.lex.wordEq(p.lex.peek(), "an") || p.lex.wordEq(p.lex.peek(), "a")) {
			p.lex.next()
		}
		if p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "hour") {
			p.lex.next()
			return Result{Time: p.ref.Add(30 * time.Minute), Truncated: UnitMinute, Direction: Future}, true
		}
		p.lex.reset(mark)
	}
	if p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "quarter") {
		p.lex.next()
		if p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "hour") {
			p.lex.next()
			return Result{Time: p.ref.Add(15 * time.Minute), Truncated: UnitMinute, Direction: Future}, true
		}
		p.lex.reset(mark)
	}
	if n, ok := p.parseNumber(); ok {
		if p.lex.peek().kind == tokWord && isBusinessDays(p.lex.val(p.lex.peek())) {
			p.lex.next()
			if unit, ok := p.peekUnit(); ok && unit == UnitDay {
				p.lex.next()
			}
			dir := Future
			if p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "ago") {
				dir = Past
				p.lex.next()
			}
			mul := 1
			if dir == Past {
				mul = -1
			}
			return Result{Time: addBusinessDays(p.ref, mul*n, p.holidays, p.weekends), Truncated: UnitDay, Direction: dir}, true
		}
		if p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "weekdays") {
			p.lex.next()
			dir := Future
			if p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "ago") {
				dir = Past
				p.lex.next()
			}
			mul := 1
			if dir == Past {
				mul = -1
			}
			return Result{Time: addWeekdays(p.ref, mul*n), Truncated: UnitDay, Direction: dir}, true
		}
	}
	p.lex.reset(mark)

	n, unit, ok := p.parseQuantity()
	if !ok {
		p.lex.reset(mark)
		return Result{}, false
	}

	dir := Future
	if p.lex.peek().kind == tokWord {
		if p.lex.wordEq(p.lex.peek(), "ago") {
			dir = Past
			p.lex.next()
		} else if p.lex.wordEq(p.lex.peek(), "from") {
			fromMark := p.lex.mark()
			p.lex.next()
			if p.lex.peek().kind == tokWord && (p.lex.wordEq(p.lex.peek(), "now") || p.lex.wordEq(p.lex.peek(), "today")) {
				p.lex.next()
			} else {
				p.lex.reset(fromMark)
			}
		}
	}

	mul := 1
	if dir == Past {
		mul = -1
	}
	return Result{Time: shiftByUnitN(p.ref, unit, mul*n), Truncated: unit, Direction: dir}, true
}

func (p *parser) parseSpecialRelative() (Result, bool) {
	mark := p.lex.mark()
	if p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "the") {
		p.lex.next()
	}
	if p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "day") {
		p.lex.next()
		if p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "after") {
			p.lex.next()
			if p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "tomorrow") {
				p.lex.next()
				return Result{Time: startOfDay(p.ref).AddDate(0, 0, 2), Truncated: UnitDay, Direction: Future}, true
			}
		} else if p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "before") {
			p.lex.next()
			if p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "yesterday") {
				p.lex.next()
				return Result{Time: startOfDay(p.ref).AddDate(0, 0, -2), Truncated: UnitDay, Direction: Past}, true
			}
		}
	}
	p.lex.reset(mark)

	if p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "quarter") {
		p.lex.next()
		if p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "past") {
			p.lex.next()
			if clk, ok := p.parseClockExpr(); ok {
				clk.min = 15
				clk.unit = UnitMinute
				base := setClock(startOfDay(p.ref), clk)
				if !base.After(p.ref) {
					base = base.AddDate(0, 0, 1)
				}
				return Result{Time: base, Truncated: UnitMinute, Direction: Future}, true
			}
		}
	}
	p.lex.reset(mark)

	if p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "half") {
		p.lex.next()
		if p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "past") {
			p.lex.next()
			if clk, ok := p.parseClockExpr(); ok {
				clk.min = 30
				clk.unit = UnitMinute
				base := setClock(startOfDay(p.ref), clk)
				if !base.After(p.ref) {
					base = base.AddDate(0, 0, 1)
				}
				return Result{Time: base, Truncated: UnitMinute, Direction: Future}, true
			}
		}
	}
	p.lex.reset(mark)

	if n, ok := p.parseNumber(); ok {
		if wd, ok := p.peekWeekday(); ok {
			p.lex.next()
			if p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "from") {
				p.lex.next()
				if p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "now") {
					p.lex.next()
				}
				return Result{Time: nthWeekdayFrom(p.ref, wd, n, Future), Truncated: UnitDay, Direction: Future}, true
			}
			if p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "ago") {
				p.lex.next()
				return Result{Time: nthWeekdayFrom(p.ref, wd, n, Past), Truncated: UnitDay, Direction: Past}, true
			}
		}
	}
	p.lex.reset(mark)

	if p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "this") {
		p.lex.next()
		if clk, ok := p.parsePartOfDayClock(); ok {
			base := setClock(startOfDay(p.ref), clk)
			dir := directionFromCompare(base, p.ref)
			if base.Before(p.ref) {
				base = base.AddDate(0, 0, 1)
				dir = Future
			}
			return Result{Time: base, Truncated: clk.unit, Direction: dir}, true
		}
		if p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "weekend") {
			p.lex.next()
			return Result{Time: weekendStart(p.ref, Present), Truncated: UnitDay, Direction: directionFromCompare(weekendStart(p.ref, Present), p.ref)}, true
		}
	}
	p.lex.reset(mark)
	return Result{}, false
}

func (p *parser) parseQuarterDate() (Result, bool) {
	t := p.lex.peek()
	if t.kind != tokWord || !p.lex.wordEq(t, "q") {
		return Result{}, false
	}
	p.lex.next()
	tq := p.lex.peek()
	if tq.kind != tokNumber {
		return Result{}, false
	}
	q := atoi(p.lex.val(tq))
	if q < 1 || q > 4 {
		return Result{}, false
	}
	p.lex.next()
	year := p.ref.Year()
	if p.lex.peek().kind == tokNumber {
		year = atoi(p.lex.val(p.lex.peek()))
		if year < 100 {
			year += 2000
		}
		p.lex.next()
	}
	base := quarterStart(year, q, p.ref.Location())
	return Result{Time: base, Truncated: UnitQuarter, Direction: directionFromCompare(base, p.ref)}, true
}

// ── month + day ────────────────────────────────────────────────────────────

func (p *parser) parseMonthDay() (Result, bool) {
	t := p.lex.peek()
	if t.kind != tokWord {
		return Result{}, false
	}
	mon, ok := wordToMonth(p.lex.val(t))
	if !ok {
		return Result{}, false
	}
	p.lex.next()
	if p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "the") {
		p.lex.next()
	}
	day, ok := p.parseOrdinalOrNumber()
	if !ok {
		return Result{}, false
	}
	clk, hasClock := p.parseAtTime()
	base, ok := nextMonthDay(p.ref, mon, day, clk, hasClock)
	if !ok {
		return Result{}, false
	}
	return Result{Time: base, Truncated: UnitDay, Direction: directionForDateResult(base, p.ref, hasClock)}, true
}

func (p *parser) parseBareMonth() (Result, bool) {
	t := p.lex.peek()
	if t.kind != tokWord {
		return Result{}, false
	}
	mon, ok := wordToMonth(p.lex.val(t))
	if !ok {
		return Result{}, false
	}
	p.lex.next()
	y := p.ref.Year()
	base := time.Date(y, mon, 1, 0, 0, 0, 0, p.ref.Location())
	if !base.After(p.ref) {
		base = time.Date(y+1, mon, 1, 0, 0, 0, 0, p.ref.Location())
	}
	return Result{Time: base, Truncated: UnitMonth, Direction: Future}, true
}

func (p *parser) parseOrdinalOfMonth() (Result, bool) {
	p.skipWords("remind", "me", "on")
	if p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "the") {
		p.lex.next()
	}
	day, ok := p.parseOrdinalOrNumber()
	if !ok {
		return Result{}, false
	}
	if p.lex.peek().kind != tokWord || !p.lex.wordEq(p.lex.peek(), "of") {
		return Result{}, false
	}
	p.lex.next()
	if p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "the") {
		p.lex.next()
	}
	t := p.lex.peek()
	if t.kind != tokWord {
		return Result{}, false
	}
	mon, ok := wordToMonth(p.lex.val(t))
	if !ok {
		return Result{}, false
	}
	p.lex.next()

	clk, hasClock := p.parseAtTime()
	base, ok := nextMonthDay(p.ref, mon, day, clk, hasClock)
	if !ok {
		return Result{}, false
	}
	return Result{Time: base, Truncated: UnitDay, Direction: directionForDateResult(base, p.ref, hasClock)}, true
}

// ── bare time ──────────────────────────────────────────────────────────────

func (p *parser) parseBareTime() (Result, bool) {
	if clk, ok := p.parseNaturalClockExpr(); ok {
		base := setClock(startOfDay(p.ref), clk)
		if !base.After(p.ref) {
			base = base.AddDate(0, 0, 1)
		}
		return Result{Time: base, Truncated: clk.unit, Direction: Future}, true
	}
	clk, ok := p.parseClockExpr()
	if !ok {
		return Result{}, false
	}
	base := setClock(startOfDay(p.ref), clk)
	if !base.After(p.ref) {
		base = base.AddDate(0, 0, 1)
	}
	return Result{Time: base, Truncated: clk.unit, Direction: Future}, true
}

// ── numeric date ───────────────────────────────────────────────────────────

func (p *parser) parseNumericDate() (Result, bool) {
	t0 := p.lex.peek()
	if t0.kind != tokNumber {
		return Result{}, false
	}
	p.lex.next()
	sep := p.lex.peek()
	if sep.kind != tokDash && sep.kind != tokSlash && sep.kind != tokDot {
		return Result{}, false
	}
	p.lex.next()
	t1 := p.lex.peek()
	if t1.kind != tokNumber {
		return Result{}, false
	}
	p.lex.next()

	v0 := atoi(p.lex.val(t0))
	v1 := atoi(p.lex.val(t1))
	if p.lex.peek().kind != sep.kind {
		// Two-part numeric date (e.g. 12/25 or 25-12).
		m, d, ok := resolveMonthDay(v0, v1, sep.kind, p.dateOrder)
		if !ok {
			return Result{}, false
		}
		clk, hasClock := p.parseAtTime()
		base, ok := nextMonthDay(p.ref, time.Month(m), d, clk, hasClock)
		if !ok {
			return Result{}, false
		}
		return Result{Time: base, Truncated: UnitDay, Direction: directionForDateResult(base, p.ref, hasClock)}, true
	}
	p.lex.next()
	t2 := p.lex.peek()
	if t2.kind != tokNumber {
		return Result{}, false
	}
	p.lex.next()
	v2 := atoi(p.lex.val(t2))

	var y, m, d int
	switch {
	case p.dateOrder == DateOrderYMD:
		y, m, d = v0, v1, v2
	case v0 > 31:
		y, m, d = v0, v1, v2
	default:
		var ok bool
		m, d, ok = resolveMonthDay(v0, v1, sep.kind, p.dateOrder)
		if !ok {
			return Result{}, false
		}
		y = v2
	}
	if y < 100 {
		y += 2000
	}
	if m < 1 || m > 12 || !validMonthDay(y, time.Month(m), d) {
		return Result{}, false
	}
	base := time.Date(y, time.Month(m), d, 0, 0, 0, 0, p.ref.Location())
	if clk, ok := p.parseAtTime(); ok {
		base = setClock(base, clk)
	}
	return Result{Time: base, Truncated: UnitDay, Direction: directionFromCompare(base, p.ref)}, true
}

func resolveMonthDay(v0, v1 int, sep tokenKind, order DateOrder) (m, d int, ok bool) {
	if v0 < 1 || v0 > 31 || v1 < 1 || v1 > 31 {
		return 0, 0, false
	}
	if v0 > 12 && v1 > 12 {
		return 0, 0, false
	}
	if order == DateOrderMDY {
		if v0 > 12 {
			return 0, 0, false
		}
		return v0, v1, true
	}
	if order == DateOrderDMY {
		if v1 > 12 {
			return 0, 0, false
		}
		return v1, v0, true
	}
	if order == DateOrderYMD {
		return 0, 0, false
	}
	if v0 > 12 { // D/M
		return v1, v0, true
	}
	if v1 > 12 { // M/D
		return v0, v1, true
	}
	if sep == tokSlash { // default US-style for slash
		return v0, v1, true
	}
	return v1, v0, true // default day/month for dash/dot
}

// ── recurring ──────────────────────────────────────────────────────────────

func (p *parser) parseRecurring() (Result, bool) {
	t := p.lex.peek()
	if t.kind != tokWord {
		return Result{}, false
	}
	switch {
	case p.lex.wordEq(t, "every"):
		p.lex.next()
		if p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "other") {
			p.lex.next()
			if unit, ok := p.peekUnit(); ok {
				p.lex.next()
				recur := Recurrence{Every: unit, Interval: 2}
				if clk, ok := p.parseAtTime(); ok {
					recur.At = Clock{Hour: clk.hour, Min: clk.min, Sec: clk.sec}
					recur.HasAt = true
				}
				next := p.nextOccurrence(&recur)
				return Result{Time: next, Recur: recur, HasRecur: true, Truncated: recur.Every, Direction: Future}, true
			}
			return Result{}, false
		}
	case p.lex.wordEq(t, "once"):
		p.lex.next()
		if p.lex.peek().kind == tokWord {
			if p.lex.wordEq(p.lex.peek(), "a") || p.lex.wordEq(p.lex.peek(), "an") {
				p.lex.next()
			}
		}
	default:
		return Result{}, false
	}

	var recur Recurrence

	if ord, ok := p.parseOrdinalModifier(); ok {
		if wd, ok := p.peekWeekday(); ok {
			p.lex.next()
			recur.Every = UnitMonth
			recur.Interval = 1
			recur.OnOrdinal = ord
			recur.OnDay = weekdayToISO(wd)
		} else {
			return Result{}, false
		}
	} else if day, ok := p.parseMonthlyDateHead(); ok {
		recur.Every = UnitMonth
		recur.Interval = 1
		recur.OnDate = int8(day)
	} else if p.peekIsQuantity() {
		qmark := p.lex.mark()
		if n, ok := p.parseOrdinalOrNumber(); ok && n > 0 {
			if wd, ok := p.peekWeekday(); ok {
				p.lex.next()
				recur.Every = UnitMonth
				recur.Interval = 1
				recur.OnOrdinal = int8(n)
				recur.OnDay = weekdayToISO(wd)
			} else {
				p.lex.reset(qmark)
			}
		} else {
			p.lex.reset(qmark)
		}
		if recur.Every == UnitNone {
			if n, unit, ok := p.parseQuantity(); ok {
				recur.Every = unit
				recur.Interval = n
			} else {
				return Result{}, false
			}
		}
	} else if p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "weekday") {
		p.lex.next()
		recur.Every = UnitDay
		recur.Interval = 1
		recur.OnWeekday = true
	} else if p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "last") && p.lex.peekAt(1).kind == tokWord && p.lex.wordEq(p.lex.peekAt(1), "day") {
		p.lex.next()
		p.lex.next()
		recur.Every = UnitMonth
		recur.Interval = 1
		recur.OnDate = -1
		if p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "of") {
			p.lex.next()
			if p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "the") {
				p.lex.next()
			}
			if p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "month") {
				p.lex.next()
			}
		}
	} else if unit, ok := p.peekUnit(); ok {
		p.lex.next()
		recur.Every = unit
		recur.Interval = 1
	} else if wd, ok := p.peekWeekday(); ok {
		p.lex.next()
		recur.Every = UnitWeek
		recur.Interval = 1
		recur.OnDay = weekdayToISO(wd)
		if also, ok := p.parseAndWeekday(); ok {
			recur.AlsoOnDay = weekdayToISO(also)
		}
	} else {
		return Result{}, false
	}
	if recur.Interval < 1 {
		return Result{}, false
	}

	// "on [the] <weekday|ordinal>"
	if p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "on") {
		p.lex.next()
		if p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "the") {
			p.lex.next()
		}
		if wd, ok := p.peekWeekday(); ok {
			p.lex.next()
			recur.OnDay = weekdayToISO(wd)
			if also, ok := p.parseAndWeekday(); ok {
				recur.AlsoOnDay = weekdayToISO(also)
			}
		} else if mon, ok := p.peekMonth(); ok {
			p.lex.next()
			recur.OnMonth = int8(mon)
			if day, ok := p.parseOrdinalOrNumber(); ok {
				if !validMonthDayInRange(p.ref.Year(), p.ref.Year()+8, mon, day) {
					return Result{}, false
				}
				recur.OnDate = int8(day)
			}
		} else if day, ok := p.parseOrdinalOrNumber(); ok {
			if day < 1 || day > 31 {
				return Result{}, false
			}
			recur.OnDate = int8(day)
			if p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "of") {
				p.lex.next()
				if p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "the") {
					p.lex.next()
				}
				if p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "month") {
					p.lex.next()
				}
			}
		} else if p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "last") {
			p.lex.next()
			if p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "day") {
				p.lex.next()
				recur.OnDate = -1
				if p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "of") {
					p.lex.next()
					if p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "the") {
						p.lex.next()
					}
					if p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "month") {
						p.lex.next()
					}
				}
			} else {
				return Result{}, false
			}
		} else {
			return Result{}, false
		}
	}

	// "of the month" or "of the week" modifier for weekday (e.g., "first monday of the month")
	if p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "of") {
		p.lex.next()
		if p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "the") {
			p.lex.next()
		}
		if p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "month") {
			p.lex.next()
			recur.Every = UnitMonth
			if recur.OnDay != 0 && recur.OnOrdinal == 0 {
				recur.OnOrdinal = 1 // Default to first occurrence
			}
		} else if p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "week") {
			p.lex.next()
			recur.Every = UnitWeek
		}
	}

	// Check for "first", "second", "third", "fourth", "last" modifiers
	if ord, ok := p.parseOrdinalModifier(); ok {
		recur.OnOrdinal = ord
	}

	// "at <time>" or named time
	if clk, ok := p.parseAtTime(); ok {
		recur.At = Clock{Hour: clk.hour, Min: clk.min, Sec: clk.sec}
		recur.HasAt = true
	} else if p.lex.peek().kind == tokWord {
		switch {
		case p.lex.wordEq(p.lex.peek(), "midnight"):
			p.lex.next()
			recur.At = Clock{0, 0, 0}
			recur.HasAt = true
		case p.lex.wordEq(p.lex.peek(), "noon"):
			p.lex.next()
			recur.At = Clock{12, 0, 0}
			recur.HasAt = true
		}
	}

	if p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "except") {
		p.lex.next()
		if wd, ok := p.peekWeekday(); ok {
			p.lex.next()
			recur.ExceptDay = weekdayToISO(wd)
		} else {
			return Result{}, false
		}
	}

	if p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "until") {
		p.lex.next()
		if until, ok := p.parseExpr(true); ok {
			recur.Until = until.Time
		} else {
			return Result{}, false
		}
	}

	if p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "for") {
		p.lex.next()
		if n, ok := p.parseNumber(); ok {
			recur.Count = n
			if unit, ok := p.peekUnit(); ok && unit == recur.Every {
				p.lex.next()
			}
		} else {
			return Result{}, false
		}
	}

	next := p.nextOccurrence(&recur)
	if !recur.Until.IsZero() && next.After(recur.Until) {
		return Result{}, false
	}
	return Result{Time: next, Recur: recur, HasRecur: true, Truncated: recur.Every, Direction: Future}, true
}

func (p *parser) nextOccurrence(r *Recurrence) time.Time {
	return nextOccurrenceFrom(p.ref, r)
}

// ── time-of-day ────────────────────────────────────────────────────────────

type clockResult struct {
	hour, min, sec int
	unit           Unit
}

func (p *parser) parseAtTime() (clockResult, bool) {
	mark := p.lex.mark()
	t := p.lex.peek()
	if t.kind == tokWord && p.lex.wordEq(t, "at") {
		p.lex.next()
	} else if t.kind == tokAt {
		p.lex.next()
	} else {
		return clockResult{}, false
	}
	if p.lex.peek().kind == tokWord {
		switch {
		case p.lex.wordEq(p.lex.peek(), "midnight"):
			p.lex.next()
			return clockResult{0, 0, 0, UnitHour}, true
		case p.lex.wordEq(p.lex.peek(), "noon"):
			p.lex.next()
			return clockResult{12, 0, 0, UnitHour}, true
		}
	}
	if clk, ok := p.parseNaturalClockExpr(); ok {
		return clk, true
	}
	clk, ok := p.parseClockExpr()
	if !ok {
		p.lex.reset(mark)
		return clockResult{}, false
	}
	return clk, true
}

func (p *parser) parseNaturalClockExpr() (clockResult, bool) {
	mark := p.lex.mark()
	if p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "quarter") {
		p.lex.next()
		if p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "past") {
			p.lex.next()
			if clk, ok := p.parseClockExpr(); ok {
				clk.min = 15
				clk.unit = UnitMinute
				return clk, true
			}
		}
	}
	p.lex.reset(mark)
	if p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "half") {
		p.lex.next()
		if p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "past") {
			p.lex.next()
			if clk, ok := p.parseClockExpr(); ok {
				clk.min = 30
				clk.unit = UnitMinute
				return clk, true
			}
		}
	}
	p.lex.reset(mark)
	return clockResult{}, false
}

func (p *parser) parsePartOfDayClock() (clockResult, bool) {
	t := p.lex.peek()
	if t.kind != tokWord {
		return clockResult{}, false
	}
	switch {
	case p.lex.wordEq(t, "morning"):
		p.lex.next()
		return clockResult{9, 0, 0, UnitHour}, true
	case p.lex.wordEq(t, "afternoon"):
		p.lex.next()
		return clockResult{15, 0, 0, UnitHour}, true
	case p.lex.wordEq(t, "evening"):
		p.lex.next()
		return clockResult{18, 0, 0, UnitHour}, true
	case p.lex.wordEq(t, "night"):
		p.lex.next()
		return clockResult{20, 0, 0, UnitHour}, true
	}
	return clockResult{}, false
}

func (p *parser) parseClockExpr() (clockResult, bool) {
	t := p.lex.peek()
	if t.kind != tokNumber {
		return clockResult{}, false
	}
	h := atoi(p.lex.val(t))
	p.lex.next()

	min, sec, unit := 0, 0, UnitHour

	if p.lex.peek().kind == tokColon {
		p.lex.next()
		tm := p.lex.peek()
		if tm.kind != tokNumber {
			return clockResult{}, false
		}
		min = atoi(p.lex.val(tm))
		p.lex.next()
		unit = UnitMinute

		if p.lex.peek().kind == tokColon {
			p.lex.next()
			ts := p.lex.peek()
			if ts.kind != tokNumber {
				return clockResult{}, false
			}
			sec = atoi(p.lex.val(ts))
			p.lex.next()
			unit = UnitSecond
		}
	}

	ampm := 0
	if p.lex.peek().kind == tokWord {
		if ap := wordToAMPM(p.lex.val(p.lex.peek())); ap != 0 {
			ampm = ap
			p.lex.next()
		}
	}
	if min > 59 || sec > 59 {
		return clockResult{}, false
	}
	if ampm != 0 {
		if h < 1 || h > 12 {
			return clockResult{}, false
		}
	} else if h > 23 {
		return clockResult{}, false
	}
	if ampm == 1 && h == 12 {
		h = 0
	} else if ampm == 2 && h != 12 {
		h += 12
	}
	return clockResult{h, min, sec, unit}, true
}

// ── quantity ───────────────────────────────────────────────────────────────

func (p *parser) parseQuantity() (int, Unit, bool) {
	n, ok := p.parseNumber()
	if !ok {
		return 0, 0, false
	}
	if p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "and") {
		saved := p.lex.mark()
		p.lex.next()
		if n2, ok2 := p.parseNumber(); ok2 {
			n += n2
		} else {
			p.lex.reset(saved)
		}
	}
	t := p.lex.peek()
	if t.kind != tokWord {
		return 0, 0, false
	}
	unit, ok2 := wordToUnit(p.lex.val(t))
	if !ok2 {
		return 0, 0, false
	}
	p.lex.next()
	return n, unit, true
}

func (p *parser) parseNumber() (int, bool) {
	t := p.lex.peek()
	switch t.kind {
	case tokNumber, tokOrdinal:
		p.lex.next()
		return atoi(p.lex.val(t)), true
	case tokWord:
		if n, ok := wordToInt(p.lex.val(t)); ok {
			p.lex.next()
			return n, true
		}
	}
	return 0, false
}

func (p *parser) parseOrdinalOrNumber() (int, bool) {
	t := p.lex.peek()
	switch t.kind {
	case tokOrdinal:
		p.lex.next()
		s := p.lex.val(t)
		return atoi(s[:len(s)-2]), true
	case tokNumber:
		p.lex.next()
		return atoi(p.lex.val(t)), true
	}
	return 0, false
}

func (p *parser) peekIsQuantity() bool {
	t := p.lex.peek()
	if t.kind == tokNumber || t.kind == tokOrdinal {
		return true
	}
	if t.kind == tokWord {
		if _, ok := wordToInt(p.lex.val(t)); ok {
			return true
		}
	}
	return false
}

func (p *parser) skipToQuantity() {
	for p.lex.remaining() > 0 && !p.peekIsQuantity() {
		p.lex.next()
	}
}

func (p *parser) peekUnit() (Unit, bool) {
	t := p.lex.peek()
	if t.kind != tokWord {
		return 0, false
	}
	return wordToUnit(p.lex.val(t))
}

func (p *parser) peekWeekday() (time.Weekday, bool) {
	t := p.lex.peek()
	if t.kind != tokWord {
		return 0, false
	}
	return wordToWeekday(p.lex.val(t))
}

func (p *parser) peekMonth() (time.Month, bool) {
	t := p.lex.peek()
	if t.kind != tokWord {
		return 0, false
	}
	return wordToMonth(p.lex.val(t))
}

func (p *parser) parseAndWeekday() (time.Weekday, bool) {
	mark := p.lex.mark()
	if p.lex.peek().kind != tokWord || !p.lex.wordEq(p.lex.peek(), "and") {
		return 0, false
	}
	p.lex.next()
	if wd, ok := p.peekWeekday(); ok {
		p.lex.next()
		return wd, true
	}
	p.lex.reset(mark)
	return 0, false
}

func (p *parser) parseMonthlyDateHead() (int, bool) {
	mark := p.lex.mark()
	day, ok := p.parseOrdinalOrNumber()
	if !ok || day < 1 || day > 31 {
		p.lex.reset(mark)
		return 0, false
	}
	if p.lex.peek().kind != tokWord || !p.lex.wordEq(p.lex.peek(), "of") {
		p.lex.reset(mark)
		return 0, false
	}
	p.lex.next()
	if p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), "the") {
		p.lex.next()
	}
	if p.lex.peek().kind != tokWord || !p.lex.wordEq(p.lex.peek(), "month") {
		p.lex.reset(mark)
		return 0, false
	}
	p.lex.next()
	return day, true
}

func (p *parser) parseOrdinalModifier() (int8, bool) {
	t := p.lex.peek()
	if t.kind != tokWord {
		return 0, false
	}
	switch {
	case p.lex.wordEq(t, "first"):
		p.lex.next()
		return 1, true
	case p.lex.wordEq(t, "second"):
		p.lex.next()
		return 2, true
	case p.lex.wordEq(t, "third"):
		p.lex.next()
		return 3, true
	case p.lex.wordEq(t, "fourth"):
		p.lex.next()
		return 4, true
	case p.lex.wordEq(t, "last"):
		p.lex.next()
		return -1, true
	}
	return 0, false
}

func (p *parser) skipWords(words ...string) {
	for _, w := range words {
		if p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), w) {
			p.lex.next()
		}
	}
}

func (p *parser) skipCasualWords() {
	if p.mode == ModeStrict {
		return
	}
	for p.lex.peek().kind == tokWord {
		t := p.lex.peek()
		if p.lex.wordEq(t, "please") || p.lex.wordEq(t, "around") || p.lex.wordEq(t, "approximately") ||
			p.lex.wordEq(t, "about") || p.lex.wordEq(t, "sometime") || p.lex.wordEq(t, "on") {
			p.lex.next()
			continue
		}
		return
	}
}

func (p *parser) withTokenMatch(r Result, startToken int) Result {
	start := 0
	end := len(p.src)
	if startToken >= 0 && startToken < p.lex.n {
		start = p.lex.tokens[startToken].lo
	}
	if p.lex.pos > 0 && p.lex.pos-1 < p.lex.n {
		end = p.lex.tokens[p.lex.pos-1].hi
	}
	if r.Confidence == 0 && p.mode != ModeStrict {
		r.Confidence = 0.8
		if p.mode == ModeFuzzy {
			r.Confidence = 0.6
		}
	}
	return withMatch(r, p.src, start, end)
}

func (l *lexer) atEnd() bool {
	for l.pos < l.n {
		switch l.tokens[l.pos].kind {
		case tokComma:
			l.pos++
		default:
			return false
		}
	}
	return true
}

// ── calendar helpers ───────────────────────────────────────────────────────

func startOfDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

func startOfMonth(t time.Time) time.Time {
	y, m, _ := t.Date()
	return time.Date(y, m, 1, 0, 0, 0, 0, t.Location())
}

func startOfQuarter(t time.Time) time.Time {
	qMonth := time.Month(((int(t.Month())-1)/3)*3 + 1)
	return time.Date(t.Year(), qMonth, 1, 0, 0, 0, 0, t.Location())
}

func quarterStart(year, quarter int, loc *time.Location) time.Time {
	return time.Date(year, time.Month((quarter-1)*3+1), 1, 0, 0, 0, 0, loc)
}

func startOfYear(t time.Time) time.Time {
	return time.Date(t.Year(), time.January, 1, 0, 0, 0, 0, t.Location())
}

func validMonthDay(year int, month time.Month, day int) bool {
	if month < time.January || month > time.December || day < 1 {
		return false
	}
	t := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	return t.Year() == year && t.Month() == month && t.Day() == day
}

func validMonthDayInRange(startYear, endYear int, month time.Month, day int) bool {
	for y := startYear; y <= endYear; y++ {
		if validMonthDay(y, month, day) {
			return true
		}
	}
	return false
}

func nextMonthDay(ref time.Time, month time.Month, day int, clk clockResult, hasClock bool) (time.Time, bool) {
	for y := ref.Year(); y <= ref.Year()+8; y++ {
		if !validMonthDay(y, month, day) {
			continue
		}
		base := time.Date(y, month, day, 0, 0, 0, 0, ref.Location())
		if hasClock {
			base = setClock(base, clk)
			if !base.Before(ref) {
				return base, true
			}
			continue
		}
		if base.After(ref) || sameDate(base, ref) {
			return base, true
		}
	}
	return time.Time{}, false
}

func nearestWeekday(ref time.Time, wd time.Weekday, dir Direction) time.Time {
	cur := ref.Weekday()
	if dir == Past {
		diff := int(cur) - int(wd)
		if diff <= 0 {
			diff += 7
		}
		return ref.AddDate(0, 0, -diff)
	}
	diff := int(wd) - int(cur)
	if diff <= 0 {
		diff += 7
	}
	return ref.AddDate(0, 0, diff)
}

func weekdayToISO(wd time.Weekday) int8 {
	if wd == time.Sunday {
		return 7
	}
	return int8(wd)
}

func isoToWeekday(d int8) time.Weekday {
	if d == 7 {
		return time.Sunday
	}
	return time.Weekday(d)
}

func nextOccurrenceFrom(ref time.Time, r *Recurrence) time.Time {
	if r.Interval < 1 {
		return ref
	}
	switch r.Every {
	case UnitSecond:
		return ref.Add(time.Duration(r.Interval) * time.Second)
	case UnitMinute:
		return ref.Add(time.Duration(r.Interval) * time.Minute)
	case UnitHour:
		return ref.Add(time.Duration(r.Interval) * time.Hour)
	case UnitDay:
		if r.OnWeekday {
			return nextWeekdayOccurrence(ref, r)
		}
		t := applyClock(startOfDay(ref), r.At, r.HasAt)
		if t.After(ref) {
			return t
		}
		return applyClock(startOfDay(ref).AddDate(0, 0, r.Interval), r.At, r.HasAt)
	case UnitWeek:
		if r.OnDay != 0 || r.OnWeekday {
			return nextWeeklyOccurrence(ref, r)
		}
		t := applyClock(startOfDay(ref), r.At, r.HasAt)
		if t.After(ref) {
			return t
		}
		return applyClock(startOfDay(ref).AddDate(0, 0, r.Interval*7), r.At, r.HasAt)
	case UnitMonth:
		if r.OnDate != 0 {
			return nextMonthlyDateOccurrence(ref, r)
		}
		if r.OnDay != 0 && r.OnOrdinal != 0 {
			return nextMonthlyOrdinalOccurrence(ref, r)
		}
		t := startOfMonth(ref).AddDate(0, r.Interval, 0)
		if r.OnDay != 0 {
			t = nearestWeekday(t, isoToWeekday(r.OnDay), Future)
		}
		return applyClock(t, r.At, r.HasAt)
	case UnitYear:
		if r.OnMonth != 0 || r.OnDate != 0 {
			return nextYearlyOccurrence(ref, r)
		}
		t := applyClock(startOfYear(ref), r.At, r.HasAt)
		if t.After(ref) {
			return t
		}
		return applyClock(startOfYear(ref).AddDate(r.Interval, 0, 0), r.At, r.HasAt)
	case UnitQuarter:
		if r.OnDate != 0 {
			return nextQuarterlyDateOccurrence(ref, r)
		}
		t := applyClock(startOfQuarter(ref), r.At, r.HasAt)
		if t.After(ref) {
			return t
		}
		return applyClock(startOfQuarter(ref).AddDate(0, r.Interval*3, 0), r.At, r.HasAt)
	}
	return ref
}

func nextWeeklyOccurrence(ref time.Time, r *Recurrence) time.Time {
	if r.OnWeekday {
		return nextWeekdayOccurrence(ref, r)
	}
	if r.AlsoOnDay != 0 {
		a := *r
		a.AlsoOnDay = 0
		first := nextWeeklyOccurrence(ref, &a)
		b := *r
		b.OnDay = r.AlsoOnDay
		b.AlsoOnDay = 0
		second := nextWeeklyOccurrence(ref, &b)
		if second.Before(first) {
			return second
		}
		return first
	}
	wd := isoToWeekday(r.OnDay)
	base := startOfDay(ref)
	diff := int(wd) - int(base.Weekday())
	if diff < 0 {
		diff += 7
	}
	t := applyClock(base.AddDate(0, 0, diff), r.At, r.HasAt)
	if t.After(ref) {
		return t
	}
	return applyClock(base.AddDate(0, 0, diff+r.Interval*7), r.At, r.HasAt)
}

func nextWeekdayOccurrence(ref time.Time, r *Recurrence) time.Time {
	base := startOfDay(ref)
	for i := 0; i < 14; i++ {
		t := applyClock(base.AddDate(0, 0, i), r.At, r.HasAt)
		if t.After(ref) && t.Weekday() != time.Saturday && t.Weekday() != time.Sunday && weekdayToISO(t.Weekday()) != r.ExceptDay {
			return t
		}
	}
	return ref
}

func nextMonthlyDateOccurrence(ref time.Time, r *Recurrence) time.Time {
	for monthOffset := 0; monthOffset <= r.Interval*120; monthOffset += r.Interval {
		t := startOfMonth(ref).AddDate(0, monthOffset, 0)
		day := int(r.OnDate)
		if r.OnDate == -1 {
			day = daysInMonth(t.Year(), t.Month())
		}
		if !validMonthDay(t.Year(), t.Month(), day) {
			continue
		}
		t = time.Date(t.Year(), t.Month(), day, 0, 0, 0, 0, ref.Location())
		t = applyClock(t, r.At, r.HasAt)
		if t.After(ref) {
			return t
		}
	}
	return ref
}

func nextQuarterlyDateOccurrence(ref time.Time, r *Recurrence) time.Time {
	for monthOffset := 0; monthOffset <= r.Interval*120; monthOffset += r.Interval * 3 {
		q := startOfQuarter(ref).AddDate(0, monthOffset, 0)
		day := int(r.OnDate)
		if r.OnDate == -1 {
			day = daysInMonth(q.Year(), q.Month())
		}
		if !validMonthDay(q.Year(), q.Month(), day) {
			continue
		}
		t := time.Date(q.Year(), q.Month(), day, 0, 0, 0, 0, ref.Location())
		t = applyClock(t, r.At, r.HasAt)
		if t.After(ref) {
			return t
		}
	}
	return ref
}

func nextMonthlyOrdinalOccurrence(ref time.Time, r *Recurrence) time.Time {
	for monthOffset := 0; monthOffset <= r.Interval*120; monthOffset += r.Interval {
		t := startOfMonth(ref).AddDate(0, monthOffset, 0)
		t = nthWeekdayOfMonth(t, r.OnDay, r.OnOrdinal)
		t = applyClock(t, r.At, r.HasAt)
		if t.After(ref) {
			return t
		}
	}
	return ref
}

// nthWeekdayOfMonth returns the nth occurrence of a weekday in the month.
// onOrd: 1=first, 2=second, 3=third, 4=fourth, -1=last
func nthWeekdayOfMonth(t time.Time, onDay int8, onOrd int8) time.Time {
	month := t.Month()
	year := t.Year()
	weekday := isoToWeekday(onDay)

	// Find the first day of the month
	firstDay := time.Date(year, month, 1, 0, 0, 0, 0, t.Location())
	firstWeekday := int(firstDay.Weekday())
	targetWeekday := int(weekday)

	// Calculate days to add to get to the first occurrence
	daysToAdd := (targetWeekday - firstWeekday + 7) % 7

	if onOrd == -1 {
		// Last occurrence: start from the last day of the month and work backward
		lastDay := time.Date(year, month+1, 0, 0, 0, 0, 0, t.Location())
		lastWeekday := int(lastDay.Weekday())
		daysBack := (lastWeekday - targetWeekday + 7) % 7
		return lastDay.AddDate(0, 0, -daysBack)
	}

	// For positive ordinals (first, second, etc.)
	day := 1 + daysToAdd + (int(onOrd)-1)*7
	result := time.Date(year, month, day, 0, 0, 0, 0, t.Location())

	// Handle case where the calculated day overflows into next month
	if result.Month() != month {
		// Try the previous week
		return time.Date(year, month, day-7, 0, 0, 0, 0, t.Location())
	}
	return result
}

func nextYearlyOccurrence(ref time.Time, r *Recurrence) time.Time {
	month := time.Month(r.OnMonth)
	if month == 0 {
		month = time.January
	}
	day := int(r.OnDate)
	if day == 0 {
		day = 1
	}
	for y := ref.Year(); y <= ref.Year()+r.Interval*32; y += r.Interval {
		if r.OnDate == -1 {
			day = daysInMonth(y, month)
		}
		if !validMonthDay(y, month, day) {
			continue
		}
		t := time.Date(y, month, day, 0, 0, 0, 0, ref.Location())
		t = applyClock(t, r.At, r.HasAt)
		if t.After(ref) {
			return t
		}
	}
	return ref
}

func setClock(base time.Time, c clockResult) time.Time {
	return time.Date(base.Year(), base.Month(), base.Day(), c.hour, c.min, c.sec, 0, base.Location())
}

func applyClock(base time.Time, clk Clock, has bool) time.Time {
	if !has {
		return base
	}
	return time.Date(base.Year(), base.Month(), base.Day(), clk.Hour, clk.Min, clk.Sec, 0, base.Location())
}

func directionFromCompare(t, ref time.Time) Direction {
	switch {
	case t.Before(ref):
		return Past
	case t.After(ref):
		return Future
	default:
		return Present
	}
}

func directionForDateResult(t, ref time.Time, hasClock bool) Direction {
	if hasClock {
		return directionFromCompare(t, ref)
	}
	if sameDate(t, ref) {
		return Present
	}
	return directionFromCompare(t, ref)
}

func sameDate(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.In(a.Location()).Date()
	return ay == by && am == bm && ad == bd
}

func periodStart(t time.Time, unit Unit) time.Time {
	switch unit {
	case UnitDay:
		return startOfDay(t)
	case UnitWeek:
		return startOfWeek(t)
	case UnitMonth:
		return startOfMonth(t)
	case UnitQuarter:
		return startOfQuarter(t)
	case UnitYear:
		return startOfYear(t)
	default:
		return t
	}
}

func fiscalPeriodStart(t time.Time, unit Unit, dir Direction, startMonth time.Month) time.Time {
	if startMonth < time.January || startMonth > time.December {
		startMonth = time.January
	}
	fyStartYear := t.Year()
	if t.Month() < startMonth {
		fyStartYear--
	}
	fyStart := time.Date(fyStartYear, startMonth, 1, 0, 0, 0, 0, t.Location())
	if unit == UnitYear {
		switch dir {
		case Past:
			return fyStart.AddDate(-1, 0, 0)
		case Future:
			return fyStart.AddDate(1, 0, 0)
		default:
			return fyStart
		}
	}
	monthsSinceStart := (int(t.Month()) - int(startMonth) + 12) % 12
	currentQuarter := monthsSinceStart / 3
	qStart := fyStart.AddDate(0, currentQuarter*3, 0)
	switch dir {
	case Past:
		return qStart.AddDate(0, -3, 0)
	case Future:
		return qStart.AddDate(0, 3, 0)
	default:
		return qStart
	}
}

func fiscalPeriodEnd(start time.Time, unit Unit) time.Time {
	if unit == UnitYear {
		return start.AddDate(1, 0, 0).Add(-time.Nanosecond)
	}
	if unit == UnitQuarter {
		return start.AddDate(0, 3, 0).Add(-time.Nanosecond)
	}
	return periodEnd(start, unit)
}

func periodEnd(t time.Time, unit Unit) time.Time {
	switch unit {
	case UnitDay:
		return endOfDay(t)
	case UnitWeek:
		return endOfWeek(t)
	case UnitMonth:
		return endOfMonth(t)
	case UnitQuarter:
		return endOfQuarter(t)
	case UnitYear:
		return endOfYear(t)
	default:
		return t
	}
}

func shiftByUnit(ref time.Time, unit Unit, dir Direction) time.Time {
	n := 1
	if dir == Past {
		n = -1
	}
	// For calendar units, truncate to the start of the target period.
	switch unit {
	case UnitSecond:
		return ref.Add(time.Duration(n) * time.Second)
	case UnitMinute:
		return ref.Add(time.Duration(n) * time.Minute)
	case UnitHour:
		return ref.Add(time.Duration(n) * time.Hour)
	case UnitDay:
		return startOfDay(ref).AddDate(0, 0, n)
	case UnitWeek:
		return startOfDay(ref).AddDate(0, 0, n*7)
	case UnitMonth:
		return startOfMonth(ref).AddDate(0, n, 0)
	case UnitYear:
		return startOfYear(ref).AddDate(n, 0, 0)
	case UnitQuarter:
		return startOfQuarter(ref).AddDate(0, n*3, 0)
	}
	return ref
}

func shiftByUnitN(ref time.Time, unit Unit, n int) time.Time {
	switch unit {
	case UnitSecond:
		return ref.Add(time.Duration(n) * time.Second)
	case UnitMinute:
		return ref.Add(time.Duration(n) * time.Minute)
	case UnitHour:
		return ref.Add(time.Duration(n) * time.Hour)
	case UnitDay:
		return ref.AddDate(0, 0, n)
	case UnitWeek:
		return ref.AddDate(0, 0, n*7)
	case UnitMonth:
		return ref.AddDate(0, n, 0)
	case UnitYear:
		return ref.AddDate(n, 0, 0)
	case UnitQuarter:
		return ref.AddDate(0, n*3, 0)
	}
	return ref
}

func atoi(s string) int {
	n := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			break
		}
		n = n*10 + int(c-'0')
	}
	return n
}

func startOfWeek(t time.Time) time.Time {
	daysBack := int(t.Weekday())
	return startOfDay(t).AddDate(0, 0, -daysBack)
}

func daysInMonth(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

func addWeekdays(t time.Time, n int) time.Time {
	if n == 0 {
		return t
	}
	current := t
	direction := 1
	if n < 0 {
		direction = -1
		n = -n
	}
	for steps := 0; steps < n; {
		current = current.AddDate(0, 0, direction)
		if current.Weekday() != time.Saturday && current.Weekday() != time.Sunday {
			steps++
		}
	}
	return current
}

func addBusinessWeeks(t time.Time, n int, holidays []time.Time, weekends []time.Weekday) time.Time {
	return addBusinessDays(t, n*5, holidays, weekends)
}

func nthWeekdayFrom(ref time.Time, wd time.Weekday, n int, dir Direction) time.Time {
	if n < 1 {
		n = 1
	}
	current := ref
	step := 1
	if dir == Past {
		step = -1
	}
	for seen := 0; seen < n; {
		current = current.AddDate(0, 0, step)
		if current.Weekday() == wd {
			seen++
		}
	}
	return startOfDay(current)
}

func weekendStart(ref time.Time, dir Direction) time.Time {
	base := startOfDay(ref)
	switch dir {
	case Past:
		days := (int(base.Weekday()) - int(time.Saturday) + 7) % 7
		if days == 0 {
			days = 7
		}
		return base.AddDate(0, 0, -days)
	case Present:
		days := (int(time.Saturday) - int(base.Weekday()) + 7) % 7
		return base.AddDate(0, 0, days)
	default:
		days := (int(time.Saturday) - int(base.Weekday()) + 7) % 7
		if days == 0 {
			days = 7
		}
		return base.AddDate(0, 0, days)
	}
}

func endOfWeek(t time.Time) time.Time {
	// Saturday is the last day of the week (weekday 6)
	daysForward := (6 - int(t.Weekday()) + 7) % 7
	return startOfDay(t).AddDate(0, 0, daysForward).Add(24*time.Hour - time.Nanosecond)
}

func endOfDay(t time.Time) time.Time {
	return startOfDay(t).Add(24*time.Hour - time.Nanosecond)
}

func endOfMonth(t time.Time) time.Time {
	return startOfMonth(t).AddDate(0, 1, 0).Add(-time.Nanosecond)
}

func endOfQuarter(t time.Time) time.Time {
	return startOfQuarter(t).AddDate(0, 3, 0).Add(-time.Nanosecond)
}

func endOfYear(t time.Time) time.Time {
	return startOfYear(t).AddDate(1, 0, 0).Add(-time.Nanosecond)
}

// addBusinessDays adds n business days to t, skipping weekends and holidays.
// n can be positive (future) or negative (past).
func addBusinessDays(t time.Time, n int, holidays []time.Time, weekends []time.Weekday) time.Time {
	if n == 0 {
		return t
	}
	current := t
	direction := 1
	if n < 0 {
		direction = -1
		n = -n
	}

	for steps := 0; steps < n; {
		current = current.AddDate(0, 0, direction)
		if isBusinessDate(current, holidays, weekends) {
			steps++
		}
	}

	return current
}

func isBusinessDate(t time.Time, holidays []time.Time, weekends []time.Weekday) bool {
	weekday := t.Weekday()
	if len(weekends) == 0 {
		if weekday == time.Saturday || weekday == time.Sunday {
			return false
		}
	} else {
		for _, weekend := range weekends {
			if weekday == weekend {
				return false
			}
		}
	}
	for _, holiday := range holidays {
		if sameDate(t, holiday) {
			return false
		}
	}
	return true
}
