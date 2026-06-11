// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package commands

import (
	_ "swarmcli/commands/command"
	_ "swarmcli/commands/command/docker"
	_ "swarmcli/commands/command/docker/config"
	_ "swarmcli/commands/command/docker/network"
	_ "swarmcli/commands/command/docker/node"
	_ "swarmcli/commands/command/docker/secret"
	_ "swarmcli/commands/command/docker/service"
	_ "swarmcli/commands/command/docker/volume"
)
