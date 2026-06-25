// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package command

import (
	"testing"

	"swarmcli/args"
	"swarmcli/views/view"

	"github.com/stretchr/testify/require"
)

func TestDevUpdateEnabledGate(t *testing.T) {
	t.Setenv("SWARMCLI_ENV", "dev")
	require.True(t, devUpdateEnabled())

	t.Setenv("SWARMCLI_ENV", "prod")
	require.False(t, devUpdateEnabled())

	t.Setenv("SWARMCLI_ENV", "")
	require.False(t, devUpdateEnabled())
}

func TestDevUpdateExecuteEmitsOpenMsg(t *testing.T) {
	msg := DevUpdate{}.Execute(nil, args.Args{})()
	open, ok := msg.(view.OpenUpdateDialogMsg)
	require.True(t, ok)
	require.Equal(t, "", open.Version)

	msg = DevUpdate{}.Execute(nil, args.Args{Positionals: []string{"v9.9.9"}})()
	open, ok = msg.(view.OpenUpdateDialogMsg)
	require.True(t, ok)
	require.Equal(t, "v9.9.9", open.Version)
}
