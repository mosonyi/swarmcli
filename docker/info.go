package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/swarm"
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

	services, err := c.ServiceList(context.Background(), swarm.ServiceListOptions{})
	if err != nil {
		return 0, err
	}
	return len(services), nil
}

// ---------- Swarm Resource Usage ----------

// GetSwarmCPUUsage returns actual CPU usage across running containers with sampling.
func GetSwarmCPUUsage() (string, error) {
	c, err := GetClient()
	if err != nil {
		l().Infof("GetSwarmCPUUsage: GetClient error: %v", err)
		return "N/A", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Get running containers
	containers, err := c.ContainerList(ctx, container.ListOptions{})
	if err != nil {
		l().Infof("GetSwarmCPUUsage: ContainerList error: %v", err)
		return "N/A", err
	}

	l().Infof("GetSwarmCPUUsage: Found %d containers", len(containers))

	if len(containers) == 0 {
		return "0.0%", nil
	}

	// Calculate total CPU usage from containers (sample up to 3)
	var totalCPUPercent float64
	statsCount := 0
	maxContainers := len(containers)
	if maxContainers > 3 {
		maxContainers = 3
	}
	
	for i := 0; i < maxContainers; i++ {
		cont := containers[i]
		
		stats, err := c.ContainerStats(context.Background(), cont.ID, false)
		if err != nil {
			l().Infof("GetSwarmCPUUsage: ContainerStats error for %s: %v", cont.ID[:12], err)
			continue
		}
		
		var s container.StatsResponse
		decodeErr := json.NewDecoder(stats.Body).Decode(&s)
		stats.Body.Close()
		
		if decodeErr != nil {
			l().Infof("GetSwarmCPUUsage: Decode error for %s: %v", cont.ID[:12], decodeErr)
			continue
		}

		// Calculate CPU percentage for this container
		cpuDelta := float64(s.CPUStats.CPUUsage.TotalUsage - s.PreCPUStats.CPUUsage.TotalUsage)
		systemDelta := float64(s.CPUStats.SystemUsage - s.PreCPUStats.SystemUsage)
		onlineCPUs := float64(s.CPUStats.OnlineCPUs)
		
		if onlineCPUs == 0 {
			onlineCPUs = float64(len(s.CPUStats.CPUUsage.PercpuUsage))
		}
		
		if systemDelta > 0 && onlineCPUs > 0 {
			cpuPercent := (cpuDelta / systemDelta) * onlineCPUs * 100.0
			totalCPUPercent += cpuPercent
			statsCount++
			l().Infof("GetSwarmCPUUsage: Container %s CPU: %.1f%%", cont.ID[:12], cpuPercent)
		}
	}

	// Scale up if we sampled
	if statsCount > 0 && len(containers) > maxContainers {
		totalCPUPercent = totalCPUPercent * float64(len(containers)) / float64(maxContainers)
	}

	result := fmt.Sprintf("%.1f%%", totalCPUPercent)
	l().Infof("GetSwarmCPUUsage: Final result: %s (from %d containers)", result, statsCount)
	return result, nil
}

// GetSwarmMemUsage returns actual memory usage across running containers.
func GetSwarmMemUsage() (string, error) {
	c, err := GetClient()
	if err != nil {
		l().Infof("GetSwarmMemUsage: GetClient error: %v", err)
		return "N/A", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Get total memory from nodes
	nodes, err := c.NodeList(ctx, swarm.NodeListOptions{})
	if err != nil {
		l().Infof("GetSwarmMemUsage: NodeList error: %v", err)
		return "N/A", err
	}

	var totalMemBytes int64
	for _, node := range nodes {
		totalMemBytes += node.Description.Resources.MemoryBytes
	}

	l().Infof("GetSwarmMemUsage: Total memory: %d bytes", totalMemBytes)

	if totalMemBytes == 0 {
		return "N/A", nil
	}

	// Get memory usage from containers (sample up to 10)
	containers, err := c.ContainerList(ctx, container.ListOptions{})
	if err != nil {
		l().Infof("GetSwarmMemUsage: ContainerList error: %v", err)
		return "N/A", err
	}

	l().Infof("GetSwarmMemUsage: Found %d containers", len(containers))

	if len(containers) == 0 {
		return "0.0%", nil
	}

	var usedMemBytes int64
	statsCount := 0
	maxContainers := len(containers)
	if maxContainers > 3 {
		maxContainers = 3
	}
	
	for i := 0; i < maxContainers; i++ {
		cont := containers[i]
		
		stats, err := c.ContainerStats(context.Background(), cont.ID, false)
		if err != nil {
			l().Infof("GetSwarmMemUsage: ContainerStats error for %s: %v", cont.ID[:12], err)
			continue
		}
		
		var s container.StatsResponse
		decodeErr := json.NewDecoder(stats.Body).Decode(&s)
		stats.Body.Close()
		
		if decodeErr != nil {
			l().Infof("GetSwarmMemUsage: Decode error for %s: %v", cont.ID[:12], decodeErr)
			continue
		}

		usedMemBytes += int64(s.MemoryStats.Usage)
		statsCount++
		l().Infof("GetSwarmMemUsage: Container %s Mem: %d bytes", cont.ID[:12], s.MemoryStats.Usage)
	}

	// Scale up if we sampled
	if statsCount > 0 && len(containers) > maxContainers {
		usedMemBytes = usedMemBytes * int64(len(containers)) / int64(maxContainers)
	}

	if statsCount == 0 {
		l().Infof("GetSwarmMemUsage: No stats collected")
		return "0.0%", nil
	}

	memPercent := float64(usedMemBytes) / float64(totalMemBytes) * 100.0
	result := fmt.Sprintf("%.1f%%", memPercent)
	l().Infof("GetSwarmMemUsage: Final result: %s (from %d containers)", result, statsCount)
	return result, nil
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
