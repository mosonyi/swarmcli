// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

// Package views triggers registration of all built-in views via blank imports.
package views

import (
	_ "github.com/Eldara-Tech/swarmcli/v2/views/charts"
	_ "github.com/Eldara-Tech/swarmcli/v2/views/configs"
	_ "github.com/Eldara-Tech/swarmcli/v2/views/contexts"
	_ "github.com/Eldara-Tech/swarmcli/v2/views/help"
	_ "github.com/Eldara-Tech/swarmcli/v2/views/inspect"
	_ "github.com/Eldara-Tech/swarmcli/v2/views/loading"
	_ "github.com/Eldara-Tech/swarmcli/v2/views/logs"
	_ "github.com/Eldara-Tech/swarmcli/v2/views/networks"
	_ "github.com/Eldara-Tech/swarmcli/v2/views/nodes"
	_ "github.com/Eldara-Tech/swarmcli/v2/views/secrets"
	_ "github.com/Eldara-Tech/swarmcli/v2/views/services"
	_ "github.com/Eldara-Tech/swarmcli/v2/views/stacks"
	_ "github.com/Eldara-Tech/swarmcli/v2/views/tasks"
	_ "github.com/Eldara-Tech/swarmcli/v2/views/volumes"
)
