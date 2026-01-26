// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package commands

import (
	"swarmcli/registry"
)

// Public passthroughs so app code can just use `commands.Get()` or `commands.All()`
func Register(cmd registry.Command)            { registry.Register(cmd) }
func Get(name string) (registry.Command, bool) { return registry.Get(name) }
func All() []registry.Command                  { return registry.All() }
func Suggest(prefix string) []string           { return registry.Suggest(prefix) }
