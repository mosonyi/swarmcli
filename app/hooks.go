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

var shutdownHooks []func()

// RegisterShutdownHook adds a function to run when the program exits.
// Must be called from init(). Hooks run synchronously in registration order
// from RunShutdownHooks, which the entry point invokes after tea.Program.Run
// returns. They also fire from a SIGINT/SIGTERM handler set up by callers
// that need defensive cleanup on panic paths.
func RegisterShutdownHook(fn func()) {
	shutdownHooks = append(shutdownHooks, fn)
}

// RunShutdownHooks executes all registered shutdown hooks in order.
// Safe to call multiple times; hooks themselves must be idempotent.
func RunShutdownHooks() {
	for _, h := range shutdownHooks {
		h()
	}
}
