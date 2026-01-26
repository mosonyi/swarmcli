// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

//go:build integration

package service

import (
	"testing"
	"time"

	"swarmcli/docker"
)

// TestScaleWhoamiService verifies scaling demo_whoami up and down safely.
func TestScaleWhoamiService(t *testing.T) {
	const serviceName = "demo_whoami"
	const timeout = 45 * time.Second

	snap, err := docker.RefreshSnapshot()
	if err != nil {
		t.Fatalf("failed to refresh snapshot: %v", err)
	}

	svc := snap.FindServiceByName(serviceName)
	if svc == nil {
		t.Fatalf("service %s not found", serviceName)
	}
	if svc.Spec.Mode.Replicated == nil {
		t.Fatalf("service %s not in replicated mode", serviceName)
	}

	original := *svc.Spec.Mode.Replicated.Replicas
	if original < 2 {
		t.Fatalf("expected demo_whoami to start with at least 2 replicas, got %d", original)
	}

	scaleTarget := original - 1
	t.Logf("Scaling service %s down from %d → %d replicas", serviceName, original, scaleTarget)
	if err := docker.ScaleServiceByName(serviceName, scaleTarget); err != nil {
		t.Fatalf("failed to scale down: %v", err)
	}

	waitUntil := time.Now().Add(timeout)
	for {
		snap2, _ := docker.RefreshSnapshot()
		svc2 := snap2.FindServiceByName(serviceName)
		if svc2 == nil {
			t.Fatalf("service %s disappeared", serviceName)
		}
		running := 0
		for _, task := range snap2.Tasks {
			if task.ServiceID == svc2.ID && task.Status.State == "running" {
				running++
			}
		}
		if *svc2.Spec.Mode.Replicated.Replicas == scaleTarget && running == int(scaleTarget) {
			break
		}
		if time.Now().After(waitUntil) {
			t.Fatalf("timeout waiting for scale down")
		}
		time.Sleep(1 * time.Second)
	}

	// Restore
	t.Logf("Restoring service %s back to %d replicas", serviceName, original)
	if err := docker.ScaleServiceByName(serviceName, original); err != nil {
		t.Fatalf("failed to restore original count: %v", err)
	}
}

// TestRestartWhoamiSingleService verifies that scaling demo_whoami_single
// down to 0 and back to 1 correctly restarts the service.
func TestScaleWhoamiSingleServiceTo0AndThenTo1(t *testing.T) {
	const serviceName = "demo_whoami_single"
	const timeout = 45 * time.Second

	snap, err := docker.RefreshSnapshot()
	if err != nil {
		t.Fatalf("failed to refresh snapshot: %v", err)
	}

	svc := snap.FindServiceByName(serviceName)
	if svc == nil {
		t.Fatalf("service %s not found", serviceName)
	}
	if svc.Spec.Mode.Replicated == nil {
		t.Fatalf("service %s not in replicated mode", serviceName)
	}

	original := *svc.Spec.Mode.Replicated.Replicas
	if original != 1 {
		t.Fatalf("expected demo_whoami_single to start with 1 replica, got %d", original)
	}

	// Scale down to 0
	t.Logf("Scaling service %s down to 0 replicas", serviceName)
	if err := docker.ScaleServiceByName(serviceName, 0); err != nil {
		t.Fatalf("failed to scale down: %v", err)
	}

	waitUntil := time.Now().Add(timeout)
	for {
		snap2, _ := docker.RefreshSnapshot()
		svc2 := snap2.FindServiceByName(serviceName)
		if svc2 == nil {
			t.Fatalf("service %s disappeared", serviceName)
		}
		running := 0
		for _, task := range snap2.Tasks {
			if task.ServiceID == svc2.ID && task.Status.State == "running" {
				running++
			}
		}
		if *svc2.Spec.Mode.Replicated.Replicas == 0 && running == 0 {
			break
		}
		if time.Now().After(waitUntil) {
			t.Fatalf("timeout waiting for service %s to stop all tasks", serviceName)
		}
		time.Sleep(1 * time.Second)
	}

	// Scale back up to 1
	t.Logf("Scaling service %s back up to 1 replica", serviceName)
	if err := docker.ScaleServiceByName(serviceName, 1); err != nil {
		t.Fatalf("failed to scale up: %v", err)
	}

	waitUntil = time.Now().Add(timeout)
	for {
		snap3, _ := docker.RefreshSnapshot()
		svc3 := snap3.FindServiceByName(serviceName)
		if svc3 == nil {
			t.Fatalf("service %s disappeared", serviceName)
		}
		running := 0
		for _, task := range snap3.Tasks {
			if task.ServiceID == svc3.ID && task.Status.State == "running" {
				running++
			}
		}
		if *svc3.Spec.Mode.Replicated.Replicas == 1 && running == 1 {
			break
		}
		if time.Now().After(waitUntil) {
			t.Fatalf("timeout waiting for service %s to restart", serviceName)
		}
		time.Sleep(1 * time.Second)
	}

	t.Logf("✅ Service %s successfully restarted (0 → 1 replica)", serviceName)
}

func TestScaleServiceUnchanged(t *testing.T) {
	const svcName = "demo_whoami_single"

	// Get current replicas
	err := docker.ScaleServiceByName(svcName, 1)
	if err != nil {
		t.Fatalf("initial scale failed: %v", err)
	}

	// Scale to the same count again
	err = docker.ScaleServiceByName(svcName, 1)
	if err != nil {
		t.Fatalf("scaling to same count failed: %v", err)
	}
}
