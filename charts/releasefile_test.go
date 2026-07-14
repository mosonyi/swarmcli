// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package charts

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func parse(t *testing.T, body string) (*ReleaseFile, error) {
	t.Helper()
	return ParseReleaseFile([]byte(body), "/wd/deploy/swarmcli-release.yaml")
}

func TestParseReleaseFile(t *testing.T) {
	rf, err := parse(t, `apiVersion: v1
repositories:
  - name: swarmcli-charts
    url: https://eldara-tech.github.io/swarmcli-charts
releases:
  - name: edge
    chart: swarmcli-charts/traefik
    version: "0.1.1"
    values: [./traefik.yaml]
  - name: hello
    chart: swarmcli-charts/whoami
    version: "0.1.8"
`)
	require.NoError(t, err)
	require.Len(t, rf.Releases, 2)
	require.Equal(t, "swarmcli-charts", rf.Repositories[0].Name)
	require.Equal(t, "0.1.1", rf.Releases[0].Version)
}

// A file an automated updater rewrites must fail loudly on a typo, not silently
// leave the release floating.
//
//nolint:misspell // "verison" is a deliberate typo — it is what the test rejects.
func TestParseRejectsUnknownKeys(t *testing.T) {
	_, err := parse(t, `releases:
  - name: hello
    chart: swarmcli-charts/whoami
    verison: "0.1.8"
`)
	require.ErrorContains(t, err, "verison")
}

// Borrowing Helmfile's key names means people will paste Helmfile syntax. Strict
// decoding turns that into a clear error naming the key rather than silent
// half-behaviour.
func TestParseRejectsHelmfileOnlyKeys(t *testing.T) {
	_, err := parse(t, `environments:
  default: {}
releases:
  - name: hello
    chart: swarmcli-charts/whoami
    version: "0.1.8"
`)
	require.ErrorContains(t, err, "environments")
}

func TestParseRequiresVersionForRepoCharts(t *testing.T) {
	_, err := parse(t, `releases:
  - name: hello
    chart: swarmcli-charts/whoami
`)
	require.ErrorContains(t, err, "version is required")
	require.ErrorContains(t, err, "hello")
	// The message should point at the fix.
	require.ErrorContains(t, err, "charts search whoami")
}

// A local chart carries its version in its own Chart.yaml, so `version:` selects
// nothing. Accepting it would be a lie — the pre-existing `--version` bug in
// imperative install, surfaced here as an explicit error.
func TestParseRejectsVersionOnLocalChart(t *testing.T) {
	_, err := parse(t, `releases:
  - name: hello
    chart: ./charts/mine
    version: "0.1.8"
`)
	require.ErrorContains(t, err, "version must be omitted")

	ok, err := parse(t, `releases:
  - name: hello
    chart: ./charts/mine
`)
	require.NoError(t, err)
	require.Equal(t, "./charts/mine", ok.Releases[0].Chart)
}

func TestParseValidationErrors(t *testing.T) {
	for name, tc := range map[string]struct{ body, want string }{
		"no releases":    {"releases: []\n", "no releases"},
		"missing name":   {"releases:\n  - chart: r/c\n    version: \"1\"\n", "name is required"},
		"missing chart":  {"releases:\n  - name: a\n", "chart is required"},
		"dup release":    {"releases:\n  - {name: a, chart: r/c, version: \"1\"}\n  - {name: a, chart: r/d, version: \"1\"}\n", "duplicate release"},
		"dup repo":       {"repositories:\n  - {name: r, url: http://x}\n  - {name: r, url: http://y}\nreleases:\n  - {name: a, chart: r/c, version: \"1\"}\n", "duplicate repository"},
		"repo no url":    {"repositories:\n  - {name: r}\nreleases:\n  - {name: a, chart: r/c, version: \"1\"}\n", "url is required"},
		"bad apiVersion": {"apiVersion: v2\nreleases:\n  - {name: a, chart: r/c, version: \"1\"}\n", "unsupported apiVersion"},
		"bad rel name":   {"releases:\n  - {name: \"Bad Name\", chart: r/c, version: \"1\"}\n", "Bad Name"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := parse(t, tc.body)
			require.ErrorContains(t, err, tc.want)
		})
	}
}

// Values and local chart paths resolve against the FILE's directory, never the
// process working directory: the file is committed to git and must produce the
// same result no matter where CI invoked swarmcli from.
func TestPathsResolveAgainstTheManifestNotTheCWD(t *testing.T) {
	rf, err := parse(t, `releases:
  - name: a
    chart: ./charts/mine
    values: [./v.yaml, ../shared.yaml, /abs/v.yaml]
`)
	require.NoError(t, err)

	got := rf.ValuesPaths(rf.Releases[0])
	require.Equal(t, []string{
		filepath.FromSlash("/wd/deploy/v.yaml"),
		filepath.FromSlash("/wd/shared.yaml"),
		filepath.FromSlash("/abs/v.yaml"),
	}, got)

	require.Equal(t, filepath.FromSlash("/wd/deploy/charts/mine"), rf.ChartRef(rf.Releases[0]))
}

// A "<repo>/<chart>" reference is passed through untouched.
func TestChartRefPassesThroughRepoReferences(t *testing.T) {
	rf, err := parse(t, "releases:\n  - {name: a, chart: swarmcli-charts/whoami, version: \"0.1.8\"}\n")
	require.NoError(t, err)
	require.Equal(t, "swarmcli-charts/whoami", rf.ChartRef(rf.Releases[0]))
}

// Whether a reference is a path must not depend on what happens to exist on the
// disk of whichever machine runs the file.
func TestIsPathRefIsSyntactic(t *testing.T) {
	for _, ref := range []string{"./x", "../x", "/x", "~/x"} {
		require.True(t, IsPathRef(ref), ref)
	}
	for _, ref := range []string{"repo/chart", "chart", "swarmcli-charts/whoami"} {
		require.False(t, IsPathRef(ref), ref)
	}
}
