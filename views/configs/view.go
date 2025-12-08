package configsview

import (
	"fmt"
	"strings"
	"swarmcli/ui"
	"swarmcli/ui/components/errordialog"
	filterlist "swarmcli/ui/components/filterable/list"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/docker/docker/api/types/swarm"
)

type configItem struct {
	Name      string
	ID        string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (i configItem) FilterValue() string { return i.Name }
func (i configItem) Title() string       { return i.Name }
func (i configItem) Description() string {
	createdStr := "N/A"
	if !i.CreatedAt.IsZero() {
		createdStr = i.CreatedAt.Format("2006-01-02 15:04:05")
	}
	updatedStr := "N/A"
	if !i.UpdatedAt.IsZero() {
		updatedStr = i.UpdatedAt.Format("2006-01-02 15:04:05")
	}
	return fmt.Sprintf("ID: %s        Created: %s        Updated: %s", i.ID, createdStr, updatedStr)
}

func configItemFromSwarm(c swarm.Config) configItem {
	return configItem{
		Name:      c.Spec.Name,
		ID:        c.ID,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

func (m *Model) View() string {
	width := 80
	if m.configsList.Viewport.Width > 0 {
		width = m.configsList.Viewport.Width
	}

	header := renderConfigsHeader(m.configsList.Items)
	content := m.configsList.View()
	footer := m.renderConfigsFooter()

	// Pad content to fill viewport height
	height := m.configsList.Viewport.Height
	if height <= 0 {
		height = 20
	}
	contentLines := strings.Split(content, "\n")
	// Account for frame borders (2), title (1), header (1) = 4 lines overhead
	availableLines := height - 4
	if availableLines < 0 {
		availableLines = 0
	}
	for len(contentLines) < availableLines {
		contentLines = append(contentLines, "")
	}
	paddedContent := strings.Join(contentLines, "\n")

	// Apply overlays to padded content BEFORE framing
	if m.confirmDialog.Visible {
		dialogView := ui.RenderConfirmDialog(m.confirmDialog.Message)
		paddedContent = ui.OverlayCentered(paddedContent, dialogView, width, 0)
	}

	if m.errorDialogActive {
		errorDialog := errordialog.Render(fmt.Sprintf("%v", m.err))
		paddedContent = ui.OverlayCentered(paddedContent, errorDialog, width, 0)
	}

	if m.state == stateLoading || m.loadingView.Visible() {
		loadingView := m.loadingView.View()
		paddedContent = ui.OverlayCentered(paddedContent, loadingView, width, 0)
	}

	// Add 4 to make frame full terminal width (app reduces viewport by 4 in normal mode)
	frameWidth := width + 4

	title := fmt.Sprintf("Docker Configs (%d)", len(m.configsList.Filtered))
	view := ui.RenderFramedBox(title, header, paddedContent, footer, frameWidth)

	return view
}

func renderConfigsHeader(items []configItem) string {
	if len(items) == 0 {
		return "NAME       ID               CREATED AT           UPDATED AT"
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
		Foreground(lipgloss.Color("15")). // white
		Bold(true)
	return headerStyle.Render(fmt.Sprintf("%-*s        %-*s        %-19s        %-19s", nameCol, "NAME", idCol, "ID", "CREATED AT", "UPDATED AT"))
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
