// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package secretsview

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"swarmcli/docker"
	"swarmcli/ui"
	"swarmcli/ui/components/errordialog"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/docker/docker/api/types/swarm"
)

// Shared dialog styles
var (
	dialogTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("15")).
				Background(lipgloss.Color("63")).
				Padding(0, 1)

	dialogBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("117"))

	dialogItemStyle = lipgloss.NewStyle().
			Padding(0, 1)

	dialogSelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("15")).
				Background(lipgloss.Color("63")).
				Padding(0, 1)

	dialogHelpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Padding(0, 1)

	dialogKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("63")).
			Bold(true)
)

type secretItem struct {
	Name      string
	ID        string
	CreatedAt time.Time
	UpdatedAt time.Time
	Labels    map[string]string
	Used      bool // true if used by any service
	UsedKnown bool // true if Used has been computed (false => loading/unknown)
}

func (i secretItem) FilterValue() string { return i.Name }
func (i secretItem) Title() string       { return i.Name }
func (i secretItem) Description() string {
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

func secretItemFromSwarm(ctx context.Context, s swarm.Secret) secretItem {
	used := false
	services, err := docker.ListServicesUsingSecretID(ctx, s.ID)
	if err == nil && len(services) > 0 {
		used = true
	}
	return secretItem{
		Name:      s.Spec.Name,
		ID:        s.ID,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
		Labels:    s.Spec.Labels,
		Used:      used,
		UsedKnown: true,
	}
}

func (m *Model) View() string {
	// If in UsedBy view, render it instead of the main secrets view
	if m.usedByViewActive {
		return m.renderUsedByView()
	}

	header := m.secretsList.RenderHeader()
	footer := m.secretsList.RenderFooter()

	frame := ui.ComputeFrameDimensions(
		m.secretsList.Viewport.Width,
		m.secretsList.Viewport.Height,
		m.width,
		m.height,
		header,
		footer,
	)

	width := frame.FrameWidth
	content := m.secretsList.VisibleContent(frame.DesiredContentLines)

	// Apply overlays to content BEFORE framing
	if m.fileBrowserActive {
		fileBrowserDialog := ui.RenderFileBrowserDialog("Select File", m.fileBrowserPath, m.fileBrowserFiles, m.fileBrowserCursor)
		content = ui.OverlayCentered(content, fileBrowserDialog, width, 0)
	} else if m.createDialogActive {
		createDialog := m.renderCreateDialog()
		content = ui.OverlayCentered(content, createDialog, width, 0)
	} else if m.confirmDialog.Visible {
		dialogView := ui.RenderConfirmDialog(m.confirmDialog.Message)
		content = ui.OverlayCentered(content, dialogView, width, 0)
	} else if m.errorDialogActive {
		errorDialog := errordialog.Render(fmt.Sprintf("%v", m.err))
		content = ui.OverlayCentered(content, errorDialog, width, 0)
	} else if m.state == stateLoading || m.loadingView.Visible() {
		loadingView := m.loadingView.View()
		content = ui.OverlayCentered(content, loadingView, width, 0)
	}

	title := fmt.Sprintf("Docker Secrets (%d)", len(m.secretsList.Filtered))
	view := ui.RenderFramedBox(title, header, content, footer, frame.FrameWidth)

	return view
}

func (m *Model) renderCreateDialog() string {
	var lines []string

	switch m.createDialogStep {
	case "source":
		lines = append(lines, dialogTitleStyle.Render(" Create Secret - Choose Source "))
		lines = append(lines, dialogItemStyle.Render(""))
		lines = append(lines, dialogItemStyle.Render("How would you like to create the secret?"))
		lines = append(lines, dialogItemStyle.Render(""))

		if m.createSecretSource == "file" {
			lines = append(lines, dialogSelectedStyle.Render("→ From file"))
		} else {
			lines = append(lines, dialogItemStyle.Render("  From file"))
		}

		if m.createSecretSource == "inline" {
			lines = append(lines, dialogSelectedStyle.Render("→ Inline editor"))
		} else {
			lines = append(lines, dialogItemStyle.Render("  Inline editor"))
		}

		lines = append(lines, dialogItemStyle.Render(""))
		helpText := fmt.Sprintf(" %s Select • %s / %s Navigate • %s Cancel",
			dialogKeyStyle.Render("<Enter>"),
			dialogKeyStyle.Render("<↑>"),
			dialogKeyStyle.Render("<↓>"),
			dialogKeyStyle.Render("<Esc>"))
		lines = append(lines, dialogHelpStyle.Render(helpText))

	case "details-file":
		lines = append(lines, dialogTitleStyle.Render(" Create Secret from File "))
		lines = append(lines, dialogItemStyle.Render(""))

		// Show error if present
		if m.createDialogError != "" {
			errorStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("196")).
				Padding(0, 1)
			lines = append(lines, errorStyle.Render("⚠ "+m.createDialogError))
			lines = append(lines, dialogItemStyle.Render(""))
		}

		lines = append(lines, dialogItemStyle.Render(m.createNameInput.View()))
		lines = append(lines, dialogItemStyle.Render(""))

		// Show file path input with browse indicator when focused
		fileLine := m.createFileInput.View()
		if m.createInputFocus == 1 {
			fileLine += "  " + dialogKeyStyle.Render("[f: Browse]")
		}
		lines = append(lines, dialogItemStyle.Render(fileLine))
		lines = append(lines, dialogItemStyle.Render(""))

		// Show labels input
		lines = append(lines, dialogItemStyle.Render(m.createLabelsInput.View()))
		lines = append(lines, dialogItemStyle.Render(""))

		// Show encode toggle
		encodeText := "Encode secret: "
		if m.createEncodeSecret {
			encodeText += "ON"
		} else {
			encodeText += "OFF"
		}
		if m.createInputFocus == 3 {
			encodeText = dialogSelectedStyle.Render(encodeText)
		} else {
			encodeText = dialogItemStyle.Render(encodeText)
		}
		lines = append(lines, encodeText)
		lines = append(lines, dialogItemStyle.Render("  (Secrets need to be base64 encoded. Disable this if already encoded.)"))
		lines = append(lines, dialogItemStyle.Render(""))

		// Change help text based on error state
		var helpText string
		if m.createDialogError != "" {
			helpText = fmt.Sprintf(" %s Fix error • %s Navigate • %s Toggle • %s Cancel",
				dialogKeyStyle.Render("<Enter>"),
				dialogKeyStyle.Render("<Tab>"),
				dialogKeyStyle.Render("<Space>"),
				dialogKeyStyle.Render("<Esc>"))
		} else {
			helpText = fmt.Sprintf(" %s Confirm • %s Navigate • %s Toggle • %s Cancel",
				dialogKeyStyle.Render("<Enter>"),
				dialogKeyStyle.Render("<Tab>"),
				dialogKeyStyle.Render("<Space>"),
				dialogKeyStyle.Render("<Esc>"))
		}
		lines = append(lines, dialogHelpStyle.Render(helpText))

	case "details-inline":
		lines = append(lines, dialogTitleStyle.Render(" Create Secret - Inline Editor "))
		lines = append(lines, dialogItemStyle.Render(""))

		// Show error if present
		if m.createDialogError != "" {
			errorStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("196")).
				Padding(0, 1)
			lines = append(lines, errorStyle.Render("⚠ "+m.createDialogError))
			lines = append(lines, dialogItemStyle.Render(""))
		}

		lines = append(lines, dialogItemStyle.Render(m.createNameInput.View()))
		lines = append(lines, dialogItemStyle.Render(""))

		// Show editor status with edit hint when focused
		editorStatus := "Content: "
		if m.createSecretData != "" {
			editorStatus += fmt.Sprintf("(%d bytes)", len(m.createSecretData))
		} else {
			editorStatus += "(empty)"
		}
		if m.createInputFocus == 1 {
			editorStatus += "  " + dialogKeyStyle.Render("[e: Edit]")
		}
		lines = append(lines, dialogItemStyle.Render(editorStatus))
		lines = append(lines, dialogItemStyle.Render(""))

		// Show labels input
		lines = append(lines, dialogItemStyle.Render(m.createLabelsInput.View()))
		lines = append(lines, dialogItemStyle.Render(""))

		// Show encode toggle
		encodeText := "Encode secret: "
		if m.createEncodeSecret {
			encodeText += "ON"
		} else {
			encodeText += "OFF"
		}
		if m.createInputFocus == 3 {
			encodeText = dialogSelectedStyle.Render(encodeText)
		} else {
			encodeText = dialogItemStyle.Render(encodeText)
		}
		lines = append(lines, encodeText)
		lines = append(lines, dialogItemStyle.Render("  (Secrets need to be base64 encoded. Disable this if already encoded.)"))
		lines = append(lines, dialogItemStyle.Render(""))

		// Change help text based on error state
		var helpText string
		if m.createDialogError != "" {
			helpText = fmt.Sprintf(" %s Fix error • %s Navigate • %s Toggle • %s Cancel",
				dialogKeyStyle.Render("<Enter>"),
				dialogKeyStyle.Render("<Tab>"),
				dialogKeyStyle.Render("<Space>"),
				dialogKeyStyle.Render("<Esc>"))
		} else {
			helpText = fmt.Sprintf(" %s Confirm • %s Navigate • %s Toggle • %s Cancel",
				dialogKeyStyle.Render("<Enter>"),
				dialogKeyStyle.Render("<Tab>"),
				dialogKeyStyle.Render("<Space>"),
				dialogKeyStyle.Render("<Esc>"))
		}
		lines = append(lines, dialogHelpStyle.Render(helpText))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return dialogBorderStyle.Render(content)
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

// formatLabelsWithScroll formats labels with horizontal scroll offset and truncation indicator
func formatLabelsWithScroll(labels map[string]string, offset int, maxWidth int) string {
	full := formatLabels(labels)
	if full == "-" {
		return full
	}

	// Apply scroll offset
	if offset > len(full) {
		offset = len(full)
	}
	visible := full[offset:]

	// Truncate if needed and add > indicator
	if len(visible) > maxWidth {
		if maxWidth > 1 {
			visible = visible[:maxWidth-1] + ">"
		} else {
			visible = ">"
		}
	}

	return visible
}

func (m *Model) renderUsedByView() string {
	// Safety check - if list is not properly initialized, show error
	if m.usedByList.Viewport.Width == 0 {
		m.usedByViewActive = false
		return "Error: UsedBy view not properly initialized"
	}

	title := fmt.Sprintf("Secret: %s - Used By Stacks (%d)", m.usedBySecretName, len(m.usedByList.Filtered))
	framed, _ := m.usedByList.RenderFramedView(title)
	return framed
}
