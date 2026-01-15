package networksview

import (
	"context"
	"fmt"
	"swarmcli/docker"
	"time"

	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/swarm"
)

type networkItem struct {
	Name      string
	ID        string
	Driver    string
	Scope     string
	CreatedAt time.Time
	Ingress   bool // true if this is the swarm routing-mesh ingress network
	Used      bool // true if used by any service
	UsedKnown bool // true if Used has been computed (false => loading/unknown)
}

func (i networkItem) FilterValue() string { return i.Name }
func (i networkItem) Title() string       { return i.Name }
func (i networkItem) Description() string {
	createdStr := "N/A"
	if !i.CreatedAt.IsZero() {
		createdStr = i.CreatedAt.Format("2006-01-02 15:04:05")
	}
	return fmt.Sprintf("ID: %s        Driver: %s        Scope: %s        Created: %s",
		i.ID, i.Driver, i.Scope, createdStr)
}

type usedByItem struct {
	StackName   string
	ServiceName string
}

func (i usedByItem) FilterValue() string { return i.StackName + " " + i.ServiceName }
func (i usedByItem) Title() string       { return fmt.Sprintf("%-24s %-24s", i.StackName, i.ServiceName) }
func (i usedByItem) Description() string { return "Service: " + i.ServiceName }

type networkWithUsage struct {
	Network  network.Summary
	Services []string
}

func (nw *networkWithUsage) PrettyJSON() ([]byte, error) {
	dockerNW := docker.NetworkWithUsage{
		Network:  nw.Network,
		Services: nw.Services,
	}
	return dockerNW.PrettyJSON()
}

// fetchNetworks retrieves all networks and their usage information
func fetchNetworks() ([]networkItem, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	networks, err := docker.ListNetworks(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list networks: %w", err)
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

	return items, nil
}

// fetchNetworkWithUsage retrieves detailed information about a network
func fetchNetworkWithUsage(networkID string) (*networkWithUsage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	net, err := docker.InspectNetwork(ctx, networkID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect network: %w", err)
	}

	services, err := docker.ListServicesUsingNetwork(ctx, networkID, net.Name)
	if err != nil {
		l().Warnf("Failed to list services for network %s: %v", net.Name, err)
		services = []string{}
	}

	// Convert Inspect result to Summary for consistency
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

	return &networkWithUsage{
		Network:  summary,
		Services: services,
	}, nil
}

// fetchUsedBy retrieves the services using a network
func fetchUsedBy(networkID, networkName string) ([]usedByItem, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := docker.GetClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get docker client: %w", err)
	}

	allServices, err := client.ServiceList(ctx, swarm.ServiceListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list services: %w", err)
	}

	items := make([]usedByItem, 0)
	for _, svc := range allServices {
		used := false

		// Ingress network dependency can be implicit via published ports.
		// Those services may not list ingress under TaskTemplate.Networks.
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

	return items, nil
}

// deleteNetwork removes a network
func deleteNetwork(networkID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := docker.RemoveNetwork(ctx, networkID)
	if err != nil {
		return fmt.Errorf("failed to delete network: %w", err)
	}

	return nil
}

// pruneNetworks removes all unused networks
func pruneNetworks() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Build an ID->Name map from the current network list so we can show
	// human-friendly names for deleted networks.
	before, err := docker.ListNetworks(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list networks before prune: %w", err)
	}
	idToName := make(map[string]string, len(before))
	for _, n := range before {
		if n.ID != "" {
			idToName[n.ID] = n.Name
		}
	}

	report, err := docker.PruneNetworks(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to prune networks: %w", err)
	}

	deleted := make([]string, 0, len(report.NetworksDeleted))
	for _, id := range report.NetworksDeleted {
		name := idToName[id]
		if name == "" {
			name = id
		}
		deleted = append(deleted, name)
	}

	return deleted, nil
}

func createNetwork(name, driver string, attachable, internal bool, ipv4Subnet, ipv4Gateway string, enableIPv6 bool, ipv6Subnet, ipv6Gateway string) (string, []string, error) {
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

	id, warnings, err := docker.CreateNetwork(ctx, name, opts)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create network: %w", err)
	}

	return id, warnings, nil
}
