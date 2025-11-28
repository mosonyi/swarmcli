package systeminfoview

import (
	"fmt"

	"github.com/briandowns/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case Msg:
		m.SetContent(msg)
		// Trigger slow status load right after fast values
		return LoadSlowStatus()
	
	case SlowStatusMsg:
		m.updateCPUMem(msg)
		return nil
	
	case TickMsg:
		// Only show spinner on first load, keep previous values during refresh
		if m.firstLoad {
			m.loadingCPU = true
			m.loadingMem = true
			m.content = m.buildContent()
		}
		// Trigger slow status reload and schedule next tick
		return tea.Batch(LoadSlowStatus(), m.tickCmd())
	
	case SpinnerTickMsg:
		// Fast animation tick - always keep running
		m.spinner++
		needsUpdate := false
		
		if m.loadingCPU || m.loadingMem {
			needsUpdate = true
		}
		
		// Handle pulsing for trend arrows - decrement every 3 ticks for slower pulse
		if m.cpuBlinkCount > 0 || m.memBlinkCount > 0 {
			// Decrement counters every 3rd tick (240ms intervals)
			if m.spinner%3 == 0 {
				if m.cpuBlinkCount > 0 {
					m.cpuBlinkCount--
				}
				if m.memBlinkCount > 0 {
					m.memBlinkCount--
				}
			}
			needsUpdate = true
		}
		
		if needsUpdate {
			m.content = m.buildContent()
		}
		
		return m.spinnerTickCmd()
	}

	var cmd tea.Cmd
	return cmd
}

func (m *Model) buildContent() string {
	// Use briandowns/spinner character set 14 (dots)
	spinnerFrames := spinner.CharSets[14]
	
	cpu := m.cpuUsage
	if m.loadingCPU {
		cpu = spinnerFrames[m.spinner%len(spinnerFrames)]
	} else if m.cpuBlinkCount > 0 && m.cpuBlinkCount%2 == 1 {
		// Hide arrow during odd blink counts (creates pulse effect)
		var cpuVal float64
		fmt.Sscanf(m.cpuUsage, "%f%%", &cpuVal)
		cpu = fmt.Sprintf("%.1f%%", cpuVal)
	}
	
	mem := m.memUsage
	if m.loadingMem {
		mem = spinnerFrames[m.spinner%len(spinnerFrames)]
	} else if m.memBlinkCount > 0 && m.memBlinkCount%2 == 1 {
		// Hide arrow during odd blink counts (creates pulse effect)
		var memVal float64
		fmt.Sscanf(m.memUsage, "%f%%", &memVal)
		mem = fmt.Sprintf("%.1f%%", memVal)
	}
	
	return content(
		m.context, m.version, cpu, mem, m.containerCount, m.serviceCount,
	)
}

func content(context, version, cpu, mem string, containers, services int) string {
	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("214")).
		Bold(true)
	
	// Pad labels to align values
	return fmt.Sprintf(
		"%s %s\n%s %s\n%s %s\n%s %s\n%s %d\n%s %d",
		labelStyle.Render("Context:   "), context,
		labelStyle.Render("Version:   "), version,
		labelStyle.Render("CPU:       "), cpu,
		labelStyle.Render("MEM:       "), mem,
		labelStyle.Render("Containers:"), containers,
		labelStyle.Render("Services:  "), services,
	)
}

func (m *Model) SetContent(msg Msg) {
	m.context = msg.context
	
	spinnerMarker := spinner.CharSets[14][0]
	
	// Only update CPU if it's not the spinner marker
	if msg.cpu == spinnerMarker {
		// Keep loading flag, don't update cpuUsage (buildContent will show spinner)
		m.loadingCPU = true
	} else if msg.cpu != "" {
		// Got a real value
		m.cpuUsage = msg.cpu
		m.loadingCPU = false
	}
	// If msg.cpu is empty, keep current state
	
	if msg.mem == spinnerMarker {
		// Keep loading flag, don't update memUsage (buildContent will show spinner)
		m.loadingMem = true
	} else if msg.mem != "" {
		// Got a real value
		m.memUsage = msg.mem
		m.loadingMem = false
	}
	// If msg.mem is empty, keep current state
	
	m.containerCount = msg.containers
	m.serviceCount = msg.services

	m.content = m.buildContent()
}

func (m *Model) updateCPUMem(msg SlowStatusMsg) {
	// Parse current CPU/MEM values and add trend arrows
	var currentCPU, currentMem float64
	fmt.Sscanf(msg.cpu, "%f%%", &currentCPU)
	fmt.Sscanf(msg.mem, "%f%%", &currentMem)
	
	// Clear loading flags and firstLoad
	m.loadingCPU = false
	m.loadingMem = false
	m.firstLoad = false
	
	// Add trend arrows if we have previous values
	if m.prevCPU > 0 {
		var currentTrend string
		if currentCPU > m.prevCPU {
			currentTrend = "up"
			m.cpuUsage = msg.cpu + " " + lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("↑")
			// Pulse on every change
			if m.prevCPUTrend != currentTrend {
				m.cpuBlinkCount = 6
			}
		} else if currentCPU < m.prevCPU {
			currentTrend = "down"
			m.cpuUsage = msg.cpu + " " + lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Render("↓")
			// Pulse on every change
			if m.prevCPUTrend != currentTrend {
				m.cpuBlinkCount = 6
			}
		} else {
			currentTrend = ""
			m.cpuUsage = msg.cpu
		}
		m.prevCPUTrend = currentTrend
	} else {
		m.cpuUsage = msg.cpu
	}
	
	if m.prevMem > 0 {
		var currentTrend string
		if currentMem > m.prevMem {
			currentTrend = "up"
			m.memUsage = msg.mem + " " + lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("↑")
			// Pulse on every change
			if m.prevMemTrend != currentTrend {
				m.memBlinkCount = 6
			}
		} else if currentMem < m.prevMem {
			currentTrend = "down"
			m.memUsage = msg.mem + " " + lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Render("↓")
			// Pulse on every change
			if m.prevMemTrend != currentTrend {
				m.memBlinkCount = 6
			}
		} else {
			currentTrend = ""
			m.memUsage = msg.mem
		}
		m.prevMemTrend = currentTrend
	} else {
		m.memUsage = msg.mem
	}
	
	// Update previous values
	m.prevCPU = currentCPU
	m.prevMem = currentMem

	m.content = m.buildContent()
}
