// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package textdiff

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLineDiff(t *testing.T) {
	a := "a\nb\nc\n"
	b := "a\nB\nc\nd\n"
	d := Lines(a, b)
	lines := strings.Split(strings.TrimRight(d, "\n"), "\n")
	require.Equal(t, "  a", lines[0])
	require.Contains(t, d, "- b")
	require.Contains(t, d, "+ B")
	require.Contains(t, d, "+ d")
}

func TestLineDiffIdentical(t *testing.T) {
	d := Lines("x\ny\n", "x\ny\n")
	require.Equal(t, "  x\n  y\n", d)
}
