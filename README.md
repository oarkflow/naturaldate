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

### Weekday References

| Input | Description |
|-------|-------------|
| `last monday` | Most recent Monday (past) |
| `next friday` | Upcoming Friday (future) |
| `this wednesday` | Upcoming Wednesday |
| `sunday` | Nearest Sunday (configurable direction) |
| `monday at 10am` | Next Monday at 10:00 |
| `last sunday at 5:30pm` | Last Sunday at 17:30 |

### Month/Year References

| Input | Description |
|-------|-------------|
| `last month` | First day of previous month |
| `next month` | First day of next month |
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
| `midnight` | 00:00 of current day |
| `noon` | 12:00 of current day |

### Numeric Dates

| Input | Description |
|-------|-------------|
| `2026-12-25` | Full ISO date |
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
| `every 2 weeks on friday` | Bi-weekly on Fridays |
| `once a month on friday midnight` | Monthly on Friday at midnight |
| `every first monday of the month` | First Monday of each month |
| `every last friday of the month` | Last Friday of each month |

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

### ParseAll

```go
func ParseAll(s string, opts ...Options) []Result
func AppendAll(dst []Result, s string, opts ...Options) []Result
```

Extracts every date expression from longer text. `AppendAll` lets callers reuse storage.

```go
results := naturaldate.ParseAll("ship tomorrow, follow up in 2 weeks", opts)
```

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

- Invalid calendar dates are rejected rather than normalized.
- Long embedded inputs are supported with fixed-size token storage and no heap allocations.
- CI runs tests and `go vet` on the oldest supported Go version and the current stable release.

## License

MIT

## Performance

The library is designed for zero-allocation, high-performance parsing. All state is stored on the stack using fixed-size arrays. Here's a quick benchmark:

```go
// Benchmark results on modern hardware
// BenchmarkParseNow-16        100000000         10.5 ns/op
// BenchmarkParseTime-16        100000000         12.1 ns/op
// BenchmarkParseEmbedded-16    50000000          25.3 ns/op
// BenchmarkParseRecurring-16   50000000          28.7 ns/op
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
