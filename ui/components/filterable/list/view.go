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
