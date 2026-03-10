// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

// EventOps abstracts Docker event watching for testability and extensibility.
type EventOps interface {
	WatchEvent() Event
}

type defaultEventOps struct{}

func (defaultEventOps) WatchEvent() Event { return WatchEvent() }
