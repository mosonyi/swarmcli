package app

import (
	"swarmcli/docker"
	"swarmcli/utils/log"
	configsview "swarmcli/views/configs"
	helpview "swarmcli/views/help"
	inspectview "swarmcli/views/inspect"
	loadingview "swarmcli/views/loading"
	logsview "swarmcli/views/logs"
	nodesview "swarmcli/views/nodes"
	servicesview "swarmcli/views/services"
	stacksview "swarmcli/views/stacks"
	"swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"

	_ "swarmcli/commands" // triggers autoload
	"swarmcli/registry"
)

const (
	appName string = "swarmcli"
)

var (
	version string = "dev"
)

var viewRegistry = map[string]view.Factory{}

func registerView(name string, factory view.Factory) {
	viewRegistry[name] = factory
}

// SetVersion sets the application version (called from main)
func SetVersion(v string) {
	version = v
}

// Init should be called once at the start of the application to register all views.
func Init() {
	swarmlog.Init(appName)
	l := swarmlog.L()
	defer swarmlog.Sync()

	l.Infow("starting Swarm CLI", "version", version)

	l.Infof("Available Commands:")
	for _, cmd := range registry.All() {
		l.Infoln("-", cmd.Name(), "→", cmd.Description())
	}

	registerView(loadingview.ViewName, func(w, h int, payload any) (view.View, tea.Cmd) {
		return loadingview.New(w, h, true, payload), nil
	})
	registerView(helpview.ViewName, func(w, h int, payload any) (view.View, tea.Cmd) {
		cmds, _ := payload.([]helpview.CommandInfo)
		return helpview.New(w, h, cmds), nil
	})
	registerView(logsview.ViewName, func(w, h int, payload any) (view.View, tea.Cmd) {
		service := payload.(docker.ServiceEntry)
		v := logsview.New(w, h, 200, service)
		return v, logsview.StartStreamingCmd(v.StreamCtx, service, 200, v.MaxLines)
	})

	registerView(configsview.ViewName, func(w, h int, payload any) (view.View, tea.Cmd) {
		return configsview.New(w, h), configsview.LoadConfigs()
	})

	registerView(inspectview.ViewName, func(w, h int, payload any) (view.View, tea.Cmd) {
		data, _ := payload.(map[string]interface{})
		title, _ := data["title"].(string)
		jsonStr, _ := data["json"].(string)
		raw := inspectview.ParseFormat(data["format"])

		v := inspectview.New(w, h, raw)
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
		return stacksview.New(w, h), stacksview.LoadStacksCmd(nodeID)
	})

	registerView(servicesview.ViewName, func(w, h int, payload any) (view.View, tea.Cmd) {
		v := servicesview.New(w, h)

		data, _ := payload.(map[string]interface{})

		var filterType servicesview.FilterType
		var nodeID, stackName string

		if n, ok := data["nodeID"].(string); ok {
			filterType = servicesview.NodeFilter
			nodeID = n
		}
		if s, ok := data["stackName"].(string); ok {
			filterType = servicesview.StackFilter
			stackName = s
		}

		entries, title := servicesview.LoadServicesForView(filterType, nodeID, stackName)

		// Initialize view and return first payload
		return v, func() tea.Msg {
			return servicesview.Msg{
				Title:      title,
				Entries:    entries,
				FilterType: filterType,
				NodeID:     nodeID,
				StackName:  stackName,
			}
		}
	})
}
