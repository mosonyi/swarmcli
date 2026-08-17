// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package servicesview

import (
	"testing"

	"github.com/stretchr/testify/require"

	filterlist "github.com/Eldara-Tech/swarmcli/ui/components/filterable/list"
)

// The grow column must be the last one, and there must be only one.
//
// A grow column in the middle does not remove the void a wide terminal opens,
// it relocates it to just after that column; two of them split the slack and
// open two. Both were shipped and reverted before this rule was written down.
func TestExactlyOneGrowColumnAndItIsLast(t *testing.T) {
	cols := testModel().layoutColumns()
	require.NotEmpty(t, cols)

	var growing []string
	for _, c := range cols {
		if c.Grow {
			growing = append(growing, c.Label)
		}
	}
	require.Len(t, growing, 1, "exactly one column may absorb leftover width")
	require.Equal(t, cols[len(cols)-1].Label, growing[0],
		"and it must be the last column, or the gap merely moves")
}

// colWidth reports the laid-out width of the named column at totalWidth, and
// the width at which the table is neither stretched nor squeezed — the baseline
// a wider terminal has to be compared against.
func colWidth(t *testing.T, m *Model, label string, totalWidth int) (width, natural int) {
	t.Helper()
	cols := m.layoutColumns()
	sortCol, _ := m.sortIndicator()
	natural = filterlist.NaturalWidth(cols, m.List.Items, sortCol)
	widths := filterlist.LayoutWidths(cols, m.List.Items, totalWidth, sortCol)
	for i, c := range cols {
		if c.Label == label {
			return widths[i], natural
		}
	}
	t.Fatalf("no %s column", label)
	return 0, 0
}

// grewOnAWideTerminal reports whether the named column takes more width at 300
// columns than the table wants for its content.
func grewOnAWideTerminal(t *testing.T, m *Model, label string) bool {
	t.Helper()
	wide, natural := colWidth(t, m, label, 300)
	require.Less(t, natural, 300, "the fixture must leave a wide terminal something to spend")
	at, _ := colWidth(t, m, label, natural)
	return wide > at
}

// ERROR grows, but on a swarm with no errors it has nothing to grow into: every
// cell is empty, so handing it the whole leftover of a wide terminal packs the
// table into the left third and leaves the rest dead. The width goes back to the
// elastic columns there, and to ERROR only once there is an error to read.
func TestWideTerminalFillsTheColumnsAnOperatorReads(t *testing.T) {
	m := testModel()
	loadServices(m, fakeEntries("alpha", "beta"))

	require.True(t, grewOnAWideTerminal(t, m, "SERVICE"),
		"with no errors the leftover is shared by the elastic columns")
	require.True(t, grewOnAWideTerminal(t, m, "IMAGE"))

	m.serviceErrorText["id-alpha"] = "task: non-zero exit (1): no such file or directory"

	require.True(t, grewOnAWideTerminal(t, m, "ERROR"),
		"once there is an error, ERROR is what the leftover is for")
	require.False(t, grewOnAWideTerminal(t, m, "IMAGE"),
		"and the columns before it stop drifting apart again")
}
