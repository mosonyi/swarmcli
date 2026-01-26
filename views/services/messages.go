// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package servicesview

import (
	"swarmcli/docker"
	"time"
)

type Msg struct {
	Title      string
	Entries    []docker.ServiceEntry
	FilterType FilterType
	NodeID     string
	Hostname   string
	StackName  string
}

type TickMsg time.Time

const PollInterval = 2 * time.Second

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
