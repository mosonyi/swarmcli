// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package chartsview

import (
	"fmt"
	"strconv"
)

// blockedAction is a release operation this view will not perform, and the
// command line that does.
//
// The keys are bound rather than left dead on purpose: `ctrl+d` deletes in the
// configs, secrets and volumes views, so an operator arriving here will press
// it. Answering with the exact command — release name already substituted —
// turns that keystroke into the answer instead of into nothing happening.
type blockedAction struct {
	title   string
	command string
	// why is the one-line reason this cannot happen here, when there is more
	// to say than "the CLI does it".
	why string
}

// upgradeAction, rollbackAction and uninstallAction describe the three
// mutations an operator is most likely to reach for from a release list.
func upgradeAction(release string) blockedAction {
	return blockedAction{
		title:   "Upgrade a release",
		command: fmt.Sprintf("swarmcli charts upgrade %s <repo/chart>", release),
		why:     "An upgrade renders a chart with your values, so it needs the chart reference this view does not have.",
	}
}

func rollbackAction(release string, revision int) blockedAction {
	target := revision - 1
	if target < 1 {
		target = 1
	}
	return blockedAction{
		title:   "Roll a release back",
		command: fmt.Sprintf("swarmcli charts rollback %s %s", release, strconv.Itoa(target)),
		why:     "Press enter on the release to see its revisions, and `d` to see what each one changed.",
	}
}

func uninstallAction(release string) blockedAction {
	return blockedAction{
		title:   "Uninstall a release",
		command: fmt.Sprintf("swarmcli charts uninstall %s", release),
		why:     "Removing the stack by hand would leave the release's revision history behind and corrupt it.",
	}
}

// message is what the dismiss-only dialog shows.
func (a blockedAction) message() string {
	msg := fmt.Sprintf("%s\n\nThis view is read-only. Run:\n\n    %s", a.title, a.command)
	if a.why != "" {
		msg += "\n\n" + a.why
	}
	return msg
}

// showBlocked opens the dismiss-only dialog for a blocked action.
func (m *Model) showBlocked(a blockedAction) {
	m.confirmDialog.Message = a.message()
	m.confirmDialog.InfoMode = true
	m.confirmDialog.Visible = true
}
