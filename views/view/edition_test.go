// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package view

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"
)

func TestBEHelpDesc_Registered(t *testing.T) {
	RegisterAction("test-be-help", func(_ string) tea.Cmd { return nil })
	defer UnregisterActionForTest("test-be-help")
	require.Equal(t, "Reveal", BEHelpDesc("test-be-help", "Reveal"))
}

func TestBEHelpDesc_NotRegistered(t *testing.T) {
	require.Equal(t, "Shell (BE)", BEHelpDesc("nonexistent-action", "Shell"))
}

func TestBEUnavailableErr_ContainsFeatureName(t *testing.T) {
	err := BEUnavailableErr("Shell")
	require.Contains(t, err.Error(), "Shell")
}

func TestBEUnavailableErr_ContainsURL(t *testing.T) {
	err := BEUnavailableErr("Shell")
	require.Contains(t, err.Error(), "https://swarmcli.io/be")
}

func TestFeatureLockedCmd_NilWhenUnset(t *testing.T) {
	require.Nil(t, FeatureLockedCmdFn)
	require.Nil(t, FeatureLockedCmd("Shell"))
}

func TestFeatureLockedCmd_ReturnsRegisteredCmd(t *testing.T) {
	type lockedMsg struct{ feature string }
	FeatureLockedCmdFn = func(feature string) tea.Cmd {
		return func() tea.Msg { return lockedMsg{feature: feature} }
	}
	defer func() { FeatureLockedCmdFn = nil }()

	cmd := FeatureLockedCmd("Shell")
	require.NotNil(t, cmd)
	require.Equal(t, lockedMsg{feature: "Shell"}, cmd())
}
