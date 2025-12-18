//go:build integration

package docker

import (
	"os/exec"
	"testing"

	"swarmcli/docker"
)

func TestGetContextFromEnv_EnvOverride(t *testing.T) {
	const want = "ci-test-context"
	t.Setenv("DOCKER_CONTEXT", want)

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

	// Ensure env is empty so the function falls back to calling
	// `docker context show`.
	t.Setenv("DOCKER_CONTEXT", "")

	ctxFromFunc, err := docker.GetContextFromEnv()
	if err != nil {
		t.Fatalf("GetContextFromEnv failed: %v", err)
	}

	// Compare against the public helper which also queries docker.
	ctxCurrent, err := docker.GetCurrentContext()
	if err != nil {
		t.Fatalf("GetCurrentContext failed: %v", err)
	}

	if ctxFromFunc != ctxCurrent {
		t.Fatalf("mismatch: GetContextFromEnv=%q GetCurrentContext=%q", ctxFromFunc, ctxCurrent)
	}
}
