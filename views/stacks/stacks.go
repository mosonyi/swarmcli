package stacksview

import swarmlog "swarmcli/utils/log"

const ViewName = "stacks"

func l() *swarmlog.SwarmLogger {
	return swarmlog.L().With("view", "stacks")
}
