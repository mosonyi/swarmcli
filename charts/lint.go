// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package charts

import "fmt"

// LintSeverity ranks a lint finding. Only LintError fails a lint.
type LintSeverity int

const (
	// LintWarning is advice: the chart works, but something is worth fixing.
	LintWarning LintSeverity = iota
	// LintError means the chart is broken — either outright, or for the engine
	// version it was linted against.
	LintError
)

func (s LintSeverity) String() string {
	if s == LintError {
		return "error"
	}
	return "warning"
}

// LintFinding is one thing lint noticed about a chart.
type LintFinding struct {
	Severity LintSeverity
	Message  string
}

// lintRelease is the placeholder release name a lint render uses. Templates that
// interpolate .Release.Name still need one; nothing is deployed.
const lintRelease = "lint"

// Lint checks a loaded chart against the chart engine named by engine — this
// build's, or one the caller is asking about via --for-version.
//
// It reports everything it finds instead of stopping at the first problem: a
// chart author wants the list, not a game of whack-a-mole.
//
// Structural validation already happened in LoadChartDir / LoadChartArchive,
// which refuse a chart with no name, version or templates, or an apiVersion this
// build cannot read. Lint covers what only becomes visible once a chart is
// rendered with its own defaults.
//
// One thing it deliberately cannot do: prove a chart runs on the version it
// declares. This binary carries one engine's behaviour and cannot emulate
// another's — rendering with a real binary of that version is the only thing
// that settles it. See CheckCompatAgainst.
func Lint(ch *Chart, engine string) []LintFinding {
	var out []LintFinding
	add := func(s LintSeverity, format string, a ...any) {
		out = append(out, LintFinding{Severity: s, Message: fmt.Sprintf(format, a...)})
	}

	switch f := CheckCompatAgainst(ch.Metadata, engine); f.Status {
	case CompatIncompatible:
		// Deliberately not CompatFinding.Message: that says "this build
		// provides X", which is a lie under --for-version, where X is a version
		// the caller is asking about and not the one running.
		add(LintError, "Chart.yaml requires swarmcli %s, which %s does not satisfy", f.Required, f.Engine)
	case CompatUnknown:
		if f.Reason != "" {
			add(LintWarning, "%s", f.Reason)
			break
		}
		// Not an error: the field is optional and most charts predate it. But a
		// chart that names no floor gives an operator on an old build nothing to
		// act on, which is the whole problem swarmcliVersion exists to fix.
		add(LintWarning, "Chart.yaml declares no swarmcliVersion: nothing states which swarmcli this chart needs")
	}

	values, err := MergeValues(ch.Values, nil, nil)
	if err != nil {
		add(LintError, "values.yaml: %v", err)
		return out // everything below renders from these values
	}
	if err := ValidateValues(ch.Schema, values); err != nil {
		add(LintError, "values.yaml does not satisfy values.schema.json: %v", err)
	}

	ctx := RenderContext{
		Values:  values,
		Release: ReleaseMeta{Name: lintRelease, Namespace: lintRelease, Revision: 1},
		Chart:   ChartMeta{Name: ch.Metadata.Name, Version: ch.Metadata.Version, AppVersion: ch.Metadata.AppVersion},
	}
	if _, err := Render(ch, ctx); err != nil {
		add(LintError, "render with default values failed: %v", err)
		return out
	}
	if _, err := RenderRequirements(ch, ctx); err != nil {
		add(LintError, "requirements.yaml: %v", err)
	}
	return out
}

// HasErrors reports whether any finding is fatal, i.e. whether the lint failed.
func HasErrors(findings []LintFinding) bool {
	for _, f := range findings {
		if f.Severity == LintError {
			return true
		}
	}
	return false
}
