package dateutil

import (
	"testing"
	"time"
)

// Test FormatDate with EU format
func TestFormatDate_EU_WithTime(t *testing.T) {
	dt := time.Date(2024, 3, 15, 14, 30, 0, 0, time.UTC)
	result := FormatDate(dt, "EU")
	expected := "15-03-2024 14:30"
	if result != expected {
		t.Errorf("FormatDate(EU with time) = %q, want %q", result, expected)
	}
}

func TestFormatDate_EU_WithoutTime(t *testing.T) {
	dt := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)
	result := FormatDate(dt, "EU")
	expected := "15-03-2024"
	if result != expected {
		t.Errorf("FormatDate(EU without time) = %q, want %q", result, expected)
	}
}

func TestFormatDate_US_WithTime(t *testing.T) {
	dt := time.Date(2024, 3, 15, 14, 30, 0, 0, time.UTC)
	result := FormatDate(dt, "US")
	expected := "03-15-2024 14:30"
	if result != expected {
		t.Errorf("FormatDate(US with time) = %q, want %q", result, expected)
	}
}

func TestFormatDate_US_WithoutTime(t *testing.T) {
	dt := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)
	result := FormatDate(dt, "US")
	expected := "03-15-2024"
	if result != expected {
		t.Errorf("FormatDate(US without time) = %q, want %q", result, expected)
	}
}

func TestFormatDate_ZeroTime(t *testing.T) {
	var dt time.Time
	result := FormatDate(dt, "EU")
	if result != "" {
		t.Errorf("FormatDate(zero time) = %q, want empty string", result)
	}
}

// Test FormatDateOnly
func TestFormatDateOnly_EU(t *testing.T) {
	dt := time.Date(2024, 3, 15, 14, 30, 0, 0, time.UTC)
	result := FormatDateOnly(dt, "EU")
	expected := "15-03-2024"
	if result != expected {
		t.Errorf("FormatDateOnly(EU) = %q, want %q", result, expected)
	}
}

func TestFormatDateOnly_US(t *testing.T) {
	dt := time.Date(2024, 3, 15, 14, 30, 0, 0, time.UTC)
	result := FormatDateOnly(dt, "US")
	expected := "03-15-2024"
	if result != expected {
		t.Errorf("FormatDateOnly(US) = %q, want %q", result, expected)
	}
}

func TestFormatDateOnly_ZeroTime(t *testing.T) {
	var dt time.Time
	result := FormatDateOnly(dt, "EU")
	if result != "" {
		t.Errorf("FormatDateOnly(zero time) = %q, want empty string", result)
	}
}

// Test ParseDate with "none" and empty string
func TestParseDate_None(t *testing.T) {
	now := time.Date(2024, 3, 15, 14, 30, 0, 0, time.UTC)
	result, err := ParseDate("none", "EU", now)
	if err != nil {
		t.Errorf("ParseDate(none) error = %v, want nil", err)
	}
	if !result.IsZero() {
		t.Errorf("ParseDate(none) = %v, want zero time", result)
	}
}

func TestParseDate_EmptyString(t *testing.T) {
	now := time.Date(2024, 3, 15, 14, 30, 0, 0, time.UTC)
	result, err := ParseDate("", "EU", now)
	if err != nil {
		t.Errorf("ParseDate(empty) error = %v, want nil", err)
	}
	if !result.IsZero() {
		t.Errorf("ParseDate(empty) = %v, want zero time", result)
	}
}

// Test ParseDate with natural language - "today"
func TestParseDate_Today(t *testing.T) {
	now := time.Date(2024, 3, 15, 14, 30, 0, 0, time.UTC)
	result, err := ParseDate("today", "EU", now)
	if err != nil {
		t.Errorf("ParseDate(today) error = %v, want nil", err)
	}
	expected := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Errorf("ParseDate(today) = %v, want %v", result, expected)
	}
}

// Test ParseDate with natural language - "tomorrow"
func TestParseDate_Tomorrow(t *testing.T) {
	now := time.Date(2024, 3, 15, 14, 30, 0, 0, time.UTC)
	result, err := ParseDate("tomorrow", "EU", now)
	if err != nil {
		t.Errorf("ParseDate(tomorrow) error = %v, want nil", err)
	}
	expected := time.Date(2024, 3, 16, 0, 0, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Errorf("ParseDate(tomorrow) = %v, want %v", result, expected)
	}
}

// Test ParseDate with natural language - "next week"
func TestParseDate_NextWeek(t *testing.T) {
	now := time.Date(2024, 3, 15, 14, 30, 0, 0, time.UTC)
	result, err := ParseDate("next week", "EU", now)
	if err != nil {
		t.Errorf("ParseDate(next week) error = %v, want nil", err)
	}
	expected := time.Date(2024, 3, 22, 0, 0, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Errorf("ParseDate(next week) = %v, want %v", result, expected)
	}
}

// Test ParseDate with weekday names
func TestParseDate_Monday(t *testing.T) {
	// Now is Friday, March 15, 2024
	now := time.Date(2024, 3, 15, 14, 30, 0, 0, time.UTC)
	result, err := ParseDate("monday", "EU", now)
	if err != nil {
		t.Errorf("ParseDate(monday) error = %v, want nil", err)
	}
	// Next Monday is March 18
	expected := time.Date(2024, 3, 18, 0, 0, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Errorf("ParseDate(monday) = %v, want %v", result, expected)
	}
}

func TestParseDate_Friday(t *testing.T) {
	// Now is Friday, March 15, 2024
	now := time.Date(2024, 3, 15, 14, 30, 0, 0, time.UTC)
	result, err := ParseDate("friday", "EU", now)
	if err != nil {
		t.Errorf("ParseDate(friday) error = %v, want nil", err)
	}
	// Next Friday is March 22 (7 days from now)
	expected := time.Date(2024, 3, 22, 0, 0, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Errorf("ParseDate(friday) = %v, want %v", result, expected)
	}
}

func TestParseDate_Tuesday(t *testing.T) {
	// Now is Friday, March 15, 2024
	now := time.Date(2024, 3, 15, 14, 30, 0, 0, time.UTC)
	result, err := ParseDate("tuesday", "EU", now)
	if err != nil {
		t.Errorf("ParseDate(tuesday) error = %v, want nil", err)
	}
	// Next Tuesday is March 19
	expected := time.Date(2024, 3, 19, 0, 0, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Errorf("ParseDate(tuesday) = %v, want %v", result, expected)
	}
}

// Test ParseDate with date strings - EU format
func TestParseDate_EU_DateOnly(t *testing.T) {
	now := time.Date(2024, 3, 15, 14, 30, 0, 0, time.UTC)
	result, err := ParseDate("25-12-2024", "EU", now)
	if err != nil {
		t.Errorf("ParseDate(EU date) error = %v, want nil", err)
	}
	expected := time.Date(2024, 12, 25, 0, 0, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Errorf("ParseDate(EU date) = %v, want %v", result, expected)
	}
}

func TestParseDate_EU_DateTime(t *testing.T) {
	now := time.Date(2024, 3, 15, 14, 30, 0, 0, time.UTC)
	result, err := ParseDate("25-12-2024 18:45", "EU", now)
	if err != nil {
		t.Errorf("ParseDate(EU datetime) error = %v, want nil", err)
	}
	expected := time.Date(2024, 12, 25, 18, 45, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Errorf("ParseDate(EU datetime) = %v, want %v", result, expected)
	}
}

// Test ParseDate with date strings - US format
func TestParseDate_US_DateOnly(t *testing.T) {
	now := time.Date(2024, 3, 15, 14, 30, 0, 0, time.UTC)
	result, err := ParseDate("12-25-2024", "US", now)
	if err != nil {
		t.Errorf("ParseDate(US date) error = %v, want nil", err)
	}
	expected := time.Date(2024, 12, 25, 0, 0, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Errorf("ParseDate(US date) = %v, want %v", result, expected)
	}
}

func TestParseDate_US_DateTime(t *testing.T) {
	now := time.Date(2024, 3, 15, 14, 30, 0, 0, time.UTC)
	result, err := ParseDate("12-25-2024 18:45", "US", now)
	if err != nil {
		t.Errorf("ParseDate(US datetime) error = %v, want nil", err)
	}
	expected := time.Date(2024, 12, 25, 18, 45, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Errorf("ParseDate(US datetime) = %v, want %v", result, expected)
	}
}

// Test edge cases
func TestParseDate_InvalidFormat(t *testing.T) {
	now := time.Date(2024, 3, 15, 14, 30, 0, 0, time.UTC)
	_, err := ParseDate("invalid date", "EU", now)
	if err == nil {
		t.Error("ParseDate(invalid) should return error")
	}
}

func TestParseDate_CaseInsensitive(t *testing.T) {
	now := time.Date(2024, 3, 15, 14, 30, 0, 0, time.UTC)

	tests := []string{"MONDAY", "Monday", "MoNdAy"}
	expected := time.Date(2024, 3, 18, 0, 0, 0, 0, time.UTC)

	for _, input := range tests {
		result, err := ParseDate(input, "EU", now)
		if err != nil {
			t.Errorf("ParseDate(%s) error = %v, want nil", input, err)
		}
		if !result.Equal(expected) {
			t.Errorf("ParseDate(%s) = %v, want %v", input, result, expected)
		}
	}
}

func TestParseDate_AllWeekdays(t *testing.T) {
	// Now is Friday, March 15, 2024
	now := time.Date(2024, 3, 15, 14, 30, 0, 0, time.UTC)

	tests := []struct {
		weekday  string
		expected time.Time
	}{
		{"sunday", time.Date(2024, 3, 17, 0, 0, 0, 0, time.UTC)},    // 2 days ahead
		{"monday", time.Date(2024, 3, 18, 0, 0, 0, 0, time.UTC)},    // 3 days ahead
		{"tuesday", time.Date(2024, 3, 19, 0, 0, 0, 0, time.UTC)},   // 4 days ahead
		{"wednesday", time.Date(2024, 3, 20, 0, 0, 0, 0, time.UTC)}, // 5 days ahead
		{"thursday", time.Date(2024, 3, 21, 0, 0, 0, 0, time.UTC)},  // 6 days ahead
		{"friday", time.Date(2024, 3, 22, 0, 0, 0, 0, time.UTC)},    // 7 days ahead
		{"saturday", time.Date(2024, 3, 16, 0, 0, 0, 0, time.UTC)},  // 1 day ahead
	}

	for _, tt := range tests {
		result, err := ParseDate(tt.weekday, "EU", now)
		if err != nil {
			t.Errorf("ParseDate(%s) error = %v, want nil", tt.weekday, err)
		}
		if !result.Equal(tt.expected) {
			t.Errorf("ParseDate(%s) = %v, want %v", tt.weekday, result, tt.expected)
		}
	}
}

// Test leading zeros in dates
func TestParseDate_EU_WithLeadingZeros(t *testing.T) {
	now := time.Date(2024, 3, 15, 14, 30, 0, 0, time.UTC)
	result, err := ParseDate("05-03-2024", "EU", now)
	if err != nil {
		t.Errorf("ParseDate(EU with leading zeros) error = %v, want nil", err)
	}
	expected := time.Date(2024, 3, 5, 0, 0, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Errorf("ParseDate(EU with leading zeros) = %v, want %v", result, expected)
	}
}

func TestParseDate_US_WithLeadingZeros(t *testing.T) {
	now := time.Date(2024, 3, 15, 14, 30, 0, 0, time.UTC)
	result, err := ParseDate("03-05-2024", "US", now)
	if err != nil {
		t.Errorf("ParseDate(US with leading zeros) error = %v, want nil", err)
	}
	expected := time.Date(2024, 3, 5, 0, 0, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Errorf("ParseDate(US with leading zeros) = %v, want %v", result, expected)
	}
}
