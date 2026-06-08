// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package volumesview

import (
	"fmt"
	"time"
)

type volumeItem struct {
	Name       string
	Stack      string
	Driver     string
	Mountpoint string
	Created    time.Time
	Host       string
	NodeID     string // swarm node ID; populated by the cross-node impl, used to address node-scoped actions
}

func (i volumeItem) FilterValue() string { return i.Name }
func (i volumeItem) Title() string       { return i.Name }
func (i volumeItem) Description() string {
	createdStr := "N/A"
	if !i.Created.IsZero() {
		createdStr = i.Created.Format("2006-01-02 15:04:05")
	}
	return fmt.Sprintf("Stack: %s        Driver: %s        Mount: %s        Created: %s        Host: %s",
		displayOrDash(i.Stack), i.Driver, i.Mountpoint, createdStr, displayOrDash(i.Host))
}

// displayOrDash renders an em-dash for empty optional fields.
func displayOrDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}
