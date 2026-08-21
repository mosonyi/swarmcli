// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package charts

import (
	"github.com/Eldara-Tech/swarmcli/v2/args"
	"github.com/Eldara-Tech/swarmcli/v2/registry"
	chartsview "github.com/Eldara-Tech/swarmcli/v2/views/charts"
	"github.com/Eldara-Tech/swarmcli/v2/views/view"

	tea "github.com/charmbracelet/bubbletea"
)

type ChartsLs struct{}

func (ChartsLs) Name() string        { return "charts" }
func (ChartsLs) Description() string { return "browse chart releases" }

func (ChartsLs) Spec() registry.CommandSpec {
	return registry.CommandSpec{
		Detail: "Opens the chart releases browser: every installed release " +
			"with its revision, recorded status and live rollout health. " +
			"The view is read-only — installing, upgrading, rolling back and " +
			"uninstalling run from the command line, as `swarmcli charts " +
			"<command>`.",
		Examples: []string{":charts"},
	}
}

func (ChartsLs) Execute(ctx any, args args.Args) tea.Cmd {
	return func() tea.Msg {
		return view.NavigateToMsg{
			ViewName: chartsview.ViewName,
			Payload:  nil,
		}
	}
}

// chartAlias is the ":chart" alias for ":charts". It inherits Description,
// Execute and Spec from the embedded primary; AliasOf folds it under "charts"
// in the command list and help.
type chartAlias struct{ ChartsLs }

func (chartAlias) Name() string    { return "chart" }
func (chartAlias) AliasOf() string { return "charts" }

func init() {
	registry.Register(ChartsLs{})
	registry.Register(chartAlias{})
}
