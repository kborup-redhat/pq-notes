package notes

import (
	"strings"
	"testing"
	"time"
)

func TestNoteType_String(t *testing.T) {
	tests := []struct {
		name     string
		noteType NoteType
		want     string
	}{
		{"meeting", Meeting, "meeting"},
		{"task", Task, "task"},
		{"reminder", Reminder, "reminder"},
		{"followup", Followup, "followup"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.noteType.String(); got != tt.want {
				t.Errorf("NoteType.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSanitizeCustomerName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"simple name", "Acme Corp", "acme-corp"},
		{"with spaces", "Big Company Inc", "big-company-inc"},
		{"with special chars", "Company & Co. Ltd!", "company-co-ltd"},
		{"multiple spaces", "Multiple   Spaces", "multiple-spaces"},
		{"leading/trailing spaces", "  Trimmed  ", "trimmed"},
		{"leading/trailing hyphens", "--hyphen--", "hyphen"},
		{"all special chars", "!@#$%^&*()", ""},
		{"mixed case", "CamelCase", "camelcase"},
		{"dots and underscores", "some.company_name", "somecompany_name"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SanitizeCustomerName(tt.input); got != tt.want {
				t.Errorf("SanitizeCustomerName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNoteFilename(t *testing.T) {
	created := time.Date(2026, 5, 6, 14, 30, 0, 0, time.UTC)
	tests := []struct {
		name    string
		title   string
		created time.Time
		want    string
	}{
		{"simple title", "Meeting with client", created, "2026-05-06-meeting-with-client.md.age"},
		{"special chars", "Q2 Review & Planning!", created, "2026-05-06-q2-review-planning.md.age"},
		{"multiple spaces", "Notes   from   call", created, "2026-05-06-notes-from-call.md.age"},
		{"uppercase", "IMPORTANT MEETING", created, "2026-05-06-important-meeting.md.age"},
		{"mixed", "Follow-up: Project X", created, "2026-05-06-follow-up-project-x.md.age"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NoteFilename(tt.title, tt.created); got != tt.want {
				t.Errorf("NoteFilename() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseNote_FullFrontmatter(t *testing.T) {
	content := `---
customer: acme-corp
type: meeting
created: 06-05-2026 14:30
due: 10-05-2026 09:00
repeat: weekly
tags:
  - planning
  - q2
status: open
priority: high
attendees:
  - john@acme.com
  - jane@acme.com
related: previous-meeting.md.age
---

# Q2 Planning Meeting

## Agenda
- Review Q1 results
- Plan Q2 objectives

## Notes
Discussion notes here.

## Action Items
- [ ] Follow up with team
`

	note, err := ParseNote(content, "02-01-2006 15:04")
	if err != nil {
		t.Fatalf("ParseNote() error = %v", err)
	}

	if note.Customer != "acme-corp" {
		t.Errorf("Customer = %v, want acme-corp", note.Customer)
	}
	if note.Type != Meeting {
		t.Errorf("Type = %v, want %v", note.Type, Meeting)
	}
	if note.CreatedRaw != "06-05-2026 14:30" {
		t.Errorf("CreatedRaw = %v, want 06-05-2026 14:30", note.CreatedRaw)
	}
	if note.DueRaw != "10-05-2026 09:00" {
		t.Errorf("DueRaw = %v, want 10-05-2026 09:00", note.DueRaw)
	}
	expectedCreated := time.Date(2026, 5, 6, 14, 30, 0, 0, time.UTC)
	if !note.Created.Equal(expectedCreated) {
		t.Errorf("Created = %v, want %v", note.Created, expectedCreated)
	}
	expectedDue := time.Date(2026, 5, 10, 9, 0, 0, 0, time.UTC)
	if !note.Due.Equal(expectedDue) {
		t.Errorf("Due = %v, want %v", note.Due, expectedDue)
	}
	if note.Repeat != "weekly" {
		t.Errorf("Repeat = %v, want weekly", note.Repeat)
	}
	if len(note.Tags) != 2 || note.Tags[0] != "planning" || note.Tags[1] != "q2" {
		t.Errorf("Tags = %v, want [planning q2]", note.Tags)
	}
	if note.Status != StatusOpen {
		t.Errorf("Status = %v, want %v", note.Status, StatusOpen)
	}
	if note.Priority != PriorityHigh {
		t.Errorf("Priority = %v, want %v", note.Priority, PriorityHigh)
	}
	if len(note.Attendees) != 2 || note.Attendees[0] != "john@acme.com" {
		t.Errorf("Attendees = %v, want [john@acme.com jane@acme.com]", note.Attendees)
	}
	if note.Related != "previous-meeting.md.age" {
		t.Errorf("Related = %v, want previous-meeting.md.age", note.Related)
	}
	if note.Title != "Q2 Planning Meeting" {
		t.Errorf("Title = %v, want Q2 Planning Meeting", note.Title)
	}
	if !strings.Contains(note.Body, "## Agenda") {
		t.Errorf("Body missing Agenda section")
	}
}

func TestParseNote_USDateFormat(t *testing.T) {
	content := `---
customer: test-customer
type: task
created: 05-06-2026 14:30
due: 05-10-2026
---

# Test Task
`

	note, err := ParseNote(content, "01-02-2006 15:04")
	if err != nil {
		t.Fatalf("ParseNote() error = %v", err)
	}

	expectedCreated := time.Date(2026, 5, 6, 14, 30, 0, 0, time.UTC)
	if !note.Created.Equal(expectedCreated) {
		t.Errorf("Created = %v, want %v", note.Created, expectedCreated)
	}
	expectedDue := time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)
	if !note.Due.Equal(expectedDue) {
		t.Errorf("Due = %v, want %v", note.Due, expectedDue)
	}
}

func TestParseNote_MinimalFrontmatter(t *testing.T) {
	content := `---
customer: test-customer
type: reminder
created: 06-05-2026
---

# Don't forget to call Bob
`

	note, err := ParseNote(content, "02-01-2006 15:04")
	if err != nil {
		t.Fatalf("ParseNote() error = %v", err)
	}

	if note.Customer != "test-customer" {
		t.Errorf("Customer = %v, want test-customer", note.Customer)
	}
	if note.Type != Reminder {
		t.Errorf("Type = %v, want %v", note.Type, Reminder)
	}
	if note.Title != "Don't forget to call Bob" {
		t.Errorf("Title = %v, want Don't forget to call Bob", note.Title)
	}
	if note.Due != (time.Time{}) {
		t.Errorf("Due should be zero value, got %v", note.Due)
	}
}

func TestGenerateTemplate_Meeting(t *testing.T) {
	created := time.Date(2026, 5, 6, 14, 30, 0, 0, time.UTC)
	due := time.Date(2026, 5, 10, 9, 0, 0, 0, time.UTC)
	note := &Note{
		Customer: "acme-corp",
		Type:     Meeting,
		Created:  created,
		Due:      due,
		Tags:     []string{"planning", "q2"},
		Status:   StatusOpen,
		Priority: PriorityHigh,
		Attendees: []string{"john@acme.com", "jane@acme.com"},
		Title:    "Q2 Planning Meeting",
	}

	content := GenerateTemplate(note, "02-01-2006 15:04")

	if !strings.Contains(content, "customer: acme-corp") {
		t.Errorf("Missing customer in frontmatter")
	}
	if !strings.Contains(content, "type: meeting") {
		t.Errorf("Missing type in frontmatter")
	}
	if !strings.Contains(content, "created: 06-05-2026 14:30") {
		t.Errorf("Missing created in frontmatter")
	}
	if !strings.Contains(content, "due: 10-05-2026 09:00") {
		t.Errorf("Missing due in frontmatter")
	}
	if !strings.Contains(content, "# Q2 Planning Meeting") {
		t.Errorf("Missing title")
	}
	if !strings.Contains(content, "## Agenda") {
		t.Errorf("Missing Agenda section")
	}
	if !strings.Contains(content, "## Notes") {
		t.Errorf("Missing Notes section")
	}
	if !strings.Contains(content, "## Action Items") {
		t.Errorf("Missing Action Items section")
	}
}

func TestGenerateTemplate_Task(t *testing.T) {
	created := time.Date(2026, 5, 6, 14, 30, 0, 0, time.UTC)
	note := &Note{
		Customer: "test-customer",
		Type:     Task,
		Created:  created,
		Status:   StatusOpen,
		Priority: PriorityNormal,
		Title:    "Implement feature X",
	}

	content := GenerateTemplate(note, "02-01-2006 15:04")

	if !strings.Contains(content, "type: task") {
		t.Errorf("Missing type in frontmatter")
	}
	if !strings.Contains(content, "# Implement feature X") {
		t.Errorf("Missing title")
	}
	if !strings.Contains(content, "## Description") {
		t.Errorf("Missing Description section")
	}
	if !strings.Contains(content, "## Acceptance Criteria") {
		t.Errorf("Missing Acceptance Criteria section")
	}
	if !strings.Contains(content, "## Notes") {
		t.Errorf("Missing Notes section")
	}
}

func TestGenerateTemplate_Reminder(t *testing.T) {
	created := time.Date(2026, 5, 6, 14, 30, 0, 0, time.UTC)
	note := &Note{
		Customer: "test-customer",
		Type:     Reminder,
		Created:  created,
		Status:   StatusOpen,
		Title:    "Call Bob",
	}

	content := GenerateTemplate(note, "02-01-2006 15:04")

	if !strings.Contains(content, "type: reminder") {
		t.Errorf("Missing type in frontmatter")
	}
	if !strings.Contains(content, "# Call Bob") {
		t.Errorf("Missing title")
	}
	// Reminder should just have the title, no extra sections
	if strings.Contains(content, "## ") {
		t.Errorf("Reminder should not have body sections")
	}
}

func TestGenerateTemplate_Followup(t *testing.T) {
	created := time.Date(2026, 5, 6, 14, 30, 0, 0, time.UTC)
	note := &Note{
		Customer: "test-customer",
		Type:     Followup,
		Created:  created,
		Status:   StatusOpen,
		Title:    "Follow up on project X",
	}

	content := GenerateTemplate(note, "02-01-2006 15:04")

	if !strings.Contains(content, "type: followup") {
		t.Errorf("Missing type in frontmatter")
	}
	if !strings.Contains(content, "# Follow up on project X") {
		t.Errorf("Missing title")
	}
	if !strings.Contains(content, "## What was agreed") {
		t.Errorf("Missing What was agreed section")
	}
	if !strings.Contains(content, "## What needs to happen") {
		t.Errorf("Missing What needs to happen section")
	}
	if !strings.Contains(content, "## Status update") {
		t.Errorf("Missing Status update section")
	}
}

func TestGenerateTemplate_OmitsEmptyFields(t *testing.T) {
	created := time.Date(2026, 5, 6, 14, 30, 0, 0, time.UTC)
	note := &Note{
		Customer: "test-customer",
		Type:     Reminder,
		Created:  created,
		Status:   StatusOpen,
		Title:    "Simple reminder",
		// No due, repeat, tags, priority, attendees, related
	}

	content := GenerateTemplate(note, "02-01-2006 15:04")

	if strings.Contains(content, "due:") {
		t.Errorf("Should not include empty due field")
	}
	if strings.Contains(content, "repeat:") {
		t.Errorf("Should not include empty repeat field")
	}
	if strings.Contains(content, "tags:") {
		t.Errorf("Should not include empty tags field")
	}
	if strings.Contains(content, "priority:") {
		t.Errorf("Should not include empty priority field")
	}
	if strings.Contains(content, "attendees:") {
		t.Errorf("Should not include empty attendees field")
	}
	if strings.Contains(content, "related:") {
		t.Errorf("Should not include empty related field")
	}
}
