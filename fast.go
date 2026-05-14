package naturaldate

import "time"

// fastParse handles common single-token inputs without lexing.
func fastParse(s string, opts Options) (Result, bool) {
	s = trimSpaceASCII(s)
	if len(s) == 0 {
		return Result{}, false
	}
	hasSpace := false
	for i := 0; i < len(s); i++ {
		if s[i] <= ' ' {
			hasSpace = true
			break
		}
	}
	if hasSpace {
		if startsWithWord(s, "every") || startsWithWord(s, "once") {
			if r, ok := fastParseRecurring(s, opts); ok {
				return r, true
			}
		}
		if opts.AllowEmbedded {
			if r, ok := fastParseEmbedded(s, opts); ok {
				return r, true
			}
		}
		return Result{}, false
	}

	ref := opts.Reference
	switch {
	case eqLower(s, "now"):
		return Result{Time: ref, Truncated: UnitSecond, Direction: Present}, true
	case eqLower(s, "today"):
		return Result{Time: startOfDay(ref), Truncated: UnitDay, Direction: Present}, true
	case eqLower(s, "yesterday"):
		return Result{Time: startOfDay(ref).AddDate(0, 0, -1), Truncated: UnitDay, Direction: Past}, true
	case eqLower(s, "tomorrow"):
		return Result{Time: startOfDay(ref).AddDate(0, 0, 1), Truncated: UnitDay, Direction: Future}, true
	case eqLower(s, "midnight"):
		t := startOfDay(ref)
		return Result{Time: t, Truncated: UnitHour, Direction: directionFromCompare(t, ref)}, true
	case eqLower(s, "noon"):
		t := startOfDay(ref).Add(12 * time.Hour)
		return Result{Time: t, Truncated: UnitHour, Direction: directionFromCompare(t, ref)}, true
	}

	if isDigit(s[0]) {
		h, m, sec, unit, ok := parseClockFast(s)
		if ok {
			base := time.Date(ref.Year(), ref.Month(), ref.Day(), h, m, sec, 0, ref.Location())
			if !base.After(ref) {
				base = base.AddDate(0, 0, 1)
			}
			return Result{Time: base, Truncated: unit, Direction: Future}, true
		}
	}

	return Result{}, false
}

func startsWithWord(s, word string) bool {
	if len(s) < len(word) {
		return false
	}
	if !eqLower(s[:len(word)], word) {
		return false
	}
	if len(s) == len(word) {
		return true
	}
	return !isAlpha(s[len(word)])
}

type fastScan struct {
	s string
	i int
	n int
}

func (sc *fastScan) nextToken() (tokenKind, int, int) {
	for sc.i < sc.n {
		c := sc.s[sc.i]
		if isAlpha(c) || isDigit(c) || c == '@' {
			break
		}
		sc.i++
	}
	if sc.i >= sc.n {
		return tokEOF, sc.n, sc.n
	}
	c := sc.s[sc.i]
	if c == '@' {
		sc.i++
		return tokAt, sc.i - 1, sc.i
	}
	if isAlpha(c) {
		start := sc.i
		for sc.i < sc.n && isAlpha(sc.s[sc.i]) {
			sc.i++
		}
		return tokWord, start, sc.i
	}
	if isDigit(c) {
		start := sc.i
		for sc.i < sc.n && isDigit(sc.s[sc.i]) {
			sc.i++
		}
		kind := tokNumber
		if sc.i+1 < sc.n && isAlpha(sc.s[sc.i]) && isAlpha(sc.s[sc.i+1]) &&
			isOrdinalSuffix(sc.s[sc.i], sc.s[sc.i+1]) {
			sc.i += 2
			kind = tokOrdinal
		}
		return kind, start, sc.i
	}
	sc.i++
	return tokEOF, sc.i, sc.i
}

func fastParseRecurring(s string, opts Options) (Result, bool) {
	if r, ok := fastParseRecurringWords(s, opts); ok {
		return r, true
	}
	sc := fastScan{s: s, n: len(s)}
	kind, lo, hi := sc.nextToken()
	if kind != tokWord {
		return Result{}, false
	}
	once := false
	if eqLower(s[lo:hi], "every") {
		once = false
	} else if eqLower(s[lo:hi], "once") {
		once = true
	} else {
		return Result{}, false
	}

	if once {
		mark := sc.i
		k, l, h := sc.nextToken()
		if k != tokWord || !(eqLower(s[l:h], "a") || eqLower(s[l:h], "an")) {
			sc.i = mark
		}
	}

	var recur Recurrence
	kind, lo, hi = sc.nextToken()
	if kind == tokNumber || kind == tokOrdinal {
		n := atoi(s[lo:hi])
		k2, l2, h2 := sc.nextToken()
		if k2 != tokWord {
			return Result{}, false
		}
		unit, ok := wordToUnit(s[l2:h2])
		if !ok {
			return Result{}, false
		}
		recur.Every = unit
		recur.Interval = n
	} else if kind == tokWord {
		if unit, ok := wordToUnit(s[lo:hi]); ok {
			recur.Every = unit
			recur.Interval = 1
		} else if wd, ok := wordToWeekday(s[lo:hi]); ok {
			recur.Every = UnitWeek
			recur.Interval = 1
			recur.OnDay = weekdayToISO(wd)
		} else if n, ok := wordToInt(s[lo:hi]); ok {
			k2, l2, h2 := sc.nextToken()
			if k2 != tokWord {
				return Result{}, false
			}
			unit, ok := wordToUnit(s[l2:h2])
			if !ok {
				return Result{}, false
			}
			recur.Every = unit
			recur.Interval = n
		} else {
			return Result{}, false
		}
	} else {
		return Result{}, false
	}
	if recur.Interval < 1 {
		return Result{}, false
	}

	// optional "on [the] <weekday|ordinal>"
	mark := sc.i
	kind, lo, hi = sc.nextToken()
	if kind == tokWord && eqLower(s[lo:hi], "on") {
		mark2 := sc.i
		k2, l2, h2 := sc.nextToken()
		if k2 != tokWord || !eqLower(s[l2:h2], "the") {
			sc.i = mark2
		}
		k3, l3, h3 := sc.nextToken()
		switch k3 {
		case tokWord:
			if wd, ok := wordToWeekday(s[l3:h3]); ok {
				recur.OnDay = weekdayToISO(wd)
			} else if n, ok := wordToInt(s[l3:h3]); ok {
				if n < 1 || n > 31 {
					return Result{}, false
				}
				recur.OnDate = int8(n)
			} else {
				return Result{}, false
			}
		case tokNumber, tokOrdinal:
			day := atoi(s[l3:h3])
			if day < 1 || day > 31 {
				return Result{}, false
			}
			recur.OnDate = int8(day)
		default:
			return Result{}, false
		}
	} else {
		sc.i = mark
	}

	// optional time
	mark = sc.i
	kind, lo, hi = sc.nextToken()
	if kind == tokWord && eqLower(s[lo:hi], "at") || kind == tokAt {
		h, m, sec, unit, ok := scanClockAfterAt(&sc)
		if !ok {
			return Result{}, false
		}
		_ = unit
		recur.At = Clock{Hour: h, Min: m, Sec: sec}
		recur.HasAt = true
	} else if kind == tokWord && eqLower(s[lo:hi], "midnight") {
		recur.At = Clock{Hour: 0, Min: 0, Sec: 0}
		recur.HasAt = true
	} else if kind == tokWord && eqLower(s[lo:hi], "noon") {
		recur.At = Clock{Hour: 12, Min: 0, Sec: 0}
		recur.HasAt = true
	} else {
		sc.i = mark
	}

	if !sc.atEnd() {
		return Result{}, false
	}

	next := nextOccurrenceFrom(opts.Reference, &recur)
	return Result{Time: next, Recur: recur, HasRecur: true, Truncated: recur.Every, Direction: Future}, true
}

type wordSpan struct {
	lo int
	hi int
}

func fastParseRecurringWords(s string, opts Options) (Result, bool) {
	var words [8]wordSpan
	nw, ok := scanWordsAlphaOnly(s, &words)
	if !ok || nw < 2 {
		return Result{}, false
	}

	i := 0
	if eqLower(s[words[0].lo:words[0].hi], "every") {
		i = 1
	} else if eqLower(s[words[0].lo:words[0].hi], "once") {
		i = 1
		if i < nw {
			w := s[words[i].lo:words[i].hi]
			if eqLower(w, "a") || eqLower(w, "an") {
				i++
			}
		}
	} else {
		return Result{}, false
	}

	if i >= nw {
		return Result{}, false
	}
	unit, ok := wordToUnit(s[words[i].lo:words[i].hi])
	if !ok {
		return Result{}, false
	}
	i++

	var recur Recurrence
	recur.Every = unit
	recur.Interval = 1

	if i < nw && eqLower(s[words[i].lo:words[i].hi], "on") {
		i++
		if i < nw && eqLower(s[words[i].lo:words[i].hi], "the") {
			i++
		}
		if i < nw {
			if wd, ok := wordToWeekday(s[words[i].lo:words[i].hi]); ok {
				recur.OnDay = weekdayToISO(wd)
				i++
			} else {
				return Result{}, false
			}
		}
	}

	if i < nw {
		w := s[words[i].lo:words[i].hi]
		if eqLower(w, "midnight") {
			recur.At = Clock{Hour: 0, Min: 0, Sec: 0}
			recur.HasAt = true
			i++
		} else if eqLower(w, "noon") {
			recur.At = Clock{Hour: 12, Min: 0, Sec: 0}
			recur.HasAt = true
			i++
		}
	}
	if i != nw {
		return Result{}, false
	}

	next := nextOccurrenceFrom(opts.Reference, &recur)
	return Result{Time: next, Recur: recur, HasRecur: true, Truncated: recur.Every, Direction: Future}, true
}

func scanWordsAlphaOnly(s string, out *[8]wordSpan) (int, bool) {
	n := 0
	i := 0
	for i < len(s) {
		c := s[i]
		if isDigit(c) {
			return 0, false
		}
		if !isAlpha(c) {
			i++
			continue
		}
		start := i
		i++
		for i < len(s) && isAlpha(s[i]) {
			i++
		}
		if n >= len(out) {
			return 0, false
		}
		out[n] = wordSpan{lo: start, hi: i}
		n++
	}
	return n, true
}

func (sc *fastScan) atEnd() bool {
	kind, _, _ := sc.nextToken()
	return kind == tokEOF
}

func fastParseEmbedded(s string, opts Options) (Result, bool) {
	if r, ok := fastParseEmbeddedInDigits(s, opts); ok {
		return r, true
	}
	sc := fastScan{s: s, n: len(s)}
	for {
		kind, lo, hi := sc.nextToken()
		if kind == tokEOF {
			return Result{}, false
		}
		if kind != tokWord || !eqLower(s[lo:hi], "in") {
			continue
		}
		kind2, lo2, hi2 := sc.nextToken()
		n, ok := parseNumberToken(s, kind2, lo2, hi2)
		if !ok {
			continue
		}
		kind3, lo3, hi3 := sc.nextToken()
		if kind3 != tokWord {
			continue
		}
		unit, ok := wordToUnit(s[lo3:hi3])
		if !ok {
			continue
		}
		dir := Future
		mark := sc.i
		kind4, lo4, hi4 := sc.nextToken()
		if kind4 == tokWord && eqLower(s[lo4:hi4], "ago") {
			dir = Past
		} else if kind4 == tokWord && eqLower(s[lo4:hi4], "from") {
			kind5, lo5, hi5 := sc.nextToken()
			if kind5 != tokWord || !(eqLower(s[lo5:hi5], "now") || eqLower(s[lo5:hi5], "today")) {
				sc.i = mark
			}
		} else {
			sc.i = mark
		}

		mul := 1
		if dir == Past {
			mul = -1
		}
		return Result{Time: shiftByUnitN(opts.Reference, unit, mul*n), Truncated: unit, Direction: dir}, true
	}
}

func fastParseEmbeddedInDigits(s string, opts Options) (Result, bool) {
	n := len(s)
	for i := 0; i+3 < n; i++ {
		if s[i] != ' ' {
			continue
		}
		a := s[i+1] | 0x20
		b := s[i+2] | 0x20
		if a != 'i' || b != 'n' || s[i+3] != ' ' {
			continue
		}
		j := i + 4
		for j < n && s[j] == ' ' {
			j++
		}
		if j >= n || !isDigit(s[j]) {
			continue
		}
		val := 0
		for j < n && isDigit(s[j]) {
			val = val*10 + int(s[j]-'0')
			j++
		}
		for j < n && s[j] == ' ' {
			j++
		}
		if j >= n || !isAlpha(s[j]) {
			continue
		}
		start := j
		j++
		for j < n && isAlpha(s[j]) {
			j++
		}
		unit, ok := wordToUnit(s[start:j])
		if !ok {
			continue
		}
		for j < n && s[j] == ' ' {
			j++
		}
		dir := Future
		if j+2 < n && (s[j]|0x20) == 'a' && (s[j+1]|0x20) == 'g' && (s[j+2]|0x20) == 'o' {
			dir = Past
		} else if j+4 < n && (s[j]|0x20) == 'f' && (s[j+1]|0x20) == 'r' && (s[j+2]|0x20) == 'o' && (s[j+3]|0x20) == 'm' {
			k := j + 4
			for k < n && s[k] == ' ' {
				k++
			}
			if k+2 < n && (s[k]|0x20) == 'n' && (s[k+1]|0x20) == 'o' && (s[k+2]|0x20) == 'w' {
				// ok
			} else if k+4 < n && (s[k]|0x20) == 't' && (s[k+1]|0x20) == 'o' && (s[k+2]|0x20) == 'd' &&
				(s[k+3]|0x20) == 'a' && (s[k+4]|0x20) == 'y' {
				// ok
			} else {
				// not a recognized suffix
			}
		}
		mul := 1
		if dir == Past {
			mul = -1
		}
		return Result{Time: shiftByUnitN(opts.Reference, unit, mul*val), Truncated: unit, Direction: dir}, true
	}
	return Result{}, false
}

func parseNumberToken(s string, kind tokenKind, lo, hi int) (int, bool) {
	switch kind {
	case tokNumber, tokOrdinal:
		return atoi(s[lo:hi]), true
	case tokWord:
		return wordToInt(s[lo:hi])
	}
	return 0, false
}

func scanClockAfterAt(sc *fastScan) (hour, min, sec int, unit Unit, ok bool) {
	for sc.i < sc.n && sc.s[sc.i] <= ' ' {
		sc.i++
	}
	start := sc.i
	for sc.i < sc.n && sc.s[sc.i] > ' ' {
		sc.i++
	}
	if start >= sc.i {
		return 0, 0, 0, 0, false
	}
	end := sc.i
	for end > start {
		c := sc.s[end-1]
		if c == ',' || c == '.' {
			end--
			continue
		}
		break
	}
	if end <= start {
		return 0, 0, 0, 0, false
	}
	return parseClockFast(sc.s[start:end])
}

func trimSpaceASCII(s string) string {
	start := 0
	for start < len(s) && s[start] <= ' ' {
		start++
	}
	end := len(s)
	for end > start && s[end-1] <= ' ' {
		end--
	}
	return s[start:end]
}

func parseClockFast(s string) (hour, min, sec int, unit Unit, ok bool) {
	i := 0
	n := len(s)
	if i >= n || !isDigit(s[i]) {
		return 0, 0, 0, 0, false
	}

	h := 0
	count := 0
	for i < n && isDigit(s[i]) {
		h = h*10 + int(s[i]-'0')
		i++
		count++
		if count > 2 {
			return 0, 0, 0, 0, false
		}
	}

	min = 0
	sec = 0
	unit = UnitHour

	if i < n && s[i] == ':' {
		i++
		if i >= n || !isDigit(s[i]) {
			return 0, 0, 0, 0, false
		}
		min = int(s[i] - '0')
		i++
		if i < n && isDigit(s[i]) {
			min = min*10 + int(s[i]-'0')
			i++
		}
		unit = UnitMinute

		if i < n && s[i] == ':' {
			i++
			if i >= n || !isDigit(s[i]) {
				return 0, 0, 0, 0, false
			}
			sec = int(s[i] - '0')
			i++
			if i < n && isDigit(s[i]) {
				sec = sec*10 + int(s[i]-'0')
				i++
			}
			unit = UnitSecond
		}
	}

	ampm := 0
	if i < n {
		var a, b byte
		l := 0
		for i < n {
			c := s[i]
			if c == '.' {
				i++
				continue
			}
			if !isAlpha(c) {
				return 0, 0, 0, 0, false
			}
			if c >= 'A' && c <= 'Z' {
				c += 32
			}
			if l == 0 {
				a = c
				l = 1
			} else if l == 1 {
				b = c
				l = 2
			} else {
				return 0, 0, 0, 0, false
			}
			i++
		}
		if l == 1 {
			if a == 'a' {
				ampm = 1
			} else if a == 'p' {
				ampm = 2
			} else {
				return 0, 0, 0, 0, false
			}
		} else if l == 2 {
			if a == 'a' && b == 'm' {
				ampm = 1
			} else if a == 'p' && b == 'm' {
				ampm = 2
			} else {
				return 0, 0, 0, 0, false
			}
		} else {
			return 0, 0, 0, 0, false
		}
	}

	if min > 59 || sec > 59 {
		return 0, 0, 0, 0, false
	}
	if ampm != 0 {
		if h < 1 || h > 12 {
			return 0, 0, 0, 0, false
		}
	} else if h > 23 {
		return 0, 0, 0, 0, false
	}
	if ampm == 1 && h == 12 {
		h = 0
	} else if ampm == 2 && h != 12 {
		h += 12
	}
	return h, min, sec, unit, true
}
