package networksview

import helpview "swarmcli/views/help"

// GetNetworksHelpContent returns categorized help for the networks view.
func GetNetworksHelpContent() []helpview.HelpCategory {
	return []helpview.HelpCategory{
		{
			Title: "General",
			Items: []helpview.HelpItem{
				{Keys: "<i>", Description: "Inspect selected network (JSON)"},
				{Keys: "<u>", Description: "Show services using the network"},
				{Keys: "</>", Description: "Filter networks"},
				{Keys: "<?>", Description: "Open this help"},
			},
		},
		{
			Title: "Sorting",
			Items: []helpview.HelpItem{
				{Keys: "<shift+n>", Description: "Order by Name"},
				{Keys: "<shift+d>", Description: "Order by Driver"},
				{Keys: "<shift+s>", Description: "Order by Scope"},
				{Keys: "<shift+u>", Description: "Order by Used"},
				{Keys: "<shift+i>", Description: "Order by ID"},
				{Keys: "<shift+c>", Description: "Order by Created"},
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
		{
			Title: "Danger Zone",
			Items: []helpview.HelpItem{
				{Keys: "<ctrl+d>", Description: "Delete selected network"},
				{Keys: "<ctrl+u>", Description: "Prune unused networks"},
			},
		},
	}
}
