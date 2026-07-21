// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package charts

import (
	"testing"

	"github.com/docker/docker/api/types/swarm"
	"github.com/stretchr/testify/require"

	"github.com/Eldara-Tech/swarmcli/docker"
)

// A named backend addresses the context it was constructed with, not the one
// the process is pointed at. DOCKER_CONTEXT is set to something else here
// precisely so a fallback would be visible.
func TestNamedBackendIgnoresTheAmbientContext(t *testing.T) {
	t.Setenv("DOCKER_CONTEXT", "ambient")

	b, ok := NewDockerBackend("swarm-b").(*dockerBackend)
	require.True(t, ok)

	name, err := b.contextName()
	require.NoError(t, err)
	require.Equal(t, "swarm-b", name)
}

// The default backend keeps resolving the ambient context, so the CLI behaves
// exactly as it did.
func TestAmbientBackendResolvesTheProcessContext(t *testing.T) {
	t.Setenv("DOCKER_CONTEXT", "ambient")

	name, err := (&dockerBackend{}).contextName()
	require.NoError(t, err)
	require.Equal(t, "ambient", name)
}

// A named backend must not write to the shared snapshot cache. The cache holds
// exactly one swarm, so a reconcile against a second one that refreshed it
// would replace another swarm's state with its own — and every reader of the
// cache, including a TUI in the same process, would silently follow.
func TestNamedBackendRefreshDoesNotTouchTheSharedCache(t *testing.T) {
	t.Cleanup(docker.InvalidateSnapshot)
	mine := &docker.SwarmSnapshot{Services: []swarm.Service{{ID: "sentinel"}}}
	docker.SetSnapshot(mine)

	require.NoError(t, NewDockerBackend("swarm-b").RefreshSnapshot())

	got := docker.GetSnapshot()
	require.NotNil(t, got)
	require.Len(t, got.Services, 1)
	require.Equal(t, "sentinel", got.Services[0].ID, "the shared cache must be untouched")
}
