---
title: "Chapter 7: Business Calendar"
order: 7
---

# Chapter 7: Business Calendar

When a task is due "in 5 business days," you don't mean 5 calendar days — you mean 5 working days, skipping weekends and holidays. The BusinessCal package provides this capability, wrapping the `rickar/cal` library with country-specific holiday sets and configurable weekend days.

Think of it as a smart calendar that knows about public holidays in 40+ countries and can answer questions like "what's the third-to-last working day this month?"

## How It Works

The BusinessCal package:

1. **Configures weekend days** — typically Saturday/Sunday, but Friday/Saturday in some countries
2. **Loads country holidays** — public holidays from a library of 40+ countries
3. **Adds custom holidays** — company-specific or personal days off
4. **Provides workday calculations** — "is this a workday?", "what date is N workdays from now?", "what's the Nth-last workday of the month?"

## Code Deep Dive

### The BusinessCal Struct

```go
type BusinessCal struct {
    bc *cal.BusinessCalendar
}

func New(cfg *config.Config) *BusinessCal {
    bc := cal.NewBusinessCalendar()

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

The constructor reads the Config and configures the underlying `cal.BusinessCalendar`:

1. **Set each weekday** as either a workday or non-workday based on the configured weekend
2. **Add country holidays** from the pre-built holiday sets
3. **Add custom holidays** from the user's config

The `isWeekend` helper does a case-insensitive match:

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

### Country Holiday Sets

The `countryHolidays` function maps ISO 3166-1 alpha-2 country codes to pre-built holiday lists:

```go
func countryHolidays(country string) []*cal.Holiday {
    switch strings.ToUpper(country) {
    case "AR":
        return ar.Holidays
    case "AT":
        return at.Holidays
    case "AU":
        return au.HolidaysNSW
    // ... 40+ countries ...
    case "DK":
        return dk.Holidays
    // ...
    case "US":
        return us.Holidays
    default:
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

The `default` case provides a minimal fallback — New Year's Day and Christmas — for countries not in the library. This ensures the calendar is never completely empty.

Each country's holiday set is imported from its own sub-package (e.g., `github.com/rickar/cal/v2/dk` for Denmark). The library handles complex rules like Easter-dependent holidays (which move every year), observed holidays (when a holiday falls on a weekend), and regional variations.

### Custom Holidays

Users can define additional holidays in their config for company-specific days off:

```go
func addCustomHolidays(bc *cal.BusinessCalendar, holidays []config.CustomHoliday) {
    for _, ch := range holidays {
        parts := strings.Split(ch.Date, "-")
        if len(parts) != 2 {
            continue
        }
        var day, month int
        fmt.Sscanf(parts[0], "%d", &day)
        fmt.Sscanf(parts[1], "%d", &month)
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

Custom holidays are defined in `DD-MM` format in the config:

```yaml
custom_holidays:
  - name: "Company Anniversary"
    date: "15-03"
  - name: "Team Building Day"
    date: "22-09"
```

The function parses the date string, converts it to a `cal.Holiday`, and adds it to the calendar. Invalid dates (wrong format) are silently skipped.

### Workday Operations

The BusinessCal exposes three operations:

```go
func (c *BusinessCal) IsWorkday(t time.Time) bool {
    return c.bc.IsWorkday(t)
}

func (c *BusinessCal) WorkdaysFrom(start time.Time, offset int) time.Time {
    return c.bc.WorkdaysFrom(start, offset)
}
```

- **`IsWorkday`** — checks if a date is a working day (not a weekend, not a holiday)
- **`WorkdaysFrom`** — adds N working days to a start date, skipping weekends and holidays

The most interesting operation is `NthLastWorkday`:

```go
func (c *BusinessCal) NthLastWorkday(year int, month time.Month, n int) time.Time {
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

This finds the Nth-last working day of a month. The algorithm:

1. Start at the last day of the month (`month+1, day 0` is Go's way of getting the last day of the previous month)
2. Walk backwards, counting workdays
3. Return when we've found the Nth workday

**Use case:** "Remind me on the second-to-last business day of each month to submit expense reports." Call `NthLastWorkday(2026, time.May, 2)` to get that date.

## Relationships

- **Config** provides the country code, weekend days, and custom holidays that configure the calendar.
- The calendar is used alongside **NoteStore** operations — for example, calculating due dates for tasks or scheduling reminders on business days.

## Key Takeaways

- **Wrap third-party libraries** in your own types to create a simpler, project-specific API.
- **`time.Date(year, month+1, 0, ...)`** is the Go idiom for "last day of the given month."
- **Fallback defaults** (New Year + Christmas) ensure the system works even with unrecognized country codes.
- **Country-specific configuration** shows how a simple 2-letter code can drive significant behavioral differences.
- **Silent skip** for invalid custom holidays keeps the system resilient without requiring strict validation at config time.

## Next Steps

In the final chapter, we'll wire everything together with the Cobra CLI framework, creating the command-line interface that users interact with.
