package tui

import (
	"filippo.io/age"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/kborup-redhat/pq-notes/internal/calendar"
	"github.com/kborup-redhat/pq-notes/internal/config"
	"github.com/kborup-redhat/pq-notes/internal/editor"
	"github.com/kborup-redhat/pq-notes/internal/notes"
)

type viewState int

const (
	viewDashboard viewState = iota
	viewSetup
	viewNewNote
	viewSearch
	viewFilter
)

type focusPane int

const (
	focusList focusPane = iota
	focusPreview
)

// App is the root Bubble Tea model providing a split-pane layout
// with a note list on the left and a preview pane on the right.
type App struct {
	cfg      *config.Config
	store    *notes.NoteStore
	cal      *calendar.BusinessCal
	identity *age.HybridIdentity

	notes          []*notes.Note
	dashboardItems []DashboardItem
	cursor         int
	focus          focusPane
	view           viewState
	width          int
	height         int
	showDone       bool
	newNote        *NewNoteModel
	err            error

	search           *SearchModel
	tagFilter        *FilterModel
	typeFilter       *FilterModel
	activeTagFilter  []string
	activeTypeFilter []string
}

// NewApp creates a new App model with the given dependencies.
func NewApp(cfg *config.Config, store *notes.NoteStore, cal *calendar.BusinessCal, identity *age.HybridIdentity) *App {
	return &App{
		cfg:      cfg,
		store:    store,
		cal:      cal,
		identity: identity,
		view:     viewDashboard,
		focus:    focusList,
	}
}

// Init returns a command that loads notes on startup.
func (a *App) Init() tea.Cmd {
	return a.loadNotes
}

func (a *App) loadNotes() tea.Msg {
	allNotes, err := a.store.List()
	if err != nil {
		return errMsg{err}
	}
	return notesLoadedMsg{notes: allNotes}
}

type notesLoadedMsg struct {
	notes []*notes.Note
}

type errMsg struct {
	err error
}

// Update handles messages and returns the updated model and any commands.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Route key presses to the new note wizard when it is active.
	if a.view == viewNewNote && a.newNote != nil {
		if kp, ok := msg.(tea.KeyPressMsg); ok {
			done, cmd := a.newNote.Update(kp)
			if done {
				a.view = viewDashboard
				a.newNote = nil
				return a, tea.Batch(cmd, a.loadNotes)
			}
			return a, cmd
		}
	}

	// Route key presses to the search overlay when it is active.
	if a.view == viewSearch && a.search != nil {
		if kp, ok := msg.(tea.KeyPressMsg); ok {
			done, _ := a.search.Update(kp)
			if done {
				a.view = viewDashboard
				a.search = nil
			}
			return a, nil
		}
	}

	// Route key presses to the filter overlay when it is active.
	if a.view == viewFilter {
		if kp, ok := msg.(tea.KeyPressMsg); ok {
			if a.tagFilter != nil {
				done := a.tagFilter.Update(kp)
				if done {
					a.activeTagFilter = a.tagFilter.SelectedItems()
					a.tagFilter = nil
					a.view = viewDashboard
				}
			} else if a.typeFilter != nil {
				done := a.typeFilter.Update(kp)
				if done {
					a.activeTypeFilter = a.typeFilter.SelectedItems()
					a.typeFilter = nil
					a.view = viewDashboard
				}
			}
			return a, nil
		}
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		return a, nil

	case notesLoadedMsg:
		a.notes = msg.notes
		return a, nil

	case noteCreatedMsg:
		return a, func() tea.Msg {
			if err := editor.Open(a.cfg.Editor, msg.path); err != nil {
				return errMsg{err}
			}
			return editorClosedMsg{path: msg.path}
		}

	case editorClosedMsg:
		return a, a.loadNotes

	case errMsg:
		a.err = msg.err
		return a, nil

	case tea.KeyPressMsg:
		return a.handleKey(msg)
	}

	return a, nil
}

func (a *App) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.Key()

	switch key.Code {
	case tea.KeyUp:
		if a.cursor > 0 {
			a.cursor--
		}
	case tea.KeyDown:
		if a.cursor < len(a.dashboardItems)-1 {
			a.cursor++
		}
	case tea.KeyTab:
		if a.focus == focusList {
			a.focus = focusPreview
		} else {
			a.focus = focusList
		}
	case 'n':
		a.newNote = NewNewNoteModel(a.cfg, a.store, a.notes)
		a.view = viewNewNote
	case 's':
		a.search = NewSearchModel(a.store)
		a.view = viewSearch
	case 't':
		a.tagFilter = NewTagFilter(a.notes)
		a.view = viewFilter
	case 'y':
		a.typeFilter = NewTypeFilter()
		a.view = viewFilter
	case 'q':
		return a, tea.Quit
	case 'a':
		a.showDone = !a.showDone
	}

	return a, nil
}

// View renders the split-pane layout.
func (a *App) View() tea.View {
	if a.view == viewNewNote && a.newNote != nil {
		return tea.NewView(a.newNote.View())
	}

	if a.view == viewSearch && a.search != nil {
		return tea.NewView(a.search.View())
	}

	if a.view == viewFilter {
		if a.tagFilter != nil {
			return tea.NewView(a.tagFilter.View())
		}
		if a.typeFilter != nil {
			return tea.NewView(a.typeFilter.View())
		}
	}

	if a.width == 0 {
		return tea.NewView("Loading...")
	}

	listWidth := a.width / 3
	previewWidth := a.width - listWidth - 3

	list := a.renderList(listWidth)
	preview := a.renderPreview(previewWidth)

	help := helpStyle.Render("[n]ew  [e]dit  [d]ue  [t]ag filter  t[y]pe filter  [s]earch  [m]ark done  [q]uit")

	content := lipgloss.JoinHorizontal(lipgloss.Top,
		borderStyle.Width(listWidth).Height(a.height-3).Render(list),
		previewBorderStyle.Width(previewWidth).Height(a.height-3).Render(preview),
	)

	return tea.NewView(lipgloss.JoinVertical(lipgloss.Left, content, help))
}

func (a *App) renderList(width int) string {
	filtered := a.applyFilters(a.notes)
	items := BuildDashboard(filtered, a.showDone, a.cfg.DateFormat)
	a.dashboardItems = items
	return RenderDashboard(items, a.cursor, width, a.cfg.DateFormat)
}

func (a *App) applyFilters(allNotes []*notes.Note) []*notes.Note {
	if len(a.activeTagFilter) == 0 && len(a.activeTypeFilter) == 0 {
		return allNotes
	}
	var result []*notes.Note
	for _, n := range allNotes {
		if len(a.activeTypeFilter) > 0 && !contains(a.activeTypeFilter, string(n.Type)) {
			continue
		}
		if len(a.activeTagFilter) > 0 && !hasAnyTag(n.Tags, a.activeTagFilter) {
			continue
		}
		result = append(result, n)
	}
	return result
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func hasAnyTag(noteTags, filterTags []string) bool {
	for _, ft := range filterTags {
		for _, nt := range noteTags {
			if ft == nt {
				return true
			}
		}
	}
	return false
}

func (a *App) renderPreview(width int) string {
	if len(a.dashboardItems) == 0 || a.cursor >= len(a.dashboardItems) {
		return dimStyle.Render("Select a note to preview")
	}
	note := a.dashboardItems[a.cursor].Note
	return RenderPreview(note, width, a.cfg.DateFormat)
}

// RunApp creates and runs the TUI application.
func RunApp(cfg *config.Config, store *notes.NoteStore, cal *calendar.BusinessCal, identity *age.HybridIdentity) error {
	app := NewApp(cfg, store, cal, identity)
	p := tea.NewProgram(app)
	_, err := p.Run()
	return err
}
