package loadingview

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"
)

func TestNew_StringPayload(t *testing.T) {
	m := New(80, 24, true, "Deploying stack...")
	require.Equal(t, "Deploying stack...", m.message)
	require.Equal(t, "Loading", m.title)
	require.True(t, m.visible)
}

func TestNew_MapPayload(t *testing.T) {
	m := New(80, 24, true, map[string]string{
		"title":   "Deploy",
		"header":  "Stack: web",
		"message": "Please wait...",
	})
	require.Equal(t, "Deploy", m.title)
	require.Equal(t, "Stack: web", m.header)
	require.Equal(t, "Please wait...", m.message)
}

func TestNew_MapInterfacePayload(t *testing.T) {
	m := New(80, 24, true, map[string]any{
		"title":   "Op",
		"message": "working",
	})
	require.Equal(t, "Op", m.title)
	require.Equal(t, "working", m.message)
}

func TestNew_NilPayload(t *testing.T) {
	m := New(80, 24, true, nil)
	require.Equal(t, "Please wait...", m.message)
}

func TestNew_ErrorMessage_SetsErrorMode(t *testing.T) {
	m := New(80, 24, true, "Error: connection refused")
	require.True(t, m.isError)
}

func TestVisible(t *testing.T) {
	m := New(80, 24, false, nil)
	require.False(t, m.Visible())
	m.SetVisible(true)
	require.True(t, m.Visible())
}

func TestName(t *testing.T) {
	m := New(80, 24, true, nil)
	require.Equal(t, "loading", m.Name())
}

func TestSpinnerTickMsg_AdvancesSpinner(t *testing.T) {
	m := New(80, 24, true, nil)
	require.Equal(t, 0, m.GetSpinnerFrame())
	cmd := m.Update(SpinnerTickMsg(time.Now()))
	require.Equal(t, 1, m.GetSpinnerFrame())
	require.NotNil(t, cmd) // should schedule next tick
}

func TestWindowSizeMsg(t *testing.T) {
	m := New(80, 24, true, nil)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	require.Equal(t, 104, m.width) // +4 for padding
	require.Equal(t, 50, m.height)
}

func TestErrorMode_EnterDismisses(t *testing.T) {
	m := New(80, 24, true, "Error: timeout")
	cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)
	msg := cmd()
	_, ok := msg.(ErrorDismissedMsg)
	require.True(t, ok)
}

func TestNonError_EnterIgnored(t *testing.T) {
	m := New(80, 24, true, "Loading...")
	cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.Nil(t, cmd)
}

func TestView_NotVisible_Empty(t *testing.T) {
	m := New(80, 24, false, nil)
	require.Equal(t, "", m.View())
}

func TestView_Visible_ContainsMessage(t *testing.T) {
	m := New(80, 24, true, "Loading secrets...")
	out := m.View()
	require.Contains(t, out, "Loading secrets...")
}

func TestView_ErrorMode_ContainsError(t *testing.T) {
	m := New(80, 24, true, "Error: cannot connect")
	out := m.View()
	require.Contains(t, out, "Error")
	require.Contains(t, out, "cannot connect")
}

func TestShortHelpItems(t *testing.T) {
	m := New(80, 24, true, nil)
	items := m.ShortHelpItems()
	require.Len(t, items, 1)
	require.Equal(t, "q", items[0].Key)
}

func TestInit_ReturnsCmd(t *testing.T) {
	m := New(80, 24, true, nil)
	cmd := m.Init()
	require.NotNil(t, cmd) // spinner tick cmd
}

func TestOnEnterOnExit(t *testing.T) {
	m := New(80, 24, true, nil)
	require.Nil(t, m.OnEnter())
	require.Nil(t, m.OnExit())
}

func TestHasErrors(t *testing.T) {
	m := New(80, 24, true, nil)
	require.False(t, m.HasErrors())
}

func TestSetSize(t *testing.T) {
	m := New(80, 24, true, nil)
	m.SetSize(120, 60)
	require.Equal(t, 120, m.width)
	require.Equal(t, 60, m.height)
}
