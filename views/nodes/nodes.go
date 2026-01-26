// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package nodesview

import swarmlog "swarmcli/utils/log"

const ViewName = "nodes"

func l() *swarmlog.SwarmLogger {
	return swarmlog.L().With("view", "nodes")
}
