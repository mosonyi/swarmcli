package filterlist

import (
	"fmt"
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

	// When true, VisibleContent won't adjust YOffset (for manual navigation like task scrolling)
	SkipOffsetAdjustment bool

	colWidth int
}

type ModeType int

const (
	ModeNormal ModeType = iota
	ModeSearching
)

// countLines returns the number of lines in a string (1 + number of newlines)
func countLines(s string) int {
	if s == "" {
		return 1
	}
	return strings.Count(s, "\n") + 1
}

// VisibleContent returns a string containing exactly `lines` rows of
// rendered items starting at the current viewport Y offset. It will adjust
// the internal Y offset to ensure the cursor is visible within the given
// number of lines. This avoids mutating the viewport's Height during
// rendering and prevents jitter when parent views trim content.
func (f *FilterableList[T]) VisibleContent(lines int) string {
	if lines < 1 {
		lines = 1
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

	// Pre-render all items to know their line counts
	renderedItems := make([]string, len(f.Filtered))
	itemLineCounts := make([]int, len(f.Filtered))
	totalLines := 0
	for i := range f.Filtered {
		if f.RenderItem != nil {
			renderedItems[i] = f.RenderItem(f.Filtered[i], i == f.Cursor, f.colWidth)
		} else {
			renderedItems[i] = fmt.Sprintf("%v", f.Filtered[i])
		}
		itemLineCounts[i] = countLines(renderedItems[i])
		totalLines += itemLineCounts[i]
	}

	// Find which item contains the cursor and ensure it's visible
	// Calculate line offset for cursor item
	cursorLineStart := 0
	for i := 0; i < f.Cursor && i < len(itemLineCounts); i++ {
		cursorLineStart += itemLineCounts[i]
	}
	cursorLineEnd := cursorLineStart + itemLineCounts[f.Cursor] - 1

	// Adjust viewport offset to keep cursor visible (unless manually managed)
	if !f.SkipOffsetAdjustment {
		// If cursor item is taller than viewport, prioritize showing the start
		if cursorLineStart < f.Viewport.YOffset {
			// Cursor item starts above viewport - scroll up to show it
			f.Viewport.YOffset = cursorLineStart
		} else if itemLineCounts[f.Cursor] > lines {
			// Cursor item is taller than viewport - show from its start to prevent oscillation
			if f.Viewport.YOffset > cursorLineStart {
				f.Viewport.YOffset = cursorLineStart
			}
		} else if cursorLineEnd >= f.Viewport.YOffset+lines {
			// Cursor item ends below viewport - scroll down to show it
			f.Viewport.YOffset = cursorLineEnd - lines + 1
			if f.Viewport.YOffset < 0 {
				f.Viewport.YOffset = 0
			}
		}
	}

	// Collect lines for display, starting from YOffset
	var result []string
	currentLine := 0
	for i := 0; i < len(renderedItems); i++ {
		itemLines := strings.Split(renderedItems[i], "\n")
		for _, line := range itemLines {
			if currentLine >= f.Viewport.YOffset && len(result) < lines {
				result = append(result, line)
			}
			currentLine++
			if len(result) >= lines {
				break
			}
		}
		if len(result) >= lines {
			break
		}
	}

	// Pad with empty lines if needed
	for len(result) < lines {
		result = append(result, "")
	}

	return strings.Join(result, "\n")
}
