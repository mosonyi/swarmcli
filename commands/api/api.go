// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package api

import (
	"context"

	"github.com/docker/docker/client"
)

// Context is the shared command execution context.
// It wraps a standard context.Context and provides
// additional handles like the Docker client.
type Context struct {
	App             any
	context.Context // embedded, so it implements context.Context
	DockerClient    *client.Client
	// add more later: Config, Logger, UI, etc.
}

// New creates a new API context with a background context.
func New(cli *client.Client) Context {
	return Context{
		Context:      context.Background(),
		DockerClient: cli,
	}
}

// WithContext creates a derived API context from an existing context.Context.
func WithContext(ctx context.Context, cli *client.Client) Context {
	return Context{
		Context:      ctx,
		DockerClient: cli,
	}
}

// WithCancel returns a cancellable derived context and a cancel func.
func WithCancel(cli *client.Client) (Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	return Context{
		Context:      ctx,
		DockerClient: cli,
	}, cancel
}
