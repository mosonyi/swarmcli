// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package charts

import (
	"bytes"
	"compress/gzip"
	"context"
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

// ServiceState is a minimal live status line for a release's services.
type ServiceState struct {
	Name     string
	Mode     string
	Replicas string // "running/desired" for replicated, "" otherwise
	Status   string
}

// Backend abstracts the Docker operations the release engine needs, so the
// lifecycle logic is unit-testable without a live Swarm.
type Backend interface {
	DeployStack(name, manifest string) error
	RemoveStack(name string) error
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
	// CreateOverlayNetwork creates an attachable, swarm-scoped overlay network.
	CreateOverlayNetwork(ctx context.Context, name string) error
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
	if err := e.ensureExternalNetworks(ctx, rel.Manifest); err != nil {
		rel.Status = StatusFailed
		return rel, err
	}
	if err := e.Backend.DeployStack(rel.Name, rel.Manifest); err != nil {
		rel.Status = StatusFailed
		// Do not record on failure: a failed deploy must leave no release Config
		// behind, so the release stays retryable (no orphaned "already exists").
		return rel, fmt.Errorf("deploy failed: %w", err)
	}
	if err := e.record(ctx, rel); err != nil {
		return rel, fmt.Errorf("deployed, but recording release failed: %w", err)
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
// external exists and is swarm-scoped before the stack is deployed. Missing
// networks are created as attachable swarm overlays; networks that exist with a
// non-swarm scope, or that cannot be created, are reported with the manual
// `docker network create` commands needed to resolve them.
func (e *Engine) ensureExternalNetworks(ctx context.Context, manifest string) error {
	names, err := externalNetworks(manifest)
	if err != nil || len(names) == 0 {
		return err
	}
	scopes, err := e.Backend.NetworkScopes(ctx)
	if err != nil {
		return fmt.Errorf("checking external networks: %w", err)
	}
	var reasons, needCreate []string
	for _, name := range names {
		switch scope, exists := scopes[name]; {
		case exists && scope == "swarm":
			// already usable
		case exists:
			reasons = append(reasons,
				fmt.Sprintf("  %s: a non-swarm (%s) network of this name already exists; remove or rename it", name, scope))
		default:
			if cerr := e.Backend.CreateOverlayNetwork(ctx, name); cerr != nil {
				reasons = append(reasons, fmt.Sprintf("  %s: auto-create failed: %v", name, cerr))
				needCreate = append(needCreate, name)
			}
		}
	}
	if len(reasons) == 0 {
		return nil
	}
	msg := "external network(s) required by this chart are unavailable:\n" + strings.Join(reasons, "\n")
	if len(needCreate) > 0 {
		var cmds strings.Builder
		for _, name := range needCreate {
			fmt.Fprintf(&cmds, "  docker network create --driver overlay --attachable %s\n", name)
		}
		msg += "\ncreate them on a swarm manager, then retry:\n" + strings.TrimRight(cmds.String(), "\n")
	}
	return fmt.Errorf("%s", msg)
}

// Uninstall removes the release's stack and, unless keepHistory, its recorded
// revisions. Volumes are retained unless purgeVolumes is set.
func (e *Engine) Uninstall(ctx context.Context, release string, purgeVolumes bool) error {
	revs, err := e.revisions(ctx, release)
	if err != nil {
		return err
	}
	if len(revs) == 0 {
		return fmt.Errorf("release %q not found", release)
	}

	if err := e.Backend.RemoveStack(release); err != nil {
		return fmt.Errorf("removing stack: %w", err)
	}

	if purgeVolumes {
		vols, err := e.Backend.StackVolumes(ctx, release)
		if err != nil {
			return fmt.Errorf("listing volumes: %w", err)
		}
		for _, v := range vols {
			if err := e.Backend.RemoveVolume(ctx, v); err != nil {
				return fmt.Errorf("removing volume %q: %w", v, err)
			}
		}
	}

	for _, r := range revs {
		if err := e.Backend.DeleteConfig(ctx, releaseConfigName(release, r.Revision)); err != nil {
			return fmt.Errorf("deleting release config: %w", err)
		}
	}
	return nil
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

// record stores a release revision as a gzipped, labeled Docker Config.
func (e *Engine) record(ctx context.Context, rel *Release) error {
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

// pruneHistory deletes the oldest revisions beyond keep (best-effort).
func (e *Engine) pruneHistory(ctx context.Context, release string, keep int) {
	revs, err := e.revisions(ctx, release)
	if err != nil || len(revs) <= keep {
		return
	}
	for _, r := range revs[:len(revs)-keep] {
		_ = e.Backend.DeleteConfig(ctx, releaseConfigName(release, r.Revision))
	}
}

// waitReady polls service convergence until all replicated services report
// their desired replica count, or the timeout elapses.
func (e *Engine) waitReady(release string, timeout time.Duration) error {
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	deadline := e.now().Add(timeout)
	for {
		states := e.Backend.StackServices(release)
		if len(states) > 0 && allConverged(states) {
			return nil
		}
		if !e.now().Before(deadline) {
			return fmt.Errorf("timed out after %s waiting for release %q to converge", timeout, release)
		}
		time.Sleep(2 * time.Second)
	}
}

func allConverged(states []ServiceState) bool {
	for _, s := range states {
		if s.Replicas == "" {
			continue // global services: no replica ratio to check
		}
		run, des, ok := strings.Cut(s.Replicas, "/")
		if !ok || run != des {
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
	raw, err := io.ReadAll(io.LimitReader(r, maxChartFileSize))
	if err != nil {
		return nil, err
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
