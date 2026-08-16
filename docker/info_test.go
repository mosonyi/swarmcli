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
