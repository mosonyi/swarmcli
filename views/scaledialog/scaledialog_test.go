// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package scaledialog

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"
)

func key(s string) tea.KeyMsg {
	switch s {
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
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
	m := New(80, 24)
	require.Equal(t, 80, m.Width)
	require.Equal(t, 24, m.Height)
	require.False(t, m.Visible)
}

func TestInit_ReturnsNil(t *testing.T) {
	require.Nil(t, New(80, 24).Init())
}

func TestShow(t *testing.T) {
	m := New(80, 24)
	m.Show("web", 3)
	require.True(t, m.Visible)
	require.Equal(t, "web", m.ServiceName)
	require.Equal(t, uint64(3), m.Replicas)
}

func TestHide(t *testing.T) {
	m := New(80, 24)
	m.Show("web", 3)
	m.Hide()
	require.False(t, m.Visible)
}

func TestUp_Increments(t *testing.T) {
	m := New(80, 24)
	m.Show("svc", 5)
	m.Update(key("up"))
	require.Equal(t, uint64(6), m.Replicas)
}

func TestDown_Decrements(t *testing.T) {
	m := New(80, 24)
	m.Show("svc", 5)
	m.Update(key("down"))
	require.Equal(t, uint64(4), m.Replicas)
}

func TestDown_AtZero_StaysZero(t *testing.T) {
	m := New(80, 24)
	m.Show("svc", 0)
	m.Update(key("down"))
	require.Equal(t, uint64(0), m.Replicas)
}

func TestUp_AtMax_StaysMax(t *testing.T) {
	m := New(80, 24)
	m.Show("svc", 1000)
	m.Update(key("up"))
	require.Equal(t, uint64(1000), m.Replicas)
}

func TestEnter_Confirms(t *testing.T) {
	m := New(80, 24)
	m.Show("svc", 7)
	cmd := m.Update(key("enter"))
	require.False(t, m.Visible)
	msg := runCmd(cmd).(ResultMsg)
	require.True(t, msg.Confirmed)
	require.Equal(t, uint64(7), msg.Replicas)
}

func TestEsc_Cancels(t *testing.T) {
	m := New(80, 24)
	m.Show("svc", 3)
	cmd := m.Update(key("esc"))
	msg := runCmd(cmd).(ResultMsg)
	require.False(t, msg.Confirmed)
}

func TestN_Cancels(t *testing.T) {
	m := New(80, 24)
	m.Show("svc", 3)
	cmd := m.Update(key("n"))
	msg := runCmd(cmd).(ResultMsg)
	require.False(t, msg.Confirmed)
}

func TestNotVisible_IgnoresKeys(t *testing.T) {
	m := New(80, 24)
	cmd := m.Update(key("up"))
	require.Nil(t, cmd)
}

func TestView_NotVisible_Empty(t *testing.T) {
	m := New(80, 24)
	require.Equal(t, "", m.View())
}

func TestView_ShowsServiceAndReplicas(t *testing.T) {
	m := New(80, 24)
	m.Show("my-service", 42)
	out := m.View()
	require.Contains(t, out, "my-service")
	require.Contains(t, out, "42")
	require.Contains(t, out, "Scale Service")
}
