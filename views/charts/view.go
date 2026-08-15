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

	lines := m.contentLines()
	m.adjustOffsetForChild(lines)

	content := m.list.VisibleContent(lines)
	switch {
	case m.state == stateLoading && len(m.list.Items) == 0:
		content = padTo([]string{"Loading..."}, lines)
	case m.state == stateReady && len(m.list.Items) == 0:
		content = padTo(emptyStateLines, lines)
	}

	if m.errorDialogActive {
		errorDialog := errordialog.Render(fmt.Sprintf("%v", m.err))
		content = ui.OverlayCentered(content, errorDialog, width, 0)
	} else if m.confirmDialog.Visible {
		content = ui.OverlayCentered(content, m.confirmDialog.View(), width, 0)
	}
	return content
}

// contentLines is how many rows the list itself gets, once the frame, the
// column header and the footer have taken theirs. Both the renderer and
// paging read it, so a page is exactly what a screen shows.
func (m *Model) contentLines() int {
	frame := ui.ComputeFrameDimensions(
		m.list.Viewport.Width, m.list.Viewport.Height,
		m.width, m.height, m.FrameHeader(), m.FrameFooter(),
	)
	if frame.DesiredContentLines < 1 {
		return 1
	}
	return frame.DesiredContentLines
}

// adjustOffsetForChild keeps the selected child on screen.
//
// VisibleContent's own scrolling treats an expanded release as a single cursor
// unit, so a release taller than the viewport is shown from its first line and
// a child further down falls off the bottom. While a child is selected the
// view therefore takes the offset over itself, exactly as the services view
// does for its inline task rows.
func (m *Model) adjustOffsetForChild(visibleLines int) {
	if m.childIndex == noChild {
		m.list.SkipOffsetAdjustment = false
		return
	}
	sel, ok := m.selected()
	if !ok || !m.expanded[sel.Name] {
		m.list.SkipOffsetAdjustment = false
		return
	}
	_, childLine := expansionBlock(sel, m.childIndex)
	if m.childIndex >= len(childLine) {
		m.list.SkipOffsetAdjustment = false
		return
	}
	m.list.SkipOffsetAdjustment = true

	// Lines above the selected release, then its own row, then the child's
	// line within the expansion block.
	offset := 0
	for i := 0; i < m.list.Cursor && i < len(m.list.Filtered); i++ {
		offset += m.itemLineCount(m.list.Filtered[i])
	}
	offset += 1 + childLine[m.childIndex]

	if visibleLines < 1 {
		visibleLines = 1
	}
	if offset < m.list.Viewport.YOffset {
		m.list.Viewport.YOffset = offset
	} else if offset >= m.list.Viewport.YOffset+visibleLines {
		m.list.Viewport.YOffset = offset - visibleLines + 1
		if m.list.Viewport.YOffset < 0 {
			m.list.Viewport.YOffset = 0
		}
	}
}

// itemLineCount is how many rendered lines an item occupies. It counts the
// expansion through the same function that renders it, so the two cannot
// disagree about how tall a release is.
func (m *Model) itemLineCount(it releaseItem) int {
	if !m.expanded[it.Name] {
		return 1
	}
	lines, _ := expansionBlock(it, noChild)
	return 1 + len(lines)
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
	return base + "\n" + ui.StatusBarStyle.Render(readOnlyHint)
}

// selectedReason is why the selected release is not converged. Convergence
// writes that sentence for display, and it is the whole value of the HEALTH
// column — the column says something is wrong, this says what.
func (m *Model) selectedReason() string {
	// The DETAIL column carries it for every row when the terminal is wide
	// enough; repeating the selected row's copy in the footer would be the same
	// sentence twice on the same screen.
	if m.hasDetailColumn() {
		return ""
	}
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
