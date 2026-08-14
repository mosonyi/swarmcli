// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package tasksview

import (
	"github.com/Eldara-Tech/swarmcli/core/primitives/hash"
	"github.com/Eldara-Tech/swarmcli/docker"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type TasksLoadedMsg struct {
	Tasks []docker.TaskEntry
	Error error
}

type TickMsg time.Time

// PollRetryMsg signals that polling found no changes; the Update handler
// should schedule the next tick.
type PollRetryMsg struct{}

// PollInterval is how often the view re-reads its resource. It is a var, not a
// const, so tests can shrink it: a tea.Tick cmd invoked synchronously blocks
// for the full interval, so a test that runs one to see what it scheduled would
// otherwise sit here for the whole period.
var PollInterval = 2 * time.Second

func LoadTasksCmd(stackName string) tea.Cmd {
	return func() tea.Msg {
		tasks, err := docker.GetTasksForStack(stackName)
		return TasksLoadedMsg{
			Tasks: tasks,
			Error: err,
		}
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(PollInterval, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// CheckTasksCmd checks if tasks have changed and returns update message if so
func CheckTasksCmd(lastHash uint64, stackName string) tea.Cmd {
	return func() tea.Msg {
		l().Info("CheckTasksCmd: Polling for task changes")

		// Trigger async refresh if needed and use cached snapshot for quick checks
		docker.TriggerRefreshIfNeeded()

		tasks, err := docker.GetTasksForStack(stackName)
		if err != nil {
			l().Errorf("CheckTasksCmd: Failed to get tasks: %v", err)
			return PollRetryMsg{}
		}

		newHash, err := hash.Compute(tasks)
		if err != nil {
			l().Errorf("CheckTasksCmd: Hash computation failed: %v", err)
			return PollRetryMsg{}
		}

		l().Infof("CheckTasksCmd: lastHash=%s, newHash=%s, taskCount=%d",
			hash.Fmt(lastHash), hash.Fmt(newHash), len(tasks))

		// Only return update message if something changed
		if newHash != lastHash {
			l().Info("CheckTasksCmd: Change detected! Refreshing task list")
			return TasksLoadedMsg{
				Tasks: tasks,
				Error: nil,
			}
		}

		l().Info("CheckTasksCmd: No changes detected, scheduling next poll")
		return PollRetryMsg{}
	}
}
