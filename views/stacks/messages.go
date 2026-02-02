// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package stacksview

import (
	"swarmcli/docker"
	"time"
)

type Msg struct {
	NodeID string
	Stacks []docker.StackEntry
	Err    error
}

type RefreshErrorMsg struct {
	Err error
}

type TickMsg time.Time

const PollInterval = 5 * time.Second

// StackTasksLoadedMsg is sent when tasks for a stack are loaded
type StackTasksLoadedMsg struct {
	StackName string
	Tasks     []docker.TaskEntry
	Error     error
}

// RemoveErrorMsg is sent when stack removal fails
type RemoveErrorMsg struct {
	StackName string
	Error     error
}
