package tui

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/kborup-redhat/pq-notes/internal/config"
	"github.com/kborup-redhat/pq-notes/internal/crypto"
)

type setupStep int

const (
	stepEncryptionKey setupStep = iota
	stepEditor
	stepDateFormat
	stepCountry
	stepWeekend
)

// setupCompleteMsg is sent when the setup wizard finishes successfully.
type setupCompleteMsg struct{}

// SetupModel implements the 5-step first-launch setup wizard.
type SetupModel struct {
	step          setupStep
	cfg           *config.Config
	notesDir      string
	configDir     string
	keyPath       string
	nameInput     string
	emailInput    string
	countryInput  string
	editorChoice  int     // 0=vi, 1=code
	dateChoice    int     // 0=EU, 1=US
	weekendDays   [7]bool // Mon-Sun
	weekendCursor int
	inputFocused  string // "name" or "email"
	err           error
	width         int
	height        int
}

// NewSetupModel creates a new setup wizard model.
func NewSetupModel(notesDir, configDir string) *SetupModel {
	return &SetupModel{
		step:         stepEncryptionKey,
		cfg:          &config.Config{},
		notesDir:     notesDir,
		configDir:    configDir,
		keyPath:      filepath.Join(configDir, "key.txt"),
		inputFocused: "name",
	}
}

func (m *SetupModel) Init() tea.Cmd {
	return nil
}

func (m *SetupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case setupCompleteMsg:
		return m, tea.Quit

	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m *SetupModel) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.Key()

	// Global: Ctrl+C / Escape to quit
	if key.Code == tea.KeyEscape {
		return m, tea.Quit
	}

	switch m.step {
	case stepEncryptionKey:
		return m.handleEncryptionKeyStep(key)
	case stepEditor:
		return m.handleEditorStep(key)
	case stepDateFormat:
		return m.handleDateFormatStep(key)
	case stepCountry:
		return m.handleCountryStep(key)
	case stepWeekend:
		return m.handleWeekendStep(key)
	}

	return m, nil
}

func (m *SetupModel) handleEncryptionKeyStep(key tea.Key) (tea.Model, tea.Cmd) {
	switch key.Code {
	case tea.KeyTab:
		if m.inputFocused == "name" {
			m.inputFocused = "email"
		} else {
			m.inputFocused = "name"
		}
	case tea.KeyEnter:
		if m.inputFocused == "name" {
			m.inputFocused = "email"
			return m, nil
		}
		// Validate
		if strings.TrimSpace(m.nameInput) == "" {
			m.err = fmt.Errorf("name is required")
			return m, nil
		}
		if strings.TrimSpace(m.emailInput) == "" {
			m.err = fmt.Errorf("email is required")
			return m, nil
		}
		m.err = nil
		m.step = stepEditor
	case tea.KeyBackspace:
		if m.inputFocused == "name" && len(m.nameInput) > 0 {
			m.nameInput = m.nameInput[:len(m.nameInput)-1]
		} else if m.inputFocused == "email" && len(m.emailInput) > 0 {
			m.emailInput = m.emailInput[:len(m.emailInput)-1]
		}
	default:
		text := key.Text
		if text != "" {
			if m.inputFocused == "name" {
				m.nameInput += text
			} else {
				m.emailInput += text
			}
		}
	}
	return m, nil
}

func (m *SetupModel) handleEditorStep(key tea.Key) (tea.Model, tea.Cmd) {
	switch key.Code {
	case tea.KeyUp:
		if m.editorChoice > 0 {
			m.editorChoice--
		}
	case tea.KeyDown:
		if m.editorChoice < 1 {
			m.editorChoice++
		}
	case tea.KeyEnter:
		if m.editorChoice == 0 {
			m.cfg.Editor = "vi"
		} else {
			m.cfg.Editor = "code"
		}
		m.step = stepDateFormat
	}
	return m, nil
}

func (m *SetupModel) handleDateFormatStep(key tea.Key) (tea.Model, tea.Cmd) {
	switch key.Code {
	case tea.KeyUp:
		if m.dateChoice > 0 {
			m.dateChoice--
		}
	case tea.KeyDown:
		if m.dateChoice < 1 {
			m.dateChoice++
		}
	case tea.KeyEnter:
		if m.dateChoice == 0 {
			m.cfg.DateFormat = "eu"
		} else {
			m.cfg.DateFormat = "us"
		}
		m.step = stepCountry
	}
	return m, nil
}

func (m *SetupModel) handleCountryStep(key tea.Key) (tea.Model, tea.Cmd) {
	switch key.Code {
	case tea.KeyEnter:
		code := strings.TrimSpace(m.countryInput)
		if len(code) != 2 {
			m.err = fmt.Errorf("country code must be exactly 2 characters")
			return m, nil
		}
		m.cfg.Country = strings.ToUpper(code)
		m.err = nil
		// Pre-select default weekend for this country
		defaultWeekend := config.DefaultWeekend(m.cfg.Country)
		m.weekendDays = weekendToBools(defaultWeekend)
		m.step = stepWeekend
	case tea.KeyBackspace:
		if len(m.countryInput) > 0 {
			m.countryInput = m.countryInput[:len(m.countryInput)-1]
		}
	default:
		text := key.Text
		if text != "" && len(m.countryInput) < 2 {
			// Only accept letters
			for _, r := range text {
				if unicode.IsLetter(r) && len(m.countryInput) < 2 {
					m.countryInput += strings.ToUpper(string(r))
				}
			}
		}
	}
	return m, nil
}

func (m *SetupModel) handleWeekendStep(key tea.Key) (tea.Model, tea.Cmd) {
	switch key.Code {
	case tea.KeyUp:
		if m.weekendCursor > 0 {
			m.weekendCursor--
		}
	case tea.KeyDown:
		if m.weekendCursor < 6 {
			m.weekendCursor++
		}
	case ' ':
		m.weekendDays[m.weekendCursor] = !m.weekendDays[m.weekendCursor]
	case tea.KeyEnter:
		return m, m.finishSetup
	}
	return m, nil
}

func (m *SetupModel) finishSetup() tea.Msg {
	// Set weekend days in config
	m.cfg.Weekend = boolsToWeekend(m.weekendDays)

	// Generate encryption key
	_, err := crypto.GenerateKey(m.keyPath)
	if err != nil {
		m.err = fmt.Errorf("failed to generate key: %w", err)
		return nil
	}

	// Save config
	if err := config.Save(m.cfg, m.configDir); err != nil {
		m.err = fmt.Errorf("failed to save config: %w", err)
		return nil
	}

	return setupCompleteMsg{}
}

func (m *SetupModel) View() tea.View {
	if m.width == 0 {
		return tea.NewView("Loading...")
	}

	var s strings.Builder

	// Header
	s.WriteString("\n")
	s.WriteString(titleStyle.Render("pq-notes Setup Wizard"))
	s.WriteString("\n\n")

	// Progress indicator
	steps := []string{"Key", "Editor", "Date", "Country", "Weekend"}
	var progress strings.Builder
	for i, name := range steps {
		if i == int(m.step) {
			progress.WriteString(selectedStyle.Render(fmt.Sprintf(" %d. %s ", i+1, name)))
		} else if i < int(m.step) {
			progress.WriteString(dimStyle.Render(fmt.Sprintf(" %d. %s ", i+1, name)))
		} else {
			progress.WriteString(helpStyle.Render(fmt.Sprintf(" %d. %s ", i+1, name)))
		}
		if i < len(steps)-1 {
			progress.WriteString(dimStyle.Render(" > "))
		}
	}
	s.WriteString(progress.String())
	s.WriteString("\n\n")

	// Step content
	switch m.step {
	case stepEncryptionKey:
		s.WriteString(m.viewEncryptionKeyStep())
	case stepEditor:
		s.WriteString(m.viewEditorStep())
	case stepDateFormat:
		s.WriteString(m.viewDateFormatStep())
	case stepCountry:
		s.WriteString(m.viewCountryStep())
	case stepWeekend:
		s.WriteString(m.viewWeekendStep())
	}

	// Error display
	if m.err != nil {
		s.WriteString("\n")
		s.WriteString(overdueStyle.Render("Error: " + m.err.Error()))
		s.WriteString("\n")
	}

	// Help
	s.WriteString("\n")
	s.WriteString(helpStyle.Render("[Esc] quit"))
	s.WriteString("\n")

	return tea.NewView(s.String())
}

func (m *SetupModel) viewEncryptionKeyStep() string {
	var s strings.Builder

	s.WriteString(headerStyle.Render("Step 1: Encryption Key"))
	s.WriteString("\n")
	s.WriteString(normalStyle.Render("Enter your name and email for key generation."))
	s.WriteString("\n\n")

	nameLabel := "  Name:  "
	emailLabel := "  Email: "

	nameValue := m.nameInput
	emailValue := m.emailInput

	inputStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#333333")).
		Padding(0, 1).
		Width(40)

	inactiveInputStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#CCCCCC")).
		Background(lipgloss.Color("#1A1A1A")).
		Padding(0, 1).
		Width(40)

	if m.inputFocused == "name" {
		s.WriteString(normalStyle.Render(nameLabel))
		s.WriteString(inputStyle.Render(nameValue + "_"))
		s.WriteString("\n")
		s.WriteString(normalStyle.Render(emailLabel))
		s.WriteString(inactiveInputStyle.Render(emailValue))
	} else {
		s.WriteString(normalStyle.Render(nameLabel))
		s.WriteString(inactiveInputStyle.Render(nameValue))
		s.WriteString("\n")
		s.WriteString(normalStyle.Render(emailLabel))
		s.WriteString(inputStyle.Render(emailValue + "_"))
	}

	s.WriteString("\n\n")
	s.WriteString(helpStyle.Render("[Tab] switch field  [Enter] next"))
	return s.String()
}

func (m *SetupModel) viewEditorStep() string {
	var s strings.Builder

	s.WriteString(headerStyle.Render("Step 2: Default Editor"))
	s.WriteString("\n")
	s.WriteString(normalStyle.Render("Choose your preferred text editor."))
	s.WriteString("\n\n")

	editors := []string{"vi", "code (VS Code)"}
	for i, editor := range editors {
		if i == m.editorChoice {
			s.WriteString(selectedStyle.Render(fmt.Sprintf("  > %s", editor)))
		} else {
			s.WriteString(normalStyle.Render(fmt.Sprintf("    %s", editor)))
		}
		s.WriteString("\n")
	}

	s.WriteString("\n")
	s.WriteString(helpStyle.Render("[Up/Down] select  [Enter] confirm"))
	return s.String()
}

func (m *SetupModel) viewDateFormatStep() string {
	var s strings.Builder

	s.WriteString(headerStyle.Render("Step 3: Date Format"))
	s.WriteString("\n")
	s.WriteString(normalStyle.Render("Choose your preferred date format."))
	s.WriteString("\n\n")

	formats := []string{"EU (DD-MM-YYYY)", "US (MM-DD-YYYY)"}
	for i, format := range formats {
		if i == m.dateChoice {
			s.WriteString(selectedStyle.Render(fmt.Sprintf("  > %s", format)))
		} else {
			s.WriteString(normalStyle.Render(fmt.Sprintf("    %s", format)))
		}
		s.WriteString("\n")
	}

	s.WriteString("\n")
	s.WriteString(helpStyle.Render("[Up/Down] select  [Enter] confirm"))
	return s.String()
}

func (m *SetupModel) viewCountryStep() string {
	var s strings.Builder

	s.WriteString(headerStyle.Render("Step 4: Country Code"))
	s.WriteString("\n")
	s.WriteString(normalStyle.Render("Enter your ISO 3166-1 alpha-2 country code (e.g., US, GB, DK)."))
	s.WriteString("\n\n")

	inputStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#333333")).
		Padding(0, 1).
		Width(10)

	s.WriteString(normalStyle.Render("  Country: "))
	s.WriteString(inputStyle.Render(m.countryInput + "_"))

	s.WriteString("\n\n")
	s.WriteString(helpStyle.Render("[Enter] confirm"))
	return s.String()
}

func (m *SetupModel) viewWeekendStep() string {
	var s strings.Builder

	s.WriteString(headerStyle.Render("Step 5: Weekend Days"))
	s.WriteString("\n")
	s.WriteString(normalStyle.Render("Select which days are weekend (non-working) days."))
	s.WriteString("\n\n")

	days := []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}
	for i, day := range days {
		checkbox := "[ ]"
		if m.weekendDays[i] {
			checkbox = "[x]"
		}
		line := fmt.Sprintf("  %s %s", checkbox, day)
		if i == m.weekendCursor {
			s.WriteString(selectedStyle.Render(line))
		} else {
			s.WriteString(normalStyle.Render(line))
		}
		s.WriteString("\n")
	}

	s.WriteString("\n")
	s.WriteString(helpStyle.Render("[Up/Down] navigate  [Space] toggle  [Enter] finish"))
	return s.String()
}

// weekendToBools converts a list of weekend day names to a [7]bool array (Mon-Sun).
func weekendToBools(weekend []string) [7]bool {
	var bools [7]bool
	dayMap := map[string]int{
		"monday": 0, "tuesday": 1, "wednesday": 2, "thursday": 3,
		"friday": 4, "saturday": 5, "sunday": 6,
	}
	for _, d := range weekend {
		if idx, ok := dayMap[strings.ToLower(d)]; ok {
			bools[idx] = true
		}
	}
	return bools
}

// boolsToWeekend converts a [7]bool array (Mon-Sun) to a list of weekend day names.
func boolsToWeekend(bools [7]bool) []string {
	days := []string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"}
	var weekend []string
	for i, b := range bools {
		if b {
			weekend = append(weekend, days[i])
		}
	}
	return weekend
}
