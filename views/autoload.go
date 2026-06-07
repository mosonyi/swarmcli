// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

// Package views triggers registration of all built-in views via blank imports.
package views

import (
	_ "swarmcli/views/configs"
	_ "swarmcli/views/contexts"
	_ "swarmcli/views/help"
	_ "swarmcli/views/inspect"
	_ "swarmcli/views/loading"
	_ "swarmcli/views/logs"
	_ "swarmcli/views/networks"
	_ "swarmcli/views/nodes"
	_ "swarmcli/views/ports"
	_ "swarmcli/views/secrets"
	_ "swarmcli/views/services"
	_ "swarmcli/views/stacks"
	_ "swarmcli/views/tasks"
	_ "swarmcli/views/volumes"
)
