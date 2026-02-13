// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package view

import tea "github.com/charmbracelet/bubbletea"

// Action is a feature action that can be triggered by key presses.
// The string parameter is the resource name (e.g. secret name).
// Registered during init() by extension packages, read-only after startup.
type Action func(name string) tea.Cmd

var actionRegistry = map[string]Action{}

// RegisterAction registers a named action. Must only be called from init() functions.
func RegisterAction(name string, fn Action) { actionRegistry[name] = fn }

// GetAction returns the action for the given name, if registered.
func GetAction(name string) (Action, bool) { fn, ok := actionRegistry[name]; return fn, ok }

// HasAction reports whether an action is registered under the given name.
func HasAction(name string) bool { _, ok := actionRegistry[name]; return ok }
