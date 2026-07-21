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
