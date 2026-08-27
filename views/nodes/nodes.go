// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package nodesview

import swarmlog "github.com/Eldara-Tech/swarmcli/v2/utils/log"

const ViewName = "nodes"

func l() *swarmlog.SwarmLogger {
	return swarmlog.L().With("view", "nodes")
}
