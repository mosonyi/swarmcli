package filterlist

import tea "github.com/charmbracelet/bubbletea"

func (f *FilterableList[T]) HandleKey(msg tea.KeyMsg) {
	// --- Searching mode ---
	if f.Mode == ModeSearching {
		switch msg.Type {
		case tea.KeyRunes:
			f.Query += string(msg.Runes)
			f.ApplyFilter(nil) // pass your own matchFunc if needed
		case tea.KeyBackspace:
			if len(f.Query) > 0 {
				f.Query = f.Query[:len(f.Query)-1]
			}
			f.ApplyFilter(nil)
		case tea.KeyEsc:
			f.Mode = ModeNormal
			f.Query = ""
			f.Filtered = f.Items
			f.Cursor = 0
			f.Viewport.GotoTop()
		}
		return
	}

	// --- Normal mode ---
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
		page := f.Viewport.Height
		if f.Cursor > page {
			f.Cursor -= page
		} else {
			f.Cursor = 0
		}
		f.ensureCursorVisible()
	case "pgdown", "d":
		page := f.Viewport.Height
		if f.Cursor+page < len(f.Filtered) {
			f.Cursor += page
		} else {
			if len(f.Filtered) > 0 {
				f.Cursor = len(f.Filtered) - 1
			} else {
				f.Cursor = 0
			}
		}
		f.ensureCursorVisible()
	case "/":
		f.Mode = ModeSearching
		f.Query = ""
		f.Cursor = 0
		f.Filtered = f.Items
		f.Viewport.GotoTop()
	}
}
