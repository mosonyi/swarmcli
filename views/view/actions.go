// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package view

import tea "github.com/charmbracelet/bubbletea"

// Action is a feature action that can be triggered by key presses.
// The string parameter is the resource name (e.g. secret name).
// Registered during init() by extension packages, read-only after startup.
type Action func(name string) tea.Cmd

type registeredAction struct {
	guard func() bool // nil = always available
	fn    Action
}

var actionRegistry = map[string]registeredAction{}

// RegisterAction registers a named action. Must only be called from init() functions.
func RegisterAction(name string, fn Action) {
	actionRegistry[name] = registeredAction{fn: fn}
}

// RegisterGatedAction registers an action with a runtime guard.
// GetAction and HasAction return false when the guard returns false.
// Must only be called from init() functions.
func RegisterGatedAction(name string, guard func() bool, fn Action) {
	actionRegistry[name] = registeredAction{guard: guard, fn: fn}
}

// GetAction returns the action for the given name, if registered and its guard passes.
func GetAction(name string) (Action, bool) {
	a, ok := actionRegistry[name]
	if !ok || (a.guard != nil && !a.guard()) {
		return nil, false
	}
	return a.fn, true
}

// HasAction reports whether an action is registered and its guard passes.
func HasAction(name string) bool {
	a, ok := actionRegistry[name]
	return ok && (a.guard == nil || a.guard())
}

// UnregisterActionForTest removes a registered action. Test-only; not safe for concurrent use.
func UnregisterActionForTest(name string) { delete(actionRegistry, name) }
