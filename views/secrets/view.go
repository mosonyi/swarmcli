// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package secretsview

import (
	"fmt"
	"github.com/Eldara-Tech/swarmcli/v2/ui"
	"github.com/Eldara-Tech/swarmcli/v2/ui/components/errordialog"
	"github.com/Eldara-Tech/swarmcli/v2/ui/dialog"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/docker/docker/api/types/swarm"
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

// secretItemFromSwarm builds a list row for one secret. usedBy is the
// secret-to-services index from docker.ServicesUsingSecrets, looked up by both
// ID and name because a service reference may carry either. A nil index means
// nothing references this secret — which is the case for one just created, and
// is why this no longer reaches for the daemon itself.
func secretItemFromSwarm(s swarm.Secret, usedBy map[string][]swarm.Service) secretItem {
	return secretItem{
		Name:      s.Spec.Name,
		ID:        s.ID,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
		Labels:    s.Spec.Labels,
		Used:      len(usedBy[s.ID]) > 0 || len(usedBy[s.Spec.Name]) > 0,
		UsedKnown: true,
	}
}

func (m *Model) FrameTitle() string {
	if m.usedByViewActive {
		return fmt.Sprintf("Secret: %s - Used By Stacks (%d)", m.usedBySecretName, len(m.usedByList.Filtered))
	}
	return fmt.Sprintf("Docker Secrets (%d)", len(m.secretsList.Filtered))
}

func (m *Model) FrameHeader() string {
	if m.usedByViewActive {
		return m.usedByList.RenderHeader()
	}
	return m.secretsList.RenderHeader()
}

func (m *Model) FrameFooter() string {
	if m.usedByViewActive {
		return m.usedByList.RenderFooter()
	}
	return m.secretsList.RenderFooter()
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

	header := m.secretsList.RenderHeader()
	footer := m.secretsList.RenderFooter()
	frame := ui.ComputeFrameDimensions(
		m.secretsList.Viewport.Width, m.secretsList.Viewport.Height,
		m.width, m.height, header, footer,
	)
	width := frame.FrameWidth
	content := m.secretsList.VisibleContent(frame.DesiredContentLines)

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

	return content
}

func (m *Model) View() string {
	if m.usedByViewActive {
		return m.renderUsedByView()
	}
	return ui.RenderViewFrame(m.FrameTitle(), m.FrameHeader(), m.FrameContent(), m.FrameFooter(),
		m.secretsList.Viewport.Width, m.secretsList.Viewport.Height, false)
}

func (m *Model) renderCreateDialog() string {
	var lines []string

	switch m.createDialogStep {
	case "source":
		lines = append(lines, dialog.TitleStyle.Render(" Create Secret - Choose Source "))
		lines = append(lines, dialog.ItemStyle.Render(""))
		lines = append(lines, dialog.ItemStyle.Render("How would you like to create the secret?"))
		lines = append(lines, dialog.ItemStyle.Render(""))

		if m.createSecretSource == "file" {
			lines = append(lines, dialog.SelectedStyle.Render("→ From file"))
		} else {
			lines = append(lines, dialog.ItemStyle.Render("  From file"))
		}

		if m.createSecretSource == "inline" {
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
		lines = append(lines, dialog.TitleStyle.Render(" Create Secret from File "))
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

		// Show encode toggle
		encodeText := "Encode secret: "
		if m.createEncodeSecret {
			encodeText += "ON"
		} else {
			encodeText += "OFF"
		}
		if m.createInputFocus == 3 {
			encodeText = dialog.SelectedStyle.Render(encodeText)
		} else {
			encodeText = dialog.ItemStyle.Render(encodeText)
		}
		lines = append(lines, encodeText)
		lines = append(lines, dialog.ItemStyle.Render("  (Base64-encode the value. Leave off unless the consumer decodes it.)"))
		lines = append(lines, dialog.ItemStyle.Render(""))

		// Change help text based on error state
		var helpText string
		if m.createDialogError != "" {
			helpText = fmt.Sprintf(" %s Fix error • %s Navigate • %s Toggle • %s Cancel",
				dialog.KeyStyle.Render("<Enter>"),
				dialog.KeyStyle.Render("<Tab>"),
				dialog.KeyStyle.Render("<Space>"),
				dialog.KeyStyle.Render("<Esc>"))
		} else {
			helpText = fmt.Sprintf(" %s Confirm • %s Navigate • %s Toggle • %s Cancel",
				dialog.KeyStyle.Render("<Enter>"),
				dialog.KeyStyle.Render("<Tab>"),
				dialog.KeyStyle.Render("<Space>"),
				dialog.KeyStyle.Render("<Esc>"))
		}
		lines = append(lines, dialog.HelpStyle.Render(helpText))

	case "details-inline":
		lines = append(lines, dialog.TitleStyle.Render(" Create Secret - Inline Editor "))
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
		if m.createSecretData != "" {
			editorStatus += fmt.Sprintf("(%d bytes)", len(m.createSecretData))
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

		// Show encode toggle
		encodeText := "Encode secret: "
		if m.createEncodeSecret {
			encodeText += "ON"
		} else {
			encodeText += "OFF"
		}
		if m.createInputFocus == 3 {
			encodeText = dialog.SelectedStyle.Render(encodeText)
		} else {
			encodeText = dialog.ItemStyle.Render(encodeText)
		}
		lines = append(lines, encodeText)
		lines = append(lines, dialog.ItemStyle.Render("  (Base64-encode the value. Leave off unless the consumer decodes it.)"))
		lines = append(lines, dialog.ItemStyle.Render(""))

		// Change help text based on error state
		var helpText string
		if m.createDialogError != "" {
			helpText = fmt.Sprintf(" %s Fix error • %s Navigate • %s Toggle • %s Cancel",
				dialog.KeyStyle.Render("<Enter>"),
				dialog.KeyStyle.Render("<Tab>"),
				dialog.KeyStyle.Render("<Space>"),
				dialog.KeyStyle.Render("<Esc>"))
		} else {
			helpText = fmt.Sprintf(" %s Confirm • %s Navigate • %s Toggle • %s Cancel",
				dialog.KeyStyle.Render("<Enter>"),
				dialog.KeyStyle.Render("<Tab>"),
				dialog.KeyStyle.Render("<Space>"),
				dialog.KeyStyle.Render("<Esc>"))
		}
		lines = append(lines, dialog.HelpStyle.Render(helpText))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return dialog.BorderStyle.Render(content)
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
