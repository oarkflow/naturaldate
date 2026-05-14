package naturaldate

import (
	"testing"
	"time"
)

func TestParseCoreExamples(t *testing.T) {
	ref := time.Date(2026, time.March, 16, 15, 4, 5, 0, time.UTC)
	opts := Options{Reference: ref}

	cases := []struct {
		name  string
		input string
		opts  Options
		want  time.Time
		unit  Unit
		dir   Direction
		recur bool
	}{
		{"now", "now", opts, ref, UnitSecond, Present, false},
		{"today", "today", opts, time.Date(2026, time.March, 16, 0, 0, 0, 0, time.UTC), UnitDay, Present, false},
		{"yesterday", "yesterday", opts, time.Date(2026, time.March, 15, 0, 0, 0, 0, time.UTC), UnitDay, Past, false},
		{"tomorrow", "tomorrow", opts, time.Date(2026, time.March, 17, 0, 0, 0, 0, time.UTC), UnitDay, Future, false},
		{"5 minutes ago", "5 minutes ago", opts, ref.Add(-5 * time.Minute), UnitMinute, Past, false},
		{"three days ago", "three days ago", opts, ref.AddDate(0, 0, -3), UnitDay, Past, false},
		{"last month", "last month", opts, time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC), UnitMonth, Past, false},
		{"next month", "next month", opts, time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC), UnitMonth, Future, false},
		{"one year from now", "one year from now", opts, ref.AddDate(1, 0, 0), UnitYear, Future, false},
		{"midnight", "midnight", opts, time.Date(2026, time.March, 16, 0, 0, 0, 0, time.UTC), UnitHour, Past, false},
		{"noon", "noon", opts, time.Date(2026, time.March, 16, 12, 0, 0, 0, time.UTC), UnitHour, Past, false},
		{"yesterday at 10am", "yesterday at 10am", opts, time.Date(2026, time.March, 15, 10, 0, 0, 0, time.UTC), UnitHour, Past, false},
		{"tomorrow at noon", "tomorrow at noon", opts, time.Date(2026, time.March, 17, 12, 0, 0, 0, time.UTC), UnitHour, Future, false},
		{"last sunday at 5:30pm", "last sunday at 5:30pm", opts, time.Date(2026, time.March, 15, 17, 30, 0, 0, time.UTC), UnitDay, Past, false},
		{"sunday at 22:45", "sunday at 22:45", opts, time.Date(2026, time.March, 22, 22, 45, 0, 0, time.UTC), UnitMinute, Future, false},
		{"next January", "next January", opts, time.Date(2027, time.January, 1, 0, 0, 0, 0, time.UTC), UnitMonth, Future, false},
		{"last February", "last February", opts, time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC), UnitMonth, Past, false},
		{"December 25th at 7:30am", "December 25th at 7:30am", opts, time.Date(2026, time.December, 25, 7, 30, 0, 0, time.UTC), UnitDay, Future, false},
		{"10am", "10am", opts, time.Date(2026, time.March, 17, 10, 0, 0, 0, time.UTC), UnitHour, Future, false},
		{"10:05pm", "10:05pm", opts, time.Date(2026, time.March, 16, 22, 5, 0, 0, time.UTC), UnitMinute, Future, false},
		{"10:05:22pm", "10:05:22pm", opts, time.Date(2026, time.March, 16, 22, 5, 22, 0, time.UTC), UnitSecond, Future, false},
		{"embedded in sentence", "Restart the server in 5 days from now", Options{Reference: ref, AllowEmbedded: true}, ref.AddDate(0, 0, 5), UnitDay, Future, false},
		{"ordinal of month", "Remind me on the 25th of December at 7:30am", Options{Reference: ref, AllowEmbedded: true}, time.Date(2026, time.December, 25, 7, 30, 0, 0, time.UTC), UnitDay, Future, false},
		{"message in two weeks", "Message me in two weeks", Options{Reference: ref, AllowEmbedded: true}, ref.AddDate(0, 0, 14), UnitWeek, Future, false},
		{"numeric 3-part", "2026-12-25", opts, time.Date(2026, time.December, 25, 0, 0, 0, 0, time.UTC), UnitDay, Future, false},
		{"numeric 2-part slash", "12/25", opts, time.Date(2026, time.December, 25, 0, 0, 0, 0, time.UTC), UnitDay, Future, false},
		{"numeric 2-part dash", "25-12", opts, time.Date(2026, time.December, 25, 0, 0, 0, 0, time.UTC), UnitDay, Future, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r, ok := Parse(tc.input, tc.opts)
			if !ok {
				t.Fatalf("expected parse success")
			}
			if !r.Time.Equal(tc.want) {
				t.Fatalf("time mismatch: got %v want %v", r.Time, tc.want)
			}
			if r.Truncated != tc.unit {
				t.Fatalf("unit mismatch: got %v want %v", r.Truncated, tc.unit)
			}
			if r.Direction != tc.dir {
				t.Fatalf("direction mismatch: got %v want %v", r.Direction, tc.dir)
			}
			if r.HasRecur != tc.recur {
				t.Fatalf("recurrence mismatch: got %v want %v", r.HasRecur, tc.recur)
			}
		})
	}
}

func TestParseRecurring(t *testing.T) {
	ref := time.Date(2026, time.March, 16, 15, 4, 5, 0, time.UTC)

	{
		r, ok := Parse("every 5 minutes", Options{Reference: ref})
		if !ok || !r.HasRecur {
			t.Fatalf("expected recurring parse")
		}
		if r.Recur.Every != UnitMinute || r.Recur.Interval != 5 {
			t.Fatalf("unexpected recurrence: %+v", r.Recur)
		}
		if !r.Time.Equal(ref.Add(5 * time.Minute)) {
			t.Fatalf("next occurrence mismatch: got %v", r.Time)
		}
	}

	{
		r, ok := Parse("every day", Options{Reference: ref})
		if !ok || !r.HasRecur {
			t.Fatalf("expected recurring parse")
		}
		if r.Recur.Every != UnitDay || r.Recur.Interval != 1 {
			t.Fatalf("unexpected recurrence: %+v", r.Recur)
		}
		if !r.Time.Equal(time.Date(2026, time.March, 17, 0, 0, 0, 0, time.UTC)) {
			t.Fatalf("next occurrence mismatch: got %v", r.Time)
		}
	}

	{
		r, ok := Parse("once a month on friday midnight", Options{Reference: ref})
		if !ok || !r.HasRecur {
			t.Fatalf("expected recurring parse")
		}
		if r.Recur.Every != UnitMonth || r.Recur.Interval != 1 {
			t.Fatalf("unexpected recurrence: %+v", r.Recur)
		}
		if !r.Recur.HasAt || r.Recur.At.Hour != 0 || r.Recur.At.Min != 0 || r.Recur.At.Sec != 0 {
			t.Fatalf("expected midnight clock")
		}
		if r.Recur.OnDay != 5 {
			t.Fatalf("expected Friday (ISO=5), got %d", r.Recur.OnDay)
		}
		if !r.Time.Equal(time.Date(2026, time.April, 3, 0, 0, 0, 0, time.UTC)) {
			t.Fatalf("next occurrence mismatch: got %v", r.Time)
		}
	}
}

func TestRecurrenceNext(t *testing.T) {
	ref := time.Date(2026, time.March, 16, 15, 4, 5, 0, time.UTC)

	r, ok := Parse("every monday", Options{Reference: ref})
	if !ok || !r.HasRecur {
		t.Fatalf("expected recurring parse")
	}
	next, ok := r.Next(ref)
	if !ok {
		t.Fatalf("expected next occurrence")
	}
	if want := time.Date(2026, time.March, 23, 0, 0, 0, 0, time.UTC); !next.Equal(want) {
		t.Fatalf("next occurrence mismatch: got %v want %v", next, want)
	}

	r, ok = Parse("every year on december 25 at noon", Options{Reference: ref})
	if !ok || !r.HasRecur {
		t.Fatalf("expected yearly recurring parse")
	}
	if r.Recur.OnMonth != int8(time.December) || r.Recur.OnDate != 25 {
		t.Fatalf("unexpected recurrence anchor: %+v", r.Recur)
	}
	if want := time.Date(2026, time.December, 25, 12, 0, 0, 0, time.UTC); !r.Time.Equal(want) {
		t.Fatalf("next occurrence mismatch: got %v want %v", r.Time, want)
	}
}

func TestParseAllAndAppendAll(t *testing.T) {
	ref := time.Date(2026, time.March, 16, 15, 4, 5, 0, time.UTC)
	input := "Ship tomorrow, follow up in 2 weeks, then every monday at 9am."

	results := ParseAll(input, Options{Reference: ref})
	if len(results) != 3 {
		t.Fatalf("result count mismatch: got %d want 3", len(results))
	}
	if want := time.Date(2026, time.March, 17, 0, 0, 0, 0, time.UTC); !results[0].Time.Equal(want) {
		t.Fatalf("first time mismatch: got %v want %v", results[0].Time, want)
	}
	if want := ref.AddDate(0, 0, 14); !results[1].Time.Equal(want) {
		t.Fatalf("second time mismatch: got %v want %v", results[1].Time, want)
	}
	if !results[2].HasRecur {
		t.Fatalf("expected recurring third result")
	}

	dst := make([]Result, 0, 4)
	dst = AppendAll(dst, input, Options{Reference: ref})
	if len(dst) != len(results) {
		t.Fatalf("append result count mismatch: got %d want %d", len(dst), len(results))
	}
}

func TestParseBusinessDays(t *testing.T) {
	ref := time.Date(2026, time.March, 16, 15, 4, 5, 0, time.UTC) // Monday
	opts := Options{Reference: ref}

	cases := []struct {
		input string
		opts  Options
		want  time.Time
	}{
		{"in 1 business day", opts, time.Date(2026, time.March, 17, 15, 4, 5, 0, time.UTC)},
		{"in 5 business days", opts, time.Date(2026, time.March, 23, 15, 4, 5, 0, time.UTC)},
		{"2 business days ago", opts, time.Date(2026, time.March, 12, 15, 4, 5, 0, time.UTC)},
		{
			"in 1 business day",
			Options{
				Reference: ref,
				Holidays:  []time.Time{time.Date(2026, time.March, 17, 0, 0, 0, 0, time.UTC)},
			},
			time.Date(2026, time.March, 18, 15, 4, 5, 0, time.UTC),
		},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			r, ok := Parse(tc.input, tc.opts)
			if !ok {
				t.Fatalf("expected parse success")
			}
			if !r.Time.Equal(tc.want) {
				t.Fatalf("time mismatch: got %v want %v", r.Time, tc.want)
			}
			if r.Truncated != UnitDay {
				t.Fatalf("unit mismatch: got %v want %v", r.Truncated, UnitDay)
			}
		})
	}
}

func TestParseAmbiguityRules(t *testing.T) {
	ref := time.Date(2026, time.March, 16, 15, 4, 5, 0, time.UTC)

	cases := []struct {
		name  string
		input string
		opts  Options
		want  time.Time
		dir   Direction
	}{
		{
			"slash prefers month day",
			"03/04",
			Options{Reference: ref},
			time.Date(2027, time.March, 4, 0, 0, 0, 0, time.UTC),
			Future,
		},
		{
			"dash prefers day month",
			"03-04",
			Options{Reference: ref},
			time.Date(2026, time.April, 3, 0, 0, 0, 0, time.UTC),
			Future,
		},
		{
			"dot prefers day month",
			"03.04",
			Options{Reference: ref},
			time.Date(2026, time.April, 3, 0, 0, 0, 0, time.UTC),
			Future,
		},
		{
			"bare weekday defaults past",
			"monday",
			Options{Reference: ref},
			time.Date(2026, time.March, 9, 0, 0, 0, 0, time.UTC),
			Past,
		},
		{
			"bare weekday can prefer future",
			"monday",
			Options{Reference: ref, WeekdayDir: Future},
			time.Date(2026, time.March, 23, 0, 0, 0, 0, time.UTC),
			Future,
		},
		{
			"bare weekday with past time moves future",
			"monday at 10am",
			Options{Reference: ref},
			time.Date(2026, time.March, 23, 10, 0, 0, 0, time.UTC),
			Future,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r, ok := Parse(tc.input, tc.opts)
			if !ok {
				t.Fatalf("expected parse success")
			}
			if !r.Time.Equal(tc.want) {
				t.Fatalf("time mismatch: got %v want %v", r.Time, tc.want)
			}
			if r.Direction != tc.dir {
				t.Fatalf("direction mismatch: got %v want %v", r.Direction, tc.dir)
			}
		})
	}
}

func TestParseInvalid(t *testing.T) {
	if _, ok := Parse("not a date", Options{}); ok {
		t.Fatalf("expected parse failure")
	}
}

func TestParseStrictRequiresWholeExpression(t *testing.T) {
	ref := time.Date(2026, time.March, 16, 15, 4, 5, 0, time.UTC)
	if _, ok := Parse("Restart the server in 5 days from now", Options{Reference: ref}); ok {
		t.Fatalf("expected embedded phrase to fail without AllowEmbedded")
	}
	if _, ok := Parse("now tomorrow", Options{Reference: ref}); ok {
		t.Fatalf("expected trailing date tokens to fail")
	}
	if _, ok := Parse("every day later", Options{Reference: ref}); ok {
		t.Fatalf("expected trailing recurrence tokens to fail")
	}
}

func TestParseRejectsInvalidCalendarAndClockValues(t *testing.T) {
	ref := time.Date(2026, time.March, 16, 15, 4, 5, 0, time.UTC)
	cases := []string{
		"February 30",
		"2026-02-29",
		"13pm",
		"12:60pm",
		"every 0 days",
		"every month on 32",
		"5 days from",
		"in 5 days from",
		"5 days from later",
	}
	for _, input := range cases {
		t.Run(input, func(t *testing.T) {
			if _, ok := Parse(input, Options{Reference: ref}); ok {
				t.Fatalf("expected parse failure")
			}
		})
	}
}

func TestParseLongInputDoesNotWrapOffsets(t *testing.T) {
	ref := time.Date(2026, time.March, 16, 15, 4, 5, 0, time.UTC)
	prefix := "this sentence is intentionally long enough to exceed two hundred and fifty five bytes before the useful date phrase appears, so the lexer must keep real integer offsets instead of wrapping byte positions through a narrow field and slicing the wrong span later in the parse "
	input := prefix + "in 5 days"
	r, ok := Parse(input, Options{Reference: ref, AllowEmbedded: true})
	if !ok {
		t.Fatalf("expected embedded parse success")
	}
	if want := ref.AddDate(0, 0, 5); !r.Time.Equal(want) {
		t.Fatalf("time mismatch: got %v want %v", r.Time, want)
	}
}

func TestParseMonthDayChoosesNextValidFutureDate(t *testing.T) {
	ref := time.Date(2025, time.March, 16, 15, 4, 5, 0, time.UTC)
	r, ok := Parse("February 29", Options{Reference: ref})
	if !ok {
		t.Fatalf("expected leap day parse success")
	}
	if want := time.Date(2028, time.February, 29, 0, 0, 0, 0, time.UTC); !r.Time.Equal(want) {
		t.Fatalf("time mismatch: got %v want %v", r.Time, want)
	}
}

func TestParseMonthDayWithTimeCanUseToday(t *testing.T) {
	ref := time.Date(2026, time.March, 16, 15, 4, 5, 0, time.UTC)
	r, ok := Parse("March 16 at 7:30pm", Options{Reference: ref})
	if !ok {
		t.Fatalf("expected parse success")
	}
	if want := time.Date(2026, time.March, 16, 19, 30, 0, 0, time.UTC); !r.Time.Equal(want) {
		t.Fatalf("time mismatch: got %v want %v", r.Time, want)
	}
}

func TestParseMonthDayCanUseTodayWithoutFutureJump(t *testing.T) {
	ref := time.Date(2026, time.March, 16, 15, 4, 5, 0, time.UTC)
	cases := []struct {
		input string
		want  time.Time
		dir   Direction
	}{
		{"March 16", time.Date(2026, time.March, 16, 0, 0, 0, 0, time.UTC), Present},
		{"03/16", time.Date(2026, time.March, 16, 0, 0, 0, 0, time.UTC), Present},
		{"March 16 at 15:04:05", ref, Present},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			r, ok := Parse(tc.input, Options{Reference: ref})
			if !ok {
				t.Fatalf("expected parse success")
			}
			if !r.Time.Equal(tc.want) {
				t.Fatalf("time mismatch: got %v want %v", r.Time, tc.want)
			}
			if r.Direction != tc.dir {
				t.Fatalf("direction mismatch: got %v want %v", r.Direction, tc.dir)
			}
		})
	}
}

func TestParseAbsoluteNumericDateDirection(t *testing.T) {
	ref := time.Date(2026, time.March, 16, 15, 4, 5, 0, time.UTC)
	cases := []struct {
		input string
		dir   Direction
	}{
		{"2025-12-25", Past},
		{"2026-03-16 at 15:04:05", Present},
		{"2026-12-25", Future},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			r, ok := Parse(tc.input, Options{Reference: ref})
			if !ok {
				t.Fatalf("expected parse success")
			}
			if r.Direction != tc.dir {
				t.Fatalf("direction mismatch: got %v want %v", r.Direction, tc.dir)
			}
		})
	}
}

func TestParseInLocation(t *testing.T) {
	ny := MustLoadTimezone("America/New_York")
	ref := time.Date(2026, time.March, 16, 15, 4, 5, 0, time.UTC)

	r, ok := Parse("tomorrow at 9am", Options{Reference: ref, Location: ny})
	if !ok {
		t.Fatalf("expected parse success")
	}
	want := time.Date(2026, time.March, 17, 9, 0, 0, 0, ny)
	if !r.Time.Equal(want) {
		t.Fatalf("time mismatch: got %v want %v", r.Time, want)
	}
	if r.Time.Location() != ny {
		t.Fatalf("location mismatch: got %v want %v", r.Time.Location(), ny)
	}
}

func TestParseTimezoneSuffix(t *testing.T) {
	ref := time.Date(2026, time.March, 16, 15, 4, 5, 0, time.UTC)

	r, ok := Parse("tomorrow at 9am America/New_York", Options{Reference: ref})
	if !ok {
		t.Fatalf("expected parse success")
	}
	if name := r.Time.Location().String(); name != "America/New_York" {
		t.Fatalf("location mismatch: got %q", name)
	}
	if want := time.Date(2026, time.March, 17, 9, 0, 0, 0, MustLoadTimezone("America/New_York")); !r.Time.Equal(want) {
		t.Fatalf("time mismatch: got %v want %v", r.Time, want)
	}

	r, ok = Parse("now UTC+05:45", Options{Reference: ref})
	if !ok {
		t.Fatalf("expected offset timezone parse success")
	}
	_, offset := r.Time.Zone()
	if offset != 5*3600+45*60 {
		t.Fatalf("offset mismatch: got %d", offset)
	}
	if !r.Time.Equal(ref) {
		t.Fatalf("instant mismatch: got %v want %v", r.Time, ref)
	}
}

func TestTimezoneHelpers(t *testing.T) {
	loc, err := LoadTimezone("+05:45")
	if err != nil {
		t.Fatalf("expected offset timezone: %v", err)
	}
	_, offset := time.Date(2026, time.March, 16, 12, 0, 0, 0, loc).Zone()
	if offset != 5*3600+45*60 {
		t.Fatalf("offset mismatch: got %d", offset)
	}

	ref := time.Date(2026, time.March, 16, 15, 4, 5, 0, time.UTC)
	converted, err := ConvertTimezone(ref, "Asia/Kathmandu")
	if err != nil {
		t.Fatalf("expected conversion success: %v", err)
	}
	if name := converted.Location().String(); name != "Asia/Kathmandu" {
		t.Fatalf("location mismatch: got %q", name)
	}
	if !converted.Equal(ref) {
		t.Fatalf("conversion changed instant: got %v want %v", converted, ref)
	}
}

func FuzzParse(f *testing.F) {
	for _, seed := range []string{
		"now",
		"yesterday at 10am",
		"Restart the server in 5 days from now",
		"February 29",
		"every month on 31",
		string(make([]byte, 300)),
	} {
		f.Add(seed)
	}
	ref := time.Date(2026, time.March, 16, 15, 4, 5, 0, time.UTC)
	f.Fuzz(func(t *testing.T, input string) {
		_, _ = Parse(input, Options{Reference: ref, AllowEmbedded: true})
	})
}

func BenchmarkParseNow(b *testing.B) {
	ref := time.Date(2026, time.March, 16, 15, 4, 5, 0, time.UTC)
	opts := Options{Reference: ref}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Parse("now", opts)
	}
}

func BenchmarkParseTime(b *testing.B) {
	ref := time.Date(2026, time.March, 16, 15, 4, 5, 0, time.UTC)
	opts := Options{Reference: ref}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Parse("10:05:22pm", opts)
	}
}

func BenchmarkParseEmbedded(b *testing.B) {
	ref := time.Date(2026, time.March, 16, 15, 4, 5, 0, time.UTC)
	opts := Options{Reference: ref, AllowEmbedded: true}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Parse("Restart the server in 5 days from now", opts)
	}
}

func BenchmarkParseRecurring(b *testing.B) {
	ref := time.Date(2026, time.March, 16, 15, 4, 5, 0, time.UTC)
	opts := Options{Reference: ref}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Parse("once a month on friday midnight", opts)
	}
}

func BenchmarkAppendAll(b *testing.B) {
	ref := time.Date(2026, time.March, 16, 15, 4, 5, 0, time.UTC)
	opts := Options{Reference: ref}
	input := "Ship tomorrow, follow up in 2 weeks, then every monday at 9am."
	dst := make([]Result, 0, 4)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dst = dst[:0]
		dst = AppendAll(dst, input, opts)
	}
}

func TestParseTimeBoundaries(t *testing.T) {
	ref := time.Date(2026, time.March, 16, 15, 4, 5, 0, time.UTC)
	opts := Options{Reference: ref}

	cases := []struct {
		name  string
		input string
		want  time.Time
	}{
		{"start of day", "start of day", time.Date(2026, time.March, 16, 0, 0, 0, 0, time.UTC)},
		{"start of week", "start of week", time.Date(2026, time.March, 15, 0, 0, 0, 0, time.UTC)},
		{"start of month", "start of month", time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC)},
		{"start of year", "start of year", time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)},
		{"end of day", "end of day", time.Date(2026, time.March, 16, 23, 59, 59, 999999999, time.UTC)},
		{"end of week", "end of week", time.Date(2026, time.March, 21, 23, 59, 59, 999999999, time.UTC)},
		{"end of month", "end of month", time.Date(2026, time.March, 31, 23, 59, 59, 999999999, time.UTC)},
		{"end of year", "end of year", time.Date(2026, time.December, 31, 23, 59, 59, 999999999, time.UTC)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r, ok := Parse(tc.input, opts)
			if !ok {
				t.Fatalf("expected parse success")
			}
			if !r.Time.Equal(tc.want) {
				t.Fatalf("time mismatch: got %v want %v", r.Time, tc.want)
			}
		})
	}
}

func TestParseEnhancedRecurring(t *testing.T) {
	ref := time.Date(2026, time.March, 10, 15, 4, 5, 0, time.UTC)

	{
		r, ok := Parse("every first monday of the month", Options{Reference: ref})
		if !ok || !r.HasRecur {
			t.Fatalf("expected recurring parse for 'first monday of the month'")
		}
		if r.Recur.OnDay != 1 {
			t.Fatalf("expected OnDay=1 (Monday), got %d", r.Recur.OnDay)
		}
		if r.Recur.OnOrdinal != 1 {
			t.Fatalf("expected OnOrdinal=1 (first), got %d", r.Recur.OnOrdinal)
		}
		// First Monday on or after March 10 is April 6
		want := time.Date(2026, time.April, 6, 0, 0, 0, 0, time.UTC)
		if !r.Time.Equal(want) {
			t.Fatalf("time mismatch: got %v want %v", r.Time, want)
		}
	}

	{
		r, ok := Parse("every last friday of the month", Options{Reference: ref})
		if !ok || !r.HasRecur {
			t.Fatalf("expected recurring parse for 'last friday of the month'")
		}
		if r.Recur.OnDay != 5 {
			t.Fatalf("expected OnDay=5 (Friday), got %d", r.Recur.OnDay)
		}
		if r.Recur.OnOrdinal != -1 {
			t.Fatalf("expected OnOrdinal=-1 (last), got %d", r.Recur.OnOrdinal)
		}
		// Last Friday of March 2026 is March 27
		want := time.Date(2026, time.March, 27, 0, 0, 0, 0, time.UTC)
		if !r.Time.Equal(want) {
			t.Fatalf("time mismatch: got %v want %v", r.Time, want)
		}
	}
}

func TestParseEssentialFeatureAdditions(t *testing.T) {
	ref := time.Date(2026, time.March, 16, 15, 4, 5, 0, time.UTC) // Monday

	cases := []struct {
		name  string
		input string
		opts  Options
		want  time.Time
		unit  Unit
	}{
		{"next business day", "next business day", Options{Reference: ref}, time.Date(2026, time.March, 17, 15, 4, 5, 0, time.UTC), UnitDay},
		{"custom weekend", "next business day", Options{Reference: ref, WeekendDays: []time.Weekday{time.Tuesday}}, time.Date(2026, time.March, 18, 15, 4, 5, 0, time.UTC), UnitDay},
		{"next quarter", "next quarter", Options{Reference: ref}, time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC), UnitQuarter},
		{"quarter literal", "Q1 2027", Options{Reference: ref}, time.Date(2027, time.January, 1, 0, 0, 0, 0, time.UTC), UnitQuarter},
		{"end of next month", "end of next month", Options{Reference: ref}, time.Date(2026, time.April, 30, 23, 59, 59, 999999999, time.UTC), UnitMonth},
		{"casual prefix", "please around tomorrow", Options{Reference: ref, Mode: ModeCasual}, time.Date(2026, time.March, 17, 0, 0, 0, 0, time.UTC), UnitDay},
		{"date order dmy", "03/04", Options{Reference: ref, DateOrder: DateOrderDMY}, time.Date(2026, time.April, 3, 0, 0, 0, 0, time.UTC), UnitDay},
		{"date order ymd", "26/04/03", Options{Reference: ref, DateOrder: DateOrderYMD}, time.Date(2026, time.April, 3, 0, 0, 0, 0, time.UTC), UnitDay},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r, ok := Parse(tc.input, tc.opts)
			if !ok {
				t.Fatalf("expected parse success")
			}
			if !r.Time.Equal(tc.want) {
				t.Fatalf("time mismatch: got %v want %v", r.Time, tc.want)
			}
			if r.Truncated != tc.unit {
				t.Fatalf("unit mismatch: got %v want %v", r.Truncated, tc.unit)
			}
		})
	}
}

func TestParseWithErrorAndMatchMetadata(t *testing.T) {
	ref := time.Date(2026, time.March, 16, 15, 4, 5, 0, time.UTC)
	r, err := ParseWithError("Ship tomorrow, then wait", Options{Reference: ref, AllowEmbedded: true})
	if err != nil {
		t.Fatalf("expected parse success: %v", err)
	}
	if r.Start != 5 || r.End != 13 || r.Text != "tomorrow" || r.Match() != "tomorrow" {
		t.Fatalf("unexpected match metadata: start=%d end=%d text=%q", r.Start, r.End, r.Text)
	}

	_, err = ParseWithError("now later", Options{Reference: ref})
	if err == nil {
		t.Fatalf("expected error")
	}
	perr, ok := err.(*ParseError)
	if !ok || perr.Kind != ErrTrailingInput {
		t.Fatalf("expected trailing input error, got %T %v", err, err)
	}
}

func TestParseRange(t *testing.T) {
	ref := time.Date(2026, time.March, 16, 15, 4, 5, 0, time.UTC)

	cases := []struct {
		input string
		wantS time.Time
		wantE time.Time
		unit  Unit
	}{
		{"this month", time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC), UnitMonth},
		{"next quarter", time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC), UnitQuarter},
		{"last 7 days", ref.AddDate(0, 0, -7), ref, UnitDay},
		{"from monday to friday", time.Date(2026, time.March, 9, 0, 0, 0, 0, time.UTC), time.Date(2026, time.March, 14, 0, 0, 0, 0, time.UTC), UnitDay},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			r, ok := ParseRange(tc.input, Options{Reference: ref})
			if !ok {
				t.Fatalf("expected range parse success")
			}
			if !r.Start.Equal(tc.wantS) || !r.End.Equal(tc.wantE) {
				t.Fatalf("range mismatch: got %v..%v want %v..%v", r.Start, r.End, tc.wantS, tc.wantE)
			}
			if r.Truncated != tc.unit {
				t.Fatalf("unit mismatch: got %v want %v", r.Truncated, tc.unit)
			}
		})
	}
}

func TestParseDurationAdditions(t *testing.T) {
	d, ok := ParseDuration("2h 30m")
	if !ok || d != 150*time.Minute {
		t.Fatalf("duration mismatch: got %v ok=%v", d, ok)
	}

	cd, ok := ParseCalendarDuration("one week and three days")
	if !ok || cd.Weeks != 1 || cd.Days != 3 {
		t.Fatalf("calendar duration mismatch: %+v ok=%v", cd, ok)
	}

	if _, ok := ParseDuration("1 month"); ok {
		t.Fatalf("time.Duration parser should reject calendar months")
	}
}

func TestParseNewRecurringForms(t *testing.T) {
	ref := time.Date(2026, time.March, 16, 15, 4, 5, 0, time.UTC)

	r, ok := Parse("every weekday at 9am", Options{Reference: ref})
	if !ok || !r.HasRecur || !r.Time.Equal(time.Date(2026, time.March, 17, 9, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected weekday recurrence: %+v ok=%v", r, ok)
	}

	r, ok = Parse("every monday and wednesday at 9am", Options{Reference: ref})
	if !ok || !r.HasRecur || !r.Time.Equal(time.Date(2026, time.March, 18, 9, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected multi-weekday recurrence: %+v ok=%v", r, ok)
	}

	r, ok = Parse("every 15th of the month", Options{Reference: ref})
	if !ok || !r.HasRecur || !r.Time.Equal(time.Date(2026, time.April, 15, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected monthly date recurrence: %+v ok=%v", r, ok)
	}
}
