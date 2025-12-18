package contexts

import swarmlog "swarmcli/utils/log"

const ViewName = "contexts"

func l() *swarmlog.SwarmLogger {
	return swarmlog.L().With("view", "contexts")
}
