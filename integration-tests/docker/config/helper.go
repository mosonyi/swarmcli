// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

//go:build integration

package config

import (
	"context"
	"fmt"
	"swarmcli/docker"
	"testing"
	"time"

	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
	"github.com/stretchr/testify/require"
)

type testEnv struct {
	ctx     context.Context
	cli     *client.Client
	cleanup []func()
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	ctx := context.Background()
	cli, err := docker.GetClient()
	require.NoError(t, err)
	env := &testEnv{ctx: ctx, cli: cli}
	return env
}

func (e *testEnv) cleanupAll(t *testing.T) {
	for _, fn := range e.cleanup {
		fn()
	}
	_ = e.cli.Close()
}

func (e *testEnv) registerConfigCleanup(id string) {
	e.cleanup = append(e.cleanup, func() {
		_ = e.cli.ConfigRemove(e.ctx, id)
	})
}

func (e *testEnv) registerServiceCleanup(id string) {
	e.cleanup = append(e.cleanup, func() {
		_ = e.cli.ServiceRemove(e.ctx, id)
	})
}

func (e *testEnv) createConfig(t *testing.T, name, data string) swarm.Config {
	t.Helper()

	spec := swarm.ConfigSpec{
		Annotations: swarm.Annotations{Name: name},
		Data:        []byte(data),
	}

	resp, err := e.cli.ConfigCreate(e.ctx, spec)
	require.NoError(t, err, "failed to create config %s", name)

	e.registerConfigCleanup(resp.ID)

	cfg, _, err := e.cli.ConfigInspectWithRaw(e.ctx, resp.ID)
	require.NoError(t, err, "failed to inspect config %s", name)

	return cfg
}

func uniqueName(base string) string {
	return fmt.Sprintf("%s-%d", base, time.Now().UnixNano())
}
