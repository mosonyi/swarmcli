// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package filterlist

func (f *FilterableList[T]) ApplyFilter() {
	if f.Query == "" {
		f.Filtered = f.Items
	} else {
		var result []T
		for _, item := range f.Items {
			if f.Match(item, f.Query) {
				result = append(result, item)
			}
		}
		f.Filtered = result
	}

	// Keep cursor in bounds
	if f.Cursor >= len(f.Filtered) {
		f.Cursor = len(f.Filtered) - 1
	}
	if f.Cursor < 0 {
		f.Cursor = 0
	}

	f.ensureCursorVisible()
}
