// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

//go:build integration_locked

// Package swarmlock holds the locked-swarm integration test. It is gated behind
// the dedicated `integration_locked` build tag (not the regular `integration`
// tag) because it autolocks and restarts its own throwaway Docker daemon, which
// must stay isolated from the shared multi-node swarm the other integration
// tests run against. A separate CI job runs only this tag.
package swarmlock

import (
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/Eldara-Tech/swarmcli/docker"
)

const (
	dindImage     = "docker:29-dind"
	containerName = "swarmcli-locked-dind"
	dindHostPort  = "23375" // distinct from the shared swarm's 22375
	lockedContext = "swarmcli-locked"
	contextHost   = "tcp://localhost:" + dindHostPort
)

// hostDocker runs a docker CLI command against the runner's default daemon
// (the one hosting the throwaway dind container), regardless of any ambient
// DOCKER_CONTEXT. It returns combined output for diagnostics.
func hostDocker(t *testing.T, args ...string) (string, error) {
	t.Helper()
	full := append([]string{"--context", "default"}, args...)
	out, err := exec.Command("docker", full...).CombinedOutput()
	return string(out), err
}

// inner runs `docker <args>` *inside* the dind container, i.e. against the
// daemon under test.
func inner(t *testing.T, args ...string) (string, error) {
	t.Helper()
	return hostDocker(t, append([]string{"exec", containerName, "docker"}, args...)...)
}

// waitInnerReady polls until the inner daemon answers `docker version`.
func waitInnerReady(t *testing.T, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if _, err := inner(t, "version"); err == nil {
			return
		}
		time.Sleep(time.Second)
	}
	t.Fatalf("inner docker daemon did not become ready within %s", timeout)
}

func TestLockedSwarm_SwitchSnapshotAndUnlock(t *testing.T) {
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not available; skipping locked-swarm integration test")
	}

	// Clean any leftovers from a prior aborted run, then ensure teardown.
	_, _ = hostDocker(t, "rm", "-f", containerName)
	_ = exec.Command("docker", "context", "rm", "-f", lockedContext).Run()
	t.Cleanup(func() {
		docker.ResetClient()
		// The context must not be current when we remove it.
		_ = exec.Command("docker", "context", "use", "default").Run()
		_ = exec.Command("docker", "context", "rm", "-f", lockedContext).Run()
		_, _ = hostDocker(t, "rm", "-f", containerName)
	})

	// 1. Start a throwaway single-node dind daemon with plain-TCP on 2375.
	if out, err := hostDocker(t, "run", "-d", "--privileged",
		"-e", "DOCKER_TLS_CERTDIR=",
		"-p", dindHostPort+":2375",
		"--name", containerName, dindImage); err != nil {
		t.Fatalf("starting dind: %v\n%s", err, out)
	}
	waitInnerReady(t, 60*time.Second)

	// 2. Init a swarm and turn on autolock, then capture the unlock key.
	if out, err := inner(t, "swarm", "init", "--advertise-addr", "eth0"); err != nil {
		t.Fatalf("swarm init: %v\n%s", err, out)
	}
	if out, err := inner(t, "swarm", "update", "--autolock=true"); err != nil {
		t.Fatalf("enabling autolock: %v\n%s", err, out)
	}
	keyOut, err := inner(t, "swarm", "unlock-key", "-q")
	if err != nil {
		t.Fatalf("reading unlock key: %v\n%s", err, keyOut)
	}
	unlockKey := strings.TrimSpace(keyOut)
	if !strings.HasPrefix(unlockKey, "SWMKEY-") {
		t.Fatalf("unexpected unlock key %q", unlockKey)
	}

	// 3. Restart the daemon so it comes back LOCKED.
	if out, err := hostDocker(t, "restart", containerName); err != nil {
		t.Fatalf("restarting dind: %v\n%s", err, out)
	}
	waitInnerReady(t, 60*time.Second)

	// 4. Point swarmcli's client at the locked daemon via a Docker context.
	if out, err := hostDocker(t, "context", "create", lockedContext,
		"--docker", "host="+contextHost); err != nil {
		t.Fatalf("creating context: %v\n%s", err, out)
	}
	t.Setenv("DOCKER_CONTEXT", lockedContext)
	docker.ResetClient()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 5. Switching into a locked swarm must succeed (CLI parity).
	if err := docker.ValidateContext(ctx, lockedContext); err != nil {
		t.Fatalf("ValidateContext should allow a locked swarm, got: %v", err)
	}

	// 6. The snapshot must report Locked with no entities (no fatal error).
	snap, err := docker.RefreshSnapshot()
	if err != nil {
		t.Fatalf("RefreshSnapshot on locked swarm returned error: %v", err)
	}
	if !snap.Locked {
		t.Fatalf("expected snapshot.Locked=true on a locked swarm")
	}
	if len(snap.Nodes) != 0 {
		t.Fatalf("expected no nodes while locked, got %d", len(snap.Nodes))
	}

	// 7. Unlocking with the captured key must succeed and clear the locked state.
	if err := docker.UnlockSwarm(ctx, unlockKey); err != nil {
		t.Fatalf("UnlockSwarm failed: %v", err)
	}

	// 8. After unlocking, the swarm becomes usable; the node list resolves.
	var after *docker.SwarmSnapshot
	deadline := time.Now().Add(20 * time.Second)
	for {
		docker.InvalidateSnapshot()
		after, err = docker.RefreshSnapshot()
		if err == nil && !after.Locked {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("swarm still locked after unlock (err=%v, locked=%v)", err, after != nil && after.Locked)
		}
		time.Sleep(time.Second)
	}
	if len(after.Nodes) < 1 {
		t.Fatalf("expected at least one node after unlock, got %d", len(after.Nodes))
	}
}
