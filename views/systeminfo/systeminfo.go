// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package systeminfoview

import swarmlog "swarmcli/utils/log"

const ViewName = "systeminfo"

func l() *swarmlog.SwarmLogger {
	return swarmlog.L().With("view", "systeminfo")
}
