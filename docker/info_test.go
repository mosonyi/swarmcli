// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

import (
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/stretchr/testify/require"
)

func TestMemUsageNoCache(t *testing.T) {
	const usage = 500

	cases := []struct {
		name  string
		stats map[string]uint64
		usage uint64
		want  uint64
	}{
		{
			name:  "cgroup v1 subtracts total_inactive_file",
			stats: map[string]uint64{"total_inactive_file": 100, "inactive_file": 400},
			usage: usage,
			want:  400,
		},
		{
			name:  "cgroup v2 subtracts inactive_file",
			stats: map[string]uint64{"inactive_file": 120},
			usage: usage,
			want:  380,
		},
		{
			// A counter at least as large as Usage is not a page-cache figure we
			// can trust, so it is ignored rather than producing an underflow.
			name:  "v1 counter not smaller than usage is ignored",
			stats: map[string]uint64{"total_inactive_file": usage},
			usage: usage,
			want:  usage,
		},
		{
			name:  "v2 counter not smaller than usage is ignored",
			stats: map[string]uint64{"inactive_file": usage + 1},
			usage: usage,
			want:  usage,
		},
		{
			name:  "no cache counters reported",
			stats: map[string]uint64{},
			usage: usage,
			want:  usage,
		},
		{
			name:  "nil stats map",
			stats: nil,
			usage: usage,
			want:  usage,
		},
		{
			name:  "stopped container reports nothing",
			stats: nil,
			usage: 0,
			want:  0,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := memUsageNoCache(container.MemoryStats{Usage: c.usage, Stats: c.stats})
			require.Equal(t, c.want, got)
		})
	}
}

// The header must report what `docker stats` reports. A raw MemoryStats.Usage
// counts the page cache, which is reclaimable, and reads high by exactly the
// amount this test pins.
func TestMemUsageNoCache_DiffersFromRawUsage(t *testing.T) {
	m := container.MemoryStats{Usage: 1 << 30, Stats: map[string]uint64{"inactive_file": 768 << 20}}
	require.Equal(t, uint64(256<<20), memUsageNoCache(m))
	require.NotEqual(t, m.Usage, memUsageNoCache(m))
}

// cpuStats builds a reading with the two cumulative counters the percentage is
// differenced from, plus the core count it is scaled by.
func cpuStats(total, system uint64, online uint32) container.CPUStats {
	return container.CPUStats{
		CPUUsage:    container.CPUUsage{TotalUsage: total},
		SystemUsage: system,
		OnlineCPUs:  online,
	}
}

// resetBaselines isolates a test from whatever a previous one left behind, since
// the baselines are package state shared by every caller in the process.
func resetBaselines(t *testing.T) {
	t.Helper()
	cpuBaselineMu.Lock()
	cpuBaselines = map[string]cpuBaseline{}
	cpuBaselineMu.Unlock()
}

func TestCPUPercentSince_FirstReadingHasNoBaseline(t *testing.T) {
	resetBaselines(t)

	_, ok := cpuPercentSince("c1", cpuStats(100, 1000, 2))
	require.False(t, ok, "a rate needs two readings; the first must not be reported")

	// ...but it is retained, so the second reading can be differenced.
	pct, ok := cpuPercentSince("c1", cpuStats(200, 2000, 2))
	require.True(t, ok)
	require.InDelta(t, 20.0, pct, 0.001) // (100/1000) * 2 * 100
}

func TestCPUPercentSince_UsesPercpuWhenOnlineCPUsAbsent(t *testing.T) {
	resetBaselines(t)

	s := cpuStats(100, 1000, 0)
	s.CPUUsage.PercpuUsage = []uint64{1, 2, 3, 4}
	cpuPercentSince("c1", s)

	s = cpuStats(200, 2000, 0)
	s.CPUUsage.PercpuUsage = []uint64{1, 2, 3, 4}
	pct, ok := cpuPercentSince("c1", s)
	require.True(t, ok)
	require.InDelta(t, 40.0, pct, 0.001) // (100/1000) * 4 * 100
}

func TestCPUPercentSince_NoCoreCountIsNotAPercentage(t *testing.T) {
	resetBaselines(t)

	cpuPercentSince("c1", cpuStats(100, 1000, 0))
	_, ok := cpuPercentSince("c1", cpuStats(200, 2000, 0))
	require.False(t, ok, "with neither OnlineCPUs nor PercpuUsage there is nothing to scale by")
}

func TestCPUPercentSince_StillSecondsOfSystemTimeIsNotZeroPercent(t *testing.T) {
	resetBaselines(t)

	cpuPercentSince("c1", cpuStats(100, 1000, 2))
	_, ok := cpuPercentSince("c1", cpuStats(150, 1000, 2))
	require.False(t, ok, "a zero system delta has no window to divide by")
}

func TestCPUPercentSince_RestartedCounterIsSkippedThenRecovers(t *testing.T) {
	resetBaselines(t)

	cpuPercentSince("c1", cpuStats(5000, 9000, 2))

	// The container restarted: its cumulative CPU time went backwards, so the
	// delta would be a large negative number rendered as a nonsense percentage.
	_, ok := cpuPercentSince("c1", cpuStats(10, 10000, 2))
	require.False(t, ok)

	// The post-restart reading became the new baseline, so the next round works.
	pct, ok := cpuPercentSince("c1", cpuStats(110, 11000, 2))
	require.True(t, ok)
	require.InDelta(t, 20.0, pct, 0.001) // (100/1000) * 2 * 100
}

func TestCPUPercentSince_BaselinesAreKeptPerContainer(t *testing.T) {
	resetBaselines(t)

	cpuPercentSince("c1", cpuStats(100, 1000, 1))
	cpuPercentSince("c2", cpuStats(500, 1000, 1))

	pct1, ok := cpuPercentSince("c1", cpuStats(200, 2000, 1))
	require.True(t, ok)
	require.InDelta(t, 10.0, pct1, 0.001)

	pct2, ok := cpuPercentSince("c2", cpuStats(1000, 2000, 1))
	require.True(t, ok)
	require.InDelta(t, 50.0, pct2, 0.001)
}

func TestPruneCPUBaselines_DropsOnlyContainersThatAreGone(t *testing.T) {
	resetBaselines(t)

	cpuPercentSince("stays", cpuStats(100, 1000, 1))
	cpuPercentSince("goes", cpuStats(100, 1000, 1))

	pruneCPUBaselines(map[string]struct{}{"stays": {}})

	cpuBaselineMu.Lock()
	_, keptLive := cpuBaselines["stays"]
	_, keptDead := cpuBaselines["goes"]
	cpuBaselineMu.Unlock()

	require.True(t, keptLive, "a still-running container keeps its baseline")
	require.False(t, keptDead, "the map tracks the node, not the process lifetime")

	// The dropped container is a first reading again, not a delta against a
	// baseline from a previous incarnation.
	_, ok := cpuPercentSince("goes", cpuStats(50, 3000, 1))
	require.False(t, ok)
}
