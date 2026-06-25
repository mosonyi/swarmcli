// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package app

import (
	"fmt"
	"strings"
)

const (
	installDocsURLCommunity = "https://swarmcli.io/docs/cli#installation"
	installDocsURLBusiness  = "https://swarmcli.io/docs/cli#installation-business"
	updateNoticeCheckbox    = "Do not show this again for this version"
)

// showUpdateNotice raises the app-level "update available" dialog for the given
// latest release. Edition selects the install link and whether the CE→BE hint
// is shown. The dialog is an info-mode confirmdialog carrying an opt-out
// checkbox; ResultMsg handling persists the dismissal when it is ticked.
func (m *Model) showUpdateNotice(latest string) {
	m.updateDialog.Visible = true
	m.updateDialog.ErrorMode = false
	m.updateDialog.InfoMode = true
	m.updateDialog.CheckboxLabel = updateNoticeCheckbox
	m.updateDialog.CheckboxChecked = false
	m.updateDialog.Message = updateNoticeMessage(latest)
	m.updateDialogActive = true
	m.pendingUpdateVersion = latest
}

// updateNoticeMessage builds the notice body. BE shows only the BE install
// link; CE shows the CE link plus a subtle one-line BE upsell.
func updateNoticeMessage(latest string) string {
	current := strings.TrimSpace(version)
	if current == "" {
		current = "unknown"
	}
	if edition == "be" {
		return fmt.Sprintf(
			"A new version of SwarmCLI Business Edition is available: %s (you have %s).\n\n"+
				"Update: %s",
			latest, current, installDocsURLBusiness)
	}
	return fmt.Sprintf(
		"A new version of SwarmCLI is available: %s (you have %s).\n\n"+
			"Update: %s\n\n"+
			"Need RBAC, SSO & port-forwarding? Try Business Edition →\n%s",
		latest, current, installDocsURLCommunity, installDocsURLBusiness)
}
