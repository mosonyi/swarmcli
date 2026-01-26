// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package servicesview

import swarmlog "swarmcli/utils/log"

func l() *swarmlog.SwarmLogger {
	return swarmlog.L().With("views", "services")
}
