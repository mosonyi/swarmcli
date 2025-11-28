package filterlist

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (f *FilterableList[T]) View() string {
	if len(f.Filtered) == 0 {
		if f.Mode == ModeSearching && f.Query != "" {
			return fmt.Sprintf("No items match: %q", f.Query)
		}
		return "No items found."
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
