package naturaldate

// token kinds
type tokenKind uint8

const (
	tokEOF     tokenKind = iota
	tokWord              // alphabetic run
	tokNumber            // digit run
	tokOrdinal           // 1st 2nd 3rd … (digit run + st/nd/rd/th)
	tokColon             // :
	tokAt                // at / @
	tokComma             // ,
	tokSlash             // /
	tokDash              // -
	tokDot               // .
)

type token struct {
	kind tokenKind
	lo   int // start index in src
	hi   int // exclusive end
}

// lexer tokenises a short natural-language string without any heap allocation.
// It works directly on a string slice so there is zero copying.
type lexer struct {
	src    string
	tokens [128]token // fixed-size buffer for short phrases and embedded snippets
	n      int        // number of valid tokens
	pos    int        // read cursor into tokens[]
}

func (l *lexer) init(s string) {
	l.src = s
	l.n = 0
	l.pos = 0
	l.scan()
}

func (l *lexer) scan() {
	i := 0
	n := 0
	buf := &l.tokens
	src := l.src
	slen := len(src)

	for i < slen && n < len(buf) {
		c := src[i]

		// skip whitespace
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			i++
			continue
		}

		start := i

		switch {
		case isAlpha(c):
			for i < slen && isAlpha(src[i]) {
				i++
			}
			buf[n] = token{tokWord, start, i}
			n++

		case isDigit(c):
			for i < slen && isDigit(src[i]) {
				i++
			}
			// check for ordinal suffix: st nd rd th
			if i+1 < slen && isAlpha(src[i]) && isAlpha(src[i+1]) &&
				isOrdinalSuffix(src[i], src[i+1]) {
				i += 2
				buf[n] = token{tokOrdinal, start, i}
			} else {
				buf[n] = token{tokNumber, start, i}
			}
			n++

		case c == ':':
			buf[n] = token{tokColon, i, i + 1}
			n++
			i++
		case c == '@':
			buf[n] = token{tokAt, i, i + 1}
			n++
			i++
		case c == ',':
			buf[n] = token{tokComma, i, i + 1}
			n++
			i++
		case c == '/':
			buf[n] = token{tokSlash, i, i + 1}
			n++
			i++
		case c == '-':
			buf[n] = token{tokDash, i, i + 1}
			n++
			i++
		case c == '.':
			buf[n] = token{tokDot, i, i + 1}
			n++
			i++
		default:
			i++ // skip unknown punctuation
		}
	}
	l.n = n
}

func (l *lexer) peek() token {
	if l.pos >= l.n {
		return token{kind: tokEOF}
	}
	return l.tokens[l.pos]
}

func (l *lexer) peekAt(offset int) token {
	idx := l.pos + offset
	if idx >= l.n {
		return token{kind: tokEOF}
	}
	return l.tokens[idx]
}

func (l *lexer) next() token {
	t := l.peek()
	if t.kind != tokEOF {
		l.pos++
	}
	return t
}

func (l *lexer) val(t token) string {
	return l.src[t.lo:t.hi]
}

func (l *lexer) wordEq(t token, s string) bool {
	v := l.src[t.lo:t.hi]
	if len(v) != len(s) {
		return false
	}
	for i := 0; i < len(v); i++ {
		cv := v[i]
		if cv >= 'A' && cv <= 'Z' {
			cv += 32
		}
		if cv != s[i] {
			return false
		}
	}
	return true
}

// remaining returns how many tokens are left (including current).
func (l *lexer) remaining() int { return l.n - l.pos }

// save / restore position for backtracking
func (l *lexer) mark() int   { return l.pos }
func (l *lexer) reset(m int) { l.pos = m }

// ── character helpers ──────────────────────────────────────────────────────

func isAlpha(c byte) bool { return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') }
func isDigit(c byte) bool { return c >= '0' && c <= '9' }

func isOrdinalSuffix(a, b byte) bool {
	a |= 0x20
	b |= 0x20
	return (a == 's' && b == 't') ||
		(a == 'n' && b == 'd') ||
		(a == 'r' && b == 'd') ||
		(a == 't' && b == 'h')
}
