package calendar

import (
	"testing"
	"time"

	"github.com/kborup-redhat/pq-notes/internal/config"
)

func TestIsWorkday(t *testing.T) {
	cfg := &config.Config{
		Country: "DK",
		Weekend: []string{"saturday", "sunday"},
	}
	c := New(cfg)

	wed := time.Date(2026, 5, 6, 12, 0, 0, 0, time.Local)
	if !c.IsWorkday(wed) {
		t.Error("Wednesday should be a workday")
	}

	sat := time.Date(2026, 5, 9, 12, 0, 0, 0, time.Local)
	if c.IsWorkday(sat) {
		t.Error("Saturday should not be a workday")
	}

	sun := time.Date(2026, 5, 10, 12, 0, 0, 0, time.Local)
	if c.IsWorkday(sun) {
		t.Error("Sunday should not be a workday")
	}
}

func TestIsWorkdaySaudiArabia(t *testing.T) {
	cfg := &config.Config{
		Country: "SA",
		Weekend: []string{"friday", "saturday"},
	}
	c := New(cfg)

	fri := time.Date(2026, 5, 8, 12, 0, 0, 0, time.Local)
	if c.IsWorkday(fri) {
		t.Error("Friday should not be a workday in SA")
	}

	sun := time.Date(2026, 5, 10, 12, 0, 0, 0, time.Local)
	if !c.IsWorkday(sun) {
		t.Error("Sunday should be a workday in SA")
	}
}

func TestNthLastWorkday(t *testing.T) {
	cfg := &config.Config{
		Country: "DK",
		Weekend: []string{"saturday", "sunday"},
	}
	c := New(cfg)

	last := c.NthLastWorkday(2026, time.May, 1)
	if last.Day() != 29 {
		t.Errorf("last workday of May 2026: expected 29, got %d", last.Day())
	}

	secondLast := c.NthLastWorkday(2026, time.May, 2)
	if secondLast.Day() != 28 {
		t.Errorf("2nd-last workday of May 2026: expected 28, got %d", secondLast.Day())
	}
}

func TestCustomHoliday(t *testing.T) {
	cfg := &config.Config{
		Country: "DK",
		Weekend: []string{"saturday", "sunday"},
		CustomHolidays: []config.CustomHoliday{
			{Name: "Company Day", Date: "15-06"},
		},
	}
	c := New(cfg)

	companyDay := time.Date(2026, 6, 15, 12, 0, 0, 0, time.Local)
	if c.IsWorkday(companyDay) {
		t.Error("Company Day (June 15) should not be a workday")
	}
}

func TestWorkdaysFrom(t *testing.T) {
	cfg := &config.Config{
		Country: "DK",
		Weekend: []string{"saturday", "sunday"},
	}
	c := New(cfg)

	start := time.Date(2026, 5, 6, 0, 0, 0, 0, time.Local)
	result := c.WorkdaysFrom(start, 5)
	if result.Day() != 13 {
		t.Errorf("5 workdays from May 6: expected May 13, got %v", result)
	}
}
