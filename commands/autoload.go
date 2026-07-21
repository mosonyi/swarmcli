// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package commands

import (
	_ "github.com/Eldara-Tech/swarmcli/commands/command"
	_ "github.com/Eldara-Tech/swarmcli/commands/command/docker"
	_ "github.com/Eldara-Tech/swarmcli/commands/command/docker/config"
	_ "github.com/Eldara-Tech/swarmcli/commands/command/docker/network"
	_ "github.com/Eldara-Tech/swarmcli/commands/command/docker/node"
	_ "github.com/Eldara-Tech/swarmcli/commands/command/docker/secret"
	_ "github.com/Eldara-Tech/swarmcli/commands/command/docker/service"
	_ "github.com/Eldara-Tech/swarmcli/commands/command/docker/volume"
)
