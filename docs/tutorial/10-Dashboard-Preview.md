---
title: "Chapter 10: Dashboard & Preview"
order: 10
---

# Chapter 10: Dashboard & Preview

When you open a task manager, the first thing you see is a prioritized overview of what needs your attention. The most urgent items sit at the top, the less pressing ones below, and items with no deadline drift to the bottom. In pq-notes, the **Dashboard** is that prioritized overview, and the **Preview** pane is the magnifying glass you hold up to any item to see its full details.

Think of the Dashboard as a hospital emergency room triage board: patients (notes) are classified by urgency and displayed in groups, so the staff (you) always knows what to handle first.

## How It Works

The Dashboard system has two layers:

1. **Data layer** (`BuildDashboard`) -- classifies every note into an urgency group, sorts them, and assigns display indices.
2. **Rendering layer** (`RenderDashboard`, `renderDashboardItem`) -- converts the sorted data into styled terminal output with group headers, cursor highlighting, and done-dimming.

The Preview pane sits beside the dashboard. When a note is selected, `RenderPreview` assembles a Markdown string from the note's metadata and body, then passes it through Glamour for rich terminal rendering. A cached renderer avoids re-creating the Glamour engine on every keystroke.

## Code Deep Dive

### The Urgency Group Enum

Notes are classified into exactly four urgency groups using a Go `iota` enum:

```go
type urgencyGroup int

const (
    groupOverdue  urgencyGroup = iota // 0 — past due date
    groupToday                        // 1 — due today
    groupUpcoming                     // 2 — due in the future
    groupNoDue                        // 3 — no due date set
)
```

Because `iota` assigns increasing integers, the natural numeric order matches the desired display order: overdue first, then today, upcoming, and finally notes with no due date.

### The DashboardItem Struct

Each note is wrapped in a `DashboardItem` that carries its urgency classification and its position in the final list:

```go
type DashboardItem struct {
    Note  *notes.Note
    Group urgencyGroup
    Index int
}
```

The `Index` field is critical for cursor tracking. The TUI stores a single `cursor int`, and each item knows its own index so the rendering code can highlight the correct row.

### Building the Dashboard

`BuildDashboard` takes all notes and produces a sorted, indexed slice:

```go
func BuildDashboard(allNotes []*notes.Note, showDone bool, dateFormat string) []DashboardItem {
    now := time.Now()
    today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
    tomorrow := today.AddDate(0, 0, 1)

    var items []DashboardItem
    for _, n := range allNotes {
        if !showDone && n.Status == notes.StatusDone {
            continue
        }

        group := groupNoDue
        if !n.Due.IsZero() {
            if n.Due.Before(today) {
                group = groupOverdue
            } else if n.Due.Before(tomorrow) {
                group = groupToday
            } else {
                group = groupUpcoming
            }
        }

        items = append(items, DashboardItem{Note: n, Group: group})
    }

    sort.Slice(items, func(i, j int) bool {
        if items[i].Group != items[j].Group {
            return items[i].Group < items[j].Group
        }
        if items[i].Note.Due.IsZero() && items[j].Note.Due.IsZero() {
            return items[i].Note.Created.After(items[j].Note.Created)
        }
        if items[i].Note.Due.IsZero() {
            return false
        }
        if items[j].Note.Due.IsZero() {
            return true
        }
        return items[i].Note.Due.Before(items[j].Note.Due)
    })

    for i := range items {
        items[i].Index = i
    }

    return items
}
```

The classification logic uses two time boundaries: midnight of today and midnight of tomorrow. A due date before today is overdue, before tomorrow is "today", and anything else is upcoming. Notes with a zero due date land in `groupNoDue`.

The sort is a two-level comparison:
1. **Primary**: group order (overdue < today < upcoming < noDue).
2. **Secondary**: within the same group, sort by due date ascending. Notes without due dates are sorted by creation date descending (newest first) so the most recent notes float to the top.

After sorting, a final pass assigns sequential `Index` values so each item knows its position.

### Rendering the Dashboard

`RenderDashboard` walks the sorted items and builds a styled string:

```go
func RenderDashboard(items []DashboardItem, cursor int, width int, dateFormat string) string {
    if len(items) == 0 {
        return headerStyle.Render("No notes yet. Press [n] to create one.")
    }

    var sb strings.Builder
    currentGroup := urgencyGroup(-1)

    groupHeaders := map[urgencyGroup]string{
        groupOverdue:  "OVERDUE",
        groupToday:    "TODAY",
        groupUpcoming: "UPCOMING",
        groupNoDue:    "NO DUE DATE",
    }

    groupStyles := map[urgencyGroup]lipgloss.Style{
        groupOverdue:  overdueStyle,
        groupToday:    todayStyle,
        groupUpcoming: upcomingStyle,
        groupNoDue:    dimStyle,
    }

    for _, item := range items {
        if item.Group != currentGroup {
            currentGroup = item.Group
            style := groupStyles[currentGroup]
            sb.WriteString("\n " + style.Render(groupHeaders[currentGroup]) + "\n")
        }

        line := renderDashboardItem(item, dateFormat)

        if item.Index == cursor {
            sb.WriteString(selectedStyle.Width(width - 2).Render(line) + "\n")
        } else if item.Note.Status == notes.StatusDone {
            sb.WriteString(dimStyle.Render(line) + "\n")
        } else {
            sb.WriteString(normalStyle.Render(line) + "\n")
        }
    }

    return sb.String()
}
```

Key rendering behaviors:
- **Group headers** are inserted whenever the group changes. The `currentGroup` tracker (initialized to -1, an invalid group) ensures the first group always gets a header.
- **Cursor selection** uses `selectedStyle` stretched to the terminal width for a full-row highlight.
- **Done notes** are rendered with `dimStyle` so they visually recede without disappearing.

### Rendering Individual Items

Each dashboard row is assembled by `renderDashboardItem`:

```go
func renderDashboardItem(item DashboardItem, dateFormat string) string {
    n := item.Note
    typeLabel := typeStyle.Render("[" + string(n.Type) + "]")

    var priority string
    switch n.Priority {
    case notes.PriorityUrgent:
        priority = urgentStyle.Render(" [URGENT]")
    case notes.PriorityHigh:
        priority = highStyle.Render(" [HIGH]")
    }

    dueStr := ""
    if !n.Due.IsZero() {
        dueStr = " Due: " + dateutil.FormatDateOnly(n.Due, dateFormat)
    }

    return fmt.Sprintf("  %s %s%s  %s%s", typeLabel, n.Folder, priority, n.Title, dueStr)
}
```

A typical rendered line looks like: `[task] work [URGENT]  Fix production bug Due: 06-05-2026`. Only urgent and high priorities get a visible indicator -- normal and low priorities are unmarked to reduce visual noise.

### The Preview Pane

When the user selects a note, `RenderPreview` builds a rich Markdown preview:

```go
func RenderPreview(note *notes.Note, width int, dateFormat string) string {
    if note == nil {
        return dimStyle.Render("Select a note to preview")
    }

    var sb strings.Builder

    sb.WriteString(fmt.Sprintf("**Folder:** %s\n\n", note.Folder))
    sb.WriteString(fmt.Sprintf("**Type:** %s\n\n", note.Type))

    if !note.Created.IsZero() {
        sb.WriteString(fmt.Sprintf("**Created:** %s\n\n", dateutil.FormatDate(note.Created, dateFormat)))
    }
    if !note.Due.IsZero() {
        sb.WriteString(fmt.Sprintf("**Due:** %s\n\n", dateutil.FormatDate(note.Due, dateFormat)))
    }
    if note.Repeat != "" {
        sb.WriteString(fmt.Sprintf("**Repeat:** %s\n\n", note.Repeat))
    }
    if len(note.Tags) > 0 {
        sb.WriteString(fmt.Sprintf("**Tags:** %s\n\n", formatTags(note.Tags)))
    }
    // ... attendees, priority ...

    sb.WriteString("---\n\n")
    sb.WriteString(note.Body)

    r := getRenderer(width)
    if r == nil {
        return sb.String()
    }

    rendered, err := r.Render(sb.String())
    if err != nil {
        return sb.String()
    }

    return rendered
}
```

The function assembles metadata as Markdown bold labels, adds a horizontal rule, then appends the note body. The result is passed to Glamour for terminal rendering -- bold text becomes bold in your terminal, headings get styled, and code blocks get syntax highlighting.

### Cached Renderer Optimization

Creating a Glamour renderer is expensive. The preview pane redraws on every cursor movement, so a cached renderer avoids that overhead:

```go
var (
    cachedRenderer      *glamour.TermRenderer
    cachedRendererWidth int
)

func getRenderer(width int) *glamour.TermRenderer {
    if cachedRenderer != nil && cachedRendererWidth == width {
        return cachedRenderer
    }
    r, err := glamour.NewTermRenderer(glamour.WithWordWrap(width - 4))
    if err != nil {
        return nil
    }
    cachedRenderer = r
    cachedRendererWidth = width
    return r
}
```

The cache is keyed on terminal width. If the user resizes their terminal, a new renderer is created with the updated word-wrap setting. Otherwise, the same renderer is reused across all preview renders.

## Relationships

- **App** (`app.go`) calls `BuildDashboard` whenever the note list changes and `RenderDashboard`/`RenderPreview` on every `View()` cycle.
- **Notes package** provides the `Note` structs that the dashboard classifies and the preview renders.
- **DateUtil** formats due dates in the user's preferred format (EU or US style).
- **Lipgloss styles** (`styles.go`) provide the visual theme -- overdue in red, today in yellow, upcoming in green, etc.

## Key Takeaways

- **Separate data from rendering** -- `BuildDashboard` produces a sorted data structure, `RenderDashboard` turns it into a string. This makes both independently testable.
- **Use `iota` enums for ordered categories** -- the numeric ordering of urgency groups drives both sorting and display order.
- **Cache expensive resources** -- the Glamour renderer is only re-created when the terminal width changes.
- **Dim completed items instead of hiding them** -- the user can see what's done without it cluttering the active view.

## Next Steps

The dashboard shows existing notes, but how do you create new ones? In the next chapter, we'll build the multi-step **New Note Wizard** that guides users through choosing a type, folder, title, due date, and more.
