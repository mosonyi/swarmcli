package nodesview

import swarmlog "swarmcli/utils/log"

const ViewName = "nodes"

func l() *swarmlog.SwarmLogger {
	return swarmlog.L().With("view", "nodes")
}
