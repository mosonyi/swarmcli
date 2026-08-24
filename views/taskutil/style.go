// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package taskutil

import (
	"github.com/Eldara-Tech/swarmcli/v2/docker"

	"github.com/charmbracelet/lipgloss"
	"github.com/docker/docker/api/types/swarm"
)

var (
	taskFailedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	taskStoppedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
)

// TaskRowStyle tints one task row by what the row is telling the operator, and
// returns base for a task with nothing to say. Red is kept for a task that went
// wrong — it failed, was rejected, orphaned, recorded an error, or its
// healthcheck is failing. A task that is simply not running gets yellow: swarm
// stops a replica on purpose on every update, scale-down and node drain, and
// the container's docker-ps state is "exited" for that exactly as it is for a
// crash, so the swarm task state is what tells the two apart (issue #601).
//
// base is the caller's normal row style rather than a colour of ours: the three
// views showing tasks each render an ordinary row in their own colour, and only
// the two exceptional tints are shared.
func TaskRowStyle(t docker.TaskEntry, base lipgloss.Style) lipgloss.Style {
	switch {
	case isFailedState(t.State), t.Error != "",
		t.Health == "unhealthy", t.ContainerState == "dead":
		return taskFailedStyle
	case isStoppedState(t.State), t.Health == "starting",
		t.ContainerState == "restarting", t.ContainerState == "exited":
		return taskStoppedStyle
	default:
		return base
	}
}

// isFailedState reports whether a swarm task state means the task went wrong.
// Orphaned counts: the task's node has been unreachable long enough that swarm
// gave up on ever hearing how it ended.
func isFailedState(state string) bool {
	switch swarm.TaskState(state) {
	case swarm.TaskStateFailed, swarm.TaskStateRejected, swarm.TaskStateOrphaned:
		return true
	}
	return false
}

// isStoppedState reports whether a swarm task state means the task is over
// without having failed — stopped by swarm, or finished on its own.
func isStoppedState(state string) bool {
	switch swarm.TaskState(state) {
	case swarm.TaskStateShutdown, swarm.TaskStateComplete, swarm.TaskStateRemove:
		return true
	}
	return false
}
