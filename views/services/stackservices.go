package servicesview

import swarmlog "swarmcli/utils/log"

const ViewName = "services"

func l() *swarmlog.SwarmLogger {
	return swarmlog.L().With("views", "services")
}
