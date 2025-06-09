package systeminfoview

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case Msg:
		m.SetContent(msg)
		return m, nil
	}

	var cmd tea.Cmd
	return m, cmd
}

func (m *Model) buildContent() string {
	return content(
		m.host, m.version, m.cpuUsage, m.memUsage, m.containerCount, m.serviceCount,
	)
}

func content(host, version, cpu, mem string, containers, services int) string {
	return fmt.Sprintf(
		"Host: %s\nVersion: %s\nCPU: %s\nMEM: %s\nContainers: %d\nServices: %d",
		host, version, cpu, mem, containers, services,
	)
}

func (m *Model) SetContent(msg Msg) {
	m.host = msg.host
	m.cpuUsage = msg.cpu
	m.memUsage = msg.mem
	m.containerCount = msg.containers
	m.serviceCount = msg.services

	m.content = m.buildContent()
}
