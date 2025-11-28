package systeminfoview

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case Msg:
		m.SetContent(msg)
		return nil
	}

	var cmd tea.Cmd
	return cmd
}

func (m *Model) buildContent() string {
	return content(
		m.host, m.version, m.cpuUsage, m.memUsage, m.containerCount, m.serviceCount,
	)
}

func content(host, version, cpu, mem string, containers, services int) string {
	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("214")).
		Bold(true)
	
	// Pad labels to align values
	return fmt.Sprintf(
		"%s %s\n%s %s\n%s %s\n%s %s\n%s %d\n%s %d",
		labelStyle.Render("Host:      "), host,
		labelStyle.Render("Version:   "), version,
		labelStyle.Render("CPU:       "), cpu,
		labelStyle.Render("MEM:       "), mem,
		labelStyle.Render("Containers:"), containers,
		labelStyle.Render("Services:  "), services,
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
