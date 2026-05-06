---
title: "Chapter 9: TUI Architecture"
order: 9
---

# Chapter 9: TUI Architecture

Everything we have built so far -- configuration, encryption, note models, scheduling, storage, and editor integration -- has been behind-the-scenes infrastructure. The TUI (Terminal User Interface) is where all of it comes together into something the user can actually see and interact with. pq-notes uses the Bubble Tea framework to render a split-pane dashboard directly in the terminal, with a note list on the left and a preview pane on the right.

Think of the TUI as the front desk of a hotel. The guest (user) interacts only with the front desk. Behind the scenes, the desk coordinates with housekeeping (NoteStore), the vault (crypto), the scheduling system (schedule parser), and the concierge (editor integration). The guest does not need to know about any of those internal systems -- they just press keys and things happen.

## How It Works

Bubble Tea follows the **Elm Architecture**, a pattern borrowed from the Elm programming language for building UIs. It has three core concepts:

1. **Model** -- a struct that holds all application state
2. **Update** -- a function that receives messages (user input, async results) and returns a new model plus optional commands
3. **View** -- a function that renders the current model to a string for display

The cycle is: render the view, wait for input, update the model, render again. The framework handles the loop -- you just implement the three functions.

There is no mutation during rendering. The View function is a pure transformation from state to string. All state changes happen in Update. This separation makes the code predictable and debuggable: if the screen looks wrong, you know the problem is either in the model (wrong state) or in the view (wrong rendering), not in some interleaved mutation.

## Code Deep Dive

### View States and Focus

The TUI uses two enums to track what the user is looking at:

```go
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
    focusList    focusPane = iota
    focusPreview
)
```

`viewState` controls which screen is shown. Most of the time the user is on `viewDashboard`, but creating a note switches to `viewNewNote`, pressing `s` opens `viewSearch`, and tag or type filtering switches to `viewFilter`. Each view state has its own rendering and input handling.

`focusPane` tracks which half of the split-pane layout has focus. The user can press Tab to toggle between the note list and the preview pane.

### The App Struct

The `App` struct is the Bubble Tea model. It holds everything the application needs:

```go
type App struct {
    // Dependencies -- injected at creation
    cfg       *config.Config
    store     *notes.NoteStore
    cal       *calendar.BusinessCal
    identity  *age.HybridIdentity
    notesDir  string
    configDir string

    // Runtime state
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

    // Modal overlays
    deleteConfirm    bool
    driveDelete      bool
    search           *SearchModel
    tagFilter        *FilterModel
    typeFilter       *FilterModel
    activeTagFilter  []string
    activeTypeFilter []string
}
```

The struct is organized into three logical sections:

**Dependencies** are the external services injected through `NewApp`. The TUI does not create these -- they are passed in from main. This makes the TUI testable (you can inject mocks) and keeps it decoupled from initialization logic.

**Runtime state** tracks what is happening right now: the loaded notes, which item the cursor is on, the terminal dimensions, and whether "done" notes should be visible.

**Modal overlays** manage transient UI states like delete confirmation, search, and filtering. When `deleteConfirm` is true, the help bar changes to show confirmation options. When `search` is non-nil, the search overlay takes over input handling.

The constructor initializes the model with sensible defaults:

```go
func NewApp(cfg *config.Config, store *notes.NoteStore, cal *calendar.BusinessCal,
    identity *age.HybridIdentity, notesDir, configDir string) *App {
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
```

### Init and Loading Notes

`Init` returns a command that loads notes asynchronously:

```go
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
```

In Bubble Tea, a `tea.Cmd` is a function that returns a `tea.Msg`. The framework runs it in a goroutine and delivers the result as a message to `Update`. This keeps the UI responsive -- decrypting hundreds of notes happens in the background while the terminal shows "Loading...".

The message types are simple structs:

```go
type notesLoadedMsg struct {
    notes []*notes.Note
}

type errMsg struct {
    err error
}
```

### Update -- Message Routing

The `Update` function is the brain of the application. It receives every message (key presses, async results, window resizes) and decides what to do:

```go
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // Phase 1: Route key presses to active overlays.
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
    // Similar delegation for viewSearch and viewFilter...

    // Phase 2: Main message switch.
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        a.width = msg.Width
        a.height = msg.Height
    case notesLoadedMsg:
        a.notes = msg.notes
        a.rebuildDashboard()
    case errMsg:
        a.err = msg.err
    case tea.KeyPressMsg:
        return a.handleKey(msg)
    }
    return a, nil
}
```

The routing strategy has two phases:

**Phase 1: Sub-component delegation.** If an overlay is active (new note wizard, search, or filter), key presses are forwarded to that overlay's own `Update` method. The parent does not know the overlay's internal logic -- it just asks "are you done?" When the overlay signals completion, the App transitions back to `viewDashboard` and nils out the overlay model. For search, the selected result is used to position the cursor on the matching dashboard item.

**Phase 2: Main switch.** Messages not intercepted by an overlay go through a type switch: `WindowSizeMsg` stores terminal dimensions, `notesLoadedMsg` triggers a dashboard rebuild, `errMsg` stores the error, and `KeyPressMsg` delegates to `handleKey`.

This pattern scales well. Adding a new overlay means adding a new `viewState` constant, a new field on `App`, and a new delegation block at the top of `Update`.

### handleKey -- Keyboard Input

The `handleKey` method processes key presses on the main dashboard:

The method starts by checking the `deleteConfirm` flag. When true, only `y`, `n`, and Escape are accepted -- all other keys are ignored. This is the **two-step delete** pattern: pressing `x` sets the flag and changes the help bar to "Delete [title]? [y]es [n]o". Only `y` actually performs the deletion. This prevents accidental data loss from a stray keypress.

The main key handling covers navigation (`Up`/`Down`), focus switching (`Tab`), and action keys:

```go
switch key.Code {
case 'n':
    a.newNote = NewNewNoteModel(a.cfg, a.store, a.notes)
    a.view = viewNewNote
case 'e', tea.KeyEnter:
    // Launch encrypted editor on selected note
    path := a.dashboardItems[a.cursor].Note.FilePath
    return a, func() tea.Msg {
        if err := editor.OpenEncrypted(a.cfg.Editor, path, a.identity); err != nil {
            return errMsg{err}
        }
        return editorClosedMsg{path: path}
    }
case 'm':
    // Toggle done/open status, then re-encrypt
case 'x':
    a.deleteConfirm = true
case 'q', tea.KeyEscape:
    return a, tea.Quit
}
```

Editor and status-toggle operations return `tea.Cmd` closures that run asynchronously. The closure captures the file path by value, ensuring the correct note is operated on even if the cursor moves before the operation completes.

### View -- Rendering the Screen

The `View` function transforms the current state into a string. Bubble Tea v2 uses `tea.View` (which wraps a string with metadata like `AltScreen`):

The View function first checks if an overlay is active -- if so, it renders that overlay's view instead of the dashboard. For the main dashboard, it computes the split-pane layout:

```go
listWidth := a.width / 3
previewWidth := a.width - listWidth - 3

content := lipgloss.JoinHorizontal(lipgloss.Top,
    borderStyle.Width(listWidth).Height(a.height-3).Render(list),
    previewBorderStyle.Width(previewWidth).Height(a.height-3).Render(preview),
)

return altView(lipgloss.JoinVertical(lipgloss.Left, content, help))
```

The list pane gets 1/3 of the terminal width, the preview pane gets the remaining 2/3 (minus 3 for borders). `lipgloss.JoinHorizontal` places them side by side, and `lipgloss.JoinVertical` stacks the panes above the help bar.

Every view is wrapped in `altView`, which enables **AltScreen mode** -- a terminal feature that gives the application a separate screen buffer. When the app exits, the terminal restores the original content, preserving your command history.

The `width == 0` guard handles the brief moment between startup and the first `WindowSizeMsg`, preventing division by zero.

### rebuildDashboard and applyFilters

When notes are loaded or filters change, the dashboard needs to be rebuilt:

```go
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
```

The cursor clamping at the end prevents the cursor from pointing at a nonexistent item after filtering reduces the list. If you had 10 notes with the cursor on item 8, and a filter narrows it to 5 notes, the cursor moves to item 4 (the last one).

The `applyFilters` function handles tag and type filters:

```go
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
```

Filters are combined with AND logic: a note must match both the type filter AND the tag filter to appear. Within each filter, matching is OR -- if you select tags "budget" and "quarterly", notes with either tag appear. The short-circuit `return allNotes` when no filters are active avoids allocating a new slice unnecessarily.

### The Style System

The visual appearance of the TUI is defined in `styles.go` using Lipgloss, a declarative styling library for terminal output:

```go
var (
    titleStyle = lipgloss.NewStyle().
        Bold(true).
        Foreground(lipgloss.Color("#FF6600")).
        Padding(0, 1)

    selectedStyle = lipgloss.NewStyle().
        Bold(true).
        Foreground(lipgloss.Color("#FFFFFF")).
        Background(lipgloss.Color("#5F5FD7"))

    normalStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("#CCCCCC"))

    dimStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("#666666"))

    overdueStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("#FF0000")).
        Bold(true)

    todayStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("#FF8800")).
        Bold(true)

    upcomingStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("#FFFF00"))

    tagStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("#87CEEB"))

    borderStyle = lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(lipgloss.Color("#5F5FD7"))

    helpStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("#666666")).
        Padding(0, 1)
)
```

Lipgloss styles work like CSS -- you chain method calls to set properties, and call `.Render(text)` to apply the style. All styles are defined as package-level variables so they are created once and reused throughout the application.

The color scheme uses semantic names: `overdueStyle` is red for urgency, `todayStyle` is orange for attention, `upcomingStyle` is yellow for awareness, `dimStyle` is gray for de-emphasis. This makes the code self-documenting -- you can read `overdueStyle.Render(dueText)` and immediately understand the intent.

## Relationships

- **NoteStore** is the data layer -- the TUI calls `List`, `Create`, `Update`, `Delete`, and `Search` through the store.
- **Editor** is invoked via `editor.OpenEncrypted` when the user presses `e` or Enter on a note. The TUI suspends while the editor is open, then reloads notes when it returns.
- **Config** provides the editor name, date format, and sync preferences.
- **BusinessCal** is passed through to components that need to calculate next occurrences for recurring notes.
- **Bubble Tea** (`charm.land/bubbletea/v2`) provides the runtime loop, terminal management, and message-passing infrastructure.
- **Lipgloss** (`charm.land/lipgloss/v2`) provides the styling system for colors, borders, padding, and layout composition.

## Key Takeaways

- **The Elm Architecture** (Model-Update-View) separates state from rendering, making TUI code predictable and testable.
- **Sub-component delegation** keeps `Update` manageable -- overlays handle their own input and signal completion to the parent.
- **AltScreen mode** gives the application a clean terminal buffer that restores the previous content on exit.
- **Two-step delete confirmation** prevents accidental data loss from a single keypress.
- **Cursor clamping** after filter changes prevents out-of-bounds errors when the list shrinks.

## Next Steps

With the TUI in place, pq-notes is a fully functional encrypted note-taking application. In the next chapter, we will explore the CLI commands that provide a non-interactive interface for scripting and automation, allowing you to create, list, and search notes from the command line.
