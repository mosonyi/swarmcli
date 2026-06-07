// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package portsview

import (
	"fmt"
	"strings"
	"swarmcli/ui"

	"github.com/charmbracelet/lipgloss"
)

func (m *Model) FrameTitle() string {
	return "Required Swarm Ports"
}

func (m *Model) FrameHeader() string {
	return ui.FrameHeaderStyle.Render("Swarm cluster port requirements (node-to-node)")
}

func (m *Model) FrameFooter() string {
	return ""
}

func (m *Model) FrameContent() string {
	header := m.FrameHeader()
	width := m.viewport.Width
	if width <= 0 {
		width = 80
	}
	frame := ui.ComputeFrameDimensions(width, m.viewport.Height, width, m.height, header, "")
	return ui.TrimOrPadContentToLines(m.viewport.View(), frame.DesiredContentLines)
}

func (m *Model) View() string {
	if !m.ready {
		return ""
	}
	return ui.RenderViewFrame(m.FrameTitle(), m.FrameHeader(), m.FrameContent(), m.FrameFooter(), m.viewport.Width, m.viewport.Height, false)
}

func (m *Model) updateViewport() {
	var b strings.Builder

	// Style tokens
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("81")) // Light blue/cyan

	infoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("246")) // Gray description text

	portHeaderStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("208")) // Orange/amber for port headings

	labelStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("117")). // Cyan labels
		Width(12)

	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")) // White values

	// Compute card width dynamically
	cardWidth := m.viewport.Width - 6
	if cardWidth < 40 {
		cardWidth = 40
	}

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")). // Sleek blue border
		Padding(0, 2).
		MarginBottom(1).
		Width(cardWidth)

	// Helper to wrap value text to fit the card width and align multi-line values
	wrapVal := func(val string, labelW int) string {
		maxValW := cardWidth - labelW - 6 // minus label width, borders, and padding
		if maxValW < 15 {
			maxValW = 15
		}
		words := strings.Fields(val)
		if len(words) == 0 {
			return ""
		}
		var lines []string
		cur := words[0]
		for _, w := range words[1:] {
			if len(cur)+1+len(w) > maxValW {
				lines = append(lines, cur)
				cur = w
			} else {
				cur += " " + w
			}
		}
		lines = append(lines, cur)
		indent := strings.Repeat(" ", labelW)
		return strings.Join(lines, "\n"+indent)
	}

	// Intro text
	b.WriteString(titleStyle.Render("Required Swarm Cluster Ports (node-to-node)"))
	b.WriteString("\n")
	b.WriteString(infoStyle.Render("These ports must be open between all Swarm nodes (managers and workers) to ensure cluster coordination, node health state sync, and container overlay networking."))
	b.WriteString("\n\n")

	// Append Real-time Diagnostics
	diagnostics := m.getDiagnosticStatus()
	if diagnostics != "" {
		b.WriteString(diagnostics)
		b.WriteString("\n")
	}

	// Render Port 2377
	port1Content := fmt.Sprintf("%s%s\n%s%s\n%s%s",
		labelStyle.Render("Protocol:"), valueStyle.Render(wrapVal("TCP", 12)),
		labelStyle.Render("Used For:"), valueStyle.Render(wrapVal("Swarm control plane (manager election, cluster commands)", 12)),
		labelStyle.Render("Direction:"), valueStyle.Render(wrapVal("Managers only (inbound to managers)", 12)),
	)
	b.WriteString(borderStyle.Render(fmt.Sprintf("%s\n\n%s", portHeaderStyle.Render("TCP 2377 — Cluster management"), port1Content)))
	b.WriteString("\n")

	// Render Port 7946
	port2Content := fmt.Sprintf("%s%s\n%s%s\n%s%s",
		labelStyle.Render("Protocol:"), valueStyle.Render(wrapVal("TCP / UDP", 12)),
		labelStyle.Render("Used For:"), valueStyle.Render(wrapVal("Gossip network (node discovery, membership, state sync)", 12)),
		labelStyle.Render("Direction:"), valueStyle.Render(wrapVal("All nodes to all nodes", 12)),
	)
	b.WriteString(borderStyle.Render(fmt.Sprintf("%s\n\n%s", portHeaderStyle.Render("TCP / UDP 7946 — Node communication"), port2Content)))
	b.WriteString("\n")

	// Render Port 4789
	port3Content := fmt.Sprintf("%s%s\n%s%s\n%s%s",
		labelStyle.Render("Protocol:"), valueStyle.Render(wrapVal("UDP", 12)),
		labelStyle.Render("Used For:"), valueStyle.Render(wrapVal("Overlay network (VXLAN) container-to-container traffic", 12)),
		labelStyle.Render("Direction:"), valueStyle.Render(wrapVal("All nodes to all nodes", 12)),
	)
	b.WriteString(borderStyle.Render(fmt.Sprintf("%s\n\n%s", portHeaderStyle.Render("UDP 4789 — Overlay network (VXLAN)"), port3Content)))
	b.WriteString("\n")

	m.viewport.SetContent(b.String())
}

func (m *Model) getDiagnosticStatus() string {
	if m.deps.Snapshot == nil {
		return ""
	}

	snap := m.deps.Snapshot.GetSnapshot()
	if snap == nil {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("Loading real-time swarm diagnostics...")
	}

	entries := snap.ToNodeEntries()
	if len(entries) == 0 {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("No nodes found to analyze port status.")
	}

	totalNodes := len(entries)
	readyNodes := 0
	downNodes := 0
	managers := 0
	reachableManagers := 0
	unreachableManagers := 0

	for _, node := range entries {
		if node.State == "ready" {
			readyNodes++
		} else {
			downNodes++
		}

		if node.Manager {
			managers++
			status := strings.ToLower(node.ManagerStatus)
			if status == "leader" || status == "reachable" {
				reachableManagers++
			} else if status == "unreachable" {
				unreachableManagers++
			}
		}
	}

	var sb strings.Builder

	// Style definitions
	okStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("2")) // Green
	warnStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("208")) // Yellow/Orange
	errStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("9")) // Red
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).Background(lipgloss.Color("63")).Padding(0, 1)

	sb.WriteString(headerStyle.Render(" REAL-TIME PORT DIAGNOSTICS (CLIENT-SIDE ANALYSES) ") + "\n\n")

	// 1. Control Plane check (TCP 2377)
	sb.WriteString("● TCP 2377 (Swarm Control Plane): ")
	if unreachableManagers > 0 {
		sb.WriteString(errStyle.Render("UNREACHABLE / BLOCKED") + fmt.Sprintf(" (Warning: %d managers are unreachable. Verify port TCP 2377 is open inbound!)\n", unreachableManagers))
	} else if reachableManagers > 0 || (managers == 1 && readyNodes > 0) {
		sb.WriteString(okStyle.Render("HEALTHY / ONLINE") + fmt.Sprintf(" (Connected. Reachable managers: %d/%d)\n", reachableManagers, managers))
	} else {
		sb.WriteString(warnStyle.Render("UNKNOWN / SINGLE NODE") + " (Gossip and leader detection status unavailable)\n")
	}

	// 2. Gossip network check (TCP/UDP 7946)
	sb.WriteString("● TCP/UDP 7946 (Gossip Node Sync): ")
	if downNodes == 0 {
		sb.WriteString(okStyle.Render("HEALTHY") + fmt.Sprintf(" (All %d nodes are Ready. Gossip state sync and node discovery active)\n", totalNodes))
	} else {
		sb.WriteString(errStyle.Render("DEGRADED / DISRUPTED") + fmt.Sprintf(" (Warning: %d/%d nodes are down/offline. Verify TCP/UDP 7946 is open between all nodes!)\n", downNodes, totalNodes))
	}

	// 3. Overlay VXLAN check (UDP 4789)
	sb.WriteString("● UDP 4789 (Overlay VXLAN Network): ")
	sb.WriteString(warnStyle.Render("UNVERIFIABLE CLIENT-SIDE") + " (Overlay network communication is only active during container task traffic. Verify UDP 4789 is open to avoid container communication drops across nodes.)\n")

	return sb.String()
}
