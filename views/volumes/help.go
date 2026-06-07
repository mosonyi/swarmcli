// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package volumesview

import helpview "swarmcli/views/help"

// GetVolumesHelpContent returns categorized help for the volumes view.
func GetVolumesHelpContent() []helpview.HelpCategory {
	return []helpview.HelpCategory{
		{
			Title: "General",
			Items: []helpview.HelpItem{
				{Keys: "<i/enter>", Description: "Inspect selected volume"},
				{Keys: "</>", Description: "Filter volumes"},
				{Keys: "<?>", Description: "Open this help"},
			},
		},
		{
			Title: "Sorting",
			Items: []helpview.HelpItem{
				{Keys: "<shift+n>", Description: "Order by Name"},
				{Keys: "<shift+s>", Description: "Order by Stack"},
				{Keys: "<shift+d>", Description: "Order by Driver"},
				{Keys: "<shift+c>", Description: "Order by Created"},
				{Keys: "<shift+h>", Description: "Order by Host"},
				{Keys: "(repeat key)", Description: "Toggle ascending/descending"},
			},
		},
		{
			Title: "Navigation",
			Items: []helpview.HelpItem{
				{Keys: "<↑/↓>", Description: "Move cursor"},
				{Keys: "<pgup>", Description: "Page up"},
				{Keys: "<pgdown>", Description: "Page down"},
				{Keys: "<esc/q>", Description: "Back"},
			},
		},
	}
}
