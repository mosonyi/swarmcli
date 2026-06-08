// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package volumesview

import (
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
