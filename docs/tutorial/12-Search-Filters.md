---
title: "Chapter 12: Search & Filters"
order: 12
---

# Chapter 12: Search & Filters

As your note collection grows from ten to a hundred to a thousand entries, scrolling through the dashboard becomes impractical. You need two tools: a **search bar** for when you know what you're looking for ("that meeting about the Q3 budget"), and **filters** for when you want to narrow by category ("show me only tasks" or "show me everything tagged #urgent").

Think of Search as a librarian who reads every book title for you: you whisper a phrase, and they pull matching volumes off the shelf. Filters, on the other hand, are like sorting books by genre -- you're not looking for a specific title, you're browsing a subset of the collection.

## How It Works

pq-notes provides three sub-components:

1. **SearchModel** -- a text input overlay that queries the note store in real time, displaying results as you type.
2. **FilterModel** -- a checkbox list overlay used for both tag filtering and type filtering.
3. **`applyFilters`** -- a function in the App that combines active tag and type filters using AND/OR logic.

All three are "sub-models" rather than full Bubble Tea models. Their `Update` methods take a `tea.KeyPressMsg` and return control signals (done/selected) rather than a `tea.Model`. The parent App decides when to show them, routes key events to them, and acts on their results.

## Code Deep Dive

### SearchModel

The search overlay holds a query string, a results slice, a cursor for navigation, and a reference to the note store:

```go
type SearchModel struct {
    query   string
    results []*notes.Note
    cursor  int
    store   *notes.NoteStore
}

func NewSearchModel(store *notes.NoteStore) *SearchModel {
    return &SearchModel{
        store: store,
    }
}
```

### Keystroke-Driven Update

The search Update method maps each key to a specific action:

```go
func (s *SearchModel) Update(msg tea.KeyPressMsg) (done bool, selected *notes.Note) {
    key := msg.Key()

    switch key.Code {
    case tea.KeyEscape:
        return true, nil
    case tea.KeyEnter:
        if len(s.results) > 0 && s.cursor < len(s.results) {
            return true, s.results[s.cursor]
        }
        return true, nil
    case tea.KeyUp:
        if s.cursor > 0 {
            s.cursor--
        }
    case tea.KeyDown:
        if s.cursor < len(s.results)-1 {
            s.cursor++
        }
    case tea.KeyBackspace:
        if len(s.query) > 0 {
            s.query = removeLastRune(s.query)
            s.search()
        }
    default:
        if key.Text != "" {
            s.query += key.Text
            s.search()
        }
    }

    return false, nil
}
```

The return signature `(done bool, selected *notes.Note)` tells the parent App two things: whether to close the overlay, and which note (if any) the user picked. Escape closes without a selection; Enter closes with the highlighted result (or nil if the list is empty).

Every character typed or deleted triggers a re-search. Arrow keys move the cursor without re-searching.

### The search() Method

The internal `search` method delegates to the store's `Search` function:

```go
func (s *SearchModel) search() {
    if strings.TrimSpace(s.query) == "" {
        s.results = nil
        s.cursor = 0
        return
    }
    results, err := s.store.Search(s.query)
    if err != nil {
        s.results = nil
        s.cursor = 0
        return
    }
    s.results = results
    if s.cursor >= len(s.results) {
        if len(s.results) > 0 {
            s.cursor = len(s.results) - 1
        } else {
            s.cursor = 0
        }
    }
}
```

A key detail: when results shrink (e.g., the user adds a letter that narrows the match), the cursor is clamped to the last valid position so it never points past the end of the list.

### Search View

The view renders an input field and the results list:

```go
func (s *SearchModel) View() string {
    var sb strings.Builder

    sb.WriteString("\n")
    sb.WriteString(titleStyle.Render("Search Notes"))
    sb.WriteString("\n\n")

    inputStyle := lipgloss.NewStyle().
        Foreground(lipgloss.Color("#FFFFFF")).
        Background(lipgloss.Color("#333333")).
        Padding(0, 1).
        Width(50)

    sb.WriteString(normalStyle.Render("  Query: "))
    sb.WriteString(inputStyle.Render(s.query + "_"))
    sb.WriteString("\n\n")

    if len(s.results) == 0 {
        if strings.TrimSpace(s.query) != "" {
            sb.WriteString(dimStyle.Render("  No results found."))
        } else {
            sb.WriteString(dimStyle.Render("  Type to search..."))
        }
        sb.WriteString("\n")
    } else {
        sb.WriteString(dimStyle.Render(fmt.Sprintf("  %d result(s)", len(s.results))))
        sb.WriteString("\n\n")

        for i, n := range s.results {
            label := fmt.Sprintf("  %s %s - %s",
                typeStyle.Render("["+string(n.Type)+"]"),
                n.Folder,
                n.Title,
            )
            if len(n.Tags) > 0 {
                label += "  " + tagStyle.Render("["+strings.Join(n.Tags, ", ")+"]")
            }

            if i == s.cursor {
                sb.WriteString(selectedStyle.Render(label))
            } else {
                sb.WriteString(normalStyle.Render(label))
            }
            sb.WriteString("\n")
        }
    }

    sb.WriteString("\n")
    sb.WriteString(helpStyle.Render("[Up/Down] navigate  [Enter] select  [Esc] cancel"))
    sb.WriteString("\n")

    return sb.String()
}
```

The trailing `_` on the query input creates a simple text cursor effect. Each result shows the note type badge, folder, title, and tags.

### FilterModel

The filter component is a reusable checkbox list. It works for both tag filtering and type filtering:

```go
type FilterModel struct {
    items    []string
    selected map[string]bool
    cursor   int
    title    string
}
```

The `selected` map tracks which items are checked. Using a map instead of a slice makes toggle operations O(1).

### Creating Filters

Two constructor functions populate filters differently:

```go
func NewTagFilter(allNotes []*notes.Note) *FilterModel {
    tagSet := make(map[string]bool)
    for _, n := range allNotes {
        for _, t := range n.Tags {
            tagSet[t] = true
        }
    }

    var tags []string
    for t := range tagSet {
        tags = append(tags, t)
    }
    sort.Strings(tags)

    return &FilterModel{
        items:    tags,
        selected: make(map[string]bool),
        title:    "Filter by Tag",
    }
}

func NewTypeFilter() *FilterModel {
    return &FilterModel{
        items:    []string{"meeting", "task", "reminder", "followup"},
        selected: make(map[string]bool),
        title:    "Filter by Type",
    }
}
```

`NewTagFilter` dynamically discovers all unique tags across all notes. `NewTypeFilter` uses the fixed set of four note types. Both produce a `FilterModel` with the same interface.

### Space-Toggle Update

The filter's Update method uses the space bar to toggle checkboxes:

```go
func (f *FilterModel) Update(msg tea.KeyPressMsg) (done bool) {
    key := msg.Key()

    switch key.Code {
    case tea.KeyEscape:
        f.selected = make(map[string]bool)
        return true
    case tea.KeyEnter:
        return true
    case tea.KeyUp:
        if f.cursor > 0 {
            f.cursor--
        }
    case tea.KeyDown:
        if f.cursor < len(f.items)-1 {
            f.cursor++
        }
    case ' ':
        if len(f.items) > 0 {
            item := f.items[f.cursor]
            if f.selected[item] {
                delete(f.selected, item)
            } else {
                f.selected[item] = true
            }
        }
    }

    return false
}
```

An important UX detail: pressing Escape clears all selections and closes the overlay (cancel semantics), while Enter preserves selections and closes (apply semantics).

### Getting Selected Items

```go
func (f *FilterModel) SelectedItems() []string {
    var result []string
    for _, item := range f.items {
        if f.selected[item] {
            result = append(result, item)
        }
    }
    return result
}
```

This iterates over `items` (not `selected`) to preserve the original sort order of the returned slice.

### Checkbox Rendering

The View renders each item with a `[x]` or `[ ]` prefix:

```go
func (f *FilterModel) View() string {
    var sb strings.Builder

    sb.WriteString("\n")
    sb.WriteString(titleStyle.Render(f.title))
    sb.WriteString("\n\n")

    if len(f.items) == 0 {
        sb.WriteString(dimStyle.Render("  No items available."))
        sb.WriteString("\n")
    } else {
        for i, item := range f.items {
            checkbox := "[ ]"
            if f.selected[item] {
                checkbox = "[x]"
            }

            label := fmt.Sprintf("  %s %s", checkbox, item)

            if i == f.cursor {
                sb.WriteString(selectedStyle.Render(label))
            } else {
                sb.WriteString(normalStyle.Render(label))
            }
            sb.WriteString("\n")
        }
    }

    sb.WriteString("\n")
    sb.WriteString(helpStyle.Render("[Up/Down] navigate  [Space] toggle  [Enter] apply  [Esc] cancel"))
    sb.WriteString("\n")

    return sb.String()
}
```

### How the App Integrates Filters

Back in `app.go`, the `applyFilters` function combines active type and tag filters:

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
```

The filter logic uses **AND between filter types, OR within a filter type**:
- A note must match at least one of the selected types **AND** at least one of the selected tags.
- If no types are selected, the type filter is skipped (all types pass). Same for tags.
- Within tags, `hasAnyTag` returns true if the note has **any** of the selected tags (OR logic).

This means filtering by types `[task, meeting]` and tags `[urgent, review]` shows all tasks and meetings that have either the `urgent` or `review` tag.

## Relationships

- **App** creates `SearchModel` and `FilterModel` instances, routes key events to them while active, and applies the results to the dashboard view.
- **NoteStore** provides the `Search()` method that the SearchModel calls.
- **Dashboard** receives the filtered note list and builds its urgency groups from the reduced set.

## Key Takeaways

- **Sub-models simplify composition** -- `SearchModel` and `FilterModel` are lightweight components with clear interfaces (`Update` returns done + result), making them easy to embed in the parent App.
- **AND between categories, OR within** -- this is the most intuitive filter behavior for users: selecting multiple tags means "any of these," selecting types means "restrict to these."
- **Escape = cancel, Enter = apply** -- consistent dismiss semantics across overlays reduces user confusion.
- **Dynamic vs. static filter items** -- tags are discovered from the data, types are hardcoded. Both use the same `FilterModel` component.
- **Cursor clamping** prevents out-of-bounds errors when search results shrink.

## Next Steps

Your notes are created, displayed, searched, and filtered -- all locally. In the next chapter, we'll take them to the cloud with **Google Drive Sync**, keeping your encrypted `.age` files backed up and accessible across machines.
