package secretsview

import swarmlog "swarmcli/utils/log"

const ViewName = "secrets"

func l() *swarmlog.SwarmLogger {
	return swarmlog.L().With("views", "secrets")
}
