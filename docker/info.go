// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/swarm"
)

// apiTimeout is the default timeout for Docker API calls. Prevents indefinite
// hangs when the Docker daemon is behind a slow proxy (e.g. RBAC proxy in BE).
const apiTimeout = 10 * time.Second

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

	ctx, cancel := context.WithTimeout(context.Background(), apiTimeout)
	defer cancel()
	containers, err := c.ContainerList(ctx, container.ListOptions{All: true})
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

	ctx, cancel := context.WithTimeout(context.Background(), apiTimeout)
	defer cancel()
	services, err := c.ServiceList(ctx, swarm.ServiceListOptions{})
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

	ctx, cancel := context.WithTimeout(context.Background(), apiTimeout)
	defer cancel()
	nodes, err := c.NodeList(ctx, swarm.NodeListOptions{})
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

	ctx, cancel := context.WithTimeout(context.Background(), apiTimeout)
	defer cancel()
	nodes, err := c.NodeList(ctx, swarm.NodeListOptions{})
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

	ctx, cancel := context.WithTimeout(context.Background(), apiTimeout)
	defer cancel()
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

			stats, err := c.ContainerStats(ctx, containerID, false)
			if err != nil {
				l().Infof("GetSwarmCPUUsage: ContainerStats error for %s: %v", containerID[:12], err)
				results <- cpuResult{err: err}
				return
			}
			defer func() { _ = stats.Body.Close() }()

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

// memUsageNoCache is the memory figure `docker stats` reports: the cgroup's
// usage less the page cache, which is reclaimable and so is not the container's
// working set. cgroup v1 names that quantity total_inactive_file and cgroup v2
// names it inactive_file; either is subtracted only when it is smaller than
// Usage, which is how docker/cli guards a counter the host did not supply.
//
// Without the subtraction the header reads high — by the whole page cache — and
// disagrees with `docker stats` and with every other view of the same number.
func memUsageNoCache(mem container.MemoryStats) uint64 {
	// cgroup v1
	if v, isCgroup1 := mem.Stats["total_inactive_file"]; isCgroup1 && v < mem.Usage {
		return mem.Usage - v
	}
	// cgroup v2
	if v := mem.Stats["inactive_file"]; v < mem.Usage {
		return mem.Usage - v
	}
	return mem.Usage
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

	ctx, cancel := context.WithTimeout(context.Background(), apiTimeout)
	defer cancel()
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

			stats, err := c.ContainerStats(ctx, containerID, false)
			if err != nil {
				l().Infof("GetSwarmMemUsage: ContainerStats error for %s: %v", containerID[:12], err)
				results <- memResult{err: err}
				return
			}
			defer func() { _ = stats.Body.Close() }()

			var s container.StatsResponse
			decodeErr := json.NewDecoder(stats.Body).Decode(&s)

			if decodeErr != nil {
				l().Infof("GetSwarmMemUsage: Decode error for %s: %v", containerID[:12], decodeErr)
				results <- memResult{err: decodeErr}
				return
			}

			results <- memResult{usage: int64(memUsageNoCache(s.MemoryStats))}
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

	ctx, cancel := context.WithTimeout(context.Background(), apiTimeout)
	defer cancel()
	info, err := c.ServerVersion(ctx)
	if err != nil {
		return "unknown", err
	}
	return info.Version, nil
}

// GetSwarmResourceUsage returns CPU and memory usage in a single pass,
// making one ContainerList call and one ContainerStats call per container
// instead of two separate passes. This halves the Docker API calls compared
// to calling GetSwarmCPUUsage + GetSwarmMemUsage independently.
func GetSwarmResourceUsage() (cpuPct string, memPct string, err error) {
	c, err := GetClient()
	if err != nil {
		return "N/A", "N/A", err
	}

	totalCapacity, capErr := GetSwarmMemCapacity()
	if capErr != nil || totalCapacity == 0 {
		l().Infof("GetSwarmResourceUsage: failed to get mem capacity: %v", capErr)
		totalCapacity = 0 // continue — CPU can still be reported
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiTimeout)
	defer cancel()
	containers, err := c.ContainerList(ctx, container.ListOptions{})
	if err != nil {
		return "N/A", "N/A", err
	}

	if len(containers) == 0 {
		memStr := "0.0%"
		if totalCapacity == 0 {
			memStr = "N/A"
		}
		return "0.0%", memStr, nil
	}

	type result struct {
		cpuPercent float64
		memUsage   int64
		err        error
	}

	results := make(chan result, len(containers))
	var wg sync.WaitGroup

	for _, cont := range containers {
		wg.Add(1)
		go func(containerID string) {
			defer wg.Done()

			stats, err := c.ContainerStats(ctx, containerID, false)
			if err != nil {
				results <- result{err: err}
				return
			}
			defer func() { _ = stats.Body.Close() }()

			var s container.StatsResponse
			if decodeErr := json.NewDecoder(stats.Body).Decode(&s); decodeErr != nil {
				results <- result{err: decodeErr}
				return
			}

			// CPU
			cpuDelta := float64(s.CPUStats.CPUUsage.TotalUsage - s.PreCPUStats.CPUUsage.TotalUsage)
			systemDelta := float64(s.CPUStats.SystemUsage - s.PreCPUStats.SystemUsage)
			onlineCPUs := float64(s.CPUStats.OnlineCPUs)
			if onlineCPUs == 0 {
				onlineCPUs = float64(len(s.CPUStats.CPUUsage.PercpuUsage))
			}
			var cpuPct float64
			if systemDelta > 0 && onlineCPUs > 0 {
				cpuPct = (cpuDelta / systemDelta) * onlineCPUs * 100.0
			}

			results <- result{cpuPercent: cpuPct, memUsage: int64(memUsageNoCache(s.MemoryStats))}
		}(cont.ID)
	}

	go func() { wg.Wait(); close(results) }()

	var totalCPU float64
	var totalMem int64
	successCount := 0
	for res := range results {
		if res.err == nil {
			totalCPU += res.cpuPercent
			totalMem += res.memUsage
			successCount++
		}
	}

	cpuPct = "0.0%"
	if successCount > 0 {
		cpuPct = fmt.Sprintf("%.1f%%", totalCPU)
	}

	memPct = "N/A"
	if totalCapacity > 0 && successCount > 0 {
		memPct = fmt.Sprintf("%.1f%%", (float64(totalMem)/float64(totalCapacity))*100.0)
	} else if successCount > 0 {
		memPct = "0.0%"
	}

	return cpuPct, memPct, nil
}
