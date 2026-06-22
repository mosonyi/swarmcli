// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package cli

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLineDiff(t *testing.T) {
	a := "a\nb\nc\n"
	b := "a\nB\nc\nd\n"
	d := lineDiff(a, b)
	lines := strings.Split(strings.TrimRight(d, "\n"), "\n")
	require.Equal(t, "  a", lines[0])
	require.Contains(t, d, "- b")
	require.Contains(t, d, "+ B")
	require.Contains(t, d, "+ d")
}

func TestLineDiffIdentical(t *testing.T) {
	d := lineDiff("x\ny\n", "x\ny\n")
	require.Equal(t, "  x\n  y\n", d)
}

func TestUpgradeFlagsParse(t *testing.T) {
	_, f, err := parseArgs([]string{"rel", "repo/chart", "--install", "--reuse-values", "--revision", "3"})
	require.NoError(t, err)
	require.True(t, f.install)
	require.True(t, f.reuseValues)
	require.Equal(t, 3, f.revision)
}
