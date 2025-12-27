package filterlist

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
)

type FilterableList[T any] struct {
	Viewport viewport.Model

	Items    []T
	Filtered []T
	Cursor   int
	Query    string
	Mode     ModeType

	// Function to render a single item (pass in computed colWidth)
	RenderItem func(item T, selected bool, colWidth int) string

	// Match function for filtering
	Match func(item T, query string) bool

	colWidth int
}

type ModeType int

const (
	ModeNormal ModeType = iota
	ModeSearching
)

// VisibleContent returns a string containing exactly `lines` rows of
// rendered items starting at the current viewport Y offset. It will adjust
// the internal Y offset to ensure the cursor is visible within the given
// number of lines. This avoids mutating the viewport's Height during
// rendering and prevents jitter when parent views trim content.
func (f *FilterableList[T]) VisibleContent(lines int) string {
	if lines < 1 {
		lines = 1
	}

	// Ensure cursor visible within the provided height
	if f.Cursor < f.Viewport.YOffset {
		f.Viewport.YOffset = f.Cursor
	} else if f.Cursor >= f.Viewport.YOffset+lines {
		f.Viewport.YOffset = f.Cursor - lines + 1
	}

	// If no items or filtered empty, return appropriate placeholder
	if len(f.Filtered) == 0 {
		if f.Items == nil {
			return f.Viewport.View()
		}
		var msg string
		if f.Mode == ModeSearching && f.Query != "" {
			msg = "No items match: " + f.Query
		} else {
			msg = "No items found."
		}
		// Pad to requested lines
		parts := make([]string, lines)
		parts[0] = msg
		for i := 1; i < lines; i++ {
			parts[i] = ""
		}
		return strings.Join(parts, "\n")
	}

	out := make([]string, lines)
	for i := 0; i < lines; i++ {
		idx := f.Viewport.YOffset + i
		if idx < 0 || idx >= len(f.Filtered) {
			out[i] = ""
			continue
		}
		if f.RenderItem != nil {
			out[i] = f.RenderItem(f.Filtered[idx], idx == f.Cursor, f.colWidth)
		} else {
			out[i] = ""
		}
	}
	return strings.Join(out, "\n")
}
