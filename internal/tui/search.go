package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/kborup-redhat/pq-notes/internal/notes"
)

// SearchModel is a sub-component for fuzzy text search across notes.
// It is not a tea.Model; its Update method takes a tea.KeyPressMsg and
// returns whether the overlay is done and an optional selected note.
type SearchModel struct {
	query   string
	results []*notes.Note
	cursor  int
	store   *notes.NoteStore
}

// NewSearchModel creates a new search overlay with the given note store.
func NewSearchModel(store *notes.NoteStore) *SearchModel {
	return &SearchModel{
		store: store,
	}
}

// Update processes a key press. Returns done=true when the overlay should
// close. If the user pressed Enter on a result, selected is non-nil.
func (s *SearchModel) Update(msg tea.KeyPressMsg) (done bool, selected *notes.Note) {
	key := msg.Key()

	switch key.Code {
	case tea.KeyEscape:
		return true, nil
	case tea.KeyEnter:
		if len(s.results) > 0 && s.cursor < len(s.results) {
			return true, s.results[s.cursor]
		}
		return true, nil
	case tea.KeyUp:
		if s.cursor > 0 {
			s.cursor--
		}
	case tea.KeyDown:
		if s.cursor < len(s.results)-1 {
			s.cursor++
		}
	case tea.KeyBackspace:
		if len(s.query) > 0 {
			s.query = removeLastRune(s.query)
			s.search()
		}
	default:
		if key.Text != "" {
			s.query += key.Text
			s.search()
		}
	}

	return false, nil
}

// search runs the query against the store and updates results.
func (s *SearchModel) search() {
	if strings.TrimSpace(s.query) == "" {
		s.results = nil
		s.cursor = 0
		return
	}
	results, err := s.store.Search(s.query)
	if err != nil {
		s.results = nil
		s.cursor = 0
		return
	}
	s.results = results
	if s.cursor >= len(s.results) {
		if len(s.results) > 0 {
			s.cursor = len(s.results) - 1
		} else {
			s.cursor = 0
		}
	}
}

// View renders the search bar and results list.
func (s *SearchModel) View() string {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString(titleStyle.Render("Search Notes"))
	sb.WriteString("\n\n")

	inputStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#333333")).
		Padding(0, 1).
		Width(50)

	sb.WriteString(normalStyle.Render("  Query: "))
	sb.WriteString(inputStyle.Render(s.query + "_"))
	sb.WriteString("\n\n")

	if len(s.results) == 0 {
		if strings.TrimSpace(s.query) != "" {
			sb.WriteString(dimStyle.Render("  No results found."))
		} else {
			sb.WriteString(dimStyle.Render("  Type to search..."))
		}
		sb.WriteString("\n")
	} else {
		sb.WriteString(dimStyle.Render(fmt.Sprintf("  %d result(s)", len(s.results))))
		sb.WriteString("\n\n")

		for i, n := range s.results {
			label := fmt.Sprintf("  %s %s - %s",
				typeStyle.Render("["+string(n.Type)+"]"),
				n.Customer,
				n.Title,
			)
			if len(n.Tags) > 0 {
				label += "  " + tagStyle.Render("["+strings.Join(n.Tags, ", ")+"]")
			}

			if i == s.cursor {
				sb.WriteString(selectedStyle.Render(label))
			} else {
				sb.WriteString(normalStyle.Render(label))
			}
			sb.WriteString("\n")
		}
	}

	sb.WriteString("\n")
	sb.WriteString(helpStyle.Render("[Up/Down] navigate  [Enter] select  [Esc] cancel"))
	sb.WriteString("\n")

	return sb.String()
}
