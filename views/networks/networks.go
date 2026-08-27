// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package networksview

import swarmlog "github.com/Eldara-Tech/swarmcli/v2/utils/log"

const ViewName = "networks"

func l() *swarmlog.SwarmLogger {
	return swarmlog.L().With("views", "networks")
}
