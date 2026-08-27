// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package commands

import (
	"github.com/Eldara-Tech/swarmcli/v2/registry"
)

// Public passthroughs so app code can just use `commands.Get()`
func Get(name string) (registry.Command, bool) { return registry.Get(name) }
func Suggest(prefix string) []string           { return registry.Suggest(prefix) }
