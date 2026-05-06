---
title: "Chapter 3: The Note Model"
order: 3
---

# Chapter 3: The Note Model

## Introduction

Think of a filing cabinet in an office. Each folder has a label on the tab -- "Meetings," "Tasks," "Follow-ups." Inside each folder, the documents follow a consistent format: a header block with metadata (date, attendees, priority) and the document body below. When you need to find something, you look at the labels and metadata first, then read the content.

pq-notes works the same way. Every note is a markdown file with a structured header (YAML frontmatter) and a body. The frontmatter is the label on the tab -- it tells the application what type of note this is, when it was created, when it is due, how important it is, and what tags it carries. The body is the actual content you write. This separation lets the TUI filter, sort, and display notes without parsing the full content of every file.

## How It Works

A note file looks like this when decrypted:

```markdown
---
folder: acme-corp
type: meeting
created: 06-05-2026
due: 10-05-2026
tags: [billing, quarterly]
status: open
priority: high
attendees: [alice, bob]
---

# Q2 Billing Review

## Agenda

- Review Q2 numbers
- Discuss billing integration

## Notes

## Action Items
```

The `---` delimiters separate the YAML frontmatter from the markdown body. The note model code is responsible for parsing this format, generating templates for new notes, and producing safe filenames.

## Code Deep Dive

All note model code lives in `internal/notes/note.go`.

### Type Enums

pq-notes defines three sets of typed constants using Go's `type` aliasing on `string`:

```go
// NoteType represents the type of a note
type NoteType string

const (
    Meeting  NoteType = "meeting"
    Task     NoteType = "task"
    Reminder NoteType = "reminder"
    Followup NoteType = "followup"
)
```

Each note type gets a different template body when created. Meetings get Agenda/Notes/Action Items sections. Tasks get Description/Acceptance Criteria/Notes. Reminders are minimal (just a title). Follow-ups get What was agreed/What needs to happen/Status update.

```go
// Status represents the completion status of a note
type Status string

const (
    StatusOpen Status = "open"
    StatusDone Status = "done"
)
```

Status is binary: notes are either open or done. The TUI defaults to showing only open notes, but you can press `a` to show all or `c` to show only closed notes.

```go
// Priority represents the priority level of a note
type Priority string

const (
    PriorityLow    Priority = "low"
    PriorityNormal Priority = "normal"
    PriorityHigh   Priority = "high"
    PriorityUrgent Priority = "urgent"
)
```

Priority levels get visual indicators in the TUI -- urgent notes are highlighted, high-priority notes are marked, and so on. Using typed string constants instead of plain strings gives you compile-time safety: if you typo `PriortiyHigh`, the compiler catches it.

### The Note Struct

The `Note` struct is the central data structure of the application:

```go
// Note represents a single note with frontmatter and body
type Note struct {
    Folder     string    `yaml:"folder"`
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
```

There is a pattern worth highlighting here: the **dual date fields**. Notice `Created time.Time` with `yaml:"-"` and `CreatedRaw string` with `yaml:"created"`. The `yaml:"-"` tag tells the YAML unmarshaler to skip that field entirely. So when YAML is parsed, the date string goes into `CreatedRaw`, and the code then manually parses it into `Created` as a proper `time.Time`. This approach avoids the pitfalls of automatic time parsing -- Go's YAML library would not know whether `06-05-2026` means June 5th or May 6th without context from the config's date format.

The same pattern applies to `Due`/`DueRaw`, and to `Title`, `Body`, and `FilePath`, which are all runtime fields that do not come from the YAML frontmatter.

### Parsing Notes

`ParseNote` turns a raw decrypted markdown string into a `Note` struct:

```go
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
```

The parsing logic follows these steps:

1. **Split on `---`** to separate frontmatter from body. The content before the first `---` is empty (notes start with `---`), `parts[1]` is the frontmatter, and `parts[2:]` is everything after the closing delimiter. Using `strings.Join(parts[2:], "---")` is clever -- it handles cases where the note body itself contains `---` (like horizontal rules in markdown).

2. **Unmarshal the YAML** into the `Note` struct. The raw string fields (`CreatedRaw`, `DueRaw`) get populated, but the `time.Time` fields are skipped thanks to `yaml:"-"`.

3. **Parse dates** using a private `parseDate` helper that respects the user's configured date format.

4. **Extract the title** by scanning the body for the first markdown heading (`# `). This means the title lives in the body content, not the frontmatter, keeping the frontmatter focused on metadata.

### The Private parseDate Helper

The note-local date parser tries the full format (with time) first, then falls back to date-only:

```go
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
        return time.Time{}, fmt.Errorf("failed to parse date %q with format %q: %w",
            dateStr, dateFormat, err)
    }

    return t, nil
}
```

This is different from the `dateutil.ParseDate` function (Chapter 4), which handles natural language. This private helper only handles formatted date strings because frontmatter always contains concrete dates, never "tomorrow" or "next week."

### Generating Templates

When you create a new note, `GenerateTemplate` produces a complete markdown document with frontmatter and a type-specific body:

```go
// GenerateTemplate generates a full note content with YAML frontmatter
// and type-specific body template
func GenerateTemplate(note *Note, dateFormat string) (string, error) {
    var sb strings.Builder

    // Build frontmatter struct
    fm := frontmatterOut{
        Folder:  note.Folder,
        Type:    note.Type.String(),
        Created: formatDate(note.Created, dateFormat),
        Status:  string(note.Status),
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

    frontmatterBytes, err := yaml.Marshal(&fm)
    if err != nil {
        return "", fmt.Errorf("marshal frontmatter: %w", err)
    }

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

    return sb.String(), nil
}
```

The function uses a separate `frontmatterOut` struct for marshaling rather than reusing `Note`. This avoids serialization issues with `time.Time` fields and gives precise control over which fields appear in the YAML output. Optional fields are only added when they have values, keeping the generated frontmatter clean.

The `switch` on `note.Type` produces different body scaffolds. This is what you see when you press `n` in the TUI and select a note type -- the template is pre-filled with the right sections for that type of note.

### Filename Sanitization

Two functions handle safe filename generation. `SanitizeFolderName` cleans folder names:

```go
// SanitizeFolderName sanitizes a folder name for use in file paths
func SanitizeFolderName(name string) string {
    name = strings.ToLower(name)
    name = strings.ReplaceAll(name, " ", "-")
    reg := regexp.MustCompile(`[^a-z0-9\-_]`)
    name = reg.ReplaceAllString(name, "")
    reg = regexp.MustCompile(`-+`)
    name = reg.ReplaceAllString(name, "-")
    name = strings.Trim(name, "-")
    return name
}
```

And `NoteFilename` generates a date-prefixed slug:

```go
// NoteFilename generates a filename for a note based on title and creation date
func NoteFilename(title string, created time.Time) string {
    slug := strings.ToLower(title)
    slug = strings.ReplaceAll(slug, " ", "-")
    reg := regexp.MustCompile(`[^a-z0-9\-]`)
    slug = reg.ReplaceAllString(slug, "")
    reg = regexp.MustCompile(`-+`)
    slug = reg.ReplaceAllString(slug, "-")
    slug = strings.Trim(slug, "-")
    datePrefix := created.Format("2006-01-02")
    return fmt.Sprintf("%s-%s.md.age", datePrefix, slug)
}
```

Given a title "Q2 Billing Review" and a creation date of May 6, 2026, this produces `2026-05-06-q2-billing-review.md.age`. The date prefix ensures chronological sorting in file listings, and the `.md.age` extension signals that it is encrypted markdown.

Both functions follow the same sanitization pipeline: lowercase, replace spaces with hyphens, strip special characters, collapse multiple hyphens, and trim edges. This is a defensive pattern -- user input can contain anything, but filenames need to be safe across operating systems.

## Relationships to Other Components

- **Configuration (Chapter 1):** The `DateFormat` from config is passed to both `ParseNote` and `GenerateTemplate` so dates in frontmatter match the user's locale.
- **Cryptography (Chapter 2):** Notes are stored encrypted. The NoteStore decrypts file contents before calling `ParseNote` and encrypts the output of `GenerateTemplate` before writing to disk.
- **Date Utilities (Chapter 4):** When the user enters a due date during note creation, the TUI uses `dateutil.ParseDate` to convert natural language like "next week" into a `time.Time`, which is then stored in the `Note` struct.
- **Business Calendar (Chapter 5):** Due dates and repeat schedules interact with the business calendar. The notification daemon uses the calendar to determine whether a due date falls on a workday.

## Key Takeaways

- Notes use a **frontmatter + body** pattern: YAML metadata above `---` delimiters, markdown content below.
- The **dual date field** pattern (`Created`/`CreatedRaw`) gives explicit control over date parsing, avoiding ambiguity between EU and US date formats.
- **Type-specific templates** (meeting, task, reminder, follow-up) scaffold the right sections when you create a note, reducing the friction of starting from a blank page.
- **Filename sanitization** ensures safe, sortable filenames across platforms: `2026-05-06-q2-billing-review.md.age`.
- A separate `frontmatterOut` struct for marshaling keeps the serialized YAML clean and avoids `time.Time` serialization issues.

## Next Steps

Notes have dates -- creation dates, due dates, and repeat schedules. But how does pq-notes understand "next Tuesday" or "tomorrow"? In [Chapter 4: Date Utilities](04-Date-Utilities.md), we will explore the date parsing and formatting system that powers the TUI's natural-language date input.
