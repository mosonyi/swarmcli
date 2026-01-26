// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package stacksview

import swarmlog "swarmcli/utils/log"

const ViewName = "stacks"

func l() *swarmlog.SwarmLogger {
	return swarmlog.L().With("view", "stacks")
}
