// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

import tea "github.com/charmbracelet/bubbletea"

// EventOps abstracts Docker event watching for testability and extensibility.
type EventOps interface {
	WatchEventsCmd() tea.Cmd
}

type defaultEventOps struct{}

func (defaultEventOps) WatchEventsCmd() tea.Cmd { return WatchEventsCmd() }
