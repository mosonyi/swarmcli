// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
)

// EventMsg is emitted when a relevant Docker event occurs.
type EventMsg struct {
	Type   string
	Action string
	Err    error
}

// WatchEventsCmd listens for Docker events (service/config/network/node changes)
// using the Docker SDK and returns a single EventMsg when an event is observed.
// The command should be re-issued after handling to continue watching.
func WatchEventsCmd() tea.Cmd {
	return func() tea.Msg {
		cli, err := GetClient()
		if err != nil {
			return EventMsg{Err: err}
		}
		defer closeCli(cli)

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		f := filters.NewArgs()
		f.Add("type", "service")
		f.Add("type", "config")
		f.Add("type", "network")
		f.Add("type", "node")

		opts := events.ListOptions{Filters: f}
		msgs, errs := cli.Events(ctx, opts)

		for {
			select {
			case ev := <-msgs:
				return EventMsg{Type: string(ev.Type), Action: string(ev.Action)}
			case e := <-errs:
				if e != nil {
					return EventMsg{Err: e}
				}
			case <-ctx.Done():
				return EventMsg{Type: "timeout", Action: "timeout"}
			}
		}
	}
}
