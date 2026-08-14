// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package chartsview

import (
	helpview "github.com/Eldara-Tech/swarmcli/views/help"
)

// GetChartsHelpContent returns categorized help for the charts view.
func GetChartsHelpContent() []helpview.HelpCategory {
	return []helpview.HelpCategory{
		{
			Title: "General",
			Items: []helpview.HelpItem{
				{Keys: "<i>", Description: "Show the rendered manifest"},
				{Keys: "<v>", Description: "Show the stored values"},
				{Keys: "<s>", Description: "Show the release's services"},
				{Keys: "</>", Description: "Filter releases"},
				{Keys: "<?>", Description: "Open this help"},
			},
		},
		{
			Title: "Sorting",
			Items: []helpview.HelpItem{
				{Keys: "<shift+n>", Description: "Order by Name"},
				{Keys: "<shift+r>", Description: "Order by Revision"},
				{Keys: "<shift+s>", Description: "Order by Status"},
				{Keys: "<shift+h>", Description: "Order by Health (worst first)"},
				{Keys: "<shift+c>", Description: "Order by Chart"},
				{Keys: "<shift+u>", Description: "Order by Updated"},
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
