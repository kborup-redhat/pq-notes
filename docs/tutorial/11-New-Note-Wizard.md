---
title: "Chapter 11: New Note Wizard"
order: 11
---

# Chapter 11: New Note Wizard

Creating a note with many fields -- type, folder, title, due date, repeat schedule, tags, priority, attendees -- could be overwhelming if presented all at once. The **New Note Wizard** solves this by breaking creation into a guided sequence of steps, like a checkout flow in an online store. Each screen asks for one piece of information, and the wizard adapts its steps based on what you've already chosen.

Think of it as a smart interview: if you say you're creating a meeting, it asks about attendees. If you're creating a task, it asks about priority. If you skip the due date, it skips the repeat schedule entirely.

## How It Works

The wizard is a sub-component (`NewNoteModel`) that lives inside the TUI app. It uses a `newNoteStep` enum to track which screen is active. On each key press, the current step's handler either updates the input for that step or advances to the next step.

The magic is in three routing functions -- `nextAfterDue`, `nextAfterRepeat`, and `nextAfterTags` -- that decide which step comes next based on the note type and whether a due date was entered. This creates an adaptive flow where users only see steps relevant to their note.

## Code Deep Dive

### The Step Enum

Every screen in the wizard maps to a constant:

```go
type newNoteStep int

const (
    stepType      newNoteStep = iota // Choose meeting/task/reminder/follow-up
    stepFolder                       // Pick or type a folder name
    stepTitle                        // Enter the note title
    stepDue                          // Enter a due date (optional)
    stepRepeat                       // Repeat schedule (only with due date)
    stepTags                         // Comma-separated tags
    stepPriority                     // Low/Normal/High/Urgent (tasks only)
    stepAttendees                    // Meeting attendees (meetings only)
    stepRelated                      // Related note (follow-ups only)
    stepConfirm                      // Review and create
)
```

Not every note type visits every step. The order defined here is the maximum possible sequence; the routing functions skip steps that don't apply.

### The NewNoteModel

The wizard stores all collected inputs alongside its current step:

```go
type NewNoteModel struct {
    step          newNoteStep
    cfg           *config.Config
    store         *notes.NoteStore
    existingNotes []*notes.Note

    typeChoice        int
    folderInput       string
    folderSuggestions []string
    folderChoice      int
    titleInput        string
    dueInput          string
    repeatChoice      int
    customRepeat      string
    tagsInput         string
    priorityChoice    int
    attendeesInput    string
    relatedChoice     int

    err error
}
```

The `existingNotes` slice serves double duty: it provides folder suggestions (via `uniqueFolders`) and the list of notes a follow-up can link to.

### The Update Loop

The `Update` method is the wizard's brain. It receives a key press and routes to the handler for the current step:

```go
func (m *NewNoteModel) Update(msg tea.KeyPressMsg) (bool, tea.Cmd) {
    key := msg.Key()

    // Escape cancels the wizard from any step
    if key.Code == tea.KeyEscape {
        return true, nil
    }

    switch m.step {
    case stepType:
        return m.handleTypeStep(key)
    case stepFolder:
        return m.handleFolderStep(key)
    case stepTitle:
        return m.handleTitleStep(key)
    case stepDue:
        return m.handleDueStep(key)
    case stepRepeat:
        return m.handleRepeatStep(key)
    // ... remaining steps
    case stepConfirm:
        return m.handleConfirmStep(key)
    }

    return false, nil
}
```

Returning `(true, nil)` means "the wizard is done, go back to the dashboard." Returning `(true, cmd)` means "done, and also execute this command" (used when creating the note). Returning `(false, nil)` means "stay in the wizard."

### Adaptive Step Flow

The three routing functions control which steps the user sees:

```go
// nextAfterDue determines which step follows the Due step.
func (m *NewNoteModel) nextAfterDue() newNoteStep {
    if m.hasDue() {
        return stepRepeat
    }
    // No due date: skip repeat
    selectedType := noteTypeValues[m.typeChoice]
    if selectedType == notes.Task {
        return stepPriority
    }
    return stepTags
}

// nextAfterRepeat determines which step follows the Repeat step.
func (m *NewNoteModel) nextAfterRepeat() newNoteStep {
    selectedType := noteTypeValues[m.typeChoice]
    if selectedType == notes.Task {
        return stepPriority
    }
    return stepTags
}

// nextAfterTags determines which step follows the Tags step.
func (m *NewNoteModel) nextAfterTags() newNoteStep {
    selectedType := noteTypeValues[m.typeChoice]
    switch selectedType {
    case notes.Meeting:
        return stepAttendees
    case notes.Followup:
        return stepRelated
    default:
        return stepConfirm
    }
}
```

Here's how the flow branches:
- **No due date** -- skip Repeat entirely.
- **Task** -- show Priority (after Due/Repeat, before Tags).
- **Meeting** -- show Attendees (after Tags, before Confirm).
- **Follow-up** -- show Related Note (after Tags, before Confirm).
- **Reminder** -- goes straight from Tags to Confirm.

### Folder Picker with Suggestions

When the user reaches the folder step, the wizard collects unique folder names from existing notes:

```go
func (m *NewNoteModel) uniqueFolders() []string {
    seen := make(map[string]bool)
    for _, n := range m.existingNotes {
        if n.Folder != "" {
            seen[n.Folder] = true
        }
    }
    var folders []string
    for f := range seen {
        folders = append(folders, f)
    }
    sort.Strings(folders)
    return folders
}
```

The folder step supports two modes:
- **Selection mode** (`folderChoice >= 0`) -- use Up/Down to pick an existing folder.
- **Freeform mode** (`folderChoice == -1`) -- type a new folder name directly.

Typing any character resets `folderChoice` to -1, switching from selection to freeform input. This is handled in `handleFolderStep`:

```go
func (m *NewNoteModel) handleFolderStep(key tea.Key) (bool, tea.Cmd) {
    switch key.Code {
    case tea.KeyEnter:
        if m.folderChoice >= 0 && m.folderChoice < len(m.folderSuggestions) {
            m.folderInput = m.folderSuggestions[m.folderChoice]
        }
        if strings.TrimSpace(m.folderInput) == "" {
            m.err = fmt.Errorf("folder name is required")
            return false, nil
        }
        m.err = nil
        m.step = stepTitle
    case tea.KeyUp:
        if m.folderChoice > 0 {
            m.folderChoice--
        }
    case tea.KeyDown:
        if m.folderChoice < len(m.folderSuggestions)-1 {
            m.folderChoice++
        }
    case tea.KeyBackspace:
        m.folderChoice = -1
        if len(m.folderInput) > 0 {
            m.folderInput = removeLastRune(m.folderInput)
        }
    default:
        if key.Text != "" {
            m.folderChoice = -1
            m.folderInput += key.Text
        }
    }
    return false, nil
}
```

### Input Validation

Each step validates its input before advancing. For example, the due date step parses the input through `dateutil.ParseDate` to catch invalid dates:

```go
func (m *NewNoteModel) handleDueStep(key tea.Key) (bool, tea.Cmd) {
    switch key.Code {
    case tea.KeyEnter:
        m.err = nil
        input := strings.TrimSpace(m.dueInput)
        if input != "" && strings.ToLower(input) != "none" {
            _, err := dateutil.ParseDate(input, m.cfg.DateFormat, time.Now())
            if err != nil {
                m.err = fmt.Errorf("invalid date: %s", input)
                return false, nil
            }
        }
        m.step = m.nextAfterDue()
    // ...
    }
    return false, nil
}
```

The title and folder steps enforce non-empty values. An error message is stored in `m.err` and rendered in red below the current step.

### Creating the Note

When the user confirms, `createNote` assembles all the collected inputs into a `Note` struct:

```go
func (m *NewNoteModel) createNote() tea.Msg {
    now := time.Now()

    note := &notes.Note{
        Folder:  strings.TrimSpace(m.folderInput),
        Type:    noteTypeValues[m.typeChoice],
        Created: now,
        Status:  notes.StatusOpen,
        Title:   strings.TrimSpace(m.titleInput),
    }

    // Due date
    if m.hasDue() {
        due, err := dateutil.ParseDate(strings.TrimSpace(m.dueInput), m.cfg.DateFormat, now)
        if err != nil {
            return errMsg{err}
        }
        note.Due = due
    }

    // Repeat (only if due date was set and a repeat option was chosen)
    if m.hasDue() && m.repeatChoice > 0 {
        if repeatOptions[m.repeatChoice] == "Custom" {
            note.Repeat = strings.TrimSpace(m.customRepeat)
        } else {
            note.Repeat = strings.ToLower(repeatOptions[m.repeatChoice])
        }
    }

    // Tags
    if raw := strings.TrimSpace(m.tagsInput); raw != "" {
        for _, t := range strings.Split(raw, ",") {
            t = strings.TrimSpace(t)
            if t != "" {
                note.Tags = append(note.Tags, t)
            }
        }
    }

    // Priority (Task only)
    if note.Type == notes.Task {
        note.Priority = priorityValues[m.priorityChoice]
    }

    // Attendees (Meeting only)
    if note.Type == notes.Meeting {
        if raw := strings.TrimSpace(m.attendeesInput); raw != "" {
            for _, a := range strings.Split(raw, ",") {
                a = strings.TrimSpace(a)
                if a != "" {
                    note.Attendees = append(note.Attendees, a)
                }
            }
        }
    }

    // Related (Followup only)
    if note.Type == notes.Followup && m.relatedChoice > 0 && m.relatedChoice <= len(m.existingNotes) {
        related := m.existingNotes[m.relatedChoice-1]
        note.Related = related.FilePath
    }

    path, err := m.store.Create(note)
    if err != nil {
        return errMsg{err}
    }

    return noteCreatedMsg{path: path}
}
```

Notice how the function is guarded: priority is only set for tasks, attendees only for meetings, related note only for follow-ups. The repeat field is only populated when the user both entered a due date and chose a repeat option.

### Progress Indicator

The wizard shows a breadcrumb trail at the top (e.g., `Type > Folder > Title > Due > Tags > Confirm`) so the user knows where they are. The `visibleSteps` function computes which steps apply to the current note type:

```go
func (m *NewNoteModel) visibleSteps() []string {
    steps := []string{"Type", "Folder", "Title", "Due"}

    if m.hasDue() {
        steps = append(steps, "Repeat")
    }

    selectedType := noteTypeValues[m.typeChoice]
    if selectedType == notes.Task {
        steps = append(steps, "Priority")
    }

    steps = append(steps, "Tags")

    if selectedType == notes.Meeting {
        steps = append(steps, "Attendees")
    }
    if selectedType == notes.Followup {
        steps = append(steps, "Related")
    }

    steps = append(steps, "Confirm")
    return steps
}
```

The `stepIndex` function finds the current step's position in this list, and the `View` method renders completed steps as dimmed, the current step as highlighted, and future steps in the help style:

```go
for i, name := range stepNames {
    if i == currentIdx {
        progress.WriteString(selectedStyle.Render(fmt.Sprintf(" %s ", name)))
    } else if i < currentIdx {
        progress.WriteString(dimStyle.Render(fmt.Sprintf(" %s ", name)))
    } else {
        progress.WriteString(helpStyle.Render(fmt.Sprintf(" %s ", name)))
    }
    if i < len(stepNames)-1 {
        progress.WriteString(dimStyle.Render(" > "))
    }
}
```

## Relationships

- **App** (`app.go`) creates a `NewNoteModel` when the user presses `n`, and forwards key events to it while the wizard is active.
- **NoteStore** (`notes/store.go`) receives the assembled note via `store.Create()` and handles encryption and disk persistence.
- **Config** provides the date format for parsing due dates.
- **DateUtil** validates and parses the due date input.

## Key Takeaways

- **Adaptive flows reduce friction** -- users only see steps relevant to their note type, avoiding empty or confusing inputs.
- **Freeform + suggestion hybrid** -- the folder picker lets users select existing folders or type new ones, combining discovery with flexibility.
- **Validate before advancing** -- each step checks its input on Enter, preventing invalid data from reaching the creation step.
- **Progress indicators reduce anxiety** -- users always know where they are in the flow and how many steps remain.
- **Guard clauses in assembly** -- the `createNote` function only populates fields that apply to the chosen note type, keeping the data model clean.

## Next Steps

With notes created and displayed on the dashboard, users need ways to find specific notes quickly. In the next chapter, we'll build the **Search and Filter** system that lets users search by text or narrow down by tags and types.
