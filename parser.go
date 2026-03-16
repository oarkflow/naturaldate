package naturaldate

import "time"

// parser holds the mutable state for one parse attempt.
// All state is on the stack or in the fixed-size arrays of lexer/token.
type parser struct {
	src  string
	ref  time.Time
	wdir Direction
	lex  lexer
}

func (p *parser) parse(embedded bool) (Result, bool) {
	p.lex.init(p.src)
	if r, ok := p.parseExpr(); ok {
		return r, true
	}
	if embedded {
		for start := 1; start < p.lex.n; start++ {
			p.lex.reset(start)
			if r, ok := p.parseExpr(); ok {
				return r, true
			}
		}
	}
	return Result{}, false
}

func (p *parser) parseExpr() (Result, bool) {
	mark := p.lex.mark()

	if r, ok := p.parseRecurring(); ok {
		return r, true
	}
	p.lex.reset(mark)

	if r, ok := p.parseRelative(); ok {
		return r, true
	}
	p.lex.reset(mark)

	if r, ok := p.parseAnchor(); ok {
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
	case p.lex.wordEq(t, "tomorrow"):
		p.lex.next()
		base = startOfDay(p.ref).AddDate(0, 0, 1)
	case p.lex.wordEq(t, "midnight"):
		p.lex.next()
		return Result{Time: startOfDay(p.ref), Truncated: UnitHour, Direction: Present}, true
	case p.lex.wordEq(t, "noon"):
		p.lex.next()
		return Result{Time: startOfDay(p.ref).Add(12 * time.Hour), Truncated: UnitHour, Direction: Present}, true
	default:
		return Result{}, false
	}
	if clk, ok := p.parseAtTime(); ok {
		base = setClock(base, clk)
		unit = clk.unit
	}
	return Result{Time: base, Truncated: unit, Direction: Present}, true
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

	t2 := p.lex.peek()
	if t2.kind != tokWord {
		return Result{}, false
	}
	v2 := p.lex.val(t2)

	if wd, ok := wordToWeekday(v2); ok {
		p.lex.next()
		base := startOfDay(nearestWeekday(p.ref, wd, dir))
		if clk, ok := p.parseAtTime(); ok {
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
	return Result{Time: base, Truncated: UnitDay, Direction: dir}, true
}

// ── relative ───────────────────────────────────────────────────────────────

func (p *parser) parseRelative() (Result, bool) {
	mark := p.lex.mark()
	p.skipToQuantity()
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
			p.lex.next()
			if p.lex.peek().kind == tokWord && (p.lex.wordEq(p.lex.peek(), "now") || p.lex.wordEq(p.lex.peek(), "today")) {
				p.lex.next()
			}
		}
	}

	mul := 1
	if dir == Past {
		mul = -1
	}
	return Result{Time: shiftByUnitN(p.ref, unit, mul*n), Truncated: unit, Direction: dir}, true
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
	y := p.ref.Year()
	base := time.Date(y, mon, day, 0, 0, 0, 0, p.ref.Location())
	if !base.After(p.ref) {
		base = time.Date(y+1, mon, day, 0, 0, 0, 0, p.ref.Location())
	}
	if clk, ok := p.parseAtTime(); ok {
		base = setClock(base, clk)
	}
	return Result{Time: base, Truncated: UnitDay, Direction: Future}, true
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

	y := p.ref.Year()
	base := time.Date(y, mon, day, 0, 0, 0, 0, p.ref.Location())
	if !base.After(p.ref) {
		base = time.Date(y+1, mon, day, 0, 0, 0, 0, p.ref.Location())
	}
	if clk, ok := p.parseAtTime(); ok {
		base = setClock(base, clk)
	}
	return Result{Time: base, Truncated: UnitDay, Direction: Future}, true
}

// ── bare time ──────────────────────────────────────────────────────────────

func (p *parser) parseBareTime() (Result, bool) {
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
		m, d, ok := resolveMonthDay(v0, v1, sep.kind)
		if !ok {
			return Result{}, false
		}
		y := p.ref.Year()
		base := time.Date(y, time.Month(m), d, 0, 0, 0, 0, p.ref.Location())
		if !base.After(p.ref) {
			base = time.Date(y+1, time.Month(m), d, 0, 0, 0, 0, p.ref.Location())
		}
		if clk, ok := p.parseAtTime(); ok {
			base = setClock(base, clk)
		}
		return Result{Time: base, Truncated: UnitDay, Direction: Future}, true
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
	case v0 > 31:
		y, m, d = v0, v1, v2
	default:
		var ok bool
		m, d, ok = resolveMonthDay(v0, v1, sep.kind)
		if !ok {
			return Result{}, false
		}
		y = v2
	}
	if y < 100 {
		y += 2000
	}
	if m < 1 || m > 12 || d < 1 || d > 31 {
		return Result{}, false
	}
	base := time.Date(y, time.Month(m), d, 0, 0, 0, 0, p.ref.Location())
	if clk, ok := p.parseAtTime(); ok {
		base = setClock(base, clk)
	}
	return Result{Time: base, Truncated: UnitDay, Direction: Future}, true
}

func resolveMonthDay(v0, v1 int, sep tokenKind) (m, d int, ok bool) {
	if v0 < 1 || v0 > 31 || v1 < 1 || v1 > 31 {
		return 0, 0, false
	}
	if v0 > 12 && v1 > 12 {
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

	if p.peekIsQuantity() {
		if n, unit, ok := p.parseQuantity(); ok {
			recur.Every = unit
			recur.Interval = n
		} else {
			return Result{}, false
		}
	} else if unit, ok := p.peekUnit(); ok {
		p.lex.next()
		recur.Every = unit
		recur.Interval = 1
	} else if wd, ok := p.peekWeekday(); ok {
		p.lex.next()
		recur.Every = UnitWeek
		recur.Interval = 1
		recur.OnDay = int8(wd)
	} else {
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
		} else if day, ok := p.parseOrdinalOrNumber(); ok {
			recur.OnDate = int8(day)
		}
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

	next := p.nextOccurrence(&recur)
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
	clk, ok := p.parseClockExpr()
	if !ok {
		p.lex.reset(mark)
		return clockResult{}, false
	}
	return clk, true
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
	if h > 23 || min > 59 || sec > 59 {
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

func (p *parser) skipWords(words ...string) {
	for _, w := range words {
		if p.lex.peek().kind == tokWord && p.lex.wordEq(p.lex.peek(), w) {
			p.lex.next()
		}
	}
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

func startOfYear(t time.Time) time.Time {
	return time.Date(t.Year(), time.January, 1, 0, 0, 0, 0, t.Location())
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
	switch r.Every {
	case UnitSecond:
		return ref.Add(time.Duration(r.Interval) * time.Second)
	case UnitMinute:
		return ref.Add(time.Duration(r.Interval) * time.Minute)
	case UnitHour:
		return ref.Add(time.Duration(r.Interval) * time.Hour)
	case UnitDay:
		t := startOfDay(ref).AddDate(0, 0, r.Interval)
		return applyClock(t, r.At, r.HasAt)
	case UnitWeek:
		t := startOfDay(ref).AddDate(0, 0, r.Interval*7)
		if r.OnDay != 0 {
			t = nearestWeekday(t, isoToWeekday(r.OnDay), Future)
		}
		return applyClock(t, r.At, r.HasAt)
	case UnitMonth:
		t := startOfMonth(ref).AddDate(0, r.Interval, 0)
		if r.OnDate != 0 {
			t = time.Date(t.Year(), t.Month(), int(r.OnDate), 0, 0, 0, 0, ref.Location())
		} else if r.OnDay != 0 {
			t = nearestWeekday(t, isoToWeekday(r.OnDay), Future)
		}
		return applyClock(t, r.At, r.HasAt)
	case UnitYear:
		t := startOfYear(ref).AddDate(r.Interval, 0, 0)
		return applyClock(t, r.At, r.HasAt)
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
