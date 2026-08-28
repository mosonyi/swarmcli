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

// cpuBaseline is one container's cumulative CPU counters from the previous
// sampling round.
//
// One-shot stats zeroes PreCPUStats, so a retained reading is the only source of
// a previous value to difference against. That is the trade one-shot asks for,
// and it is worth taking: the alternative — `stream=0` without `one-shot` —
// makes the daemon subscribe the container to its global stats collector and
// discard the first frame to prime the delta itself, which costs a second or two
// per call and keeps every sampled container on a 1 Hz cgroup treadmill for as
// long as the request is open (moby daemon/stats.go, daemon/stats/collector.go).
// Differencing over our own longer window also gives a steadier figure than the
// one-second window `docker stats` reads off.
type cpuBaseline struct {
	total  uint64
	system uint64
}

var (
	cpuBaselineMu sync.Mutex
	cpuBaselines  = map[string]cpuBaseline{}
)

// cpuPercentSince differences a fresh reading against this container's retained
// baseline and replaces it. ok is false when there is nothing to difference
// against — the first time a container is seen — and when a counter has gone
// backwards, which means the container restarted and the delta would be
// nonsense. In both cases the caller omits this container from the CPU figure
// rather than contributing a wrong number; the next round has a baseline.
func cpuPercentSince(id string, s container.CPUStats) (float64, bool) {
	cpuBaselineMu.Lock()
	prev, seen := cpuBaselines[id]
	cpuBaselines[id] = cpuBaseline{total: s.CPUUsage.TotalUsage, system: s.SystemUsage}
	cpuBaselineMu.Unlock()

	if !seen || s.CPUUsage.TotalUsage < prev.total || s.SystemUsage < prev.system {
		return 0, false
	}
	systemDelta := float64(s.SystemUsage - prev.system)
	if systemDelta <= 0 {
		return 0, false
	}
	onlineCPUs := float64(s.OnlineCPUs)
	if onlineCPUs == 0 {
		onlineCPUs = float64(len(s.CPUUsage.PercpuUsage))
	}
	if onlineCPUs == 0 {
		return 0, false
	}
	return (float64(s.CPUUsage.TotalUsage-prev.total) / systemDelta) * onlineCPUs * 100.0, true
}

// pruneCPUBaselines drops the baselines of containers that are no longer
// running, so the map tracks the node rather than growing for the lifetime of
// the process.
func pruneCPUBaselines(live map[string]struct{}) {
	cpuBaselineMu.Lock()
	defer cpuBaselineMu.Unlock()
	for id := range cpuBaselines {
		if _, ok := live[id]; !ok {
			delete(cpuBaselines, id)
		}
	}
}

// GetSwarmResourceUsage returns CPU and memory usage for the containers on the
// connected node, in a single pass: one ContainerList plus one one-shot
// ContainerStats per container.
//
// Memory is instantaneous and is reported from the first round. CPU is a rate,
// so it needs two readings; a container is counted once it has a baseline (see
// cpuBaseline).
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

	live := make(map[string]struct{}, len(containers))
	for _, cont := range containers {
		live[cont.ID] = struct{}{}
	}
	pruneCPUBaselines(live)

	if len(containers) == 0 {
		memStr := "0.0%"
		if totalCapacity == 0 {
			memStr = "N/A"
		}
		return "0.0%", memStr, nil
	}

	type result struct {
		cpuPercent float64
		hasCPU     bool
		memUsage   int64
		err        error
	}

	results := make(chan result, len(containers))
	var wg sync.WaitGroup

	for _, cont := range containers {
		wg.Add(1)
		go func(containerID string) {
			defer wg.Done()

			stats, err := c.ContainerStatsOneShot(ctx, containerID)
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

			pct, hasCPU := cpuPercentSince(containerID, s.CPUStats)
			results <- result{
				cpuPercent: pct,
				hasCPU:     hasCPU,
				memUsage:   int64(memUsageNoCache(s.MemoryStats)),
			}
		}(cont.ID)
	}

	go func() { wg.Wait(); close(results) }()

	var totalCPU float64
	var totalMem int64
	cpuCount, memCount := 0, 0
	for res := range results {
		if res.err != nil {
			continue
		}
		memCount++
		totalMem += res.memUsage
		if res.hasCPU {
			cpuCount++
			totalCPU += res.cpuPercent
		}
	}

	// No baseline yet on any container (the very first round) reads as unknown,
	// not as an idle cluster: "0.0%" there would be a measurement we did not make.
	cpuPct = "N/A"
	if cpuCount > 0 {
		cpuPct = fmt.Sprintf("%.1f%%", totalCPU)
	}

	memPct = "N/A"
	if memCount > 0 && totalCapacity > 0 {
		memPct = fmt.Sprintf("%.1f%%", (float64(totalMem)/float64(totalCapacity))*100.0)
	}

	return cpuPct, memPct, nil
}
