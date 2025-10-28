package docker

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	swarmTypes "github.com/docker/docker/api/types/swarm"
)

// ---------- Swarm Node / Service Info ----------

type SwarmNode struct {
	ID            string
	Hostname      string
	Status        string
	Availability  string
	ManagerStatus string
}

func (s SwarmNode) String() string {
	return strings.Join(StructFieldsAsStringArray(s), " ")
}

// ListSwarmNodes returns all swarm nodes.
func ListSwarmNodes() ([]SwarmNode, error) {
	c, err := GetClient()
	if err != nil {
		return nil, err
	}

	nodes, err := c.NodeList(context.Background(), types.NodeListOptions{})
	if err != nil {
		return nil, err
	}

	res := make([]SwarmNode, 0, len(nodes))
	for _, n := range nodes {
		managerStatus := ""
		if n.ManagerStatus != nil {
			managerStatus = string(n.ManagerStatus.Reachability)
		}
		res = append(res, SwarmNode{
			ID:            n.ID,
			Hostname:      n.Description.Hostname,
			Status:        string(n.Status.State),
			Availability:  string(n.Spec.Availability),
			ManagerStatus: managerStatus,
		})
	}
	return res, nil
}

// ---------- Container / Service Counts ----------

func GetContainerCount() (int, error) {
	c, err := GetClient()
	if err != nil {
		return 0, err
	}

	containers, err := c.ContainerList(context.Background(), container.ListOptions{All: true})
	if err != nil {
		return 0, err
	}
	return len(containers), nil
}

func GetServiceCount() (int, error) {
	c, err := GetClient()
	if err != nil {
		return 0, err
	}

	services, err := c.ServiceList(context.Background(), types.ServiceListOptions{})
	if err != nil {
		return 0, err
	}
	return len(services), nil
}

// ---------- Swarm Resource Usage ----------

func GetSwarmCPUUsage() (string, error) {
	c, err := GetClient()
	if err != nil {
		return "0%", err
	}

	stats, err := c.ContainerList(context.Background(), container.ListOptions{})
	if err != nil {
		return "0%", err
	}

	var cpuTotal float64
	for _, cont := range stats {
		// For now we cannot get precise CPU usage via SDK without streaming stats
		// So we leave as placeholder
		_ = cont
	}

	return fmt.Sprintf("%.1f%%", cpuTotal), nil
}

func GetSwarmMemUsage() (string, error) {
	c, err := GetClient()
	if err != nil {
		return "0%", err
	}

	stats, err := c.ContainerList(context.Background(), container.ListOptions{})
	if err != nil {
		return "0%", err
	}

	var memTotal float64
	for _, cont := range stats {
		_ = cont
	}

	return fmt.Sprintf("%.1f%%", memTotal), nil
}

// ---------- Docker Version ----------

func GetDockerVersion() (string, error) {
	c, err := GetClient()
	if err != nil {
		return "unknown", err
	}

	info, err := c.ServerVersion(context.Background())
	if err != nil {
		return "unknown", err
	}
	return info.Version, nil
}

// ---------- Docker Config Commands ----------

func ListConfigs() ([]swarmTypes.Config, error) {
	c, err := GetClient()
	if err != nil {
		return nil, err
	}

	return c.ConfigList(context.Background(), types.ConfigListOptions{})
}

func InspectConfig(configID string) (*swarmTypes.Config, error) {
	c, err := GetClient()
	if err != nil {
		return nil, err
	}

	cfg, _, err := c.ConfigInspectWithRaw(context.Background(), configID)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func RemoveConfig(configID string) error {
	c, err := GetClient()
	if err != nil {
		return err
	}
	return c.ConfigRemove(context.Background(), configID)
}
