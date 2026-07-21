// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package charts

import (
	"fmt"
	"sort"
	"time"

	"gopkg.in/yaml.v3"
)

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
// files and sets are extra values layered over the chart defaults for the render
// check, exactly as `charts template -f/--set` would: a chart that requires an
// input it deliberately leaves undefaulted (a {{ required }} / {{ fail }} guard)
// cannot render from bare defaults, so linting it needs the same values a real
// install would supply. Pass nil for both to lint against defaults alone.
//
// It reports everything it finds instead of stopping at the first problem: a
// chart author wants the list, not a game of whack-a-mole.
//
// Structural validation already happened in LoadChartDir / LoadChartArchive,
// which refuse a chart with no name, version or templates, or an apiVersion this
// build cannot read. Lint covers what only becomes visible once a chart is
// rendered.
//
// One thing it deliberately cannot do: prove a chart runs on the version it
// declares. This binary carries one engine's behaviour and cannot emulate
// another's — rendering with a real binary of that version is the only thing
// that settles it. See CheckCompatAgainst.
func Lint(ch *Chart, engine string, files [][]byte, sets []string) []LintFinding {
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

	values, err := MergeValues(ch.Values, files, sets)
	if err != nil {
		add(LintError, "%v", err)
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
	manifest, err := Render(ch, ctx)
	if err != nil {
		add(LintError, "render with default values failed: %v", err)
		return out
	}
	lintHealthcheckMonitor(manifest, add)
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

// composeHealthLint is the sliver of the rendered stack that lintHealthcheckMonitor
// reads. Durations stay strings because compose writes them as "30s"/"1m30s".
type composeHealthLint struct {
	Services map[string]struct {
		Healthcheck *struct {
			Test        any    `yaml:"test"`
			Interval    string `yaml:"interval"`
			Retries     *int   `yaml:"retries"`
			StartPeriod string `yaml:"start_period"`
			Disable     bool   `yaml:"disable"`
		} `yaml:"healthcheck"`
		Deploy struct {
			UpdateConfig *struct {
				Monitor string `yaml:"monitor"`
			} `yaml:"update_config"`
		} `yaml:"deploy"`
	} `yaml:"services"`
}

// Docker's own defaults for an unspecified healthcheck field.
const (
	defaultHealthInterval = 30 * time.Second
	defaultHealthRetries  = 3
)

// swarmDefaultMonitor is what UpdateConfig.Monitor is when a compose file sets
// none. Same fact as defaultStabilityWindow in release.go, which is the window
// --wait has to sit through; naming it separately here keeps the lint readable
// without a second copy of the number.
const swarmDefaultMonitor = defaultStabilityWindow

// lintHealthcheckMonitor warns when a service cannot fail its healthcheck before
// swarm stops watching.
//
// Swarm counts a task failure against a rollout only if it happens within
// UpdateConfig.Monitor of the task being created. A container that goes
// unhealthy after that window does NOT trigger update_config.failure_action —
// the rollout is reported completed and the task quietly restart-loops. So if
// monitor is shorter than the healthcheck's own worst case, a broken deploy can
// report success. Kubernetes has no analogue of this, so it reliably surprises
// people arriving from there.
//
// An unset monitor is the common way to hit this, not an exemption from it:
// swarm's default is 5s, shorter than almost any healthcheck's worst case. The
// rule originally skipped that case, which meant it fired on nothing at all.
//
// The figure is a floor, not a budget. Swarm starts the window when it CREATES
// the task, so pulling the image happens inside it too — a large first pull on
// a cold node can consume the whole window before the container even starts.
// Pull time cannot be known from the manifest (it depends on image size,
// registry and how warm the node is), so it is named in the finding rather than
// guessed at with a constant: a fabricated allowance would be wrong for every
// chart in a different direction, and would re-fire the rule on charts whose
// monitors were set from this very formula.
//
// Warning, not error: a short monitor is legitimate when the operator wants a
// fast rollout and is watching by other means.
func lintHealthcheckMonitor(manifest string, add func(LintSeverity, string, ...any)) {
	var doc composeHealthLint
	if err := yaml.Unmarshal([]byte(manifest), &doc); err != nil {
		return // the manifest already rendered; shape problems are not lint's business here
	}

	// Map iteration is random; sort so findings do not shuffle between runs.
	names := make([]string, 0, len(doc.Services))
	for name := range doc.Services {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		svc := doc.Services[name]
		hc := svc.Healthcheck
		if hc == nil || hc.Disable || isHealthcheckNone(hc.Test) {
			continue
		}
		monitor, declared := swarmDefaultMonitor, false
		if svc.Deploy.UpdateConfig != nil && svc.Deploy.UpdateConfig.Monitor != "" {
			d, err := time.ParseDuration(svc.Deploy.UpdateConfig.Monitor)
			if err != nil {
				continue
			}
			monitor, declared = d, true
		}

		interval := defaultHealthInterval
		if hc.Interval != "" {
			if d, err := time.ParseDuration(hc.Interval); err == nil {
				interval = d
			}
		}
		retries := defaultHealthRetries
		if hc.Retries != nil {
			retries = *hc.Retries
		}
		var startPeriod time.Duration
		if hc.StartPeriod != "" {
			if d, err := time.ParseDuration(hc.StartPeriod); err == nil {
				startPeriod = d
			}
		}
		if retries < 0 {
			continue
		}

		needed := startPeriod + time.Duration(retries)*interval
		if monitor >= needed {
			continue
		}
		had := fmt.Sprintf("deploy.update_config.monitor (%s)", monitor)
		if !declared {
			had = fmt.Sprintf("no deploy.update_config.monitor, so swarm's %s default applies, which", monitor)
		}
		add(LintWarning,
			"service %q: %s is shorter than the healthcheck needs to fail (%s = start_period %s + interval %s x retries %d); a container that goes unhealthy after the monitor window does not fail the rollout, so a broken deploy reports success. Set deploy.update_config.monitor to at least %s — and higher if the image is large, since swarm starts the window when it creates the task, so the pull runs inside it and %s does not account for it",
			name, had, needed, startPeriod, interval, retries, needed, needed)
	}
}

// isHealthcheckNone reports the compose "disable the image healthcheck" spelling,
// test: ["NONE"] or test: NONE.
func isHealthcheckNone(test any) bool {
	switch v := test.(type) {
	case string:
		return v == "NONE"
	case []any:
		return len(v) > 0 && v[0] == "NONE"
	}
	return false
}
