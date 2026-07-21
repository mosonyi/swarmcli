// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package app

import (
	"strings"

	swarmlog "github.com/Eldara-Tech/swarmcli/utils/log"

	_ "github.com/Eldara-Tech/swarmcli/commands" // triggers command autoload
	"github.com/Eldara-Tech/swarmcli/registry"
	_ "github.com/Eldara-Tech/swarmcli/views" // triggers view autoload
	"github.com/Eldara-Tech/swarmcli/views/helpbar"
)

const (
	appName string = "swarmcli"
)

var (
	version string = "dev"
	edition string = "ce"
)

func normalizeEdition(e string) string {
	normalized := strings.ToLower(strings.TrimSpace(e))
	if normalized == "" {
		return "ce"
	}

	return normalized
}

// SetVersion sets the application version (called from main)
func SetVersion(v string) {
	version = v
}

// SetEdition sets the application edition (called from main).
func SetEdition(e string) {
	edition = normalizeEdition(e)
	helpbar.SetEditionLabel(editionLabel(edition))
}

func editionLabel(e string) string {
	switch e {
	case "ce":
		return "Community Edition"
	case "be":
		return "Business Edition"
	default:
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
