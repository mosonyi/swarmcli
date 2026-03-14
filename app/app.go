// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package app

import (
	"strings"

	swarmlog "swarmcli/utils/log"

	_ "swarmcli/commands" // triggers command autoload
	"swarmcli/registry"
	_ "swarmcli/views" // triggers view autoload
	"swarmcli/views/helpbar"
)

const (
	appName string = "swarmcli"
)

var (
	version string = "dev"
	edition string = "ce"
)

// SetVersion sets the application version (called from main)
func SetVersion(v string) {
	version = v
}

// SetEdition sets the application edition (called from main).
func SetEdition(e string) {
	if e == "" {
		edition = "ce"
	} else {
		edition = e
	}
	helpbar.EditionLabel = editionLabel(edition)
}

func editionLabel(e string) string {
	switch strings.ToLower(strings.TrimSpace(e)) {
	case "ce":
		return "Community Edition"
	case "be":
		return "Business Edition"
	default:
		if e == "" {
			return "Community Edition"
		}
		return strings.ToUpper(e) + " Edition"
	}
}

// Init should be called once at the start of the application to initialize logging.
func Init() {
	swarmlog.Init(appName)
	l := swarmlog.L()
	defer swarmlog.Sync()

	l.Infow("starting Swarm CLI", "version", version, "edition", edition)

	l.Infof("Available Commands:")
	for _, cmd := range registry.All() {
		l.Infoln("-", cmd.Name(), "→", cmd.Description())
	}
}
