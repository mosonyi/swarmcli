// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package networksview

import (
	"context"
	"fmt"
	"time"

	"github.com/Eldara-Tech/swarmcli/docker"
	inspectview "github.com/Eldara-Tech/swarmcli/views/inspect"
	"github.com/Eldara-Tech/swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/swarm"
)

const PollInterval = 5 * time.Second

type TickMsg time.Time

// PollRetryMsg signals that polling found no changes; the Update handler
// should schedule the next tick.
type PollRetryMsg struct{}

type SpinnerTickMsg time.Time

type NetworksLoadedMsg struct {
	Networks []networkItem
	Err      error
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

func (m *Model) loadNetworksCmd() tea.Cmd {
	networkOps := m.deps.Networks
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		networks, err := networkOps.ListNetworks(ctx)
		if err != nil {
			return NetworksLoadedMsg{Err: fmt.Errorf("failed to list networks: %w", err)}
		}

		items := make([]networkItem, 0, len(networks))
		for _, net := range networks {
			items = append(items, networkItem{
				Name:      net.Name,
				ID:        net.ID,
				Driver:    net.Driver,
				Scope:     net.Scope,
				CreatedAt: net.Created,
				Ingress:   net.Ingress,
				Used:      false,
				UsedKnown: false,
			})
		}

		return NetworksLoadedMsg{
			Networks: items,
			Err:      nil,
		}
	}
}

func (m *Model) inspectNetworkCmd(networkID string) tea.Cmd {
	networkOps := m.deps.Networks
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		title := fmt.Sprintf("Network: %s", networkID)

		net, err := networkOps.InspectNetwork(ctx, networkID)
		if err != nil {
			return view.NavigateToMsg{
				ViewName: inspectview.ViewName,
				Payload: map[string]any{
					"title":  title,
					"json":   fmt.Sprintf("# Error inspecting network:\n# %v", err),
					"format": inspectview.FormatRaw,
				},
			}
		}

		services, svcErr := networkOps.ListServicesUsingNetwork(ctx, networkID, net.Name)
		if svcErr != nil {
			l().Warnf("Failed to list services for network %s: %v", net.Name, svcErr)
			services = []string{}
		}

		summary := network.Summary{
			Name:       net.Name,
			ID:         net.ID,
			Created:    net.Created,
			Scope:      net.Scope,
			Driver:     net.Driver,
			EnableIPv6: net.EnableIPv6,
			IPAM:       net.IPAM,
			Internal:   net.Internal,
			Attachable: net.Attachable,
			Ingress:    net.Ingress,
			ConfigFrom: network.ConfigReference{Network: net.ConfigFrom.Network},
			ConfigOnly: net.ConfigOnly,
			Containers: net.Containers,
			Options:    net.Options,
			Labels:     net.Labels,
		}

		nw := docker.NetworkWithUsage{Network: summary, Services: services}
		content, jsonErr := nw.JSON()
		if jsonErr != nil {
			return view.NavigateToMsg{
				ViewName: inspectview.ViewName,
				Payload: map[string]any{
					"title":  fmt.Sprintf("Network: %s", net.Name),
					"json":   fmt.Sprintf("# Error marshalling network:\n# %v", jsonErr),
					"format": inspectview.FormatRaw,
				},
			}
		}

		return view.NavigateToMsg{
			ViewName: inspectview.ViewName,
			Payload: map[string]any{
				"title": fmt.Sprintf("Network: %s", net.Name),
				"json":  string(content),
			},
		}
	}
}

func (m *Model) deleteNetworkCmd(networkID string) tea.Cmd {
	networkOps := m.deps.Networks
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := networkOps.RemoveNetwork(ctx, networkID)
		if err != nil {
			return NetworkDeletedMsg{Err: fmt.Errorf("failed to delete network: %w", err)}
		}
		return NetworkDeletedMsg{Err: nil}
	}
}

func (m *Model) createNetworkCmd(name, driver string, attachable, internal bool, ipv4Subnet, ipv4Gateway string, enableIPv6 bool, ipv6Subnet, ipv6Gateway string) tea.Cmd {
	networkOps := m.deps.Networks
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		opts := network.CreateOptions{
			Driver:     driver,
			Attachable: attachable,
			Internal:   internal,
		}
		if enableIPv6 {
			v := true
			opts.EnableIPv6 = &v
		}

		configs := make([]network.IPAMConfig, 0, 2)
		if ipv4Subnet != "" || ipv4Gateway != "" {
			configs = append(configs, network.IPAMConfig{Subnet: ipv4Subnet, Gateway: ipv4Gateway})
		}
		if enableIPv6 && (ipv6Subnet != "" || ipv6Gateway != "") {
			configs = append(configs, network.IPAMConfig{Subnet: ipv6Subnet, Gateway: ipv6Gateway})
		}
		if len(configs) > 0 {
			opts.IPAM = &network.IPAM{Driver: "default", Config: configs}
		}

		id, warnings, err := networkOps.CreateNetwork(ctx, name, opts)
		if err != nil {
			return NetworkCreatedMsg{Name: name, Err: fmt.Errorf("failed to create network: %w", err)}
		}
		return NetworkCreatedMsg{Name: name, ID: id, Warnings: warnings, Err: nil}
	}
}

func (m *Model) pruneNetworksCmd() tea.Cmd {
	networkOps := m.deps.Networks
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		before, err := networkOps.ListNetworks(ctx)
		if err != nil {
			return NetworksPrunedMsg{Err: fmt.Errorf("failed to list networks before prune: %w", err)}
		}
		idToName := make(map[string]string, len(before))
		for _, n := range before {
			if n.ID != "" {
				idToName[n.ID] = n.Name
			}
		}

		report, err := networkOps.PruneNetworks(ctx)
		if err != nil {
			return NetworksPrunedMsg{Err: fmt.Errorf("failed to prune networks: %w", err)}
		}

		deleted := make([]string, 0, len(report.NetworksDeleted))
		for _, id := range report.NetworksDeleted {
			name := idToName[id]
			if name == "" {
				name = id
			}
			deleted = append(deleted, name)
		}

		return NetworksPrunedMsg{Deleted: deleted, Err: nil}
	}
}

func (m *Model) loadUsedByCmd(networkID, networkName string) tea.Cmd {
	clientOps := m.deps.Client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		client, err := clientOps.GetClient()
		if err != nil {
			return UsedByLoadedMsg{Err: fmt.Errorf("failed to get docker client: %w", err)}
		}

		allServices, err := client.ServiceList(ctx, swarm.ServiceListOptions{})
		if err != nil {
			return UsedByLoadedMsg{Err: fmt.Errorf("failed to list services: %w", err)}
		}

		items := make([]usedByItem, 0)
		for _, svc := range allServices {
			used := false

			if !used && networkName == "ingress" && svc.Spec.EndpointSpec != nil {
				for _, port := range svc.Spec.EndpointSpec.Ports {
					if port.PublishMode == "ingress" {
						used = true
						break
					}
				}
			}

			for _, net := range svc.Spec.TaskTemplate.Networks {
				if net.Target == networkID || (networkName != "" && net.Target == networkName) {
					used = true
					break
				}
			}
			if !used {
				continue
			}

			stackName := "N/A"
			if stack, ok := svc.Spec.Labels["com.docker.stack.namespace"]; ok {
				stackName = stack
			}

			items = append(items, usedByItem{
				StackName:   stackName,
				ServiceName: svc.Spec.Name,
			})
		}

		return UsedByLoadedMsg{
			Services: items,
			Err:      nil,
		}
	}
}

func (m *Model) computeNetworkUsedCmd(networks []networkItem) tea.Cmd {
	clientOps := m.deps.Client
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

		client, err := clientOps.GetClient()
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
