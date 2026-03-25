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
	RegisterAction("test-action", func(name string) tea.Cmd {
		called = true
		return nil
	})

	action, ok := GetAction("test-action")
	require.True(t, ok)
	require.NotNil(t, action)

	action("")
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

	RegisterAction("has-test", func(name string) tea.Cmd { return nil })
	require.True(t, HasAction("has-test"))
}

func TestGetAction_Payload(t *testing.T) {
	defer delete(actionRegistry, "payload-test")

	var received string
	RegisterAction("payload-test", func(name string) tea.Cmd {
		received = name
		return nil
	})

	action, _ := GetAction("payload-test")
	action("my-secret")
	require.Equal(t, "my-secret", received)
}

func TestRegisterGatedAction_GuardTrue(t *testing.T) {
	defer delete(actionRegistry, "gated-true")

	called := false
	RegisterGatedAction("gated-true", func() bool { return true }, func(name string) tea.Cmd {
		called = true
		return nil
	})

	require.True(t, HasAction("gated-true"))
	action, ok := GetAction("gated-true")
	require.True(t, ok)
	require.NotNil(t, action)
	action("")
	require.True(t, called)
}

func TestRegisterGatedAction_GuardFalse(t *testing.T) {
	defer delete(actionRegistry, "gated-false")

	RegisterGatedAction("gated-false", func() bool { return false }, func(name string) tea.Cmd {
		return nil
	})

	require.False(t, HasAction("gated-false"))
	action, ok := GetAction("gated-false")
	require.False(t, ok)
	require.Nil(t, action)
}

func TestRegisterGatedAction_GuardChanges(t *testing.T) {
	defer delete(actionRegistry, "gated-toggle")

	enabled := false
	RegisterGatedAction("gated-toggle", func() bool { return enabled }, func(name string) tea.Cmd {
		return nil
	})

	require.False(t, HasAction("gated-toggle"))
	_, ok := GetAction("gated-toggle")
	require.False(t, ok)

	enabled = true
	require.True(t, HasAction("gated-toggle"))
	_, ok = GetAction("gated-toggle")
	require.True(t, ok)

	enabled = false
	require.False(t, HasAction("gated-toggle"))
}
