package commandinput

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"
)

func key(s string) tea.KeyMsg {
	switch s {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	}
	if len(s) == 1 {
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
	return tea.KeyMsg{}
}

func runCmd(cmd tea.Cmd) tea.Msg {
	if cmd == nil {
		return nil
	}
	return cmd()
}

func TestNew(t *testing.T) {
	m := New()
	require.False(t, m.Visible())
}

func TestShow_MakesVisible(t *testing.T) {
	m := New()
	m.Show()
	require.True(t, m.Visible())
}

func TestHide_MakesInvisible(t *testing.T) {
	m := New()
	m.Show()
	m.Hide()
	require.False(t, m.Visible())
}

func TestShowError_SetsError(t *testing.T) {
	m := New()
	m.Show()
	m.ShowError("bad command")
	require.Equal(t, "bad command", m.errorMsg)
}

func TestEnter_DispatchesSubmitMsg(t *testing.T) {
	m := New()
	m.Show()
	m.input.SetValue("stacks")
	cmd := m.Update(key("enter"))
	require.False(t, m.Visible(), "should hide after enter")
	msg := runCmd(cmd)
	sub, ok := msg.(SubmitMsg)
	require.True(t, ok)
	require.Equal(t, "stacks", sub.Command)
}

func TestEnter_TrimsWhitespace(t *testing.T) {
	m := New()
	m.Show()
	m.input.SetValue("  stacks  ")
	cmd := m.Update(key("enter"))
	msg := runCmd(cmd).(SubmitMsg)
	require.Equal(t, "stacks", msg.Command)
}

func TestEsc_HidesWithoutSubmit(t *testing.T) {
	m := New()
	m.Show()
	m.input.SetValue("something")
	cmd := m.Update(key("esc"))
	require.False(t, m.Visible())
	require.Nil(t, cmd)
}

func TestUpDown_CyclesSuggestions(t *testing.T) {
	m := New()
	m.Show()
	m.suggestions = []string{"a", "b", "c"}
	m.selected = 0

	m.Update(key("down"))
	require.Equal(t, 1, m.selected)

	m.Update(key("down"))
	require.Equal(t, 2, m.selected)

	m.Update(key("down"))
	require.Equal(t, 0, m.selected) // wraps

	m.Update(key("up"))
	require.Equal(t, 2, m.selected) // wraps back
}

func TestTab_AutocompletesSuggestion(t *testing.T) {
	m := New()
	m.Show()
	m.suggestions = []string{"stacks", "services"}
	m.selected = 1

	m.Update(key("tab"))
	require.Equal(t, "services ", m.input.Value())
}

func TestTab_NoSuggestions_NoOp(t *testing.T) {
	m := New()
	m.Show()
	m.suggestions = nil
	m.Update(key("tab"))
	require.Equal(t, "", m.input.Value())
}

func TestNotActive_IgnoresKeys(t *testing.T) {
	m := New()
	cmd := m.Update(key("enter"))
	require.Nil(t, cmd)
}

func TestView_NotActive_Empty(t *testing.T) {
	m := New()
	require.Equal(t, "", m.View())
}

func TestView_Active_ShowsInput(t *testing.T) {
	m := New()
	m.Show()
	m.input.SetValue("test")
	out := m.View()
	require.Contains(t, out, "test")
	require.Contains(t, out, ">")
}
