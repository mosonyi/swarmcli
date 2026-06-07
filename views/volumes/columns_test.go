// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package volumesview

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/require"
)

func TestColWidths_SumToWidth(t *testing.T) {
	for _, w := range []int{60, 80, 100, 120, 200} {
		widths := (&Model{}).volumeColWidths(w)
		require.Len(t, widths, 6)
		sum := 0
		for _, cw := range widths {
			require.Greater(t, cw, 0)
			sum += cw
		}
		// columns + 5 two-space separators reconstruct the effective width.
		require.LessOrEqual(t, sum+5*2, w+1, "columns must fit within the available width at w=%d", w)
	}
}

func TestHeaderAndRow_Aligned(t *testing.T) {
	for _, w := range []int{80, 120} {
		m := testModel()
		m.volumesList.Viewport.Width = w
		loadVolumes(m, fakeVolumes("alpha"))

		header := m.renderVolumesHeader()
		row := m.volumesList.RenderItem(m.volumesList.Filtered[0], false, 0)
		require.Equal(t, lipgloss.Width(header), lipgloss.Width(row),
			"header and row widths must match at width %d", w)
	}
}
