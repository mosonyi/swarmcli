// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package charts

import (
	"bytes"
	"context"
	"fmt"
	"maps"
	"os"

	"gopkg.in/yaml.v3"
)

// Action is what apply will do to one release.
type Action string

const (
	ActionInstall   Action = "install"
	ActionUpgrade   Action = "upgrade"
	ActionUnchanged Action = "unchanged"
)

// ReleasePlan is the computed desired state of one release.
type ReleasePlan struct {
	Name   string `json:"name"`
	Ref    string `json:"ref"`
	Action Action `json:"action"`
	// FromVersion is the currently deployed chart version, empty for an install.
	FromVersion string `json:"fromVersion,omitempty"`
	ToVersion   string `json:"toVersion"`

	Chart        ReleaseChart   `json:"chart"`
	Values       map[string]any `json:"values,omitempty"`
	Manifest     string         `json:"manifest"`
	Requirements *Requirements  `json:"requirements,omitempty"`
	// Files are the chart files Manifest references, resolved while the chart
	// was still in scope. Apply re-attaches them to InstallOptions per release,
	// exactly as it does Requirements.
	//
	// json:"-" because ReleasePlan is a wire type — a controller serialises a
	// plan to show what it would do — and base64 file bytes are neither
	// readable there nor anything a reader of a plan asked for. It is a
	// consequence, not a limitation: nothing reconstructs an Engine.Apply from
	// a plan's JSON.
	Files map[string][]byte `json:"-"`
	// CurrentManifest is the deployed manifest, for diffing. Empty for an install.
	CurrentManifest string `json:"currentManifest,omitempty"`
	// Compat is the chart's engine requirement checked against this build.
	// Planning records it but never acts on it: apply's contract is to plan
	// every release before converging any, so the whole plan is gated at once
	// by the caller — which is also the layer that knows whether blocking is
	// appropriate for the verb being run.
	Compat CompatFinding `json:"compat"`
}

// Plan is what apply would do to the whole swarm.
type Plan struct {
	// Owner is the owner id this plan was classified against: PlanOptions.Owner
	// when the caller supplied one, otherwise "apply/" and the release file's
	// `owner:` key. Empty when neither names an owner, in which case nothing on
	// the swarm is claimable and Orphaned is always empty.
	//
	// It is also the id Apply should stamp on what it writes — pass it as
	// InstallOptions.Owner — so that the next plan recognises these releases as
	// its own rather than as somebody else's.
	Owner string `json:"owner,omitempty"`
	// Releases, in file order.
	Releases []ReleasePlan `json:"releases"`
	// Unmanaged names releases that exist on the swarm, are absent from the
	// file, and carry no stamp saying this file produced them. Apply never
	// touches them — see Engine.Apply.
	Unmanaged []string `json:"unmanaged,omitempty"`
	// Orphaned names releases this file's own owner installed that the file no
	// longer declares. Unlike Unmanaged they are provably obsolete rather than
	// merely unrecognised, which is what makes deleting them safe. Apply still
	// does not delete them.
	Orphaned []string `json:"orphaned,omitempty"`
}

// Counts summarises a plan.
func (p *Plan) Counts() (install, upgrade, unchanged int) {
	for _, r := range p.Releases {
		switch r.Action {
		case ActionInstall:
			install++
		case ActionUpgrade:
			upgrade++
		case ActionUnchanged:
			unchanged++
		}
	}
	return install, upgrade, unchanged
}

// PlanOptions tune planning. The zero value is what `swarmcli charts apply`
// uses, and reproduces the behaviour of every release before these existed.
type PlanOptions struct {
	// Owner is the owner id the plan classifies deployed releases against,
	// overriding the "apply/<owner>" the release file would imply. A controller
	// installs under an id of its own — InstallOptions.Owner documents "cd/edge"
	// — and without this its own releases fail the ownership check and report as
	// Unmanaged from the first reconcile.
	//
	// Empty derives the id from the release file, which is what keeps a manifest
	// applied from the command line and a controller that happened to pick the
	// same name from claiming each other's releases. Set, it replaces that
	// derivation entirely: the file's `owner:` key is not consulted.
	Owner string
	// ReadFile reads one values file named by the release file, by the resolved
	// path ReleaseFile.ValuesPaths produced. Nil is os.ReadFile.
	//
	// It exists so that a caller can see and transform the bytes between "this
	// path was named" and "these values were merged" — decrypting a values file
	// committed encrypted, or serving it from a git object rather than a local
	// path at all — without the material having to reach a filesystem first.
	ReadFile func(path string) ([]byte, error)
}

// PlanApply computes what Apply would do, without writing anything.
//
// Every release is resolved, merged, schema-validated and rendered BEFORE any of
// them is deployed. A bad value in the third release therefore aborts the whole
// apply instead of leaving the swarm half-converged — and `--dry-run` is just
// "stop after planning".
func (e *Engine) PlanApply(ctx context.Context, rf *ReleaseFile, src ChartSource, opts PlanOptions) (*Plan, error) {
	owner := rf.ownerID()
	if opts.Owner != "" {
		if err := validateOwnerID(opts.Owner); err != nil {
			return nil, err
		}
		owner = opts.Owner
	}
	read := opts.ReadFile
	if read == nil {
		read = os.ReadFile
	}

	current, err := e.List(ctx)
	if err != nil {
		return nil, err
	}
	deployed := make(map[string]Release, len(current))
	for _, rel := range current {
		deployed[rel.Name] = rel
	}

	plan := &Plan{Owner: owner}
	managed := map[string]bool{}

	for _, spec := range rf.Releases {
		managed[spec.Name] = true

		rp, err := e.planRelease(rf, spec, src, deployed, read, owner)
		if err != nil {
			return nil, err
		}
		plan.Releases = append(plan.Releases, rp)
	}

	for _, rel := range current {
		switch {
		case managed[rel.Name]:
		case plan.Owner != "" && ownedBy(rel, plan.Owner):
			plan.Orphaned = append(plan.Orphaned, rel.Name)
		default:
			plan.Unmanaged = append(plan.Unmanaged, rel.Name)
		}
	}
	return plan, nil
}

// ownedBy reports whether a deployed release carries a stamp that this owner
// wrote for that exact release.
//
// Both halves have to match. An id-only comparison would re-introduce the bare
// owner label this encoding exists to avoid: a stamp copied onto a second
// release would read as owned, and prune would then delete a release nobody
// installed under that name. A stamp that does not parse is likewise not
// evidence of anything, so it counts as unowned.
func ownedBy(rel Release, id string) bool {
	own, err := ParseOwner(rel.Owner)
	if err != nil {
		return false
	}
	return own == OwnerRef{ID: id, Kind: OwnerKindRelease, Name: rel.Name}
}

// owner is the id PlanApply resolved for the whole plan (opts.Owner over the
// file's own), not rf.ownerID() — planRelease cannot re-derive it.
func (e *Engine) planRelease(rf *ReleaseFile, spec ReleaseSpec, src ChartSource, deployed map[string]Release, read func(string) ([]byte, error), owner string) (ReleasePlan, error) {
	ref := rf.ChartRef(spec)

	ch, err := src.Load(ref, spec.Version)
	if err != nil {
		return ReleasePlan{}, fmt.Errorf("%s: release %q: %w", rf.Path, spec.Name, err)
	}
	compat := CheckCompat(ch.Metadata)

	files, err := readFiles(rf.ValuesPaths(spec), read)
	if err != nil {
		return ReleasePlan{}, fmt.Errorf("%s: release %q: %w", rf.Path, spec.Name, err)
	}
	// No --set equivalent: the file is the only source of truth, so a value that
	// is not in it cannot influence what gets deployed.
	values, err := MergeValues(ch.Values, files, nil)
	if err != nil {
		return ReleasePlan{}, fmt.Errorf("%s: release %q: %w", rf.Path, spec.Name, err)
	}
	if err := ValidateValues(ch.Schema, values); err != nil {
		return ReleasePlan{}, fmt.Errorf("%s: release %q: %w", rf.Path, spec.Name, err)
	}

	rc := ReleaseChartOf(ch)
	rctx := RenderContext{
		Values:  values,
		Release: ReleaseMeta{Name: spec.Name, Namespace: spec.Name, Revision: 1},
		Chart:   ChartMeta{Name: rc.Name, Version: rc.Version, AppVersion: rc.AppVersion},
	}
	// A chart needing a newer engine usually fails here rather than reaching the
	// caller's gate, so name the requirement alongside whatever the missing
	// feature produced (compatHint is empty unless the chart is incompatible).
	manifest, err := Render(ch, rctx)
	if err != nil {
		return ReleasePlan{}, fmt.Errorf("%s: release %q: %w%s", rf.Path, spec.Name, err, compatHint(compat))
	}
	req, err := RenderRequirements(ch, rctx)
	if err != nil {
		return ReleasePlan{}, fmt.Errorf("%s: release %q: %w%s", rf.Path, spec.Name, err, compatHint(compat))
	}
	// One of the two places a *Chart and a rendered manifest coexist, so one of
	// the two places the manifest's file: keys can be resolved at all — the
	// chart is out of scope by the time this returns. A refusal fails the whole
	// plan before anything is deployed, and its text survives intact under the
	// same file/release prefix every other failure here carries: for a chart
	// that reads a path outside itself, that message is the entire migration
	// path there is.
	chartFiles, err := ResolveManifestFiles(manifest, ch.Files)
	if err != nil {
		return ReleasePlan{}, fmt.Errorf("%s: release %q: %w", rf.Path, spec.Name, err)
	}

	rp := ReleasePlan{
		Name:         spec.Name,
		Ref:          spec.Chart,
		ToVersion:    rc.Version,
		Chart:        rc,
		Values:       values,
		Manifest:     manifest,
		Requirements: req,
		Files:        chartFiles,
		Compat:       compat,
	}

	cur, ok := deployed[spec.Name]
	switch {
	case !ok:
		rp.Action = ActionInstall
	default:
		rp.FromVersion = cur.Chart.Version
		rp.CurrentManifest = cur.Manifest
		same, err := unchanged(&cur, rc, values, manifest, owner, chartFiles)
		if err != nil {
			return ReleasePlan{}, fmt.Errorf("%s: release %q: %w", rf.Path, spec.Name, err)
		}
		if same {
			rp.Action = ActionUnchanged
		} else {
			rp.Action = ActionUpgrade
		}
	}
	return rp, nil
}

// ApplyResult is what Apply actually did to one release.
type ApplyResult struct {
	Name     string `json:"name"`
	Action   Action `json:"action"`
	Revision int    `json:"revision,omitempty"` // 0 when unchanged (nothing was recorded)
}

// Apply converges the swarm to a plan, in file order.
//
// It never deletes. A release on the swarm that is absent from the file is
// reported and left alone — either as Plan.Unmanaged, where nothing says which
// manifest produced it and so it may belong to a second file or to a human, or
// as Plan.Orphaned, where its owner stamp names this file and it is therefore
// provably obsolete. Only the second is safe to remove, which is what the stamp
// exists to establish; acting on it is a separate change.
//
// Unchanged releases are skipped entirely. That is not an optimisation but a
// requirement: history is one Docker Config per revision, so an apply that
// recorded a revision even when nothing changed would grow the swarm's config
// store on every CI run, forever.
//
// It stops at the first failure and returns the results completed so far
// alongside the error, so a partial apply still reports what it did. Re-running
// is safe: the successful releases become no-ops.
//
// A cancelled context stops it the same way, and at the same seam: the next
// release is never started, and the error is the context's, so a caller can tell
// a shutdown from a release that failed. Checking here rather than relying on the
// deploy to fail is what keeps the boundary clean — Upgrade would otherwise be
// entered, and a stack half-deployed by a killed CLI is worse than one not
// deployed at all.
func (e *Engine) Apply(ctx context.Context, plan *Plan, opts InstallOptions) ([]ApplyResult, error) {
	var results []ApplyResult
	for _, rp := range plan.Releases {
		if err := ctx.Err(); err != nil {
			return results, err
		}
		if rp.Action == ActionUnchanged {
			results = append(results, ApplyResult{Name: rp.Name, Action: ActionUnchanged})
			continue
		}
		o := opts
		o.Requirements = rp.Requirements
		o.Files = rp.Files
		// Always Upgrade with Install set, never Install: a release that appeared
		// between planning and applying then upgrades cleanly instead of failing
		// with "already exists".
		o.Install = true
		rel, err := e.Upgrade(ctx, rp.Name, rp.Chart, rp.Values, rp.Manifest, o)
		if err != nil {
			return results, fmt.Errorf("release %q: %w", rp.Name, err)
		}
		results = append(results, ApplyResult{Name: rp.Name, Action: rp.Action, Revision: rel.Revision})
	}
	return results, nil
}

// unchanged reports whether the current revision already encodes the desired
// state: same chart, same rendered manifest, same files, same values — and the
// ownership this plan is being asked to establish.
//
// Files are in the comparison because they are deployed content that nothing
// else here can stand in for. A chart's files/ can change with neither a
// version bump nor a manifest change — editing files/nginx.conf and re-running
// `charts apply ./mychart` is the local-development loop — and while that
// comparison was missing, such a release planned as ActionUnchanged and the
// edited bytes never left the machine. Nil and empty compare equal: one means
// "this release ships no files" and so does the other.
//
// Requirements are deliberately not compared, and that asymmetry is meant. They
// declare what must already exist on the swarm for a deploy to work — external
// networks, secrets, configs — and are re-rendered and re-checked at pre-flight
// on every deploy that happens; they are not content the deploy sends. A
// requirements-only edit therefore changes nothing about the deployed release,
// and forcing a revision for it would spend a Docker Config to record a
// no-change. Adding them here would need its own argument, not this one.
//
// Ownership is part of the desired state because the stamp is the only evidence
// prune has. deployAndRecord is the sole writer of rel.Owner, and Apply skips
// ActionUnchanged without calling it, so leaving the owner out of this
// comparison meant a handover between two manifests with byte-identical content
// silently never happened: the plan reported "unchanged", which was true of the
// deployed resources and false of the ownership, and every later plan kept
// classifying the release under the departed owner (#511).
//
// A stamp is only *contradicted* when it names someone else. An absent stamp is
// not a mismatch: an unowned release is already classified conservatively — it
// falls to Plan.Unmanaged rather than Plan.Orphaned — so forcing a revision to
// stamp it would buy no safety, and would re-deploy every release installed
// before owner stamping existed on the first apply after upgrading. An
// unparseable stamp is no evidence of anything, so it is a mismatch and gets
// healed. An empty plan owner never forces: an apply that claims nothing must
// not strip a stamp somebody else wrote.
func unchanged(cur *Release, chart ReleaseChart, values map[string]any, manifest, owner string, files map[string][]byte) (bool, error) {
	if cur == nil || cur.Status == StatusUninstalled {
		return false, nil
	}
	if cur.Chart.Name != chart.Name || cur.Chart.Version != chart.Version {
		return false, nil
	}
	if cur.Manifest != manifest {
		return false, nil
	}
	if !maps.EqualFunc(cur.Files, files, bytes.Equal) {
		return false, nil
	}
	if owner != "" && cur.Owner != "" && !ownedBy(*cur, owner) {
		return false, nil
	}
	return sameValues(cur.Values, values)
}

// sameValues compares a freshly merged values map against one decoded from a
// stored release.
//
// reflect.DeepEqual is wrong here. The stored map came back through YAML, so a
// value written as 1.0 may decode as int(1) where the in-memory merge holds
// float64(1.0). DeepEqual would call those different, every release would look
// changed, and apply would write a new revision on every single run — the exact
// failure this function exists to prevent. Round-tripping both through the same
// encoder erases the type skew, and yaml.Marshal sorts map keys, so the result is
// canonical and order-independent. It is also precisely the encoding that gets
// persisted, so this compares what is actually stored.
func sameValues(a, b map[string]any) (bool, error) {
	ca, err := canonicalValues(a)
	if err != nil {
		return false, err
	}
	cb, err := canonicalValues(b)
	if err != nil {
		return false, err
	}
	return ca == cb, nil
}

func canonicalValues(v map[string]any) (string, error) {
	first, err := yaml.Marshal(v)
	if err != nil {
		return "", err
	}
	var back map[string]any
	if err := yaml.Unmarshal(first, &back); err != nil {
		return "", err
	}
	second, err := yaml.Marshal(back)
	if err != nil {
		return "", err
	}
	return string(second), nil
}

func readFiles(paths []string, read func(string) ([]byte, error)) ([][]byte, error) {
	var out [][]byte
	for _, p := range paths {
		b, err := read(p)
		if err != nil {
			return nil, fmt.Errorf("read values file %q: %w", p, err)
		}
		out = append(out, b)
	}
	return out, nil
}
