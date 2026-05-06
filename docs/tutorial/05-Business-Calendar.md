---
title: "Chapter 5: Business Calendar"
order: 5
---

# Chapter 5: Business Calendar

## Introduction

Imagine you are a project manager and you tell your team, "This is due in 10 days." Simple enough -- until you realize that those 10 days include a weekend, a national holiday, and your company's founding day celebration. The actual deadline is not 10 calendar days away; it is 10 *working* days away, which might be two full weeks on the calendar.

This is the problem the business calendar solves. It understands which days are workdays and which are not, taking into account weekends (which vary by country), national holidays (which vary dramatically across 50+ countries), and custom holidays (which vary by organization). When pq-notes calculates due dates, checks for overdue notes, or runs the notification daemon, it uses this calendar to ensure deadlines respect business reality.

## How It Works

The business calendar wraps the [rickar/cal](https://github.com/rickar/cal) library, which provides a `BusinessCalendar` type with holiday-aware workday calculations. pq-notes configures it in three layers:

1. **Weekend configuration:** Sets which days of the week are non-working days, based on the user's config. Saturday and Sunday for most countries, Friday and Saturday for countries like Saudi Arabia and Israel.
2. **National holidays:** Loads the full holiday list for the user's country using the `cal` library's built-in country packages. These include fixed dates (January 1st), moveable dates (Easter Monday), and observation rules (if Christmas falls on Sunday, Monday is the holiday).
3. **Custom holidays:** Adds any organization-specific holidays the user has defined in their config.

The result is a `BusinessCal` object that can answer three questions: "Is this date a workday?", "What date is N workdays from now?", and "What is the Nth-last workday of a given month?"

## Code Deep Dive

All business calendar code lives in `internal/calendar/calendar.go`.

### The BusinessCal Wrapper

The package defines a thin wrapper around `cal.BusinessCalendar`:

```go
// BusinessCal wraps a cal.BusinessCalendar with country-specific holidays
// and configurable weekend days.
type BusinessCal struct {
    bc *cal.BusinessCalendar
}
```

This wrapper is not just for show. It serves as an **abstraction boundary**: the rest of the application interacts with `BusinessCal` and never imports the `cal` package directly. If the underlying library changes or needs to be replaced, only this file needs updating.

### Construction from Config

The `New` function takes a `*config.Config` and builds a fully configured business calendar:

```go
// New creates a BusinessCal from the application config.
func New(cfg *config.Config) *BusinessCal {
    bc := cal.NewBusinessCalendar()

    // Configure weekend days
    for _, day := range []time.Weekday{
        time.Monday, time.Tuesday, time.Wednesday,
        time.Thursday, time.Friday, time.Saturday, time.Sunday,
    } {
        bc.SetWorkday(day, !isWeekend(day, cfg.Weekend))
    }

    addCountryHolidays(bc, cfg.Country)
    addCustomHolidays(bc, cfg.CustomHolidays)

    return &BusinessCal{bc: bc}
}
```

The weekend configuration is interesting. Rather than only marking the weekend days, it iterates through *all seven days* and explicitly sets each one as either a workday or not. This is defensive programming -- it ensures the calendar starts with a clean state regardless of what defaults `cal.NewBusinessCalendar()` might set.

The `isWeekend` helper does a case-insensitive string comparison:

```go
func isWeekend(day time.Weekday, weekendDays []string) bool {
    dayName := strings.ToLower(day.String())
    for _, wd := range weekendDays {
        if strings.ToLower(wd) == dayName {
            return true
        }
    }
    return false
}
```

This converts `time.Saturday.String()` (which returns `"Saturday"`) to lowercase and compares it against the config's weekend list (which stores `"saturday"`). The case-insensitive comparison is a small but important detail -- user configuration should not break because someone typed "Saturday" instead of "saturday."

### Workday Queries

The three public methods expose the calendar's functionality:

```go
// IsWorkday reports whether the given time falls on a working day.
func (c *BusinessCal) IsWorkday(t time.Time) bool {
    return c.bc.IsWorkday(t)
}
```

`IsWorkday` checks both the day of the week (is it a weekend?) and the date (is it a holiday?). The notification daemon uses this to decide whether to send reminders -- if today is not a workday, a note due "today" might not need an urgent notification.

```go
// WorkdaysFrom returns the date that is offset working days from start.
func (c *BusinessCal) WorkdaysFrom(start time.Time, offset int) time.Time {
    return c.bc.WorkdaysFrom(start, offset)
}
```

`WorkdaysFrom` is used for repeating notes and offset calculations. If a task repeats "every 5 workdays," this function takes the last occurrence and adds 5 working days, skipping weekends and holidays.

```go
// NthLastWorkday returns the nth-last working day in the given month.
// For example, NthLastWorkday(2026, May, 1) returns the last workday of May 2026.
// Returns the zero value of time.Time if n exceeds available workdays.
func (c *BusinessCal) NthLastWorkday(year int, month time.Month, n int) time.Time {
    // Start from last day of month and walk backwards counting workdays
    end := time.Date(year, month+1, 0, 0, 0, 0, 0, time.Local)
    count := 0
    for d := end; d.Month() == month; d = d.AddDate(0, 0, -1) {
        if c.bc.IsWorkday(d) {
            count++
            if count == n {
                return d
            }
        }
    }
    return time.Time{}
}
```

`NthLastWorkday` is the most specialized method. It answers questions like "What is the last workday of the month?" (n=1) or "What is the second-to-last workday?" (n=2). This is useful for financial tasks (month-end reports, payroll) and recurring reminders.

The implementation uses a well-known Go trick for finding the last day of a month: `time.Date(year, month+1, 0, ...)` gives you day 0 of the next month, which Go normalizes to the last day of the current month. From there, it walks backward day by day, counting workdays until it reaches the desired position.

### Country-Specific Holidays

The `countryHolidays` function is a large switch statement mapping ISO country codes to holiday lists:

```go
func countryHolidays(country string) []*cal.Holiday {
    switch strings.ToUpper(country) {
    case "AR":
        return ar.Holidays
    case "AT":
        return at.Holidays
    case "AU":
        return au.HolidaysNSW
    case "BE":
        return be.Holidays
    case "BG":
        return bg.Holidays
    case "BR":
        return br.Holidays
    case "CA":
        return ca.Holidays
    case "CH":
        return ch.Holidays
    // ... 40+ more countries ...
    case "US":
        return us.Holidays
    case "ZA":
        return za.Holidays
    default:
        // Basic set for unknown countries
        return []*cal.Holiday{
            {
                Name:  "New Year's Day",
                Type:  cal.ObservancePublic,
                Month: time.January,
                Day:   1,
                Func:  cal.CalcDayOfMonth,
            },
            {
                Name:  "Christmas Day",
                Type:  cal.ObservancePublic,
                Month: time.December,
                Day:   25,
                Func:  cal.CalcDayOfMonth,
            },
        }
    }
}
```

The supported countries span the globe: Argentina, Austria, Australia, Belgium, Bulgaria, Brazil, Canada, Switzerland, Cyprus, Czech Republic, Germany, Denmark, Estonia, Spain, Finland, France, Great Britain, Greece, Croatia, Hungary, Ireland, Iceland, Italy, Japan, Kenya, Lithuania, Luxembourg, Latvia, Malta, Malawi, Mexico, Netherlands, Norway, New Zealand, Poland, Portugal, Romania, Serbia, Russia, Sweden, Slovenia, Slovakia, Thailand, Ukraine, United States, and South Africa.

The default case provides a minimal fallback for countries without specific support -- New Year's Day and Christmas are nearly universal. This means the calendar always provides *some* holiday awareness, even for unsupported countries.

The `addCountryHolidays` helper feeds these into the business calendar:

```go
func addCountryHolidays(bc *cal.BusinessCalendar, country string) {
    holidays := countryHolidays(country)
    bc.AddHoliday(holidays...)
}
```

### Custom Holidays

Beyond national holidays, users can define their own:

```go
func addCustomHolidays(bc *cal.BusinessCalendar, holidays []config.CustomHoliday) {
    for _, ch := range holidays {
        parts := strings.Split(ch.Date, "-")
        if len(parts) != 2 {
            continue
        }
        var day, month int
        if _, err := fmt.Sscanf(parts[0], "%d", &day); err != nil {
            continue
        }
        if _, err := fmt.Sscanf(parts[1], "%d", &month); err != nil {
            continue
        }
        if day < 1 || day > 31 || month < 1 || month > 12 {
            continue
        }
        h := &cal.Holiday{
            Name:  ch.Name,
            Type:  cal.ObservanceOther,
            Month: time.Month(month),
            Day:   day,
            Func:  cal.CalcDayOfMonth,
        }
        bc.AddHoliday(h)
    }
}
```

Custom holidays use `DD-MM` format (day-month, no year) so they repeat annually. The parsing is deliberately defensive:

- If the date does not split into exactly 2 parts, skip it.
- If either part fails to parse as an integer, skip it.
- If the day or month is out of range, skip it.
- Only valid entries get added.

This `continue`-based error handling is a pattern for processing lists where individual failures should not abort the whole operation. A malformed custom holiday in the config file is silently skipped rather than crashing the application.

The holiday type `cal.ObservanceOther` distinguishes custom holidays from the national ones (`cal.ObservancePublic`), though both are treated equally for workday calculations.

For reference, a config with custom holidays looks like this:

```yaml
custom_holidays:
  - name: "Company Founding Day"
    date: "15-06"
  - name: "Summer Party"
    date: "21-08"
```

## Relationships to Other Components

- **Configuration (Chapter 1):** The `New()` constructor takes a `*config.Config` directly, reading `Country` for national holidays, `Weekend` for weekend days, and `CustomHolidays` for organization-specific days off. The `DefaultWeekend()` function in the config package sets the initial weekend days based on country.
- **Date Utilities (Chapter 4):** Dates parsed by `dateutil.ParseDate` (like "next friday") are checked against the business calendar. The two packages work together: dateutil resolves *what date the user means*, and the calendar determines *whether that date is a workday*.
- **Note Model (Chapter 3):** The `Repeat` field in notes (e.g., "weekly," "monthly") uses the business calendar to calculate the next occurrence, skipping non-working days.
- **Notification Daemon:** The daemon uses `IsWorkday` to decide whether to send notifications on a given day, and `WorkdaysFrom` to calculate how many working days until a note is due.
- **TUI:** The note list displays a visual indicator when notes are overdue, using the business calendar to determine if the due date has passed in business-day terms.

## Key Takeaways

- The business calendar combines three layers: weekend configuration, national holidays (50+ countries), and custom holidays.
- `NthLastWorkday` uses Go's month-boundary trick (`time.Date(year, month+1, 0, ...)`) to find the last day of any month, then walks backward counting workdays.
- Defensive parsing in `addCustomHolidays` silently skips malformed entries rather than crashing -- a good pattern for user-provided configuration.
- The `BusinessCal` wrapper provides an abstraction boundary: the rest of the codebase never imports the `cal` library directly.
- Weekend configuration iterates all 7 days explicitly, ensuring a clean state regardless of library defaults.

## Where to Go From Here

You have now walked through the five foundational packages of pq-notes:

1. **Configuration** -- how the app stores and loads settings
2. **Cryptography** -- how notes are encrypted with post-quantum hybrid keys
3. **Note Model** -- how notes are structured, parsed, and generated
4. **Date Utilities** -- how dates are formatted and parsed with natural language
5. **Business Calendar** -- how workdays are calculated with holiday awareness

These packages form the core that everything else builds on. The TUI (Bubble Tea), Drive sync (Google Drive API), note sharing, and notification daemon all compose these building blocks. With your understanding of the foundations, exploring those higher-level features in the source code should feel familiar -- they follow the same patterns of clean interfaces, defensive error handling, and configuration-driven behavior.
