// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package app

import tea "github.com/charmbracelet/bubbletea"

// PreUpdateHook can intercept messages before the core Update dispatch.
// Return handled=true to stop processing; the returned cmd is used.
// Return handled=false to let the message continue to core Update.
type PreUpdateHook func(viewName string, msg tea.Msg) (handled bool, cmd tea.Cmd)

var preUpdateHooks []PreUpdateHook

// RegisterPreUpdateHook adds a hook. Must be called from init().
func RegisterPreUpdateHook(hook PreUpdateHook) {
	preUpdateHooks = append(preUpdateHooks, hook)
}

// StartupOverlay is a component displayed on top of the initial TUI view.
// While Active, the app routes KeyMsg exclusively to its Update and
// composites its View on top of the rendered output.
type StartupOverlay interface {
	Active() bool
	Update(tea.Msg) tea.Cmd
	View() string
}

var startupOverlay StartupOverlay

// SetStartupOverlay registers a startup overlay. Must be called from init().
func SetStartupOverlay(o StartupOverlay) {
	startupOverlay = o
}
