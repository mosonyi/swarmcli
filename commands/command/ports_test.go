// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package command

import (
	"testing"

	"swarmcli/args"
	"swarmcli/registry"

	"github.com/stretchr/testify/require"
)

func TestPorts_Name(t *testing.T) {
	require.Equal(t, "ports", portsCmd.Name())
}

func TestPorts_Description(t *testing.T) {
	require.NotEmpty(t, portsCmd.Description())
}

func TestPorts_Execute(t *testing.T) {
	cmd := portsCmd.Execute(nil, args.Args{})
	require.NotNil(t, cmd)
}

func TestPorts_RegisteredAlias(t *testing.T) {
	cmd, ok := registry.Get("port")
	require.True(t, ok)
	a, ok := cmd.(registry.Aliaser)
	require.True(t, ok)
	require.Equal(t, "ports", a.AliasOf())
}
