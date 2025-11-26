package configsview

import (
	"fmt"
	"io"
	"swarmcli/ui"
	filterlist "swarmcli/ui/components/filterable/list"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/docker/docker/api/types/swarm"
)

type configItem struct {
	Name string
	ID   string
}

func (i configItem) FilterValue() string { return i.Name }
func (i configItem) Title() string       { return i.Name }
func (i configItem) Description() string { return fmt.Sprintf("ID: %s", i.ID) }

type itemDelegate struct{}

func (d itemDelegate) Height() int  { return 1 }
func (d itemDelegate) Spacing() int { return 0 }

func (d itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }

func (d itemDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	cfg := item.(configItem)
	selected := index == m.Index()
	if selected {
		fmt.Fprintf(w, "> %s", cfg.Name)
	} else {
		fmt.Fprintf(w, "  %s", cfg.Name)
	}
}

func configItemFromSwarm(c swarm.Config) configItem {
	return configItem{Name: c.Spec.Name, ID: c.ID}
}

func (m *Model) View() string {
	width := 80
	height := 24
	if m.configsList.Viewport.Width > 0 {
		width = m.configsList.Viewport.Width
	}
	if m.configsList.Viewport.Height > 0 {
		height = m.configsList.Viewport.Height
	}

	header := renderConfigsHeader(m.configsList.Items)
	content := m.configsList.View()
	footer := m.renderConfigsFooter()

	title := fmt.Sprintf("Docker Configs (%d)", len(m.configsList.Filtered))
	view := ui.RenderFramedBox(title, header, content, footer, width)

	if m.confirmDialog.Visible {
		view = ui.OverlayCentered(view, m.confirmDialog.View(), width, height)
	}

	if m.state == stateLoading || m.loadingView.Visible() {
		view = ui.OverlayCentered(view, m.loadingView.View(), width, height)
	}

	return view
}

func renderConfigsHeader(items []configItem) string {
	if len(items) == 0 {
		return "NAME       ID"
	}

	// compute column widths
	colWidths := map[string]int{
		"Name": len("NAME"),
		"ID":   len("ID"),
	}

	for _, cfg := range items {
		if len(cfg.Name) > colWidths["Name"] {
			colWidths["Name"] = len(cfg.Name)
		}
		if len(cfg.ID) > colWidths["ID"] {
			colWidths["ID"] = len(cfg.ID)
		}
	}

	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("75")).Bold(true)

	return headerStyle.Render(fmt.Sprintf(
		"%-*s  %-*s",
		colWidths["Name"], "NAME",
		colWidths["ID"], "ID",
	))
}

func (m *Model) renderConfigsFooter() string {
	status := fmt.Sprintf("Config %d of %d", m.configsList.Cursor+1, len(m.configsList.Filtered))
	statusBar := ui.StatusBarStyle.Render(status)

	var footer string
	if m.configsList.Mode == filterlist.ModeSearching {
		footer = ui.StatusBarStyle.Render("Filter: " + m.configsList.Query)
	}

	if footer != "" {
		return statusBar + "\n" + footer
	}
	return statusBar
}
