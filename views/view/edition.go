// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package view

import "fmt"

// BEHelpDesc returns desc if the action is registered, or desc+" (BE)" if not.
func BEHelpDesc(actionName, desc string) string {
	if HasAction(actionName) {
		return desc
	}
	return desc + " (BE)"
}

// BEUnavailableErr returns an error indicating a feature requires Business Edition.
func BEUnavailableErr(featureName string) error {
	return fmt.Errorf("%s is a Business Edition feature.\nFor more information, visit: https://swarmcli.io/be", featureName)
}
