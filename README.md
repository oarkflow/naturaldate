# naturaldate

<p align="center">
  <a href="https://pkg.go.dev/github.com/oarkflow/naturaldate">
    <img src="https://pkg.go.dev/badge/github.com/oarkflow/naturaldate.svg" alt="Go Reference">
  </a>
  <a href="https://github.com/oarkflow/naturaldate/actions">
    <img src="https://github.com/oarkflow/naturaldate/workflows/Test/badge.svg" alt="Tests">
  </a>
  <a href="https://goreportcard.com/report/github.com/oarkflow/naturaldate">
    <img src="https://goreportcard.com/badge/github.com/oarkflow/naturaldate" alt="Go Report Card">
  </a>
</p>

A high-performance, zero-allocation Go library for parsing natural language date and time expressions. It supports absolute dates, relative expressions, recurring schedules, and optional embedded date extraction from free-form text.

## Features

- **Zero allocations** - All parsing is done on the stack with fixed-size arrays
- **High performance** - Optimized lexer and parser designed for speed
- **Flexible parsing** - Supports a wide variety of natural language date formats
- **Recurring schedules** - Parse "every week", "once a month", etc.
- **Embedded dates** - Extract dates from within larger text strings
- **Date ranges and durations** - Parse reporting windows and mixed calendar durations
- **Detailed errors and match spans** - Opt into structured failures and source offsets
- **Business and fiscal calendars** - Configure weekends, holidays, business-day end, and fiscal year starts

## Installation

```bash
go get github.com/oarkflow/naturaldate
```

## Quick Start

```go
package main

import (
	"fmt"
	"time"

	"github.com/oarkflow/naturaldate"
)

func main() {
	// Basic parsing
	result, ok := naturaldate.Parse("yesterday", naturaldate.Options{})
	if ok {
		fmt.Println(result.Time)
	}

	// With reference time (default is time.Now())
	ref := time.Date(2026, time.March, 16, 15, 4, 5, 0, time.UTC)
	result, ok = naturaldate.Parse("5 minutes ago", naturaldate.Options{Reference: ref})
	if ok {
		fmt.Println(result.Time) // 2026-03-16 14:59:05 +0000 UTC
	}

	// MustParse panics on failure
	result = naturaldate.MustParse("next friday at 5pm")
}
```

## Supported Formats

### Absolute Dates

| Input | Output (relative to reference) |
|-------|--------------------------------|
| `now` | Current time |
| `today` | Start of today |
| `yesterday` | Start of yesterday |
| `tomorrow` | Start of tomorrow |
| `midnight` | 00:00 of current day |
| `noon` | 12:00 of current day |

### Relative Expressions

| Input | Description |
|-------|-------------|
| `5 minutes ago` | 5 minutes before reference |
| `three days ago` | 3 days before reference |
| `one year from now` | 1 year after reference |
| `in 2 weeks` | 2 weeks after reference |
| `in 5 business days` | 5 weekdays after reference, skipping configured holidays |
| `next business day` | Next configured working day |
| `next 3 weekdays` | 3 weekdays after reference |
| `last 2 business weeks` | 10 configured business days before reference |
| `in half an hour` | 30 minutes after reference |
| `the day after tomorrow` | 2 days after today |
| `the day before yesterday` | 2 days before today |
| `2 Mondays from now` | Second Monday after reference |
| `3 Fridays ago` | Third Friday before reference |

### Weekday References

| Input | Description |
|-------|-------------|
| `last monday` | Most recent Monday (past) |
| `next friday` | Upcoming Friday (future) |
| `this wednesday` | Upcoming Wednesday |
| `sunday` | Nearest Sunday (configurable direction) |
| `monday at 10am` | Next Monday at 10:00 |
| `last sunday at 5:30pm` | Last Sunday at 17:30 |
| `friday evening` | Friday at 18:00 |
| `next weekend` | Start of the next Saturday |

### Month/Year References

| Input | Description |
|-------|-------------|
| `last month` | First day of previous month |
| `next month` | First day of next month |
| `next quarter` | First day of next quarter |
| `next fiscal quarter` | First day of the next configured fiscal quarter |
| `last january` | January of previous year (if past) |
| `next december` | December of next year |
| `last february` | February of current or previous year |

### Specific Dates

| Input | Description |
|-------|-------------|
| `december 25th` | December 25th of current/next year |
| `march 15` | March 15th of current/next year |
| `january 1st at 7:30am` | January 1st at 07:30 |
| `the 25th of december` | December 25th |
| `remind me on the 15th of march at noon` | March 15th at 12:00 |

### Time Expressions

| Input | Description |
|-------|-------------|
| `10am` | 10:00 of next day |
| `10:05pm` | 22:05 of current/next day |
| `10:05:22pm` | 22:05:22 |
| `quarter past 5` | 05:15 of current/next day |
| `half past 3` | 03:30 of current/next day |
| `tomorrow morning` | Tomorrow at 09:00 |
| `this afternoon` | Today/tomorrow at 15:00 |
| `tonight` | Today/tomorrow at 20:00 |
| `EOD` / `COB` | Configured business-day end, default 17:00 |
| `midnight` | 00:00 of current day |
| `noon` | 12:00 of current day |

### Numeric Dates

| Input | Description |
|-------|-------------|
| `2026-12-25` | Full ISO date |
| `Q1 2026` | First day of quarter 1 in 2026 |
| `12/25` | Month/Day (US format) |
| `25-12` | Day/Month (EU format) |
| `12.25` | Day.Month alternative |

### Recurring Schedules

| Input | Description |
|-------|-------------|
| `every 5 minutes` | Every 5 minutes |
| `every day` | Daily |
| `every week` | Weekly |
| `every month` | Monthly |
| `once a week` | Weekly |
| `every monday` | Every Monday |
| `every weekday at 9am` | Every weekday at 09:00 |
| `every monday and wednesday at 9am` | Every Monday and Wednesday at 09:00 |
| `every 2 weeks on friday` | Bi-weekly on Fridays |
| `every 15th of the month` | Monthly on the 15th |
| `once a month on friday midnight` | Monthly on Friday at midnight |
| `every first monday of the month` | First Monday of each month |
| `every last friday of the month` | Last Friday of each month |
| `every other day` | Every 2 days |
| `every 3rd Friday` | Third Friday of each month |
| `every weekday at 9am except Friday` | Weekdays except Friday at 09:00 |
| `every Monday until June` | Weekly with an end bound |
| `every day for 10 days` | Daily with a count bound |
| `every month on the last day` | Monthly on the month end |
| `every quarter on the 15th` | Quarterly on the first month's 15th |

### Timezones

| Input | Description |
|-------|-------------|
| `tomorrow at 9am America/New_York` | Parse in an IANA timezone |
| `now UTC+05:45` | Parse with a numeric UTC offset |
| `tomorrow at 9am EST` | Parse with a fixed-offset abbreviation |

## Grammar and Ambiguity Rules

`naturaldate` intentionally uses a deterministic grammar. When an input is ambiguous, the parser follows these rules instead of guessing from locale or user history:

- **Reference time**: all relative expressions are resolved from `Options.Reference`, or `time.Now()` when `Reference` is zero.
- **Timezone precedence**: a trailing timezone suffix in the input wins over `Options.Location`; otherwise `Options.Location` wins over `Reference.Location()`.
- **Strict by default**: `Parse` requires the whole input to be a date expression. Set `AllowEmbedded` to scan inside longer text.
- **Bare weekdays**: a bare weekday like `monday` uses `Options.WeekdayDir`; the default is `Past`. If a time is attached and the past candidate is before the reference, it moves to the next future occurrence.
- **`last`, `next`, and `this`**: `last` always points backward; `next` and `this` point forward.
- **Bare times**: a standalone time like `10am` resolves to today only when it is after the reference; otherwise it resolves to tomorrow.
- **Named month/day dates**: dates like `March 16` resolve to the same date when it is today, otherwise to the next valid future occurrence. Leap days search up to the next valid leap year.
- **Numeric dates**: `YYYY-MM-DD` is treated as an absolute date. Two-part slash dates prefer US order (`MM/DD`); dash and dot dates prefer day/month order when both sides are ambiguous. If one side is greater than 12, it is treated as the day.
- **Two-digit years**: numeric years below 100 are interpreted as 2000-based years, e.g. `03/16/26` means 2026.
- **Relative suffixes**: `ago` means past. `from` is accepted only as `from now` or `from today`; incomplete forms like `5 days from` are rejected.
- **Business days**: business-day expressions skip Saturdays, Sundays, and any dates in `Options.Holidays`.
- **Custom weekends**: set `Options.WeekendDays` when your working week differs from Monday-Friday.
- **Parts of day**: `morning`, `afternoon`, `evening`, and `tonight` resolve to 09:00, 15:00, 18:00, and 20:00.
- **Business-day end**: `EOD`, `COB`, and `end of business day` use `Options.BusinessDayEnd`, defaulting to 17:00.
- **Fiscal periods**: fiscal quarter/year expressions use `Options.FiscalYearStartMonth`, defaulting to January.
- **Date order**: set `Options.DateOrder` to force MDY, DMY, or YMD parsing for ambiguous numeric dates.
- **Recurrences**: recurring parses return the first occurrence strictly after the reference time. `Result.Next(after)` also returns the first occurrence strictly after `after`.
- **Invalid values**: invalid calendar dates, invalid clock values, zero recurrence intervals, and unsupported trailing words are rejected.

## API Reference

### Parse

```go
func Parse(s string, opts ...Options) (Result, bool)
```

Parses a natural language date string. Returns the parsed result and a boolean indicating success. By default, the full input must be a date expression; use `AllowEmbedded` to scan inside longer text.

```go
result, ok := naturaldate.Parse("yesterday")
if !ok {
    // Handle parse failure
}
```

### MustParse

```go
func MustParse(s string, opts ...Options) Result
```

Like Parse but panics on failure. Use when you're confident the input is valid.

```go
result := naturaldate.MustParse("next monday at 9am")
```

### ParseWithError

```go
func ParseWithError(s string, opts ...Options) (Result, error)
```

Returns a structured `*ParseError` with a failure kind such as empty input, invalid expression, or trailing input.

### ParseAll

```go
func ParseAll(s string, opts ...Options) []Result
func ParseAllWithError(s string, opts ...Options) ([]Result, error)
func AppendAll(dst []Result, s string, opts ...Options) []Result
```

Extracts every date expression from longer text. `AppendAll` lets callers reuse storage.

```go
results := naturaldate.ParseAll("ship tomorrow, follow up in 2 weeks", opts)
```

Each `Result` includes `Start`, `End`, and `Text` metadata for the matched source span.

### ParseRange

```go
func ParseRange(s string, opts ...Options) (DateRange, bool)
func ParseRangeWithError(s string, opts ...Options) (DateRange, error)
```

Parses reporting/scheduling ranges. `DateRange.Start` is inclusive and `DateRange.End` is exclusive.

```go
window, ok := naturaldate.ParseRange("last 7 days", opts)
quarter, ok := naturaldate.ParseRange("Q1 2026", opts)
mtd, err := naturaldate.ParseRangeWithError("month to date", opts)
```

### ParseDuration

```go
func ParseDuration(s string) (time.Duration, bool)
func ParseCalendarDuration(s string) (CalendarDuration, bool)
```

`ParseDuration` handles exact clock-safe durations such as `2h 30m`. `ParseCalendarDuration` also supports calendar units such as months and years.

### Recurrence.Next

```go
func (r Recurrence) Next(after time.Time) (time.Time, bool)
func (r Result) Next(after time.Time) (time.Time, bool)
```

Returns the next occurrence for recurring results.

```go
result, _ := naturaldate.Parse("every monday at 9am", opts)
next, ok := result.Next(time.Now())
```

### Options

```go
type Options struct {
    // Now returns the default reference time when Reference is zero.
    Now func() time.Time

    // Reference is the "now" moment used for relative expressions.
    // Defaults to time.Now() when zero.
    Reference time.Time

    // Location is the timezone used for parsing calendar expressions.
    // When set, Reference is converted into this location before parsing.
    Location *time.Location

    // WeekdayDir is the direction preference when a bare weekday
    // name appears (e.g., "monday"). Default: Past.
    WeekdayDir Direction

    // AllowEmbedded enables scanning the whole input string
    // for date expressions rather than requiring the whole string to be a date.
    AllowEmbedded bool

    // Holidays contains dates to skip for business-day expressions.
    Holidays []time.Time

    // WeekendDays configures non-working weekdays for business-day expressions.
    WeekendDays []time.Weekday

    // BusinessDayEnd is the clock used for COB/EOD expressions.
    BusinessDayEnd Clock

    // FiscalYearStartMonth configures fiscal quarter/year parsing.
    FiscalYearStartMonth time.Month

    // DateOrder controls ambiguous numeric dates.
    DateOrder DateOrder

    // Mode controls tolerance for filler words.
    Mode ParseMode
}
```

### Result

```go
type Result struct {
    Time      time.Time    // Parsed time
    Recur     Recurrence   // Valid when HasRecur is true
    HasRecur  bool         // Whether this is a recurring schedule
    Direction Direction    // Past, Present, or Future
    Truncated Unit         // Finest unit mentioned (e.g., Hour for "10am")
    Start     int          // Byte offset of the matched expression
    End       int          // Byte offset immediately after the match
    Text      string       // Matched source text
    Confidence float64     // 1.0 for deterministic parses
}
```

### Direction

```go
const (
    Past    Direction = -1
    Present Direction = 0
    Future  Direction = 1
)
```

### Unit

```go
const (
    UnitNone   Unit = iota
    UnitSecond          // "10:05:22pm"
    UnitMinute          // "10:05pm"
    UnitHour            // "10am"
    UnitDay             // "today", "tomorrow"
    UnitWeek            // "next week"
    UnitMonth           // "last month"
    UnitYear            // "next year"
    UnitQuarter         // "next quarter", "Q1 2026"
)
```

## Embedded Date Extraction

Use `AllowEmbedded: true` to extract dates from within larger text:

```go
ref := time.Date(2026, time.March, 16, 15, 4, 5, 0, time.UTC)
opts := naturaldate.Options{Reference: ref, AllowEmbedded: true}

result, ok := naturaldate.Parse("restart the server in 5 days from now", opts)
// result.Time = 2026-03-21 (5 days from reference)
```

## Production Notes

- Parsing is deterministic and English-focused.
- Invalid calendar dates are rejected rather than normalized.
- Long embedded inputs are supported with fixed-size token storage and no heap allocations.
- CI runs tests and `go vet` on the oldest supported Go version and the current stable release.

## License

MIT

## Performance

The library is designed for zero-allocation, high-performance parsing. All state is stored on the stack using fixed-size arrays. Run benchmarks on your target hardware with:

```bash
go test -bench=. -benchmem ./...
```

## Error Handling

```go
// Check if parsing succeeded
result, ok := naturaldate.Parse("invalid input")
if !ok {
    // Handle error - result is zero value
}

// Or use MustParse which panics on failure
result := naturaldate.MustParse("next friday")
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT License - see LICENSE file for details.
