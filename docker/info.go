package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

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

// GetSwarmCPUCapacity returns total CPU cores across all nodes (fast).
func GetSwarmCPUCapacity() (float64, error) {
	c, err := GetClient()
	if err != nil {
		return 0, err
	}

	nodes, err := c.NodeList(context.Background(), swarm.NodeListOptions{})
	if err != nil {
		return 0, err
	}

	var totalCPUs float64
	for _, node := range nodes {
		if node.Status.State == swarm.NodeStateReady {
			totalCPUs += float64(node.Description.Resources.NanoCPUs) / 1e9
		}
	}
	return totalCPUs, nil
}

// GetSwarmMemCapacity returns total memory across all nodes (fast).
func GetSwarmMemCapacity() (int64, error) {
	c, err := GetClient()
	if err != nil {
		return 0, err
	}

	nodes, err := c.NodeList(context.Background(), swarm.NodeListOptions{})
	if err != nil {
		return 0, err
	}

	var totalMem int64
	for _, node := range nodes {
		if node.Status.State == swarm.NodeStateReady {
			totalMem += node.Description.Resources.MemoryBytes
		}
	}
	return totalMem, nil
}

// GetSwarmCPUUsage returns actual CPU usage across running containers.
func GetSwarmCPUUsage() (string, error) {
	c, err := GetClient()
	if err != nil {
		l().Infof("GetSwarmCPUUsage: GetClient error: %v", err)
		return "N/A", err
	}

	ctx := context.Background()
	containers, err := c.ContainerList(ctx, container.ListOptions{})
	if err != nil {
		l().Infof("GetSwarmCPUUsage: ContainerList error: %v", err)
		return "N/A", err
	}

	if len(containers) == 0 {
		return "0.0%", nil
	}

	l().Infof("GetSwarmCPUUsage: Collecting stats from %d containers in parallel", len(containers))

	// Use goroutines to collect stats in parallel
	type cpuResult struct {
		percent float64
		err     error
	}

	results := make(chan cpuResult, len(containers))
	var wg sync.WaitGroup

	for _, cont := range containers {
		wg.Add(1)
		go func(containerID string) {
			defer wg.Done()

			stats, err := c.ContainerStats(context.Background(), containerID, false)
			if err != nil {
				l().Infof("GetSwarmCPUUsage: ContainerStats error for %s: %v", containerID[:12], err)
				results <- cpuResult{err: err}
				return
			}

			var s container.StatsResponse
			decodeErr := json.NewDecoder(stats.Body).Decode(&s)

			if decodeErr != nil {
				l().Infof("GetSwarmCPUUsage: Decode error for %s: %v", containerID[:12], decodeErr)
				results <- cpuResult{err: decodeErr}
				return
			}

			// Calculate CPU percentage
			cpuDelta := float64(s.CPUStats.CPUUsage.TotalUsage - s.PreCPUStats.CPUUsage.TotalUsage)
			systemDelta := float64(s.CPUStats.SystemUsage - s.PreCPUStats.SystemUsage)
			onlineCPUs := float64(s.CPUStats.OnlineCPUs)

			if onlineCPUs == 0 {
				onlineCPUs = float64(len(s.CPUStats.CPUUsage.PercpuUsage))
			}

			var cpuPercent float64
			if systemDelta > 0 && onlineCPUs > 0 {
				cpuPercent = (cpuDelta / systemDelta) * onlineCPUs * 100.0
			}

			results <- cpuResult{percent: cpuPercent}
		}(cont.ID)
	}

	// Close results channel after all goroutines complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	var totalCPU float64
	successCount := 0
	for res := range results {
		if res.err == nil {
			totalCPU += res.percent
			successCount++
		}
	}

	if successCount == 0 {
		return "0.0%", nil
	}

	result := fmt.Sprintf("%.1f%%", totalCPU)
	l().Infof("GetSwarmCPUUsage: Final result: %s (from %d/%d containers)", result, successCount, len(containers))
	return result, nil
}

// GetSwarmMemUsage returns actual memory usage across running containers.
func GetSwarmMemUsage() (string, error) {
	c, err := GetClient()
	if err != nil {
		l().Infof("GetSwarmMemUsage: GetClient error: %v", err)
		return "N/A", err
	}

	// Get total memory capacity from nodes
	totalCapacity, err := GetSwarmMemCapacity()
	if err != nil || totalCapacity == 0 {
		l().Infof("GetSwarmMemUsage: failed to get capacity: %v", err)
		return "N/A", err
	}

	ctx := context.Background()
	containers, err := c.ContainerList(ctx, container.ListOptions{})
	if err != nil {
		l().Infof("GetSwarmMemUsage: ContainerList error: %v", err)
		return "N/A", err
	}

	if len(containers) == 0 {
		return "0.0%", nil
	}

	l().Infof("GetSwarmMemUsage: Collecting stats from %d containers in parallel", len(containers))

	// Use goroutines to collect stats in parallel
	type memResult struct {
		usage int64
		err   error
	}

	results := make(chan memResult, len(containers))
	var wg sync.WaitGroup

	for _, cont := range containers {
		wg.Add(1)
		go func(containerID string) {
			defer wg.Done()

			stats, err := c.ContainerStats(context.Background(), containerID, false)
			if err != nil {
				l().Infof("GetSwarmMemUsage: ContainerStats error for %s: %v", containerID[:12], err)
				results <- memResult{err: err}
				return
			}

			var s container.StatsResponse
			decodeErr := json.NewDecoder(stats.Body).Decode(&s)

			if decodeErr != nil {
				l().Infof("GetSwarmMemUsage: Decode error for %s: %v", containerID[:12], decodeErr)
				results <- memResult{err: decodeErr}
				return
			}

			results <- memResult{usage: int64(s.MemoryStats.Usage)}
		}(cont.ID)
	}

	// Close results channel after all goroutines complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	var totalUsedBytes int64
	successCount := 0
	for res := range results {
		if res.err == nil {
			totalUsedBytes += res.usage
			successCount++
		}
	}

	if successCount == 0 {
		return "0.0%", nil
	}

	// Calculate percentage
	memPercent := (float64(totalUsedBytes) / float64(totalCapacity)) * 100.0

	result := fmt.Sprintf("%.1f%%", memPercent)
	l().Infof("GetSwarmMemUsage: Final result: %s (%.1f GB used of %.1f GB total, from %d/%d containers)",
		result, float64(totalUsedBytes)/(1024*1024*1024), float64(totalCapacity)/(1024*1024*1024), successCount, len(containers))
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
