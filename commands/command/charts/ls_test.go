// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package charts

import (
	"testing"

	"github.com/Eldara-Tech/swarmcli/args"
	"github.com/Eldara-Tech/swarmcli/registry"
	chartsview "github.com/Eldara-Tech/swarmcli/views/charts"
	"github.com/Eldara-Tech/swarmcli/views/view"

	"github.com/stretchr/testify/require"
)

func TestChartsCommandNavigatesToTheView(t *testing.T) {
	cmd, ok := registry.Get("charts")
	require.True(t, ok, ":charts must be registered")

	msg := cmd.Execute(nil, args.Args{})()
	nav, ok := msg.(view.NavigateToMsg)
	require.True(t, ok, "expected navigation, got %T", msg)
	require.Equal(t, chartsview.ViewName, nav.ViewName)
}

func TestChartAliasResolvesToTheSameView(t *testing.T) {
	alias, ok := registry.Get("chart")
	require.True(t, ok, ":chart must be registered")
	require.Equal(t, "charts", alias.(registry.Aliaser).AliasOf())

	nav := alias.Execute(nil, args.Args{})().(view.NavigateToMsg)
	require.Equal(t, chartsview.ViewName, nav.ViewName)
}

// The spec is what makes strict flag validation and the per-command help
// screen work at all: a command that declares none skips validateFlags
// entirely and falls back to generic help. The parse half of that is asserted
// in commands/api, which cannot be imported here — it pulls in the autoload
// package that registers this command.
func TestSpecIsDeclaredAndFlagless(t *testing.T) {
	spec, hasSpec := registry.SpecOf(ChartsLs{})
	require.True(t, hasSpec)
	require.NotEmpty(t, spec.Detail)
	require.Empty(t, spec.Flags, ":charts reads no flags")

	aliasSpec, hasAliasSpec := registry.SpecOf(chartAlias{})
	require.True(t, hasAliasSpec, "the alias inherits the primary's spec")
	require.Equal(t, spec.Detail, aliasSpec.Detail)
}
