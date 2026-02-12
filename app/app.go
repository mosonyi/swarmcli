// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package app

import (
	swarmlog "swarmcli/utils/log"

	_ "swarmcli/commands" // triggers command autoload
	"swarmcli/registry"
	_ "swarmcli/views" // triggers view autoload
)

const (
	appName string = "swarmcli"
)

var (
	version string = "dev"
)

// SetVersion sets the application version (called from main)
func SetVersion(v string) {
	version = v
}

// Init should be called once at the start of the application to initialize logging.
func Init() {
	swarmlog.Init(appName)
	l := swarmlog.L()
	defer swarmlog.Sync()

	l.Infow("starting Swarm CLI", "version", version)

	l.Infof("Available Commands:")
	for _, cmd := range registry.All() {
		l.Infoln("-", cmd.Name(), "→", cmd.Description())
	}
}
