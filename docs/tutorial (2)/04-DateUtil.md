---
title: "Chapter 4: DateUtil"
order: 4
---

# Chapter 4: Date Utilities

Dates are everywhere in a notes application — creation timestamps, due dates, recurring schedules. Users don't want to type `06-05-2026` every time; they want to type "tomorrow" or "friday" and have the application figure it out. The DateUtil package bridges the gap between human-friendly date expressions and Go's `time.Time` type.

Think of DateUtil as a translator that speaks both "human date" and "computer date" fluently.

## How It Works

The package provides three capabilities:

1. **Format dates** for display — converting `time.Time` to strings in EU (`DD-MM-YYYY`) or US (`MM-DD-YYYY`) format
2. **Parse dates** from user input — accepting formal dates, natural language, and weekday names
3. **Handle optional time** — omitting the `HH:MM` portion when it's midnight (meaning "all day")

## Code Deep Dive

### Formatting Dates

The `FormatDate` function converts a `time.Time` into a human-readable string:

```go
func FormatDate(t time.Time, format string) string {
    if t.IsZero() {
        return ""
    }

    var dateFormat string
    if strings.ToUpper(format) == "US" {
        dateFormat = "01-02-2006"
    } else {
        dateFormat = "02-01-2006"
    }

    if t.Hour() == 0 && t.Minute() == 0 {
        return t.Format(dateFormat)
    }

    return t.Format(dateFormat + " 15:04")
}
```

Go's date formatting uses a **reference time** — the specific date `Mon Jan 2 15:04:05 MST 2006`. Instead of abstract format codes like `%Y-%m-%d`, you arrange the components of this reference time in your desired layout:

| Component | Reference Value | Meaning |
|---|---|---|
| `01` | January | Month (zero-padded) |
| `02` | 2nd | Day (zero-padded) |
| `2006` | 2006 | Year (4-digit) |
| `15` | 3 PM | Hour (24-hour) |
| `04` | 4 minutes | Minute (zero-padded) |

So `"02-01-2006"` means day-month-year (EU format), and `"01-02-2006"` means month-day-year (US format).

The function smartly omits the time component when both hour and minute are zero — this means "all day" events and date-only fields display cleanly as `06-05-2026` instead of `06-05-2026 00:00`.

A companion function `FormatDateOnly` always omits the time, useful when you specifically want just the date portion.

### Parsing Dates

The `ParseDate` function is the most complex part of the package, handling multiple input formats:

```go
func ParseDate(input, format string, now time.Time) (time.Time, error) {
    input = strings.TrimSpace(input)

    if input == "" || strings.ToLower(input) == "none" {
        return time.Time{}, nil
    }

    lowerInput := strings.ToLower(input)

    switch lowerInput {
    case "today":
        return time.Date(now.Year(), now.Month(), now.Day(),
            0, 0, 0, 0, time.UTC), nil
    case "tomorrow":
        tomorrow := now.AddDate(0, 0, 1)
        return time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(),
            0, 0, 0, 0, time.UTC), nil
    case "next week":
        nextWeek := now.AddDate(0, 0, 7)
        return time.Date(nextWeek.Year(), nextWeek.Month(), nextWeek.Day(),
            0, 0, 0, 0, time.UTC), nil
    }

    // Weekday names
    weekdayMap := map[string]time.Weekday{
        "sunday":    time.Sunday,
        "monday":    time.Monday,
        "tuesday":   time.Tuesday,
        "wednesday": time.Wednesday,
        "thursday":  time.Thursday,
        "friday":    time.Friday,
        "saturday":  time.Saturday,
    }

    if targetWeekday, ok := weekdayMap[lowerInput]; ok {
        return nextWeekday(now, targetWeekday), nil
    }

    // Try parsing as a date string
    // ...
}
```

The function tries inputs in order of specificity:

1. **Empty or "none"** — returns the zero value of `time.Time`, meaning "no date"
2. **Natural language** — "today", "tomorrow", "next week" are resolved relative to `now`
3. **Weekday names** — "monday", "friday", etc. resolve to the *next* occurrence of that day
4. **Formatted dates** — tried with time first (`DD-MM-YYYY HH:MM`), then without (`DD-MM-YYYY`)

The `now` parameter is injected rather than calling `time.Now()` directly. This is a **testability pattern** — by passing the current time as a parameter, unit tests can use a fixed point in time and get deterministic results.

### Finding the Next Weekday

When the user types a day name like "friday", we need to find the next occurrence:

```go
func nextWeekday(now time.Time, targetWeekday time.Weekday) time.Time {
    currentWeekday := now.Weekday()
    daysUntil := int(targetWeekday - currentWeekday)

    if daysUntil <= 0 {
        daysUntil += 7
    }

    next := now.AddDate(0, 0, daysUntil)
    return time.Date(next.Year(), next.Month(), next.Day(),
        0, 0, 0, 0, time.UTC)
}
```

The math is elegant:
- Subtract the current weekday number from the target weekday number
- If the result is zero or negative (today or earlier this week), add 7 to get next week's occurrence
- Use `time.Date()` to construct a clean midnight time, stripping any time-of-day components

For example, if today is Wednesday (3) and the target is Friday (5): `5 - 3 = 2` days ahead. If the target is Monday (1): `1 - 3 = -2`, then `-2 + 7 = 5` days ahead (next Monday).

### Date String Parsing

When natural language doesn't match, the function tries formal date formats:

```go
var dateFormat, dateTimeFormat string
if strings.ToUpper(format) == "US" {
    dateFormat = "01-02-2006"
    dateTimeFormat = "01-02-2006 15:04"
} else {
    dateFormat = "02-01-2006"
    dateTimeFormat = "02-01-2006 15:04"
}

if strings.Contains(input, " ") {
    if t, err := time.Parse(dateTimeFormat, input); err == nil {
        return time.Date(t.Year(), t.Month(), t.Day(),
            t.Hour(), t.Minute(), 0, 0, time.UTC), nil
    }
}

if t, err := time.Parse(dateFormat, input); err == nil {
    return time.Date(t.Year(), t.Month(), t.Day(),
        0, 0, 0, 0, time.UTC), nil
}
```

The function tries the most specific format first (with time), then falls back to date-only. This "try and recover" pattern is common in Go — rather than validating the format upfront, you attempt to parse and handle the error.

## Relationships

- **Note** model uses date formatting when generating frontmatter for notes.
- **NoteStore** uses date layout strings derived from the format setting to parse and format dates within notes.
- **Config** provides the `DateFormat` setting ("EU" or "US") that controls which format is used.

## Key Takeaways

- Go's **reference time** format (`Mon Jan 2 15:04:05 MST 2006`) is unique — memorize it or look it up.
- **Inject `time.Now()`** as a parameter for testable date logic instead of calling it directly.
- **Try-and-recover** parsing (attempt parse, check error) is idiomatic Go for handling multiple formats.
- **Zero values** (`time.Time{}`) are meaningful — they represent "no date set" rather than an error.
- **Case-insensitive matching** via `strings.ToLower` makes user input forgiving.

## Next Steps

Users need to actually *write* their notes. In the next chapter, we'll build the editor integration that opens notes in the user's preferred terminal editor.
