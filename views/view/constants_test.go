// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package view

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsTopLevel(t *testing.T) {
	topLevel := []string{
		NameStacks, NameNodes, NameConfigs, NameSecrets,
		NameNetworks, NameVolumes, NameContexts, NameLoading, NameHelp,
		NameCharts,
	}
	for _, name := range topLevel {
		require.True(t, IsTopLevel(name), "expected %q to be top-level", name)
	}

	nested := []string{NameServices, NameTasks, NameInspect, NameLogs}
	for _, name := range nested {
		require.False(t, IsTopLevel(name), "expected %q to NOT be top-level", name)
	}

	require.False(t, IsTopLevel("nonexistent"))
}
