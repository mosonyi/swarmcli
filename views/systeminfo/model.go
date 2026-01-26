// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package systeminfoview

import (
	"fmt"
	"time"

	"swarmcli/docker"

	"github.com/briandowns/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	// We don't need a viewport here, as we will use a fixed size for the content.
	content string

	version string

	context        string
	cpuUsage       string
	memUsage       string
	cpuCapacity    string // Total CPU cores
	memCapacity    string // Total memory
	containerCount int
	serviceCount   int

	// For tracking trends
	prevCPU        float64
	prevMem        float64
	lastUpdate     time.Time
	updateInterval time.Duration

	// Loading state
	loadingCPU bool
	loadingMem bool
	spinner    int
	firstLoad  bool

	// Trend arrow state
	prevCPUTrend  string // "up", "down", or ""
	prevMemTrend  string
	cpuBlinkCount int
	memBlinkCount int
}

// Create a new instance
func New(version string) *Model {
	// Get initial context synchronously to display immediately
	context, _ := docker.GetCurrentContext()
	return &Model{
		content:        content(context, version, "", "", 0, 0),
		version:        version,
		context:        context,
		updateInterval: 8 * time.Second,
		lastUpdate:     time.Now(),
		loadingCPU:     true,
		loadingMem:     true,
		firstLoad:      true,
	}
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.tickCmd(), m.spinnerTickCmd())
}

func (m *Model) tickCmd() tea.Cmd {
	return tea.Tick(m.updateInterval, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

func (m *Model) spinnerTickCmd() tea.Cmd {
	return tea.Tick(80*time.Millisecond, func(t time.Time) tea.Msg {
		return SpinnerTickMsg(t)
	})
}

func LoadStatus() tea.Cmd {
	return func() tea.Msg {
		// Get fast values immediately
		context, _ := docker.GetCurrentContext()
		containers, _ := docker.GetContainerCount()
		services, _ := docker.GetServiceCount()

		// Get capacity (fast) - show immediately
		cpuCapacity, _ := docker.GetSwarmCPUCapacity()
		memCapacity, _ := docker.GetSwarmMemCapacity()

		cpuCapStr := ""
		if cpuCapacity > 0 {
			cpuCapStr = fmt.Sprintf("%.0f cores", cpuCapacity)
		} else {
			cpuCapStr = "-- cores"
		}

		memCapStr := ""
		if memCapacity > 0 {
			memCapStr = fmt.Sprintf("%.0f GB", float64(memCapacity)/1024/1024/1024)
		} else {
			memCapStr = "--- GB"
		}

		// Return immediately with fast values, spinner marker for CPU/MEM usage
		// Using first frame of spinner charset 14 as marker
		spinnerMarker := spinner.CharSets[14][0]
		return Msg{
			context:     context,
			cpu:         spinnerMarker,
			mem:         spinnerMarker,
			cpuCapacity: cpuCapStr,
			memCapacity: memCapStr,
			containers:  containers,
			services:    services,
		}
	}
}

func LoadSlowStatus() tea.Cmd {
	return func() tea.Msg {
		l().Info("LoadSlowStatus: Starting background stats collection")

		// Get CPU/MEM - these are slow
		cpu, err := docker.GetSwarmCPUUsage()
		if err != nil {
			l().Error("LoadSlowStatus: GetSwarmCPUUsage failed: %v", err)
			cpu = "N/A"
		}
		if cpu == "" {
			cpu = "0.0%"
		}
		l().Info("LoadSlowStatus: CPU usage collected: %s", cpu)

		mem, err := docker.GetSwarmMemUsage()
		if err != nil {
			l().Error("LoadSlowStatus: GetSwarmMemUsage failed: %v", err)
			mem = "N/A"
		}
		if mem == "" {
			mem = "0.0%"
		}
		l().Info("LoadSlowStatus: Memory usage collected: %s", mem)

		// Return only CPU/MEM update
		return SlowStatusMsg{
			cpu: cpu,
			mem: mem,
		}
	}
}
