// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package servicesview

import (
	"github.com/Eldara-Tech/swarmcli/docker"
	"time"
)

type Msg struct {
	Scope      string
	Entries    []docker.ServiceEntry
	FilterType FilterType
	NodeID     string
	Hostname   string
	StackName  string
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

const userActionTimeout = 15 * time.Second

// RestartErrorMsg is sent when a service restart fails
type RestartErrorMsg struct {
	ServiceName string
	Error       error
}

// ScaleErrorMsg is sent when a service scale operation fails
type ScaleErrorMsg struct {
	ServiceName string
	Error       error
}

// RemoveErrorMsg is sent when a service remove operation fails
type RemoveErrorMsg struct {
	ServiceName string
	Error       error
}

// RollbackErrorMsg is sent when a service rollback operation fails
type RollbackErrorMsg struct {
	ServiceName string
	Error       error
}

// TasksLoadedMsg is sent when tasks for a service are loaded
type TasksLoadedMsg struct {
	ServiceID string
	Tasks     []docker.TaskEntry
}

// AllTasksLoadedMsg carries refreshed tasks for all expanded services at once.
type AllTasksLoadedMsg struct {
	Tasks map[string][]docker.TaskEntry
}
