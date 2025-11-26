package configsview

import (
	"fmt"
	"swarmcli/ui"
	filterlist "swarmcli/ui/components/filterable/list"

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

	// Compute max widths
	nameCol := len("NAME")
	idCol := len("ID")
	for _, cfg := range items {
		if len(cfg.Name) > nameCol {
			nameCol = len(cfg.Name)
		}
		if len(cfg.ID) > idCol {
			idCol = len(cfg.ID)
		}
	}

	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("75")). // blueish tone
		Bold(true)
	return headerStyle.Render(fmt.Sprintf("%-*s  %-*s", nameCol, "NAME", idCol, "ID"))
}
func (m *Model) renderConfigsFooter() string {
	status := fmt.Sprintf("Config %d of %d", m.configsList.Cursor+1, len(m.configsList.Filtered))
	statusBar := ui.StatusBarStyle.Render(status)

	var footer string
	if m.configsList.Mode == filterlist.ModeSearching {
		footer = ui.StatusBarStyle.Render("Filter (type then Enter): " + m.configsList.Query)
	} else if m.configsList.Query != "" {
		footer = ui.StatusBarStyle.Render("Filter: " + m.configsList.Query)
	}

	if footer != "" {
		return statusBar + "\n" + footer
	}
	return statusBar
}
