// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package contexts

import swarmlog "github.com/Eldara-Tech/swarmcli/utils/log"

const ViewName = "contexts"

func l() *swarmlog.SwarmLogger {
	return swarmlog.L().With("view", "contexts")
}
