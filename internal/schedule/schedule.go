package schedule

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/kborup-redhat/pq-notes/internal/calendar"
)

// RepeatKind identifies the type of repeating schedule.
type RepeatKind int

const (
	Daily RepeatKind = iota
	Weekly
	WeeklyDay
	BiWeeklyDay
	MonthlyDay
	LastWorkday
	NthLastWorkday
)

// Repeat represents a parsed repeating schedule.
type Repeat struct {
	Kind    RepeatKind
	Weekday time.Weekday
	Day     int    // for MonthlyDay
	N       int    // for NthLastWorkday (e.g., 2 = 2nd-last)
	Raw     string // original input
}

var (
	reNthLastWorkday = regexp.MustCompile(`(?i)^every\s+(\d+)(?:st|nd|rd|th)-last\s+workday$`)
	reLastWorkday    = regexp.MustCompile(`(?i)^every\s+last\s+workday$`)
	reBiWeeklyDay    = regexp.MustCompile(`(?i)^every\s+2\s+weeks\s+(\w+)$`)
	reEveryWeekday   = regexp.MustCompile(`(?i)^every\s+(\w+)$`)
	reMonthlyDay     = regexp.MustCompile(`(?i)^monthly\s+(\d+)$`)
	reEveryNth       = regexp.MustCompile(`(?i)^every\s+(\d+)(?:st|nd|rd|th)$`)
)

// weekdayNames maps lowercase day names to time.Weekday values.
var weekdayNames = map[string]time.Weekday{
	"sunday":    time.Sunday,
	"monday":    time.Monday,
	"tuesday":   time.Tuesday,
	"wednesday": time.Wednesday,
	"thursday":  time.Thursday,
	"friday":    time.Friday,
	"saturday":  time.Saturday,
}

// ParseRepeat parses a schedule string into a Repeat.
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

// NextOccurrence returns the next occurrence of this schedule after from.
func (r *Repeat) NextOccurrence(from time.Time, cal *calendar.BusinessCal) time.Time {
	switch r.Kind {
	case Daily:
		return time.Date(from.Year(), from.Month(), from.Day()+1, 0, 0, 0, 0, from.Location())

	case Weekly:
		return time.Date(from.Year(), from.Month(), from.Day()+7, 0, 0, 0, 0, from.Location())

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

// nextWeekday returns the next date after from that falls on the given weekday.
func nextWeekday(from time.Time, wd time.Weekday) time.Time {
	current := from.Weekday()
	daysAhead := int(wd) - int(current)
	if daysAhead <= 0 {
		daysAhead += 7
	}
	return time.Date(from.Year(), from.Month(), from.Day()+daysAhead, 0, 0, 0, 0, from.Location())
}

// nextMonthlyDay returns the next occurrence of the given day-of-month after from.
func nextMonthlyDay(from time.Time, day int) time.Time {
	candidate := time.Date(from.Year(), from.Month(), day, 0, 0, 0, 0, from.Location())
	if candidate.After(from) {
		return candidate
	}
	return time.Date(from.Year(), from.Month()+1, day, 0, 0, 0, 0, from.Location())
}

// nextNthLastWorkday returns the nth-last workday of the current month if it's
// still after from; otherwise it returns the nth-last workday of the next month.
func nextNthLastWorkday(from time.Time, n int, cal *calendar.BusinessCal) time.Time {
	candidate := cal.NthLastWorkday(from.Year(), from.Month(), n)
	if candidate.After(from) {
		return candidate
	}
	// Move to next month
	nextMonth := from.Month() + 1
	nextYear := from.Year()
	if nextMonth > 12 {
		nextMonth = 1
		nextYear++
	}
	return cal.NthLastWorkday(nextYear, nextMonth, n)
}
