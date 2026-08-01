// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package charts

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

// chartFiles is the chart every test below resolves against: one file at the
// top of files/ and one nested, so the recursive shape is always available.
func chartFiles() map[string][]byte {
	return map[string][]byte{
		"files/nginx.conf":  []byte("server { listen 80; }"),
		"files/tls/ca.pem":  []byte("-----BEGIN CERTIFICATE-----"),
		"files/unused.conf": []byte("nothing references this"),
	}
}

func quoted(p string) string { return strconv.Quote(p) }

// The three manifest shapes, one per key that reads a path. Each names exactly
// one path so a refusal can only have come from the key under test.
func configManifest(p string) string {
	return "services:\n  web:\n    image: nginx\nconfigs:\n  site:\n    file: " + quoted(p) + "\n"
}

func secretManifest(p string) string {
	return "services:\n  web:\n    image: nginx\nsecrets:\n  site:\n    file: " + quoted(p) + "\n"
}

func envFileManifest(p string) string {
	return "services:\n  web:\n    image: nginx\n    env_file: " + quoted(p) + "\n"
}

// TestResolveManifestFilesRefusesEveryShapeOnEveryKey is the specification: the
// guard covers three keys, and a test that covers one proves nothing about the
// other two — swarmcli-cd#99 was one key, and this is the same defect three
// times. Dropping any single key from manifestFileRefs fails four subtests.
func TestResolveManifestFilesRefusesEveryShapeOnEveryKey(t *testing.T) {
	keys := []struct {
		name     string
		manifest func(string) string
		errKey   string
	}{
		{"configs.file", configManifest, "configs.site.file"},
		{"secrets.file", secretManifest, "secrets.site.file"},
		{"services.env_file", envFileManifest, "services.web.env_file"},
	}
	shapes := []struct {
		name  string
		path  string
		wants []string
	}{
		{
			name: "absolute",
			path: "/etc/shadow",
			// The absolute case is the one with a working chart behind it, so
			// its message is the entire migration path (R-4): the rule, and the
			// operator-managed replacement by name.
			wants: []string{
				`"/etc/shadow"`, "absolute path", "files/",
				"external: true", "docker config create", "docker secret create",
			},
		},
		{
			name:  "escapes the chart",
			path:  "../../etc/shadow",
			wants: []string{`"../../etc/shadow"`, "escapes the chart", "files/"},
		},
		{
			name:  "well-formed but absent",
			path:  "files/missing.conf",
			wants: []string{`"files/missing.conf"`, "not in the chart", "files/"},
		},
		{
			// The likeliest author mistake, so the message has to say files/ and
			// show the corrected path.
			name:  "outside files/",
			path:  "nginx.conf",
			wants: []string{`"nginx.conf"`, "outside files/", `"files/nginx.conf"`},
		},
	}
	for _, k := range keys {
		for _, s := range shapes {
			t.Run(k.name+"/"+s.name, func(t *testing.T) {
				got, err := ResolveManifestFiles(k.manifest(s.path), chartFiles())
				require.Nil(t, got)
				require.Error(t, err)
				require.Contains(t, err.Error(), k.errKey, "the error must name the offending key")
				for _, want := range s.wants {
					require.Contains(t, err.Error(), want)
				}
			})
		}
	}
}

func TestResolveManifestFilesEscapesFromInsideFiles(t *testing.T) {
	// files/../../etc/shadow cleans to ../etc/shadow: it leaves the chart root
	// even though it started under files/, which is what path.Clean is for.
	_, err := ResolveManifestFiles(configManifest("files/../../etc/shadow"), chartFiles())
	require.ErrorContains(t, err, `"files/../../etc/shadow" escapes the chart`)

	// files/../nginx.conf stays inside the chart but leaves files/, so it is the
	// outside-files/ refusal and not the escape one.
	_, err = ResolveManifestFiles(configManifest("files/../nginx.conf"), chartFiles())
	require.ErrorContains(t, err, "is outside files/")
}

func TestResolveManifestFilesResolvesWhatTheManifestNames(t *testing.T) {
	got, err := ResolveManifestFiles(configManifest("files/nginx.conf"), chartFiles())
	require.NoError(t, err)
	require.Equal(t, map[string][]byte{"files/nginx.conf": []byte("server { listen 80; }")}, got)
}

func TestResolveManifestFilesResolvesANestedFile(t *testing.T) {
	got, err := ResolveManifestFiles(secretManifest("files/tls/ca.pem"), chartFiles())
	require.NoError(t, err)
	require.Equal(t, map[string][]byte{"files/tls/ca.pem": []byte("-----BEGIN CERTIFICATE-----")}, got)
}

func TestResolveManifestFilesNormalisesTheKey(t *testing.T) {
	// ./files/nginx.conf resolves to the same file the docker CLI would read, so
	// it comes back under the key Chart.Files uses — the key the materialiser
	// writes to.
	got, err := ResolveManifestFiles(configManifest("./files/nginx.conf"), chartFiles())
	require.NoError(t, err)
	require.Equal(t, map[string][]byte{"files/nginx.conf": []byte("server { listen 80; }")}, got)
}

func TestResolveManifestFilesReturnsOnlyTheReferencedSubset(t *testing.T) {
	manifest := "services:\n" +
		"  web:\n    image: nginx\n    env_file: files/tls/ca.pem\n" +
		"configs:\n  site:\n    file: files/nginx.conf\n"
	got, err := ResolveManifestFiles(manifest, chartFiles())
	require.NoError(t, err)
	require.Equal(t, map[string][]byte{
		"files/nginx.conf": []byte("server { listen 80; }"),
		"files/tls/ca.pem": []byte("-----BEGIN CERTIFICATE-----"),
	}, got)
	require.NotContains(t, got, "files/unused.conf")
}

func TestResolveManifestFilesReturnsNilWhenNothingIsNamed(t *testing.T) {
	cases := []struct {
		name     string
		manifest string
	}{
		{"no configs, secrets or env_file", "services:\n  web:\n    image: nginx\n"},
		{"external config carries no file", "configs:\n  site:\n    external: true\n"},
		{"driver-backed secret carries no file", "secrets:\n  s:\n    driver: vault\n"},
		{"empty manifest", ""},
		{"services entry is not a map", "services:\n  web: nginx\n"},
		{"services block is a list", "services:\n  - web\n"},
		{"configs entry is not a map", "configs:\n  site: true\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ResolveManifestFiles(tc.manifest, chartFiles())
			require.NoError(t, err)
			require.Nil(t, got, "a manifest naming no file must yield nil, not an empty map")
		})
	}
}

func TestResolveManifestFilesReadsBothEnvFileShapes(t *testing.T) {
	t.Run("bare string", func(t *testing.T) {
		got, err := ResolveManifestFiles(envFileManifest("files/nginx.conf"), chartFiles())
		require.NoError(t, err)
		require.Equal(t, map[string][]byte{"files/nginx.conf": []byte("server { listen 80; }")}, got)
	})
	t.Run("list", func(t *testing.T) {
		manifest := "services:\n  web:\n    image: nginx\n    env_file:\n" +
			"      - files/nginx.conf\n      - files/tls/ca.pem\n"
		got, err := ResolveManifestFiles(manifest, chartFiles())
		require.NoError(t, err)
		require.Len(t, got, 2)
		require.Contains(t, got, "files/nginx.conf")
		require.Contains(t, got, "files/tls/ca.pem")
	})
	t.Run("a bad path anywhere in the list is refused", func(t *testing.T) {
		manifest := "services:\n  web:\n    image: nginx\n    env_file:\n" +
			"      - files/nginx.conf\n      - /etc/shadow\n"
		_, err := ResolveManifestFiles(manifest, chartFiles())
		require.ErrorContains(t, err, `"/etc/shadow"`)
	})
}

// TestResolveManifestFilesSeesEntriesBesideAMalformedOne is the load-bearing
// test for walking yaml.Nodes rather than map[string]any: a single non-string
// key makes yaml.v3 hand back a map[any]any for the WHOLE mapping, so a type
// assertion would see no services at all and let /etc/shadow through.
func TestResolveManifestFilesSeesEntriesBesideAMalformedOne(t *testing.T) {
	cases := []string{
		"services:\n  1:\n    image: nginx\n  web:\n    env_file: /etc/shadow\n",
		"services:\n  broken: nginx\n  web:\n    env_file: /etc/shadow\n",
		"configs:\n  1:\n    external: true\n  site:\n    file: /etc/shadow\n",
	}
	for _, manifest := range cases {
		_, err := ResolveManifestFiles(manifest, chartFiles())
		require.ErrorContains(t, err, `"/etc/shadow"`, manifest)
	}
}

func TestResolveManifestFilesRefusesAManifestItCannotParse(t *testing.T) {
	_, err := ResolveManifestFiles("services:\n  web:\n   - :\n  bad\n", chartFiles())
	require.ErrorContains(t, err, "parse manifest")
}

// TestResolveManifestFilesRefusesDeterministically pins the sort in
// manifestFileRefs: without it, which of several bad paths is reported depends
// on Go's map iteration order and the error text flakes.
func TestResolveManifestFilesRefusesDeterministically(t *testing.T) {
	manifest := "configs:\n" +
		"  a:\n    file: /etc/shadow\n" +
		"  b:\n    file: ../../etc/passwd\n" +
		"  c:\n    file: nginx.conf\n"
	first, err := ResolveManifestFiles(manifest, chartFiles())
	require.Nil(t, first)
	require.Error(t, err)
	for range 20 {
		_, again := ResolveManifestFiles(manifest, chartFiles())
		require.EqualError(t, again, err.Error())
	}
	require.Contains(t, err.Error(), "configs.a.file")
}

func TestResolveManifestFilesRefusesAgainstAChartWithNoFiles(t *testing.T) {
	_, err := ResolveManifestFiles(configManifest("files/nginx.conf"), nil)
	require.ErrorContains(t, err, `"files/nginx.conf" is not in the chart`)
}
