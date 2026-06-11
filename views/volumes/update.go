// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package volumesview

import (
	"sort"
	"strings"
	"time"

	"swarmcli/core/primitives/hash"
	"swarmcli/ui"
	filterlist "swarmcli/ui/components/filterable/list"
	helpview "swarmcli/views/help"
	view "swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
)

func formatCreated(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	return t.Format("2006-01-02 15:04")
}

// buildColumns declares the volumes table columns for the shared content-aware
// layout. NAME, MOUNT POINT, and HOST flex (absorb slack and horizontally
// scroll when truncated on the selected row); the rest size to content.
func (m *Model) buildColumns() []filterlist.Column[volumeItem] {
	return []filterlist.Column[volumeItem]{
		{Label: "NAME", MinWidth: 4, Flex: true, Cell: func(v volumeItem) string { return v.Name }},
		{Label: "STACK", MinWidth: 5, Cell: func(v volumeItem) string { return displayOrDash(v.Stack) }},
		{Label: "DRIVER", MinWidth: 6, Cell: func(v volumeItem) string { return v.Driver }},
		{Label: "MOUNT POINT", MinWidth: 10, Flex: true, Cell: func(v volumeItem) string { return v.Mountpoint }},
		{Label: "CREATED", MinWidth: 16, Cell: func(v volumeItem) string { return formatCreated(v.Created) }},
		{Label: "HOST", MinWidth: 4, Flex: true, Cell: func(v volumeItem) string { return displayOrDash(v.Host) }},
	}
}

// sortColumnIndex maps the active sort field to its column index for the header
// sort arrow. The mapping is not 1:1 because MOUNT POINT (column 3) is
// unsortable, so CREATED and HOST sit one column past their sort-field ordinal.
func (m *Model) sortColumnIndex() int {
	switch m.sortField {
	case SortByName:
		return 0
	case SortByStack:
		return 1
	case SortByDriver:
		return 2
	case SortByCreated:
		return 4
	case SortByHost:
		return 5
	}
	return 0
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
	case "left":
		m.volumesList.ScrollLeft()
		m.volumesList.Viewport.SetContent(m.volumesList.View())
	case "right":
		m.volumesList.ScrollRight()
		m.volumesList.Viewport.SetContent(m.volumesList.View())
	case "up", "k":
		if m.volumesList.Cursor > 0 {
			m.volumesList.Cursor--
			m.volumesList.ResetColumnScroll()
			m.volumesList.Viewport.SetContent(m.volumesList.View())
		}
	case "down", "j":
		if m.volumesList.Cursor < len(m.volumesList.Filtered)-1 {
			m.volumesList.Cursor++
			m.volumesList.ResetColumnScroll()
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
		m.volumesList.ResetColumnScroll()
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
		m.volumesList.ResetColumnScroll()
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
	case "p":
		// Prune is selection-independent; the action owns the node picker.
		return m.dispatchAction("volume-prune", "Prune volumes", "")
	case "b":
		if sel, ok := m.selectedVolume(); ok {
			return m.dispatchAction("volume-browse", "Volume browser", view.EncodeRef(sel.NodeID, sel.Name, sel.Host))
		}
		return nil
	case "ctrl+d":
		if sel, ok := m.selectedVolume(); ok {
			return m.dispatchAction("volume-delete", "Delete volume", view.EncodeRef(sel.NodeID, sel.Name, sel.Host))
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
		if cmd := view.FeatureLockedCmd(label); cmd != nil {
			return cmd
		}
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
	m.volumesList.RenderItem = func(item volumeItem, selected bool, _ int) string {
		row := m.volumesList.RenderRow(item, selected)
		if selected {
			return ui.ListSelectedStyle.Render(row)
		}
		return ui.ListItemStyle.Render(row)
	}
	m.volumesList.Viewport.SetContent(m.volumesList.View())
}
