// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package volumesview

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/require"
)

// longVolName is wide enough to overflow the NAME column on a narrow terminal
// and to verify full visibility on a wide one.
const longVolName = "volume_with_a_fairly_long_descriptive_name"

// longVolume returns the loaded item whose name overflows, for scroll tests.
func longVolume(m *Model) volumeItem {
	for _, v := range m.volumesList.Filtered {
		if v.Name == longVolName {
			return v
		}
	}
	return volumeItem{}
}

func TestHeaderAndRow_Aligned(t *testing.T) {
	for _, w := range []int{60, 120, 220} {
		m := testModel()
		m.volumesList.Viewport.Width = w
		loadVolumes(m, fakeVolumes(longVolName, "z2"))

		header := m.volumesList.RenderHeader()
		row := m.volumesList.RenderItem(m.volumesList.Filtered[0], false, 0)
		require.Equal(t, lipgloss.Width(header), lipgloss.Width(row),
			"header and row widths must match at width %d", w)
	}
}

func TestContentAwareName_WideTerminal(t *testing.T) {
	m := testModel()
	m.volumesList.Viewport.Width = 220
	loadVolumes(m, fakeVolumes(longVolName, "z2"))
	row := m.volumesList.RenderItem(longVolume(m), false, 0)
	require.Contains(t, row, longVolName, "full name must be visible on a wide terminal")
}

func TestArrowScrollMovesWindow(t *testing.T) {
	m := testModel()
	m.volumesList.Viewport.Width = 70 // narrow → flex columns overflow
	loadVolumes(m, fakeVolumes(longVolName, "z2"))
	before := m.volumesList.RenderItem(longVolume(m), true, 0)
	m.Update(key("right"))
	after := m.volumesList.RenderItem(longVolume(m), true, 0)
	require.NotEqual(t, before, after, "right arrow should shift the truncated cell window")
}

func TestResetScrollOnCursorMove(t *testing.T) {
	m := testModel()
	m.volumesList.Viewport.Width = 70
	loadVolumes(m, fakeVolumes(longVolName, "z2"))
	m.Update(key("right"))
	m.Update(key("right"))
	scrolled := m.volumesList.RenderItem(longVolume(m), true, 0)

	m.Update(key("down"))
	m.Update(key("up"))

	require.NotEqual(t, scrolled, m.volumesList.RenderItem(longVolume(m), true, 0),
		"scroll offset must reset when the cursor moves")
}

// A wide terminal must be filled, not left half dead.
//
// Confining the leftover to the trailing column was tried first, so the columns
// before it would stop drifting apart. It stops the drift by spending the whole
// surplus on one cell: on a 360-column terminal the volumes table needs 108 and
// MOUNT POINT got a 300-cell column to hold a 43-character path, so everything
// packed into the left third. Operators asked for the width back.
func TestWideTerminalSpreadsTheLeftoverAcrossTheElasticColumns(t *testing.T) {
	positions := func(width int) map[string]int {
		m := testModel()
		m.volumesList.Viewport.Width = width
		loadVolumes(m, fakeVolumes("openclaw_data", "pg_data"))
		header := m.volumesList.RenderHeader()
		out := map[string]int{}
		for _, col := range []string{"STACK", "DRIVER", "CREATED", "HOST", "MOUNT POINT"} {
			i := strings.Index(header, col)
			require.GreaterOrEqual(t, i, 0, "%q not in %q", col, header)
			out[col] = lipgloss.Width(header[:i])
		}
		return out
	}

	narrow, wide := positions(140), positions(240)
	require.Greater(t, wide["STACK"], narrow["STACK"],
		"NAME shares the leftover, so every later column starts further right")
	require.Greater(t, wide["MOUNT POINT"], narrow["MOUNT POINT"])

	m := testModel()
	m.volumesList.Viewport.Width = 240
	loadVolumes(m, fakeVolumes("openclaw_data"))
	require.Equal(t, 240, lipgloss.Width(m.volumesList.RenderRow(m.volumesList.Filtered[0], true)),
		"the row still spans the frame, so the selection highlight does too")
}
