package filterlist

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (l *FilterableList[T]) View() string {
	var lines []string
	for i, item := range l.Filtered {
		line := l.RenderItem(item, i == l.Cursor)
		lines = append(lines, line)
	}

	content := strings.Join(lines, "\n")
	l.Viewport.SetContent(content) // update viewport content
	return l.Viewport.View()       // returns clipped viewport
}

func (l *FilterableList[T]) ensureCursorVisible() {
	h := l.Viewport.Height
	if h < 1 {
		h = 1
	}

	if l.Cursor < l.Viewport.YOffset {
		l.Viewport.YOffset = l.Cursor
	} else if l.Cursor >= l.Viewport.YOffset+h {
		l.Viewport.YOffset = l.Cursor - h + 1
	}
}

func (l *FilterableList[T]) ComputeColWidth(render func(item T) string, minWidth int) int {
	if len(l.Items) == 0 {
		return minWidth
	}

	// Find maximum width of all items (not just filtered)
	maxName := minWidth
	for _, item := range l.Items {
		w := lipgloss.Width(render(item))
		if w > maxName {
			maxName = w
		}
	}

	// Ensure it fits within viewport
	available := l.Viewport.Width - 2 // leave room for padding/borders
	if available < minWidth {
		return minWidth
	}
	if maxName > available {
		return available
	}
	return maxName
}
