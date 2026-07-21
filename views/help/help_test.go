// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package helpview

import (
	"github.com/Eldara-Tech/swarmcli/ui"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"
)

func TestNew_CommandList(t *testing.T) {
	cmds := []CommandInfo{
		{Name: "stacks", Description: "List stacks"},
		{Name: "help", Description: "Show help"},
	}
	m := New(80, 24, cmds)
	require.True(t, m.Visible)
	require.Len(t, m.commands, 2)
	require.Equal(t, 80, m.width)
}

func TestNewDetailed_Categories(t *testing.T) {
	cats := []HelpCategory{
		{Title: "General", Items: []HelpItem{
			{Keys: "<n>", Description: "New"},
		}},
	}
	m := NewDetailed(80, 24, cats)
	require.True(t, m.Visible)
	require.Len(t, m.categories, 1)
}

func TestSupportContact_CheatSheet(t *testing.T) {
	defer func() { SupportContact = "" }()
	cats := []HelpCategory{
		{Title: "General", Items: []HelpItem{{Keys: "<n>", Description: "New"}}},
	}
	m := NewDetailed(80, 24, cats)

	// Default (empty): OSS shows no support line.
	SupportContact = ""
	require.NotContains(t, m.buildCategorizedContent(), "SUPPORT")

	// Set: the full address is rendered (no truncation), lifted one blank
	// line off the footer (second-to-last body line, blank spacer below).
	SupportContact = "be-support@swarmcli.io"
	out := m.buildCategorizedContent()
	require.Contains(t, out, "SUPPORT")
	require.Contains(t, out, "be-support@swarmcli.io")
	lines := strings.Split(out, "\n")
	require.Equal(t, "", strings.TrimSpace(lines[len(lines)-1]), "blank spacer below SUPPORT")
	require.Contains(t, lines[len(lines)-2], "be-support@swarmcli.io")
}

func TestSupportContact_CommandList(t *testing.T) {
	defer func() { SupportContact = "" }()
	cmds := []CommandInfo{
		{Name: "stacks", Description: "List stacks"},
		{Name: "help", Description: "Show help"},
	}
	m := New(80, 24, cmds)

	// Default (empty): OSS shows no support line in the :help command list.
	SupportContact = ""
	require.NotContains(t, m.FrameContent(), "SUPPORT")

	// Set: the support line is surfaced in the :help command list too.
	SupportContact = "be-support@swarmcli.io"
	out := m.FrameContent()
	require.Contains(t, out, "SUPPORT")
	require.Contains(t, out, "be-support@swarmcli.io")
}

func TestName(t *testing.T) {
	m := New(80, 24, nil)
	require.Equal(t, "help", m.Name())
}

func TestInit(t *testing.T) {
	m := New(80, 24, nil)
	require.Nil(t, m.Init())
}

func TestShortHelpItems(t *testing.T) {
	m := New(80, 24, nil)
	items := m.ShortHelpItems()
	require.Len(t, items, 1)
	require.Equal(t, "esc", items[0].Key)
	require.Equal(t, "Close", items[0].Desc)
}

func TestOnEnterOnExit(t *testing.T) {
	m := New(80, 24, nil)
	require.Nil(t, m.OnEnter())
	require.Nil(t, m.OnExit())
}

func TestHasErrors(t *testing.T) {
	m := New(80, 24, nil)
	require.False(t, m.HasErrors())
}

func TestWindowSizeMsg(t *testing.T) {
	m := New(80, 24, nil)
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	require.Equal(t, 120, m.width)
	require.Equal(t, 40, m.height)
}

func TestView_CommandList(t *testing.T) {
	cmds := []CommandInfo{
		{Name: "stacks", Description: "List stacks"},
	}
	m := New(80, 24, cmds)
	out := m.View()
	require.Contains(t, out, "stacks")
	require.Contains(t, out, "List stacks")
}

func TestNew_AliasesTableLayout(t *testing.T) {
	cmds := []CommandInfo{
		{Name: "contexts", Description: "List and switch Docker contexts", Aliases: []string{"ctx", "context"}},
		{Name: "help", Description: "Show help"},
	}
	m := New(120, 24, cmds)
	content := m.content
	// Header line
	require.Contains(t, content, "COMMAND")
	require.Contains(t, content, "DESCRIPTION")
	require.Contains(t, content, "ALIASES")
	// Aliases column uses bare list, no "aliases:" prefix
	require.Contains(t, content, "ctx, context")
	require.NotContains(t, content, "aliases:")
	// Command without aliases has no trailing text after description
	for _, line := range strings.Split(content, "\n") {
		if strings.Contains(line, ":help") {
			require.NotContains(t, line, "ctx")
		}
	}
}

func TestNew_NoAliases_NoSuffix(t *testing.T) {
	cmds := []CommandInfo{
		{Name: "stacks", Description: "List stacks"},
	}
	m := New(80, 24, cmds)
	// Header still present
	require.Contains(t, m.content, "ALIASES")
	// No "aliases:" prefix anywhere
	require.NotContains(t, m.content, "aliases:")
	// Data row has no alias text
	for _, line := range strings.Split(m.content, "\n") {
		if strings.Contains(line, ":stacks") {
			trimmed := strings.TrimRight(line, " \t")
			require.True(t, strings.HasSuffix(trimmed, "List stacks"), "expected row to end with description, got: %q", trimmed)
		}
	}
}

func TestView_Categorized(t *testing.T) {
	cats := []HelpCategory{
		{Title: "Navigation", Items: []HelpItem{
			{Keys: "<esc>", Description: "Go back"},
			{Keys: "<ctrl+q>", Description: "Quit"},
		}},
		{Title: "Actions", Items: []HelpItem{
			{Keys: "<n>", Description: "New item"},
		}},
	}
	m := NewDetailed(120, 40, cats)
	out := m.View()
	require.Contains(t, out, "NAVIGATION")
	require.Contains(t, out, "ACTIONS")
	require.Contains(t, out, "Go back")
}

func TestView_NotVisible_Empty(t *testing.T) {
	m := New(80, 24, nil)
	m.Visible = false
	require.Equal(t, "", m.View())
}

func TestView_EmptyCategories_CommandList(t *testing.T) {
	m := New(80, 24, nil)
	out := m.View()
	// Should render command list path (even if empty)
	require.Contains(t, out, "Help")
}

// --- Filterable (app-level "/" filter) tests ---

func TestApplySearchQuery_CommandMode(t *testing.T) {
	cmds := []CommandInfo{
		{Name: "stacks", Description: "List stacks"},
		{Name: "help", Description: "Show help"},
		{Name: "contexts", Description: "List contexts", Aliases: []string{"ctx"}},
	}
	m := New(120, 24, cmds)
	m.ApplySearchQuery("stack")
	require.Contains(t, m.content, ":stacks")
	require.NotContains(t, m.content, ":help")
	require.NotContains(t, m.content, ":contexts")
}

func TestClearSearchQuery_CommandMode(t *testing.T) {
	cmds := []CommandInfo{
		{Name: "stacks", Description: "List stacks"},
		{Name: "help", Description: "Show help"},
	}
	m := New(120, 24, cmds)
	m.ApplySearchQuery("stack")
	m.ClearSearchQuery()
	require.Contains(t, m.content, ":stacks")
	require.Contains(t, m.content, ":help")
}

func TestApplySearchQuery_CommandMode_ByAlias(t *testing.T) {
	cmds := []CommandInfo{
		{Name: "contexts", Description: "List contexts", Aliases: []string{"ctx"}},
		{Name: "help", Description: "Show help"},
	}
	m := New(120, 24, cmds)
	m.ApplySearchQuery("ctx")
	require.Contains(t, m.content, ":contexts")
	require.NotContains(t, m.content, ":help")
}

func TestApplySearchQuery_CategorizedMode(t *testing.T) {
	cats := []HelpCategory{
		{Title: "Navigation", Items: []HelpItem{
			{Keys: "<esc>", Description: "Go back"},
			{Keys: "<ctrl+q>", Description: "Quit"},
		}},
		{Title: "Actions", Items: []HelpItem{
			{Keys: "<n>", Description: "New item"},
		}},
	}
	m := NewDetailed(120, 40, cats)
	m.ApplySearchQuery("quit")
	out := m.View()
	require.Contains(t, out, "Quit")
	require.NotContains(t, out, "Go back")
}

func TestClearSearchQuery_CategorizedMode(t *testing.T) {
	cats := []HelpCategory{
		{Title: "Navigation", Items: []HelpItem{
			{Keys: "<esc>", Description: "Go back"},
		}},
	}
	m := NewDetailed(120, 40, cats)
	m.ApplySearchQuery("nonexistent")
	m.ClearSearchQuery()
	out := m.View()
	require.Contains(t, out, "Go back")
}

func TestCommandListTip(t *testing.T) {
	list := New(80, 24, []CommandInfo{{Name: "help", Description: "Show help"}})

	// Frame header stays single-line so the box border is not broken.
	require.Equal(t, ui.FrameHeaderStyle.Render("Available Commands"), list.FrameHeader())

	// Tip is the first body line, then a blank line, then the table header.
	require.True(t, strings.HasPrefix(list.content, commandListTip+"\n\n"),
		"tip + blank line must precede the command table, got:\n%q", list.content)
	require.Contains(t, list.content, "COMMAND")

	// Tip stays after filtering (content is rebuilt).
	list.ApplySearchQuery("zzz-no-match")
	require.True(t, strings.HasPrefix(list.content, commandListTip+"\n\n"))

	// Tip must not leak into per-command or keybinding help.
	perCmd := NewCommandHelp(80, 24, CommandHelp{
		Title:    ":help",
		Sections: []HelpCategory{{Title: "Usage", Items: []HelpItem{{Keys: ":help"}}}},
	})
	require.NotContains(t, perCmd.View(), commandListTip)

	keys := NewDetailed(80, 24, []HelpCategory{
		{Title: "General", Items: []HelpItem{{Keys: "<n>", Description: "New"}}},
	})
	require.NotContains(t, keys.View(), commandListTip)
}
