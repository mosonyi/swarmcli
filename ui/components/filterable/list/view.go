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

	content := strings.Join(lines, "\n")
	// Only update viewport content here
	f.Viewport.SetContent(content)
	return f.Viewport.View()
}

// ensureCursorVisible keeps the cursor in the visible viewport range
func (f *FilterableList[T]) ensureCursorVisible() {
	h := f.Viewport.Height
	if h < 1 {
		h = 1
	}
	if f.Cursor < f.Viewport.YOffset {
		f.Viewport.YOffset = f.Cursor
	} else if f.Cursor >= f.Viewport.YOffset+h {
		f.Viewport.YOffset = f.Cursor - h + 1
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
