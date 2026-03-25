// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package view

import (
	"errors"
	"fmt"
)

// BEHelpDesc returns desc if the action is registered, or desc+" (BE)" if not.
func BEHelpDesc(actionName, desc string) string {
	if HasAction(actionName) {
		return desc
	}
	return desc + " (BE)"
}

// BEUnavailableFormat is the format string used by BEUnavailableErr.
// Contains one %s verb for the feature name. Override in init() to customise.
var BEUnavailableFormat = "%s is a Business Edition feature.\nFor more information, visit: https://swarmcli.io/be"

// BEUnavailableFormatFn, if set, returns the error message for a feature.
// Takes precedence over BEUnavailableFormat. Override in init() to customise.
var BEUnavailableFormatFn func(featureName string) string

// BEUnavailableErr returns an error indicating a feature requires Business Edition.
func BEUnavailableErr(featureName string) error {
	if BEUnavailableFormatFn != nil {
		return errors.New(BEUnavailableFormatFn(featureName))
	}
	return fmt.Errorf(BEUnavailableFormat, featureName)
}
