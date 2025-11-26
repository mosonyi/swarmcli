package filterlist

func (f *FilterableList[T]) ApplyFilter(matchFunc func(T, string) bool) {
	if f.Query == "" {
		f.Filtered = f.Items
		f.Cursor = 0
		f.Viewport.GotoTop()
		return
	}

	var result []T
	for _, item := range f.Items {
		if matchFunc(item, f.Query) {
			result = append(result, item)
		}
	}
	f.Filtered = result

	if len(f.Filtered) == 0 {
		f.Cursor = 0
		f.Viewport.GotoTop()
		return
	}

	if f.Cursor >= len(f.Filtered) {
		f.Cursor = len(f.Filtered) - 1
	}
	if f.Cursor < 0 {
		f.Cursor = 0
	}
}
