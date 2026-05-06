package schedule

import (
	"testing"
	"time"

	"github.com/kborup-redhat/pq-notes/internal/calendar"
	"github.com/kborup-redhat/pq-notes/internal/config"
)

func testCal() *calendar.BusinessCal {
	return calendar.New(&config.Config{
		Country: "DK",
		Weekend: []string{"saturday", "sunday"},
	})
}

func TestParseRepeat(t *testing.T) {
	tests := []struct {
		input string
		kind  RepeatKind
	}{
		{"daily", Daily},
		{"weekly", Weekly},
		{"monthly 15", MonthlyDay},
		{"every monday", WeeklyDay},
		{"every 1st", MonthlyDay},
		{"every last workday", LastWorkday},
		{"every 2nd-last workday", NthLastWorkday},
	}
	for _, tt := range tests {
		r, err := ParseRepeat(tt.input)
		if err != nil {
			t.Errorf("ParseRepeat(%q): %v", tt.input, err)
			continue
		}
		if r.Kind != tt.kind {
			t.Errorf("ParseRepeat(%q): expected kind %v, got %v", tt.input, tt.kind, r.Kind)
		}
	}
}

func TestParseRepeatCaseInsensitive(t *testing.T) {
	tests := []struct {
		input string
		kind  RepeatKind
	}{
		{"Daily", Daily},
		{"WEEKLY", Weekly},
		{"Every Monday", WeeklyDay},
		{"EVERY LAST WORKDAY", LastWorkday},
	}
	for _, tt := range tests {
		r, err := ParseRepeat(tt.input)
		if err != nil {
			t.Errorf("ParseRepeat(%q): %v", tt.input, err)
			continue
		}
		if r.Kind != tt.kind {
			t.Errorf("ParseRepeat(%q): expected kind %v, got %v", tt.input, tt.kind, r.Kind)
		}
	}
}

func TestParseRepeatInvalid(t *testing.T) {
	_, err := ParseRepeat("nonsense")
	if err == nil {
		t.Error("expected error for invalid input, got nil")
	}
}

func TestParseRepeatWeekday(t *testing.T) {
	r, err := ParseRepeat("every friday")
	if err != nil {
		t.Fatalf("ParseRepeat: %v", err)
	}
	if r.Weekday != time.Friday {
		t.Errorf("expected Friday, got %v", r.Weekday)
	}
}

func TestParseRepeatMonthlyDay(t *testing.T) {
	r, err := ParseRepeat("monthly 15")
	if err != nil {
		t.Fatalf("ParseRepeat: %v", err)
	}
	if r.Day != 15 {
		t.Errorf("expected Day=15, got %v", r.Day)
	}
}

func TestParseRepeatEveryNth(t *testing.T) {
	r, err := ParseRepeat("every 1st")
	if err != nil {
		t.Fatalf("ParseRepeat: %v", err)
	}
	if r.Kind != MonthlyDay {
		t.Errorf("expected MonthlyDay, got %v", r.Kind)
	}
	if r.Day != 1 {
		t.Errorf("expected Day=1, got %v", r.Day)
	}
}

func TestParseRepeatNthLastWorkday(t *testing.T) {
	r, err := ParseRepeat("every 2nd-last workday")
	if err != nil {
		t.Fatalf("ParseRepeat: %v", err)
	}
	if r.N != 2 {
		t.Errorf("expected N=2, got %v", r.N)
	}
}

func TestParseRepeatLastWorkdayN(t *testing.T) {
	r, err := ParseRepeat("every last workday")
	if err != nil {
		t.Fatalf("ParseRepeat: %v", err)
	}
	if r.N != 1 {
		t.Errorf("expected N=1, got %v", r.N)
	}
}

func TestParseRepeatBiWeekly(t *testing.T) {
	r, err := ParseRepeat("every 2 weeks friday")
	if err != nil {
		t.Fatalf("ParseRepeat: %v", err)
	}
	if r.Kind != BiWeeklyDay {
		t.Errorf("expected BiWeeklyDay, got %v", r.Kind)
	}
	if r.Weekday != time.Friday {
		t.Errorf("expected Friday, got %v", r.Weekday)
	}
}

func TestParseRepeatRawPreserved(t *testing.T) {
	r, err := ParseRepeat("Every Monday")
	if err != nil {
		t.Fatalf("ParseRepeat: %v", err)
	}
	if r.Raw != "Every Monday" {
		t.Errorf("expected Raw=%q, got %q", "Every Monday", r.Raw)
	}
}

func TestNextOccurrenceDaily(t *testing.T) {
	r, _ := ParseRepeat("daily")
	cal := testCal()
	from := time.Date(2026, 5, 6, 0, 0, 0, 0, time.Local)
	next := r.NextOccurrence(from, cal)
	expected := time.Date(2026, 5, 7, 0, 0, 0, 0, time.Local)
	if !next.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, next)
	}
}

func TestNextOccurrenceWeekly(t *testing.T) {
	r, _ := ParseRepeat("weekly")
	cal := testCal()
	from := time.Date(2026, 5, 6, 0, 0, 0, 0, time.Local)
	next := r.NextOccurrence(from, cal)
	expected := time.Date(2026, 5, 13, 0, 0, 0, 0, time.Local)
	if !next.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, next)
	}
}

func TestNextOccurrenceEveryMonday(t *testing.T) {
	r, _ := ParseRepeat("every monday")
	cal := testCal()
	// May 6, 2026 is a Wednesday
	from := time.Date(2026, 5, 6, 0, 0, 0, 0, time.Local)
	next := r.NextOccurrence(from, cal)
	expected := time.Date(2026, 5, 11, 0, 0, 0, 0, time.Local)
	if !next.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, next)
	}
}

func TestNextOccurrenceEveryFriday(t *testing.T) {
	r, _ := ParseRepeat("every friday")
	cal := testCal()
	// May 6, 2026 is a Wednesday
	from := time.Date(2026, 5, 6, 0, 0, 0, 0, time.Local)
	next := r.NextOccurrence(from, cal)
	expected := time.Date(2026, 5, 8, 0, 0, 0, 0, time.Local)
	if !next.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, next)
	}
}

func TestNextOccurrenceLastWorkday(t *testing.T) {
	r, _ := ParseRepeat("every last workday")
	cal := testCal()
	from := time.Date(2026, 5, 6, 0, 0, 0, 0, time.Local)
	next := r.NextOccurrence(from, cal)
	// Last workday of May 2026 is May 29 (Friday)
	if next.Day() != 29 || next.Month() != 5 {
		t.Errorf("expected May 29, got %v", next)
	}
}

func TestNextOccurrence2ndLastWorkday(t *testing.T) {
	r, _ := ParseRepeat("every 2nd-last workday")
	cal := testCal()
	from := time.Date(2026, 5, 6, 0, 0, 0, 0, time.Local)
	next := r.NextOccurrence(from, cal)
	// 2nd-last workday of May 2026 is May 28 (Thursday)
	if next.Day() != 28 || next.Month() != 5 {
		t.Errorf("expected May 28, got %v", next)
	}
}

func TestNextOccurrenceMonthly15(t *testing.T) {
	r, _ := ParseRepeat("monthly 15")
	cal := testCal()
	from := time.Date(2026, 5, 6, 0, 0, 0, 0, time.Local)
	next := r.NextOccurrence(from, cal)
	expected := time.Date(2026, 5, 15, 0, 0, 0, 0, time.Local)
	if !next.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, next)
	}
}

func TestNextOccurrenceMonthly15AfterThe15th(t *testing.T) {
	r, _ := ParseRepeat("monthly 15")
	cal := testCal()
	from := time.Date(2026, 5, 20, 0, 0, 0, 0, time.Local)
	next := r.NextOccurrence(from, cal)
	expected := time.Date(2026, 6, 15, 0, 0, 0, 0, time.Local)
	if !next.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, next)
	}
}

func TestNextOccurrenceLastWorkdayAfterLastWorkday(t *testing.T) {
	r, _ := ParseRepeat("every last workday")
	cal := testCal()
	// From May 30, which is past the last workday (May 29)
	from := time.Date(2026, 5, 30, 0, 0, 0, 0, time.Local)
	next := r.NextOccurrence(from, cal)
	// Should roll to June
	if next.Month() != 6 {
		t.Errorf("expected June, got %v", next.Month())
	}
}

func TestNextOccurrenceBiWeeklyFriday(t *testing.T) {
	r, _ := ParseRepeat("every 2 weeks friday")
	cal := testCal()
	// May 6, 2026 is a Wednesday
	from := time.Date(2026, 5, 6, 0, 0, 0, 0, time.Local)
	next := r.NextOccurrence(from, cal)
	// Next Friday is May 8, then +7 = May 15
	expected := time.Date(2026, 5, 15, 0, 0, 0, 0, time.Local)
	if !next.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, next)
	}
}
