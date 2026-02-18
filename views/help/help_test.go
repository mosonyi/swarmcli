package helpview

import (
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
	require.Equal(t, "q", items[0].Key)
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

func TestNew_AliasesRenderedInline(t *testing.T) {
	cmds := []CommandInfo{
		{Name: "contexts", Description: "List and switch Docker contexts", Aliases: []string{"ctx", "context"}},
		{Name: "help", Description: "Show help"},
	}
	m := New(120, 24, cmds)
	content := m.content
	require.Contains(t, content, "aliases: ctx, context")
	// Command without aliases should not contain "aliases:"
	require.NotContains(t, content, "help"+strings.Repeat(" ", 10)+"Show help    aliases:")
}

func TestNew_NoAliases_NoSuffix(t *testing.T) {
	cmds := []CommandInfo{
		{Name: "stacks", Description: "List stacks"},
	}
	m := New(80, 24, cmds)
	require.NotContains(t, m.content, "aliases:")
}

func TestView_Categorized(t *testing.T) {
	cats := []HelpCategory{
		{Title: "Navigation", Items: []HelpItem{
			{Keys: "<esc>", Description: "Go back"},
			{Keys: "<q>", Description: "Quit"},
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
