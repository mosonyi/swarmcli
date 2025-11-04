//go:build integration

package service

import (
	"context"
	"testing"
	"time"

	"swarmcli/docker"
)

// restartTestCase defines one restart scenario.
type restartTestCase struct {
	name         string
	serviceName  string
	expectAllNew bool // true = all replicas must be replaced
	timeout      time.Duration
}

// TestRestartServiceAndWait_Parametrized verifies both single- and multi-replica
// service restart behavior, using RestartServiceAndWait as the restart mechanism.
func TestRestartServiceAndWait(t *testing.T) {
	cases := []restartTestCase{
		{
			name:         "single replica service (demo_whoami_single)",
			serviceName:  "demo_whoami_single",
			expectAllNew: true,
			timeout:      45 * time.Second,
		},
		{
			name:         "multi replica service (demo_whoami)",
			serviceName:  "demo_whoami",
			expectAllNew: true, // wait until *all* replicas replaced
			timeout:      2 * time.Minute,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), tc.timeout)
			defer cancel()

			// Initial snapshot
			snap, err := docker.RefreshSnapshot()
			if err != nil {
				t.Fatalf("failed to refresh snapshot: %v", err)
			}

			svc := snap.FindServiceByName(tc.serviceName)
			if svc == nil {
				t.Fatalf("service %s not found", tc.serviceName)
			}
			if svc.Spec.Mode.Replicated == nil {
				t.Fatalf("service %s not in replicated mode", tc.serviceName)
			}

			replicas := *svc.Spec.Mode.Replicated.Replicas
			oldTasks := map[string]bool{}
			for _, task := range snap.Tasks {
				if task.ServiceID == svc.ID && task.Status.State == "running" {
					oldTasks[task.ID] = true
				}
			}
			t.Logf("📦 Found %d old running tasks for %s", len(oldTasks), tc.serviceName)

			// Restart idiomatically and wait
			t.Logf("🔁 Restarting service %s (replicas: %d)...", tc.serviceName, replicas)
			start := time.Now()
			if err := docker.RestartServiceAndWait(ctx, tc.serviceName); err != nil {
				t.Fatalf("failed to restart service: %v", err)
			}

			// Verify new tasks appeared
			snap2, err := docker.RefreshSnapshot()
			if err != nil {
				t.Fatalf("failed to refresh snapshot after restart: %v", err)
			}

			svc2 := snap2.FindServiceByName(tc.serviceName)
			if svc2 == nil {
				t.Fatalf("service %s disappeared after restart", tc.serviceName)
			}

			newTasks := map[string]bool{}
			for _, task := range snap2.Tasks {
				if task.ServiceID == svc2.ID && task.Status.State == "running" {
					newTasks[task.ID] = true
				}
			}

			if len(newTasks) != int(replicas) {
				t.Fatalf("expected %d running tasks after restart, got %d", replicas, len(newTasks))
			}

			// Determine how many tasks changed
			changed := 0
			for id := range newTasks {
				if !oldTasks[id] {
					changed++
				}
			}

			if tc.expectAllNew {
				if changed != len(newTasks) {
					t.Fatalf("expected all %d tasks replaced, but only %d changed", len(newTasks), changed)
				}
			} else {
				if changed == 0 {
					t.Fatalf("expected at least one task replaced, but none changed")
				}
			}

			t.Logf("✅ %s successfully restarted (%d/%d new tasks) in %v",
				tc.serviceName, changed, len(newTasks), time.Since(start))
		})
	}
}

// TestRestartServiceAndWait_Timeout verifies that RestartServiceAndWait
// returns a context timeout error when the deadline is exceeded.
func TestRestartServiceAndWait_Timeout(t *testing.T) {
	const serviceName = "demo_whoami_single"

	// Intentionally use a very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	start := time.Now()
	err := docker.RestartServiceAndWait(ctx, serviceName)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatalf("expected timeout error, got nil")
	}
	t.Logf("✅ RestartServiceAndWait correctly timed out after %v: %v", elapsed, err)
}

// TestRestartServiceAndWait_ServiceNotFound checks behaviour for a missing service.
func TestRestartServiceAndWait_ServiceNotFound(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := docker.RestartServiceAndWait(ctx, "nonexistent_demo_service")
	if err == nil {
		t.Fatalf("expected error when restarting nonexistent service, got nil")
	}
	t.Logf("✅ Correctly returned error for nonexistent service: %v", err)
}
