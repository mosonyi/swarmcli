package filterlist

import (
	"fmt"
	"strings"
)

func (f *FilterableList[T]) View() string {
	if len(f.Filtered) == 0 {
		if f.Mode == ModeSearching && f.Query != "" {
			return fmt.Sprintf("No items match: %q", f.Query)
		}
		return "No items found."
	}

	var lines []string
	for i, item := range f.Filtered {
		lines = append(lines, f.RenderItem(item, i == f.Cursor))
	}
	return strings.Join(lines, "\n")
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
