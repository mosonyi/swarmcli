// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package systeminfoview

import "time"

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
