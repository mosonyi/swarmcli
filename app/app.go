package app

import (
	"fmt"
	l "swarmcli/utils/log"
	helpview "swarmcli/views/help"
	inspectview "swarmcli/views/inspect"
	loadingview "swarmcli/views/loading"
	logsview "swarmcli/views/logs"
	nodesview "swarmcli/views/nodes"
	nodeservicesview "swarmcli/views/nodeservices"
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
	l.InitDebug()

	for _, cmd := range registry.All() {
		fmt.Println("-", cmd.Name(), "→", cmd.Description())
	}

	registerView(loadingview.ViewName, func(w, h int, payload any) (view.View, tea.Cmd) {
		return loadingview.New(w, h, true, payload), nil
	})
	registerView(helpview.ViewName, func(w, h int, payload any) (view.View, tea.Cmd) {
		cmds, _ := payload.([]helpview.CommandInfo)
		return helpview.New(w, h, cmds), nil
	})
	registerView(logsview.ViewName, func(w, h int, payload any) (view.View, tea.Cmd) {
		return logsview.New(w, h), logsview.Load(payload.(string))
	})

	registerView(inspectview.ViewName, func(w, h int, payload any) (view.View, tea.Cmd) {
		data, _ := payload.(map[string]interface{})
		title, _ := data["title"].(string)
		jsonStr, _ := data["json"].(string)

		v := inspectview.New(w, h)
		return v, inspectview.LoadInspectItem(title, jsonStr)
	})

	registerView(nodesview.ViewName, func(w, h int, payload any) (view.View, tea.Cmd) {
		return nodesview.New(w, h), nodesview.LoadNodesCmd()
	})
	registerView(stacksview.ViewName, func(w, h int, payload any) (view.View, tea.Cmd) {
		var nodeID string
		if payload != nil {
			nodeID, _ = payload.(string)
		}
		return stacksview.New(w, h), stacksview.LoadStacks(nodeID)
	})

	registerView(nodeservicesview.ViewName, func(w, h int, payload any) (view.View, tea.Cmd) {
		v := nodeservicesview.New(w, h)

		data, _ := payload.(map[string]interface{})

		var filterType nodeservicesview.FilterType
		var nodeID, stackName string

		if n, ok := data["nodeID"].(string); ok {
			filterType = nodeservicesview.NodeFilter
			nodeID = n
		}
		if s, ok := data["stackName"].(string); ok {
			filterType = nodeservicesview.StackFilter
			stackName = s
		}

		entries, title := nodeservicesview.LoadServicesForView(filterType, nodeID, stackName)

		// Initialize view and return first payload
		return v, func() tea.Msg {
			return nodeservicesview.Msg{
				Title:      title,
				Entries:    entries,
				FilterType: filterType,
				NodeID:     nodeID,
				StackName:  stackName,
			}
		}
	})
}
