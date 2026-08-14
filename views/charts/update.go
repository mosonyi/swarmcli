// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package chartsview

import (
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Eldara-Tech/swarmcli/core/primitives/hash"
	"github.com/Eldara-Tech/swarmcli/ui"
	filterlist "github.com/Eldara-Tech/swarmcli/ui/components/filterable/list"
	helpview "github.com/Eldara-Tech/swarmcli/views/help"
	servicesview "github.com/Eldara-Tech/swarmcli/views/services"
	"github.com/Eldara-Tech/swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
)

func formatCreated(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	return t.Format("2006-01-02 15:04")
}

// buildColumns declares the release table columns for the shared content-aware
// layout. NAME and CHART flex; the rest size to content.
//
// STATUS and HEALTH are both present on purpose: the first is the record we
// wrote when we deployed, the second is what the swarm is doing now.
func (m *Model) buildColumns() []filterlist.Column[releaseItem] {
	return []filterlist.Column[releaseItem]{
		{Label: "NAME", MinWidth: 4, Flex: true, Cell: func(r releaseItem) string { return r.Name }},
		{Label: "REV", MinWidth: 3, Cell: func(r releaseItem) string { return strconv.Itoa(r.revision()) }},
		{Label: "STATUS", MinWidth: 6, Cell: func(r releaseItem) string { return displayOrDash(r.status()) }},
		{Label: "HEALTH", MinWidth: 6, Cell: func(r releaseItem) string { return r.healthLabel() }},
		{Label: "CHART", MinWidth: 5, Flex: true, Cell: func(r releaseItem) string { return r.chartRef() }},
		{Label: "UPDATED", MinWidth: 16, Cell: func(r releaseItem) string { return formatCreated(r.Created) }},
	}
}

// sortColumnIndex maps the active sort field to its column index for the header
// sort arrow. Every column is sortable, so the mapping is 1:1.
func (m *Model) sortColumnIndex() int {
	switch m.sortField {
	case SortByName:
		return 0
	case SortByRevision:
		return 1
	case SortByStatus:
		return 2
	case SortByHealth:
		return 3
	case SortByChart:
		return 4
	case SortByUpdated:
		return 5
	}
	return 0
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case SpinnerTickMsg:
		m.spinner++
		if m.state == stateLoading {
			m.list.Viewport.SetContent(m.list.View())
		}
		return spinnerTickCmd()

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.Viewport.Width = msg.Width
		m.list.Viewport.Height = msg.Height
		m.list.SetOuterSize(msg.Width, msg.Height)
		if m.firstResize {
			m.list.Viewport.YOffset = 0
			m.firstResize = false
		} else if m.list.Cursor == 0 {
			m.list.Viewport.YOffset = 0
		}
		return nil

	case ReleasesLoadedMsg:
		return m.handleReleasesLoaded(msg)

	case TickMsg:
		if m.visible && m.state == stateReady && !m.errorDialogActive {
			return tea.Batch(m.checkReleasesCmd(m.lastSnapshot), tickCmd())
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

func (m *Model) handleReleasesLoaded(msg ReleasesLoadedMsg) tea.Cmd {
	if msg.Err != nil {
		// A single unreadable release record fails the whole listing, so a
		// transient or one-off decode failure must not empty a view that
		// already has good data — keep it and say the refresh failed.
		if m.state == stateReady && len(m.list.Items) > 0 {
			l().Warnf("ChartsView: background refresh failed: %v", msg.Err)
			m.showToast("Refresh failed (will retry)")
			return nil
		}
		m.state = stateError
		m.err = msg.Err
		m.errorDialogActive = true
		return nil
	}

	selectedName := ""
	if !m.resetCursorOnNextLoad {
		if sel, ok := m.selected(); ok {
			selectedName = sel.Name
		}
	}

	if h, err := hash.Compute(stableReleases(msg.Releases)); err == nil {
		m.lastSnapshot = h
	} else {
		l().Errorf("ChartsView: Error computing hash: %v", err)
	}

	m.list.Items = msg.Releases
	m.list.ApplyFilter()
	m.applySorting()

	if m.resetCursorOnNextLoad {
		m.list.Cursor = 0
		m.list.Viewport.YOffset = 0
		m.resetCursorOnNextLoad = false
	} else if selectedName != "" {
		for i, r := range m.list.Filtered {
			if r.Name == selectedName {
				m.list.Cursor = i
				break
			}
		}
	}

	m.state = stateReady
	m.list.Viewport.SetContent(m.list.View())
	return nil
}

func (m *Model) handleNormalKeys(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "?":
		return func() tea.Msg {
			return view.NavigateToMsg{ViewName: helpview.ViewName, Payload: GetChartsHelpContent()}
		}
	case "N":
		m.toggleSort(SortByName)
	case "R":
		m.toggleSort(SortByRevision)
	case "S":
		m.toggleSort(SortByStatus)
	case "H":
		m.toggleSort(SortByHealth)
	case "C":
		m.toggleSort(SortByChart)
	case "U":
		m.toggleSort(SortByUpdated)
	case "left":
		m.list.ScrollLeft()
		m.list.Viewport.SetContent(m.list.View())
	case "right":
		m.list.ScrollRight()
		m.list.Viewport.SetContent(m.list.View())
	case "up", "k":
		if m.list.Cursor > 0 {
			m.list.Cursor--
			m.list.ResetColumnScroll()
			m.list.Viewport.SetContent(m.list.View())
		}
	case "down", "j":
		if m.list.Cursor < len(m.list.Filtered)-1 {
			m.list.Cursor++
			m.list.ResetColumnScroll()
			m.list.Viewport.SetContent(m.list.View())
		}
	case "pgup":
		m.movePage(-1)
	case "pgdown":
		m.movePage(1)
	case "i":
		if sel, ok := m.selected(); ok {
			return m.inspectRevisionCmd(sel.Name, sel.current())
		}
	case "v":
		if sel, ok := m.selected(); ok {
			return m.inspectValuesCmd(sel.Name, sel.current())
		}
	case "s":
		if sel, ok := m.selected(); ok {
			// A release deploys a stack named after itself, so the services
			// view's existing stack scope is the right drill-down.
			name := sel.Name
			return func() tea.Msg {
				return view.NavigateToMsg{
					ViewName: servicesview.ViewName,
					Payload:  map[string]any{"stackName": name},
				}
			}
		}
	}
	return nil
}

func (m *Model) movePage(dir int) {
	page := m.list.Viewport.Height
	if page < 1 {
		page = 10
	}
	m.list.Cursor += dir * page
	if m.list.Cursor < 0 {
		m.list.Cursor = 0
	}
	if m.list.Cursor >= len(m.list.Filtered) {
		m.list.Cursor = len(m.list.Filtered) - 1
	}
	if m.list.Cursor < 0 {
		m.list.Cursor = 0
	}
	m.list.ResetColumnScroll()
	m.list.Viewport.SetContent(m.list.View())
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

func cmpInt(a, b int, asc bool) bool {
	if asc {
		return a < b
	}
	return a > b
}

func (m *Model) applySorting() {
	if len(m.list.Filtered) == 0 {
		return
	}
	cursorName := ""
	if sel, ok := m.selected(); ok {
		cursorName = sel.Name
	}

	f := m.list.Filtered
	asc := m.sortAscending
	byName := func(i, j int) bool { return strings.ToLower(f[i].Name) < strings.ToLower(f[j].Name) }
	switch m.sortField {
	case SortByName:
		sort.SliceStable(f, func(i, j int) bool {
			return cmpStr(strings.ToLower(f[i].Name), strings.ToLower(f[j].Name), asc)
		})
	case SortByRevision:
		sort.SliceStable(f, func(i, j int) bool {
			if f[i].revision() == f[j].revision() {
				return byName(i, j)
			}
			return cmpInt(f[i].revision(), f[j].revision(), asc)
		})
	case SortByStatus:
		sort.SliceStable(f, func(i, j int) bool {
			if f[i].status() == f[j].status() {
				return byName(i, j)
			}
			return cmpStr(f[i].status(), f[j].status(), asc)
		})
	case SortByHealth:
		sort.SliceStable(f, func(i, j int) bool {
			if f[i].healthRank() == f[j].healthRank() {
				return byName(i, j)
			}
			return cmpInt(f[i].healthRank(), f[j].healthRank(), asc)
		})
	case SortByChart:
		sort.SliceStable(f, func(i, j int) bool {
			if f[i].chartRef() == f[j].chartRef() {
				return byName(i, j)
			}
			return cmpStr(strings.ToLower(f[i].chartRef()), strings.ToLower(f[j].chartRef()), asc)
		})
	case SortByUpdated:
		sort.SliceStable(f, func(i, j int) bool {
			if f[i].Created.Equal(f[j].Created) {
				return byName(i, j)
			}
			if asc {
				return f[i].Created.Before(f[j].Created)
			}
			return f[i].Created.After(f[j].Created)
		})
	}

	if cursorName != "" {
		for i, r := range f {
			if r.Name == cursorName {
				m.list.Cursor = i
				break
			}
		}
	}
	m.list.Viewport.SetContent(m.list.View())
}

func (m *Model) setRenderItem() {
	m.list.RenderItem = func(item releaseItem, selected bool, _ int) string {
		row := m.list.RenderRow(item, selected)
		if selected {
			return ui.ListSelectedStyle.Render(row)
		}
		return ui.ListItemStyle.Render(row)
	}
	m.list.Viewport.SetContent(m.list.View())
}
