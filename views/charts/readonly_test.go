// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package chartsview

import (
	"testing"

	"github.com/Eldara-Tech/swarmcli/v2/charts"

	"github.com/stretchr/testify/require"
)

func upgraded(name string, revisions int) []charts.Release {
	out := make([]charts.Release, 0, revisions)
	for n := 1; n <= revisions; n++ {
		status := charts.StatusSuperseded
		if n == revisions {
			status = charts.StatusDeployed
		}
		out = append(out, rev(name, n, status, "c", "1.0."+string(rune('0'+n))))
	}
	return out
}

// The keys are bound rather than dead: ctrl+d deletes in every other resource
// view, so an operator will press it here. It must answer with the command.
func TestBlockedActionsNameTheirCommand(t *testing.T) {
	m := sized(testModel(), 120, 24)
	loadReleases(t, m, map[string][]charts.Release{"hello": upgraded("hello", 3)}, nil)

	for _, tc := range []struct{ key, want string }{
		{"u", "swarmcli charts upgrade hello <repo/chart>"},
		{"r", "swarmcli charts rollback hello 2"},
		{"ctrl+d", "swarmcli charts uninstall hello"},
	} {
		require.Nil(t, m.Update(key(tc.key)), "a blocked action issues no command")
		require.True(t, m.confirmDialog.Visible, "key %q must explain itself", tc.key)
		require.True(t, m.confirmDialog.InfoMode, "dismiss-only: there is nothing to confirm")
		require.Contains(t, m.confirmDialog.Message, tc.want)
		require.Contains(t, m.View(), tc.want, "the dialog must actually be on screen")

		m.Update(key("esc"))
		m.Update(dismiss())
		require.False(t, m.confirmDialog.Visible)
	}
}

// The rollback target is the revision below the current one, which is what an
// operator reaching for rollback almost always wants.
func TestRollbackCommandTargetsThePreviousRevision(t *testing.T) {
	require.Contains(t, rollbackAction("app", 5).command, "rollback app 4")
	require.Contains(t, rollbackAction("app", 2).command, "rollback app 1")
	require.Contains(t, rollbackAction("app", 1).command, "rollback app 1",
		"a single-revision release has nothing below it; naming revision 0 would be a lie")
}

func TestBlockedActionsDoNothingOnAnEmptyList(t *testing.T) {
	m := sized(testModel(), 120, 24)
	loadReleases(t, m, nil, nil)
	for _, k := range []string{"u", "r", "ctrl+d"} {
		require.Nil(t, m.Update(key(k)))
		require.False(t, m.confirmDialog.Visible, "key %q with nothing selected", k)
	}
}

// While the dialog is up it owns the keyboard, so a stray sort or navigation
// key does not act on the list behind it.
func TestDialogCapturesInput(t *testing.T) {
	m := sized(testModel(), 120, 24)
	loadReleases(t, m, map[string][]charts.Release{
		"a": deployed("a", "c", "1.0.0"),
		"b": deployed("b", "c", "1.0.0"),
	}, nil)

	m.Update(key("u"))
	require.True(t, m.CapturesInput())

	m.Update(key("down"))
	require.Equal(t, 0, m.list.Cursor, "the list must not move behind the dialog")

	m.Update(key("esc"))
	m.Update(dismiss())
	require.False(t, m.CapturesInput())
	m.Update(key("down"))
	require.Equal(t, 1, m.list.Cursor)
}

// The stacks view jumps here with a release name. The view is empty when the
// factory runs, so the selection has to survive until the first read lands.
func TestCrossLinkSelectsAndExpandsTheRequestedRelease(t *testing.T) {
	m := sized(testModel(), 120, 24)
	m.pendingSelect = "wanted"

	loadReleases(t, m, map[string][]charts.Release{
		"aaa":    deployed("aaa", "c", "1.0.0"),
		"wanted": deployed("wanted", "c", "1.0.0"),
		"zzz":    deployed("zzz", "c", "1.0.0"),
	}, nil)

	sel, ok := m.selected()
	require.True(t, ok)
	require.Equal(t, "wanted", sel.Name)
	require.True(t, m.isExpanded(), "the release the operator asked for opens on arrival")
	require.Empty(t, m.pendingSelect, "consumed")
	require.Len(t, m.list.Filtered, 3, "selecting is not filtering; the other releases stay visible")
}

// OnEnter runs after the factory and normally jumps to the top of the list,
// which would undo the selection the payload asked for.
func TestCrossLinkSurvivesOnEnter(t *testing.T) {
	m := sized(testModel(), 120, 24)
	m.pendingSelect = "wanted"
	m.OnEnter()

	loadReleases(t, m, map[string][]charts.Release{
		"aaa":    deployed("aaa", "c", "1.0.0"),
		"wanted": deployed("wanted", "c", "1.0.0"),
	}, nil)

	sel, _ := m.selected()
	require.Equal(t, "wanted", sel.Name)
}

// A release that is not there must not leave the view fighting the cursor on
// every poll — but an empty first read is not proof it is missing.
func TestCrossLinkGivesUpOnlyAfterARealList(t *testing.T) {
	m := sized(testModel(), 120, 24)
	m.pendingSelect = "wanted"

	loadReleases(t, m, nil, nil)
	require.Equal(t, "wanted", m.pendingSelect, "an empty read proves nothing")

	loadReleases(t, m, map[string][]charts.Release{"other": deployed("other", "c", "1.0.0")}, nil)
	require.Empty(t, m.pendingSelect, "the release is genuinely not installed")

	m.Update(key("down"))
	loadReleases(t, m, map[string][]charts.Release{"other": deployed("other", "c", "1.0.0")}, nil)
	require.Equal(t, 0, m.list.Cursor, "and the cursor is left alone from then on")
}

// Every command the help offers must be one the CLI actually has.
func TestHelpListsTheChartOperations(t *testing.T) {
	var found bool
	for _, cat := range GetChartsHelpContent() {
		if cat.Title != "Chart operations (CLI only)" {
			continue
		}
		found = true
		joined := ""
		for _, it := range cat.Items {
			joined += it.Keys + " " + it.Description + "\n"
		}
		for _, want := range []string{"upgrade", "rollback", "uninstall", "install", "apply", "repo update"} {
			require.Contains(t, joined, want)
		}
	}
	require.True(t, found, "the help must say where the mutations live")
}
