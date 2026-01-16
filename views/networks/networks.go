package networksview

import swarmlog "swarmcli/utils/log"

const ViewName = "networks"

func l() *swarmlog.SwarmLogger {
	return swarmlog.L().With("views", "networks")
}
