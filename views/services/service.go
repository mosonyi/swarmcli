// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package servicesview

import (
	"swarmcli/core/primitives/hash"
	"swarmcli/docker"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) refreshServicesCmd(nodeID, stackName string, filterType FilterType) tea.Cmd {
	snapshotOps := m.deps.Snapshot
	serviceOps := m.deps.Services
	return func() tea.Msg {
		// Explicit user-initiated refresh: perform synchronous refresh but keep it defensive.
		_, err := snapshotOps.RefreshSnapshot()
		if err != nil {
			// If refresh fails, fall back to cached snapshot and continue
			l().Errorf("refreshServicesCmd: RefreshSnapshot failed: %v", err)
		}

		entries, title := loadServicesForViewWith(serviceOps, filterType, nodeID, stackName)
		return Msg{
			Title:      title,
			Entries:    entries,
			FilterType: filterType,
			NodeID:     nodeID,
			StackName:  stackName,
		}
	}
}

func (m *Model) loadServicesForView(filterType FilterType, nodeID, stackName string) (entries []docker.ServiceEntry, title string) {
	return loadServicesForViewWith(m.deps.Services, filterType, nodeID, stackName)
}

func loadServicesForViewWith(serviceOps docker.ServiceOps, filterType FilterType, nodeID, stackName string) (entries []docker.ServiceEntry, title string) {
	switch filterType {
	case NodeFilter:
		entries = serviceOps.LoadNodeServices(nodeID)
		title = "Services on Node: " + nodeID
	case StackFilter:
		entries = serviceOps.LoadStackServices(stackName)
		title = "Services in Stack: " + stackName
	case NoStackFilter:
		// docker marks services without a stack namespace as stack "-".
		entries = serviceOps.LoadStackServices("-")
		title = "Services (no stack)"
	default: // All services
		entries = serviceOps.LoadAllServices()
		title = "All Services"
	}
	return
}

// checkServicesCmd checks if services have changed and returns update message if so
func (m *Model) checkServicesCmd(lastHash uint64, filterType FilterType, nodeID, stackName string) tea.Cmd {
	snapshotOps := m.deps.Snapshot
	serviceOps := m.deps.Services
	return func() tea.Msg {
		l().Info("checkServicesCmd: Polling for service changes")

		// Do not block the UI waiting for network calls. Trigger an async refresh if needed
		// and use the cached snapshot for quick checks.
		snapshotOps.TriggerRefreshIfNeeded()

		entries, title := loadServicesForViewWith(serviceOps, filterType, nodeID, stackName)
		newHash, err := hash.Compute(entries)
		if err != nil {
			l().Errorf("checkServicesCmd: Hash computation failed: %v", err)
			return PollRetryMsg{}
		}

		l().Infof("checkServicesCmd: lastHash=%s, newHash=%s, serviceCount=%d",
			hash.Fmt(lastHash), hash.Fmt(newHash), len(entries))

		// Only return update message if something changed
		if newHash != lastHash {
			l().Info("checkServicesCmd: Change detected! Refreshing service list")
			return Msg{
				Title:      title,
				Entries:    entries,
				FilterType: filterType,
				NodeID:     nodeID,
				StackName:  stackName,
			}
		}

		l().Info("checkServicesCmd: No changes detected, scheduling next poll")
		return PollRetryMsg{}
	}
}

// refreshExpandedTasksCmd refreshes tasks for all expanded services.
// It ensures the snapshot is fresh before reading tasks so that expanded
// task rows update on the fly instead of showing stale cached data.
func (m *Model) refreshExpandedTasksCmd(expandedServices map[string]bool) tea.Cmd {
	if len(expandedServices) == 0 {
		return nil
	}

	snapshotOps := m.deps.Snapshot
	taskOps := m.deps.Tasks

	// Copy expanded service IDs to avoid closing over the mutable map.
	var serviceIDs []string
	for sid, expanded := range expandedServices {
		if expanded {
			serviceIDs = append(serviceIDs, sid)
		}
	}
	if len(serviceIDs) == 0 {
		return nil
	}

	return func() tea.Msg {
		// Ensure the snapshot is fresh before reading tasks.
		if _, err := snapshotOps.GetOrRefreshSnapshot(); err != nil {
			l().Errorf("refreshExpandedTasksCmd: GetOrRefreshSnapshot failed: %v", err)
		}

		result := make(map[string][]docker.TaskEntry, len(serviceIDs))
		for _, sid := range serviceIDs {
			tasks, err := taskOps.GetTasksForService(sid)
			if err != nil {
				l().Errorf("Failed to refresh tasks for service %s: %v", sid, err)
				tasks = []docker.TaskEntry{}
			}
			result[sid] = tasks
		}
		return AllTasksLoadedMsg{Tasks: result}
	}
}
