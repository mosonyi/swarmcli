package app

import (
	tea "github.com/charmbracelet/bubbletea"
	inspectview "swarmcli/views/inspect"
	logsview "swarmcli/views/logs"
	stacksview "swarmcli/views/stacks"
	"swarmcli/views/view"
)

const (
	modeNodes mode   = "nodes"
	version   string = "dev"
)

var viewRegistry = map[string]view.Factory{}

func registerView(name string, factory view.Factory) {
	viewRegistry[name] = factory
}

// Should be called once at the start of the application to register all views
func Init() {
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
