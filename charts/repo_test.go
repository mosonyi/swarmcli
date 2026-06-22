// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package charts

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

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

	s := NewRepoStoreAt(t.TempDir())
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

	s := NewRepoStoreAt(t.TempDir())
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

func TestRepoStoreAddRejectsBadURL(t *testing.T) {
	s := NewRepoStoreAt(t.TempDir())
	require.Error(t, s.Add("x", "not-a-url"))
	require.Error(t, s.Add("x", "ftp://example.com"))
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

	s := NewRepoStoreAt(t.TempDir())

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

	s := NewRepoStoreAt(t.TempDir())
	require.NoError(t, s.Add("eldara", srv.URL))
	e, base, err := s.Resolve("eldara/demo", "")
	require.NoError(t, err)
	ch, err := s.Pull(e, base)
	require.NoError(t, err)
	require.Equal(t, "demo", ch.Metadata.Name)
}

func TestCompareVersions(t *testing.T) {
	require.Equal(t, 1, compareVersions("0.2.0", "0.1.0"))
	require.Equal(t, -1, compareVersions("1.0.0", "1.0.1"))
	require.Equal(t, 1, compareVersions("v3.5.0", "3.4.0"))
}
