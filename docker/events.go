// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

import (
	"context"
	"time"

	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
)

// Event represents a Docker event (or an error/timeout while watching).
type Event struct {
	Type   string
	Action string
	Err    error
}

// WatchEvent listens for Docker events (service/config/network/node changes)
// using the Docker SDK and returns a single Event when one is observed.
// This is a blocking call; callers should wrap it in a goroutine or tea.Cmd.
func WatchEvent() Event {
	cli, err := GetClient()
	if err != nil {
		return Event{Err: err}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	f := filters.NewArgs()
	f.Add("type", "service")
	f.Add("type", "config")
	f.Add("type", "network")
	f.Add("type", "node")

	opts := events.ListOptions{Filters: f}
	// The Docker SDK cleans up the event stream goroutine when ctx is cancelled.
	msgs, errs := cli.Events(ctx, opts)

	for {
		select {
		case ev := <-msgs:
			return Event{Type: string(ev.Type), Action: string(ev.Action)}
		case e := <-errs:
			if e != nil {
				return Event{Err: e}
			}
		case <-ctx.Done():
			return Event{Type: "timeout", Action: "timeout"}
		}
	}
}
