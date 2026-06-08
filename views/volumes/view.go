// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package volumesview

import (
	"fmt"
	"strings"
	"time"

	"swarmcli/features"
	"swarmcli/ui"
	"swarmcli/ui/components/errordialog"
	"swarmcli/views/view"
)

// connectedNodeHint is shown in the footer while the all-nodes capability is
// unavailable, making the connected-node-only scope explicit.
var connectedNodeHint = "Connected node only · listing volumes on all nodes is a Business Edition feature: " + view.BELandingURL

func (m *Model) FrameTitle() string {
	return fmt.Sprintf("Docker Volumes (%d)", len(m.volumesList.Filtered))
}

func (m *Model) FrameHeader() string {
	return m.volumesList.RenderHeader()
}

func (m *Model) FrameFooter() string {
	return m.volumesList.RenderFooter()
}

func (m *Model) FrameContent() string {
	return m.buildMainContent()
}

func (m *Model) View() string {
	return ui.RenderViewFrame(m.FrameTitle(), m.FrameHeader(), m.FrameContent(), m.FrameFooter(),
		m.volumesList.Viewport.Width, m.volumesList.Viewport.Height, false)
}

func (m *Model) buildMainContent() string {
	width := 80
	if m.volumesList.Viewport.Width > 0 {
		width = m.volumesList.Viewport.Width
	} else if m.width > 0 {
		width = m.width
	}

	header := m.FrameHeader()
	footer := m.FrameFooter()
	frame := ui.ComputeFrameDimensions(
		m.volumesList.Viewport.Width, m.volumesList.Viewport.Height,
		m.width, m.height, header, footer,
	)

	content := m.volumesList.VisibleContent(frame.DesiredContentLines)
	if m.state == stateLoading && len(m.volumesList.Items) == 0 {
		lines := frame.DesiredContentLines
		if lines < 1 {
			lines = 1
		}
		parts := make([]string, lines)
		parts[0] = "Loading..."
		for i := 1; i < lines; i++ {
			parts[i] = ""
		}
		content = strings.Join(parts, "\n")
	}

	if m.errorDialogActive {
		errorDialog := errordialog.Render(fmt.Sprintf("%v", m.err))
		content = ui.OverlayCentered(content, errorDialog, width, 0)
	}
	return content
}

func (m *Model) renderVolumesFooter() string {
	base := m.baseFooter()
	if m.partialWarn != "" {
		base += "\n" + ui.StatusBarStyle.Render("⚠ "+m.partialWarn)
	}
	if features.IsEnabled(allNodesFeature) {
		return base
	}
	return base + "\n" + ui.StatusBarStyle.Render(connectedNodeHint)
}

func (m *Model) baseFooter() string {
	if m.toastMessage != "" && time.Now().Before(m.toastUntil) {
		return ui.StatusBarStyle.Render(m.toastMessage)
	}
	if m.toastMessage != "" && time.Now().After(m.toastUntil) {
		m.toastMessage = ""
	}

	if m.state == stateLoading {
		return ui.StatusBarStyle.Render(fmt.Sprintf("Loading Docker volumes… %s", ui.SpinnerCharAt(m.spinner)))
	}

	total := len(m.volumesList.Items)
	filtered := len(m.volumesList.Filtered)
	cursor := m.volumesList.Cursor + 1

	if filtered == 0 {
		return ui.StatusBarStyle.Render("No volumes")
	}
	if m.HasActiveFilter() {
		return ui.StatusBarStyle.Render(fmt.Sprintf("%d/%d (filtered from %d)", cursor, filtered, total))
	}
	return ui.StatusBarStyle.Render(fmt.Sprintf("%d/%d", cursor, total))
}
