package nodeservicesview

import (
	"context"
	"swarmcli/docker"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Message sent when a service has finished restarting
type serviceRestartedMsg struct {
	ServiceName string
	Err         error
}

func restartServiceCmd(serviceName string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		err := docker.RestartServiceAndWait(ctx, serviceName)

		return serviceRestartedMsg{
			ServiceName: serviceName,
			Err:         err,
		}
	}
}

func refreshServicesCmd(nodeID, stackName string, filterType FilterType) tea.Cmd {
	return func() tea.Msg {
		_, err := docker.RefreshSnapshot()
		if err != nil {
			return nil
		}

		entries, title := LoadServicesForView(filterType, nodeID, stackName)
		return Msg{
			Title:      title,
			Entries:    entries,
			FilterType: filterType,
			NodeID:     nodeID,
			StackName:  stackName,
		}
	}
}

func LoadServicesForView(filterType FilterType, nodeID, stackName string) (entries []ServiceEntry, title string) {
	switch filterType {
	case NodeFilter:
		entries = LoadNodeServices(nodeID)
		title = "Services on Node: " + nodeID
	case StackFilter:
		entries = LoadStackServices(stackName)
		title = "Services in Stack: " + stackName
	default: // All services
		entries = LoadStackServices("")
		title = "All Services"
	}
	return
}
