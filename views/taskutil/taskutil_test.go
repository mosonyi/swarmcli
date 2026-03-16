// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package taskutil

import (
	"testing"
	"time"

	"github.com/docker/docker/api/types/swarm"
)

func makeTask(serviceID string, slot int, desired, actual swarm.TaskState, ts time.Time, err string) swarm.Task {
	return swarm.Task{
		ServiceID:    serviceID,
		Slot:         slot,
		DesiredState: desired,
		Status: swarm.TaskStatus{
			State:     actual,
			Timestamp: ts,
			Err:       err,
		},
	}
}

func TestTaskKeyForService_Replicated(t *testing.T) {
	task := swarm.Task{ServiceID: "svc1", Slot: 3}
	if got := TaskKeyForService(task); got != "svc1:3" {
		t.Errorf("got %q, want %q", got, "svc1:3")
	}
}

func TestTaskKeyForService_Global(t *testing.T) {
	task := swarm.Task{ServiceID: "svc1", Slot: 0, NodeID: "node1"}
	if got := TaskKeyForService(task); got != "svc1:node1" {
		t.Errorf("got %q, want %q", got, "svc1:node1")
	}
}

func TestTaskTimestamp_UsesStatusTimestamp(t *testing.T) {
	ts := time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC)
	task := swarm.Task{
		Meta:   swarm.Meta{CreatedAt: ts.Add(-time.Hour)},
		Status: swarm.TaskStatus{Timestamp: ts},
	}
	if got := TaskTimestamp(task); !got.Equal(ts) {
		t.Errorf("got %v, want %v", got, ts)
	}
}

func TestTaskTimestamp_FallsBackToCreatedAt(t *testing.T) {
	ts := time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC)
	task := swarm.Task{
		Meta: swarm.Meta{CreatedAt: ts},
	}
	if got := TaskTimestamp(task); !got.Equal(ts) {
		t.Errorf("got %v, want %v", got, ts)
	}
}

func TestLatestTasksByServiceKey_PrefersRunning(t *testing.T) {
	now := time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC)
	tasks := []swarm.Task{
		makeTask("svc1", 1, swarm.TaskStateShutdown, swarm.TaskStateFailed, now.Add(time.Second), "boom"),
		makeTask("svc1", 1, swarm.TaskStateRunning, swarm.TaskStateRunning, now, ""),
	}
	result := LatestTasksByServiceKey(tasks)
	if len(result) != 1 {
		t.Fatalf("expected 1 task, got %d", len(result))
	}
	if result[0].Status.State != swarm.TaskStateRunning {
		t.Errorf("expected running task to win, got %s", result[0].Status.State)
	}
}

func TestLatestTasksByServiceKey_SameTierNewerWins(t *testing.T) {
	now := time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC)
	tasks := []swarm.Task{
		makeTask("svc1", 1, swarm.TaskStateRunning, swarm.TaskStateRunning, now, ""),
		makeTask("svc1", 1, swarm.TaskStateRunning, swarm.TaskStateRunning, now.Add(time.Minute), ""),
	}
	result := LatestTasksByServiceKey(tasks)
	if len(result) != 1 {
		t.Fatalf("expected 1 task, got %d", len(result))
	}
	if !result[0].Status.Timestamp.Equal(now.Add(time.Minute)) {
		t.Error("expected newer task to win")
	}
}

func TestActiveDeploymentErrors_EmptyInput(t *testing.T) {
	result := ActiveDeploymentErrorsByService(nil)
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestActiveDeploymentErrors_FailedRollingUpdate(t *testing.T) {
	now := time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC)
	tasks := []swarm.Task{
		// Old task still running
		makeTask("svc1", 1, swarm.TaskStateRunning, swarm.TaskStateRunning, now.Add(-time.Minute), ""),
		// New task failed (newer than running)
		makeTask("svc1", 1, swarm.TaskStateRunning, swarm.TaskStateFailed, now, "image not found"),
	}
	result := ActiveDeploymentErrorsByService(tasks)
	if len(result) != 1 {
		t.Fatalf("expected 1 error, got %d", len(result))
	}
	if result["svc1"] != "image not found" {
		t.Errorf("got %q, want %q", result["svc1"], "image not found")
	}
}

func TestActiveDeploymentErrors_RecoveredService(t *testing.T) {
	now := time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC)
	tasks := []swarm.Task{
		// Old failed task
		makeTask("svc1", 1, swarm.TaskStateRunning, swarm.TaskStateFailed, now.Add(-time.Minute), "boom"),
		// New running task (newer than error)
		makeTask("svc1", 1, swarm.TaskStateRunning, swarm.TaskStateRunning, now, ""),
	}
	result := ActiveDeploymentErrorsByService(tasks)
	if len(result) != 0 {
		t.Errorf("expected no errors for recovered service, got %v", result)
	}
}

func TestActiveDeploymentErrors_StaleErrorIgnored(t *testing.T) {
	now := time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC)
	tasks := []swarm.Task{
		// Running task is newest
		makeTask("svc1", 1, swarm.TaskStateRunning, swarm.TaskStateRunning, now, ""),
		// Old error from > 5 minutes ago
		makeTask("svc1", 2, swarm.TaskStateRunning, swarm.TaskStateFailed, now.Add(-10*time.Minute), "old error"),
	}
	result := ActiveDeploymentErrorsByService(tasks)
	if len(result) != 0 {
		t.Errorf("expected stale error to be ignored, got %v", result)
	}
}

func TestActiveDeploymentErrors_EmptyErrMsg(t *testing.T) {
	now := time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC)
	tasks := []swarm.Task{
		makeTask("svc1", 1, swarm.TaskStateRunning, swarm.TaskStateFailed, now, ""),
	}
	result := ActiveDeploymentErrorsByService(tasks)
	if len(result) != 1 {
		t.Fatalf("expected 1 error, got %d", len(result))
	}
	if result["svc1"] == "" {
		t.Error("expected non-empty fallback error message for failed task with no Err text")
	}
}

func TestActiveDeploymentErrors_NoRunningTask(t *testing.T) {
	now := time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC)
	tasks := []swarm.Task{
		makeTask("svc1", 1, swarm.TaskStateRunning, swarm.TaskStateRejected, now, "no suitable node"),
	}
	result := ActiveDeploymentErrorsByService(tasks)
	if len(result) != 1 {
		t.Fatalf("expected 1 error, got %d", len(result))
	}
	if result["svc1"] != "no suitable node" {
		t.Errorf("got %q, want %q", result["svc1"], "no suitable node")
	}
}
