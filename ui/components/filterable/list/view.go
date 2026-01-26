// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package filterlist

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (f *FilterableList[T]) View() string {
	if len(f.Filtered) == 0 {
		// If the underlying Items slice is still nil, the list hasn't been
		// initialized yet (e.g. still loading). In that case keep any
		// placeholder content set on the viewport by the parent view so it
		// can control sizing (e.g. a loading line).
		if f.Items == nil {
			return f.Viewport.View()
		}

		// For an empty-but-initialized list, render a message into the
		// viewport instead of returning a raw string. This ensures parent
		// views that pad/trim the viewport content (to occupy full height)
		// will receive the message as viewport content and can size
		// correctly.
		var msg string
		if f.Mode == ModeSearching && f.Query != "" {
			msg = fmt.Sprintf("No items match: %q", f.Query)
		} else {
			msg = "No items found."
		}
		f.Viewport.SetContent(msg)
		return f.Viewport.View()
	}

	lines := make([]string, len(f.Filtered))
	for i, item := range f.Filtered {
		lines[i] = f.RenderItem(item, i == f.Cursor, f.colWidth)
	}

	// Ensure cursor is visible before setting content
	f.ensureCursorVisible()

	content := strings.Join(lines, "\n")
	// Only update viewport content here
	f.Viewport.SetContent(content)
	return f.Viewport.View()
}

// ensureCursorVisible keeps the cursor in the visible viewport range
// This now accounts for multi-line items
func (f *FilterableList[T]) ensureCursorVisible() {
	h := f.Viewport.Height
	if h < 1 {
		h = 1
	}

	// Count lines for each item to find cursor position in lines
	renderedItems := make([]string, len(f.Filtered))
	itemLineCounts := make([]int, len(f.Filtered))
	for i := range f.Filtered {
		if f.RenderItem != nil {
			renderedItems[i] = f.RenderItem(f.Filtered[i], i == f.Cursor, f.colWidth)
		} else {
			renderedItems[i] = fmt.Sprintf("%v", f.Filtered[i])
		}
		if renderedItems[i] == "" {
			itemLineCounts[i] = 1
		} else {
			itemLineCounts[i] = strings.Count(renderedItems[i], "\n") + 1
		}
	}

	// Calculate line offset for cursor item
	cursorLineStart := 0
	for i := 0; i < f.Cursor && i < len(itemLineCounts); i++ {
		cursorLineStart += itemLineCounts[i]
	}
	cursorLineEnd := cursorLineStart
	if f.Cursor < len(itemLineCounts) {
		cursorLineEnd = cursorLineStart + itemLineCounts[f.Cursor] - 1
	}

	// Adjust viewport offset to keep cursor visible
	if cursorLineStart < f.Viewport.YOffset {
		f.Viewport.YOffset = cursorLineStart
	} else if cursorLineEnd >= f.Viewport.YOffset+h {
		f.Viewport.YOffset = cursorLineEnd - h + 1
		if f.Viewport.YOffset < 0 {
			f.Viewport.YOffset = 0
		}
	}
}

func (f *FilterableList[T]) ComputeAndSetColWidth(renderName func(item T) string, minWidth int) {
	if len(f.Items) == 0 {
		f.colWidth = minWidth
		return
	}

	maxName := minWidth
	for _, item := range f.Items {
		if w := lipgloss.Width(renderName(item)); w > maxName {
			maxName = w
		}
	}

	available := f.Viewport.Width - 2
	switch {
	case available < minWidth:
		f.colWidth = minWidth
	case maxName > available:
		f.colWidth = available
	default:
		f.colWidth = maxName
	}
}

// GetColWidth returns the computed column width
func (f *FilterableList[T]) GetColWidth() int {
	return f.colWidth
}
