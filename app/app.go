package app

import (
	"swarmcli/docker"
	swarmlog "swarmcli/utils/log"
	configsview "swarmcli/views/configs"
	contextsview "swarmcli/views/contexts"
	helpview "swarmcli/views/help"
	inspectview "swarmcli/views/inspect"
	loadingview "swarmcli/views/loading"
	logsview "swarmcli/views/logs"
	networksview "swarmcli/views/networks"
	nodesview "swarmcli/views/nodes"
	revealsecretview "swarmcli/views/revealsecret"
	secretsview "swarmcli/views/secrets"
	servicesview "swarmcli/views/services"
	stacksview "swarmcli/views/stacks"
	tasksview "swarmcli/views/tasks"
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
		// Check if detailed help with categories
		if categories, ok := payload.([]helpview.HelpCategory); ok {
			return helpview.NewDetailed(w, h, categories), nil
		}
		// Legacy command list help
		cmds, _ := payload.([]helpview.CommandInfo)
		return helpview.New(w, h, cmds), nil
	})
	registerView(logsview.ViewName, func(w, h int, payload any) (view.View, tea.Cmd) {
		service := payload.(docker.ServiceEntry)
		v := logsview.New(w, h, 10000, service)
		return v, logsview.StartStreamingCmd(v.StreamCtx, service, 200, v.MaxLines)
	})

	registerView(configsview.ViewName, func(w, h int, payload any) (view.View, tea.Cmd) {
		model := configsview.New(w, h)
		return model, model.Init()
	})

	registerView(secretsview.ViewName, func(w, h int, payload any) (view.View, tea.Cmd) {
		model := secretsview.New(w, h)
		return model, model.Init()
	})

	registerView(inspectview.ViewName, func(w, h int, payload any) (view.View, tea.Cmd) {
		data, _ := payload.(map[string]interface{})
		title, _ := data["title"].(string)
		jsonStr, _ := data["json"].(string)
		raw := inspectview.ParseFormat(data["format"])

		v := inspectview.New(w, h, raw)
		return v, inspectview.LoadInspectItem(title, jsonStr)
	})

	registerView(revealsecretview.ViewName, func(w, h int, payload any) (view.View, tea.Cmd) {
		data, _ := payload.(map[string]interface{})
		secretName, _ := data["secretName"].(string)

		v := revealsecretview.New(w, h)
		v.SetSecretName(secretName)
		return v, tea.Batch(v.Init(), revealsecretview.LoadSecret(secretName))
	})

	registerView(nodesview.ViewName, func(w, h int, payload any) (view.View, tea.Cmd) {
		model := nodesview.New(w, h)
		return model, tea.Batch(model.Init(), nodesview.LoadNodesCmd())
	})

	registerView(contextsview.ViewName, func(w, h int, payload any) (view.View, tea.Cmd) {
		model := contextsview.New()
		model.Visible = true
		model.SetSize(w, h)
		model.SetLoading(true)
		return model, tea.Batch(
			func() tea.Msg { return contextsview.LoadContextsCmd() },
			contextsview.StartTickerCmd(),
		)
	})

	registerView(stacksview.ViewName, func(w, h int, payload any) (view.View, tea.Cmd) {
		var nodeID string
		if payload != nil {
			nodeID, _ = payload.(string)
		}
		model := stacksview.New(w, h)
		model.Visible = true
		return model, tea.Batch(model.Init(), stacksview.LoadStacksCmd(nodeID))
	})

	registerView(servicesview.ViewName, func(w, h int, payload any) (view.View, tea.Cmd) {
		v := servicesview.New(w, h)

		data, _ := payload.(map[string]interface{})

		filterType := servicesview.AllFilter
		var nodeID, stackName string
		var serviceName string
		var selectServiceName string
		var noStack bool

		if n, ok := data["nodeID"].(string); ok {
			filterType = servicesview.NodeFilter
			nodeID = n
		}
		if s, ok := data["stackName"].(string); ok {
			filterType = servicesview.StackFilter
			stackName = s
		}
		if b, ok := data["noStack"].(bool); ok {
			noStack = b
		}
		if noStack {
			filterType = servicesview.NoStackFilter
			stackName = ""
			nodeID = ""
		}
		if s, ok := data["serviceName"].(string); ok {
			serviceName = s
		}
		if s, ok := data["selectServiceName"].(string); ok {
			selectServiceName = s
		}
		if serviceName != "" {
			v.List.Query = serviceName
		}
		if selectServiceName != "" {
			v.SetPendingSelectServiceName(selectServiceName)
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

	registerView(tasksview.ViewName, func(w, h int, payload any) (view.View, tea.Cmd) {
		stackName, _ := payload.(string)
		model := tasksview.New(w, h, stackName)
		return model, model.OnEnter()
	})

	registerView(networksview.ViewName, func(w, h int, payload any) (view.View, tea.Cmd) {
		model := networksview.New(w, h)
		return model, model.Init()
	})
}
