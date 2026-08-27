// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

//go:build integration

package docker

import (
	"os/exec"
	"testing"

	"github.com/Eldara-Tech/swarmcli/v2/docker"
)

func TestGetContextFromEnv_EnvOverride(t *testing.T) {
	const want = "ci-test-context"
	t.Setenv("DOCKER_CONTEXT", want)
	// The context is resolved once per process and then pinned, so a test that
	// changes what the resolution would return has to drop the pin first — and
	// leave the next test to resolve its own.
	docker.ResetSessionContext()
	t.Cleanup(docker.ResetSessionContext)

	ctx, err := docker.GetContextFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ctx != want {
		t.Fatalf("expected context %q, got %q", want, ctx)
	}
}

func TestGetContextFromEnv_FallbackToDocker(t *testing.T) {
	// Ensure docker is available; otherwise skip this integration test.
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not available; skipping integration test")
	}

	// Ensure env is empty so the resolution falls back to calling
	// `docker context show`.
	t.Setenv("DOCKER_CONTEXT", "")
	docker.ResetSessionContext()
	t.Cleanup(docker.ResetSessionContext)

	ctxFromFunc, err := docker.GetContextFromEnv()
	if err != nil {
		t.Fatalf("GetContextFromEnv failed: %v", err)
	}

	// Compare against the public helper. Both report the session pin, and that
	// they cannot disagree is the point: they used to make the lookup
	// separately, so a `docker context use` between the two calls moved one and
	// not the other.
	ctxCurrent, err := docker.GetCurrentContext()
	if err != nil {
		t.Fatalf("GetCurrentContext failed: %v", err)
	}

	if ctxFromFunc != ctxCurrent {
		t.Fatalf("mismatch: GetContextFromEnv=%q GetCurrentContext=%q", ctxFromFunc, ctxCurrent)
	}
}

// TestSessionContext_SurvivesAContextSwitch is the daemon-backed half of the
// #611 regression test: a real `docker context use` run against the real config
// file must not move a session that has already resolved its context.
func TestSessionContext_SurvivesAContextSwitch(t *testing.T) {
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not available; skipping integration test")
	}
	t.Setenv("DOCKER_CONTEXT", "")
	docker.ResetSessionContext()
	t.Cleanup(docker.ResetSessionContext)

	pinned, err := docker.SessionContext()
	if err != nil {
		t.Fatalf("SessionContext failed: %v", err)
	}

	other := "default"
	if pinned == other {
		t.Skip("already on the default context; nothing to switch away to")
	}
	if err := exec.Command("docker", "context", "use", other).Run(); err != nil {
		t.Skipf("could not switch context: %v", err)
	}
	t.Cleanup(func() {
		_ = exec.Command("docker", "context", "use", pinned).Run()
	})

	live, err := docker.ConfigFileContext()
	if err != nil {
		t.Fatalf("ConfigFileContext failed: %v", err)
	}
	if live != other {
		t.Fatalf("expected the config file to name %q, got %q", other, live)
	}

	got, err := docker.SessionContext()
	if err != nil {
		t.Fatalf("SessionContext failed: %v", err)
	}
	if got != pinned {
		t.Fatalf("the session moved with the config file: pinned %q, now %q", pinned, got)
	}
}
