// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package portsview

import (
	"swarmcli/docker"
	"time"

	swarmlog "swarmcli/utils/log"
)

const ViewName = "ports"

// PollInterval is how often the tick fires to re-render diagnostics.
const PollInterval = 3 * time.Second

// ProbeInterval is how often we launch a fresh round of port probes.
// Probes involve real network I/O, so we space them out more than renders.
const ProbeInterval = 10 * time.Second

type TickMsg time.Time

// ProbeResultMsg carries finished probe results back to the Update loop.
type ProbeResultMsg struct {
	Results []docker.NodeProbeResult
}

func l() *swarmlog.SwarmLogger {
	return swarmlog.L().With("view", "ports")
}
