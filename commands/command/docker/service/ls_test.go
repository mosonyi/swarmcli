// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package service

import (
	"testing"

	"swarmcli/args"
	"swarmcli/registry"
	servicesview "swarmcli/views/services"
	"swarmcli/views/view"

	"github.com/stretchr/testify/require"
)

func TestServiceLs_Name(t *testing.T) {
	require.Equal(t, "service", lsCmd.Name())
}

func TestServiceLs_Description(t *testing.T) {
	require.NotEmpty(t, lsCmd.Description())
}

func TestServiceLs_ExecuteNavigatesToAllServices(t *testing.T) {
	cmd := lsCmd.Execute(nil, args.Args{})
	require.NotNil(t, cmd)

	nav, ok := cmd().(view.NavigateToMsg)
	require.True(t, ok, "expected a NavigateToMsg")
	require.Equal(t, servicesview.ViewName, nav.ViewName)
	// A nil payload makes the services factory default to AllFilter (all stacks).
	require.Nil(t, nav.Payload)
}

func TestServiceLs_RegisteredWithSvcAlias(t *testing.T) {
	_, ok := registry.Get("service")
	require.True(t, ok)

	alias, ok := registry.Get("svc")
	require.True(t, ok)
	a, ok := alias.(registry.Aliaser)
	require.True(t, ok)
	require.Equal(t, "service", a.AliasOf())
}
