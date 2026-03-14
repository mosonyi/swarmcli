// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package systeminfoview

import "time"

// SystemInfoMsg is implemented by all messages owned by the systeminfo component.
// The app-level router uses this to forward messages without listing each type.
type SystemInfoMsg interface {
	systemInfoMsg()
}

func (Msg) systemInfoMsg()              {}
func (SlowStatusMsg) systemInfoMsg()    {}
func (TickMsg) systemInfoMsg()          {}
func (SpinnerTickMsg) systemInfoMsg()   {}
func (LatestVersionMsg) systemInfoMsg() {}

type Msg struct {
	context     string
	cpu         string
	mem         string
	cpuCapacity string
	memCapacity string
	containers  int
	services    int
}

type SlowStatusMsg struct {
	cpu string
	mem string
}

type TickMsg time.Time

type SpinnerTickMsg time.Time

type LatestVersionMsg struct {
	latestVersion string
	message       string
}
