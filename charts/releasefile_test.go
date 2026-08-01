// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package charts

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
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

// A release declares its wave, or is in wave 0 by having said nothing.
func TestParseReadsWaves(t *testing.T) {
	rf, err := parse(t, `releases:
  - {name: db, chart: r/pg, version: "1"}
  - {name: migrate, chart: r/migrate, version: "1", wave: 1}
  - {name: api, chart: r/api, version: "1", wave: 2}
  - {name: early, chart: r/early, version: "1", wave: -1}
`)
	require.NoError(t, err)
	require.Equal(t, []int{0, 1, 2, -1}, []int{
		rf.Releases[0].Wave, rf.Releases[1].Wave, rf.Releases[2].Wave, rf.Releases[3].Wave,
	}, "an undeclared wave is 0 and a negative one is legal; the file's own order is untouched by parsing")
}

// A release file round-trips through Marshal without growing a wave nobody
// wrote.
//
// This is not tidiness. swarmcli-cd synthesises a release file for its
// one-application-one-chart source by marshalling this struct and handing the
// bytes back to ParseReleaseFile, so an un-omitted zero would put `wave: 0` into
// every such application — a key an operator never wrote, in a file they cannot
// see, for a file with one release where waves cannot mean anything.
func TestMarshalOmitsAnUndeclaredWave(t *testing.T) {
	doc, err := yaml.Marshal(ReleaseFile{
		APIVersion: "v1",
		Releases:   []ReleaseSpec{{Name: "solo", Chart: "./charts/app"}},
	})
	require.NoError(t, err)
	require.NotContains(t, string(doc), "wave")

	// And a declared one survives the same trip, so the omission is about zero
	// rather than about the field.
	doc, err = yaml.Marshal(ReleaseFile{
		APIVersion: "v1",
		Releases:   []ReleaseSpec{{Name: "solo", Chart: "./charts/app", Wave: 3}},
	})
	require.NoError(t, err)
	require.Contains(t, string(doc), "wave: 3")

	rf, err := parse(t, string(doc))
	require.NoError(t, err)
	require.Equal(t, 3, rf.Releases[0].Wave)
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
		"no releases":   {"releases: []\n", "no releases"},
		"missing name":  {"releases:\n  - chart: r/c\n    version: \"1\"\n", "name is required"},
		"missing chart": {"releases:\n  - name: a\n", "chart is required"},
		"dup release":   {"releases:\n  - {name: a, chart: r/c, version: \"1\"}\n  - {name: a, chart: r/d, version: \"1\"}\n", "duplicate release"},
		"dup repo":      {"repositories:\n  - {name: r, url: http://x}\n  - {name: r, url: http://y}\nreleases:\n  - {name: a, chart: r/c, version: \"1\"}\n", "duplicate repository"},
		"repo no url":   {"repositories:\n  - {name: r}\nreleases:\n  - {name: a, chart: r/c, version: \"1\"}\n", "url is required"},
		"repo no name":  {"repositories:\n  - {url: http://x}\nreleases:\n  - {name: a, chart: r/c, version: \"1\"}\n", "name is required"},
		// The name becomes a path component of the cached index, so a manifest
		// out of a git repository must not be able to steer where that is written.
		"repo bad name":  {"repositories:\n  - {name: \"../../../../pwned\", url: https://x}\nreleases:\n  - {name: a, chart: r/c, version: \"1\"}\n", "invalid repository name"},
		"bad apiVersion": {"apiVersion: v2\nreleases:\n  - {name: a, chart: r/c, version: \"1\"}\n", "unsupported apiVersion"},
		"bad rel name":   {"releases:\n  - {name: \"Bad Name\", chart: r/c, version: \"1\"}\n", "Bad Name"},
		// A wave is an ordering, so anything that is not a number is a mistake
		// rather than something to coerce. yaml refuses it for us and reports the
		// line, though not the key — same as any other type mismatch in this file.
		// Asserted so that stays true if the field's type is ever widened.
		"wave not a number": {"releases:\n  - {name: a, chart: r/c, version: \"1\", wave: soon}\n", "cannot unmarshal"},
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

// chartNameOf backs the "run `charts search <chart>`" hint in the version-required
// error; a reference with no separator must fall back to the whole string.
func TestChartNameOf(t *testing.T) {
	require.Equal(t, "whoami", chartNameOf("swarmcli-charts/whoami"))
	require.Equal(t, "whoami", chartNameOf("whoami"))
	require.Equal(t, "c", chartNameOf("a/b/c"))
}

// The owner is optional and namespaced under "apply/" when stamped, so a
// manifest applied from the command line and an application reconciled by a
// controller cannot claim each other's releases by picking the same name.
func TestReleaseFileOwner(t *testing.T) {
	rf, err := ParseReleaseFile([]byte(`owner: prod-swarm
releases:
  - name: hello
    chart: repo/demo
    version: "1.0.0"
`), "f.yaml")
	require.NoError(t, err)
	require.Equal(t, "prod-swarm", rf.Owner)
	require.Equal(t, "apply/prod-swarm", rf.ownerID())

	rf, err = ParseReleaseFile([]byte(`releases:
  - name: hello
    chart: repo/demo
    version: "1.0.0"
`), "f.yaml")
	require.NoError(t, err)
	require.Empty(t, rf.ownerID(), "no owner declared means nothing is claimed")
}

// ':' separates the id from the resource half of a stamp, so an owner carrying
// one would decode as a different owner than it was written as. Catch it at
// parse time, where the error can name the file.
func TestReleaseFileRejectsOwnerWithColon(t *testing.T) {
	_, err := ParseReleaseFile([]byte(`owner: "prod:swarm"
releases:
  - name: hello
    chart: repo/demo
    version: "1.0.0"
`), "f.yaml")
	require.ErrorContains(t, err, "must not contain ':'")
}
