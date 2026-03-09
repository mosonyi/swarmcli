// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package searchinput

// SearchQueryMsg is emitted on every keystroke so the active view can
// apply the filter incrementally.
type SearchQueryMsg struct{ Query string }

// SearchClearedMsg is emitted when the user cancels the search (Esc or
// backspace on an empty query). The active view should clear any filter.
type SearchClearedMsg struct{}
