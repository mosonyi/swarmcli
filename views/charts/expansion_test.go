// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package chartsview

import (
	"strings"
	"testing"

	"github.com/Eldara-Tech/swarmcli/charts"
	inspectview "github.com/Eldara-Tech/swarmcli/views/inspect"
	servicesview "github.com/Eldara-Tech/swarmcli/views/services"
	"github.com/Eldara-Tech/swarmcli/views/view"

	"github.com/stretchr/testify/require"
)

// twoRevs is a release upgraded once, with one service running.
func twoRevs(t *testing.T, m *Model) {
	t.Helper()
	loadReleases(t, m,
		map[string][]charts.Release{"app": {
			rev("app", 1, charts.StatusSuperseded, "c", "1.0.0"),
			rev("app", 2, charts.StatusDeployed, "c", "2.0.0"),
		}},
		map[string][]charts.ServiceState{"app": {converged("app_web")}})
}

func TestEnterExpandsAndCollapses(t *testing.T) {
	m := sized(testModel(), 120, 24)
	twoRevs(t, m)

	require.False(t, m.isExpanded())
	m.Update(key("enter"))
	require.True(t, m.isExpanded())

	out := m.View()
	require.Contains(t, out, "superseded", "the older revision")
	require.Contains(t, out, "app_web", "the live service")
	require.Contains(t, out, "OWNER")

	m.Update(key("enter"))
	require.False(t, m.isExpanded())
	require.NotContains(t, m.View(), "app_web")
}

// Down walks into the children before moving on; up mirrors it, so both
// directions traverse the same sequence of rows.
func TestChildNavigationIsSymmetric(t *testing.T) {
	m := sized(testModel(), 120, 24)
	twoRevs(t, m)
	m.Update(key("enter"))

	require.Equal(t, noChild, m.childIndex)
	for want := 0; want < 3; want++ { // 2 revisions + 1 service
		m.Update(key("down"))
		require.Equal(t, want, m.childIndex)
	}
	m.Update(key("down"))
	require.Equal(t, 2, m.childIndex, "the last child must not run past the end of a single release")

	for want := 1; want >= 0; want-- {
		m.Update(key("up"))
		require.Equal(t, want, m.childIndex)
	}
	m.Update(key("up"))
	require.Equal(t, noChild, m.childIndex, "up from the first child returns to the release row")
}

func TestDownFromTheLastChildMovesToTheNextRelease(t *testing.T) {
	m := sized(testModel(), 120, 24)
	loadReleases(t, m, map[string][]charts.Release{
		"a": deployed("a", "c", "1.0.0"),
		"b": deployed("b", "c", "1.0.0"),
	}, nil)

	m.Update(key("enter")) // expand "a"
	m.Update(key("down"))  // its only child: the "(no services)" block has none, so revision 1
	require.Equal(t, 0, m.childIndex)

	m.Update(key("down"))
	require.Equal(t, "b", m.list.Filtered[m.list.Cursor].Name)
	require.Equal(t, noChild, m.childIndex)
}

// Moving up into an expanded release lands on its last child, not its header
// row, or the two directions would visit different rows.
func TestUpEntersThePreviousReleaseAtItsLastChild(t *testing.T) {
	m := sized(testModel(), 120, 24)
	loadReleases(t, m, map[string][]charts.Release{
		"a": deployed("a", "c", "1.0.0"),
		"b": deployed("b", "c", "1.0.0"),
	}, nil)

	m.Update(key("enter")) // expand "a"
	m.Update(key("down"))  // into a's child
	m.Update(key("down"))  // onto b
	require.Equal(t, "b", m.list.Filtered[m.list.Cursor].Name)

	m.Update(key("up"))
	require.Equal(t, "a", m.list.Filtered[m.list.Cursor].Name)
	require.Equal(t, 0, m.childIndex, "expected a's last child")
}

// esc walks back out one level at a time. The app only forwards esc when
// IsRowExpanded says there is somewhere to go.
func TestEscWalksBackOutOfTheExpansion(t *testing.T) {
	m := sized(testModel(), 120, 24)
	twoRevs(t, m)

	require.False(t, m.IsRowExpanded(), "with nothing expanded esc must pop the view instead")

	m.Update(key("enter"))
	m.Update(key("down"))
	require.True(t, m.IsRowExpanded())

	m.Update(key("esc"))
	require.Equal(t, noChild, m.childIndex, "first esc returns to the release row")
	require.True(t, m.isExpanded())
	require.True(t, m.IsRowExpanded())

	m.Update(key("esc"))
	require.False(t, m.isExpanded(), "second esc collapses the release")
	require.False(t, m.IsRowExpanded(), "a third esc belongs to the app")
}

// Expansion must not disable the command bar: that would be the cost of
// routing esc through CapturesInput instead of its own probe.
func TestExpansionDoesNotCaptureInput(t *testing.T) {
	m := sized(testModel(), 120, 24)
	twoRevs(t, m)
	m.Update(key("enter"))
	m.Update(key("down"))
	require.False(t, m.CapturesInput())
}

func TestInspectActsOnTheSelectedRevision(t *testing.T) {
	m := sized(testModel(), 120, 24)
	twoRevs(t, m)
	m.Update(key("enter"))
	m.Update(key("down")) // revision 1

	payload := runCmd(m.Update(key("i"))).(view.NavigateToMsg).Payload.(map[string]any)
	require.Contains(t, payload["title"], "rev 1")
	require.Contains(t, payload["json"], "c:1.0.0")

	m.Update(key("down")) // revision 2
	payload = runCmd(m.Update(key("v"))).(view.NavigateToMsg).Payload.(map[string]any)
	require.Contains(t, payload["title"], "rev 2")
	require.Contains(t, payload["json"], "replicas: 2")
}

func TestDiffComparesAgainstThePreviousRevision(t *testing.T) {
	m := sized(testModel(), 120, 24)
	twoRevs(t, m)
	m.Update(key("enter"))
	m.Update(key("down")) // revision 1
	m.Update(key("down")) // revision 2

	nav := runCmd(m.Update(key("d"))).(view.NavigateToMsg)
	require.Equal(t, inspectview.ViewName, nav.ViewName)
	payload := nav.Payload.(map[string]any)
	require.Contains(t, payload["title"], "rev 1 → rev 2")
	require.Equal(t, inspectview.FormatRaw, payload["format"],
		"a diff is not a YAML document; the tree view would eat the +/- prefixes")

	diff := payload["json"].(string)
	require.Contains(t, diff, "- "+"    image: c:1.0.0", "the outgoing image, marked removed")
	require.Contains(t, diff, "+ "+"    image: c:2.0.0", "the incoming image, marked added")
	require.Contains(t, diff, "  "+"  app:", "unchanged lines carry a blank marker")
}

func TestDiffOfTheFirstRevisionSaysSo(t *testing.T) {
	m := sized(testModel(), 120, 24)
	twoRevs(t, m)
	m.Update(key("enter"))
	m.Update(key("down")) // revision 1

	payload := runCmd(m.Update(key("d"))).(view.NavigateToMsg).Payload.(map[string]any)
	require.Contains(t, payload["title"], "first revision")
	diff := payload["json"].(string)
	require.Contains(t, diff, "+ services:", "everything is an addition")
	require.NotContains(t, diff, "- ", "there is no previous revision to remove anything from")
}

func TestDiffDoesNothingOnAServiceRow(t *testing.T) {
	m := sized(testModel(), 120, 24)
	twoRevs(t, m)
	m.Update(key("enter"))
	m.Update(key("down"))
	m.Update(key("down"))
	m.Update(key("down")) // the service row

	child, ok := m.selectedChild()
	require.True(t, ok)
	require.Equal(t, childService, child.kind)
	require.Nil(t, m.Update(key("d")))
}

func TestEnterOnAServiceRowDrillsIntoIt(t *testing.T) {
	m := sized(testModel(), 120, 24)
	twoRevs(t, m)
	m.Update(key("enter"))
	m.Update(key("down"))
	m.Update(key("down"))
	m.Update(key("down")) // the service row

	nav := runCmd(m.Update(key("enter"))).(view.NavigateToMsg)
	require.Equal(t, servicesview.ViewName, nav.ViewName)
	require.Equal(t, map[string]any{
		"stackName": "app", "selectServiceName": "app_web",
	}, nav.Payload)
	require.True(t, m.isExpanded(), "drilling in must not also collapse the row")
}

// A background poll must not disturb what the operator has open.
func TestExpansionSurvivesAReload(t *testing.T) {
	m := sized(testModel(), 120, 24)
	twoRevs(t, m)
	m.Update(key("enter"))
	m.Update(key("down"))
	require.Equal(t, 0, m.childIndex)

	twoRevs(t, m)
	require.True(t, m.isExpanded())
	require.Equal(t, 0, m.childIndex)
}

// A reload can land on a release with fewer children than the last one had.
func TestChildSelectionIsClampedWhenChildrenShrink(t *testing.T) {
	m := sized(testModel(), 120, 24)
	twoRevs(t, m)
	m.Update(key("enter"))
	for i := 0; i < 3; i++ {
		m.Update(key("down"))
	}
	require.Equal(t, 2, m.childIndex)

	// Same release, one revision and no services.
	loadReleases(t, m, map[string][]charts.Release{"app": deployed("app", "c", "1.0.0")}, nil)
	require.Equal(t, 0, m.childIndex, "the selection must land inside the shorter release")

	child, ok := m.selectedChild()
	require.True(t, ok)
	require.Equal(t, childRevision, child.kind)
}

func TestFilteringDropsTheChildSelection(t *testing.T) {
	m := sized(testModel(), 120, 24)
	loadReleases(t, m, map[string][]charts.Release{
		"app":   deployed("app", "c", "1.0.0"),
		"other": deployed("other", "c", "1.0.0"),
	}, nil)
	m.Update(key("enter"))
	m.Update(key("down"))
	require.Equal(t, 0, m.childIndex)

	m.ApplySearchQuery("other")
	require.Equal(t, noChild, m.childIndex)
	require.Nil(t, m.Update(key("d")), "no child is selected, so nothing to diff")
}

// The whole point of taking the offset over: a child below the fold must still
// be rendered. Assert on the line carrying content, not on the row count — a
// padded layout has the same number of rows either way.
func TestSelectedChildStaysOnScreenInAShortViewport(t *testing.T) {
	m := sized(testModel(), 120, 6)

	svcs := make([]charts.ServiceState, 0, 12)
	for _, n := range []string{"s01", "s02", "s03", "s04", "s05", "s06", "s07", "s08", "s09", "s10", "s11", "s12"} {
		svcs = append(svcs, converged(n))
	}
	loadReleases(t, m,
		map[string][]charts.Release{"app": deployed("app", "c", "1.0.0")},
		map[string][]charts.ServiceState{"app": svcs})

	m.Update(key("enter"))
	for i := 0; i < 13; i++ { // 1 revision + 12 services
		m.Update(key("down"))
	}
	child, ok := m.selectedChild()
	require.True(t, ok)
	require.Equal(t, "s12", child.svc.Name, "the fixture must actually reach the last service")

	require.Contains(t, m.View(), "s12", "the selected child scrolled out of view")
}

// expansionBlock returns the rendered lines and each child's line index
// together so the scroll math cannot disagree with the renderer about how tall
// a release is. Hold them to that.
func TestChildLineIndexesMatchTheRenderedLines(t *testing.T) {
	m := sized(testModel(), 120, 24)
	twoRevs(t, m)
	sel, ok := m.selected()
	require.True(t, ok)

	rows := sel.children()
	lines, childLine := expansionBlock(sel, noChild)
	require.Len(t, childLine, len(rows))

	for i, row := range rows {
		require.Less(t, childLine[i], len(lines))
		want := row.svc.Name
		if row.kind == childRevision {
			want = row.rev.Chart.Name + "-" + row.rev.Chart.Version
		}
		require.Contains(t, lines[childLine[i]], want,
			"child %d claims line %d, which holds something else", i, childLine[i])
	}

	m.expanded[sel.Name] = true
	require.Equal(t, 1+len(lines), m.itemLineCount(sel))
}

func TestExpansionSaysWhenThereAreNoServices(t *testing.T) {
	m := sized(testModel(), 120, 24)
	loadReleases(t, m, map[string][]charts.Release{"app": deployed("app", "c", "1.0.0")}, nil)
	m.Update(key("enter"))
	require.Contains(t, m.View(), "(no services)")
}

// A release with no owner stamp reads as unowned rather than as a blank cell.
func TestRevisionRowShowsTheOwnerStamp(t *testing.T) {
	m := sized(testModel(), 120, 24)
	owned := rev("app", 1, charts.StatusDeployed, "c", "1.0.0")
	owned.Owner = "apply/prod-swarm:release/app"
	loadReleases(t, m, map[string][]charts.Release{
		"app":   {owned},
		"plain": deployed("plain", "c", "1.0.0"),
	}, nil)

	m.Update(key("enter"))
	require.Contains(t, m.View(), "apply/prod-swarm:release/app")

	m.Update(key("esc"))
	m.Update(key("down"))
	m.Update(key("enter"))
	block, _ := expansionBlock(m.list.Filtered[m.list.Cursor], noChild)
	require.Contains(t, strings.Join(block, "\n"), "—")
}
