// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package chartsview

import (
	"fmt"
	"strings"
	"time"

	"github.com/Eldara-Tech/swarmcli/ui"
	"github.com/Eldara-Tech/swarmcli/ui/components/errordialog"
)

// emptyStateLines is what an operator sees when nothing is installed. Charts
// are an opt-in feature, so a bare empty box reads as a broken view rather
// than as "you have no releases".
var emptyStateLines = []string{
	"No chart releases found.",
	"",
	"Install one with:  swarmcli charts install <release> <repo/chart>",
}

func (m *Model) FrameTitle() string {
	return fmt.Sprintf("Chart Releases (%d)", len(m.list.Filtered))
}

func (m *Model) FrameHeader() string { return m.list.RenderHeader() }

func (m *Model) FrameFooter() string { return m.list.RenderFooter() }

func (m *Model) FrameContent() string { return m.buildMainContent() }

func (m *Model) View() string {
	return ui.RenderViewFrame(m.FrameTitle(), m.FrameHeader(), m.FrameContent(), m.FrameFooter(),
		m.list.Viewport.Width, m.list.Viewport.Height, false)
}

func (m *Model) buildMainContent() string {
	width := 80
	if m.list.Viewport.Width > 0 {
		width = m.list.Viewport.Width
	} else if m.width > 0 {
		width = m.width
	}

	frame := ui.ComputeFrameDimensions(
		m.list.Viewport.Width, m.list.Viewport.Height,
		m.width, m.height, m.FrameHeader(), m.FrameFooter(),
	)

	content := m.list.VisibleContent(frame.DesiredContentLines)
	switch {
	case m.state == stateLoading && len(m.list.Items) == 0:
		content = padTo([]string{"Loading..."}, frame.DesiredContentLines)
	case m.state == stateReady && len(m.list.Items) == 0:
		content = padTo(emptyStateLines, frame.DesiredContentLines)
	}

	if m.errorDialogActive {
		errorDialog := errordialog.Render(fmt.Sprintf("%v", m.err))
		content = ui.OverlayCentered(content, errorDialog, width, 0)
	}
	return content
}

// padTo renders lines into exactly n rows, so the frame does not collapse.
func padTo(lines []string, n int) string {
	if n < 1 {
		n = 1
	}
	parts := make([]string, n)
	for i := range parts {
		if i < len(lines) {
			parts[i] = lines[i]
		}
	}
	return strings.Join(parts, "\n")
}

// renderFooter is the status area: counts or a transient toast, then the
// convergence reason for the selected release.
func (m *Model) renderFooter() string {
	base := m.baseFooter()
	if reason := m.selectedReason(); reason != "" {
		base += "\n" + ui.StatusBarStyle.Render(reason)
	}
	return base
}

// selectedReason is why the selected release is not converged. Convergence
// writes that sentence for display, and it is the whole value of the HEALTH
// column — the column says something is wrong, this says what.
func (m *Model) selectedReason() string {
	sel, ok := m.selected()
	if !ok || !sel.HasHealth || sel.Health.Reason == "" {
		return ""
	}
	return sel.Health.Reason
}

func (m *Model) baseFooter() string {
	if m.toastMessage != "" && time.Now().Before(m.toastUntil) {
		return ui.StatusBarStyle.Render(m.toastMessage)
	}
	if m.toastMessage != "" && time.Now().After(m.toastUntil) {
		m.toastMessage = ""
	}

	if m.state == stateLoading {
		return ui.StatusBarStyle.Render(fmt.Sprintf("Loading chart releases… %s", ui.SpinnerCharAt(m.spinner)))
	}

	total := len(m.list.Items)
	filtered := len(m.list.Filtered)
	cursor := m.list.Cursor + 1

	if filtered == 0 {
		return ui.StatusBarStyle.Render("No releases")
	}
	if m.HasActiveFilter() {
		return ui.StatusBarStyle.Render(fmt.Sprintf("%d/%d (filtered from %d)", cursor, filtered, total))
	}
	return ui.StatusBarStyle.Render(fmt.Sprintf("%d/%d", cursor, total))
}
