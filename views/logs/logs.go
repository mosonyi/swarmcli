package logsview

import swarmlog "swarmcli/utils/log"

const ViewName = "logs"

func l() *swarmlog.SwarmLogger {
	return swarmlog.L().With("views", "logs")
}
