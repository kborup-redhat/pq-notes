package tui

import (
	"filippo.io/age"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/kborup-redhat/pq-notes/internal/calendar"
	"github.com/kborup-redhat/pq-notes/internal/config"
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
	err            error
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
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		return a, nil

	case notesLoadedMsg:
		a.notes = msg.notes
		return a, nil

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
	case 'q':
		return a, tea.Quit
	case 'a':
		a.showDone = !a.showDone
	}

	return a, nil
}

// View renders the split-pane layout.
func (a *App) View() tea.View {
	if a.width == 0 {
		return tea.NewView("Loading...")
	}

	listWidth := a.width / 3
	previewWidth := a.width - listWidth - 3

	list := a.renderList(listWidth)
	preview := a.renderPreview(previewWidth)

	help := helpStyle.Render("[n]ew  [e]dit  [d]ue  [t]ag filter  [s]earch  [m]ark done  [q]uit")

	content := lipgloss.JoinHorizontal(lipgloss.Top,
		borderStyle.Width(listWidth).Height(a.height-3).Render(list),
		previewBorderStyle.Width(previewWidth).Height(a.height-3).Render(preview),
	)

	return tea.NewView(lipgloss.JoinVertical(lipgloss.Left, content, help))
}

func (a *App) renderList(width int) string {
	items := BuildDashboard(a.notes, a.showDone, a.cfg.DateFormat)
	a.dashboardItems = items
	return RenderDashboard(items, a.cursor, width, a.cfg.DateFormat)
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
