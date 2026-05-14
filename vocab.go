package naturaldate

import "time"

// ── numeric words ──────────────────────────────────────────────────────────

// wordToInt converts written-out number words (one…thirty) to integers.
// Returns (n, true) on success. Case-insensitive.
func wordToInt(s string) (int, bool) {
	switch {
	case eqLower(s, "zero"):
		return 0, true
	case eqLower(s, "one") || eqLower(s, "a") || eqLower(s, "an"):
		return 1, true
	case eqLower(s, "two"):
		return 2, true
	case eqLower(s, "three"):
		return 3, true
	case eqLower(s, "four"):
		return 4, true
	case eqLower(s, "five"):
		return 5, true
	case eqLower(s, "six"):
		return 6, true
	case eqLower(s, "seven"):
		return 7, true
	case eqLower(s, "eight"):
		return 8, true
	case eqLower(s, "nine"):
		return 9, true
	case eqLower(s, "ten"):
		return 10, true
	case eqLower(s, "eleven"):
		return 11, true
	case eqLower(s, "twelve"):
		return 12, true
	case eqLower(s, "thirteen"):
		return 13, true
	case eqLower(s, "fourteen"):
		return 14, true
	case eqLower(s, "fifteen"):
		return 15, true
	case eqLower(s, "sixteen"):
		return 16, true
	case eqLower(s, "seventeen"):
		return 17, true
	case eqLower(s, "eighteen"):
		return 18, true
	case eqLower(s, "nineteen"):
		return 19, true
	case eqLower(s, "twenty"):
		return 20, true
	case eqLower(s, "thirty"):
		return 30, true
	case eqLower(s, "forty"):
		return 40, true
	case eqLower(s, "fifty"):
		return 50, true
	case eqLower(s, "hundred"):
		return 100, true
	}
	return 0, false
}

// ── weekday names ──────────────────────────────────────────────────────────

// wordToWeekday maps a word token to a time.Weekday.
// Returns (wd, true) on match.
func wordToWeekday(s string) (time.Weekday, bool) {
	switch {
	case eqLower(s, "monday") || eqLower(s, "mon"):
		return time.Monday, true
	case eqLower(s, "tuesday") || eqLower(s, "tue") || eqLower(s, "tues"):
		return time.Tuesday, true
	case eqLower(s, "wednesday") || eqLower(s, "wed"):
		return time.Wednesday, true
	case eqLower(s, "thursday") || eqLower(s, "thu") || eqLower(s, "thur") || eqLower(s, "thurs"):
		return time.Thursday, true
	case eqLower(s, "friday") || eqLower(s, "fri"):
		return time.Friday, true
	case eqLower(s, "saturday") || eqLower(s, "sat"):
		return time.Saturday, true
	case eqLower(s, "sunday") || eqLower(s, "sun"):
		return time.Sunday, true
	}
	return 0, false
}

// ── month names ────────────────────────────────────────────────────────────

func wordToMonth(s string) (time.Month, bool) {
	switch {
	case eqLower(s, "january") || eqLower(s, "jan"):
		return time.January, true
	case eqLower(s, "february") || eqLower(s, "feb"):
		return time.February, true
	case eqLower(s, "march") || eqLower(s, "mar"):
		return time.March, true
	case eqLower(s, "april") || eqLower(s, "apr"):
		return time.April, true
	case eqLower(s, "may"):
		return time.May, true
	case eqLower(s, "june") || eqLower(s, "jun"):
		return time.June, true
	case eqLower(s, "july") || eqLower(s, "jul"):
		return time.July, true
	case eqLower(s, "august") || eqLower(s, "aug"):
		return time.August, true
	case eqLower(s, "september") || eqLower(s, "sep") || eqLower(s, "sept"):
		return time.September, true
	case eqLower(s, "october") || eqLower(s, "oct"):
		return time.October, true
	case eqLower(s, "november") || eqLower(s, "nov"):
		return time.November, true
	case eqLower(s, "december") || eqLower(s, "dec"):
		return time.December, true
	}
	return 0, false
}

// ── unit names ─────────────────────────────────────────────────────────────

// wordToUnit maps singular/plural unit words to a Unit constant.
func wordToUnit(s string) (Unit, bool) {
	switch {
	case eqLower(s, "second") || eqLower(s, "seconds") || eqLower(s, "sec") || eqLower(s, "secs") || eqLower(s, "s"):
		return UnitSecond, true
	case eqLower(s, "minute") || eqLower(s, "minutes") || eqLower(s, "min") || eqLower(s, "mins"):
		return UnitMinute, true
	case eqLower(s, "hour") || eqLower(s, "hours") || eqLower(s, "hr") || eqLower(s, "hrs") || eqLower(s, "h"):
		return UnitHour, true
	case eqLower(s, "day") || eqLower(s, "days") || eqLower(s, "d"):
		return UnitDay, true
	case eqLower(s, "week") || eqLower(s, "weeks") || eqLower(s, "wk") || eqLower(s, "wks") || eqLower(s, "w"):
		return UnitWeek, true
	case eqLower(s, "month") || eqLower(s, "months") || eqLower(s, "mo") || eqLower(s, "mos"):
		return UnitMonth, true
	case eqLower(s, "year") || eqLower(s, "years") || eqLower(s, "yr") || eqLower(s, "yrs") || eqLower(s, "y"):
		return UnitYear, true
	}
	return 0, false
}

// isBusinessDays returns true if the word is "business days"
func isBusinessDays(s string) bool {
	return eqLower(s, "business") || eqLower(s, "businessday") || eqLower(s, "businessdays")
}

// ampm returns 1 for am/a.m., 2 for pm/p.m., 0 if not an am/pm token.
func wordToAMPM(s string) int {
	switch {
	case eqLower(s, "am") || eqLower(s, "a"):
		return 1
	case eqLower(s, "pm") || eqLower(s, "p"):
		return 2
	}
	return 0
}

// ── helper ─────────────────────────────────────────────────────────────────

// eqLower compares s to a lower-case ASCII literal without allocating.
func eqLower(s, lit string) bool {
	if len(s) != len(lit) {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		if c != lit[i] {
			return false
		}
	}
	return true
}
