// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package view

import tea "github.com/charmbracelet/bubbletea"

type NavigateToMsg struct {
	ViewName string
	Payload  any // Can be service ID, stack ID, etc.
	// Replace indicates whether the target view should replace the current
	// view (i.e., not be pushed onto the navigation stack). When false,
	// the view manager should push the new view onto the history stack.
	Replace bool
}

type NavigateBackMsg struct{}

type Navigator interface {
	NavigateTo(name string, payload any) tea.Cmd
}
