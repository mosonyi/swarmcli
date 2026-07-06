// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package view

import (
	"errors"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// BEHelpDesc returns desc if the action is registered, or desc+" (BE)" if not.
func BEHelpDesc(actionName, desc string) string {
	if HasAction(actionName) {
		return desc
	}
	return desc + " (BE)"
}

// BELandingURL is the public Business Edition landing page (trial info,
// feature overview). Exported so the BE module references the same
// string instead of re-hardcoding it.
const BELandingURL = "https://swarmcli.io/be"

// BEUnavailableFormat is the format string used by BEUnavailableErr.
// Contains one %s verb for the feature name. Override in init() to customise.
var BEUnavailableFormat = "%s is a Business Edition feature.\nFor more information, visit: " + BELandingURL

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

// FeatureLockedCmdFn, if set by an extension, returns a tea.Cmd to run when a
// license-gated feature is requested without entitlement. When set it takes
// precedence over the caller's local error dialog, letting the extension show
// a richer prompt (e.g. an inline license dialog) instead of the flat
// BEUnavailableErr text. Override in init().
var FeatureLockedCmdFn func(featureName string) tea.Cmd

// FeatureLockedCmd returns the extension's rich locked-feature command for
// featureName, or nil when no extension is registered — in which case callers
// fall back to surfacing BEUnavailableErr in their own error UI.
func FeatureLockedCmd(featureName string) tea.Cmd {
	if FeatureLockedCmdFn != nil {
		return FeatureLockedCmdFn(featureName)
	}
	return nil
}

// ServicesHealthHint, when set by the BE module, returns a one-line services-
// view footer note (e.g. why container health is unavailable on the current
// context), or "" when there is nothing to say. It lets BE surface a context-
// specific note without CE having to know about managed contexts. Set in
// init(); read on the render goroutine.
var ServicesHealthHint func() string
