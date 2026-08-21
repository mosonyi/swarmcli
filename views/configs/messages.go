// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package configsview

import (
	"github.com/Eldara-Tech/swarmcli/v2/docker"
	"time"

	"github.com/docker/docker/api/types/swarm"
)

// Messages for async ops
type (
	configsLoadedMsg []docker.ConfigWithDecodedData
	configRotatedMsg struct {
		Old docker.ConfigWithDecodedData
		New docker.ConfigWithDecodedData
	}
	configDeletedMsg struct {
		Name  string
		Index int
	}
	editConfigMsg struct {
		Name string
	}
	createConfigMsg struct {
		Name string
	}
	editConfigDoneMsg struct {
		Name      string
		Changed   bool
		OldConfig docker.ConfigWithDecodedData
		NewConfig docker.ConfigWithDecodedData
	}
	configCreatedMsg struct {
		Config swarm.Config
	}
	configCreateErrorMsg struct {
		err error
	}
	editorContentMsg struct {
		Content string
	}
	editorContentReadyMsg struct {
		Name string
		Data []byte
		Err  error
	}
	fileContentReadyMsg struct {
		Name     string
		FilePath string
		Data     []byte
		Err      error
	}
	editConfigErrorMsg struct {
		err error
	}
	filesLoadedMsg struct {
		Path  string
		Files []string
		Error error
	}
	usedByMsg struct {
		ConfigName string
		UsedBy     []usedByItem
		Error      error
	}
	errorMsg error
)

type TickMsg struct{ Gen uint64 }

// PollRetryMsg signals that polling found no changes; the Update handler
// should schedule the next tick.
type PollRetryMsg struct{}

// PollInterval is how often the view re-reads its resource. It is a var, not a
// const, so tests can shrink it: a tea.Tick cmd invoked synchronously blocks
// for the full interval, so a test that runs one to see what it scheduled would
// otherwise sit here for five seconds.
var PollInterval = 5 * time.Second

const pollTimeout = 4 * time.Second
const userActionTimeout = 15 * time.Second

type SpinnerTickMsg time.Time
