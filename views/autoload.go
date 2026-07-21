// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

// Package views triggers registration of all built-in views via blank imports.
package views

import (
	_ "github.com/Eldara-Tech/swarmcli/views/configs"
	_ "github.com/Eldara-Tech/swarmcli/views/contexts"
	_ "github.com/Eldara-Tech/swarmcli/views/help"
	_ "github.com/Eldara-Tech/swarmcli/views/inspect"
	_ "github.com/Eldara-Tech/swarmcli/views/loading"
	_ "github.com/Eldara-Tech/swarmcli/views/logs"
	_ "github.com/Eldara-Tech/swarmcli/views/networks"
	_ "github.com/Eldara-Tech/swarmcli/views/nodes"
	_ "github.com/Eldara-Tech/swarmcli/views/secrets"
	_ "github.com/Eldara-Tech/swarmcli/views/services"
	_ "github.com/Eldara-Tech/swarmcli/views/stacks"
	_ "github.com/Eldara-Tech/swarmcli/views/tasks"
	_ "github.com/Eldara-Tech/swarmcli/views/volumes"
)
