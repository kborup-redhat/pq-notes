---
title: "Chapter 6: Schedule Parser"
order: 6
---

# Chapter 6: Schedule Parser

Some notes need to repeat. A weekly status report, a monthly invoice reminder, a task that recurs every other Tuesday. The Schedule Parser takes natural language like `"every Monday"` or `"every 2nd-last workday"` and turns it into a structured schedule that the application can reason about.

Think of the Schedule Parser as a translator that speaks two languages: human ("every last workday") and machine (a struct with a kind, a day number, and a function to compute the next occurrence). It bridges the gap so users can type what feels natural, and the system can calculate dates precisely.

## How It Works

The schedule system has three layers:

1. **Data model** -- `RepeatKind` and `Repeat` define the types of recurring schedules
2. **Parser** -- `ParseRepeat()` uses regex patterns to interpret natural language input
3. **Calculator** -- `NextOccurrence()` computes the next date for any schedule, with business calendar awareness

The parser supports seven schedule types, from simple ("daily") to complex ("every 2nd-last workday"). Each type has its own regex pattern and date calculation logic.

## Code Deep Dive

### The RepeatKind Enum

Go doesn't have traditional enums, but the `iota` pattern creates auto-incrementing integer constants:

```go
type RepeatKind int

const (
    Daily          RepeatKind = iota // 0
    Weekly                          // 1
    WeeklyDay                       // 2
    BiWeeklyDay                     // 3
    MonthlyDay                      // 4
    LastWorkday                     // 5
    NthLastWorkday                  // 6
)
```

Each constant gets the next integer value automatically. `Daily` is 0, `Weekly` is 1, and so on. Using a named type (`RepeatKind`) instead of a bare `int` means the compiler will catch mistakes where you accidentally pass a plain integer where a schedule kind is expected.

Here is what each kind means:

| Kind | Example Input | Meaning |
|------|--------------|---------|
| `Daily` | `"daily"` | Every single day |
| `Weekly` | `"weekly"` | Every 7 days from the reference date |
| `WeeklyDay` | `"every Monday"` | Every week on a specific day |
| `BiWeeklyDay` | `"every 2 weeks Tuesday"` | Every other week on a specific day |
| `MonthlyDay` | `"monthly 15"` or `"every 1st"` | A specific day of every month |
| `LastWorkday` | `"every last workday"` | The last business day of each month |
| `NthLastWorkday` | `"every 2nd-last workday"` | The Nth-to-last business day |

### The Repeat Struct

The `Repeat` struct holds the parsed result:

```go
type Repeat struct {
    Kind    RepeatKind
    Weekday time.Weekday
    Day     int    // for MonthlyDay
    N       int    // for NthLastWorkday (e.g., 2 = 2nd-last)
    Raw     string // original input
}
```

Not every field is used by every kind. `Weekday` only matters for `WeeklyDay` and `BiWeeklyDay`. `Day` only matters for `MonthlyDay`. `N` only matters for `NthLastWorkday` (and `LastWorkday`, where it is set to 1). `Raw` preserves the original user input for display purposes -- no information is lost during parsing.

### Regex Patterns

The parser uses six compiled regex patterns. Compiling them at package initialization (via `var`) means the cost is paid once, not on every call:

```go
var (
    reNthLastWorkday = regexp.MustCompile(`(?i)^every\s+(\d+)(?:st|nd|rd|th)-last\s+workday$`)
    reLastWorkday    = regexp.MustCompile(`(?i)^every\s+last\s+workday$`)
    reBiWeeklyDay    = regexp.MustCompile(`(?i)^every\s+2\s+weeks\s+(\w+)$`)
    reEveryWeekday   = regexp.MustCompile(`(?i)^every\s+(\w+)$`)
    reMonthlyDay     = regexp.MustCompile(`(?i)^monthly\s+(\d+)$`)
    reEveryNth       = regexp.MustCompile(`(?i)^every\s+(\d+)(?:st|nd|rd|th)$`)
)
```

A few things to notice:

- **`(?i)`** makes matching case-insensitive, so `"Every Monday"` and `"every monday"` both work.
- **`\s+`** matches one or more whitespace characters, tolerating extra spaces.
- **`(\d+)`** captures a numeric value (day number or N) into a group.
- **`(?:st|nd|rd|th)`** is a non-capturing group that matches ordinal suffixes without storing them.

The weekday lookup table maps lowercase day names to Go's `time.Weekday` values:

```go
var weekdayNames = map[string]time.Weekday{
    "sunday":    time.Sunday,
    "monday":    time.Monday,
    "tuesday":   time.Tuesday,
    "wednesday": time.Wednesday,
    "thursday":  time.Thursday,
    "friday":    time.Friday,
    "saturday":  time.Saturday,
}
```

### The ParseRepeat Function

`ParseRepeat` tries each pattern in a specific order. The order matters -- more specific patterns must be checked before more general ones:

```go
func ParseRepeat(input string) (*Repeat, error) {
    trimmed := strings.TrimSpace(input)
    lower := strings.ToLower(trimmed)

    // "daily"
    if lower == "daily" {
        return &Repeat{Kind: Daily, Raw: input}, nil
    }

    // "weekly"
    if lower == "weekly" {
        return &Repeat{Kind: Weekly, Raw: input}, nil
    }

    // "every Nth-last workday" (must check before "every <weekday>")
    if m := reNthLastWorkday.FindStringSubmatch(trimmed); m != nil {
        n, _ := strconv.Atoi(m[1])
        return &Repeat{Kind: NthLastWorkday, N: n, Raw: input}, nil
    }

    // "every last workday"
    if reLastWorkday.MatchString(trimmed) {
        return &Repeat{Kind: LastWorkday, N: 1, Raw: input}, nil
    }

    // "every 2 weeks <weekday>"
    if m := reBiWeeklyDay.FindStringSubmatch(trimmed); m != nil {
        wd, ok := weekdayNames[strings.ToLower(m[1])]
        if ok {
            return &Repeat{Kind: BiWeeklyDay, Weekday: wd, Raw: input}, nil
        }
    }

    // "every Nth" (e.g., "every 1st", "every 15th")
    if m := reEveryNth.FindStringSubmatch(trimmed); m != nil {
        day, _ := strconv.Atoi(m[1])
        return &Repeat{Kind: MonthlyDay, Day: day, Raw: input}, nil
    }

    // "every <weekday>"
    if m := reEveryWeekday.FindStringSubmatch(trimmed); m != nil {
        wd, ok := weekdayNames[strings.ToLower(m[1])]
        if ok {
            return &Repeat{Kind: WeeklyDay, Weekday: wd, Raw: input}, nil
        }
    }

    // "monthly <day>"
    if m := reMonthlyDay.FindStringSubmatch(trimmed); m != nil {
        day, _ := strconv.Atoi(m[1])
        return &Repeat{Kind: MonthlyDay, Day: day, Raw: input}, nil
    }

    return nil, fmt.Errorf("unrecognised schedule: %q", input)
}
```

The ordering is critical. Consider the input `"every 2nd-last workday"`:

- If `reEveryWeekday` (`every <word>`) were checked first, it would match `"every 2nd-last"` and try to look up `"2nd-last"` as a weekday name. That lookup would fail, but the function might not get a chance to try the correct pattern.
- By checking `reNthLastWorkday` first, the more specific pattern gets priority.

Similarly, `"every 15th"` needs to be checked against `reEveryNth` (monthly) before `reEveryWeekday` (weekly), because `"15th"` is not a weekday name.

### NextOccurrence -- The Date Calculator

Once a schedule is parsed, `NextOccurrence` computes the next date after a given reference time:

```go
func (r *Repeat) NextOccurrence(from time.Time, cal *calendar.BusinessCal) time.Time {
    switch r.Kind {
    case Daily:
        return time.Date(from.Year(), from.Month(), from.Day()+1,
            0, 0, 0, 0, from.Location())

    case Weekly:
        return time.Date(from.Year(), from.Month(), from.Day()+7,
            0, 0, 0, 0, from.Location())

    case WeeklyDay:
        return nextWeekday(from, r.Weekday)

    case BiWeeklyDay:
        first := nextWeekday(from, r.Weekday)
        return first.AddDate(0, 0, 7)

    case MonthlyDay:
        return nextMonthlyDay(from, r.Day)

    case LastWorkday:
        return nextNthLastWorkday(from, 1, cal)

    case NthLastWorkday:
        return nextNthLastWorkday(from, r.N, cal)
    }

    return time.Time{}
}
```

Notice how `LastWorkday` and `NthLastWorkday` both delegate to the same helper but with different N values. `LastWorkday` is just `NthLastWorkday` where N=1. The `cal` parameter (a `BusinessCal`) is what makes workday calculations country-aware -- it knows about weekends and holidays for the user's configured country.

### Helper Functions

Three helper functions handle the date arithmetic:

**nextWeekday** finds the next occurrence of a specific day of the week:

```go
func nextWeekday(from time.Time, wd time.Weekday) time.Time {
    current := from.Weekday()
    daysAhead := int(wd) - int(current)
    if daysAhead <= 0 {
        daysAhead += 7
    }
    return time.Date(from.Year(), from.Month(), from.Day()+daysAhead,
        0, 0, 0, 0, from.Location())
}
```

If today is Wednesday (3) and we want the next Monday (1), that is `1 - 3 = -2`. Since it is negative, we add 7 to get 5. Five days from Wednesday is the following Monday.

**nextMonthlyDay** finds the next occurrence of a specific day-of-month:

```go
func nextMonthlyDay(from time.Time, day int) time.Time {
    candidate := time.Date(from.Year(), from.Month(), day,
        0, 0, 0, 0, from.Location())
    if candidate.After(from) {
        return candidate
    }
    return time.Date(from.Year(), from.Month()+1, day,
        0, 0, 0, 0, from.Location())
}
```

It first tries the current month. If that date has already passed, it rolls forward to the next month. Go's `time.Date` handles month overflow automatically -- `from.Month()+1` when the month is December correctly becomes January of the next year.

**nextNthLastWorkday** finds the Nth-to-last business day of a month:

```go
func nextNthLastWorkday(from time.Time, n int, cal *calendar.BusinessCal) time.Time {
    candidate := cal.NthLastWorkday(from.Year(), from.Month(), n)
    if candidate.After(from) {
        return candidate
    }
    nextMonth := from.Month() + 1
    nextYear := from.Year()
    if nextMonth > 12 {
        nextMonth = 1
        nextYear++
    }
    return cal.NthLastWorkday(nextYear, nextMonth, n)
}
```

This delegates the heavy lifting to `BusinessCal.NthLastWorkday()`, which counts backwards from the end of the month, skipping weekends and holidays. If this month's candidate has already passed, it moves to the next month. The year-rollover logic is explicit here (unlike `nextMonthlyDay`) because the `BusinessCal` method takes separate year and month arguments.

## Relationships

- **BusinessCal** provides `NthLastWorkday()`, which is essential for workday-based schedules. Without it, the parser would have no way to account for weekends and holidays.
- **Note Model** stores the raw schedule string in the `Repeat` field of a note's frontmatter. When displaying upcoming occurrences, the TUI parses this string with `ParseRepeat` and calls `NextOccurrence`.
- **Dashboard** uses `NextOccurrence` to determine which notes are due soon and to sort recurring items by their next occurrence date.

## Key Takeaways

- **`iota`** creates auto-incrementing constants, which is Go's idiomatic approach to enums.
- **Regex ordering matters** -- more specific patterns must be tested before general ones to avoid false matches.
- **Compile regexes once** at package level (`var` block) rather than inside functions to avoid repeated compilation cost.
- **Delegate to a calendar** for business-day calculations rather than hardcoding weekend rules into the schedule parser.
- **Preserve the original input** (`Raw` field) so you can always display what the user typed, even after parsing.

## Next Steps

Now that we can parse schedules and calculate dates, we need somewhere to store notes. In the next chapter, we will build the NoteStore -- the encrypted CRUD layer that creates, reads, updates, and deletes notes on disk.
