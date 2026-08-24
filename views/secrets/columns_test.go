// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package secretsview

import (
	"testing"

	filterlist "github.com/Eldara-Tech/swarmcli/v2/ui/components/filterable/list"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/require"
)

const longSecretName = "a-very-long-secret-name-that-overflows-columns"

func readySecrets(m *Model, width int, names ...string) {
	loadSecrets(m, fakeSecrets(names...))
	m.setRenderItem()
	m.secretsList.Viewport.Width = width
	m.secretsList.Viewport.Height = 20
}

func TestUsedHeaderIsShortened(t *testing.T) {
	m := testModel()
	readySecrets(m, 120, "web")
	header := m.secretsList.RenderHeader()
	require.Contains(t, header, "USED")
	require.NotContains(t, header, "SECRET USED")
}

func TestContentAwareName_WideTerminal(t *testing.T) {
	m := testModel()
	readySecrets(m, 200, longSecretName, "tls")
	row := m.secretsList.RenderItem(m.secretsList.Filtered[0], false, 0)
	require.Contains(t, row, longSecretName, "full name must be visible on a wide terminal")
}

func TestHeaderRowAlignment(t *testing.T) {
	m := testModel()
	loadSecrets(m, fakeSecrets(longSecretName, "tls"))
	m.setRenderItem()
	for _, w := range []int{50, 120, 200} {
		m.secretsList.Viewport.Width = w
		header := m.secretsList.RenderHeader()
		row := m.secretsList.RenderItem(m.secretsList.Filtered[0], false, 0)
		require.Equal(t, lipgloss.Width(header), lipgloss.Width(row), "width=%d header/row mismatch", w)
	}
}

func TestArrowScrollMovesWindow(t *testing.T) {
	m := testModel()
	readySecrets(m, 60, longSecretName, "tls") // narrow → NAME overflows and scrolls
	before := m.secretsList.RenderItem(m.secretsList.Filtered[0], true, 0)
	m.Update(key("right"))
	after := m.secretsList.RenderItem(m.secretsList.Filtered[0], true, 0)
	require.NotEqual(t, before, after, "right arrow should shift the truncated cell window")

	m.Update(key("left"))
	back := m.secretsList.RenderItem(m.secretsList.Filtered[0], true, 0)
	require.Equal(t, before, back, "left arrow should restore the window")
}

func TestResetScrollOnCursorMove(t *testing.T) {
	m := testModel()
	readySecrets(m, 60, longSecretName, "tls")
	m.Update(key("right"))
	m.Update(key("right"))
	scrolledRow0 := m.secretsList.RenderItem(m.secretsList.Filtered[0], true, 0)

	m.Update(key("down")) // move to row 1 → scroll resets
	m.Update(key("up"))   // back to row 0, offset should be 0 again

	require.NotEqual(t, scrolledRow0, m.secretsList.RenderItem(m.secretsList.Filtered[0], true, 0),
		"scroll offset must reset when the cursor moves")
}

func TestScrollWindowUsesSharedHelper(t *testing.T) {
	// Sanity: the shared rune-aware window backs the labels behavior.
	require.Equal(t, "key=longvalue", filterlist.ScrollWindow("longkey=longvalue", 4, 20))
}
