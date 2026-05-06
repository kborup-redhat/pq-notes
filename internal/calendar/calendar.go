package calendar

import (
	"fmt"
	"strings"
	"time"

	"github.com/kborup-redhat/pq-notes/internal/config"
	"github.com/rickar/cal/v2"
	"github.com/rickar/cal/v2/ar"
	"github.com/rickar/cal/v2/at"
	"github.com/rickar/cal/v2/au"
	"github.com/rickar/cal/v2/be"
	"github.com/rickar/cal/v2/bg"
	"github.com/rickar/cal/v2/br"
	"github.com/rickar/cal/v2/ca"
	"github.com/rickar/cal/v2/ch"
	"github.com/rickar/cal/v2/cy"
	"github.com/rickar/cal/v2/cz"
	"github.com/rickar/cal/v2/de"
	"github.com/rickar/cal/v2/dk"
	"github.com/rickar/cal/v2/ee"
	"github.com/rickar/cal/v2/es"
	"github.com/rickar/cal/v2/fi"
	"github.com/rickar/cal/v2/fr"
	"github.com/rickar/cal/v2/gb"
	"github.com/rickar/cal/v2/gr"
	"github.com/rickar/cal/v2/hr"
	"github.com/rickar/cal/v2/hu"
	"github.com/rickar/cal/v2/ie"
	"github.com/rickar/cal/v2/is"
	"github.com/rickar/cal/v2/it"
	"github.com/rickar/cal/v2/jp"
	"github.com/rickar/cal/v2/ke"
	"github.com/rickar/cal/v2/lt"
	"github.com/rickar/cal/v2/lu"
	"github.com/rickar/cal/v2/lv"
	"github.com/rickar/cal/v2/mt"
	"github.com/rickar/cal/v2/mw"
	"github.com/rickar/cal/v2/mx"
	"github.com/rickar/cal/v2/nl"
	"github.com/rickar/cal/v2/no"
	"github.com/rickar/cal/v2/nz"
	"github.com/rickar/cal/v2/pl"
	"github.com/rickar/cal/v2/pt"
	"github.com/rickar/cal/v2/ro"
	"github.com/rickar/cal/v2/rs"
	"github.com/rickar/cal/v2/ru"
	"github.com/rickar/cal/v2/se"
	"github.com/rickar/cal/v2/si"
	"github.com/rickar/cal/v2/sk"
	"github.com/rickar/cal/v2/th"
	"github.com/rickar/cal/v2/ua"
	"github.com/rickar/cal/v2/us"
	"github.com/rickar/cal/v2/za"
)

// BusinessCal wraps a cal.BusinessCalendar with country-specific holidays
// and configurable weekend days.
type BusinessCal struct {
	bc *cal.BusinessCalendar
}

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

// IsWorkday reports whether the given time falls on a working day.
func (c *BusinessCal) IsWorkday(t time.Time) bool {
	return c.bc.IsWorkday(t)
}

// WorkdaysFrom returns the date that is offset working days from start.
func (c *BusinessCal) WorkdaysFrom(start time.Time, offset int) time.Time {
	return c.bc.WorkdaysFrom(start, offset)
}

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

func isWeekend(day time.Weekday, weekendDays []string) bool {
	dayName := strings.ToLower(day.String())
	for _, wd := range weekendDays {
		if strings.ToLower(wd) == dayName {
			return true
		}
	}
	return false
}

// countryHolidays returns the national holidays for the given ISO 3166-1 alpha-2
// country code. Returns a basic set (New Year, Christmas) for unknown countries.
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
	case "CY":
		return cy.Holidays
	case "CZ":
		return cz.Holidays
	case "DE":
		return de.Holidays
	case "DK":
		return dk.Holidays
	case "EE":
		return ee.Holidays
	case "ES":
		return es.Holidays
	case "FI":
		return fi.Holidays
	case "FR":
		return fr.Holidays
	case "GB":
		return gb.Holidays
	case "GR":
		return gr.Holidays
	case "HR":
		return hr.Holidays
	case "HU":
		return hu.Holidays
	case "IE":
		return ie.Holidays
	case "IS":
		return is.Holidays
	case "IT":
		return it.Holidays
	case "JP":
		return jp.Holidays
	case "KE":
		return ke.Holidays
	case "LT":
		return lt.Holidays
	case "LU":
		return lu.Holidays
	case "LV":
		return lv.Holidays
	case "MT":
		return mt.Holidays
	case "MW":
		return mw.Holidays
	case "MX":
		return mx.Holidays
	case "NL":
		return nl.Holidays
	case "NO":
		return no.Holidays
	case "NZ":
		return nz.Holidays
	case "PL":
		return pl.Holidays
	case "PT":
		return pt.Holidays
	case "RO":
		return ro.Holidays
	case "RS":
		return rs.Holidays
	case "RU":
		return ru.Holidays
	case "SE":
		return se.Holidays
	case "SI":
		return si.Holidays
	case "SK":
		return sk.Holidays
	case "TH":
		return th.Holidays
	case "UA":
		return ua.Holidays
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

func addCountryHolidays(bc *cal.BusinessCalendar, country string) {
	holidays := countryHolidays(country)
	bc.AddHoliday(holidays...)
}

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
