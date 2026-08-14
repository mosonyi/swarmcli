// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package chartsview

import (
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Eldara-Tech/swarmcli/charts"
	"github.com/Eldara-Tech/swarmcli/core/primitives/hash"
	"github.com/Eldara-Tech/swarmcli/ui"
	filterlist "github.com/Eldara-Tech/swarmcli/ui/components/filterable/list"
	"github.com/Eldara-Tech/swarmcli/views/confirmdialog"
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
		{Label: "UPD", MinWidth: 3, Cell: m.updCell},
	}
}

// updCell distinguishes the two reasons a release has no newer version, which
// an empty cell would run together: nothing newer is published, versus this
// machine has no cached index to compare against. The second is "?" rather
// than a footer line because it is a per-row fact and the footer is already
// carrying the counts, the convergence reason and the read-only hint.
func (m *Model) updCell(r releaseItem) string {
	if r.Latest != "" {
		return r.Latest
	}
	if !m.haveIndexes {
		return "?"
	}
	return "—"
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
		if m.confirmDialog.Visible {
			return m.confirmDialog.Update(msg)
		}
		return m.handleNormalKeys(msg)

	case confirmdialog.ResultMsg:
		m.confirmDialog.Visible = false
		m.confirmDialog.InfoMode = false
		return nil
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

	m.haveIndexes = msg.HaveIndexes

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

	m.applyPendingSelect()
	m.clampChild()
	m.state = stateReady
	m.list.Viewport.SetContent(m.list.View())
	return nil
}

// applyPendingSelect honours a cross-link's requested release once there is a
// list to select it in.
//
// It selects and expands rather than filtering: the "/" filter belongs to the
// app's search bar, and setting a query the operator never typed would leave
// the list narrowed with nothing on screen saying why.
func (m *Model) applyPendingSelect() {
	if m.pendingSelect == "" {
		return
	}
	for i, r := range m.list.Filtered {
		if r.Name == m.pendingSelect {
			m.list.Cursor = i
			m.expanded[r.Name] = true
			m.childIndex = noChild
			m.pendingSelect = ""
			return
		}
	}
	// Not found. Give up only once there was a real list to look in — the
	// first read can land empty — because retrying forever would fight the
	// operator's cursor on every poll.
	if len(m.list.Filtered) > 0 {
		m.pendingSelect = ""
	}
}

// clampChild keeps the child selection inside the release under the cursor. A
// reload can land on a release with fewer revisions than the last one had, and
// a filter or sort can move the cursor to a different release entirely.
func (m *Model) clampChild() {
	if m.childIndex == noChild {
		return
	}
	sel, ok := m.selected()
	if !ok || !m.expanded[sel.Name] {
		m.childIndex = noChild
		return
	}
	if n := len(sel.children()); m.childIndex >= n {
		m.childIndex = n - 1
	}
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
		m.moveUp()
	case "down", "j":
		m.moveDown()
	case "pgup":
		m.movePage(-1)
	case "pgdown":
		m.movePage(1)
	case "esc":
		// Only reached because IsRowExpanded told the app to forward it.
		m.collapseOneLevel()
	case "enter":
		return m.handleEnter()
	case "i":
		if rev, ok := m.selectedRevision(); ok {
			sel, _ := m.selected()
			return m.inspectRevisionCmd(sel.Name, rev)
		}
	case "v":
		if rev, ok := m.selectedRevision(); ok {
			sel, _ := m.selected()
			return m.inspectValuesCmd(sel.Name, rev)
		}
	case "d":
		if child, ok := m.selectedChild(); ok && child.kind == childRevision {
			sel, _ := m.selected()
			return m.diffRevisionCmd(sel.Name, child)
		}
	case "s":
		if sel, ok := m.selected(); ok {
			return servicesForStackCmd(sel.Name, "")
		}
	case "u":
		if sel, ok := m.selected(); ok {
			m.showBlocked(upgradeAction(sel.Name))
		}
	case "r":
		if sel, ok := m.selected(); ok {
			m.showBlocked(rollbackAction(sel.Name, sel.revision()))
		}
	case "ctrl+d":
		if sel, ok := m.selected(); ok {
			m.showBlocked(uninstallAction(sel.Name))
		}
	}
	return nil
}

// selectedRevision is the revision `i` and `v` act on: the one under the cursor
// when a revision child is selected, otherwise the release's current revision.
func (m *Model) selectedRevision() (charts.Release, bool) {
	sel, ok := m.selected()
	if !ok {
		return charts.Release{}, false
	}
	if child, isChild := m.selectedChild(); isChild {
		if child.kind != childRevision {
			return charts.Release{}, false
		}
		return child.rev, true
	}
	return sel.current(), true
}

// handleEnter expands or collapses the release, unless a service child is
// selected — from there it drills into that service.
func (m *Model) handleEnter() tea.Cmd {
	sel, ok := m.selected()
	if !ok {
		return nil
	}
	if child, isChild := m.selectedChild(); isChild {
		if child.kind == childService {
			return servicesForStackCmd(sel.Name, child.svc.Name)
		}
		return nil
	}
	m.expanded[sel.Name] = !m.expanded[sel.Name]
	m.childIndex = noChild
	m.list.Viewport.SetContent(m.list.View())
	return nil
}

// servicesForStackCmd opens the services view scoped to the release's stack. A
// release deploys a stack named after itself, so the services view's existing
// stack scope is the right drill-down, and scale, restart and logs stay in the
// view that owns them.
func servicesForStackCmd(release, selectService string) tea.Cmd {
	payload := map[string]any{"stackName": release}
	if selectService != "" {
		payload["selectServiceName"] = selectService
	}
	return func() tea.Msg {
		return view.NavigateToMsg{ViewName: servicesview.ViewName, Payload: payload}
	}
}

// collapseOneLevel walks esc back out: from a child to the release row, then
// from an expanded release to a collapsed one.
func (m *Model) collapseOneLevel() {
	if m.childIndex != noChild {
		m.childIndex = noChild
		m.list.Viewport.SetContent(m.list.View())
		return
	}
	if sel, ok := m.selected(); ok {
		delete(m.expanded, sel.Name)
		m.list.Viewport.SetContent(m.list.View())
	}
}

// moveDown steps into an expanded release's children before moving on to the
// next release.
func (m *Model) moveDown() {
	if sel, ok := m.selected(); ok && m.expanded[sel.Name] {
		if n := len(sel.children()); m.childIndex < n-1 {
			m.childIndex++
			m.list.Viewport.SetContent(m.list.View())
			return
		}
	}
	if m.list.Cursor < len(m.list.Filtered)-1 {
		m.list.Cursor++
		m.childIndex = noChild
		m.list.ResetColumnScroll()
		m.list.Viewport.SetContent(m.list.View())
	}
}

// moveUp is moveDown's mirror: out of the children to the release row, then to
// the previous release — landing on its last child when it is expanded, so the
// two directions traverse the same sequence of rows.
func (m *Model) moveUp() {
	if m.childIndex != noChild {
		m.childIndex--
		m.list.Viewport.SetContent(m.list.View())
		return
	}
	if m.list.Cursor > 0 {
		m.list.Cursor--
		m.childIndex = noChild
		if sel, ok := m.selected(); ok && m.expanded[sel.Name] {
			m.childIndex = len(sel.children()) - 1
		}
		m.list.ResetColumnScroll()
		m.list.Viewport.SetContent(m.list.View())
	}
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
	m.childIndex = noChild
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
	m.clampChild()
	m.list.Viewport.SetContent(m.list.View())
}

func (m *Model) setRenderItem() {
	m.list.RenderItem = func(item releaseItem, selected bool, _ int) string {
		row := m.list.RenderRow(item, selected)
		// The release row loses its highlight while a child is selected, so
		// exactly one line on screen reads as the cursor.
		if selected && m.childIndex == noChild {
			row = ui.ListSelectedStyle.Render(row)
		} else {
			row = ui.ListItemStyle.Render(row)
		}
		if !m.expanded[item.Name] {
			return row
		}
		child := noChild
		if selected {
			child = m.childIndex
		}
		lines, _ := expansionBlock(item, child)
		return row + "\n" + strings.Join(lines, "\n")
	}
	m.list.Viewport.SetContent(m.list.View())
}
