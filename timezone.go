package naturaldate

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// LoadTimezone resolves IANA names, common abbreviations, "UTC"/"GMT",
// "Local", and numeric offsets such as "+05:45", "-0700", or "UTC+2".
func LoadTimezone(name string) (*time.Location, error) {
	loc, ok := ParseTimezone(name)
	if ok {
		return loc, nil
	}
	return nil, fmt.Errorf("naturaldate: unknown timezone %q", name)
}

// MustLoadTimezone is like LoadTimezone but panics on failure.
func MustLoadTimezone(name string) *time.Location {
	loc, err := LoadTimezone(name)
	if err != nil {
		panic(err)
	}
	return loc
}

// ParseTimezone resolves a timezone string without returning an error.
func ParseTimezone(name string) (*time.Location, bool) {
	s := strings.TrimSpace(name)
	if s == "" {
		return nil, false
	}
	s = strings.Trim(s, ".,;()[]{}")
	if s == "" {
		return nil, false
	}

	switch {
	case eqLower(s, "z"), eqLower(s, "utc"), eqLower(s, "ut"), eqLower(s, "gmt"):
		return time.UTC, true
	case eqLower(s, "local"):
		return time.Local, true
	}

	if loc, ok := commonTimezone(s); ok {
		return loc, true
	}
	if loc, ok := parseOffsetTimezone(s); ok {
		return loc, true
	}
	if loc, err := time.LoadLocation(s); err == nil {
		return loc, true
	}
	return nil, false
}

// ConvertTimezone converts t to the named timezone.
func ConvertTimezone(t time.Time, timezone string) (time.Time, error) {
	loc, err := LoadTimezone(timezone)
	if err != nil {
		return time.Time{}, err
	}
	return t.In(loc), nil
}

// ConvertLocation converts t to loc. A nil loc leaves t unchanged.
func ConvertLocation(t time.Time, loc *time.Location) time.Time {
	if loc == nil {
		return t
	}
	return t.In(loc)
}

// ParseInLocation parses s using loc as the target location.
func ParseInLocation(s string, loc *time.Location, opts ...Options) (Result, bool) {
	options := firstOptions(opts)
	options.Location = loc
	return Parse(s, options)
}

// ParseInTimezone parses s using timezone as the target location.
func ParseInTimezone(s, timezone string, opts ...Options) (Result, bool) {
	loc, err := LoadTimezone(timezone)
	if err != nil {
		return Result{}, false
	}
	return ParseInLocation(s, loc, opts...)
}

// In returns r with its Time converted to loc. Recurrence rules are unchanged.
func (r Result) In(loc *time.Location) Result {
	if loc == nil {
		return r
	}
	r.Time = r.Time.In(loc)
	return r
}

// InTimezone returns r with its Time converted to timezone.
func (r Result) InTimezone(timezone string) (Result, error) {
	loc, err := LoadTimezone(timezone)
	if err != nil {
		return Result{}, err
	}
	return r.In(loc), nil
}

func firstOptions(opts []Options) Options {
	var options Options
	if len(opts) > 0 {
		options = opts[0]
	}
	return options
}

func normalizeOptions(options Options) Options {
	loc := options.Location
	if options.Reference.IsZero() {
		if loc != nil {
			options.Reference = time.Now().In(loc)
		} else {
			options.Reference = time.Now()
		}
	} else if loc != nil {
		options.Reference = options.Reference.In(loc)
	}
	if options.WeekdayDir == 0 {
		options.WeekdayDir = Past
	}
	return options
}

func stripTimezoneSuffix(s string) (string, *time.Location, bool) {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return s, nil, false
	}

	lastStart, lastEnd := lastField(trimmed, len(trimmed))
	if lastStart < 0 {
		return s, nil, false
	}
	zoneStart, zoneEnd := trimFieldPunctuation(trimmed, lastStart, lastEnd)
	if zoneStart < zoneEnd && likelyTimezoneSuffix(trimmed[zoneStart:zoneEnd]) {
		if loc, ok := ParseTimezone(trimmed[zoneStart:zoneEnd]); ok {
			body := strings.TrimSpace(trimmed[:lastStart])
			if body != "" {
				return body, loc, true
			}
		}
	}

	prevStart, prevEnd := lastField(trimmed, lastStart)
	if prevStart >= 0 {
		wordStart, wordEnd := trimFieldPunctuation(trimmed, prevStart, prevEnd)
		if wordStart < wordEnd && eqLower(trimmed[wordStart:wordEnd], "in") && zoneStart < zoneEnd && likelyTimezoneSuffix(trimmed[zoneStart:zoneEnd]) {
			loc, ok := ParseTimezone(trimmed[zoneStart:zoneEnd])
			body := strings.TrimSpace(trimmed[:prevStart])
			if ok && body != "" {
				return body, loc, true
			}
		}
	}

	return s, nil, false
}

func lastField(s string, before int) (int, int) {
	i := before
	for i > 0 && s[i-1] <= ' ' {
		i--
	}
	if i == 0 {
		return -1, -1
	}
	end := i
	for i > 0 && s[i-1] > ' ' {
		i--
	}
	return i, end
}

func trimFieldPunctuation(s string, start, end int) (int, int) {
	for start < end {
		switch s[start] {
		case '.', ',', ';', '(', ')', '[', ']', '{', '}':
			start++
		default:
			goto trimEnd
		}
	}
trimEnd:
	for end > start {
		switch s[end-1] {
		case '.', ',', ';', '(', ')', '[', ']', '{', '}':
			end--
		default:
			return start, end
		}
	}
	return start, end
}

func likelyTimezoneSuffix(s string) bool {
	if s == "" {
		return false
	}
	if s[0] == '+' || s[0] == '-' {
		return true
	}
	if hasByte(s, '/') && hasAlphaByte(s) {
		return true
	}
	switch {
	case eqLower(s, "z"), eqLower(s, "utc"), eqLower(s, "ut"), eqLower(s, "gmt"), eqLower(s, "local"):
		return true
	case hasUTCOrGMTOffsetPrefix(s):
		return true
	}
	_, ok := commonTimezone(s)
	return ok
}

func hasByte(s string, b byte) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return true
		}
	}
	return false
}

func hasAlphaByte(s string) bool {
	for i := 0; i < len(s); i++ {
		if isAlpha(s[i]) {
			return true
		}
	}
	return false
}

func hasUTCOrGMTOffsetPrefix(s string) bool {
	if len(s) <= 3 {
		return false
	}
	prefix := s[:3]
	if !eqLower(prefix, "utc") && !eqLower(prefix, "gmt") {
		return false
	}
	return s[3] == '+' || s[3] == '-'
}

func commonTimezone(name string) (*time.Location, bool) {
	switch {
	case eqLower(name, "est"):
		return time.FixedZone("EST", -5*3600), true
	case eqLower(name, "edt"):
		return time.FixedZone("EDT", -4*3600), true
	case eqLower(name, "cst"):
		return time.FixedZone("CST", -6*3600), true
	case eqLower(name, "cdt"):
		return time.FixedZone("CDT", -5*3600), true
	case eqLower(name, "mst"):
		return time.FixedZone("MST", -7*3600), true
	case eqLower(name, "mdt"):
		return time.FixedZone("MDT", -6*3600), true
	case eqLower(name, "pst"):
		return time.FixedZone("PST", -8*3600), true
	case eqLower(name, "pdt"):
		return time.FixedZone("PDT", -7*3600), true
	case eqLower(name, "cet"):
		return time.FixedZone("CET", 1*3600), true
	case eqLower(name, "cest"):
		return time.FixedZone("CEST", 2*3600), true
	case eqLower(name, "eet"):
		return time.FixedZone("EET", 2*3600), true
	case eqLower(name, "eest"):
		return time.FixedZone("EEST", 3*3600), true
	case eqLower(name, "ist"):
		return time.FixedZone("IST", 5*3600+30*60), true
	case eqLower(name, "jst"):
		return time.FixedZone("JST", 9*3600), true
	case eqLower(name, "kst"):
		return time.FixedZone("KST", 9*3600), true
	case eqLower(name, "aest"):
		return time.FixedZone("AEST", 10*3600), true
	case eqLower(name, "aedt"):
		return time.FixedZone("AEDT", 11*3600), true
	case eqLower(name, "nzst"):
		return time.FixedZone("NZST", 12*3600), true
	case eqLower(name, "nzdt"):
		return time.FixedZone("NZDT", 13*3600), true
	}
	return nil, false
}

func parseOffsetTimezone(s string) (*time.Location, bool) {
	original := s
	upper := strings.ToUpper(s)
	switch {
	case strings.HasPrefix(upper, "UTC"):
		s = s[3:]
	case strings.HasPrefix(upper, "GMT"):
		s = s[3:]
	}
	if s == "" {
		return time.UTC, true
	}
	if s[0] != '+' && s[0] != '-' {
		return nil, false
	}
	sign := 1
	if s[0] == '-' {
		sign = -1
	}
	raw := s[1:]
	if raw == "" {
		return nil, false
	}

	hour, min, ok := parseOffsetParts(raw)
	if !ok || hour > 23 || min > 59 {
		return nil, false
	}
	offset := sign * ((hour * 3600) + (min * 60))
	return time.FixedZone(original, offset), true
}

func parseOffsetParts(raw string) (int, int, bool) {
	if strings.Contains(raw, ":") {
		parts := strings.Split(raw, ":")
		if len(parts) != 2 {
			return 0, 0, false
		}
		hour, err := strconv.Atoi(parts[0])
		if err != nil {
			return 0, 0, false
		}
		min, err := strconv.Atoi(parts[1])
		if err != nil {
			return 0, 0, false
		}
		return hour, min, true
	}
	if len(raw) <= 2 {
		hour, err := strconv.Atoi(raw)
		return hour, 0, err == nil
	}
	if len(raw) != 4 {
		return 0, 0, false
	}
	hour, err := strconv.Atoi(raw[:2])
	if err != nil {
		return 0, 0, false
	}
	min, err := strconv.Atoi(raw[2:])
	if err != nil {
		return 0, 0, false
	}
	return hour, min, true
}
