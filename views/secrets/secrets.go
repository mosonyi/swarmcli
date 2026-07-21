// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package secretsview

import swarmlog "github.com/Eldara-Tech/swarmcli/utils/log"

const ViewName = "secrets"

func l() *swarmlog.SwarmLogger {
	return swarmlog.L().With("views", "secrets")
}
