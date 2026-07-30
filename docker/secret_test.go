// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/docker/docker/client"
	"github.com/stretchr/testify/require"
)

// The reference question answered for every secret in one service listing.
//
// ListServicesUsingSecretID/Name each list every service and filter, so asking
// about N secrets costs N listings — which is what the secrets view did on every
// load and every poll.
func TestServicesUsingSecretsIndexesInOneListing(t *testing.T) {
	var lists int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1.44/services", r.URL.Path)
		lists++
		w.Header().Set("Content-Type", "application/json")
		// web mounts the same secret at two paths; db mounts another; noop has no
		// ContainerSpec at all, which a real swarm does produce.
		_, _ = w.Write([]byte(`[
			{"ID":"web","Spec":{"Name":"web","TaskTemplate":{"ContainerSpec":{"Secrets":[
				{"SecretID":"s1","SecretName":"alpha"},
				{"SecretID":"s1","SecretName":"alpha"}
			]}}}},
			{"ID":"db","Spec":{"Name":"db","TaskTemplate":{"ContainerSpec":{"Secrets":[
				{"SecretID":"s2","SecretName":"beta"}
			]}}}},
			{"ID":"noop","Spec":{"Name":"noop","TaskTemplate":{}}}
		]`))
	}))
	defer ts.Close()

	cli, err := client.NewClientWithOpts(client.WithHost(ts.URL), client.WithVersion("1.44"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = cli.Close() })

	idx, err := ServicesUsingSecretsWith(context.Background(), cli)
	require.NoError(t, err)
	require.Equal(t, 1, lists, "one listing answers every secret")

	// Reachable by ID and by name, because a caller may hold either.
	require.Len(t, idx["s1"], 1)
	require.Equal(t, "web", idx["s1"][0].ID)
	require.Len(t, idx["alpha"], 1, "mounting one secret twice must not list the service twice")
	require.Equal(t, "web", idx["alpha"][0].ID)

	require.Len(t, idx["s2"], 1)
	require.Equal(t, "db", idx["s2"][0].ID)
	require.Len(t, idx["beta"], 1)

	// A secret nothing references is simply absent, so len() on the miss is 0.
	require.Empty(t, idx["s3"])
	// The service with no ContainerSpec contributed no keys and did not panic.
	require.Len(t, idx, 4)
}
