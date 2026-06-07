// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package volumesview

import swarmlog "swarmcli/utils/log"

const ViewName = "volumes"

// allNodesFeature gates the "list volumes across all nodes" capability. The
// base build lists the connected node only; an extension build enables this
// flag when it provides a multi-node volume implementation, which suppresses
// the connected-node-only footer hint.
const allNodesFeature = "volumes-all-nodes"

func l() *swarmlog.SwarmLogger {
	return swarmlog.L().With("views", "volumes")
}
