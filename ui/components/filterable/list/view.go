package filterlist

import (
	"strings"
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
