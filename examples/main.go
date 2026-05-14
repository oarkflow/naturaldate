package main

import (
	"fmt"
	"time"

	"github.com/oarkflow/naturaldate"
)

func main() {
	ref := time.Date(2026, time.March, 16, 15, 4, 5, 0, time.UTC)

	opts := naturaldate.Options{Reference: ref}
	embeddedOpts := naturaldate.Options{Reference: ref, AllowEmbedded: true}

	strictExamples := []struct {
		group string
		input string
		opts  naturaldate.Options
	}{
		// Anchor words.
		{"anchors", "now", opts},
		{"anchors", "today", opts},
		{"anchors", "yesterday", opts},
		{"anchors", "tomorrow", opts},
		{"anchors", "midnight", opts},
		{"anchors", "noon", opts},
		{"anchors", "yesterday at 10am", opts},
		{"anchors", "tomorrow @noon", opts},

		// Relative quantities.
		{"relative", "5 minutes ago", opts},
		{"relative", "three days ago", opts},
		{"relative", "one year from now", opts},
		{"relative", "in 2 weeks", opts},
		{"relative", "in twenty and one days", opts},
		{"relative", "a month from today", opts},
		{"business days", "in 1 business day", opts},
		{"business days", "in 5 business days", opts},
		{"business days", "2 business days ago", opts},
		{"business days", "next business day", opts},
		{"business days", "last business day", opts},
		{"business days", "in 1 business day", naturaldate.Options{
			Reference: ref,
			Holidays:  []time.Time{time.Date(2026, time.March, 17, 0, 0, 0, 0, time.UTC)},
		}},
		{"business days", "next business day", naturaldate.Options{
			Reference:   ref,
			WeekendDays: []time.Weekday{time.Tuesday},
		}},

		// Weekdays.
		{"weekdays", "last monday", opts},
		{"weekdays", "next friday", opts},
		{"weekdays", "this wednesday", opts},
		{"weekdays", "sunday", opts},
		{"weekdays", "monday at 10am", opts},
		{"weekdays", "last sunday at 5:30pm", opts},
		{"weekdays", "sunday at 22:45", opts},
		{"weekdays", "fri @ 6p", opts},
		{"weekdays", "monday", naturaldate.Options{Reference: ref, WeekdayDir: naturaldate.Future}},

		// Months and named dates.
		{"months", "last month", opts},
		{"months", "next month", opts},
		{"months", "next quarter", opts},
		{"months", "last quarter", opts},
		{"months", "last january", opts},
		{"months", "next december", opts},
		{"months", "last february", opts},
		{"named dates", "december 25th", opts},
		{"named dates", "march 15", opts},
		{"named dates", "January 1st at 7:30am", opts},
		{"named dates", "March 16 at 7:30pm", opts},
		{"named dates", "the 25th of december", opts},
		{"named dates", "15 of march at noon", opts},
		{"named dates", "February 29", naturaldate.Options{Reference: time.Date(2025, time.March, 16, 15, 4, 5, 0, time.UTC)}},

		// Times.
		{"times", "10am", opts},
		{"times", "10:05pm", opts},
		{"times", "10:05:22pm", opts},
		{"times", "22:45", opts},
		{"times", "12am", opts},
		{"times", "12pm", opts},

		// Numeric dates.
		{"numeric", "2026-12-25", opts},
		{"numeric", "Q1 2026", opts},
		{"numeric", "12/25", opts},
		{"numeric", "25-12", opts},
		{"numeric", "12.25", opts},
		{"numeric", "03/16/26 at 7:30pm", opts},
		{"numeric ambiguity", "03/04", opts},
		{"numeric ambiguity", "03/04", naturaldate.Options{Reference: ref, DateOrder: naturaldate.DateOrderDMY}},
		{"numeric ambiguity", "26/04/03", naturaldate.Options{Reference: ref, DateOrder: naturaldate.DateOrderYMD}},
		{"numeric ambiguity", "03-04", opts},
		{"numeric ambiguity", "03.04", opts},
		{"numeric direction", "2025-12-25", opts},
		{"numeric direction", "2026-03-16 at 15:04:05", opts},

		// Time boundaries.
		{"boundaries", "start of day", opts},
		{"boundaries", "start of week", opts},
		{"boundaries", "start of month", opts},
		{"boundaries", "start of quarter", opts},
		{"boundaries", "start of year", opts},
		{"boundaries", "beginning of next quarter", opts},
		{"boundaries", "end of day", opts},
		{"boundaries", "end of week", opts},
		{"boundaries", "end of month", opts},
		{"boundaries", "end of quarter", opts},
		{"boundaries", "end of next month", opts},
		{"boundaries", "end of year", opts},

		// Recurring schedules.
		{"recurring", "every 5 minutes", opts},
		{"recurring", "every day", opts},
		{"recurring", "every week", opts},
		{"recurring", "every month", opts},
		{"recurring", "every quarter", opts},
		{"recurring", "once a week", opts},
		{"recurring", "every monday", opts},
		{"recurring", "every weekday at 9am", opts},
		{"recurring", "every monday and wednesday at 9am", opts},
		{"recurring", "every 2 weeks on friday", opts},
		{"recurring", "every 15th of the month", opts},
		{"recurring", "once a month on friday midnight", opts},
		{"recurring", "every year on december 25 at noon", opts},
		{"recurring", "every first monday of the month", naturaldate.Options{Reference: time.Date(2026, time.March, 10, 15, 4, 5, 0, time.UTC)}},
		{"recurring", "every last friday of the month", naturaldate.Options{Reference: time.Date(2026, time.March, 10, 15, 4, 5, 0, time.UTC)}},
	}

	fmt.Printf("Reference: %s (fixed demo clock; not the current time)\n\n", ref.Format(time.RFC3339))
	fmt.Println("Strict Parse examples")
	fmt.Println("---------------------")
	printExamples(strictExamples)

	fmt.Println()
	fmt.Println("Casual Parse examples")
	fmt.Println("---------------------")
	printExamples([]struct {
		group string
		input string
		opts  naturaldate.Options
	}{
		{"casual", "please around tomorrow", naturaldate.Options{Reference: ref, Mode: naturaldate.ModeCasual}},
		{"casual", "approximately next friday", naturaldate.Options{Reference: ref, Mode: naturaldate.ModeFuzzy}},
	})

	fmt.Println()
	fmt.Println("Embedded Parse examples")
	fmt.Println("-----------------------")
	printExamples([]struct {
		group string
		input string
		opts  naturaldate.Options
	}{
		{"embedded", "restart the server in 5 days from now", embeddedOpts},
		{"embedded", "Remind me on the 25th of December at 7:30am", embeddedOpts},
		{"embedded", "Message me in two weeks", embeddedOpts},
		{"embedded", "ship tomorrow, then follow up", embeddedOpts},
	})

	fmt.Println()
	fmt.Println("ParseAll examples")
	fmt.Println("-----------------")
	input := "Ship tomorrow, follow up in 2 weeks, then every monday at 9am."
	for i, result := range naturaldate.ParseAll(input, opts) {
		fmt.Printf("%d. %s", i+1, formatResult(result))
		fmt.Println()
	}

	fmt.Println()
	fmt.Println("ParseWithError examples")
	fmt.Println("-----------------------")
	printWithErrorExamples([]struct {
		input string
		opts  naturaldate.Options
	}{
		{"tomorrow", opts},
		{"Ship tomorrow, then follow up", embeddedOpts},
		{"now later", opts},
		{"", opts},
		{"not a date", opts},
	})

	fmt.Println()
	fmt.Println("Date range examples")
	fmt.Println("-------------------")
	printRangeExamples([]struct {
		input string
		opts  naturaldate.Options
	}{
		{"this week", opts},
		{"this month", opts},
		{"next quarter", opts},
		{"Q1 2026", opts},
		{"last 7 days", opts},
		{"from monday to friday", opts},
		{"between march 1 and march 5", opts},
	})

	fmt.Println()
	fmt.Println("Duration examples")
	fmt.Println("-----------------")
	printDurationExamples([]string{
		"5 minutes",
		"2h 30m",
		"one week and three days",
		"1 month 2 days",
		"1 quarter",
		"1 year 2 months",
	})

	fmt.Println()
	fmt.Println("Timezone examples with fixed reference")
	fmt.Println("--------------------------------------")
	ny := naturaldate.MustLoadTimezone("America/New_York")
	kathmandu := naturaldate.MustLoadTimezone("Asia/Kathmandu")
	printExamples([]struct {
		group string
		input string
		opts  naturaldate.Options
	}{
		{"timezones", "tomorrow at 9am", naturaldate.Options{Reference: ref, Location: ny}},
		{"timezones", "tomorrow at 9am America/New_York", opts},
		{"timezones", "now UTC+05:45", opts},
	})
	converted, err := naturaldate.ConvertTimezone(ref, "Asia/Kathmandu")
	if err != nil {
		panic(err)
	}
	fmt.Printf("%-48q -> %s\n", "convert reference to Asia/Kathmandu", formatTime(converted))
	fmt.Printf("%-48q -> %s\n", "convert reference with loaded location", formatTime(naturaldate.ConvertLocation(ref, kathmandu)))

	fmt.Println()
	fmt.Println("Live current-time examples")
	fmt.Println("--------------------------")
	now := time.Now()
	fmt.Printf("%-48q -> %s\n", "time.Now()", formatTime(now))
	fmt.Printf("%-48q -> %s\n", "time.Now() in Asia/Kathmandu", formatTime(naturaldate.ConvertLocation(now, kathmandu)))
	printExamples([]struct {
		group string
		input string
		opts  naturaldate.Options
	}{
		{"live", "now", naturaldate.Options{Location: kathmandu}},
		{"live", "now UTC+05:45", naturaldate.Options{}},
	})

	fmt.Println()
	fmt.Println("Invalid examples")
	fmt.Println("----------------")
	printInvalidExamples([]struct {
		input string
		opts  naturaldate.Options
	}{
		{"not a date", opts},
		{"", opts},
		{"February 30", opts},
		{"2026-02-29", opts},
		{"13pm", opts},
		{"12:60pm", opts},
		{"every 0 days", opts},
		{"every month on 32", opts},
		{"5 days from", opts},
		{"Restart the server in 5 days from now", opts},
	})
}

func printExamples(examples []struct {
	group string
	input string
	opts  naturaldate.Options
}) {
	lastGroup := ""
	for _, example := range examples {
		if example.group != lastGroup {
			if lastGroup != "" {
				fmt.Println()
			}
			fmt.Printf("[%s]\n", example.group)
			lastGroup = example.group
		}
		result, ok := naturaldate.Parse(example.input, example.opts)
		if !ok {
			fmt.Printf("%-48q -> parse failed\n", example.input)
			continue
		}
		fmt.Printf("%-48q -> %s\n", example.input, formatResult(result))
	}
}

func printInvalidExamples(examples []struct {
	input string
	opts  naturaldate.Options
}) {
	for _, example := range examples {
		result, ok := naturaldate.Parse(example.input, example.opts)
		if ok {
			fmt.Printf("%-48q -> unexpected success: %s\n", example.input, formatResult(result))
			continue
		}
		fmt.Printf("%-48q -> parse failed as expected\n", example.input)
	}
}

func printWithErrorExamples(examples []struct {
	input string
	opts  naturaldate.Options
}) {
	for _, example := range examples {
		result, err := naturaldate.ParseWithError(example.input, example.opts)
		if err != nil {
			fmt.Printf("%-48q -> %s\n", example.input, err)
			continue
		}
		fmt.Printf("%-48q -> %s\n", example.input, formatResult(result))
	}
}

func printRangeExamples(examples []struct {
	input string
	opts  naturaldate.Options
}) {
	for _, example := range examples {
		result, ok := naturaldate.ParseRange(example.input, example.opts)
		if !ok {
			fmt.Printf("%-48q -> range parse failed\n", example.input)
			continue
		}
		fmt.Printf("%-48q -> %s .. %s unit=%s direction=%s match=%q\n",
			example.input,
			formatTime(result.Start),
			formatTime(result.End),
			formatUnit(result.Truncated),
			formatDirection(result.Direction),
			result.Text,
		)
	}
}

func printDurationExamples(inputs []string) {
	for _, input := range inputs {
		if d, ok := naturaldate.ParseDuration(input); ok {
			fmt.Printf("%-48q -> duration=%s\n", input, d)
			continue
		}
		if d, ok := naturaldate.ParseCalendarDuration(input); ok {
			fmt.Printf("%-48q -> calendar=%s applied-to-ref=%s\n",
				input,
				formatCalendarDuration(d),
				formatTime(d.AddTo(time.Date(2026, time.March, 16, 15, 4, 5, 0, time.UTC))),
			)
			continue
		}
		fmt.Printf("%-48q -> duration parse failed\n", input)
	}
}

func formatResult(result naturaldate.Result) string {
	match := ""
	if result.Text != "" {
		match = fmt.Sprintf(" match=%q[%d:%d]", result.Text, result.Start, result.End)
	}
	if result.HasRecur {
		return fmt.Sprintf("%s recurrence=%s next=%s%s",
			formatTime(result.Time),
			formatRecurrence(result.Recur),
			formatNext(result),
			match,
		)
	}
	return fmt.Sprintf("%s direction=%s truncated=%s%s",
		formatTime(result.Time),
		formatDirection(result.Direction),
		formatUnit(result.Truncated),
		match,
	)
}

func formatTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05.999999999 MST")
}

func formatNext(result naturaldate.Result) string {
	next, ok := result.Next(result.Time)
	if !ok {
		return "-"
	}
	return formatTime(next)
}

func formatRecurrence(recur naturaldate.Recurrence) string {
	description := fmt.Sprintf("every %d %s", recur.Interval, formatUnit(recur.Every))
	if recur.OnMonth != 0 {
		description += fmt.Sprintf(" on month %d", recur.OnMonth)
	}
	if recur.OnDate != 0 {
		description += fmt.Sprintf(" on day %d", recur.OnDate)
	}
	if recur.OnDay != 0 {
		description += fmt.Sprintf(" on weekday %d", recur.OnDay)
	}
	if recur.AlsoOnDay != 0 {
		description += fmt.Sprintf(" and weekday %d", recur.AlsoOnDay)
	}
	if recur.OnOrdinal != 0 {
		description += fmt.Sprintf(" ordinal %d", recur.OnOrdinal)
	}
	if recur.OnWeekday {
		description += " on weekdays"
	}
	if recur.HasAt {
		description += fmt.Sprintf(" at %02d:%02d:%02d", recur.At.Hour, recur.At.Min, recur.At.Sec)
	}
	return description
}

func formatCalendarDuration(d naturaldate.CalendarDuration) string {
	return fmt.Sprintf("years=%d months=%d weeks=%d days=%d clock=%s",
		d.Years,
		d.Months,
		d.Weeks,
		d.Days,
		d.Duration,
	)
}

func formatDirection(direction naturaldate.Direction) string {
	switch direction {
	case naturaldate.Past:
		return "past"
	case naturaldate.Present:
		return "present"
	case naturaldate.Future:
		return "future"
	default:
		return "unknown"
	}
}

func formatUnit(unit naturaldate.Unit) string {
	switch unit {
	case naturaldate.UnitSecond:
		return "second"
	case naturaldate.UnitMinute:
		return "minute"
	case naturaldate.UnitHour:
		return "hour"
	case naturaldate.UnitDay:
		return "day"
	case naturaldate.UnitWeek:
		return "week"
	case naturaldate.UnitMonth:
		return "month"
	case naturaldate.UnitYear:
		return "year"
	case naturaldate.UnitQuarter:
		return "quarter"
	default:
		return "none"
	}
}
