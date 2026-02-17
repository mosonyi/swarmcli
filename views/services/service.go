// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package servicesview

import (
	"swarmcli/core/primitives/hash"
	"swarmcli/docker"
	"time"

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
		entries = serviceOps.LoadStackServices("")
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
			return tickCmd()
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
		// Schedule next poll in 5 seconds
		return tea.Tick(PollInterval, func(t time.Time) tea.Msg {
			return TickMsg(t)
		})()
	}
}

// refreshExpandedTasksCmd refreshes tasks for all expanded services
func (m *Model) refreshExpandedTasksCmd(expandedServices map[string]bool) tea.Cmd {
	if len(expandedServices) == 0 {
		return nil
	}

	taskOps := m.deps.Tasks

	// Create a batch of commands to fetch tasks for each expanded service
	var cmds []tea.Cmd
	for serviceID, expanded := range expandedServices {
		if expanded {
			// Capture serviceID in closure
			sid := serviceID
			cmds = append(cmds, func() tea.Msg {
				tasks, err := taskOps.GetTasksForService(sid)
				if err != nil {
					l().Errorf("Failed to refresh tasks for service %s: %v", sid, err)
					tasks = []docker.TaskEntry{}
				}
				return TasksLoadedMsg{
					ServiceID: sid,
					Tasks:     tasks,
				}
			})
		}
	}

	if len(cmds) == 0 {
		return nil
	}

	return tea.Batch(cmds...)
}
