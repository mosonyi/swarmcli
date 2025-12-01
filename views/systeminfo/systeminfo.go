package systeminfoview

import swarmlog "swarmcli/utils/log"

const ViewName = "systeminfo"

func l() *swarmlog.SwarmLogger {
	return swarmlog.L().With("view", "systeminfo")
}
