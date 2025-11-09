package configsview

import swarmlog "swarmcli/utils/log"

const ViewName = "configs"

func l() *swarmlog.SwarmLogger {
	return swarmlog.L().With("docker", "client")
}
