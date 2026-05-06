package notes

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// NoteType represents the type of a note
type NoteType string

const (
	Meeting  NoteType = "meeting"
	Task     NoteType = "task"
	Reminder NoteType = "reminder"
	Followup NoteType = "followup"
)

// String returns the string representation of NoteType
func (nt NoteType) String() string {
	return string(nt)
}

// Status represents the completion status of a note
type Status string

const (
	StatusOpen Status = "open"
	StatusDone Status = "done"
)

// Priority represents the priority level of a note
type Priority string

const (
	PriorityLow    Priority = "low"
	PriorityNormal Priority = "normal"
	PriorityHigh   Priority = "high"
	PriorityUrgent Priority = "urgent"
)

// Note represents a single note with frontmatter and body
type Note struct {
	Customer   string    `yaml:"customer"`
	Type       NoteType  `yaml:"type"`
	Created    time.Time `yaml:"-"`
	CreatedRaw string    `yaml:"created"`
	Due        time.Time `yaml:"-"`
	DueRaw     string    `yaml:"due,omitempty"`
	Repeat     string    `yaml:"repeat,omitempty"`
	Tags       []string  `yaml:"tags,omitempty"`
	Status     Status    `yaml:"status"`
	Priority   Priority  `yaml:"priority,omitempty"`
	Attendees  []string  `yaml:"attendees,omitempty"`
	Related    string    `yaml:"related,omitempty"`
	Title      string    `yaml:"-"`
	Body       string    `yaml:"-"`
	FilePath   string    `yaml:"-"`
}

// ParseNote parses a note file content into a Note struct
func ParseNote(content, dateFormat string) (*Note, error) {
	// Split on frontmatter delimiters
	parts := strings.Split(content, "---")
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid note format: missing frontmatter delimiters")
	}

	frontmatterContent := strings.TrimSpace(parts[1])
	bodyContent := strings.TrimSpace(strings.Join(parts[2:], "---"))

	var note Note
	if err := yaml.Unmarshal([]byte(frontmatterContent), &note); err != nil {
		return nil, fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	// Parse Created date from CreatedRaw
	if note.CreatedRaw != "" {
		created, err := parseDate(note.CreatedRaw, dateFormat)
		if err != nil {
			return nil, fmt.Errorf("failed to parse created date: %w", err)
		}
		note.Created = created
	}

	// Parse Due date from DueRaw if present
	if note.DueRaw != "" {
		due, err := parseDate(note.DueRaw, dateFormat)
		if err != nil {
			return nil, fmt.Errorf("failed to parse due date: %w", err)
		}
		note.Due = due
	}

	// Extract title from first "# " line in body
	lines := strings.Split(bodyContent, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			note.Title = strings.TrimSpace(strings.TrimPrefix(line, "# "))
			break
		}
	}

	note.Body = bodyContent

	return &note, nil
}

// parseDate attempts to parse a date string using the given format
// It tries both the full format (with time) and date-only format
func parseDate(dateStr, dateFormat string) (time.Time, error) {
	// Try parsing with the full format first
	t, err := time.Parse(dateFormat, dateStr)
	if err == nil {
		return t, nil
	}

	// Try parsing with date-only format (remove time component from format)
	dateOnlyFormat := strings.Fields(dateFormat)[0]
	t, err = time.Parse(dateOnlyFormat, dateStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse date %q with format %q: %w", dateStr, dateFormat, err)
	}

	return t, nil
}

// frontmatterOut is used for marshaling to avoid time.Time serialization issues
type frontmatterOut struct {
	Customer  string   `yaml:"customer"`
	Type      string   `yaml:"type"`
	Created   string   `yaml:"created"`
	Due       string   `yaml:"due,omitempty"`
	Repeat    string   `yaml:"repeat,omitempty"`
	Tags      []string `yaml:"tags,omitempty"`
	Status    string   `yaml:"status"`
	Priority  string   `yaml:"priority,omitempty"`
	Attendees []string `yaml:"attendees,omitempty"`
	Related   string   `yaml:"related,omitempty"`
}

// GenerateTemplate generates a full note content with YAML frontmatter and type-specific body template
func GenerateTemplate(note *Note, dateFormat string) string {
	var sb strings.Builder

	// Build frontmatter struct
	fm := frontmatterOut{
		Customer: note.Customer,
		Type:     note.Type.String(),
		Created:  formatDate(note.Created, dateFormat),
		Status:   string(note.Status),
	}

	// Add optional fields only if they have values
	if !note.Due.IsZero() {
		fm.Due = formatDate(note.Due, dateFormat)
	}
	if note.Repeat != "" {
		fm.Repeat = note.Repeat
	}
	if len(note.Tags) > 0 {
		fm.Tags = note.Tags
	}
	if note.Priority != "" {
		fm.Priority = string(note.Priority)
	}
	if len(note.Attendees) > 0 {
		fm.Attendees = note.Attendees
	}
	if note.Related != "" {
		fm.Related = note.Related
	}

	// Marshal frontmatter
	frontmatterBytes, _ := yaml.Marshal(&fm)

	// Write frontmatter
	sb.WriteString("---\n")
	sb.Write(frontmatterBytes)
	sb.WriteString("---\n\n")

	// Write title
	sb.WriteString("# ")
	sb.WriteString(note.Title)
	sb.WriteString("\n\n")

	// Write type-specific body template
	switch note.Type {
	case Meeting:
		sb.WriteString("## Agenda\n\n")
		sb.WriteString("## Notes\n\n")
		sb.WriteString("## Action Items\n")
	case Task:
		sb.WriteString("## Description\n\n")
		sb.WriteString("## Acceptance Criteria\n\n")
		sb.WriteString("## Notes\n")
	case Reminder:
		// Reminder just has the title, no additional sections
	case Followup:
		sb.WriteString("## What was agreed\n\n")
		sb.WriteString("## What needs to happen\n\n")
		sb.WriteString("## Status update\n")
	}

	return sb.String()
}

// formatDate formats a time.Time using the given format
func formatDate(t time.Time, dateFormat string) string {
	// Check if the time has hour/minute components
	if t.Hour() == 0 && t.Minute() == 0 {
		// Use date-only format
		dateOnlyFormat := strings.Fields(dateFormat)[0]
		return t.Format(dateOnlyFormat)
	}
	return t.Format(dateFormat)
}

// SanitizeCustomerName sanitizes a customer name for use in file paths
func SanitizeCustomerName(name string) string {
	// Convert to lowercase
	name = strings.ToLower(name)

	// Replace spaces with hyphens
	name = strings.ReplaceAll(name, " ", "-")

	// Remove special characters except hyphens and underscores
	reg := regexp.MustCompile(`[^a-z0-9\-_]`)
	name = reg.ReplaceAllString(name, "")

	// Replace multiple consecutive hyphens with a single hyphen
	reg = regexp.MustCompile(`-+`)
	name = reg.ReplaceAllString(name, "-")

	// Trim leading and trailing hyphens
	name = strings.Trim(name, "-")

	return name
}

// NoteFilename generates a filename for a note based on title and creation date
func NoteFilename(title string, created time.Time) string {
	// Convert title to lowercase
	slug := strings.ToLower(title)

	// Replace spaces with hyphens
	slug = strings.ReplaceAll(slug, " ", "-")

	// Remove special characters except hyphens
	reg := regexp.MustCompile(`[^a-z0-9\-]`)
	slug = reg.ReplaceAllString(slug, "")

	// Replace multiple consecutive hyphens with a single hyphen
	reg = regexp.MustCompile(`-+`)
	slug = reg.ReplaceAllString(slug, "-")

	// Trim leading and trailing hyphens
	slug = strings.Trim(slug, "-")

	// Format: YYYY-MM-DD-slug.md.age
	datePrefix := created.Format("2006-01-02")
	return fmt.Sprintf("%s-%s.md.age", datePrefix, slug)
}
