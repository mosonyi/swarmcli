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

func restartServiceCmd(serviceName string, filterType FilterType, nodeID, stackName string) tea.Cmd {
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

// refreshServicesCmd refreshes the services list (optional)
func refreshServicesCmd(nodeID, stackName string, filterType FilterType) tea.Cmd {
	return func() tea.Msg {
		_, err := docker.RefreshSnapshot()
		if err != nil {
			return nil
		}

		var entries []ServiceEntry
		title := ""

		switch filterType {
		case NodeFilter:
			entries = LoadNodeServices(nodeID)
			title = "Node Services"
		case StackFilter:
			entries = LoadStackServices(stackName)
			title = "Stack Services"
		}

		return Msg{
			Title:      title,
			Entries:    entries,
			FilterType: filterType,
			NodeID:     nodeID,
			StackName:  stackName,
		}
	}
}
