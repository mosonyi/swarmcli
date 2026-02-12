// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package view

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"
)

func TestRegisterAction_And_GetAction(t *testing.T) {
	// Clean up after test
	defer delete(actionRegistry, "test-action")

	called := false
	RegisterAction("test-action", func(payload any) tea.Cmd {
		called = true
		return nil
	})

	action, ok := GetAction("test-action")
	require.True(t, ok)
	require.NotNil(t, action)

	action(nil)
	require.True(t, called)
}

func TestGetAction_NotRegistered(t *testing.T) {
	action, ok := GetAction("nonexistent-action")
	require.False(t, ok)
	require.Nil(t, action)
}

func TestHasAction(t *testing.T) {
	defer delete(actionRegistry, "has-test")

	require.False(t, HasAction("has-test"))

	RegisterAction("has-test", func(payload any) tea.Cmd { return nil })
	require.True(t, HasAction("has-test"))
}

func TestGetAction_Payload(t *testing.T) {
	defer delete(actionRegistry, "payload-test")

	var received any
	RegisterAction("payload-test", func(payload any) tea.Cmd {
		received = payload
		return nil
	})

	action, _ := GetAction("payload-test")
	action("my-secret")
	require.Equal(t, "my-secret", received)
}
