---
title: "Chapter 4: Date Utilities"
order: 4
---

# Chapter 4: Date Utilities

## Introduction

Imagine you are scheduling a meeting with a colleague. You do not say "let's meet on 2026-05-13 at 14:00 UTC." You say "let's meet next Wednesday." Humans think in relative, natural time -- today, tomorrow, next week, Tuesday. But computers need exact dates to set reminders, calculate deadlines, and sort lists.

The date utilities package bridges this gap. It lets you type "tomorrow" or "friday" when setting a due date, and it converts that into a precise `time.Time` value that the rest of the application can work with. It also handles the formatting side -- displaying dates in either European (DD-MM-YYYY) or American (MM-DD-YYYY) format depending on your configuration.

## How It Works

The dateutil package provides three main capabilities:

1. **Formatting:** Convert a `time.Time` into a human-readable string, respecting the EU or US convention.
2. **Parsing:** Convert a human-written string (including natural language) into a `time.Time`.
3. **Weekday calculation:** Find the next occurrence of a named weekday.

When you press `n` to create a note in the TUI and type a due date, the input goes through `ParseDate`. It first checks for special keywords ("today," "tomorrow," "next week"), then checks for weekday names ("monday" through "sunday"), and finally tries to parse a formatted date string. This cascading approach means you can use whichever input style feels natural.

## Code Deep Dive

All date utility code lives in `internal/dateutil/dateutil.go`.

### Formatting Dates

The `FormatDate` function converts a `time.Time` to a display string:

```go
// FormatDate formats a time.Time according to the specified format (EU or US).
// Returns empty string for zero time.
// Omits time part if hour and minute are both 0.
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

    // If hour and minute are both 0, omit the time part
    if t.Hour() == 0 && t.Minute() == 0 {
        return t.Format(dateFormat)
    }

    return t.Format(dateFormat + " 15:04")
}
```

The function makes smart decisions about output:
- **Zero time check:** `time.Time{}` (the zero value) returns an empty string. This handles optional dates like `Due` -- when a note has no due date, its `Due` field is the zero value, and formatting it produces nothing rather than "01-01-0001."
- **Time component check:** If the time is midnight (hour and minute both zero), only the date part is shown. This avoids cluttering the display with "00:00" for dates that do not have a specific time.
- **Format awareness:** EU format puts the day first (`02-01-2006`), US format puts the month first (`01-02-2006`). These use Go's reference date -- the specific date January 2, 2006 at 3:04 PM -- where each component has a unique number.

If you are new to Go's time formatting, here is the trick: Go does not use format codes like `%Y-%m-%d`. Instead, you write what the reference date January 2, 2006 would look like in your desired format. `02` = day, `01` = month, `2006` = year, `15` = hour (24h), `04` = minute.

### Formatting Date Only

A companion function strips the time part entirely:

```go
// FormatDateOnly formats only the date part (no time) according to the specified format.
// Returns empty string for zero time.
func FormatDateOnly(t time.Time, format string) string {
    if t.IsZero() {
        return ""
    }

    var dateFormat string
    if strings.ToUpper(format) == "US" {
        dateFormat = "01-02-2006"
    } else {
        dateFormat = "02-01-2006"
    }

    return t.Format(dateFormat)
}
```

This is used in contexts where only the date matters, such as note list displays where showing time would add clutter. While similar to `FormatDate`, having a separate function makes the intent explicit at the call site -- the caller is saying "I only want the date, regardless of what time this holds."

### Why Two Format Functions?

You might wonder why there are two functions when `FormatDate` already omits the time for midnight values. The difference is about **guarantees**. `FormatDate` *conditionally* omits the time -- if the time happens to be 00:00, it omits it. `FormatDateOnly` *unconditionally* omits the time -- even if the time is 14:30, it only shows the date. This matters in contexts like note list columns where consistent date-only formatting is required regardless of the underlying time value.

Here is a concrete example:

| Input Time | `FormatDate` Output | `FormatDateOnly` Output |
|-----------|-------------------|----------------------|
| 2026-05-06 00:00 | `06-05-2026` | `06-05-2026` |
| 2026-05-06 14:30 | `06-05-2026 14:30` | `06-05-2026` |
| zero time | *(empty)* | *(empty)* |

### Parsing Dates with Natural Language

`ParseDate` is the most feature-rich function in the package. It accepts human input and produces a `time.Time`:

```go
// ParseDate parses a date string into a time.Time.
// Supports:
// - "none" or empty string -> returns zero time
// - Natural language: "today", "tomorrow", "next week"
// - Weekday names: "monday", "tuesday", etc. (case insensitive)
// - Date strings in EU or US format with optional time
func ParseDate(input, format string, now time.Time) (time.Time, error) {
    input = strings.TrimSpace(input)

    // Handle "none" and empty string
    if input == "" || strings.ToLower(input) == "none" {
        return time.Time{}, nil
    }

    // Normalize input for case-insensitive matching
    lowerInput := strings.ToLower(input)

    // Natural language parsing
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
```

Let's break down the design decisions:

- **The `now` parameter:** Rather than calling `time.Now()` internally, the function accepts the current time as a parameter. This is a testability pattern -- in tests, you can pass any fixed time and get deterministic results. No mocking `time.Now()` required.
- **"none" handling:** Returning a zero `time.Time` for "none" or empty string lets the caller distinguish between "no date was provided" and "an invalid date was provided" (which returns an error).
- **`time.Date` reconstruction:** Even though `now` already contains a date, the code reconstructs it with `time.Date(...)` to zero out the time components. This ensures "today" means "today at midnight," not "right now at 14:37."

The function continues with weekday name handling:

```go
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
```

Typing "wednesday" gives you the next Wednesday. If today is Wednesday, it gives you next Wednesday (7 days out), not today. This is handled by the `nextWeekday` helper.

Finally, it falls back to parsing formatted date strings:

```go
    // Try parsing as a date string
    var dateFormat, dateTimeFormat string
    if strings.ToUpper(format) == "US" {
        dateFormat = "01-02-2006"
        dateTimeFormat = "01-02-2006 15:04"
    } else {
        dateFormat = "02-01-2006"
        dateTimeFormat = "02-01-2006 15:04"
    }

    // Try with time first
    if strings.Contains(input, " ") {
        if t, err := time.Parse(dateTimeFormat, input); err == nil {
            return time.Date(t.Year(), t.Month(), t.Day(),
                t.Hour(), t.Minute(), 0, 0, time.UTC), nil
        }
    }

    // Try without time
    if t, err := time.Parse(dateFormat, input); err == nil {
        return time.Date(t.Year(), t.Month(), t.Day(),
            0, 0, 0, 0, time.UTC), nil
    }

    return time.Time{}, fmt.Errorf("unable to parse date: %s", input)
}
```

The parsing order is intentional: try the more specific format (date + time) before the less specific one (date only). The space check (`strings.Contains(input, " ")`) is a quick optimization -- no point trying the datetime format if the input has no space.

### The nextWeekday Helper

This private function calculates the next occurrence of a given weekday:

```go
// nextWeekday returns the next occurrence of the specified weekday after now.
// If now is already on the target weekday, it returns the same day next week.
func nextWeekday(now time.Time, targetWeekday time.Weekday) time.Time {
    currentWeekday := now.Weekday()
    daysUntil := int(targetWeekday - currentWeekday)

    // If the target weekday is today or in the past this week, add 7 days
    if daysUntil <= 0 {
        daysUntil += 7
    }

    next := now.AddDate(0, 0, daysUntil)
    return time.Date(next.Year(), next.Month(), next.Day(),
        0, 0, 0, 0, time.UTC)
}
```

The math here is elegant. `time.Weekday` values are integers (Sunday=0 through Saturday=6). Subtracting the current weekday from the target gives the number of days to add. If the result is zero (same day) or negative (already passed this week), adding 7 pushes it to next week.

For example, if today is Wednesday (3) and you want Friday (5): `5 - 3 = 2`, so Friday is 2 days away. If today is Wednesday (3) and you want Monday (1): `1 - 3 = -2`, which is <= 0, so `-2 + 7 = 5` -- Monday is 5 days away.

Here is a table showing all the cases when called on a Wednesday:

| Target Day | Weekday Value | Calculation | Days Until |
|-----------|--------------|-------------|-----------|
| Thursday | 4 | 4 - 3 = 1 | 1 |
| Friday | 5 | 5 - 3 = 2 | 2 |
| Saturday | 6 | 6 - 3 = 3 | 3 |
| Sunday | 0 | 0 - 3 = -3, +7 = 4 | 4 |
| Monday | 1 | 1 - 3 = -2, +7 = 5 | 5 |
| Tuesday | 2 | 2 - 3 = -1, +7 = 6 | 6 |
| Wednesday | 3 | 3 - 3 = 0, +7 = 7 | 7 |

This approach always returns a future date. Typing "wednesday" on a Wednesday gives you the *next* Wednesday, not today. This is the expected behavior for due-date entry -- if you meant today, you would type "today."

### How the Pieces Fit Together

To see the full picture, here is what happens when you type "friday" as a due date while creating a note:

1. The TUI collects your input: `"friday"`
2. It calls `dateutil.ParseDate("friday", "EU", time.Now())`
3. `ParseDate` normalizes to lowercase: `"friday"`
4. The natural language switch does not match (it is not "today", "tomorrow", or "next week")
5. The weekday map matches `"friday"` to `time.Friday`
6. `nextWeekday` calculates the date of the next Friday
7. The resulting `time.Time` is stored in the note's `Due` field
8. When the note is saved, `notes.GenerateTemplate` formats it back to a date string for the frontmatter
9. When the note is displayed, `dateutil.FormatDate` renders it in the user's preferred locale

## Relationships to Other Components

- **Configuration (Chapter 1):** The `DateFormat` config field (`"02-01-2006"` for EU or `"01-02-2006"` for US) determines which format `FormatDate` and `ParseDate` use. The config stores the Go format string directly.
- **Note Model (Chapter 3):** The note model has its own private `parseDate` and `formatDate` helpers that work with Go format strings directly. The dateutil package works at a higher level, accepting `"EU"` or `"US"` as format identifiers and handling natural language. The TUI uses dateutil for user input; the note model uses its own helpers for frontmatter serialization.
- **Business Calendar (Chapter 5):** Dates produced by `ParseDate` are passed to the business calendar to check if they fall on workdays. If a user types "friday" as a due date but Friday is a holiday, the daemon can alert them.
- **TUI:** The note creation dialog uses `ParseDate` for the due date input field, and `FormatDate` / `FormatDateOnly` for displaying dates in the note list and preview pane.

## Key Takeaways

- `ParseDate` accepts natural language ("today," "tomorrow," "next week"), weekday names ("monday" through "sunday"), and formatted date strings -- all in one function.
- The `now` parameter makes the function **deterministically testable** by removing the dependency on the system clock.
- EU format (`DD-MM-YYYY`) and US format (`MM-DD-YYYY`) are both supported, driven by user configuration.
- `FormatDate` intelligently omits the time component when it is midnight, keeping displayed dates clean.
- The `nextWeekday` helper uses modular arithmetic on `time.Weekday` values to find the next occurrence, always returning a future date (never today).

## Next Steps

We now know how pq-notes formats and parses dates. But what about business days? When a note is due "in 5 working days," weekends and holidays should not count. In [Chapter 5: Business Calendar](05-Business-Calendar.md), we will explore how pq-notes builds a country-aware business calendar with holiday support for over 50 countries.
