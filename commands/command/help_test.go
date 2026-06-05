// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package command

import (
	"testing"

	"swarmcli/args"
	"swarmcli/registry"
	helpview "swarmcli/views/help"
	"swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"
)

type fullSpecCmd struct{}

func (fullSpecCmd) Name() string                       { return "test_fullspec" }
func (fullSpecCmd) Description() string                { return "a documented command" }
func (fullSpecCmd) Execute(_ any, _ args.Args) tea.Cmd { return nil }
func (fullSpecCmd) Spec() registry.CommandSpec {
	return registry.CommandSpec{
		Usage:    "<arg>",
		Flags:    []registry.FlagSpec{{Name: "force", Short: "f", Description: "force it"}},
		Examples: []string{":test_fullspec --force"},
	}
}

type noSpecCmd struct{}

func (noSpecCmd) Name() string                       { return "test_nospec_cmd" }
func (noSpecCmd) Description() string                { return "undocumented" }
func (noSpecCmd) Execute(_ any, _ args.Args) tea.Cmd { return nil }

func titles(cats []helpview.HelpCategory) []string {
	out := make([]string, len(cats))
	for i, c := range cats {
		out[i] = c.Title
	}
	return out
}

func TestCommandHelpCategories_WithSpec(t *testing.T) {
	cats := CommandHelpCategories(fullSpecCmd{})
	require.Equal(t, []string{"Usage", "Flags", "Examples"}, titles(cats))
	require.Equal(t, ":test_fullspec <arg>", cats[0].Items[0].Keys)
	require.Equal(t, "--force, -f", cats[1].Items[0].Keys)
	require.Equal(t, ":test_fullspec --force", cats[2].Items[0].Keys)
}

func TestCommandHelpCategories_NoSpecFallback(t *testing.T) {
	cats := CommandHelpCategories(noSpecCmd{})
	require.Len(t, cats, 1)
	require.Equal(t, "Usage", cats[0].Title)
	require.Equal(t, ":test_nospec_cmd", cats[0].Items[0].Keys)
	require.Equal(t, "undocumented", cats[0].Items[0].Description)
}

func TestHelp_Execute_NoArgs_GlobalList(t *testing.T) {
	msg := Help{}.Execute(nil, args.Args{})()
	nav, ok := msg.(view.NavigateToMsg)
	require.True(t, ok)
	_, ok = nav.Payload.([]helpview.CommandInfo)
	require.True(t, ok, "no-arg :help must keep the []CommandInfo payload")
}

func TestHelp_Execute_PerCommand(t *testing.T) {
	msg := Help{}.Execute(nil, args.Args{Positionals: []string{"quit"}})()
	nav, ok := msg.(view.NavigateToMsg)
	require.True(t, ok)
	require.Equal(t, helpview.ViewName, nav.ViewName)
	ch, ok := nav.Payload.(helpview.CommandHelp)
	require.True(t, ok, ":help <cmd> must use the CommandHelp payload")
	require.Equal(t, ":quit", ch.Title)
	require.NotEmpty(t, ch.Sections)
}

func TestHelp_Execute_UnknownCommand(t *testing.T) {
	msg := Help{}.Execute(nil, args.Args{Positionals: []string{"quti"}})()
	errMsg, ok := msg.(view.AppErrorMsg)
	require.True(t, ok)
	require.Contains(t, errMsg.Error, "unknown command: quti")
	require.Contains(t, errMsg.Error, "did you mean :quit?")
}
