package filterlist

func (l *FilterableList[T]) ApplyFilter() {
	if l.Query == "" {
		l.Filtered = l.Items
	} else {
		var result []T
		for _, item := range l.Items {
			if l.Match(item, l.Query) {
				result = append(result, item)
			}
		}
		l.Filtered = result
	}

	// Ensure cursor stays in bounds
	if l.Cursor >= len(l.Filtered) {
		l.Cursor = len(l.Filtered) - 1
	}
	if l.Cursor < 0 {
		l.Cursor = 0
	}

	l.ensureCursorVisible()
}
