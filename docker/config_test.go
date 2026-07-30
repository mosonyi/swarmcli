// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

import (
	"bytes"
	"compress/gzip"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/docker/docker/client"
	"github.com/stretchr/testify/require"
)

func TestConfigDisplayData(t *testing.T) {
	// gzip payloads (e.g. chart release records) are transparently decompressed.
	plain := []byte("name: whoami\nrevision: 1\n")
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, err := zw.Write(plain)
	require.NoError(t, err)
	require.NoError(t, zw.Close())

	gz := &ConfigWithDecodedData{Data: buf.Bytes()}
	require.Equal(t, plain, gz.DisplayData())

	// non-gzip payloads are returned unchanged.
	raw := &ConfigWithDecodedData{Data: []byte("plain config text")}
	require.Equal(t, []byte("plain config text"), raw.DisplayData())
}

// One list call, not a list plus an inspect per config.
//
// The daemon converts a listed config and an inspected one with the same
// function, so the listing already carries every field a caller can read. The
// inspect loop this replaces cost one round trip per config on every read of
// release history and on every poll of the configs view (issue #510).
func TestListConfigsDoesNotInspectEachOne(t *testing.T) {
	var lists, inspects int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1.44/configs" {
			inspects++
			w.WriteHeader(http.StatusNotFound)
			return
		}
		lists++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"ID":"c2","Version":{"Index":7},"CreatedAt":"2026-01-02T03:04:05Z","Spec":{"Name":"beta","Labels":{"k":"v"},"Data":"aGk="}},
			{"ID":"c1","Spec":{"Name":"alpha"}}
		]`))
	}))
	defer ts.Close()

	cli, err := client.NewClientWithOpts(client.WithHost(ts.URL), client.WithVersion("1.44"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = cli.Close() })

	got, err := ListConfigsWith(context.Background(), cli)
	require.NoError(t, err)
	require.Equal(t, 1, lists)
	require.Zero(t, inspects, "the listing already holds what an inspect would return")

	require.Len(t, got, 2)
	require.Equal(t, "alpha", got[0].Spec.Name, "results stay sorted by name")
	require.Equal(t, "beta", got[1].Spec.Name)

	// The fields the callers actually read, all from the list response.
	beta := got[1]
	require.Equal(t, "2026-01-02T03:04:05Z", beta.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"))
	require.Equal(t, uint64(7), beta.Version.Index)
	require.Equal(t, map[string]string{"k": "v"}, beta.Spec.Labels)
	require.Equal(t, []byte("hi"), beta.Spec.Data)
}

// The reference question answered for every config in one service listing.
//
// ListServicesUsingConfigID/Name each list every service and filter, so asking
// about N configs costs N listings — which is what the configs view did on every
// load and every poll.
func TestServicesUsingConfigsIndexesInOneListing(t *testing.T) {
	var lists int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1.44/services", r.URL.Path)
		lists++
		w.Header().Set("Content-Type", "application/json")
		// web mounts the same config at two paths; db mounts another; noop has no
		// ContainerSpec at all, which a real swarm does produce.
		_, _ = w.Write([]byte(`[
			{"ID":"web","Spec":{"Name":"web","TaskTemplate":{"ContainerSpec":{"Configs":[
				{"ConfigID":"c1","ConfigName":"alpha"},
				{"ConfigID":"c1","ConfigName":"alpha"}
			]}}}},
			{"ID":"db","Spec":{"Name":"db","TaskTemplate":{"ContainerSpec":{"Configs":[
				{"ConfigID":"c2","ConfigName":"beta"}
			]}}}},
			{"ID":"noop","Spec":{"Name":"noop","TaskTemplate":{}}}
		]`))
	}))
	defer ts.Close()

	cli, err := client.NewClientWithOpts(client.WithHost(ts.URL), client.WithVersion("1.44"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = cli.Close() })

	idx, err := ServicesUsingConfigsWith(context.Background(), cli)
	require.NoError(t, err)
	require.Equal(t, 1, lists, "one listing answers every config")

	// Reachable by ID and by name, because a caller may hold either.
	require.Len(t, idx["c1"], 1)
	require.Equal(t, "web", idx["c1"][0].ID)
	require.Len(t, idx["alpha"], 1, "mounting one config twice must not list the service twice")
	require.Equal(t, "web", idx["alpha"][0].ID)

	require.Len(t, idx["c2"], 1)
	require.Equal(t, "db", idx["c2"][0].ID)
	require.Len(t, idx["beta"], 1)

	// A config nothing references is simply absent, so len() on the miss is 0.
	require.Empty(t, idx["c3"])
	// The service with no ContainerSpec contributed no keys and did not panic.
	require.Len(t, idx, 4)
}
