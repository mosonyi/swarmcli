package app

import (
	"fmt"
	"swarmcli/utils/log"
	helpview "swarmcli/views/help"
	inspectview "swarmcli/views/inspect"
	logsview "swarmcli/views/logs"
	nodesview "swarmcli/views/nodes"
	stacksview "swarmcli/views/stacks"
	"swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"

	_ "swarmcli/commands" // triggers autoload
	"swarmcli/registry"
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
	log.InitDebug()

	for _, cmd := range registry.All() {
		fmt.Println("-", cmd.Name(), "→", cmd.Description())
	}

	registerView(helpview.ViewName, func(w, h int, payload any) (view.View, tea.Cmd) {
		cmds, _ := payload.([]helpview.CommandInfo)
		return helpview.New(w, h, cmds), nil
	})
	registerView(logsview.ViewName, func(w, h int, payload any) (view.View, tea.Cmd) {
		return logsview.New(w, h), logsview.Load(payload.(string))
	})
	registerView(inspectview.ViewName, func(w, h int, payload any) (view.View, tea.Cmd) {
		return inspectview.New(w, h), inspectview.LoadInspectItem(payload.(string))
	})
	registerView(nodesview.ViewName, func(w, h int, payload any) (view.View, tea.Cmd) {
		return nodesview.New(w, h), nodesview.LoadNodes()
	})
	registerView(stacksview.ViewName, func(w, h int, payload any) (view.View, tea.Cmd) {
		var nodeID string
		if payload != nil {
			nodeID, _ = payload.(string)
		}
		return stacksview.New(w, h), stacksview.LoadStacks(nodeID)
	})
}
