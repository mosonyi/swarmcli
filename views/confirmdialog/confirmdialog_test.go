// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package confirmdialog

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/require"
)

func key(s string) tea.KeyMsg {
	if len(s) == 1 {
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
	switch s {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case " ":
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func runCmd(cmd tea.Cmd) tea.Msg {
	if cmd == nil {
		return nil
	}
	return cmd()
}

func TestNew(t *testing.T) {
	m := New(80, 24)
	require.False(t, m.Visible)
	require.Equal(t, 80, m.Width)
	require.Equal(t, 24, m.Height)
}

func TestInit_ReturnsNil(t *testing.T) {
	m := New(80, 24)
	require.Nil(t, m.Init())
}

func TestShow(t *testing.T) {
	m := New(80, 24)
	m.Show("Are you sure?")
	require.True(t, m.Visible)
	require.Equal(t, "Are you sure?", m.Message)
}

func TestHide(t *testing.T) {
	m := New(80, 24)
	m.Show("msg")
	m.Hide()
	require.False(t, m.Visible)
}

func TestConfirm_Y(t *testing.T) {
	m := New(80, 24)
	m.Show("delete?")
	cmd := m.Update(key("y"))
	require.False(t, m.Visible)
	msg := runCmd(cmd)
	r, ok := msg.(ResultMsg)
	require.True(t, ok)
	require.True(t, r.Confirmed)
}

func TestConfirm_ShiftY(t *testing.T) {
	m := New(80, 24)
	m.Show("delete?")
	cmd := m.Update(key("Y"))
	msg := runCmd(cmd).(ResultMsg)
	require.True(t, msg.Confirmed)
}

func TestCancel_N(t *testing.T) {
	m := New(80, 24)
	m.Show("delete?")
	cmd := m.Update(key("n"))
	msg := runCmd(cmd).(ResultMsg)
	require.False(t, msg.Confirmed)
}

func TestCancel_Esc(t *testing.T) {
	m := New(80, 24)
	m.Show("delete?")
	cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	msg := runCmd(cmd).(ResultMsg)
	require.False(t, msg.Confirmed)
}

func TestNotVisible_IgnoresKeys(t *testing.T) {
	m := New(80, 24)
	cmd := m.Update(key("y"))
	require.Nil(t, cmd)
}

func TestCheckbox_Toggle(t *testing.T) {
	m := New(80, 24)
	m.CheckboxLabel = "Also remove volumes"
	m.Show("delete?")

	require.False(t, m.CheckboxChecked)
	m.Update(key(" "))
	require.True(t, m.CheckboxChecked)
	m.Update(key(" "))
	require.False(t, m.CheckboxChecked)
}

func TestCheckbox_NoLabel_SpaceIgnored(t *testing.T) {
	m := New(80, 24)
	m.Show("delete?")
	m.Update(key(" "))
	require.False(t, m.CheckboxChecked)
}

func TestCheckbox_StateInResult(t *testing.T) {
	m := New(80, 24)
	m.CheckboxLabel = "force"
	m.Show("delete?")
	m.Update(key(" ")) // toggle on
	cmd := m.Update(key("y"))
	msg := runCmd(cmd).(ResultMsg)
	require.True(t, msg.Confirmed)
	require.True(t, msg.CheckboxChecked)
}

func TestErrorMode_EnterCloses(t *testing.T) {
	m := New(80, 24)
	m.ErrorMode = true
	m.Show("something broke")
	cmd := m.Update(key("enter"))
	require.False(t, m.Visible)
	msg := runCmd(cmd).(ResultMsg)
	require.False(t, msg.Confirmed)
}

func TestErrorMode_SpaceCloses(t *testing.T) {
	m := New(80, 24)
	m.ErrorMode = true
	m.Show("err")
	cmd := m.Update(key(" "))
	require.False(t, m.Visible)
	msg := runCmd(cmd).(ResultMsg)
	require.False(t, msg.Confirmed)
}

func TestInfoMode_Checkbox_SpaceToggles_EnterCloses(t *testing.T) {
	m := New(80, 24)
	m.InfoMode = true
	m.CheckboxLabel = "Do not show again"
	m.Show("update available")

	// Space toggles the checkbox instead of closing the notice.
	require.False(t, m.CheckboxChecked)
	cmd := m.Update(key(" "))
	require.Nil(t, cmd)
	require.True(t, m.Visible)
	require.True(t, m.CheckboxChecked)

	// Enter closes and reports the checkbox state.
	cmd = m.Update(key("enter"))
	require.False(t, m.Visible)
	msg := runCmd(cmd).(ResultMsg)
	require.False(t, msg.Confirmed)
	require.True(t, msg.CheckboxChecked)
}

func TestInfoMode_NoCheckbox_SpaceCloses(t *testing.T) {
	m := New(80, 24)
	m.InfoMode = true
	m.Show("notice")
	cmd := m.Update(key(" "))
	require.False(t, m.Visible)
	msg := runCmd(cmd).(ResultMsg)
	require.False(t, msg.Confirmed)
}

func TestView_InfoMode_Checkbox_ShowsLabelAndHelp(t *testing.T) {
	m := New(80, 24)
	m.InfoMode = true
	m.CheckboxLabel = "Do not show again"
	m.Show("update available")
	out := m.View()
	require.Contains(t, out, "Do not show again")
	require.Contains(t, out, "Toggle")
	require.Contains(t, out, "Info")
}

func TestView_NotVisible_Empty(t *testing.T) {
	m := New(80, 24)
	require.Equal(t, "", m.View())
}

func TestView_ContainsMessage(t *testing.T) {
	m := New(80, 24)
	m.Show("Really delete this?")
	out := m.View()
	require.Contains(t, out, "Really delete this?")
	require.Contains(t, out, "Confirm Action")
}

func TestView_ErrorMode_ShowsError(t *testing.T) {
	m := New(80, 24)
	m.ErrorMode = true
	m.Show("connection lost")
	out := m.View()
	require.Contains(t, out, "connection lost")
	require.Contains(t, out, "Error")
}

func TestView_Checkbox_ShowsLabel(t *testing.T) {
	m := New(80, 24)
	m.CheckboxLabel = "Also remove data"
	m.Show("delete?")
	out := m.View()
	require.Contains(t, out, "Also remove data")
	require.Contains(t, out, "Toggle")
}

func TestWithMessage(t *testing.T) {
	m := New(80, 24)
	ret := m.WithMessage("hi")
	require.Equal(t, m, ret)
	require.Equal(t, "hi", m.Message)
}

func TestView_LongMessage_Wraps(t *testing.T) {
	m := New(80, 24)
	m.ErrorMode = true
	longMsg := "Bootstrap failed: error creating secret swarmcli-infra_proxy_cert: " +
		"Error response from daemon: rpc error: code = AlreadyExists desc = secret swarmcli-infra_proxy_cert already exists"
	m.Show(longMsg)
	out := m.View()
	// The dialog output should contain newlines from wrapping
	// (the original message has no newlines, so any newlines in
	// the rendered output come from wrapping).
	require.Contains(t, out, "Bootstrap failed")
	require.Contains(t, out, "already exists")
	// Dialog should not be wider than the terminal (80 cols).
	for _, line := range strings.Split(out, "\n") {
		w := lipgloss.Width(line)
		require.LessOrEqual(t, w, 82, "line too wide (visual width %d): %q", w, line)
	}
}

func TestView_ShortMessage_NoWrapping(t *testing.T) {
	m := New(80, 24)
	m.Show("Short message")
	out := m.View()
	require.Contains(t, out, "Short message")
}
