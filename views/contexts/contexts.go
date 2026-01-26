// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package contexts

import swarmlog "swarmcli/utils/log"

const ViewName = "contexts"

func l() *swarmlog.SwarmLogger {
	return swarmlog.L().With("view", "contexts")
}
