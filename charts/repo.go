// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package charts

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
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

// maxChartArchiveSize bounds a chart tarball download. LoadChartArchive bounds
// the decompressed size; this bounds the compressed transfer, so the whole body
// can be buffered and hashed before any of it reaches the archive parser.
const maxChartArchiveSize = 20 << 20 // 20 MiB

// AllowPlaintextEnv is the environment variable a host program may honour to set
// AllowPlaintext. The name lives here so the refusal message and whatever reads
// it cannot drift, but this package never reads the environment itself: whether
// an operator is allowed to opt out is the embedder's call, and cli's answer
// (yes, for an interactive user's own machine) is not automatically a daemon's.
const AllowPlaintextEnv = "SWARMCLI_CHARTS_ALLOW_PLAINTEXT"

// RepoStore persists configured repositories and caches their indexes under a
// base directory (default: the XDG state dir, ~/.local/state/swarmcli/charts).
type RepoStore struct {
	dir    string
	client *http.Client

	// Warnf, when set, receives non-fatal diagnostics: a chart whose index entry
	// publishes no digest, so its integrity could not be verified, and a
	// repository whose index could not be refreshed, so a cached one is in use.
	// charts is a library with no output of its own; nil is silent, and cli wires
	// this to stderr.
	Warnf func(format string, a ...any)

	// AllowPlaintext permits repositories reached over plain http. It defaults to
	// false because a chart repository serves the tarball that *becomes* the
	// deployed workload, so whoever sits on the path to it chooses what runs on
	// the swarm. The published digest does not close that hole: it travels in the
	// same index.yaml over the same channel, so an on-path attacker rewrites both
	// — and an entry that publishes no digest only warns. swarmcli-cd refuses
	// plaintext git remotes for exactly this reason; the same argument applies one
	// layer down, to where the chart itself comes from.
	//
	// Set it only for an internal registry on a network you already trust. cli
	// wires it to AllowPlaintextEnv so an existing plaintext setup keeps working
	// with one deliberate, greppable opt-out rather than none.
	AllowPlaintext bool
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

// indexFile is where a repository's cached index lives. It validates rather than
// trusting its caller because names arrive here from repos.json as well as from
// the command line, and this process is not that file's only possible author.
//
// The concatenation is the sharp edge: the name is glued on *after* "index-", so
// "index-.." is an ordinary path segment and filepath.Join's Clean has nothing to
// collapse. A traversing name is not neutralised, it merely costs one extra "../"
// — and what then lands outside the cache directory is whatever the repository
// served for index.yaml.
func (s *RepoStore) indexFile(name string) (string, error) {
	if err := validateRepoName(name); err != nil {
		return "", err
	}
	return filepath.Join(s.dir, "cache", "index-"+name+".yaml"), nil
}

// validateRepoName constrains a repository name to the charset a release name
// uses. The name is not just a label: it is a path component of the cached index
// file (see indexFile) and the part before the "/" in a "<repo>/<chart>"
// reference, so anything exotic in it is a bug at best.
func validateRepoName(name string) error {
	switch {
	case name == "":
		return fmt.Errorf("repository name is required")
	case !isPlainName(name):
		return fmt.Errorf("invalid repository name %q: use letters, digits, '-', '_', '.'", name)
	case name == "." || name == "..":
		// Harmless today only because indexFile prefixes the name, which leaves
		// "index-.." rather than "..". Refusing them keeps that accident from
		// quietly becoming load-bearing.
		return fmt.Errorf("invalid repository name %q", name)
	}
	return nil
}

// checkRepoURL rejects a URL this store must not fetch an index from: anything
// that is not an absolute http(s) URL, and plain http unless AllowPlaintext.
func (s *RepoStore) checkRepoURL(repoURL string) error {
	u, err := url.ParseRequestURI(repoURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return fmt.Errorf("repository URL must be an absolute http(s) URL")
	}
	return s.checkPlaintext(u)
}

// checkPlaintext refuses a plain-http URL unless the caller opted in; see
// AllowPlaintext for why the default is a refusal. It echoes the URL redacted,
// as checkRedirect does: a repository URL may carry credentials, and this error
// goes to a terminal or a CI log.
func (s *RepoStore) checkPlaintext(u *url.URL) error {
	if u.Scheme != "http" || s.AllowPlaintext {
		return nil
	}
	return fmt.Errorf("refusing the plaintext URL %q: anything on the path to it decides what gets "+
		"deployed on your swarm, and the index digest travels the same path; set %s=1 if this is an "+
		"internal registry on a network you already trust", u.Redacted(), AllowPlaintextEnv)
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
// names, names it will not build a cache path from, and URLs it will not fetch
// from — all before the download, so a bad request is reported as one.
func (s *RepoStore) Add(name, repoURL string) error {
	if err := validateRepoName(name); err != nil {
		return err
	}
	if err := s.checkRepoURL(repoURL); err != nil {
		return err
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
	path, err := s.indexFile(name)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, idx, 0o644); err != nil {
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
	// Best-effort, and deliberately skipped for a name indexFile will not vouch
	// for: the entry is gone from repos.json either way, and a path we refused to
	// build is not one to hand to os.Remove.
	if path, perr := s.indexFile(name); perr == nil {
		_ = os.Remove(path)
	}
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
		// Before the network round trip: a name repos.json should never have held
		// is not one to fetch an index for, let alone write one under.
		path, err := s.indexFile(r.Name)
		if err != nil {
			return changed, unchanged, fmt.Errorf("update %q: %w", r.Name, err)
		}
		idx, err := s.fetchIndex(r.URL)
		if err != nil {
			return changed, unchanged, fmt.Errorf("update %q: %w", r.Name, err)
		}
		// The served index is byte-stable between releases, so an identical
		// payload means nothing new — report it as already up-to-date.
		existing, _ := os.ReadFile(path)
		if existing != nil && bytes.Equal(existing, idx) {
			unchanged = append(unchanged, r.Name)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return changed, unchanged, err
		}
		if err := os.WriteFile(path, idx, 0o644); err != nil {
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
	path, err := s.indexFile(name)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
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

// Indexes returns the cached index of every configured repository, keyed by
// repository name. Repositories with no cached index are skipped, as in Search.
func (s *RepoStore) Indexes() (map[string]*Index, error) {
	repos, err := s.List()
	if err != nil {
		return nil, err
	}
	out := make(map[string]*Index, len(repos))
	for _, r := range repos {
		idx, err := s.LoadIndex(r.Name)
		if err != nil {
			continue
		}
		out[r.Name] = idx
	}
	return out, nil
}

// EnsureRepos makes the repositories a release manifest declares available,
// adding those that are absent and refreshing the rest. It exists so that
// `charts apply -f file` is the only command a CI job needs to run.
//
// What it writes is a name-to-URL mapping plus a cached index — a cache, not user
// data. The one thing it will not do is silently repoint an existing repository
// at a different origin.
func (s *RepoStore) EnsureRepos(specs []RepoSpec) error {
	if len(specs) == 0 {
		return nil
	}
	repos, err := s.List()
	if err != nil {
		return err
	}
	existing := make(map[string]string, len(repos))
	for _, r := range repos {
		existing[r.Name] = r.URL
	}

	for _, spec := range specs {
		want := strings.TrimRight(spec.URL, "/")
		have, ok := existing[spec.Name]
		switch {
		case !ok:
			if err := s.Add(spec.Name, want); err != nil {
				return err
			}
		case have != want:
			return fmt.Errorf("repository %q is already configured with a different URL (%s); "+
				"refusing to repoint it — run `swarmcli charts repo remove %s` first if that is intended",
				spec.Name, have, spec.Name)
		default:
			// Refreshing is best-effort. Every version in the manifest is pinned,
			// so a stale cache can only fail to resolve a chart — it can never
			// resolve one to the wrong version. Failing the whole apply because
			// the network blipped would be worse than proceeding offline.
			if _, _, err := s.Update(spec.Name); err != nil {
				s.warnf("could not refresh repository %q (%v); using the cached index\n", spec.Name, err)
			}
		}
	}
	return nil
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
	// The index may point the tarball anywhere — a CDN, a release asset — so the
	// scheme has to be checked here too, not only where the repository was added.
	if err := s.checkPlaintext(u); err != nil {
		return nil, err
	}
	resp, err := s.client.Get(tarURL)
	if err != nil {
		return nil, fmt.Errorf("download chart: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("download chart: HTTP %s", resp.Status)
	}
	// Buffer the whole archive before parsing it: the digest must be checked
	// against the complete body, and a streaming parser would have consumed
	// unverified bytes by the time the hash was known.
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxChartArchiveSize+1))
	if err != nil {
		return nil, fmt.Errorf("download chart: %w", err)
	}
	if len(body) > maxChartArchiveSize {
		return nil, fmt.Errorf("chart archive exceeds %d bytes", maxChartArchiveSize)
	}
	if err := verifyDigest(entry, body); err != nil {
		if errors.Is(err, errNoDigest) {
			s.warnf("chart %q version %s: repository index publishes no digest; integrity not verified\n",
				entry.Name, entry.Version)
		} else {
			return nil, err
		}
	}
	return LoadChartArchive(bytes.NewReader(body))
}

// errNoDigest reports an index entry that carries no digest, so the archive
// cannot be verified. It is a warning rather than an error: nothing was verified
// before this check existed, and failing closed would break every repository —
// including hand-written and older ones — that does not publish digests.
var errNoDigest = errors.New("no digest in index entry")

// staleIndexHint is appended to a digest-mismatch error: the common cause is a
// chart rebuilt and republished after the local index was cached, so refreshing
// the index makes the two agree again.
const staleIndexHint = "\nhint: the cached repository index may be stale (e.g. the chart was " +
	"rebuilt since you last refreshed); run `swarmcli charts repo update` to update it"

// verifyDigest checks a downloaded chart archive against the digest published in
// the repository index. The index is fetched over HTTPS from the repository, but
// the tarball URL may point anywhere (a GitHub Release asset, a CDN) — the digest
// is what binds the two together, so a mismatch is always fatal.
func verifyDigest(entry IndexEntry, body []byte) error {
	want := entry.Digest
	if want == "" {
		return errNoDigest
	}
	// Helm's `repo index` writes a bare hex sha256; this repo's generated index
	// writes the "sha256:" prefixed form. Accept both, but reject any algorithm
	// we cannot actually check rather than silently skipping verification.
	if alg, hexsum, ok := strings.Cut(want, ":"); ok {
		if alg != "sha256" {
			return fmt.Errorf("chart %q version %s: unsupported digest algorithm %q",
				entry.Name, entry.Version, alg)
		}
		want = hexsum
	}
	sum := sha256.Sum256(body)
	got := hex.EncodeToString(sum[:])
	if !strings.EqualFold(got, want) {
		return fmt.Errorf("chart %q version %s: digest mismatch (index says %s, download is %s); "+
			"the repository index and the chart archive disagree — refusing to install"+staleIndexHint,
			entry.Name, entry.Version, want, got)
	}
	return nil
}

func (s *RepoStore) warnf(format string, a ...any) {
	if s.Warnf != nil {
		s.Warnf(format, a...)
	}
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
	// Add checks this before anything else, to fail without a round trip; the
	// re-check is for Update, whose URLs come out of repos.json — including ones
	// added before this store had an opinion about plain http.
	if err := s.checkRepoURL(repoURL); err != nil {
		return nil, err
	}
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
