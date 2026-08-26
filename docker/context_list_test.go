// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestListContexts_MarksTheSessionContextCurrent — `docker context ls` reports
// whichever context ~/.docker/config.json names, which after a switch made in
// another terminal is not the one this session is talking to. Trusting it made
// the contexts view display the very mismatch #611 is about, and it is also
// where the view reads the context being left when a switch is confirmed.
func TestListContexts_MarksTheSessionContextCurrent(t *testing.T) {
	stubContextShow(t, "swarm-a")
	_, err := SessionContext()
	require.NoError(t, err)

	origList := listContextsFn
	t.Cleanup(func() { listContextsFn = origList })
	// swarm-b is what the config file now says. swarm-a is what we are using.
	listContextsFn = func() ([]byte, error) {
		return []byte(`{"Name":"swarm-a","Current":false,"Description":"a","DockerEndpoint":"tcp://a:2376"}
{"Name":"swarm-b","Current":true,"Description":"b","DockerEndpoint":"tcp://b:2376"}
`), nil
	}

	contexts, err := ListContexts()
	require.NoError(t, err)
	require.Len(t, contexts, 2)

	current := map[string]bool{}
	for _, c := range contexts {
		current[c.Name] = c.Current
	}
	require.True(t, current["swarm-a"], "the pinned context is the one in use")
	require.False(t, current["swarm-b"], "the config file's choice is not this session's")
}
