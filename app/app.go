package app

import (
	helpview "swarmcli/views/help"
	inspectview "swarmcli/views/inspect"
	logsview "swarmcli/views/logs"
	stacksview "swarmcli/views/stacks"
	"swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"

	_ "swarmcli/commands"
)

const (
	version string = "dev"
)

var viewRegistry = map[string]view.Factory{}

func registerView(name string, factory view.Factory) {
	viewRegistry[name] = factory
}

// Init should be called once at the start of the application to register all views.
func Init() {
	registerView(helpview.ViewName, func(w, h int, payload any) (view.View, tea.Cmd) {
		return helpview.New(w, h), logsview.Load(payload.(string))
	})
	registerView(logsview.ViewName, func(w, h int, payload any) (view.View, tea.Cmd) {
		return logsview.New(w, h), logsview.Load(payload.(string))
	})
	registerView(inspectview.ViewName, func(w, h int, payload any) (view.View, tea.Cmd) {
		return inspectview.New(w, h), inspectview.LoadInspectItem(payload.(string))
	})
	registerView(stacksview.ViewName, func(w, h int, payload any) (view.View, tea.Cmd) {
		return stacksview.New(w, h), stacksview.LoadNodeStacks(payload.(string))
	})
}
