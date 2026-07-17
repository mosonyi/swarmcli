// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package charts

import (
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
)

// maxConstraintLen bounds swarmcliVersion before it reaches the constraint
// parser, which is regexp-backed and fed from chart metadata that may have come
// off the network. A real constraint is a handful of characters.
const maxConstraintLen = 256

// CompatStatus classifies a chart's declared engine requirement against this build.
type CompatStatus int

const (
	// CompatUnknown means the chart declared no requirement, this build reports
	// no engine version, or the declared constraint could not be parsed.
	// Callers must not block on it: it is not evidence of an incompatible chart.
	CompatUnknown CompatStatus = iota
	// CompatOK means this build's chart engine satisfies the constraint.
	CompatOK
	// CompatIncompatible means it does not. This is the only status callers
	// block on.
	CompatIncompatible
)

// CompatFinding is the result of checking one chart against this build.
type CompatFinding struct {
	Chart    string // "<name> <version>", for messages
	Required string // the chart's constraint as declared, e.g. ">= 1.13.0"
	Engine   string // this build's chart-engine version; "" when unstamped
	Status   CompatStatus
	Reason   string // why the check was skipped; set only with CompatUnknown
}

// CheckCompat classifies a chart's swarmcliVersion constraint against the chart
// engine this binary embeds.
func CheckCompat(cf Chartfile) CompatFinding { return CheckCompatAgainst(cf, engineVersion) }

// CheckCompatAgainst classifies a chart's swarmcliVersion against an arbitrary
// chart-engine version rather than this build's. It is what lets `charts lint
// --for-version` ask "does this chart's declared floor admit X?" without an X to
// hand.
//
// Note what that question is NOT: whether the chart actually runs on X. This
// binary carries one engine's behaviour, so it cannot emulate another's — only a
// real X can prove that. This checks the claim's shape, not its truth.
//
// It never returns an error. A constraint this build cannot make sense of yields
// CompatUnknown, not a failure: the check is a compatibility aid, not a security
// boundary — a chart already renders to an arbitrary stack — so failing open on
// our own inability to parse costs nothing, whereas failing closed would break
// working charts for a cosmetic reason.
func CheckCompatAgainst(cf Chartfile, engine string) CompatFinding {
	f := CompatFinding{
		Chart:    strings.TrimSpace(cf.Name + " " + cf.Version),
		Required: cf.SwarmcliVersion,
		Engine:   engine,
	}
	if cf.SwarmcliVersion == "" {
		return f // nothing declared: the common case, and silently fine
	}
	if len(cf.SwarmcliVersion) > maxConstraintLen {
		f.Reason = fmt.Sprintf("swarmcliVersion is longer than %d bytes", maxConstraintLen)
		return f
	}
	c, err := semver.NewConstraint(cf.SwarmcliVersion)
	if err != nil {
		f.Reason = fmt.Sprintf("swarmcliVersion %q is not a valid SemVer constraint", cf.SwarmcliVersion)
		return f
	}
	if strings.TrimSpace(engine) == "" {
		f.Reason = "this build reports no chart-engine version"
		return f
	}
	core, ok := coreVersion(engine)
	if !ok {
		f.Reason = fmt.Sprintf("chart-engine version %q is not valid SemVer", engine)
		return f
	}
	if c.Check(core) {
		f.Status = CompatOK
	} else {
		f.Status = CompatIncompatible
	}
	return f
}

// coreVersion parses an engine version down to major.minor.patch, dropping any
// prerelease and build metadata.
//
// The drop is deliberate. SemVer constraints exclude prereleases by default, so
// ">= 1.13.0" does not match "1.13.0-rc1" — yet a release candidate of 1.13.0
// carries 1.13.0's chart features. Without this, every chart requiring the
// version under test would be reported incompatible precisely while that release
// is being tested.
func coreVersion(v string) (*semver.Version, bool) {
	if strings.TrimSpace(v) == "" {
		return nil, false
	}
	parsed, err := semver.NewVersion(strings.TrimSpace(v))
	if err != nil {
		return nil, false
	}
	return semver.New(parsed.Major(), parsed.Minor(), parsed.Patch(), "", ""), true
}

// Message renders the finding as a one-line diagnostic naming what the chart
// wants and what this build has.
//
// binaryVersion is the version the binary reports for itself. When it differs
// from the engine's, both are named: a binary embedding this module may carry
// its own version, and naming only the engine's would cite a release the user
// cannot map back to anything they installed. Pass "" to name only the engine.
func (f CompatFinding) Message(binaryVersion string) string {
	var provides string
	switch {
	case f.Engine == "":
		provides = "this build reports no chart-engine version"
	case binaryVersion == "" || normalizeSemver(f.Engine) == normalizeSemver(binaryVersion):
		provides = fmt.Sprintf("this build provides %s", f.Engine)
	default:
		provides = fmt.Sprintf("this build provides chart engine %s (swarmcli %s)", f.Engine, binaryVersion)
	}
	return fmt.Sprintf("chart %s requires swarmcli %s; %s", f.Chart, f.Required, provides)
}

// compatHint returns a trailing note for an error an incompatible chart engine
// probably caused, or "" when it did not.
//
// A chart needing a newer engine usually fails inside Render first — with
// whatever error the missing feature happens to produce — so the finding never
// reaches the caller's compatibility gate. Naming it alongside that error turns
// `function "toYamlPretty" not defined` into a diagnosis.
func compatHint(f CompatFinding) string {
	if f.Status != CompatIncompatible {
		return ""
	}
	return "\n  (" + f.Message("") + " — this error is likely a consequence)"
}
