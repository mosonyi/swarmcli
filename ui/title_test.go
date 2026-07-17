// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package ui

import (
	"strings"
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

func TestScopedTitleFiltered_NoFilter_EqualsScopedTitle(t *testing.T) {
	// With no active filter the title is identical to the plain ScopedTitle.
	require.Equal(t, ScopedTitle("Stacks", "all", 3), ScopedTitleFiltered("Stacks", "all", 3, ""))
}

func TestScopedTitleFiltered_WithFilter_AppendsFragment(t *testing.T) {
	// k9s-style: scope kept, count is the (caller-supplied, post-filter) value,
	// filter appended as a " </query>" fragment rather than folded into scope.
	out := ScopedTitleFiltered("Stacks", "all", 1, "pos")
	require.Equal(t, "Stacks(all)[1] </pos>", ansi.Strip(out))
}

func TestFilterFragment_Empty(t *testing.T) {
	require.Equal(t, "", FilterFragment(""))
}

func TestFilterFragment_Truncates(t *testing.T) {
	stripped := ansi.Strip(FilterFragment(strings.Repeat("x", 40)))
	require.Contains(t, stripped, "…")
	// " </" + at most maxFilterFragmentRunes runes + ">".
	require.LessOrEqual(t, len([]rune(stripped)), len(" </>")+maxFilterFragmentRunes)
}

func TestFilterFragment_TokenStyles(t *testing.T) {
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(prev)

	out := FilterFragment("pos")
	require.Contains(t, out, titleScopeStyle.Render("/pos")) // "/query" reuses the scope colour
	require.Contains(t, out, titleLabelStyle.Render("<"))    // brackets share the label colour
}

func TestStyleFrameTitle_PassesThroughStyled(t *testing.T) {
	styled := "\x1b[95mpre-styled\x1b[0m"
	require.Equal(t, styled, styleFrameTitle(styled))
}

func TestStyleFrameTitle_StylesPlain(t *testing.T) {
	require.Equal(t, FrameTitleStyle.Render("plain"), styleFrameTitle("plain"))
}
