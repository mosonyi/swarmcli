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

// InstallOptions tune an install.
type InstallOptions struct {
	DryRun     bool
	Wait       bool
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

	rel := &Release{
		Name:      release,
		Revision:  nextRevision(revs),
		Status:    StatusDeployed,
		Chart:     chart,
		Values:    values,
		Manifest:  manifest,
		Namespace: release,
		Created:   e.now().UTC().Format(time.RFC3339),
	}

	if opts.DryRun {
		return rel, nil
	}

	if err := e.Backend.DeployStack(release, manifest); err != nil {
		rel.Status = StatusFailed
		// Best-effort: record the failed revision for auditability.
		_ = e.record(ctx, rel)
		return rel, fmt.Errorf("deploy failed: %w", err)
	}
	if err := e.record(ctx, rel); err != nil {
		return rel, fmt.Errorf("deployed, but recording release failed: %w", err)
	}
	if opts.Wait {
		if err := e.waitReady(release, opts.Timeout); err != nil {
			return rel, err
		}
	}
	if opts.HistoryMax > 0 {
		e.pruneHistory(ctx, release, opts.HistoryMax)
	}
	return rel, nil
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
