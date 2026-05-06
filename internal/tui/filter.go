package tui

import (
	"fmt"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/kborup-redhat/pq-notes/internal/notes"
)

// FilterModel is a sub-component for filtering notes by tag or type.
// It renders a checkbox list and lets the user toggle items on/off.
// It is not a tea.Model; its Update method takes a tea.KeyPressMsg.
type FilterModel struct {
	items    []string
	selected map[string]bool
	cursor   int
	title    string
}

// NewTagFilter creates a FilterModel populated with all unique tags
// found across the given notes.
func NewTagFilter(allNotes []*notes.Note) *FilterModel {
	tagSet := make(map[string]bool)
	for _, n := range allNotes {
		for _, t := range n.Tags {
			tagSet[t] = true
		}
	}

	var tags []string
	for t := range tagSet {
		tags = append(tags, t)
	}
	sort.Strings(tags)

	return &FilterModel{
		items:    tags,
		selected: make(map[string]bool),
		title:    "Filter by Tag",
	}
}

// NewTypeFilter creates a FilterModel with the fixed set of note types.
func NewTypeFilter() *FilterModel {
	return &FilterModel{
		items:    []string{"meeting", "task", "reminder", "followup"},
		selected: make(map[string]bool),
		title:    "Filter by Type",
	}
}

// Update processes a key press. Returns done=true when the overlay
// should close (Esc cancels selections, Enter applies them).
func (f *FilterModel) Update(msg tea.KeyPressMsg) (done bool) {
	key := msg.Key()

	switch key.Code {
	case tea.KeyEscape:
		// Cancel: clear selections so no filter is applied
		f.selected = make(map[string]bool)
		return true
	case tea.KeyEnter:
		return true
	case tea.KeyUp:
		if f.cursor > 0 {
			f.cursor--
		}
	case tea.KeyDown:
		if f.cursor < len(f.items)-1 {
			f.cursor++
		}
	default:
		if key.Text == " " && len(f.items) > 0 {
			item := f.items[f.cursor]
			if f.selected[item] {
				delete(f.selected, item)
			} else {
				f.selected[item] = true
			}
		}
	}

	return false
}

// SelectedItems returns the list of items that are currently checked.
func (f *FilterModel) SelectedItems() []string {
	var result []string
	for _, item := range f.items {
		if f.selected[item] {
			result = append(result, item)
		}
	}
	return result
}

// View renders the checkbox list overlay.
func (f *FilterModel) View() string {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString(titleStyle.Render(f.title))
	sb.WriteString("\n\n")

	if len(f.items) == 0 {
		sb.WriteString(dimStyle.Render("  No items available."))
		sb.WriteString("\n")
	} else {
		for i, item := range f.items {
			checkbox := "[ ]"
			if f.selected[item] {
				checkbox = "[x]"
			}

			label := fmt.Sprintf("  %s %s", checkbox, item)

			if i == f.cursor {
				sb.WriteString(selectedStyle.Render(label))
			} else {
				sb.WriteString(normalStyle.Render(label))
			}
			sb.WriteString("\n")
		}
	}

	sb.WriteString("\n")
	sb.WriteString(helpStyle.Render("[Up/Down] navigate  [Space] toggle  [Enter] apply  [Esc] cancel"))
	sb.WriteString("\n")

	return sb.String()
}
