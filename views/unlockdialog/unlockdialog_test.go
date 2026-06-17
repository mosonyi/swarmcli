// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package unlockdialog

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
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func runCmd(cmd tea.Cmd) tea.Msg {
	if cmd == nil {
		return nil
	}
	return cmd()
}

func typeRunes(m *Model, s string) {
	for _, r := range s {
		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
}

func TestNew(t *testing.T) {
	m := New(80, 24)
	require.Equal(t, 80, m.Width)
	require.Equal(t, 24, m.Height)
	require.False(t, m.Visible)
}

func TestShowHide(t *testing.T) {
	m := New(80, 24)
	m.Show()
	require.True(t, m.Visible)
	m.Hide()
	require.False(t, m.Visible)
}

func TestShow_ClearsPreviousValue(t *testing.T) {
	m := New(80, 24)
	m.Show()
	typeRunes(m, "stale")
	cmd := m.Update(key("esc"))
	_ = runCmd(cmd)
	m.Show()
	cmd = m.Update(key("enter"))
	msg := runCmd(cmd).(ResultMsg)
	require.Equal(t, "", msg.Key)
}

func TestEnter_ConfirmsWithKey(t *testing.T) {
	m := New(80, 24)
	m.Show()
	typeRunes(m, "SWMKEY-1-abc")
	cmd := m.Update(key("enter"))
	require.False(t, m.Visible)
	msg := runCmd(cmd).(ResultMsg)
	require.True(t, msg.Confirmed)
	require.Equal(t, "SWMKEY-1-abc", msg.Key)
}

func TestEsc_Cancels(t *testing.T) {
	m := New(80, 24)
	m.Show()
	cmd := m.Update(key("esc"))
	require.False(t, m.Visible)
	msg := runCmd(cmd).(ResultMsg)
	require.False(t, msg.Confirmed)
}

func TestNotVisible_IgnoresKeys(t *testing.T) {
	m := New(80, 24)
	require.Nil(t, m.Update(key("enter")))
}

func TestView_NotVisible_Empty(t *testing.T) {
	require.Equal(t, "", New(80, 24).View())
}

func TestView_MasksKeyAndShowsTitle(t *testing.T) {
	m := New(80, 24)
	m.Show()
	typeRunes(m, "SECRETKEY")
	out := m.View()
	require.Contains(t, out, "Unlock Swarm")
	require.NotContains(t, out, "SECRETKEY")
}
