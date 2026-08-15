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
			// The set is responsive, so an operator on a narrow terminal does
			// not go looking for a column that is not there.
			Title: "Columns",
			Items: []helpview.HelpItem{
				{Keys: "STATUS", Description: "The recorded state of the release: what we wrote when we deployed"},
				{Keys: "HEALTH", Description: "What the swarm is doing now: converged, progressing or wedged"},
				{Keys: "DETAIL", Description: "Why it is not converged, or what it runs. Shown when the terminal has room"},
				{Keys: "OWNER", Description: "The `charts apply` manifest that installed it. Shown on a wide terminal"},
			},
		},
		{
			Title: "The LATEST column",
			Items: []helpview.HelpItem{
				{Keys: "0.2.0", Description: "A newer chart version is published in a cached repository index"},
				{Keys: "—", Description: "Nothing newer, or the chart is in no index (a local chart)"},
				{Keys: "?", Description: "No cached index to compare against — run `swarmcli charts repo update`"},
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
			// This view never changes a release. The commands that do are
			// listed here rather than left for the operator to remember,
			// because "the CLI does it" is not an answer without the verb.
			Title: "Chart operations (CLI only)",
			Items: []helpview.HelpItem{
				{Keys: "<u>", Description: "Show the `charts upgrade` command for this release"},
				{Keys: "<r>", Description: "Show the `charts rollback` command for this release"},
				{Keys: "<ctrl+d>", Description: "Show the `charts uninstall` command for this release"},
				{Keys: "install", Description: "swarmcli charts install <release> <repo/chart>"},
				{Keys: "apply", Description: "swarmcli charts apply -f swarmcli-release.yaml"},
				{Keys: "repo update", Description: "swarmcli charts repo update  (populates the LATEST column)"},
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
