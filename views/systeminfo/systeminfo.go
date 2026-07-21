// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package systeminfoview

import swarmlog "github.com/Eldara-Tech/swarmcli/utils/log"

const ViewName = "systeminfo"

func l() *swarmlog.SwarmLogger {
	return swarmlog.L().With("view", "systeminfo")
}
