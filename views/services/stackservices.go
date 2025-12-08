package servicesview

import swarmlog "swarmcli/utils/log"

func l() *swarmlog.SwarmLogger {
	return swarmlog.L().With("views", "services")
}
