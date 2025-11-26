package filterlist

import tea "github.com/charmbracelet/bubbletea"

// HandleKey updates cursor and filter state, automatically keeping the cursor visible
func (f *FilterableList[T]) HandleKey(msg tea.KeyMsg) {
	// Searching mode
	if f.Mode == ModeSearching {
		switch msg.Type {
		case tea.KeyRunes:
			f.Query += string(msg.Runes)
			f.ApplyFilter()
		case tea.KeyBackspace:
			if len(f.Query) > 0 {
				f.Query = f.Query[:len(f.Query)-1]
			} else if len(f.Query) == 0 {
				f.Mode = ModeNormal
				f.Query = ""
				f.ApplyFilter()
				f.Cursor = 0
				f.Viewport.GotoTop()
			}
			f.ApplyFilter()
		case tea.KeyEsc:
			f.Mode = ModeNormal
			f.Query = ""
			f.ApplyFilter()
			f.Cursor = 0
			f.Viewport.GotoTop()
		}
	}

	// Normal mode
	switch msg.String() {
	case "up", "k":
		if f.Cursor > 0 {
			f.Cursor--
			f.ensureCursorVisible()
		}
	case "down", "j":
		if f.Cursor < len(f.Filtered)-1 {
			f.Cursor++
			f.ensureCursorVisible()
		}
	case "pgup", "u":
		h := f.Viewport.Height
		if f.Cursor > h {
			f.Cursor -= h
		} else {
			f.Cursor = 0
		}
		f.ensureCursorVisible()
	case "pgdown", "d":
		h := f.Viewport.Height
		if f.Cursor+h < len(f.Filtered) {
			f.Cursor += h
		} else if len(f.Filtered) > 0 {
			f.Cursor = len(f.Filtered) - 1
		}
		f.ensureCursorVisible()
	case "/":
		f.Mode = ModeSearching
		f.Query = ""
		f.Cursor = 0
		f.ApplyFilter()
		f.Viewport.GotoTop()
	}
}
