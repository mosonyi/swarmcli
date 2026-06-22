// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package charts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// httpTimeout bounds index and tarball downloads.
const httpTimeout = 30 * time.Second

// maxIndexSize bounds an index.yaml download (defense against huge responses).
const maxIndexSize = 16 << 20 // 16 MiB

// RepoStore persists configured repositories and caches their indexes under a
// base directory (default: the XDG state dir, ~/.local/state/swarmcli/charts).
type RepoStore struct {
	dir    string
	client *http.Client
}

// NewRepoStore returns a store rooted at the standard charts state directory.
func NewRepoStore() (*RepoStore, error) {
	dir, err := chartsStateDir()
	if err != nil {
		return nil, err
	}
	return NewRepoStoreAt(dir), nil
}

// NewRepoStoreAt returns a store rooted at dir (used in tests).
func NewRepoStoreAt(dir string) *RepoStore {
	return &RepoStore{dir: dir, client: &http.Client{Timeout: httpTimeout, CheckRedirect: checkRedirect}}
}

// checkRedirect bounds redirects and refuses any hop to a non-http(s) URL,
// blocking scheme-downgrade tricks (e.g. file://) when following a chart or
// index download to an attacker-influenced location.
func checkRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= 10 {
		return fmt.Errorf("stopped after 10 redirects")
	}
	if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
		return fmt.Errorf("refusing redirect to non-http(s) URL %q", req.URL.Redacted())
	}
	return nil
}

func (s *RepoStore) reposFile() string { return filepath.Join(s.dir, "repos.json") }
func (s *RepoStore) indexFile(name string) string {
	return filepath.Join(s.dir, "cache", "index-"+name+".yaml")
}

// List returns the configured repositories sorted by name.
func (s *RepoStore) List() ([]RepoEntry, error) {
	data, err := os.ReadFile(s.reposFile())
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read repos: %w", err)
	}
	var repos []RepoEntry
	if err := json.Unmarshal(data, &repos); err != nil {
		return nil, fmt.Errorf("parse repos: %w", err)
	}
	sort.Slice(repos, func(i, j int) bool { return repos[i].Name < repos[j].Name })
	return repos, nil
}

// Add registers a repository and downloads its index. It rejects duplicate
// names and invalid URLs.
func (s *RepoStore) Add(name, repoURL string) error {
	if name == "" {
		return fmt.Errorf("repository name is required")
	}
	if u, err := url.ParseRequestURI(repoURL); err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return fmt.Errorf("repository URL must be an absolute http(s) URL")
	}
	repos, err := s.List()
	if err != nil {
		return err
	}
	for _, r := range repos {
		if r.Name == name {
			return fmt.Errorf("repository %q already exists", name)
		}
	}
	repoURL = strings.TrimRight(repoURL, "/")
	// Download and validate the index before persisting anything, so a failed
	// download leaves no half-added repository behind.
	idx, err := s.fetchIndex(repoURL)
	if err != nil {
		return fmt.Errorf("index download failed, repository not added: %w%s", err, githubPagesHint(repoURL))
	}
	if err := os.MkdirAll(filepath.Dir(s.indexFile(name)), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(s.indexFile(name), idx, 0o644); err != nil {
		return err
	}
	repos = append(repos, RepoEntry{Name: name, URL: repoURL})
	return s.save(repos)
}

// Remove deletes a repository and its cached index.
func (s *RepoStore) Remove(name string) error {
	repos, err := s.List()
	if err != nil {
		return err
	}
	out := repos[:0]
	found := false
	for _, r := range repos {
		if r.Name == name {
			found = true
			continue
		}
		out = append(out, r)
	}
	if !found {
		return fmt.Errorf("repository %q not found", name)
	}
	if err := s.save(out); err != nil {
		return err
	}
	_ = os.Remove(s.indexFile(name))
	return nil
}

// Update downloads and caches the index for one repository (or all when name is
// empty) and returns the names refreshed.
func (s *RepoStore) Update(name string) (changed, unchanged []string, err error) {
	repos, err := s.List()
	if err != nil {
		return nil, nil, err
	}
	for _, r := range repos {
		if name != "" && r.Name != name {
			continue
		}
		idx, err := s.fetchIndex(r.URL)
		if err != nil {
			return changed, unchanged, fmt.Errorf("update %q: %w", r.Name, err)
		}
		// The served index is byte-stable between releases, so an identical
		// payload means nothing new — report it as already up-to-date.
		existing, _ := os.ReadFile(s.indexFile(r.Name))
		if existing != nil && bytes.Equal(existing, idx) {
			unchanged = append(unchanged, r.Name)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(s.indexFile(r.Name)), 0o755); err != nil {
			return changed, unchanged, err
		}
		if err := os.WriteFile(s.indexFile(r.Name), idx, 0o644); err != nil {
			return changed, unchanged, err
		}
		changed = append(changed, r.Name)
	}
	if name != "" && len(changed)+len(unchanged) == 0 {
		return nil, nil, fmt.Errorf("repository %q not found", name)
	}
	return changed, unchanged, nil
}

// LoadIndex returns the cached, parsed index for a repository.
func (s *RepoStore) LoadIndex(name string) (*Index, error) {
	data, err := os.ReadFile(s.indexFile(name))
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("no cached index for %q; run `charts repo update`", name)
	}
	if err != nil {
		return nil, err
	}
	var idx Index
	if err := yaml.Unmarshal(data, &idx); err != nil {
		return nil, fmt.Errorf("parse index for %q: %w", name, err)
	}
	return &idx, nil
}

// ChartHit is a search result: one chart version in a repository.
type ChartHit struct {
	Repo  string
	Entry IndexEntry
}

// Search returns the latest version of every chart across all repos whose name
// or description contains keyword (case-insensitive). An empty keyword lists
// all charts. Results are sorted by "repo/name".
func (s *RepoStore) Search(keyword string) ([]ChartHit, error) {
	repos, err := s.List()
	if err != nil {
		return nil, err
	}
	kw := strings.ToLower(keyword)
	var hits []ChartHit
	for _, r := range repos {
		idx, err := s.LoadIndex(r.Name)
		if err != nil {
			continue // skip repos without a cached index
		}
		for chartName, versions := range idx.Entries {
			if len(versions) == 0 {
				continue
			}
			latest := latestVersion(versions)
			if kw != "" && !strings.Contains(strings.ToLower(chartName), kw) &&
				!strings.Contains(strings.ToLower(latest.Description), kw) {
				continue
			}
			hits = append(hits, ChartHit{Repo: r.Name, Entry: latest})
		}
	}
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].Repo != hits[j].Repo {
			return hits[i].Repo < hits[j].Repo
		}
		return hits[i].Entry.Name < hits[j].Entry.Name
	})
	return hits, nil
}

// Resolve looks up a "repo/chart" reference and returns the chosen index entry
// (the requested version, or the latest when version is empty) plus the repo's
// base URL for resolving relative tarball URLs.
func (s *RepoStore) Resolve(ref, version string) (IndexEntry, string, error) {
	repoName, chartName, ok := strings.Cut(ref, "/")
	if !ok || repoName == "" || chartName == "" {
		return IndexEntry{}, "", fmt.Errorf("chart reference must be <repo>/<chart>, got %q", ref)
	}
	repos, err := s.List()
	if err != nil {
		return IndexEntry{}, "", err
	}
	var baseURL string
	for _, r := range repos {
		if r.Name == repoName {
			baseURL = r.URL
		}
	}
	if baseURL == "" {
		return IndexEntry{}, "", fmt.Errorf("repository %q not found", repoName)
	}
	idx, err := s.LoadIndex(repoName)
	if err != nil {
		return IndexEntry{}, "", err
	}
	versions := idx.Entries[chartName]
	if len(versions) == 0 {
		return IndexEntry{}, "", fmt.Errorf("chart %q not found in repository %q", chartName, repoName)
	}
	if version == "" {
		return latestVersion(versions), baseURL, nil
	}
	for _, v := range versions {
		// SemVer-aware match: a chart version is plain SemVer ("0.1.3"), but the
		// release git tag carries a "v" prefix ("v0.1.3"), so users naturally type
		// either. compareVersions normalizes both and falls back to exact string
		// equality for non-SemVer versions.
		if compareVersions(v.Version, version) == 0 {
			return v, baseURL, nil
		}
	}
	return IndexEntry{}, "", fmt.Errorf("chart %q version %q not found", chartName, version)
}

// Pull downloads and loads the chart described by entry, resolving relative URLs
// against baseURL.
func (s *RepoStore) Pull(entry IndexEntry, baseURL string) (*Chart, error) {
	if len(entry.URLs) == 0 {
		return nil, fmt.Errorf("chart %q has no download URL", entry.Name)
	}
	tarURL := entry.URLs[0]
	u, err := url.Parse(tarURL)
	if err != nil {
		return nil, fmt.Errorf("invalid chart download URL %q: %w", tarURL, err)
	}
	if !u.IsAbs() {
		tarURL = strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(tarURL, "/")
		if u, err = url.Parse(tarURL); err != nil {
			return nil, fmt.Errorf("invalid chart download URL %q: %w", tarURL, err)
		}
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("chart download URL must be http(s), got %q", tarURL)
	}
	resp, err := s.client.Get(tarURL)
	if err != nil {
		return nil, fmt.Errorf("download chart: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("download chart: HTTP %s", resp.Status)
	}
	return LoadChartArchive(resp.Body)
}

// githubPagesHint suggests the GitHub Pages URL when a user points the store at
// a github.com repository page, which serves an HTML view rather than a chart
// index. Returns "" for any other host.
func githubPagesHint(repoURL string) string {
	u, err := url.Parse(repoURL)
	if err != nil || (u.Host != "github.com" && u.Host != "www.github.com") {
		return ""
	}
	const base = "\nhint: github.com serves repository pages, not chart indexes; use the GitHub Pages URL"
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return base + " (https://<org>.github.io/<repo>)"
	}
	return fmt.Sprintf("%s https://%s.github.io/%s", base, strings.ToLower(parts[0]), parts[1])
}

func (s *RepoStore) fetchIndex(repoURL string) ([]byte, error) {
	indexURL := strings.TrimRight(repoURL, "/") + "/index.yaml"
	resp, err := s.client.Get(indexURL)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", indexURL, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("fetch %s: HTTP %s", indexURL, resp.Status)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxIndexSize+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxIndexSize {
		return nil, fmt.Errorf("index.yaml exceeds %d bytes", maxIndexSize)
	}
	var probe Index
	if err := yaml.Unmarshal(data, &probe); err != nil {
		return nil, fmt.Errorf("index.yaml is not valid: %w", err)
	}
	return data, nil
}

func (s *RepoStore) save(repos []RepoEntry) error {
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(repos, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.reposFile(), data, 0o644)
}

// latestVersion returns the entry with the highest SemVer; on parse ambiguity it
// falls back to lexical ordering.
func latestVersion(versions []IndexEntry) IndexEntry {
	best := versions[0]
	for _, v := range versions[1:] {
		if compareVersions(v.Version, best.Version) > 0 {
			best = v
		}
	}
	return best
}

// chartsStateDir resolves the base directory for charts state, mirroring the
// logger's XDG-with-fallback strategy so all swarmcli state lives together.
func chartsStateDir() (string, error) {
	const appName = "swarmcli"
	if xdg := os.Getenv("XDG_STATE_HOME"); xdg != "" {
		return filepath.Join(xdg, appName, "charts"), nil
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".local", "state", appName, "charts"), nil
	}
	return filepath.Join(os.TempDir(), appName, "charts"), nil
}
