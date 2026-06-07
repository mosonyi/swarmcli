// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package volumesview

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"swarmcli/core/primitives/hash"
	"swarmcli/ui"
	helpview "swarmcli/views/help"
	view "swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func truncateWithEllipsis(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if maxWidth <= 1 {
		return "…"
	}
	if lipgloss.Width(s) <= maxWidth {
		return s
	}
	if maxWidth == 2 {
		return s[:1] + "…"
	}
	return s[:maxWidth-1] + "…"
}

func formatCreated(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	return t.Format("2006-01-02 15:04")
}

// volumeColWidths allocates the six column widths as percentages of the
// available width, with the MOUNT POINT column absorbing rounding remainder.
func (m *Model) volumeColWidths(totalWidth int) []int {
	if totalWidth <= 0 {
		totalWidth = 80
	}
	const (
		sepLen  = 2
		numCols = 6
		flexCol = 3 // MOUNT POINT
	)
	effWidth := totalWidth - (numCols-1)*sepLen
	if effWidth < 30 {
		effWidth = totalWidth
	}

	weights := []int{20, 12, 9, 21, 24, 14} // NAME STACK DRIVER MOUNT CREATED HOST
	mins := []int{8, 5, 5, 10, 12, 5}

	widths := make([]int, numCols)
	sum := 0
	for i := 0; i < numCols; i++ {
		w := (effWidth * weights[i]) / 100
		if w < 1 {
			w = 1
		}
		widths[i] = w
		sum += w
	}
	widths[flexCol] += effWidth - sum

	for i := range widths {
		if widths[i] < mins[i] {
			widths[i] = mins[i]
		}
	}

	sum = 0
	for _, w := range widths {
		sum += w
	}
	if sum != effWidth {
		widths[flexCol] += effWidth - sum
		if widths[flexCol] < mins[flexCol] {
			widths[flexCol] = mins[flexCol]
		}
	}
	return widths
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case SpinnerTickMsg:
		m.spinner++
		if m.state == stateLoading {
			m.volumesList.Viewport.SetContent(m.volumesList.View())
		}
		return m.spinnerTickCmd()

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.volumesList.Viewport.Width = msg.Width
		m.volumesList.Viewport.Height = msg.Height
		m.volumesList.SetOuterSize(msg.Width, msg.Height)
		if m.firstResize {
			m.volumesList.Viewport.YOffset = 0
			m.firstResize = false
		} else if m.volumesList.Cursor == 0 {
			m.volumesList.Viewport.YOffset = 0
		}
		return nil

	case VolumesLoadedMsg:
		return m.handleVolumesLoaded(msg)

	case TickMsg:
		if m.visible && m.state == stateReady && !m.errorDialogActive {
			return tea.Batch(m.checkVolumesCmd(m.lastSnapshot), tickCmd())
		}
		return tickCmd()

	case PollRetryMsg:
		return tickCmd()

	case tea.KeyMsg:
		if m.errorDialogActive {
			if msg.String() == "enter" || msg.String() == "esc" {
				m.errorDialogActive = false
			}
			return nil
		}
		return m.handleNormalKeys(msg)
	}
	return nil
}

func (m *Model) handleVolumesLoaded(msg VolumesLoadedMsg) tea.Cmd {
	if msg.Err != nil {
		// Background refresh failure with existing data: keep current view.
		if m.state == stateReady && len(m.volumesList.Items) > 0 {
			l().Warnf("VolumesView: background refresh failed: %v", msg.Err)
			m.showToast("Refresh failed (will retry)")
			return nil
		}
		m.state = stateError
		m.err = msg.Err
		m.errorDialogActive = true
		return nil
	}

	// Persist (or clear) the non-fatal partial-failure banner.
	m.partialWarn = msg.Warn

	selectedName := ""
	if !m.resetCursorOnNextLoad && m.volumesList.Cursor < len(m.volumesList.Filtered) {
		selectedName = m.volumesList.Filtered[m.volumesList.Cursor].Name
	}

	if h, err := hash.Compute(stableVolumes(msg.Volumes)); err == nil {
		m.lastSnapshot = h
	} else {
		l().Errorf("VolumesView: Error computing hash: %v", err)
	}

	m.volumesList.Items = msg.Volumes
	m.volumesList.ApplyFilter()
	m.applySorting()

	if m.resetCursorOnNextLoad {
		m.volumesList.Cursor = 0
		m.volumesList.Viewport.YOffset = 0
		m.resetCursorOnNextLoad = false
	} else if selectedName != "" {
		for i, v := range m.volumesList.Filtered {
			if v.Name == selectedName {
				m.volumesList.Cursor = i
				break
			}
		}
	}

	m.state = stateReady
	m.volumesList.Viewport.SetContent(m.volumesList.View())
	return nil
}

func (m *Model) handleNormalKeys(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		// Volumes is a root view, no back navigation.
		return nil
	case "?":
		return func() tea.Msg {
			return view.NavigateToMsg{ViewName: helpview.ViewName, Payload: GetVolumesHelpContent()}
		}
	case "N":
		m.toggleSort(SortByName)
		return nil
	case "S":
		m.toggleSort(SortByStack)
		return nil
	case "D":
		m.toggleSort(SortByDriver)
		return nil
	case "C":
		m.toggleSort(SortByCreated)
		return nil
	case "H":
		m.toggleSort(SortByHost)
		return nil
	case "up", "k":
		if m.volumesList.Cursor > 0 {
			m.volumesList.Cursor--
			m.volumesList.Viewport.SetContent(m.volumesList.View())
		}
	case "down", "j":
		if m.volumesList.Cursor < len(m.volumesList.Filtered)-1 {
			m.volumesList.Cursor++
			m.volumesList.Viewport.SetContent(m.volumesList.View())
		}
	case "pgup":
		page := m.volumesList.Viewport.Height
		if page < 1 {
			page = 10
		}
		m.volumesList.Cursor -= page
		if m.volumesList.Cursor < 0 {
			m.volumesList.Cursor = 0
		}
		m.volumesList.Viewport.SetContent(m.volumesList.View())
	case "pgdown":
		page := m.volumesList.Viewport.Height
		if page < 1 {
			page = 10
		}
		m.volumesList.Cursor += page
		if m.volumesList.Cursor >= len(m.volumesList.Filtered) {
			m.volumesList.Cursor = len(m.volumesList.Filtered) - 1
		}
		m.volumesList.Viewport.SetContent(m.volumesList.View())
	case "i", "enter":
		if len(m.volumesList.Filtered) == 0 {
			return nil
		}
		selected := m.volumesList.Filtered[m.volumesList.Cursor]
		return m.inspectVolumeCmd(selected.Name)
	case "c":
		// Create is selection-independent; the action owns the node picker.
		return m.dispatchAction("volume-create", "Create volume", "")
	case "b":
		if sel, ok := m.selectedVolume(); ok {
			return m.dispatchAction("volume-browse", "Volume browser", view.EncodeRef(sel.NodeID, sel.Name))
		}
		return nil
	case "ctrl+d":
		if sel, ok := m.selectedVolume(); ok {
			return m.dispatchAction("volume-delete", "Delete volume", view.EncodeRef(sel.NodeID, sel.Name))
		}
		return nil
	}
	return nil
}

// selectedVolume returns the volume under the cursor, or false if the list is empty.
func (m *Model) selectedVolume() (volumeItem, bool) {
	if len(m.volumesList.Filtered) == 0 {
		return volumeItem{}, false
	}
	return m.volumesList.Filtered[m.volumesList.Cursor], true
}

// dispatchAction invokes a registered action, or surfaces the standard
// "Business Edition feature" dialog when it is not available. The action
// keybindings are inert in builds that do not register them.
func (m *Model) dispatchAction(actionName, label, arg string) tea.Cmd {
	action, ok := view.GetAction(actionName)
	if !ok {
		m.err = view.BEUnavailableErr(label)
		m.errorDialogActive = true
		return nil
	}
	return action(arg)
}

func (m *Model) toggleSort(field SortField) {
	if m.sortField == field {
		m.sortAscending = !m.sortAscending
	} else {
		m.sortField = field
		m.sortAscending = true
	}
	m.applySorting()
}

func cmpStr(a, b string, asc bool) bool {
	if asc {
		return a < b
	}
	return a > b
}

func (m *Model) applySorting() {
	if len(m.volumesList.Filtered) == 0 {
		return
	}
	cursorName := ""
	if m.volumesList.Cursor < len(m.volumesList.Filtered) {
		cursorName = m.volumesList.Filtered[m.volumesList.Cursor].Name
	}

	f := m.volumesList.Filtered
	asc := m.sortAscending
	switch m.sortField {
	case SortByName:
		sort.SliceStable(f, func(i, j int) bool {
			return cmpStr(strings.ToLower(f[i].Name), strings.ToLower(f[j].Name), asc)
		})
	case SortByStack:
		sort.SliceStable(f, func(i, j int) bool {
			if f[i].Stack == f[j].Stack {
				return strings.ToLower(f[i].Name) < strings.ToLower(f[j].Name)
			}
			return cmpStr(strings.ToLower(f[i].Stack), strings.ToLower(f[j].Stack), asc)
		})
	case SortByDriver:
		sort.SliceStable(f, func(i, j int) bool {
			return cmpStr(strings.ToLower(f[i].Driver), strings.ToLower(f[j].Driver), asc)
		})
	case SortByHost:
		sort.SliceStable(f, func(i, j int) bool {
			return cmpStr(strings.ToLower(f[i].Host), strings.ToLower(f[j].Host), asc)
		})
	case SortByCreated:
		sort.SliceStable(f, func(i, j int) bool {
			if asc {
				return f[i].Created.Before(f[j].Created)
			}
			return f[i].Created.After(f[j].Created)
		})
	}

	if cursorName != "" {
		for i, v := range f {
			if v.Name == cursorName {
				m.volumesList.Cursor = i
				break
			}
		}
	}
	m.volumesList.Viewport.SetContent(m.volumesList.View())
}

func (m *Model) setRenderItem() {
	m.volumesList.RenderItem = func(item volumeItem, selected bool, colWidth int) string {
		style := ui.ListItemStyle
		if selected {
			style = ui.ListSelectedStyle
		}

		widths := m.volumesList.ColWidths()
		if len(widths) < 6 {
			return item.Name
		}
		nameW, stackW, driverW := widths[0], widths[1], widths[2]
		mountW, createdW, hostW := widths[3], widths[4], widths[5]

		innerNameW := nameW
		if innerNameW > 1 {
			innerNameW--
		}
		name := " " + truncateWithEllipsis(item.Name, innerNameW)
		stack := truncateWithEllipsis(displayOrDash(item.Stack), stackW)
		driver := truncateWithEllipsis(item.Driver, driverW)
		mount := truncateWithEllipsis(item.Mountpoint, mountW)
		created := truncateWithEllipsis(formatCreated(item.Created), createdW)
		host := truncateWithEllipsis(displayOrDash(item.Host), hostW)

		sep := strings.Repeat(" ", 2)
		line := fmt.Sprintf("%-*s%s%-*s%s%-*s%s%-*s%s%-*s%s%-*s",
			nameW, name, sep,
			stackW, stack, sep,
			driverW, driver, sep,
			mountW, mount, sep,
			createdW, created, sep,
			hostW, host,
		)
		return style.Render(line)
	}
	m.volumesList.Viewport.SetContent(m.volumesList.View())
}
