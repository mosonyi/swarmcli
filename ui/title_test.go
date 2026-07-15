// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package ui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
	"github.com/stretchr/testify/require"
)

func TestScopedTitle_Format(t *testing.T) {
	cases := []struct {
		label, scope string
		count        int
		want         string
	}{
		{"Services", "postgres-ha", 8, "Services(postgres-ha)[8]"},
		{"Services", "all", 0, "Services(all)[0]"},
		{"Services", "no stack", 3, "Services(no stack)[3]"},
		{"Stacks", "all", 38, "Stacks(all)[38]"},
	}
	for _, c := range cases {
		require.Equal(t, c.want, ansi.Strip(ScopedTitle(c.label, c.scope, c.count)))
	}
}

func TestScopedTitle_TokenStyles(t *testing.T) {
	// Lock the colours the issue specifies for each token.
	require.Equal(t, lipgloss.Color("#8be4e4"), titleLabelStyle.GetForeground())
	require.Equal(t, lipgloss.Color("#ff04ff"), titleScopeStyle.GetForeground())
	require.Equal(t, lipgloss.Color("#ffefd5"), titleCountStyle.GetForeground())

	// Under a colour-capable profile each token is wrapped in its own style
	// (label/separators, scope, count) rather than a single uniform colour.
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(prev)

	out := ScopedTitle("Services", "postgres-ha", 8)
	require.Contains(t, out, titleLabelStyle.Render("Services"))
	require.Contains(t, out, titleScopeStyle.Render("postgres-ha"))
	require.Contains(t, out, titleCountStyle.Render("8"))
	require.Equal(t, "Services(postgres-ha)[8]", ansi.Strip(out))
}

func TestStyleFrameTitle_PassesThroughStyled(t *testing.T) {
	styled := "\x1b[95mpre-styled\x1b[0m"
	require.Equal(t, styled, styleFrameTitle(styled))
}

func TestStyleFrameTitle_StylesPlain(t *testing.T) {
	require.Equal(t, FrameTitleStyle.Render("plain"), styleFrameTitle("plain"))
}
