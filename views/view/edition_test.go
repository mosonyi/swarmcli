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
