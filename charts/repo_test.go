// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package charts

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// newTestStore returns a store in a fresh directory that accepts the plain-http
// URL an httptest server hands out. Chart repositories are https-only by default
// (see RepoStore.AllowPlaintext); the tests below are about everything else, and
// the gate itself is covered by TestRepoStoreRefusesPlaintextURL.
func newTestStore(t *testing.T) *RepoStore {
	t.Helper()
	s := NewRepoStoreAt(t.TempDir())
	s.AllowPlaintext = true
	return s
}

func TestRepoStoreAddListRemove(t *testing.T) {
	idx := `apiVersion: v1
entries:
  demo:
    - name: demo
      version: 0.1.0
      description: demo chart
      urls: ["demo-0.1.0.tgz"]
    - name: demo
      version: 0.2.0
      description: demo chart
      urls: ["demo-0.2.0.tgz"]
`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/index.yaml") {
			_, _ = w.Write([]byte(idx))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	s := newTestStore(t)
	require.NoError(t, s.Add("eldara", srv.URL))

	// duplicate rejected
	require.Error(t, s.Add("eldara", srv.URL))

	repos, err := s.List()
	require.NoError(t, err)
	require.Len(t, repos, 1)
	require.Equal(t, "eldara", repos[0].Name)

	// search finds the latest version only
	hits, err := s.Search("demo")
	require.NoError(t, err)
	require.Len(t, hits, 1)
	require.Equal(t, "0.2.0", hits[0].Entry.Version)

	// resolve latest + specific
	e, base, err := s.Resolve("eldara/demo", "")
	require.NoError(t, err)
	require.Equal(t, "0.2.0", e.Version)
	require.Equal(t, srv.URL, base)

	e, _, err = s.Resolve("eldara/demo", "0.1.0")
	require.NoError(t, err)
	require.Equal(t, "0.1.0", e.Version)

	_, _, err = s.Resolve("eldara/demo", "9.9.9")
	require.Error(t, err)

	require.NoError(t, s.Remove("eldara"))
	require.Error(t, s.Remove("eldara")) // gone
}

// Resolve must accept both the plain SemVer chart version ("0.1.3") and the
// "v"-prefixed form users copy from the release git tag ("v0.1.3").
func TestResolveVersionPrefixNormalization(t *testing.T) {
	idx := `apiVersion: v1
entries:
  whoami:
    - name: whoami
      version: 0.1.3
      description: demo chart
      urls: ["whoami-v0.1.3.tgz"]
`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/index.yaml") {
			_, _ = w.Write([]byte(idx))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	s := newTestStore(t)
	require.NoError(t, s.Add("eldara", srv.URL))

	for _, want := range []string{"0.1.3", "v0.1.3"} {
		e, _, err := s.Resolve("eldara/whoami", want)
		require.NoErrorf(t, err, "version %q should resolve", want)
		require.Equal(t, "0.1.3", e.Version)
	}

	// A genuinely absent version still errors, prefixed or not.
	_, _, err := s.Resolve("eldara/whoami", "v9.9.9")
	require.Error(t, err)
}

// Update reports a repo as unchanged when the served index is byte-identical to
// the cached copy, and as changed once the served content differs.
func TestUpdateReportsChangedVsUnchanged(t *testing.T) {
	body := `apiVersion: v1
entries:
  demo:
    - {name: demo, version: 0.1.0, urls: ["demo-0.1.0.tgz"]}
`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/index.yaml") {
			_, _ = w.Write([]byte(body))
		} else {
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	s := newTestStore(t)
	require.NoError(t, s.Add("eldara", srv.URL)) // Add caches the index

	// Nothing changed since Add → already up-to-date.
	changed, unchanged, err := s.Update("eldara")
	require.NoError(t, err)
	require.Empty(t, changed)
	require.Equal(t, []string{"eldara"}, unchanged)

	// Serve different content → reported as changed.
	body = `apiVersion: v1
entries:
  demo:
    - {name: demo, version: 0.2.0, urls: ["demo-0.2.0.tgz"]}
`
	changed, unchanged, err = s.Update("eldara")
	require.NoError(t, err)
	require.Equal(t, []string{"eldara"}, changed)
	require.Empty(t, unchanged)
}

func TestRepoStoreAddRejectsBadURL(t *testing.T) {
	s := newTestStore(t)
	require.Error(t, s.Add("x", "not-a-url"))
	require.Error(t, s.Add("x", "ftp://example.com"))
}

// A repository name is a path component of its cached index, and indexFile glues
// it on *after* "index-" — so "index-.." is an ordinary segment that Clean has
// nothing to collapse, and a traversing name escapes for the price of one extra
// "../". These assertions are about where the bytes landed, not only about the
// error returned: containment is the property under test.
func TestRepoStoreContainsTraversingRepositoryName(t *testing.T) {
	body := "apiVersion: v1\nentries:\n  demo:\n    - {name: demo, version: 0.1.0, urls: [\"d.tgz\"]}\n"
	up := true
	srv := newIndexServer(t, &body, &up)

	// The store lives two directories below root, and the index would be written
	// a third down in "cache" — so four "../" reach root: one is eaten unescaping
	// the "index-.." segment, the other three walk back up.
	root := t.TempDir()
	s := NewRepoStoreAt(filepath.Join(root, "state"))
	s.AllowPlaintext = true
	const escape = "../../../../pwned"
	landing := filepath.Join(root, "pwned.yaml")

	require.ErrorContains(t, s.Add(escape, srv.URL), "invalid repository name")
	require.NoFileExists(t, landing)

	// Add settles the name before it fetches anything, so an unreachable
	// repository does not mask what is actually wrong with the request.
	up = false
	require.ErrorContains(t, s.Add(escape, srv.URL), "invalid repository name")
	up = true

	// The same name arriving from repos.json instead of the command line — an
	// older store, or one somebody hand-edited — must be refused just as hard.
	// That is why the check lives in indexFile and not only in Add.
	require.NoError(t, s.save([]RepoEntry{{Name: escape, URL: srv.URL}}))
	_, _, err := s.Update("")
	require.ErrorContains(t, err, "invalid repository name")
	require.NoFileExists(t, landing)

	_, err = s.LoadIndex(escape)
	require.ErrorContains(t, err, "invalid repository name")

	// And an ordinary name still lands exactly where it always did.
	good := newTestStore(t)
	require.NoError(t, good.Add("eldara", srv.URL))
	require.FileExists(t, filepath.Join(good.dir, "cache", "index-eldara.yaml"))
}

// Plain http is refused by default. A chart repository serves the tarball that
// *becomes* the deployed workload, so whoever is on the path chooses what runs —
// and the digest that would catch a swap is published in the same index, over the
// same channel. The refusal names its opt-out, because an internal registry on a
// trusted network is a setup that worked until this change.
func TestRepoStoreRefusesPlaintextURL(t *testing.T) {
	body := "apiVersion: v1\nentries: {}\n"
	up := true
	srv := newIndexServer(t, &body, &up)

	t.Run("add is refused and records nothing", func(t *testing.T) {
		s := NewRepoStoreAt(t.TempDir())
		err := s.Add("eldara", srv.URL)
		require.ErrorContains(t, err, "plaintext")
		require.ErrorContains(t, err, AllowPlaintextEnv)
		require.NotContains(t, err.Error(), "index download failed",
			"the scheme is settled before the fetch, so nothing may blame the download")

		repos, lerr := s.List()
		require.NoError(t, lerr)
		require.Empty(t, repos)
	})

	// Otherwise the default would only bind repositories nobody has added yet.
	t.Run("an already-configured http repository is refused too", func(t *testing.T) {
		s := NewRepoStoreAt(t.TempDir())
		require.NoError(t, s.save([]RepoEntry{{Name: "old", URL: srv.URL}}))
		_, _, err := s.Update("")
		require.ErrorContains(t, err, "plaintext")
	})

	// The index can point the tarball anywhere, so the repository's own scheme
	// does not settle the download's.
	t.Run("a plaintext tarball URL is refused", func(t *testing.T) {
		s := NewRepoStoreAt(t.TempDir())
		_, err := s.Pull(IndexEntry{Name: "demo", URLs: []string{"http://example.com/demo.tgz"}}, "https://example.com")
		require.ErrorContains(t, err, "plaintext")
	})

	t.Run("a malformed URL is still refused, and not as plaintext", func(t *testing.T) {
		s := NewRepoStoreAt(t.TempDir())
		err := s.Add("eldara", "not-a-url")
		require.ErrorContains(t, err, "absolute http(s)")
		require.NotContains(t, err.Error(), "plaintext")
		require.NotContains(t, err.Error(), "index download failed")
	})

	// https goes through. Only the test CA is added to the store's own client, so
	// its timeout and redirect policy still apply — this is the real path.
	t.Run("https is accepted", func(t *testing.T) {
		tsrv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasSuffix(r.URL.Path, "/index.yaml") {
				_, _ = w.Write([]byte(body))
				return
			}
			http.NotFound(w, r)
		}))
		defer tsrv.Close()

		s := NewRepoStoreAt(t.TempDir())
		s.client.Transport = tsrv.Client().Transport
		require.NoError(t, s.Add("secure", tsrv.URL))
	})

	t.Run("the opt-out restores an internal plaintext registry", func(t *testing.T) {
		s := newTestStore(t)
		require.NoError(t, s.Add("internal", srv.URL))
	})
}

// Pull must refuse a chart URL that is not http(s) (e.g. a malicious index
// pointing at file://), before any download happens.
func TestPullRejectsNonHTTPScheme(t *testing.T) {
	s := newTestStore(t)
	_, err := s.Pull(IndexEntry{Name: "demo", URLs: []string{"file:///etc/passwd"}}, "http://example.com")
	require.ErrorContains(t, err, "http(s)")
}

func TestUpdateUnknownRepo(t *testing.T) {
	s := newTestStore(t)
	_, _, err := s.Update("ghost")
	require.ErrorContains(t, err, "not found")
}

func TestLoadIndexMissing(t *testing.T) {
	s := newTestStore(t)
	_, err := s.LoadIndex("ghost")
	require.ErrorContains(t, err, "no cached index")
}

// Update over all repos surfaces a mid-loop fetch failure with the repo name.
func TestUpdateReportsFetchFailure(t *testing.T) {
	serve := true
	idx := "apiVersion: v1\nentries: {}\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if serve && strings.HasSuffix(r.URL.Path, "/index.yaml") {
			_, _ = w.Write([]byte(idx))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	s := newTestStore(t)
	require.NoError(t, s.Add("eldara", srv.URL))
	serve = false
	_, _, err := s.Update("") // all repos
	require.ErrorContains(t, err, `update 'eldara'`)
}

// A failed index download must not leave the repository half-added: List stays
// empty and the name can be reused once the index is reachable.
func TestRepoStoreAddIndexFailureDoesNotPersist(t *testing.T) {
	var serve bool
	idx := "apiVersion: v1\nentries: {}\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if serve && strings.HasSuffix(r.URL.Path, "/index.yaml") {
			_, _ = w.Write([]byte(idx))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	s := newTestStore(t)

	// index 404s → add fails and persists nothing
	require.Error(t, s.Add("eldara", srv.URL))
	repos, err := s.List()
	require.NoError(t, err)
	require.Empty(t, repos)

	// same name succeeds once the index is reachable
	serve = true
	require.NoError(t, s.Add("eldara", srv.URL))
	repos, err = s.List()
	require.NoError(t, err)
	require.Len(t, repos, 1)
}

// A github.com repository URL never serves a chart index; the hint should point
// the user at the lowercased GitHub Pages URL. Other hosts get no hint.
func TestGithubPagesHint(t *testing.T) {
	require.Contains(t,
		githubPagesHint("https://github.com/Eldara-Tech/swarmcli-charts"),
		"https://eldara-tech.github.io/swarmcli-charts")
	require.Contains(t, githubPagesHint("https://github.com/Eldara-Tech"), "<org>.github.io")
	require.Empty(t, githubPagesHint("https://eldara-tech.github.io/swarmcli-charts"))
	require.Empty(t, githubPagesHint("https://example.com/charts"))
}

func TestPullChart(t *testing.T) {
	tgz := packDirToTgz(t, "testdata/demo", "demo")
	idx := `apiVersion: v1
entries:
  demo:
    - name: demo
      version: 0.1.0
      urls: ["demo-0.1.0.tgz"]
`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/index.yaml"):
			_, _ = w.Write([]byte(idx))
		case strings.HasSuffix(r.URL.Path, "demo-0.1.0.tgz"):
			_, _ = w.Write([]byte(tgz))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	s := newTestStore(t)
	require.NoError(t, s.Add("eldara", srv.URL))
	e, base, err := s.Resolve("eldara/demo", "")
	require.NoError(t, err)
	ch, err := s.Pull(e, base)
	require.NoError(t, err)
	require.Equal(t, "demo", ch.Metadata.Name)
}

// serveChart stands up a repository whose index entry for demo-0.1.0 carries the
// given digest (verbatim — "" means the entry publishes none), and returns the
// store plus the resolved entry and base URL, ready to Pull.
func serveChart(t *testing.T, digest string, tgz []byte) (*RepoStore, IndexEntry, string) {
	t.Helper()
	digestLine := ""
	if digest != "" {
		digestLine = "\n      digest: " + digest
	}
	idx := `apiVersion: v1
entries:
  demo:
    - name: demo
      version: 0.1.0
      urls: ["demo-0.1.0.tgz"]` + digestLine + "\n"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/index.yaml"):
			_, _ = w.Write([]byte(idx))
		case strings.HasSuffix(r.URL.Path, "demo-0.1.0.tgz"):
			_, _ = w.Write(tgz)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	s := newTestStore(t)
	require.NoError(t, s.Add("eldara", srv.URL))
	e, base, err := s.Resolve("eldara/demo", "")
	require.NoError(t, err)
	return s, e, base
}

func TestPullVerifiesDigest(t *testing.T) {
	tgz := []byte(packDirToTgz(t, "testdata/demo", "demo"))
	sum := sha256.Sum256(tgz)
	good := hex.EncodeToString(sum[:])

	t.Run("prefixed digest matches", func(t *testing.T) {
		s, e, base := serveChart(t, "sha256:"+good, tgz)
		ch, err := s.Pull(e, base)
		require.NoError(t, err)
		require.Equal(t, "demo", ch.Metadata.Name)
	})

	// `helm repo index` writes a bare hex sum, with no algorithm prefix.
	t.Run("bare hex digest matches", func(t *testing.T) {
		s, e, base := serveChart(t, good, tgz)
		ch, err := s.Pull(e, base)
		require.NoError(t, err)
		require.Equal(t, "demo", ch.Metadata.Name)
	})

	// Hex is case-insensitive, and index generators differ. Without this case,
	// replacing strings.EqualFold with != still passes the suite — while rejecting
	// every repository that publishes uppercase hex.
	t.Run("uppercase hex digest matches", func(t *testing.T) {
		s, e, base := serveChart(t, "sha256:"+strings.ToUpper(good), tgz)
		ch, err := s.Pull(e, base)
		require.NoError(t, err)
		require.Equal(t, "demo", ch.Metadata.Name)
	})

	t.Run("mismatch is fatal", func(t *testing.T) {
		s, e, base := serveChart(t, "sha256:"+strings.Repeat("0", 64), tgz)
		ch, err := s.Pull(e, base)
		require.ErrorContains(t, err, "digest mismatch")
		require.ErrorContains(t, err, "charts repo update")
		require.Nil(t, ch)
	})

	// An algorithm we cannot check must fail closed rather than silently skip
	// verification.
	t.Run("unsupported algorithm is fatal", func(t *testing.T) {
		s, e, base := serveChart(t, "sha512:"+good, tgz)
		ch, err := s.Pull(e, base)
		require.ErrorContains(t, err, "unsupported digest algorithm")
		require.Nil(t, ch)
	})

	// Nothing was verified before this check existed, so an index that publishes
	// no digest still installs — loudly.
	t.Run("absent digest warns but installs", func(t *testing.T) {
		s, e, base := serveChart(t, "", tgz)
		var warnings []string
		s.Warnf = func(format string, a ...any) {
			warnings = append(warnings, fmt.Sprintf(format, a...))
		}
		ch, err := s.Pull(e, base)
		require.NoError(t, err)
		require.Equal(t, "demo", ch.Metadata.Name)
		require.Len(t, warnings, 1)
		require.Contains(t, warnings[0], "no digest")
	})
}

func TestPullRejectsOversizedArchive(t *testing.T) {
	s, e, base := serveChart(t, "", bytes.Repeat([]byte{0}, maxChartArchiveSize+1))
	_, err := s.Pull(e, base)
	require.ErrorContains(t, err, "exceeds")
}

func TestCompareVersions(t *testing.T) {
	require.Equal(t, 1, compareVersions("0.2.0", "0.1.0"))
	require.Equal(t, -1, compareVersions("1.0.0", "1.0.1"))
	require.Equal(t, 1, compareVersions("v3.5.0", "3.4.0"))
}

// --- EnsureRepos: the gate every `charts apply` passes through ---

// newIndexServer serves a mutable index and reports whether it is currently up.
func newIndexServer(t *testing.T, body *string, up *bool) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if *up && strings.HasSuffix(r.URL.Path, "/index.yaml") {
			_, _ = w.Write([]byte(*body))
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestEnsureReposAddsAbsentRepository(t *testing.T) {
	body := "apiVersion: v1\nentries:\n  demo:\n    - {name: demo, version: 0.1.0, urls: [\"d.tgz\"]}\n"
	up := true
	srv := newIndexServer(t, &body, &up)

	s := newTestStore(t)
	require.NoError(t, s.EnsureRepos([]RepoSpec{{Name: "eldara", URL: srv.URL}}))

	repos, err := s.List()
	require.NoError(t, err)
	require.Len(t, repos, 1)
	// The index must actually be cached, not merely the name recorded — apply
	// resolves against the cache immediately afterwards.
	idx, err := s.LoadIndex("eldara")
	require.NoError(t, err)
	require.Contains(t, idx.Entries, "demo")
}

// A trailing slash in the manifest's URL must take the refresh branch, not the
// "different URL" hard error. Add normalises with TrimRight, so without this the
// refusal would fire on every manifest whose url ends in "/".
func TestEnsureReposRefreshesSameRepositoryAndToleratesTrailingSlash(t *testing.T) {
	body := "apiVersion: v1\nentries:\n  demo:\n    - {name: demo, version: 0.1.0, urls: [\"d.tgz\"]}\n"
	up := true
	srv := newIndexServer(t, &body, &up)

	s := newTestStore(t)
	require.NoError(t, s.Add("eldara", srv.URL))

	body = "apiVersion: v1\nentries:\n  demo:\n    - {name: demo, version: 0.2.0, urls: [\"d.tgz\"]}\n"
	require.NoError(t, s.EnsureRepos([]RepoSpec{{Name: "eldara", URL: srv.URL + "/"}}))

	idx, err := s.LoadIndex("eldara")
	require.NoError(t, err)
	require.Equal(t, "0.2.0", idx.Entries["demo"][0].Version, "the index must have been refreshed")
}

// Never silently repoint an existing repository at a different origin: that would
// let a manifest swap out where every one of its charts comes from.
func TestEnsureReposRefusesToRepointADifferentURL(t *testing.T) {
	body := "apiVersion: v1\nentries: {}\n"
	up := true
	srvA := newIndexServer(t, &body, &up)
	srvB := newIndexServer(t, &body, &up)

	s := newTestStore(t)
	require.NoError(t, s.Add("eldara", srvA.URL))

	err := s.EnsureRepos([]RepoSpec{{Name: "eldara", URL: srvB.URL}})
	require.ErrorContains(t, err, "different URL")

	// The refusal is the point: assert the store was not repointed.
	repos, lerr := s.List()
	require.NoError(t, lerr)
	require.Equal(t, srvA.URL, repos[0].URL)
}

// A network blip must not fail the apply. Every version in a manifest is pinned,
// so a stale cache can only fail to resolve a chart — never resolve it to the
// wrong one. This is the "CI keeps working when the network wobbles" contract.
func TestEnsureReposWarnsAndUsesCacheWhenRefreshFails(t *testing.T) {
	body := "apiVersion: v1\nentries:\n  demo:\n    - {name: demo, version: 0.1.0, urls: [\"d.tgz\"]}\n"
	up := true
	srv := newIndexServer(t, &body, &up)

	s := newTestStore(t)
	require.NoError(t, s.Add("eldara", srv.URL))

	var warnings []string
	s.Warnf = func(format string, a ...any) { warnings = append(warnings, fmt.Sprintf(format, a...)) }

	up = false // the repository goes away
	require.NoError(t, s.EnsureRepos([]RepoSpec{{Name: "eldara", URL: srv.URL}}),
		"a failed refresh must not fail the apply")
	require.Len(t, warnings, 1)
	require.Contains(t, warnings[0], "could not refresh repository")

	// The cached index is still usable.
	idx, err := s.LoadIndex("eldara")
	require.NoError(t, err)
	require.Equal(t, "0.1.0", idx.Entries["demo"][0].Version)
}

func TestEnsureReposNoSpecsIsANoOp(t *testing.T) {
	s := newTestStore(t)
	require.NoError(t, s.EnsureRepos(nil))
	repos, err := s.List()
	require.NoError(t, err)
	require.Empty(t, repos)
}

// Indexes backs `charts outdated`, which must not blow up on a repository whose
// index was never fetched.
func TestIndexesSkipsRepositoriesWithNoCachedIndex(t *testing.T) {
	body := "apiVersion: v1\nentries:\n  demo:\n    - {name: demo, version: 0.1.0, urls: [\"d.tgz\"]}\n"
	up := true
	srv := newIndexServer(t, &body, &up)

	s := newTestStore(t)
	require.NoError(t, s.Add("good", srv.URL))

	// A repo recorded with no cached index (e.g. the cache file was cleared).
	require.NoError(t, s.save([]RepoEntry{{Name: "good", URL: srv.URL}, {Name: "stale", URL: srv.URL}}))

	idxs, err := s.Indexes()
	require.NoError(t, err)
	require.Len(t, idxs, 1)
	require.Contains(t, idxs, "good")
	require.NotContains(t, idxs, "stale")
}

// A repository in the manifest that cannot be reached must fail the apply loudly,
// not be silently skipped — every chart it provides would otherwise fail to
// resolve with a far more confusing error.
func TestEnsureReposFailsWhenAnAbsentRepositoryCannotBeAdded(t *testing.T) {
	body := "apiVersion: v1\nentries: {}\n"
	up := false
	srv := newIndexServer(t, &body, &up)

	s := newTestStore(t)
	err := s.EnsureRepos([]RepoSpec{{Name: "eldara", URL: srv.URL}})
	require.Error(t, err)

	repos, lerr := s.List()
	require.NoError(t, lerr)
	require.Empty(t, repos, "a failed add must persist nothing")
}

// age backdates a repository's cached index so it reads as unverified.
func age(t *testing.T, s *RepoStore, name string, d time.Duration) {
	t.Helper()
	path, err := s.indexFile(name)
	require.NoError(t, err)
	when := time.Now().Add(-d)
	require.NoError(t, os.Chtimes(path, when, when))
}

// Resolution reads the cache and never refetches, so a cache nobody has
// refreshed keeps answering "latest" with a version that stopped being it. Say
// so, or an install deploys the stale answer with nothing on screen to suggest
// there is a newer chart.
func TestResolveWarnsOnceTheCachedIndexGoesUnverified(t *testing.T) {
	body := "apiVersion: v1\nentries:\n  demo:\n    - {name: demo, version: 0.1.0, urls: [\"d.tgz\"]}\n"
	up := true
	srv := newIndexServer(t, &body, &up)

	var warnings []string
	s := newTestStore(t)
	s.Warnf = func(format string, a ...any) { warnings = append(warnings, fmt.Sprintf(format, a...)) }
	require.NoError(t, s.Add("eldara", srv.URL))

	// Just fetched: nothing to say.
	_, _, err := s.Resolve("eldara/demo", "")
	require.NoError(t, err)
	require.Empty(t, warnings)

	age(t, s, "eldara", 3*24*time.Hour)

	// Twice: an apply resolves every release in the manifest against the same
	// repository, and one warning per release would bury the plan it precedes.
	for range 2 {
		_, _, err = s.Resolve("eldara/demo", "")
		require.NoError(t, err)
	}
	require.Len(t, warnings, 1)
	require.Contains(t, warnings[0], "'eldara'")
	require.Contains(t, warnings[0], "3 days old")
	require.Contains(t, warnings[0], "charts repo update")
}

// A repository that publishes rarely serves the same bytes for weeks. Update
// writes nothing in that case, so unless it records the check some other way,
// the operator who refreshes daily is the one who gets nagged.
func TestUpdateMarksAnUnchangedIndexAsVerified(t *testing.T) {
	body := "apiVersion: v1\nentries:\n  demo:\n    - {name: demo, version: 0.1.0, urls: [\"d.tgz\"]}\n"
	up := true
	srv := newIndexServer(t, &body, &up)

	var warnings []string
	s := newTestStore(t)
	s.Warnf = func(format string, a ...any) { warnings = append(warnings, fmt.Sprintf(format, a...)) }
	require.NoError(t, s.Add("eldara", srv.URL))
	age(t, s, "eldara", 3*24*time.Hour)

	_, unchanged, err := s.Update("eldara")
	require.NoError(t, err)
	require.Equal(t, []string{"eldara"}, unchanged)

	_, _, err = s.Resolve("eldara/demo", "")
	require.NoError(t, err)
	require.Empty(t, warnings, "an index just confirmed current is not stale")
}
