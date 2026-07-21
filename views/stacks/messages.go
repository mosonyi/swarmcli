// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package stacksview

import (
	"github.com/Eldara-Tech/swarmcli/docker"
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

// PollRetryMsg signals that polling found no changes; the Update handler
// should schedule the next tick.
type PollRetryMsg struct{}

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

// StackDeleteIntentMsg carries the result of checking, when a stack delete is
// requested, whether that stack belongs to a chart release. ChartRelease is
// non-empty when it does, so the confirm dialog can warn before removal.
type StackDeleteIntentMsg struct {
	StackName    string
	ChartRelease string
}

// editorContentMsg is sent when editor returns content
type editorContentMsg struct {
	Content         string
	OriginalContent string // populated in edit mode to detect no-change
}

// stackCreateErrorMsg is sent when stack creation has an error
type stackCreateErrorMsg struct {
	Err error
}

// stackUpdateErrorMsg is sent when stack update (edit) has an error
type stackUpdateErrorMsg struct {
	StackName string
	Err       error
}

// filesLoadedMsg is sent when files are loaded from a directory
type filesLoadedMsg struct {
	Path  string
	Files []string
	Error error
}

// stackSavedMsg is sent when stack YAML is successfully saved to file
type stackSavedMsg struct {
	Path string
}

// stackSaveErrorMsg is sent when saving stack YAML fails
type stackSaveErrorMsg struct {
	Err error
}
