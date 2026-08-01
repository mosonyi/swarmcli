// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package configsview

import (
	"fmt"
	"github.com/Eldara-Tech/swarmcli/ui"
	"github.com/Eldara-Tech/swarmcli/ui/components/errordialog"
	"github.com/Eldara-Tech/swarmcli/ui/dialog"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/docker/docker/api/types/swarm"
)

type configItem struct {
	Name      string
	ID        string
	CreatedAt time.Time
	UpdatedAt time.Time
	Labels    map[string]string
	Used      bool // true if used by any service
	UsedKnown bool // true if Used has been computed (false => loading/unknown)
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

type usedByItem struct {
	StackName   string
	ServiceName string
}

func (i usedByItem) FilterValue() string { return i.StackName + " " + i.ServiceName }
func (i usedByItem) Title() string       { return fmt.Sprintf("%-24s %-24s", i.StackName, i.ServiceName) }
func (i usedByItem) Description() string { return "Service: " + i.ServiceName }

// configItemFromSwarm builds a list row for one config. usedBy is the
// config-to-services index from docker.ServicesUsingConfigs, looked up by both
// ID and name because a service reference may carry either. A nil index means
// nothing references this config — which is the case for one just created, and
// is why this no longer reaches for the daemon itself.
func configItemFromSwarm(c swarm.Config, usedBy map[string][]swarm.Service) configItem {
	return configItem{
		Name:      c.Spec.Name,
		ID:        c.ID,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
		Labels:    c.Spec.Labels,
		Used:      len(usedBy[c.ID]) > 0 || len(usedBy[c.Spec.Name]) > 0,
		UsedKnown: true,
	}
}

func (m *Model) FrameTitle() string {
	if m.usedByViewActive {
		return fmt.Sprintf("Config: %s - Used By Stacks (%d)", m.usedByConfigName, len(m.usedByList.Filtered))
	}
	return fmt.Sprintf("Docker Configs (%d)", len(m.configsList.Filtered))
}

func (m *Model) FrameHeader() string {
	if m.usedByViewActive {
		return m.usedByList.RenderHeader()
	}
	return m.configsList.RenderHeader()
}

func (m *Model) FrameFooter() string {
	if m.usedByViewActive {
		return m.usedByList.RenderFooter()
	}
	return m.configsList.RenderFooter()
}

func (m *Model) FrameContent() string {
	if m.usedByViewActive {
		header := m.usedByList.RenderHeader()
		footer := m.usedByList.RenderFooter()
		frame := ui.ComputeFrameDimensions(
			m.usedByList.Viewport.Width, m.usedByList.Viewport.Height,
			m.width, m.height, header, footer,
		)
		return m.usedByList.VisibleContent(frame.DesiredContentLines)
	}

	width := 80
	if m.configsList.Viewport.Width > 0 {
		width = m.configsList.Viewport.Width
	} else if m.width > 0 {
		width = m.width
	}

	header := m.configsList.RenderHeader()
	footer := m.configsList.RenderFooter()
	frame := ui.ComputeFrameDimensions(
		m.configsList.Viewport.Width, m.configsList.Viewport.Height,
		m.width, m.height, header, footer,
	)
	content := m.configsList.VisibleContent(frame.DesiredContentLines)

	if m.fileBrowserActive {
		fileBrowserDialog := ui.RenderFileBrowserDialog("Select File", m.fileBrowserPath, m.fileBrowserFiles, m.fileBrowserCursor)
		content = ui.OverlayCentered(content, fileBrowserDialog, width, 0)
	} else if m.createDialogActive {
		createDialog := m.renderCreateDialog()
		content = ui.OverlayCentered(content, createDialog, width, 0)
	} else if m.confirmDialog.Visible {
		// Dismiss-only Info/Error dialogs render via the component so the footer
		// reads "Enter/Esc Close"; ui.RenderConfirmDialog is y/n-only and
		// mode-blind, so it would show a y/n prompt that does nothing here.
		var dialogView string
		if m.confirmDialog.InfoMode || m.confirmDialog.ErrorMode {
			dialogView = m.confirmDialog.View()
		} else {
			dialogView = ui.RenderConfirmDialog(m.confirmDialog.Message)
		}
		content = ui.OverlayCentered(content, dialogView, width, 0)
	} else if m.errorDialogActive {
		errorDialog := errordialog.Render(fmt.Sprintf("%v", m.err))
		content = ui.OverlayCentered(content, errorDialog, width, 0)
	} else if m.state == stateLoading || m.loadingView.Visible() {
		loadingView := m.loadingView.View()
		content = ui.OverlayCentered(content, loadingView, width, 0)
	}

	return content
}

func (m *Model) View() string {
	if m.usedByViewActive {
		return m.renderUsedByView()
	}
	return ui.RenderViewFrame(m.FrameTitle(), m.FrameHeader(), m.FrameContent(), m.FrameFooter(),
		m.configsList.Viewport.Width, m.configsList.Viewport.Height, false)
}

func (m *Model) renderCreateDialog() string {
	var lines []string

	switch m.createDialogStep {
	case "source":
		lines = append(lines, dialog.TitleStyle.Render(" Create Config - Choose Source "))
		lines = append(lines, dialog.ItemStyle.Render(""))
		lines = append(lines, dialog.ItemStyle.Render("How would you like to create the config?"))
		lines = append(lines, dialog.ItemStyle.Render(""))

		if m.createConfigSource == "file" {
			lines = append(lines, dialog.SelectedStyle.Render("→ From file"))
		} else {
			lines = append(lines, dialog.ItemStyle.Render("  From file"))
		}

		if m.createConfigSource == "inline" {
			lines = append(lines, dialog.SelectedStyle.Render("→ Inline editor"))
		} else {
			lines = append(lines, dialog.ItemStyle.Render("  Inline editor"))
		}

		lines = append(lines, dialog.ItemStyle.Render(""))
		helpText := fmt.Sprintf(" %s Select • %s / %s Navigate • %s Cancel",
			dialog.KeyStyle.Render("<Enter>"),
			dialog.KeyStyle.Render("<↑>"),
			dialog.KeyStyle.Render("<↓>"),
			dialog.KeyStyle.Render("<Esc>"))
		lines = append(lines, dialog.HelpStyle.Render(helpText))

	case "details-file":
		lines = append(lines, dialog.TitleStyle.Render(" Create Config from File "))
		lines = append(lines, dialog.ItemStyle.Render(""))

		// Show error if present
		if m.createDialogError != "" {
			errorStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("196")).
				Padding(0, 1)
			lines = append(lines, errorStyle.Render("⚠ "+m.createDialogError))
			lines = append(lines, dialog.ItemStyle.Render(""))
		}

		lines = append(lines, dialog.ItemStyle.Render(m.createNameInput.View()))
		lines = append(lines, dialog.ItemStyle.Render(""))

		// Show file path input with browse indicator. The key is a chord, so it
		// works from any focus and the hint is always shown.
		fileLine := m.createFileInput.View() + "  " + dialog.KeyStyle.Render(dialog.BrowseHint)
		lines = append(lines, dialog.ItemStyle.Render(fileLine))
		lines = append(lines, dialog.ItemStyle.Render(""))

		// Show labels input
		lines = append(lines, dialog.ItemStyle.Render(m.createLabelsInput.View()))
		lines = append(lines, dialog.ItemStyle.Render(""))

		// Change help text based on error state
		var helpText string
		if m.createDialogError != "" {
			helpText = fmt.Sprintf(" %s Fix error • %s Navigate • %s Cancel",
				dialog.KeyStyle.Render("<Enter>"),
				dialog.KeyStyle.Render("<Tab>"),
				dialog.KeyStyle.Render("<Esc>"))
		} else {
			helpText = fmt.Sprintf(" %s Confirm • %s Navigate • %s Cancel",
				dialog.KeyStyle.Render("<Enter>"),
				dialog.KeyStyle.Render("<Tab>"),
				dialog.KeyStyle.Render("<Esc>"))
		}
		lines = append(lines, dialog.HelpStyle.Render(helpText))

	case "details-inline":
		lines = append(lines, dialog.TitleStyle.Render(" Create Config - Inline Editor "))
		lines = append(lines, dialog.ItemStyle.Render(""))

		// Show error if present
		if m.createDialogError != "" {
			errorStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("196")).
				Padding(0, 1)
			lines = append(lines, errorStyle.Render("⚠ "+m.createDialogError))
			lines = append(lines, dialog.ItemStyle.Render(""))
		}

		lines = append(lines, dialog.ItemStyle.Render(m.createNameInput.View()))
		lines = append(lines, dialog.ItemStyle.Render(""))

		// Show editor status with edit hint when focused
		editorStatus := "Content: "
		if m.createConfigData != "" {
			editorStatus += fmt.Sprintf("(%d bytes)", len(m.createConfigData))
		} else {
			editorStatus += "(empty)"
		}
		if m.createInputFocus == 1 {
			editorStatus += "  " + dialog.KeyStyle.Render("[e: Edit]")
		}
		lines = append(lines, dialog.ItemStyle.Render(editorStatus))
		lines = append(lines, dialog.ItemStyle.Render(""))

		// Show labels input
		lines = append(lines, dialog.ItemStyle.Render(m.createLabelsInput.View()))
		lines = append(lines, dialog.ItemStyle.Render(""))

		// Change help text based on error state
		var helpText string
		if m.createDialogError != "" {
			helpText = fmt.Sprintf(" %s Fix error • %s Navigate • %s Cancel",
				dialog.KeyStyle.Render("<Enter>"),
				dialog.KeyStyle.Render("<Tab>"),
				dialog.KeyStyle.Render("<Esc>"))
		} else {
			helpText = fmt.Sprintf(" %s Confirm • %s Navigate • %s Cancel",
				dialog.KeyStyle.Render("<Enter>"),
				dialog.KeyStyle.Render("<Tab>"),
				dialog.KeyStyle.Render("<Esc>"))
		}
		lines = append(lines, dialog.HelpStyle.Render(helpText))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return dialog.BorderStyle.Render(content)
}

func (m *Model) renderUsedByView() string {
	// Safety check - if list is not properly initialized, show error
	if m.usedByList.Viewport.Width == 0 {
		m.usedByViewActive = false
		return "Error: UsedBy view not properly initialized"
	}

	title := fmt.Sprintf("Config: %s - Used By Stacks (%d)", m.usedByConfigName, len(m.usedByList.Filtered))
	framed, _ := m.usedByList.RenderFramedView(title)
	return framed
}

// formatLabels formats labels map to sorted key=value string
func formatLabels(labels map[string]string) string {
	if len(labels) == 0 {
		return "-"
	}

	var parts []string
	for k, v := range labels {
		// Skip internal swarmcli labels
		if !strings.HasPrefix(k, "swarmcli.") {
			parts = append(parts, fmt.Sprintf("%s=%s", k, v))
		}
	}
	// Sort for consistent display
	sort.Strings(parts)
	if len(parts) == 0 {
		return "-"
	}
	return strings.Join(parts, ",")
}
