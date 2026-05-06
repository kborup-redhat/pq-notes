package tui

import (
	"path/filepath"

	"filippo.io/age"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/kborup-redhat/pq-notes/internal/calendar"
	"github.com/kborup-redhat/pq-notes/internal/config"
	"github.com/kborup-redhat/pq-notes/internal/drive"
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
	cfg       *config.Config
	store     *notes.NoteStore
	cal       *calendar.BusinessCal
	identity  *age.HybridIdentity
	notesDir  string
	configDir string

	notes          []*notes.Note
	dashboardItems []DashboardItem
	cursor         int
	focus          focusPane
	view           viewState
	width          int
	height         int
	showDone       bool
	showClosedOnly bool
	newNote        *NewNoteModel
	err            error

	deleteConfirm    bool
	driveDelete      bool
	search           *SearchModel
	tagFilter        *FilterModel
	typeFilter       *FilterModel
	activeTagFilter  []string
	activeTypeFilter []string
}

// NewApp creates a new App model with the given dependencies.
func NewApp(cfg *config.Config, store *notes.NoteStore, cal *calendar.BusinessCal, identity *age.HybridIdentity, notesDir, configDir string) *App {
	return &App{
		cfg:       cfg,
		store:     store,
		cal:       cal,
		identity:  identity,
		notesDir:  notesDir,
		configDir: configDir,
		view:      viewDashboard,
		focus:     focusList,
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

type driveSyncDoneMsg struct{}
type driveDeleteDoneMsg struct{}

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
			done, selected := a.search.Update(kp)
			if done {
				a.view = viewDashboard
				a.search = nil
				if selected != nil {
					for i, item := range a.dashboardItems {
						if item.Note.FilePath == selected.FilePath {
							a.cursor = i
							break
						}
					}
				}
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
					a.rebuildDashboard()
				}
			} else if a.typeFilter != nil {
				done := a.typeFilter.Update(kp)
				if done {
					a.activeTypeFilter = a.typeFilter.SelectedItems()
					a.typeFilter = nil
					a.view = viewDashboard
					a.rebuildDashboard()
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
		a.rebuildDashboard()
		return a, nil

	case noteCreatedMsg:
		return a, func() tea.Msg {
			if err := editor.OpenEncrypted(a.cfg.Editor, msg.path, a.identity); err != nil {
				return errMsg{err}
			}
			return editorClosedMsg{path: msg.path}
		}

	case editorClosedMsg:
		cmds := []tea.Cmd{a.loadNotes}
		if a.cfg.DriveAutoSync {
			path := msg.path
			cmds = append(cmds, func() tea.Msg {
				drive.SyncFile(path, a.notesDir, a.configDir, a.identity)
				return driveSyncDoneMsg{}
			})
		}
		return a, tea.Batch(cmds...)

	case driveSyncDoneMsg:
		return a, nil

	case driveDeleteDoneMsg:
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

	if a.deleteConfirm {
		switch key.Code {
		case 'y':
			a.deleteConfirm = false
			if len(a.dashboardItems) > 0 && a.cursor < len(a.dashboardItems) {
				path := a.dashboardItems[a.cursor].Note.FilePath
				return a, func() tea.Msg {
					if err := a.store.Delete(path); err != nil {
						return errMsg{err}
					}
					return editorClosedMsg{path: path}
				}
			}
		case 'd':
			a.deleteConfirm = false
			if a.cfg.DriveAutoSync && len(a.dashboardItems) > 0 && a.cursor < len(a.dashboardItems) {
				note := a.dashboardItems[a.cursor].Note
				notePath := note.FilePath
				relPath, _ := filepath.Rel(a.notesDir, notePath)
				return a, func() tea.Msg {
					if err := a.store.Delete(notePath); err != nil {
						return errMsg{err}
					}
					drive.DeleteFile(filepath.ToSlash(relPath), a.notesDir, a.configDir, a.identity)
					return driveDeleteDoneMsg{}
				}
			}
		case 'n', tea.KeyEscape:
			a.deleteConfirm = false
		}
		return a, nil
	}

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
	case 'e', tea.KeyEnter:
		if len(a.dashboardItems) > 0 && a.cursor < len(a.dashboardItems) {
			path := a.dashboardItems[a.cursor].Note.FilePath
			return a, func() tea.Msg {
				if err := editor.OpenEncrypted(a.cfg.Editor, path, a.identity); err != nil {
					return errMsg{err}
				}
				return editorClosedMsg{path: path}
			}
		}
	case 'm':
		if len(a.dashboardItems) > 0 && a.cursor < len(a.dashboardItems) {
			item := a.dashboardItems[a.cursor]
			note := item.Note
			if note.Status == notes.StatusDone {
				note.Status = notes.StatusOpen
			} else {
				note.Status = notes.StatusDone
			}
			return a, func() tea.Msg {
				if err := a.store.Update(note.FilePath, note); err != nil {
					return errMsg{err}
				}
				return editorClosedMsg{path: note.FilePath}
			}
		}
	case 'q', tea.KeyEscape:
		return a, tea.Quit
	case 'a':
		a.showDone = !a.showDone
		if a.showDone {
			a.showClosedOnly = false
		}
		a.rebuildDashboard()
	case 'c':
		a.showClosedOnly = !a.showClosedOnly
		if a.showClosedOnly {
			a.showDone = false
		}
		a.cursor = 0
		a.rebuildDashboard()
	case 'x':
		if len(a.dashboardItems) > 0 && a.cursor < len(a.dashboardItems) {
			a.deleteConfirm = true
		}
	}

	return a, nil
}

func altView(s string) tea.View {
	v := tea.NewView(s)
	v.AltScreen = true
	return v
}

// View renders the split-pane layout.
func (a *App) View() tea.View {
	if a.view == viewNewNote && a.newNote != nil {
		return altView(a.newNote.View())
	}

	if a.view == viewSearch && a.search != nil {
		return altView(a.search.View())
	}

	if a.view == viewFilter {
		if a.tagFilter != nil {
			return altView(a.tagFilter.View())
		}
		if a.typeFilter != nil {
			return altView(a.typeFilter.View())
		}
	}

	if a.width == 0 {
		return altView("Loading...")
	}

	listWidth := a.width / 3
	previewWidth := a.width - listWidth - 3

	list := a.renderList(listWidth)
	preview := a.renderPreview(previewWidth)

	var help string
	if a.deleteConfirm && len(a.dashboardItems) > 0 && a.cursor < len(a.dashboardItems) {
		title := a.dashboardItems[a.cursor].Note.Title
		if a.cfg.DriveAutoSync {
			help = overdueStyle.Render("Delete " + title + "?  [y]es  [n]o  [d]rive+local")
		} else {
			help = overdueStyle.Render("Delete " + title + "?  [y]es  [n]o")
		}
	} else {
		help = helpStyle.Render("[n]ew  [e]dit  [t]ag filter  t[y]pe filter  [s]earch  [m]ark done  [c]losed  [x] delete  [a]ll  [q]uit")
	}

	content := lipgloss.JoinHorizontal(lipgloss.Top,
		borderStyle.Width(listWidth).Height(a.height-3).Render(list),
		previewBorderStyle.Width(previewWidth).Height(a.height-3).Render(preview),
	)

	return altView(lipgloss.JoinVertical(lipgloss.Left, content, help))
}

func (a *App) rebuildDashboard() {
	filtered := a.applyFilters(a.notes)
	if a.showClosedOnly {
		var closed []*notes.Note
		for _, n := range filtered {
			if n.Status == notes.StatusDone {
				closed = append(closed, n)
			}
		}
		a.dashboardItems = BuildDashboard(closed, true, a.cfg.DateFormat)
	} else {
		a.dashboardItems = BuildDashboard(filtered, a.showDone, a.cfg.DateFormat)
	}
	if a.cursor >= len(a.dashboardItems) {
		if len(a.dashboardItems) > 0 {
			a.cursor = len(a.dashboardItems) - 1
		} else {
			a.cursor = 0
		}
	}
}

func (a *App) renderList(width int) string {
	if a.showClosedOnly {
		return headerStyle.Render("CLOSED NOTES") + "\n" + RenderDashboard(a.dashboardItems, a.cursor, width, a.cfg.DateFormat)
	}
	return RenderDashboard(a.dashboardItems, a.cursor, width, a.cfg.DateFormat)
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
func RunApp(cfg *config.Config, store *notes.NoteStore, cal *calendar.BusinessCal, identity *age.HybridIdentity, notesDir, configDir string) error {
	app := NewApp(cfg, store, cal, identity, notesDir, configDir)
	p := tea.NewProgram(app)
	_, err := p.Run()
	return err
}
