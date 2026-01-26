// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package networksview

import (
	"context"
	"time"

	"swarmcli/docker"

	"github.com/docker/docker/api/types/swarm"

	tea "github.com/charmbracelet/bubbletea"
)

const PollInterval = 5 * time.Second

type TickMsg time.Time
type SpinnerTickMsg time.Time

type NetworksLoadedMsg struct {
	Networks []networkItem
	Err      error
}

type NetworkInspectMsg struct {
	NetworkWithUsage *networkWithUsage
	Err              error
}

type NetworkDeletedMsg struct {
	Err error
}

type NetworkCreatedMsg struct {
	Name     string
	ID       string
	Warnings []string
	Err      error
}

type NetworksPrunedMsg struct {
	Deleted []string
	Err     error
}

type UsedByLoadedMsg struct {
	Services []usedByItem
	Err      error
}

// usedStatusUpdatedMsg carries a map of network ID -> used boolean
type usedStatusUpdatedMsg map[string]bool

// ViewStackMsg is sent when user wants to go to the stacks/services view
type ViewStackMsg struct {
	StackName string
}

func loadNetworksCmd() tea.Cmd {
	return func() tea.Msg {
		networks, err := fetchNetworks()
		return NetworksLoadedMsg{
			Networks: networks,
			Err:      err,
		}
	}
}

func inspectNetworkCmd(networkID string) tea.Cmd {
	return func() tea.Msg {
		nw, err := fetchNetworkWithUsage(networkID)
		return NetworkInspectMsg{
			NetworkWithUsage: nw,
			Err:              err,
		}
	}
}

func deleteNetworkCmd(networkID string) tea.Cmd {
	return func() tea.Msg {
		err := deleteNetwork(networkID)
		return NetworkDeletedMsg{Err: err}
	}
}

func createNetworkCmd(name, driver string, attachable, internal bool, ipv4Subnet, ipv4Gateway string, enableIPv6 bool, ipv6Subnet, ipv6Gateway string) tea.Cmd {
	return func() tea.Msg {
		id, warnings, err := createNetwork(name, driver, attachable, internal, ipv4Subnet, ipv4Gateway, enableIPv6, ipv6Subnet, ipv6Gateway)
		return NetworkCreatedMsg{Name: name, ID: id, Warnings: warnings, Err: err}
	}
}

func pruneNetworksCmd() tea.Cmd {
	return func() tea.Msg {
		deleted, err := pruneNetworks()
		return NetworksPrunedMsg{Deleted: deleted, Err: err}
	}
}

func loadUsedByCmd(networkID, networkName string) tea.Cmd {
	return func() tea.Msg {
		services, err := fetchUsedBy(networkID, networkName)
		return UsedByLoadedMsg{
			Services: services,
			Err:      err,
		}
	}
}

func computeNetworkUsedCmd(networks []networkItem) tea.Cmd {
	return func() tea.Msg {
		used := make(map[string]bool, len(networks))
		keyToID := make(map[string]string, len(networks)*2)
		ingressIDs := make([]string, 0, 1)
		for _, n := range networks {
			used[n.ID] = false
			if n.ID != "" {
				keyToID[n.ID] = n.ID
			}
			if n.Name != "" {
				keyToID[n.Name] = n.ID
			}
			if n.Ingress || n.Name == "ingress" {
				ingressIDs = append(ingressIDs, n.ID)
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		client, err := docker.GetClient()
		if err != nil {
			return usedStatusUpdatedMsg(used)
		}

		services, err := client.ServiceList(ctx, swarm.ServiceListOptions{})
		if err != nil {
			return usedStatusUpdatedMsg(used)
		}

		for _, svc := range services {
			// In Swarm, a service can depend on the ingress network implicitly when it
			// publishes ports in ingress mode (routing mesh). Those dependencies are
			// not always present in TaskTemplate.Networks.
			if svc.Spec.EndpointSpec != nil {
				for _, port := range svc.Spec.EndpointSpec.Ports {
					if port.PublishMode == "ingress" {
						for _, id := range ingressIDs {
							if id != "" {
								used[id] = true
							}
						}
						break
					}
				}
			}

			for _, net := range svc.Spec.TaskTemplate.Networks {
				if net.Target != "" {
					if id, ok := keyToID[net.Target]; ok {
						used[id] = true
					}
				}
			}
		}

		return usedStatusUpdatedMsg(used)
	}
}
