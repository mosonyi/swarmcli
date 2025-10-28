package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
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

// GetSwarmCPUUsage returns the total CPU usage percentage of all containers.
func GetSwarmCPUUsage() (string, error) {
	c, err := GetClient()
	if err != nil {
		return "0%", err
	}

	containers, err := c.ContainerList(context.Background(), container.ListOptions{All: true})
	if err != nil {
		return "0%", err
	}

	var totalCPU float64
	for _, cont := range containers {
		stats, err := c.ContainerStats(context.Background(), cont.ID, false)
		if err != nil {
			continue
		}
		var s container.Stats
		if err := json.NewDecoder(stats.Body).Decode(&s); err != nil {
			stats.Body.Close()
			continue
		}
		stats.Body.Close()

		// Docker calculates CPU % as (cpu_delta / system_cpu_delta) * online_cpus * 100
		cpuDelta := float64(s.CPUStats.CPUUsage.TotalUsage - s.PreCPUStats.CPUUsage.TotalUsage)
		systemDelta := float64(s.CPUStats.SystemUsage - s.PreCPUStats.SystemUsage)
		onlineCPUs := float64(s.CPUStats.OnlineCPUs)
		if onlineCPUs == 0 {
			onlineCPUs = float64(len(s.CPUStats.CPUUsage.PercpuUsage))
		}
		if systemDelta > 0 && onlineCPUs > 0 {
			totalCPU += (cpuDelta / systemDelta) * onlineCPUs * 100
		}
	}

	return fmt.Sprintf("%.1f%%", totalCPU), nil
}

// GetSwarmMemUsage returns total memory usage percentage across all containers.
func GetSwarmMemUsage() (string, error) {
	c, err := GetClient()
	if err != nil {
		return "0%", err
	}

	containers, err := c.ContainerList(context.Background(), container.ListOptions{All: true})
	if err != nil {
		return "0%", err
	}

	var totalMemPercent float64
	for _, cont := range containers {
		stats, err := c.ContainerStats(context.Background(), cont.ID, false)
		if err != nil {
			continue
		}
		var s container.Stats
		if err := json.NewDecoder(stats.Body).Decode(&s); err != nil {
			stats.Body.Close()
			continue
		}
		stats.Body.Close()

		if s.MemoryStats.Limit > 0 {
			totalMemPercent += float64(s.MemoryStats.Usage) / float64(s.MemoryStats.Limit) * 100
		}
	}

	return fmt.Sprintf("%.1f%%", totalMemPercent), nil
}

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
