package tui

import (
	"fmt"
	"strings"

	"charm.land/glamour/v2"
	"github.com/kborup-redhat/pq-notes/internal/dateutil"
	"github.com/kborup-redhat/pq-notes/internal/notes"
)

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

// RenderPreview renders a note as a Glamour-styled markdown preview.
// Returns a dimmed placeholder when note is nil.
func RenderPreview(note *notes.Note, width int, dateFormat string) string {
	if note == nil {
		return dimStyle.Render("Select a note to preview")
	}

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("**Customer:** %s\n\n", note.Customer))
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
	if note.Priority != "" && note.Priority != notes.PriorityNormal {
		sb.WriteString(fmt.Sprintf("**Priority:** %s\n\n", note.Priority))
	}
	if len(note.Attendees) > 0 {
		sb.WriteString(fmt.Sprintf("**Attendees:** %s\n\n", strings.Join(note.Attendees, ", ")))
	}

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

func formatTags(tags []string) string {
	formatted := make([]string, len(tags))
	for i, t := range tags {
		formatted[i] = "#" + t
	}
	return strings.Join(formatted, " ")
}
