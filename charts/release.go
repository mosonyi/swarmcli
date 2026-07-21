// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package charts

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// maxConfigPayload guards against Docker's per-Config size limit (~500 KB). We
// gzip the release payload before storing; if it still exceeds this it cannot
// be saved as a Config.
const maxConfigPayload = 500 << 10 // 500 KiB

// ConfigMeta is the edition-agnostic view of a stored release Config.
type ConfigMeta struct {
	Name   string
	Labels map[string]string
}

// ServiceState is a live status line for a release's services, plus the facts
// --wait needs to decide whether the rollout is actually finished.
type ServiceState struct {
	Name     string
	Mode     string
	Replicas string // "running/desired" for replicated, "" otherwise — display only
	Status   string

	// Running counts tasks that are actually running on an active node.
	// Deliberately not derived from Replicas: that string counts tasks by
	// DESIRED state, so it reaches its target the moment Swarm schedules the
	// tasks rather than when they are up (see issue #480).
	Running int
	// Desired is the target task count over active nodes.
	Desired int
	// UpdateState is swarm's UpdateStatus.State, empty when the service has
	// never been updated. Empty means "no rollout has ever run" — NOT "the
	// rollout finished", which is why a fresh install cannot rely on it.
	UpdateState string
	// Monitor is UpdateConfig.Monitor: the window after a task is created in
	// which its failure still counts against the rollout.
	Monitor time.Duration
}

// Backend abstracts the Docker operations the release engine needs, so the
// lifecycle logic is unit-testable without a live Swarm.
type Backend interface {
	DeployStack(name, manifest string) error
	RemoveStack(name string) error
	// RefreshSnapshot invalidates the shared Docker state cache after a mutation
	// so subsequent reads (status, convergence polling) do not see stale data.
	RefreshSnapshot() error
	CreateConfig(ctx context.Context, name string, data []byte, labels map[string]string) error
	ListConfigs(ctx context.Context) ([]ConfigMeta, error)
	InspectConfig(ctx context.Context, name string) ([]byte, error)
	DeleteConfig(ctx context.Context, name string) error
	StackServices(name string) []ServiceState
	StackVolumes(ctx context.Context, name string) ([]string, error)
	RemoveVolume(ctx context.Context, name string) error
	// NetworkScopes returns existing network names mapped to their scope
	// (e.g. "swarm", "local"), used to pre-flight a chart's external networks.
	NetworkScopes(ctx context.Context) (map[string]string, error)
	// CreateOverlayNetwork creates a swarm-scoped network with the given driver
	// and attachability (driver defaults to "overlay" when a chart does not
	// declare one in requirements.yaml).
	CreateOverlayNetwork(ctx context.Context, name, driver string, attachable bool) error
	// RemoveOverlayNetwork removes a network by name, used to roll back networks
	// auto-created for an install whose deploy then failed. A no-op if absent.
	RemoveOverlayNetwork(ctx context.Context, name string) error
	// SecretNames returns the set of existing swarm secret names, used to
	// pre-flight a chart's external secrets (which cannot be auto-created).
	SecretNames(ctx context.Context) (map[string]struct{}, error)
}

// Engine drives release lifecycle operations against a Backend.
type Engine struct {
	Backend Backend
	// now returns the current time; overridable in tests.
	now func() time.Time
}

// NewEngine returns an Engine bound to the live Docker backend.
func NewEngine() *Engine {
	return &Engine{Backend: &dockerBackend{}, now: time.Now}
}

// NewEngineWith returns an Engine bound to a custom backend (used in tests).
func NewEngineWith(b Backend) *Engine {
	return &Engine{Backend: b, now: time.Now}
}

// InstallOptions tune an install or upgrade.
type InstallOptions struct {
	DryRun     bool
	Wait       bool
	Install    bool // upgrade: create the release if it does not exist
	Timeout    time.Duration
	HistoryMax int // 0 = keep all
	// Requirements is the chart's parsed requirements.yaml, when present. It
	// drives the external-resource pre-flight (auto-create vs validate-only, the
	// network driver/attachability, and remediation descriptions) and, when set,
	// every external resource the manifest references must be declared in it. Nil
	// falls back to manifest-driven pre-flight (auto-create attachable overlays).
	Requirements *Requirements
}

// Install deploys a freshly rendered manifest as revision 1 of a new release and
// records it. It refuses to install over an existing, non-uninstalled release
// (use upgrade — Phase 2). manifest must already be rendered and validated.
func (e *Engine) Install(ctx context.Context, release string, chart ReleaseChart, values map[string]any, manifest string, opts InstallOptions) (*Release, error) {
	if err := validateReleaseName(release); err != nil {
		return nil, err
	}
	revs, err := e.revisions(ctx, release)
	if err != nil {
		return nil, err
	}
	if cur := currentRevision(revs); cur != nil && cur.Status != StatusUninstalled {
		return nil, fmt.Errorf("release %q already exists (revision %d); use upgrade", release, cur.Revision)
	}

	rel := e.newRevision(release, nextRevision(revs), chart, values, manifest)
	return e.deployAndRecord(ctx, rel, opts)
}

// Upgrade deploys a new revision of an existing release. When the release does
// not exist it errors unless opts.Install is set (the `upgrade --install`
// behavior). manifest must already be rendered and validated.
func (e *Engine) Upgrade(ctx context.Context, release string, chart ReleaseChart, values map[string]any, manifest string, opts InstallOptions) (*Release, error) {
	if err := validateReleaseName(release); err != nil {
		return nil, err
	}
	revs, err := e.revisions(ctx, release)
	if err != nil {
		return nil, err
	}
	cur := currentRevision(revs)
	if cur == nil || cur.Status == StatusUninstalled {
		if !opts.Install {
			return nil, fmt.Errorf("release %q does not exist; use install or upgrade --install", release)
		}
	}
	rel := e.newRevision(release, nextRevision(revs), chart, values, manifest)
	return e.deployAndRecord(ctx, rel, opts)
}

// Rollback deploys a new revision whose content is copied from a previous
// revision (append-only, mirroring Helm). targetRev must be an existing,
// non-failed revision.
func (e *Engine) Rollback(ctx context.Context, release string, targetRev int, opts InstallOptions) (*Release, error) {
	revs, err := e.revisions(ctx, release)
	if err != nil {
		return nil, err
	}
	if len(revs) == 0 {
		return nil, fmt.Errorf("release %q not found", release)
	}
	var target *Release
	for i := range revs {
		if revs[i].Revision == targetRev {
			target = &revs[i]
		}
	}
	if target == nil {
		return nil, fmt.Errorf("release %q has no revision %d", release, targetRev)
	}
	if target.Status == StatusFailed {
		return nil, fmt.Errorf("cannot roll back to failed revision %d", targetRev)
	}
	rel := e.newRevision(release, nextRevision(revs), target.Chart, target.Values, target.Manifest)
	return e.deployAndRecord(ctx, rel, opts)
}

// History returns every stored revision of a release, ascending, with derived
// display statuses.
func (e *Engine) History(ctx context.Context, release string) ([]Release, error) {
	revs, err := e.revisions(ctx, release)
	if err != nil {
		return nil, err
	}
	if len(revs) == 0 {
		return nil, fmt.Errorf("release %q not found", release)
	}
	return revs, nil
}

// GetRevision returns a specific revision of a release, or the current one when
// rev <= 0.
func (e *Engine) GetRevision(ctx context.Context, release string, rev int) (*Release, error) {
	revs, err := e.revisions(ctx, release)
	if err != nil {
		return nil, err
	}
	if len(revs) == 0 {
		return nil, fmt.Errorf("release %q not found", release)
	}
	if rev <= 0 {
		return currentRevision(revs), nil
	}
	for i := range revs {
		if revs[i].Revision == rev {
			return &revs[i], nil
		}
	}
	return nil, fmt.Errorf("release %q has no revision %d", release, rev)
}

// newRevision builds an unsaved, deployed-status revision.
func (e *Engine) newRevision(release string, rev int, chart ReleaseChart, values map[string]any, manifest string) *Release {
	return &Release{
		Name:      release,
		Revision:  rev,
		Status:    StatusDeployed,
		Chart:     chart,
		Values:    values,
		Manifest:  manifest,
		Namespace: release,
		Created:   e.now().UTC().Format(time.RFC3339),
	}
}

// deployAndRecord deploys a revision's manifest and records it. On DryRun it
// returns the prospective revision without touching Docker. On deploy failure
// it records nothing, leaving the release retryable (no orphaned revision).
func (e *Engine) deployAndRecord(ctx context.Context, rel *Release, opts InstallOptions) (*Release, error) {
	if opts.DryRun {
		return rel, nil
	}
	// Validate prerequisites that cannot be auto-created (external secrets and
	// configs) before mutating any swarm state, so a missing one fails fast
	// without leaving auto-created networks behind.
	if err := e.ensureExternalSecretsConfigs(ctx, rel.Manifest, opts.Requirements); err != nil {
		rel.Status = StatusFailed
		return rel, err
	}
	created, err := e.ensureExternalNetworks(ctx, rel.Manifest, opts.Requirements)
	if err != nil {
		rel.Status = StatusFailed
		for _, n := range created {
			_ = e.Backend.RemoveOverlayNetwork(ctx, n)
		}
		return rel, err
	}
	if err := e.Backend.DeployStack(rel.Name, rel.Manifest); err != nil {
		rel.Status = StatusFailed
		// Roll back networks we auto-created for this install so a failed deploy
		// leaves no trace; pre-existing networks are untouched.
		for _, n := range created {
			_ = e.Backend.RemoveOverlayNetwork(ctx, n)
		}
		// Do not record on failure: a failed deploy must leave no release Config
		// behind, so the release stays retryable (no orphaned "already exists").
		return rel, fmt.Errorf("deploy failed: %w", err)
	}
	// The deploy mutated swarm state; invalidate the shared cache so the
	// convergence poll and any follow-up status read see fresh data.
	_ = e.Backend.RefreshSnapshot()
	// Persist the networks we auto-created this revision so uninstall can report
	// what it leaves behind (see Uninstall). Networks already present from an
	// earlier revision are not re-created here, so uninstall unions across all
	// revisions.
	rel.ManagedNetworks = created
	if err := e.record(ctx, rel); err != nil {
		return rel, fmt.Errorf("stack %q was deployed but recording its release history failed: %w; re-run install/upgrade to reconcile", rel.Name, err)
	}
	if opts.Wait {
		if err := e.waitReady(rel.Name, opts.Timeout); err != nil {
			return rel, err
		}
	}
	if opts.HistoryMax > 0 {
		e.pruneHistory(ctx, rel.Name, opts.HistoryMax)
	}
	return rel, nil
}

// ensureExternalNetworks makes sure every network the manifest declares
// external exists and is swarm-scoped before the stack is deployed.
//
// When the chart ships a requirements.yaml (req != nil) it is authoritative:
// every external network the manifest references must be declared there, and the
// declaration drives behaviour — autoCreate:true networks are created with the
// declared driver/attachability, autoCreate:false networks are only validated
// (a missing one is a hard error with the declared description, never created).
// Without a requirements.yaml (req == nil) it falls back to the historical
// behaviour: create any missing network as an attachable overlay.
//
// Networks that exist with a non-swarm scope, that cannot be created, or that
// are required-present-but-missing are reported with the manual remediation
// needed to resolve them. ensureExternalNetworks returns the names of networks
// it auto-created, so a caller can roll them back if a later step (the deploy)
// fails.
func (e *Engine) ensureExternalNetworks(ctx context.Context, manifest string, req *Requirements) (created []string, err error) {
	names, err := externalNetworks(manifest)
	if err != nil || len(names) == 0 {
		return nil, err
	}
	// Enforce the requirements.yaml contract first: an undeclared external
	// resource is a chart-authoring error, distinct from one being unavailable.
	if err := requireDeclared(req, names, "network", "networks"); err != nil {
		return nil, err
	}
	scopes, err := e.Backend.NetworkScopes(ctx)
	if err != nil {
		return nil, fmt.Errorf("checking external networks: %w", err)
	}
	var reasons, fixes []string
	for _, name := range names {
		nr, declared := req.network(name) // declared == (req != nil), per the contract check above
		scope, exists := scopes[name]
		switch {
		case exists && scope == "swarm":
			// already usable
		case exists:
			reasons = append(reasons,
				fmt.Sprintf("  %s: a non-swarm (%s) network of this name already exists; remove or rename it", name, scope))
		case declared && !*nr.AutoCreate:
			// Validate-only: the chart depends on a network it must not create.
			reasons = append(reasons, fmt.Sprintf("  %s: does not exist%s", name, describe(nr.Description)))
			fixes = append(fixes, fmt.Sprintf("  docker network create --driver %s%s %s", networkDriver(nr), attachableFlag(nr), name))
		default:
			driver, attachable := "overlay", true
			if declared {
				driver, attachable = networkDriver(nr), *nr.Attachable
			}
			if cerr := e.Backend.CreateOverlayNetwork(ctx, name, driver, attachable); cerr != nil {
				reasons = append(reasons, fmt.Sprintf("  %s: auto-create failed: %v", name, cerr))
				fixes = append(fixes, fmt.Sprintf("  docker network create --driver %s%s %s", driver, attachableFlagBool(attachable), name))
			} else {
				created = append(created, name)
			}
		}
	}
	if len(reasons) == 0 {
		return created, nil
	}
	msg := "external network(s) required by this chart are unavailable:\n" + strings.Join(reasons, "\n")
	if len(fixes) > 0 {
		msg += "\ncreate them on a swarm manager, then retry:\n" + strings.Join(fixes, "\n")
	}
	return created, fmt.Errorf("%s", msg)
}

// requireDeclared enforces the requirements.yaml contract: when req is non-nil,
// every external resource name the manifest references must be declared in it.
// It returns a chart-authoring error listing any undeclared names, or nil when
// req is nil (no requirements.yaml — manifest-driven fallback) or all declared.
func requireDeclared(req *Requirements, names []string, kind, key string) error {
	if req == nil {
		return nil
	}
	var undeclared []string
	for _, n := range names {
		var ok bool
		switch key {
		case "networks":
			_, ok = req.network(n)
		case "secrets":
			_, ok = req.secret(n)
		case "configs":
			_, ok = req.config(n)
		}
		if !ok {
			undeclared = append(undeclared, fmt.Sprintf("  %s", n))
		}
	}
	if len(undeclared) == 0 {
		return nil
	}
	return fmt.Errorf("%s(s) the manifest declares external are not declared in %s:\n%s\n"+
		"declare them under %s: in %s (it is authoritative when present)",
		kind, requirementsName, strings.Join(undeclared, "\n"), key, requirementsName)
}

// networkDriver returns the declared driver, defaulting to overlay (defaults are
// normally applied at parse time; this guards direct callers/tests).
func networkDriver(nr *NetworkRequirement) string {
	if nr.Driver == "" {
		return "overlay"
	}
	return nr.Driver
}

func attachableFlag(nr *NetworkRequirement) string {
	return attachableFlagBool(nr.Attachable == nil || *nr.Attachable)
}

func attachableFlagBool(attachable bool) string {
	if attachable {
		return " --attachable"
	}
	return ""
}

// describe renders an optional requirement description as a trailing " (…)"
// clause for remediation messages, or "" when absent.
func describe(desc string) string {
	if desc == "" {
		return ""
	}
	return " (" + desc + ")"
}

// ensureExternalSecretsConfigs verifies that every secret and config the
// manifest declares external already exists on the swarm. Unlike networks,
// these cannot be auto-created — their content is not part of the chart — so a
// missing one is a hard error that lists the `docker secret/config create`
// commands needed to resolve it. A manifest with no external secrets/configs is
// a no-op.
//
// When the chart ships a requirements.yaml (req != nil) it is authoritative:
// every external secret/config the manifest references must be declared there
// (else a hard error), and a declared description enriches the remediation.
func (e *Engine) ensureExternalSecretsConfigs(ctx context.Context, manifest string, req *Requirements) error {
	secrets := externalResourceNames(manifest, "secrets")
	configs := externalResourceNames(manifest, "configs")
	if len(secrets) == 0 && len(configs) == 0 {
		return nil
	}
	// Enforce the requirements.yaml contract first (chart-authoring error),
	// separately from the existence check below (operator remediation).
	if err := requireDeclared(req, secrets, "secret", "secrets"); err != nil {
		return err
	}
	if err := requireDeclared(req, configs, "config", "configs"); err != nil {
		return err
	}

	var reasons, cmds []string

	if len(secrets) > 0 {
		have, err := e.Backend.SecretNames(ctx)
		if err != nil {
			return fmt.Errorf("checking external secrets: %w", err)
		}
		for _, name := range secrets {
			rr, _ := req.secret(name)
			if _, ok := have[name]; !ok {
				reasons = append(reasons, fmt.Sprintf("  secret %q does not exist%s", name, requirementDescription(rr)))
				cmds = append(cmds, fmt.Sprintf("  docker secret create %s <file>", name))
			}
		}
	}

	if len(configs) > 0 {
		metas, err := e.Backend.ListConfigs(ctx)
		if err != nil {
			return fmt.Errorf("checking external configs: %w", err)
		}
		have := make(map[string]struct{}, len(metas))
		for _, m := range metas {
			have[m.Name] = struct{}{}
		}
		for _, name := range configs {
			rr, _ := req.config(name)
			if _, ok := have[name]; !ok {
				reasons = append(reasons, fmt.Sprintf("  config %q does not exist%s", name, requirementDescription(rr)))
				cmds = append(cmds, fmt.Sprintf("  docker config create %s <file>", name))
			}
		}
	}

	if len(reasons) == 0 {
		return nil
	}
	return fmt.Errorf("external secret(s)/config(s) required by this chart do not exist:\n%s\n"+
		"create them on a swarm manager, then retry:\n%s",
		strings.Join(reasons, "\n"), strings.Join(cmds, "\n"))
}

// requirementDescription renders a resource requirement's description as a
// trailing " (…)" clause, or "" when there is no requirement or description.
func requirementDescription(rr *ResourceRequirement) string {
	if rr == nil {
		return ""
	}
	return describe(rr.Description)
}

// UninstallResult reports what an uninstall left behind. OrphanedNetworks are
// the external networks swarmcli auto-created for the release that still exist
// after the stack is removed — `docker stack rm` does not remove external
// networks, and swarmcli deliberately leaves them (they may be shared with other
// stacks) and reports them instead.
type UninstallResult struct {
	OrphanedNetworks []string
}

// Uninstall removes the release's stack and its recorded revisions, retaining
// volumes unless purgeVolumes is set. It returns an UninstallResult describing
// the auto-created external networks left in place so the caller can surface
// them; the networks themselves are not removed.
func (e *Engine) Uninstall(ctx context.Context, release string, purgeVolumes bool) (*UninstallResult, error) {
	revs, err := e.revisions(ctx, release)
	if err != nil {
		return nil, err
	}
	if len(revs) == 0 {
		return nil, fmt.Errorf("release %q not found", release)
	}

	// Continue cleanup on partial failure rather than aborting: a stranded
	// history Config or volume must not be left behind because an earlier step
	// failed (e.g. a retry where the stack is already gone). Errors are
	// aggregated and returned together.
	var errs []error
	if err := e.Backend.RemoveStack(release); err != nil {
		errs = append(errs, fmt.Errorf("removing stack: %w", err))
	} else {
		_ = e.Backend.RefreshSnapshot()
	}

	if purgeVolumes {
		if vols, err := e.Backend.StackVolumes(ctx, release); err != nil {
			errs = append(errs, fmt.Errorf("listing volumes: %w", err))
		} else {
			for _, v := range vols {
				if err := e.Backend.RemoveVolume(ctx, v); err != nil {
					errs = append(errs, fmt.Errorf("removing volume %q: %w", v, err))
				}
			}
		}
	}

	// Collect the networks swarmcli auto-created across all revisions (a network
	// created in an early revision is not re-created later, so it only appears on
	// that revision's record — union them), keeping only those that still exist.
	result := &UninstallResult{OrphanedNetworks: e.orphanedManagedNetworks(ctx, revs)}

	for _, r := range revs {
		if err := e.Backend.DeleteConfig(ctx, releaseConfigName(release, r.Revision)); err != nil {
			errs = append(errs, fmt.Errorf("deleting release config: %w", err))
		}
	}
	return result, errors.Join(errs...)
}

// orphanedManagedNetworks returns the sorted, de-duplicated set of networks
// swarmcli auto-created for the release (recorded per-revision) that still exist
// on the swarm. A failure to query existing networks degrades to reporting the
// recorded union (better to over-report a cleanup hint than to hide it).
func (e *Engine) orphanedManagedNetworks(ctx context.Context, revs []Release) []string {
	seen := map[string]struct{}{}
	var managed []string
	for _, r := range revs {
		for _, n := range r.ManagedNetworks {
			if _, ok := seen[n]; !ok {
				seen[n] = struct{}{}
				managed = append(managed, n)
			}
		}
	}
	if len(managed) == 0 {
		return nil
	}
	scopes, err := e.Backend.NetworkScopes(ctx)
	if err != nil {
		sort.Strings(managed)
		return managed
	}
	var still []string
	for _, n := range managed {
		if _, ok := scopes[n]; ok {
			still = append(still, n)
		}
	}
	sort.Strings(still)
	return still
}

// List returns the current (highest) revision of every release.
func (e *Engine) List(ctx context.Context) ([]Release, error) {
	byRelease, err := e.allRevisions(ctx)
	if err != nil {
		return nil, err
	}
	var out []Release
	for _, revs := range byRelease {
		if cur := currentRevision(revs); cur != nil && cur.Status != StatusUninstalled {
			out = append(out, *cur)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// Status returns the current release plus live service states for its stack.
func (e *Engine) Status(ctx context.Context, release string) (*Release, []ServiceState, error) {
	revs, err := e.revisions(ctx, release)
	if err != nil {
		return nil, nil, err
	}
	cur := currentRevision(revs)
	if cur == nil {
		return nil, nil, fmt.Errorf("release %q not found", release)
	}
	return cur, e.Backend.StackServices(release), nil
}

// --- helpers ---

// revisions returns all stored revisions for one release, ascending.
func (e *Engine) revisions(ctx context.Context, release string) ([]Release, error) {
	all, err := e.allRevisions(ctx)
	if err != nil {
		return nil, err
	}
	return all[release], nil
}

// allRevisions loads every release revision, grouped by release name (ascending).
func (e *Engine) allRevisions(ctx context.Context) (map[string][]Release, error) {
	metas, err := e.Backend.ListConfigs(ctx)
	if err != nil {
		return nil, err
	}
	out := map[string][]Release{}
	for _, m := range metas {
		if m.Labels[LabelType] != TypeRelease {
			continue
		}
		data, err := e.Backend.InspectConfig(ctx, m.Name)
		if err != nil {
			return nil, fmt.Errorf("read release config %q: %w", m.Name, err)
		}
		rel, err := decodeRelease(data)
		if err != nil {
			return nil, fmt.Errorf("decode release config %q: %w", m.Name, err)
		}
		out[rel.Name] = append(out[rel.Name], *rel)
	}
	for name := range out {
		revs := out[name]
		sort.Slice(revs, func(i, j int) bool { return revs[i].Revision < revs[j].Revision })
		out[name] = deriveStatuses(revs)
	}
	return out, nil
}

// deriveStatuses computes display statuses from the immutable append-only log:
// the highest revision keeps its stored status; every lower revision that was
// deployed becomes "superseded".
func deriveStatuses(revs []Release) []Release {
	if len(revs) == 0 {
		return revs
	}
	top := len(revs) - 1
	for i := range revs {
		if i != top && revs[i].Status == StatusDeployed {
			revs[i].Status = StatusSuperseded
		}
	}
	return revs
}

func currentRevision(revs []Release) *Release {
	if len(revs) == 0 {
		return nil
	}
	r := revs[len(revs)-1]
	return &r
}

func nextRevision(revs []Release) int {
	if len(revs) == 0 {
		return 1
	}
	return revs[len(revs)-1].Revision + 1
}

// maxRecordRetries bounds the TOCTOU retry when a concurrent install/upgrade
// claims the same revision number.
const maxRecordRetries = 5

// record stores a release revision as a gzipped, labeled Docker Config. The
// revision number is computed before deploy, so a concurrent install/upgrade
// can claim the same name; CreateConfig is atomic on the name, so on collision
// we re-read the history and retry with the next free revision, keeping the
// append-only log consistent.
func (e *Engine) record(ctx context.Context, rel *Release) error {
	for attempt := 0; attempt <= maxRecordRetries; attempt++ {
		err := e.storeRevision(ctx, rel)
		if err == nil || !isAlreadyExists(err) {
			return err
		}
		revs, rerr := e.revisions(ctx, rel.Name)
		if rerr != nil {
			return err // surface the original collision error
		}
		rel.Revision = nextRevision(revs)
	}
	return fmt.Errorf("could not allocate a free revision for release %q after %d attempts", rel.Name, maxRecordRetries)
}

// isAlreadyExists reports whether err is a Docker "config already exists"
// conflict (the name was claimed concurrently).
func isAlreadyExists(err error) bool {
	return err != nil && strings.Contains(err.Error(), "already exists")
}

// storeRevision writes a single release revision as a gzipped, labeled Config.
func (e *Engine) storeRevision(ctx context.Context, rel *Release) error {
	payload, err := yaml.Marshal(rel)
	if err != nil {
		return err
	}
	gz, err := gzipBytes(payload)
	if err != nil {
		return err
	}
	if len(gz) > maxConfigPayload {
		return fmt.Errorf("release payload is %d bytes gzipped, exceeding the %d-byte Docker Config limit", len(gz), maxConfigPayload)
	}
	labels := map[string]string{
		LabelType:         TypeRelease,
		LabelRelease:      rel.Name,
		LabelChart:        rel.Chart.Name,
		LabelChartVersion: rel.Chart.Version,
		LabelRevision:     strconv.Itoa(rel.Revision),
		LabelStatus:       rel.Status,
		LabelCreated:      rel.Created,
	}
	return e.Backend.CreateConfig(ctx, releaseConfigName(rel.Name, rel.Revision), gz, labels)
}

// PruneAction is the keep/delete decision for one revision in a prune.
type PruneAction struct {
	Revision int
	Delete   bool
	Current  bool // the live (highest) revision; never deleted
}

// PruneResult reports what a prune did (or, for a dry run, would do) for one
// release. Actions are ascending by revision.
type PruneResult struct {
	Release string
	Actions []PruneAction
}

// Deleted returns the revision numbers Prune removed (or would remove).
func (r PruneResult) Deleted() []int {
	var out []int
	for _, a := range r.Actions {
		if a.Delete {
			out = append(out, a.Revision)
		}
	}
	return out
}

// planPrune decides, for revisions sorted ascending, which to delete to retain
// the newest keep. keep <= 0 keeps everything. The highest (current) revision is
// always retained regardless of keep, so the live release is never destroyed.
func planPrune(revs []Release, keep int) []PruneAction {
	actions := make([]PruneAction, len(revs))
	top := len(revs) - 1
	for i, r := range revs {
		current := i == top
		del := keep > 0 && i < len(revs)-keep && !current
		actions[i] = PruneAction{Revision: r.Revision, Delete: del, Current: current}
	}
	return actions
}

// Prune deletes superseded revisions of one release beyond the keep window,
// always retaining the current (highest) revision. keep <= 0 keeps everything.
// On a dry run no Config is touched. Deletion failures are aggregated and
// returned, but pruning of the remaining revisions continues.
func (e *Engine) Prune(ctx context.Context, release string, keep int, dryRun bool) (PruneResult, error) {
	revs, err := e.revisions(ctx, release)
	if err != nil {
		return PruneResult{}, err
	}
	if len(revs) == 0 {
		return PruneResult{}, fmt.Errorf("release %q not found", release)
	}
	return e.pruneRevs(ctx, release, revs, keep, dryRun)
}

// pruneRevs applies the retention plan to a release whose revisions are already
// loaded (ascending, non-empty). Splitting this out lets PruneAll prune from the
// single allRevisions load instead of re-reading every config per release.
func (e *Engine) pruneRevs(ctx context.Context, release string, revs []Release, keep int, dryRun bool) (PruneResult, error) {
	res := PruneResult{Release: release, Actions: planPrune(revs, keep)}
	if dryRun {
		return res, nil
	}
	var errs []error
	for _, a := range res.Actions {
		if !a.Delete {
			continue
		}
		if derr := e.Backend.DeleteConfig(ctx, releaseConfigName(release, a.Revision)); derr != nil {
			errs = append(errs, fmt.Errorf("deleting %s: %w", releaseConfigName(release, a.Revision), derr))
		}
	}
	return res, errors.Join(errs...)
}

// PruneAll prunes every release to the keep window, returning one result per
// release (sorted by name). Per-release errors are aggregated; a failing release
// does not stop the others.
func (e *Engine) PruneAll(ctx context.Context, keep int, dryRun bool) ([]PruneResult, error) {
	byRelease, err := e.allRevisions(ctx)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(byRelease))
	for name := range byRelease {
		names = append(names, name)
	}
	sort.Strings(names)
	var results []PruneResult
	var errs []error
	for _, name := range names {
		res, perr := e.pruneRevs(ctx, name, byRelease[name], keep, dryRun)
		if perr != nil {
			errs = append(errs, perr)
		}
		results = append(results, res)
	}
	return results, errors.Join(errs...)
}

// pruneHistory trims a release's history to keep revisions after a successful
// deploy. It is best-effort: deletion errors are ignored because the deploy
// itself already succeeded and recorded. Retention logic is shared with Prune
// via planPrune, so the keep <= 0 / current-revision guarantees hold here too.
func (e *Engine) pruneHistory(ctx context.Context, release string, keep int) {
	revs, err := e.revisions(ctx, release)
	if err != nil {
		return
	}
	for _, a := range planPrune(revs, keep) {
		if a.Delete {
			_ = e.Backend.DeleteConfig(ctx, releaseConfigName(release, a.Revision))
		}
	}
}

// waitPollInterval is how often waitReady re-reads service state. A variable so
// tests can shrink it; a real sleep in a unit test is otherwise unavoidable
// because the stability window is wall-clock.
var waitPollInterval = 2 * time.Second

// defaultStabilityWindow mirrors the Docker CLI's own default monitor period.
// Reaching the target replica count is necessary but not sufficient: a task that
// starts and then dies inside UpdateConfig.Monitor counts as a rollout failure,
// so returning at first parity reports success for a service that is about to
// crash-loop.
const defaultStabilityWindow = 5 * time.Second

// waitReady polls until every service in the release is running its target task
// count and has held it for the stability window, or the timeout elapses.
//
// It returns early with an error when swarm reports the rollout wedged
// ("paused" / "rollback_paused"): swarmkit never rolls back a rollback, so
// those states need a human and waiting out the timeout only delays the report.
func (e *Engine) waitReady(release string, timeout time.Duration) error {
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	deadline := e.now().Add(timeout)
	var convergedAt time.Time
	for {
		states := e.Backend.StackServices(release)
		if err := rolloutWedged(states); err != nil {
			return fmt.Errorf("release %q: %w", release, err)
		}
		if len(states) > 0 && allConverged(states) {
			if convergedAt.IsZero() {
				convergedAt = e.now()
			}
			if !e.now().Before(convergedAt.Add(stabilityWindow(states))) {
				return nil
			}
		} else {
			// Lost parity — a task died inside the window. Start it again rather
			// than counting the earlier, now-invalidated stretch.
			convergedAt = time.Time{}
		}
		if !e.now().Before(deadline) {
			return fmt.Errorf("timed out after %s waiting for release %q to converge", timeout, release)
		}
		time.Sleep(waitPollInterval)
	}
}

// stabilityWindow is the longest monitor period across the release's services,
// so the slowest service governs. Services declaring none use the CLI default.
func stabilityWindow(states []ServiceState) time.Duration {
	window := defaultStabilityWindow
	for _, s := range states {
		if s.Monitor > window {
			window = s.Monitor
		}
	}
	return window
}

// rolloutWedged reports a rollout swarm has given up on.
func rolloutWedged(states []ServiceState) error {
	for _, s := range states {
		switch s.UpdateState {
		case "paused":
			return fmt.Errorf("service %q: update paused after a task failure; swarm will not continue without intervention", s.Name)
		case "rollback_paused":
			return fmt.Errorf("service %q: rollback paused; swarm never rolls back a rollback, so this needs manual recovery", s.Name)
		}
	}
	return nil
}

// allConverged reports whether every service is running its target task count.
//
// It uses the actual running count rather than the Replicas display string.
// That string counts tasks by DESIRED state, so it reaches parity the instant
// swarm schedules the tasks — before any container is up — which made --wait
// return immediately on a fresh install, where UpdateState is empty and the
// in-flight guard below cannot fire either (issue #473).
func allConverged(states []ServiceState) bool {
	for _, s := range states {
		// A rolling update in flight is not converged even if the task count
		// already reads N/N: the count may still be the outgoing generation.
		switch s.UpdateState {
		case "updating", "rollback_started":
			return false
		}
		if s.Running < s.Desired {
			return false
		}
	}
	return true
}

func releaseConfigName(release string, rev int) string {
	return fmt.Sprintf("swarmcli.release.%s.v%d", release, rev)
}

func gzipBytes(b []byte) ([]byte, error) {
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	if _, err := w.Write(b); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func decodeRelease(gz []byte) (*Release, error) {
	r, err := gzip.NewReader(bytes.NewReader(gz))
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.Close() }()
	raw, err := io.ReadAll(io.LimitReader(r, maxChartFileSize+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > maxChartFileSize {
		return nil, fmt.Errorf("release payload exceeds %d bytes", maxChartFileSize)
	}
	var rel Release
	if err := yaml.Unmarshal(raw, &rel); err != nil {
		return nil, err
	}
	return &rel, nil
}

// validateReleaseName mirrors Docker stack naming constraints.
func validateReleaseName(name string) error {
	if name == "" {
		return fmt.Errorf("release name is required")
	}
	for _, r := range name {
		if !(r == '-' || r == '_' || r == '.' ||
			(r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
			return fmt.Errorf("invalid release name %q: use letters, digits, '-', '_', '.'", name)
		}
	}
	return nil
}
