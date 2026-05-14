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
		{"yesterday", "yesterday", opts, time.Date(2026, time.March, 15, 0, 0, 0, 0, time.UTC), UnitDay, Present, false},
		{"tomorrow", "tomorrow", opts, time.Date(2026, time.March, 17, 0, 0, 0, 0, time.UTC), UnitDay, Present, false},
		{"5 minutes ago", "5 minutes ago", opts, ref.Add(-5 * time.Minute), UnitMinute, Past, false},
		{"three days ago", "three days ago", opts, ref.AddDate(0, 0, -3), UnitDay, Past, false},
		{"last month", "last month", opts, time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC), UnitMonth, Past, false},
		{"next month", "next month", opts, time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC), UnitMonth, Future, false},
		{"one year from now", "one year from now", opts, ref.AddDate(1, 0, 0), UnitYear, Future, false},
		{"yesterday at 10am", "yesterday at 10am", opts, time.Date(2026, time.March, 15, 10, 0, 0, 0, time.UTC), UnitHour, Present, false},
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
