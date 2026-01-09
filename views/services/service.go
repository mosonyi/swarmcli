package servicesview

import (
	"swarmcli/core/primitives/hash"
	"swarmcli/docker"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)


func refreshServicesCmd(nodeID, stackName string, filterType FilterType) tea.Cmd {
	return func() tea.Msg {
		// Explicit user-initiated refresh: perform synchronous refresh but keep it defensive.
		_, err := docker.RefreshSnapshot()
		if err != nil {
			// If refresh fails, fall back to cached snapshot and continue
			l().Errorf("refreshServicesCmd: RefreshSnapshot failed: %v", err)
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

func LoadServicesForView(filterType FilterType, nodeID, stackName string) (entries []docker.ServiceEntry, title string) {
	switch filterType {
	case NodeFilter:
		entries = docker.LoadNodeServices(nodeID)
		title = "Services on Node: " + nodeID
	case StackFilter:
		entries = docker.LoadStackServices(stackName)
		title = "Services in Stack: " + stackName
	default: // All services
		entries = docker.LoadStackServices("")
		title = "All Services"
	}
	return
}

// CheckServicesCmd checks if services have changed and returns update message if so
func CheckServicesCmd(lastHash uint64, filterType FilterType, nodeID, stackName string) tea.Cmd {
	return func() tea.Msg {
		l().Info("CheckServicesCmd: Polling for service changes")

		// Do not block the UI waiting for network calls. Trigger an async refresh if needed
		// and use the cached snapshot for quick checks.
		docker.TriggerRefreshIfNeeded()

		entries, title := LoadServicesForView(filterType, nodeID, stackName)
		newHash, err := hash.Compute(entries)
		if err != nil {
			l().Errorf("CheckServicesCmd: Hash computation failed: %v", err)
			return tickCmd()
		}

		l().Infof("CheckServicesCmd: lastHash=%s, newHash=%s, serviceCount=%d",
			hash.Fmt(lastHash), hash.Fmt(newHash), len(entries))

		// Only return update message if something changed
		if newHash != lastHash {
			l().Info("CheckServicesCmd: Change detected! Refreshing service list")
			return Msg{
				Title:      title,
				Entries:    entries,
				FilterType: filterType,
				NodeID:     nodeID,
				StackName:  stackName,
			}
		}

		l().Info("CheckServicesCmd: No changes detected, scheduling next poll")
		// Schedule next poll in 5 seconds
		return tea.Tick(PollInterval, func(t time.Time) tea.Msg {
			return TickMsg(t)
		})()
	}
}
