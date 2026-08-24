// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package command

import (
	"testing"

	"github.com/Eldara-Tech/swarmcli/v2/args"
	"github.com/Eldara-Tech/swarmcli/v2/registry"

	"github.com/stretchr/testify/require"
)

func TestQuit_Name(t *testing.T) {
	require.Equal(t, "quit", quitCmd.Name())
}

func TestQuit_Description(t *testing.T) {
	require.NotEmpty(t, quitCmd.Description())
}

func TestQuit_Execute(t *testing.T) {
	cmd := quitCmd.Execute(nil, args.Args{})
	require.NotNil(t, cmd)
}

func TestQuit_RegisteredAliasQ(t *testing.T) {
	cmd, ok := registry.Get("q")
	require.True(t, ok)
	a, ok := cmd.(registry.Aliaser)
	require.True(t, ok)
	require.Equal(t, "quit", a.AliasOf())
}
