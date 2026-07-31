// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// The stack commands shell out, so a cancelled context is the only thing that
// can reach the running `docker` process — and what comes back has to say so.
// Left to itself the dead child reports "signal: killed", which is
// indistinguishable from a daemon that rejected the deploy; a controller being
// shut down would record a failed sync for work it cancelled itself.
func TestStackCommandsReportCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := DeployStackInContext(ctx, "no-such-context", "web", "services:\n  a:\n    image: x\n", ResolveImageDefault)
	require.ErrorIs(t, err, context.Canceled)

	require.ErrorIs(t, RemoveStackCLIInContext(ctx, "no-such-context", "web"), context.Canceled)
}
