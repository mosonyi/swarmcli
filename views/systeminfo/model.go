package systeminfoview

import (
	tea "github.com/charmbracelet/bubbletea"
	"os"
	"swarmcli/docker"
)

type Model struct {
	// We don't need a viewport here, as we will use a fixed size for the content.
	content string

	version string

	host           string
	cpuUsage       string
	memUsage       string
	containerCount int
	serviceCount   int
}

// Create a new instance
func New(version string) *Model {
	return &Model{
		content: content("", version, "", "", 0, 0),
		version: version,
	}
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func LoadStatus() tea.Cmd {
	return func() tea.Msg {
		cpu, _ := docker.GetSwarmCPUUsage()
		mem, _ := docker.GetSwarmMemUsage()
		containers, _ := docker.GetContainerCount()
		services, _ := docker.GetServiceCount()

		host, _ := os.Hostname()
		return Msg{
			host:       host,
			cpu:        cpu,
			mem:        mem,
			containers: containers,
			services:   services,
		}
	}
}
