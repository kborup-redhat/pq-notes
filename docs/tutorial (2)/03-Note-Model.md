---
title: "Chapter 3: Note Model"
order: 3
---

# Chapter 3: The Note Model

The Note model is the heart of pq-notes. It defines what a note *is* — its metadata, its body, and how it translates between a Go struct and a Markdown file with YAML frontmatter. In this chapter, we'll explore the data structures, parsing logic, and template generation that make notes work.

Think of the Note model as a blueprint that defines the shape of every note in the system, plus the machinery to convert between the on-disk format and the in-memory representation.

## How It Works

Each note is a Markdown file with YAML frontmatter at the top. Here's what a meeting note looks like on disk (before encryption):

```markdown
---
customer: acme-corp
type: meeting
created: 06-05-2026
due: 10-05-2026
tags:
  - quarterly-review
  - budget
status: open
priority: high
attendees:
  - alice@acme.com
  - bob@acme.com
---

# Q2 Budget Review

## Agenda

## Notes

## Action Items
```

The Note model handles three things:
1. **Type definitions** — enums for note type, status, and priority
2. **Parsing** — converting file content into a `Note` struct
3. **Template generation** — creating new notes with type-specific body sections
4. **Filename generation** — creating safe, date-prefixed filenames

## Code Deep Dive

### Type Definitions

pq-notes uses Go's type alias pattern to create enums:

```go
type NoteType string

const (
    Meeting  NoteType = "meeting"
    Task     NoteType = "task"
    Reminder NoteType = "reminder"
    Followup NoteType = "followup"
)

type Status string

const (
    StatusOpen Status = "open"
    StatusDone Status = "done"
)

type Priority string

const (
    PriorityLow    Priority = "low"
    PriorityNormal Priority = "normal"
    PriorityHigh   Priority = "high"
    PriorityUrgent Priority = "urgent"
)
```

Using typed strings instead of raw strings catches typos at compile time. You can't accidentally assign `"meating"` to a `NoteType` field without an explicit cast.

### The Note Struct

The `Note` struct combines YAML-serializable fields with computed fields:

```go
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
```

Notice the dual representation for dates:
- **`Created time.Time`** with `yaml:"-"` — the parsed Go time value, excluded from YAML serialization
- **`CreatedRaw string`** with `yaml:"created"` — the raw string from the YAML file

This pattern exists because `time.Time` doesn't serialize to the custom date formats pq-notes uses (EU: `DD-MM-YYYY`, US: `MM-DD-YYYY`). The raw string is what lives in the YAML; the parsed `time.Time` is what the application works with.

Fields tagged `yaml:"-"` are excluded from serialization entirely. `Title`, `Body`, and `FilePath` are derived from parsing — they don't exist in the frontmatter.

### Parsing a Note

The `ParseNote` function converts file content into a `Note` struct:

```go
func ParseNote(content, dateFormat string) (*Note, error) {
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

The parsing works in stages:

1. **Split on `---`** to separate frontmatter from body. The content is split into at least 3 parts: before the first `---` (empty), the frontmatter, and the body. Using `strings.Join(parts[2:], "---")` handles the case where the body itself contains `---` separators.

2. **Unmarshal YAML** into the Note struct. The `yaml` library matches struct tags to YAML keys automatically.

3. **Parse date strings** into `time.Time` values using a helper that tries both date-time and date-only formats.

4. **Extract the title** by scanning the body for the first Markdown heading (`# `).

### Template Generation

When creating a new note, `GenerateTemplate` builds the full file content:

```go
func GenerateTemplate(note *Note, dateFormat string) string {
    var sb strings.Builder

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
    // ... more optional fields ...

    frontmatterBytes, _ := yaml.Marshal(&fm)

    sb.WriteString("---\n")
    sb.Write(frontmatterBytes)
    sb.WriteString("---\n\n")

    sb.WriteString("# ")
    sb.WriteString(note.Title)
    sb.WriteString("\n\n")

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
        // Reminder just has the title
    case Followup:
        sb.WriteString("## What was agreed\n\n")
        sb.WriteString("## What needs to happen\n\n")
        sb.WriteString("## Status update\n")
    }

    return sb.String()
}
```

There are a few design choices worth noting:

- **`strings.Builder`** is used instead of string concatenation. While the difference is negligible for small outputs like this, `Builder` avoids creating intermediate string copies — a good habit for Go.

- **`frontmatterOut`** is a separate struct from `Note`, with all string fields. This avoids `time.Time` serialization issues and gives full control over which fields appear in the YAML output.

- **Type-specific body templates** use a `switch` statement to generate different Markdown sections based on the note type. Each type gets a scaffold that guides the user's writing.

### Safe Filenames

Two functions handle converting user-provided text into filesystem-safe names:

```go
func SanitizeCustomerName(name string) string {
    name = strings.ToLower(name)
    name = strings.ReplaceAll(name, " ", "-")
    reg := regexp.MustCompile(`[^a-z0-9\-_]`)
    name = reg.ReplaceAllString(name, "")
    reg = regexp.MustCompile(`-+`)
    name = reg.ReplaceAllString(name, "-")
    name = strings.Trim(name, "-")
    return name
}

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

Both functions follow the same sanitization pattern:
1. Lowercase everything
2. Replace spaces with hyphens
3. Strip special characters (keep only `a-z`, `0-9`, `-`, and optionally `_`)
4. Collapse multiple consecutive hyphens
5. Trim leading/trailing hyphens

`NoteFilename` adds a date prefix in ISO format (`YYYY-MM-DD`) for natural chronological sorting. The `.md.age` extension signals that it's an age-encrypted Markdown file.

Example: A meeting titled "Q2 Budget Review" created on May 6, 2026 becomes `2026-05-06-q2-budget-review.md.age`.

## Relationships

- **NoteStore** calls `ParseNote` to read notes and `GenerateTemplate` to create new ones. It uses `SanitizeCustomerName` for directory paths and `NoteFilename` for file naming.
- **DateUtil** provides the date formatting and parsing utilities used by the note's frontmatter serialization.
- **Editor** opens the decrypted note content for editing, then the modified content is parsed back through `ParseNote`.

## Key Takeaways

- **Dual representation** (raw string + parsed value) is a practical pattern when serialization format doesn't match the application's internal model.
- **`yaml:"-"`** excludes fields from serialization — useful for computed or derived values.
- **`omitempty`** keeps output clean by omitting zero-value optional fields.
- **Type aliases** (`type NoteType string`) create lightweight enums that prevent invalid values at the type level.
- **`strings.Builder`** is the idiomatic way to build strings incrementally in Go.
- **Filename sanitization** is essential when user input becomes part of filesystem paths.

## Next Steps

Notes need dates — creation dates, due dates, and natural language inputs like "tomorrow" or "next monday". In the next chapter, we'll build the date utility package that handles all of this.
