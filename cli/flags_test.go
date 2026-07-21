// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package cli

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUpgradeFlagsParse(t *testing.T) {
	_, f, err := parseArgs([]string{"rel", "repo/chart", "--install", "--reuse-values", "--revision", "3"})
	require.NoError(t, err)
	require.True(t, f.install)
	require.True(t, f.reuseValues)
	require.Equal(t, 3, f.revision)
}

func TestParseArgsResolveImage(t *testing.T) {
	for _, mode := range []string{"always", "changed", "never"} {
		_, f, err := parseArgs([]string{"rel", "repo/chart", "--resolve-image", mode})
		require.NoError(t, err)
		require.Equal(t, mode, f.resolveImage)
	}

	_, f, err := parseArgs([]string{"rel", "repo/chart", "--resolve-image=changed"})
	require.NoError(t, err)
	require.Equal(t, "changed", f.resolveImage)

	// Unset means "pass no flag", leaving Docker's own default of always.
	_, f, err = parseArgs([]string{"rel", "repo/chart"})
	require.NoError(t, err)
	require.Empty(t, f.resolveImage)
}

// A typo must fail loudly rather than silently deploying with the default,
// matching the strict-flag philosophy of the other charts flags.
func TestParseArgsResolveImageRejectsGarbage(t *testing.T) {
	for _, bad := range []string{"latest", "Always", ""} {
		_, _, err := parseArgs([]string{"rel", "repo/chart", "--resolve-image=" + bad})
		require.Error(t, err, "mode %q should be rejected", bad)
		require.Contains(t, err.Error(), "--resolve-image")
	}
}
