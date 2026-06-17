// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

import (
	"context"
	"strings"

	"github.com/docker/docker/api/types/swarm"
)

// IsSwarmLockedErr reports whether err is the Docker daemon's "swarm is locked"
// error. A locked swarm is reachable (ping/info succeed) but every store-backed
// call (node/service/task/stack list, ...) fails with this message until the
// swarm is unlocked. See docs.docker.com/engine/swarm/swarm_manager_locking.
func IsSwarmLockedErr(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "swarm is encrypted") ||
		strings.Contains(s, "needs to be unlocked")
}

// isUsableSwarmState reports whether a node state lets swarmcli switch into the
// context. Locked is usable: the node IS a swarm member, it just needs unlocking.
func isUsableSwarmState(s swarm.LocalNodeState) bool {
	return s == swarm.LocalNodeStateActive || s == swarm.LocalNodeStateLocked
}

// UnlockSwarm submits the unlock key to the daemon for the current context.
func UnlockSwarm(ctx context.Context, key string) error {
	cli, err := GetClient()
	if err != nil {
		return err
	}
	return cli.SwarmUnlock(ctx, swarm.UnlockRequest{UnlockKey: strings.TrimSpace(key)})
}
