//go:build integration

package service

import (
	"context"
	"testing"
	"time"

	"swarmcli/docker"
)

func TestRestartServiceWithProgress(t *testing.T) {
	cases := []struct {
		name        string
		serviceName string
		timeout     time.Duration
	}{
		{"single replica", "demo_whoami_single", 45 * time.Second},
		{"multi replica", "demo_whoami", 2 * time.Minute},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), tc.timeout)
			defer cancel()

			progressCh := make(chan docker.ProgressUpdate)

			// Start restart in a goroutine
			go func() {
				err := docker.RestartServiceWithProgress(ctx, tc.serviceName, progressCh)
				if err != nil {
					t.Logf("RestartServiceWithProgress returned error: %v", err)
				}
				close(progressCh)
			}()

			var lastProgress docker.ProgressUpdate
			for update := range progressCh {
				t.Logf("Progress: %d/%d", update.Replaced, update.Total)
				lastProgress = update
			}

			// Verify that all replicas were replaced
			if lastProgress.Replaced != lastProgress.Total || lastProgress.Total == 0 {
				t.Fatalf("expected all tasks replaced, got %d/%d", lastProgress.Replaced, lastProgress.Total)
			}

			t.Logf("✅ Service %s restarted successfully with %d/%d tasks replaced", tc.serviceName, lastProgress.Replaced, lastProgress.Total)
		})
	}
}
