package tasksview

import (
	swarmlog "swarmcli/utils/log"
)

const ViewName = "tasks"

func l() *swarmlog.SwarmLogger {
	return swarmlog.L()
}
