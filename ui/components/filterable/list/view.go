package filterlist

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (l *FilterableList[T]) View() string {
	if len(l.Filtered) == 0 {
		if l.Mode == ModeSearching && l.Query != "" {
			return fmt.Sprintf("No items match: %q", l.Query)
		}
		return "No items found."
	}

	var lines []string
	for i, item := range l.Filtered {
		lines = append(lines, l.RenderItem(item, i == l.Cursor, l.colWidth))
	}

	content := strings.Join(lines, "\n")
	l.Viewport.SetContent(content) // update viewport content
	return l.Viewport.View()
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

func (l *FilterableList[T]) ComputeAndSetColWidth(renderName func(item T) string, minWidth int) {
	if len(l.Items) == 0 {
		l.colWidth = minWidth
		return
	}

	maxName := minWidth
	for _, item := range l.Items {
		if w := lipgloss.Width(renderName(item)); w > maxName {
			maxName = w
		}
	}

	available := l.Viewport.Width - 2 // leave room for borders
	if available < minWidth {
		l.colWidth = minWidth
	} else if maxName > available {
		l.colWidth = available
	} else {
		l.colWidth = maxName
	}
}
