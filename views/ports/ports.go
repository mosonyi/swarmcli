// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package portsview

import (
	"time"

	swarmlog "swarmcli/utils/log"
)

const ViewName = "ports"

type TickMsg time.Time

const PollInterval = 3 * time.Second

func l() *swarmlog.SwarmLogger {
	return swarmlog.L().With("view", "ports")
}
