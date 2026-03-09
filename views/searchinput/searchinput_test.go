// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package searchinput

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	m := New()
	require.False(t, m.Visible())
	require.False(t, m.Editing())
	require.Equal(t, "", m.Query())
	require.Equal(t, "", m.View())
}

func TestShowHide(t *testing.T) {
	m := New()
	cmd := m.Show()
	require.True(t, m.Visible())
	require.True(t, m.Editing())
	require.NotNil(t, cmd) // textinput.Blink

	m.Hide()
	require.False(t, m.Visible())
	require.False(t, m.Editing())
	require.Equal(t, "", m.Query())
}

func TestUpdateIgnoredWhenInactive(t *testing.T) {
	m := New()
	cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	require.Nil(t, cmd)
	require.Equal(t, "", m.Query())
}

func TestTypingEmitsSearchQueryMsg(t *testing.T) {
	m := New()
	m.Show()

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	require.NotNil(t, cmd)
	require.Equal(t, "w", m.Query())

	// Execute the batch and find the SearchQueryMsg
	found := false
	if batch, ok := cmd().(tea.BatchMsg); ok {
		for _, c := range batch {
			if msg, ok := c().(SearchQueryMsg); ok {
				require.Equal(t, "w", msg.Query)
				found = true
			}
		}
	}
	require.True(t, found, "expected SearchQueryMsg in batch")
}

func TestEnterConfirmsToPassiveMode(t *testing.T) {
	m := New()
	m.Show()
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.True(t, m.Visible(), "box should stay visible")
	require.False(t, m.Editing(), "should no longer be editing")
	require.Equal(t, "x", m.Query(), "query should be preserved")
	require.Nil(t, cmd)
}

func TestEscEmitsSearchClearedMsg(t *testing.T) {
	m := New()
	m.Show()
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	require.False(t, m.Visible())
	require.NotNil(t, cmd)

	msg := cmd()
	_, ok := msg.(SearchClearedMsg)
	require.True(t, ok, "expected SearchClearedMsg")
}

func TestBackspaceOnEmptyEmitsSearchClearedMsg(t *testing.T) {
	m := New()
	m.Show()

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	require.False(t, m.Visible())
	require.NotNil(t, cmd)

	msg := cmd()
	_, ok := msg.(SearchClearedMsg)
	require.True(t, ok, "expected SearchClearedMsg")
}

func TestBackspaceOnNonEmptyEmitsSearchQueryMsg(t *testing.T) {
	m := New()
	m.Show()
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	require.Equal(t, "ab", m.Query())

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	require.True(t, m.Visible())
	require.Equal(t, "a", m.Query())

	found := false
	if batch, ok := cmd().(tea.BatchMsg); ok {
		for _, c := range batch {
			if msg, ok := c().(SearchQueryMsg); ok {
				require.Equal(t, "a", msg.Query)
				found = true
			}
		}
	}
	require.True(t, found, "expected SearchQueryMsg in batch")
}

func TestViewRendersWhenActive(t *testing.T) {
	m := New()
	require.Equal(t, "", m.View())

	m.Show()
	v := m.View()
	require.NotEmpty(t, v)
	require.Contains(t, v, "/")
}

func TestPassiveModeIgnoresKeys(t *testing.T) {
	m := New()
	m.Show()
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m.Confirm()

	require.True(t, m.Visible())
	require.False(t, m.Editing())

	// Keys should be ignored in passive mode
	cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	require.Nil(t, cmd)
	require.Equal(t, "a", m.Query(), "query should not change in passive mode")
}

func TestResumeTransitionsToEditing(t *testing.T) {
	m := New()
	m.Show()
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m.Confirm()
	require.False(t, m.Editing())

	cmd := m.Resume()
	require.True(t, m.Visible())
	require.True(t, m.Editing())
	require.NotNil(t, cmd, "Resume should return blink command")
	require.Equal(t, "a", m.Query(), "query should be preserved after resume")
}

func TestViewRendersInPassiveMode(t *testing.T) {
	m := New()
	m.Show()
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m.Confirm()

	v := m.View()
	require.NotEmpty(t, v, "passive mode should still render")
	require.Contains(t, v, "/")
	require.Contains(t, v, "x")
}

func TestConfirmThenHide(t *testing.T) {
	m := New()
	m.Show()
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m.Confirm()

	require.True(t, m.Visible())
	require.Equal(t, "q", m.Query())

	m.Hide()
	require.False(t, m.Visible())
	require.False(t, m.Editing())
	require.Equal(t, "", m.Query())
}
