// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package charts

import (
	"fmt"
	"os"
	"strings"
)

// ChartSource resolves a chart reference to a loaded chart. It is the seam that
// lets release planning be unit-tested without a repository, a network or a
// filesystem: the whole of Engine.PlanApply depends on this interface and not on
// RepoStore.
type ChartSource interface {
	// Load returns the chart named by ref. ref is either a local path (a chart
	// directory or a .tgz) or a "<repo>/<chart>" reference resolved through the
	// configured repositories. version selects a repository chart version; it is
	// meaningless for a local path and rejected there rather than ignored.
	Load(ref, version string) (*Chart, error)
}

// NewChartSource returns the standard source, backed by the configured chart
// repositories for "<repo>/<chart>" references.
func NewChartSource(store *RepoStore) ChartSource { return &repoSource{store: store} }

type repoSource struct{ store *RepoStore }

func (s *repoSource) Load(ref, version string) (*Chart, error) {
	if IsPathRef(ref) {
		if version != "" {
			// Previously the flag was accepted and silently dropped here, so
			// `install foo ./chart --version 2.0.0` quietly installed whatever
			// Chart.yaml said. A local directory has exactly one version.
			return nil, fmt.Errorf("chart %q is a local path: --version does not apply (the chart's own Chart.yaml sets the version)", ref)
		}
		return loadLocalChart(ref)
	}
	// Not syntactically a path, but it might still be a bare directory name
	// ("./" omitted). Keep resolving those for backwards compatibility.
	if info, err := os.Stat(ref); err == nil {
		if version != "" {
			return nil, fmt.Errorf("chart %q is a local path: --version does not apply (the chart's own Chart.yaml sets the version)", ref)
		}
		return loadStatted(ref, info.IsDir())
	}
	if s.store == nil {
		return nil, fmt.Errorf("chart %q not found on disk and no repositories are configured", ref)
	}
	entry, base, err := s.store.Resolve(ref, version)
	if err != nil {
		return nil, err
	}
	return s.store.Pull(entry, base)
}

// IsPathRef reports whether ref names a local chart path rather than a
// "<repo>/<chart>" reference. The test is deliberately SYNTACTIC: a release file
// is committed to git and must resolve the same way on every machine, so whether
// a reference is a path cannot depend on what happens to exist on the disk of
// whichever CI runner picked up the job.
func IsPathRef(ref string) bool {
	return strings.HasPrefix(ref, "./") ||
		strings.HasPrefix(ref, "../") ||
		strings.HasPrefix(ref, "/") ||
		strings.HasPrefix(ref, "~")
}

func loadLocalChart(path string) (*Chart, error) {
	info, err := os.Stat(path)
	if err != nil {
		// An explicit path that does not exist is a typo, not a repository
		// reference. Say so, instead of the misleading "must be <repo>/<chart>".
		return nil, fmt.Errorf("chart path %q not found", path)
	}
	return loadStatted(path, info.IsDir())
}

func loadStatted(path string, isDir bool) (*Chart, error) {
	if isDir {
		return LoadChartDir(path)
	}
	fh, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = fh.Close() }()
	return LoadChartArchive(fh)
}

// ReleaseChartOf projects a loaded chart into the metadata recorded on a release.
func ReleaseChartOf(ch *Chart) ReleaseChart {
	return ReleaseChart{Name: ch.Metadata.Name, Version: ch.Metadata.Version, AppVersion: ch.Metadata.AppVersion}
}
