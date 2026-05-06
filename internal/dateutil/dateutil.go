package dateutil

import (
	"fmt"
	"strings"
	"time"
)

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
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC), nil
	case "tomorrow":
		tomorrow := now.AddDate(0, 0, 1)
		return time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 0, 0, 0, 0, time.UTC), nil
	case "next week":
		nextWeek := now.AddDate(0, 0, 7)
		return time.Date(nextWeek.Year(), nextWeek.Month(), nextWeek.Day(), 0, 0, 0, 0, time.UTC), nil
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
			return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), 0, 0, time.UTC), nil
		}
	}

	// Try without time
	if t, err := time.Parse(dateFormat, input); err == nil {
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), nil
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", input)
}

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
	return time.Date(next.Year(), next.Month(), next.Day(), 0, 0, 0, 0, time.UTC)
}
