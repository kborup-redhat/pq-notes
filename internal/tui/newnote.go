package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/kborup-redhat/pq-notes/internal/config"
	"github.com/kborup-redhat/pq-notes/internal/dateutil"
	"github.com/kborup-redhat/pq-notes/internal/notes"
)

type newNoteStep int

const (
	stepType newNoteStep = iota
	stepFolder
	stepTitle
	stepDue
	stepRepeat
	stepTags
	stepPriority
	stepAttendees
	stepRelated
	stepConfirm
)

// noteCreatedMsg is sent when a new note is successfully created.
type noteCreatedMsg struct {
	path string
}

// editorClosedMsg is sent when the editor closes after editing a new note.
type editorClosedMsg struct {
	path string
}

// NewNoteModel is a sub-component that collects metadata for a new note
// via a multi-step wizard. It is not a tea.Model; its Update method takes
// a tea.KeyPressMsg and returns (done bool, cmd tea.Cmd).
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
	dueInput       string
	repeatChoice   int
	customRepeat   string
	tagsInput      string
	priorityChoice int
	attendeesInput string
	relatedChoice  int

	err error
}

// NewNewNoteModel creates a new wizard model with the given dependencies.
func NewNewNoteModel(cfg *config.Config, store *notes.NoteStore, existing []*notes.Note) *NewNoteModel {
	return &NewNoteModel{
		step:          stepType,
		cfg:           cfg,
		store:         store,
		existingNotes: existing,
	}
}

var noteTypes = []string{"Meeting", "Task", "Reminder", "Follow-up"}
var noteTypeValues = []notes.NoteType{notes.Meeting, notes.Task, notes.Reminder, notes.Followup}
var repeatOptions = []string{"None", "Daily", "Weekly", "Monthly", "Custom"}
var priorityOptions = []string{"Low", "Normal", "High", "Urgent"}
var priorityValues = []notes.Priority{notes.PriorityLow, notes.PriorityNormal, notes.PriorityHigh, notes.PriorityUrgent}

// Update processes a key press and advances the wizard. Returns done=true
// when the wizard is finished (note created or cancelled), along with any
// tea.Cmd to execute.
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
	case stepTags:
		return m.handleTagsStep(key)
	case stepPriority:
		return m.handlePriorityStep(key)
	case stepAttendees:
		return m.handleAttendeesStep(key)
	case stepRelated:
		return m.handleRelatedStep(key)
	case stepConfirm:
		return m.handleConfirmStep(key)
	}

	return false, nil
}

func (m *NewNoteModel) handleTypeStep(key tea.Key) (bool, tea.Cmd) {
	switch key.Code {
	case tea.KeyUp:
		if m.typeChoice > 0 {
			m.typeChoice--
		}
	case tea.KeyDown:
		if m.typeChoice < len(noteTypes)-1 {
			m.typeChoice++
		}
	case tea.KeyEnter:
		m.folderSuggestions = m.uniqueFolders()
		m.folderChoice = -1
		m.step = stepFolder
	}
	return false, nil
}

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

func (m *NewNoteModel) handleTitleStep(key tea.Key) (bool, tea.Cmd) {
	switch key.Code {
	case tea.KeyEnter:
		if strings.TrimSpace(m.titleInput) == "" {
			m.err = fmt.Errorf("title is required")
			return false, nil
		}
		m.err = nil
		m.step = stepDue
	case tea.KeyBackspace:
		if len(m.titleInput) > 0 {
			m.titleInput = removeLastRune(m.titleInput)
		}
	default:
		if key.Text != "" {
			m.titleInput += key.Text
		}
	}
	return false, nil
}

func (m *NewNoteModel) handleDueStep(key tea.Key) (bool, tea.Cmd) {
	switch key.Code {
	case tea.KeyEnter:
		m.err = nil
		// Validate the date input if non-empty
		input := strings.TrimSpace(m.dueInput)
		if input != "" && strings.ToLower(input) != "none" {
			_, err := dateutil.ParseDate(input, m.cfg.DateFormat, time.Now())
			if err != nil {
				m.err = fmt.Errorf("invalid date: %s", input)
				return false, nil
			}
		}
		m.step = m.nextAfterDue()
	case tea.KeyBackspace:
		if len(m.dueInput) > 0 {
			m.dueInput = removeLastRune(m.dueInput)
		}
	default:
		if key.Text != "" {
			m.dueInput += key.Text
		}
	}
	return false, nil
}

func (m *NewNoteModel) handleRepeatStep(key tea.Key) (bool, tea.Cmd) {
	switch key.Code {
	case tea.KeyUp:
		if m.repeatChoice > 0 {
			m.repeatChoice--
		}
	case tea.KeyDown:
		if m.repeatChoice < len(repeatOptions)-1 {
			m.repeatChoice++
		}
	case tea.KeyEnter:
		if repeatOptions[m.repeatChoice] == "Custom" {
			// If custom is selected but no text entered yet, stay to collect it
			if strings.TrimSpace(m.customRepeat) == "" {
				m.err = fmt.Errorf("enter a custom repeat value (e.g., 'every 2 weeks')")
				return false, nil
			}
			m.err = nil
		}
		m.step = m.nextAfterRepeat()
	case tea.KeyBackspace:
		if repeatOptions[m.repeatChoice] == "Custom" && len(m.customRepeat) > 0 {
			m.customRepeat = removeLastRune(m.customRepeat)
		}
	default:
		if key.Text != "" && repeatOptions[m.repeatChoice] == "Custom" {
			m.customRepeat += key.Text
			m.err = nil
		}
	}
	return false, nil
}

func (m *NewNoteModel) handleTagsStep(key tea.Key) (bool, tea.Cmd) {
	switch key.Code {
	case tea.KeyEnter:
		m.err = nil
		m.step = m.nextAfterTags()
	case tea.KeyBackspace:
		if len(m.tagsInput) > 0 {
			m.tagsInput = removeLastRune(m.tagsInput)
		}
	default:
		if key.Text != "" {
			m.tagsInput += key.Text
		}
	}
	return false, nil
}

func (m *NewNoteModel) handlePriorityStep(key tea.Key) (bool, tea.Cmd) {
	switch key.Code {
	case tea.KeyUp:
		if m.priorityChoice > 0 {
			m.priorityChoice--
		}
	case tea.KeyDown:
		if m.priorityChoice < len(priorityOptions)-1 {
			m.priorityChoice++
		}
	case tea.KeyEnter:
		m.step = stepTags
	}
	return false, nil
}

func (m *NewNoteModel) handleAttendeesStep(key tea.Key) (bool, tea.Cmd) {
	switch key.Code {
	case tea.KeyEnter:
		m.err = nil
		m.step = stepConfirm
	case tea.KeyBackspace:
		if len(m.attendeesInput) > 0 {
			m.attendeesInput = removeLastRune(m.attendeesInput)
		}
	default:
		if key.Text != "" {
			m.attendeesInput += key.Text
		}
	}
	return false, nil
}

func (m *NewNoteModel) handleRelatedStep(key tea.Key) (bool, tea.Cmd) {
	switch key.Code {
	case tea.KeyUp:
		if m.relatedChoice > 0 {
			m.relatedChoice--
		}
	case tea.KeyDown:
		max := len(m.existingNotes)
		if m.relatedChoice < max {
			m.relatedChoice++
		}
	case tea.KeyEnter:
		m.step = stepConfirm
	}
	return false, nil
}

func (m *NewNoteModel) handleConfirmStep(key tea.Key) (bool, tea.Cmd) {
	switch key.Code {
	case tea.KeyEnter:
		return true, m.createNote
	}
	return false, nil
}

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

// hasDue returns true if the user entered a due date.
func (m *NewNoteModel) hasDue() bool {
	input := strings.TrimSpace(m.dueInput)
	return input != "" && strings.ToLower(input) != "none"
}

// createNote builds the Note from wizard inputs and persists it via the store.
func (m *NewNoteModel) createNote() tea.Msg {
	now := time.Now()

	note := &notes.Note{
		Folder: strings.TrimSpace(m.folderInput),
		Type:     noteTypeValues[m.typeChoice],
		Created:  now,
		Status:   notes.StatusOpen,
		Title:    strings.TrimSpace(m.titleInput),
	}

	// Due date
	if m.hasDue() {
		due, err := dateutil.ParseDate(strings.TrimSpace(m.dueInput), m.cfg.DateFormat, now)
		if err != nil {
			return errMsg{err}
		}
		note.Due = due
	}

	// Repeat
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

// View renders the current wizard step.
func (m *NewNoteModel) View() string {
	var s strings.Builder

	s.WriteString("\n")
	s.WriteString(titleStyle.Render("New Note"))
	s.WriteString("\n\n")

	// Progress indicator
	stepNames := m.visibleSteps()
	var progress strings.Builder
	currentIdx := m.stepIndex()
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
	s.WriteString(progress.String())
	s.WriteString("\n\n")

	// Step content
	switch m.step {
	case stepType:
		s.WriteString(m.viewTypeStep())
	case stepFolder:
		s.WriteString(m.viewFolderStep())
	case stepTitle:
		s.WriteString(m.viewTitleStep())
	case stepDue:
		s.WriteString(m.viewDueStep())
	case stepRepeat:
		s.WriteString(m.viewRepeatStep())
	case stepTags:
		s.WriteString(m.viewTagsStep())
	case stepPriority:
		s.WriteString(m.viewPriorityStep())
	case stepAttendees:
		s.WriteString(m.viewAttendeesStep())
	case stepRelated:
		s.WriteString(m.viewRelatedStep())
	case stepConfirm:
		s.WriteString(m.viewConfirmStep())
	}

	// Error display
	if m.err != nil {
		s.WriteString("\n")
		s.WriteString(overdueStyle.Render("Error: " + m.err.Error()))
		s.WriteString("\n")
	}

	// Help
	s.WriteString("\n")
	s.WriteString(helpStyle.Render("[Esc] cancel"))
	s.WriteString("\n")

	return s.String()
}

func (m *NewNoteModel) viewTypeStep() string {
	var s strings.Builder
	s.WriteString(headerStyle.Render("Note Type"))
	s.WriteString("\n")
	s.WriteString(normalStyle.Render("Choose the type of note to create."))
	s.WriteString("\n\n")

	for i, t := range noteTypes {
		if i == m.typeChoice {
			s.WriteString(selectedStyle.Render(fmt.Sprintf("  > %s", t)))
		} else {
			s.WriteString(normalStyle.Render(fmt.Sprintf("    %s", t)))
		}
		s.WriteString("\n")
	}

	s.WriteString("\n")
	s.WriteString(helpStyle.Render("[Up/Down] select  [Enter] confirm"))
	return s.String()
}

func (m *NewNoteModel) viewFolderStep() string {
	var s strings.Builder
	s.WriteString(headerStyle.Render("Folder"))
	s.WriteString("\n")
	s.WriteString(normalStyle.Render("Select an existing folder or type a new name."))
	s.WriteString("\n\n")

	if len(m.folderSuggestions) > 0 {
		for i, f := range m.folderSuggestions {
			if i == m.folderChoice {
				s.WriteString(selectedStyle.Render(fmt.Sprintf("  > %s", f)))
			} else {
				s.WriteString(normalStyle.Render(fmt.Sprintf("    %s", f)))
			}
			s.WriteString("\n")
		}
		s.WriteString("\n")
	}

	inputStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#333333")).
		Padding(0, 1).
		Width(40)

	s.WriteString(normalStyle.Render("  Folder: "))
	if m.folderChoice >= 0 && m.folderChoice < len(m.folderSuggestions) {
		s.WriteString(inputStyle.Render(m.folderSuggestions[m.folderChoice] + "_"))
	} else {
		s.WriteString(inputStyle.Render(m.folderInput + "_"))
	}

	s.WriteString("\n\n")
	s.WriteString(helpStyle.Render("[Up/Down] select existing  [Enter] next"))
	return s.String()
}

func (m *NewNoteModel) viewTitleStep() string {
	var s strings.Builder
	s.WriteString(headerStyle.Render("Title"))
	s.WriteString("\n")
	s.WriteString(normalStyle.Render("Enter a title for the note."))
	s.WriteString("\n\n")

	inputStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#333333")).
		Padding(0, 1).
		Width(50)

	s.WriteString(normalStyle.Render("  Title: "))
	s.WriteString(inputStyle.Render(m.titleInput + "_"))

	s.WriteString("\n\n")
	s.WriteString(helpStyle.Render("[Enter] next"))
	return s.String()
}

func (m *NewNoteModel) viewDueStep() string {
	var s strings.Builder
	s.WriteString(headerStyle.Render("Due Date"))
	s.WriteString("\n")
	s.WriteString(normalStyle.Render("Enter a due date, or leave empty for none."))
	s.WriteString("\n")
	s.WriteString(dimStyle.Render("  Accepts: tomorrow, friday, 25-12-2026, none"))
	s.WriteString("\n\n")

	inputStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#333333")).
		Padding(0, 1).
		Width(30)

	s.WriteString(normalStyle.Render("  Due: "))
	s.WriteString(inputStyle.Render(m.dueInput + "_"))

	s.WriteString("\n\n")
	s.WriteString(helpStyle.Render("[Enter] next"))
	return s.String()
}

func (m *NewNoteModel) viewRepeatStep() string {
	var s strings.Builder
	s.WriteString(headerStyle.Render("Repeat"))
	s.WriteString("\n")
	s.WriteString(normalStyle.Render("How often should this note repeat?"))
	s.WriteString("\n\n")

	for i, opt := range repeatOptions {
		if i == m.repeatChoice {
			s.WriteString(selectedStyle.Render(fmt.Sprintf("  > %s", opt)))
		} else {
			s.WriteString(normalStyle.Render(fmt.Sprintf("    %s", opt)))
		}
		s.WriteString("\n")
	}

	if repeatOptions[m.repeatChoice] == "Custom" {
		s.WriteString("\n")
		inputStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#333333")).
			Padding(0, 1).
			Width(30)
		s.WriteString(normalStyle.Render("  Custom: "))
		s.WriteString(inputStyle.Render(m.customRepeat + "_"))
	}

	s.WriteString("\n\n")
	s.WriteString(helpStyle.Render("[Up/Down] select  [Enter] confirm"))
	return s.String()
}

func (m *NewNoteModel) viewTagsStep() string {
	var s strings.Builder
	s.WriteString(headerStyle.Render("Tags"))
	s.WriteString("\n")
	s.WriteString(normalStyle.Render("Enter tags, separated by commas (optional)."))
	s.WriteString("\n\n")

	inputStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#333333")).
		Padding(0, 1).
		Width(50)

	s.WriteString(normalStyle.Render("  Tags: "))
	s.WriteString(inputStyle.Render(m.tagsInput + "_"))

	s.WriteString("\n\n")
	s.WriteString(helpStyle.Render("[Enter] next"))
	return s.String()
}

func (m *NewNoteModel) viewPriorityStep() string {
	var s strings.Builder
	s.WriteString(headerStyle.Render("Priority"))
	s.WriteString("\n")
	s.WriteString(normalStyle.Render("Select the priority level for this task."))
	s.WriteString("\n\n")

	for i, opt := range priorityOptions {
		if i == m.priorityChoice {
			s.WriteString(selectedStyle.Render(fmt.Sprintf("  > %s", opt)))
		} else {
			s.WriteString(normalStyle.Render(fmt.Sprintf("    %s", opt)))
		}
		s.WriteString("\n")
	}

	s.WriteString("\n")
	s.WriteString(helpStyle.Render("[Up/Down] select  [Enter] confirm"))
	return s.String()
}

func (m *NewNoteModel) viewAttendeesStep() string {
	var s strings.Builder
	s.WriteString(headerStyle.Render("Attendees"))
	s.WriteString("\n")
	s.WriteString(normalStyle.Render("Enter attendees, separated by commas (optional)."))
	s.WriteString("\n\n")

	inputStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#333333")).
		Padding(0, 1).
		Width(50)

	s.WriteString(normalStyle.Render("  Attendees: "))
	s.WriteString(inputStyle.Render(m.attendeesInput + "_"))

	s.WriteString("\n\n")
	s.WriteString(helpStyle.Render("[Enter] next"))
	return s.String()
}

func (m *NewNoteModel) viewRelatedStep() string {
	var s strings.Builder
	s.WriteString(headerStyle.Render("Related Note"))
	s.WriteString("\n")
	s.WriteString(normalStyle.Render("Select a related note for this follow-up (optional)."))
	s.WriteString("\n\n")

	// First option is "None"
	if m.relatedChoice == 0 {
		s.WriteString(selectedStyle.Render("  > (None)"))
	} else {
		s.WriteString(normalStyle.Render("    (None)"))
	}
	s.WriteString("\n")

	for i, n := range m.existingNotes {
		label := fmt.Sprintf("%s - %s", n.Folder, n.Title)
		if i+1 == m.relatedChoice {
			s.WriteString(selectedStyle.Render(fmt.Sprintf("  > %s", label)))
		} else {
			s.WriteString(normalStyle.Render(fmt.Sprintf("    %s", label)))
		}
		s.WriteString("\n")
	}

	s.WriteString("\n")
	s.WriteString(helpStyle.Render("[Up/Down] select  [Enter] confirm"))
	return s.String()
}

func (m *NewNoteModel) viewConfirmStep() string {
	var s strings.Builder
	s.WriteString(headerStyle.Render("Confirm"))
	s.WriteString("\n")
	s.WriteString(normalStyle.Render("Review your note and press Enter to create it."))
	s.WriteString("\n\n")

	summaryLabel := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#87CEEB")).
		Bold(true)

	s.WriteString(summaryLabel.Render("  Type:     "))
	s.WriteString(normalStyle.Render(noteTypes[m.typeChoice]))
	s.WriteString("\n")

	s.WriteString(summaryLabel.Render("  Folder:   "))
	s.WriteString(normalStyle.Render(m.folderInput))
	s.WriteString("\n")

	s.WriteString(summaryLabel.Render("  Title:    "))
	s.WriteString(normalStyle.Render(m.titleInput))
	s.WriteString("\n")

	dueDisplay := "(none)"
	if m.hasDue() {
		dueDisplay = m.dueInput
	}
	s.WriteString(summaryLabel.Render("  Due:      "))
	s.WriteString(normalStyle.Render(dueDisplay))
	s.WriteString("\n")

	if m.hasDue() && m.repeatChoice > 0 {
		repeatDisplay := repeatOptions[m.repeatChoice]
		if repeatOptions[m.repeatChoice] == "Custom" {
			repeatDisplay = m.customRepeat
		}
		s.WriteString(summaryLabel.Render("  Repeat:   "))
		s.WriteString(normalStyle.Render(repeatDisplay))
		s.WriteString("\n")
	}

	if strings.TrimSpace(m.tagsInput) != "" {
		s.WriteString(summaryLabel.Render("  Tags:     "))
		s.WriteString(normalStyle.Render(m.tagsInput))
		s.WriteString("\n")
	}

	if noteTypeValues[m.typeChoice] == notes.Task {
		s.WriteString(summaryLabel.Render("  Priority: "))
		s.WriteString(normalStyle.Render(priorityOptions[m.priorityChoice]))
		s.WriteString("\n")
	}

	if noteTypeValues[m.typeChoice] == notes.Meeting && strings.TrimSpace(m.attendeesInput) != "" {
		s.WriteString(summaryLabel.Render("  Attendees:"))
		s.WriteString(normalStyle.Render(" " + m.attendeesInput))
		s.WriteString("\n")
	}

	if noteTypeValues[m.typeChoice] == notes.Followup && m.relatedChoice > 0 && m.relatedChoice <= len(m.existingNotes) {
		related := m.existingNotes[m.relatedChoice-1]
		s.WriteString(summaryLabel.Render("  Related:  "))
		s.WriteString(normalStyle.Render(fmt.Sprintf("%s - %s", related.Folder, related.Title)))
		s.WriteString("\n")
	}

	s.WriteString("\n")
	s.WriteString(helpStyle.Render("[Enter] create note  [Esc] cancel"))
	return s.String()
}

// visibleSteps returns the step names that apply to the current note type.
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

// stepIndex returns the index of the current step within the visible steps list.
func (m *NewNoteModel) stepIndex() int {
	steps := m.visibleSteps()
	name := m.currentStepName()
	for i, s := range steps {
		if s == name {
			return i
		}
	}
	return 0
}

func (m *NewNoteModel) currentStepName() string {
	switch m.step {
	case stepType:
		return "Type"
	case stepFolder:
		return "Folder"
	case stepTitle:
		return "Title"
	case stepDue:
		return "Due"
	case stepRepeat:
		return "Repeat"
	case stepTags:
		return "Tags"
	case stepPriority:
		return "Priority"
	case stepAttendees:
		return "Attendees"
	case stepRelated:
		return "Related"
	case stepConfirm:
		return "Confirm"
	}
	return ""
}
