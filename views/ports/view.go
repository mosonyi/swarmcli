// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package portsview

import (
	"fmt"
	"strings"
	"swarmcli/docker"
	"swarmcli/ui"
	"time"

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

	// ── Style tokens ──────────────────────────────────────────────────────────
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("81"))
	infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("246"))
	portHeaderStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("208"))
	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("117")).Width(12)
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("255"))

	okStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("2"))    // green
	warnStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("208")) // amber
	errStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("9"))   // red
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	cardWidth := m.viewport.Width - 6
	if cardWidth < 40 {
		cardWidth = 40
	}
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(0, 2).
		MarginBottom(1).
		Width(cardWidth)

	wrapVal := func(val string, labelW int) string {
		maxValW := cardWidth - labelW - 6
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

	// ── Intro ─────────────────────────────────────────────────────────────────
	b.WriteString(titleStyle.Render("Required Swarm Cluster Ports (node-to-node)"))
	b.WriteString("\n")
	b.WriteString(infoStyle.Render("These ports must be open between all Swarm nodes to ensure cluster coordination, node health state sync, and container overlay networking."))
	b.WriteString("\n\n")

	// ── Port reference cards ──────────────────────────────────────────────────
	port1Content := fmt.Sprintf("%s%s\n%s%s\n%s%s",
		labelStyle.Render("Protocol:"), valueStyle.Render(wrapVal("TCP", 12)),
		labelStyle.Render("Used For:"), valueStyle.Render(wrapVal("Swarm control plane (manager election, cluster commands)", 12)),
		labelStyle.Render("Direction:"), valueStyle.Render(wrapVal("Managers only (inbound to managers)", 12)),
	)
	b.WriteString(borderStyle.Render(fmt.Sprintf("%s\n\n%s", portHeaderStyle.Render("TCP 2377 — Cluster management"), port1Content)))
	b.WriteString("\n")

	port2Content := fmt.Sprintf("%s%s\n%s%s\n%s%s",
		labelStyle.Render("Protocol:"), valueStyle.Render(wrapVal("TCP / UDP", 12)),
		labelStyle.Render("Used For:"), valueStyle.Render(wrapVal("Gossip network (node discovery, membership, state sync)", 12)),
		labelStyle.Render("Direction:"), valueStyle.Render(wrapVal("All nodes to all nodes", 12)),
	)
	b.WriteString(borderStyle.Render(fmt.Sprintf("%s\n\n%s", portHeaderStyle.Render("TCP / UDP 7946 — Node communication"), port2Content)))
	b.WriteString("\n")

	port3Content := fmt.Sprintf("%s%s\n%s%s\n%s%s",
		labelStyle.Render("Protocol:"), valueStyle.Render(wrapVal("UDP", 12)),
		labelStyle.Render("Used For:"), valueStyle.Render(wrapVal("Overlay network (VXLAN) container-to-container traffic", 12)),
		labelStyle.Render("Direction:"), valueStyle.Render(wrapVal("All nodes to all nodes", 12)),
	)
	b.WriteString(borderStyle.Render(fmt.Sprintf("%s\n\n%s", portHeaderStyle.Render("UDP 4789 — Overlay network (VXLAN)"), port3Content)))
	b.WriteString("\n\n")

	// ── Live probe results ────────────────────────────────────────────────────
	sectionHeader := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("63")).
		Padding(0, 1)

	m.probeMu.RLock()
	results := m.probeResults
	probing := m.probing
	lastAt := m.lastProbeAt
	m.probeMu.RUnlock()

	// Header line
	probeAge := ""
	if !lastAt.IsZero() {
		probeAge = fmt.Sprintf("  last probe: %s ago", formatDuration(time.Since(lastAt)))
	}
	if probing {
		b.WriteString(sectionHeader.Render(" LIVE PORT PROBES — probing… "))
	} else {
		b.WriteString(sectionHeader.Render(" LIVE PORT PROBES — press r to re-probe"+probeAge+" "))
	}
	b.WriteString("\n\n")

	if len(results) == 0 {
		if probing {
			b.WriteString(dimStyle.Render("  Probing nodes — please wait…"))
		} else {
			b.WriteString(dimStyle.Render("  No probe data yet. Probes run automatically every 10 s, or press r."))
		}
		b.WriteString("\n")
	} else {
		b.WriteString(m.renderProbeTable(results, okStyle, warnStyle, errStyle, dimStyle, cardWidth))
	}

	b.WriteString("\n")

	// ── Swarm-state diagnostics (indirect) ────────────────────────────────────
	diag := m.getDiagnosticStatus()
	if diag != "" {
		b.WriteString(diag)
		b.WriteString("\n")
	}

	m.viewport.SetContent(b.String())
}

// renderProbeTable builds a compact per-node, per-port result table.
func (m *Model) renderProbeTable(
	results []docker.NodeProbeResult,
	okStyle, warnStyle, errStyle, dimStyle lipgloss.Style,
	tableWidth int,
) string {
	var b strings.Builder

	// Column widths
	colHost := 18
	colAddr := 16
	colPort := 12

	hdrStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("117"))
	sep := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("│")

	header := fmt.Sprintf("  %-*s %s %-*s %s %-*s %s %-*s %s %-*s",
		colHost, hdrStyle.Render("HOSTNAME"),
		sep,
		colAddr, hdrStyle.Render("ADDRESS"),
		sep,
		colPort, hdrStyle.Render("TCP 2377"),
		sep,
		colPort, hdrStyle.Render("TCP 7946"),
		sep,
		colPort, hdrStyle.Render("UDP 7946"),
	)
	b.WriteString(header)
	b.WriteString("\n")
	b.WriteString("  " + strings.Repeat("─", min(tableWidth-4, 80)))
	b.WriteString("\n")

	for _, r := range results {
		hostname := truncate(r.Hostname, colHost)
		addr := r.Addr
		if addr == "" {
			addr = "—"
		}
		addr = truncate(addr, colAddr)

		// If Docker reports this node as not ready, the probe timeout is caused
		// by a dead host — not a firewall. Show NODE DOWN across all columns so
		// users can distinguish a stopped node from a live node with blocked ports.
		nodeDown := r.NodeState != "" && r.NodeState != "ready"
		downCell := errStyle.Render("● NODE DOWN")

		var tcp2377, tcp7946, udp7946 string
		if nodeDown {
			tcp2377 = downCell
			tcp7946 = downCell
			udp7946 = downCell
		} else {
			tcp2377 = renderStatus(r.TCP2377, okStyle, warnStyle, errStyle, dimStyle)
			tcp7946 = renderStatus(r.TCP7946, okStyle, warnStyle, errStyle, dimStyle)
			udp7946 = renderStatus(r.UDP7946, okStyle, warnStyle, errStyle, dimStyle)
		}

		row := fmt.Sprintf("  %-*s %s %-*s %s %-*s %s %-*s %s %-*s",
			colHost, hostname,
			sep,
			colAddr, addr,
			sep,
			colPort, tcp2377,
			sep,
			colPort, tcp7946,
			sep,
			colPort, udp7946,
		)
		b.WriteString(row)
		b.WriteString("\n")
	}

	// UDP 4789 note
	b.WriteString("\n")
	note := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(
		"  UDP 4789 (VXLAN) is handled by the kernel's networking stack and cannot be\n" +
			"  probed from userspace without raw socket privileges (CAP_NET_RAW).")
	b.WriteString(note)
	b.WriteString("\n")

	return b.String()
}

func renderStatus(s docker.PortStatus, ok, warn, err, dim lipgloss.Style) string {
	switch s {
	case docker.PortOpen:
		return ok.Render("● OPEN    ")
	case docker.PortRefused:
		return err.Render("● CLOSED  ")
	case docker.PortFiltered:
		return warn.Render("● FILTERED")
	default:
		return dim.Render("  UNKNOWN ")
	}
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
	okStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("2"))
	warnStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("208"))
	errStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("9"))
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).Background(lipgloss.Color("63")).Padding(0, 1)

	sb.WriteString(headerStyle.Render(" SWARM-STATE DIAGNOSTICS (inferred from Docker daemon) ") + "\n\n")

	// TCP 2377
	sb.WriteString("● TCP 2377 (Swarm Control Plane): ")
	if unreachableManagers > 0 {
		sb.WriteString(errStyle.Render("UNREACHABLE / BLOCKED") + fmt.Sprintf(" (%d managers unreachable — verify TCP 2377 inbound!)\n", unreachableManagers))
	} else if reachableManagers > 0 || (managers == 1 && readyNodes > 0) {
		sb.WriteString(okStyle.Render("HEALTHY / ONLINE") + fmt.Sprintf(" (reachable managers: %d/%d)\n", reachableManagers, managers))
	} else {
		sb.WriteString(warnStyle.Render("UNKNOWN / SINGLE NODE") + " (leader detection unavailable)\n")
	}

	// TCP/UDP 7946
	sb.WriteString("● TCP/UDP 7946 (Gossip Node Sync): ")
	if downNodes == 0 {
		sb.WriteString(okStyle.Render("HEALTHY") + fmt.Sprintf(" (all %d nodes ready)\n", totalNodes))
	} else {
		sb.WriteString(errStyle.Render("DEGRADED / DISRUPTED") + fmt.Sprintf(" (%d/%d nodes down — verify TCP/UDP 7946 open between all nodes!)\n", downNodes, totalNodes))
	}

	// UDP 4789
	sb.WriteString("● UDP 4789 (Overlay VXLAN Network): ")
	sb.WriteString(warnStyle.Render("UNVERIFIABLE CLIENT-SIDE") + " (verify UDP 4789 is open to avoid container communication drops)\n")

	_ = readyNodes // used indirectly through totalNodes and downNodes
	return sb.String()
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-1] + "…"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
}
