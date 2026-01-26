// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package logsview

import swarmlog "swarmcli/utils/log"

const ViewName = "logs"

func l() *swarmlog.SwarmLogger {
	return swarmlog.L().With("views", "logs")
}
