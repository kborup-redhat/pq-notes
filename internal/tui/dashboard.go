package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/kborup-redhat/pq-notes/internal/dateutil"
	"github.com/kborup-redhat/pq-notes/internal/notes"
)

type urgencyGroup int

const (
	groupOverdue urgencyGroup = iota
	groupToday
	groupUpcoming
	groupNoDue
)

// DashboardItem wraps a note with its urgency group and display index.
type DashboardItem struct {
	Note  *notes.Note
	Group urgencyGroup
	Index int
}

// BuildDashboard classifies notes into urgency groups and sorts them.
// Overdue notes come first, then today, upcoming, and finally notes
// without a due date (sorted by most recently created).
func BuildDashboard(allNotes []*notes.Note, showDone bool, dateFormat string) []DashboardItem {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	tomorrow := today.AddDate(0, 0, 1)

	var items []DashboardItem
	for _, n := range allNotes {
		if !showDone && n.Status == notes.StatusDone {
			continue
		}

		group := groupNoDue
		if !n.Due.IsZero() {
			if n.Due.Before(today) {
				group = groupOverdue
			} else if n.Due.Before(tomorrow) {
				group = groupToday
			} else {
				group = groupUpcoming
			}
		}

		items = append(items, DashboardItem{Note: n, Group: group})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].Group != items[j].Group {
			return items[i].Group < items[j].Group
		}
		if items[i].Note.Due.IsZero() && items[j].Note.Due.IsZero() {
			return items[i].Note.Created.After(items[j].Note.Created)
		}
		if items[i].Note.Due.IsZero() {
			return false
		}
		if items[j].Note.Due.IsZero() {
			return true
		}
		return items[i].Note.Due.Before(items[j].Note.Due)
	})

	for i := range items {
		items[i].Index = i
	}

	return items
}

// RenderDashboard renders the grouped and sorted dashboard items as a string.
func RenderDashboard(items []DashboardItem, cursor int, width int, dateFormat string) string {
	if len(items) == 0 {
		return headerStyle.Render("No notes yet. Press [n] to create one.")
	}

	var sb strings.Builder
	currentGroup := urgencyGroup(-1)

	groupHeaders := map[urgencyGroup]string{
		groupOverdue:  "OVERDUE",
		groupToday:    "TODAY",
		groupUpcoming: "UPCOMING",
		groupNoDue:    "NO DUE DATE",
	}

	groupStyles := map[urgencyGroup]lipgloss.Style{
		groupOverdue:  overdueStyle,
		groupToday:    todayStyle,
		groupUpcoming: upcomingStyle,
		groupNoDue:    dimStyle,
	}

	for _, item := range items {
		if item.Group != currentGroup {
			currentGroup = item.Group
			style := groupStyles[currentGroup]
			sb.WriteString("\n " + style.Render(groupHeaders[currentGroup]) + "\n")
		}

		line := renderDashboardItem(item, dateFormat)

		if item.Index == cursor {
			sb.WriteString(selectedStyle.Width(width - 2).Render(line) + "\n")
		} else if item.Note.Status == notes.StatusDone {
			sb.WriteString(dimStyle.Render(line) + "\n")
		} else {
			sb.WriteString(normalStyle.Render(line) + "\n")
		}
	}

	return sb.String()
}

func renderDashboardItem(item DashboardItem, dateFormat string) string {
	n := item.Note
	typeLabel := typeStyle.Render("[" + string(n.Type) + "]")

	var priority string
	switch n.Priority {
	case notes.PriorityUrgent:
		priority = urgentStyle.Render(" [URGENT]")
	case notes.PriorityHigh:
		priority = highStyle.Render(" [HIGH]")
	}

	dueStr := ""
	if !n.Due.IsZero() {
		dueStr = " Due: " + dateutil.FormatDateOnly(n.Due, dateFormat)
	}

	return fmt.Sprintf("  %s %s%s  %s%s", typeLabel, n.Customer, priority, n.Title, dueStr)
}
