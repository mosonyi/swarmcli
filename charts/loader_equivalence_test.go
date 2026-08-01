// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package charts

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// The same chart reaches the engine through two loaders — a repository serves
// the tarball, a local path serves the directory — and they have drifted
// before: the archive loader bounded itself with maxChartFiles/maxChartFileSize/
// maxChartTotalSize while the directory loader applied none of them, and the
// archive loader skipped symlinks while the directory loader followed them. No
// per-loader test can see that class of bug, because each loader is
// self-consistent. Only comparing them can.
func TestLoadersAgreeOnEveryMember(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "templates"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "files", "tls"), 0o755))

	wantFiles := map[string][]byte{
		"files/nginx.conf": []byte("server { listen 80; }\n"),
		"files/tls/ca.pem": []byte("-----BEGIN CERTIFICATE-----\nnot-a-real-cert\n-----END CERTIFICATE-----\n"),
	}
	for name, body := range map[string]string{
		"Chart.yaml": "apiVersion: v1\nname: equiv\nversion: 0.1.0\nappVersion: \"1.0\"\n" +
			"description: Chart exercising every member both loaders read\n" +
			"maintainers:\n  - name: Eldara Tech\n    email: hello@eldara.io\n",
		"values.yaml":           "replicas: 2\nimage:\n  repo: traefik\n  tag: v3.0.0\n",
		"values.schema.json":    "{\n  \"type\": \"object\",\n  \"required\": [\"replicas\", \"image\"]\n}\n",
		"requirements.yaml":     "networks:\n  - name: traefik-public\n    description: shared ingress\n",
		"README.md":             "# equiv\n\nChart used by the loader-equivalence test.\n",
		"templates/stack.yaml":  "services:\n  web:\n    image: {{ .Values.image.repo }}:{{ .Values.image.tag }}\n",
		"templates/extras.yaml": "# nothing here, but it must reach Templates\n",
		"files/nginx.conf":      string(wantFiles["files/nginx.conf"]),
		"files/tls/ca.pem":      string(wantFiles["files/tls/ca.pem"]),
	} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, filepath.FromSlash(name)), []byte(body), 0o644))
	}

	fromDir, err := LoadChartDir(dir)
	require.NoError(t, err)

	// A packaged chart is a single top-level directory the archive loader
	// strips; pack it flat and the loader finds no chart at all, so the
	// comparison below would be against nothing.
	fromArchive, err := LoadChartArchive(strings.NewReader(packDirToTgz(t, dir, "equiv")))
	require.NoError(t, err)

	// The whole struct, never a field list: a test that names the fields keeps
	// passing when a field is added, which is the drift it exists to catch.
	require.Equal(t, *fromDir, *fromArchive)

	// Equality only proves the loaders agree — two of them leaving a field zero
	// agree vacuously. Reflection rather than a written-out list so that a field
	// added later fails here until the fixture above exercises it too. If a
	// field ever legitimately cannot be exercised, assert on it by name with the
	// reason, rather than dropping it from the loop.
	v := reflect.ValueOf(*fromDir)
	for i := range v.NumField() {
		require.Falsef(t, v.Field(i).IsZero(),
			"Chart.%s is zero: extend this chart so the loader comparison covers it", v.Type().Field(i).Name)
	}

	// And equality says nothing about whether they agree on the right answer.
	// The intended shape, stated once: keys relative to the chart root,
	// slash-separated, recursive under files/.
	require.Equal(t, wantFiles, fromDir.Files)
}

// Where the loaders deliberately do NOT agree, and why it is safe.
//
// A symlink under files/ is refused by LoadChartDir (see the directory loader's
// own tests). The archive loader never gets the chance: tar.TypeSymlink is not
// tar.TypeReg, so the entry is skipped before anything reads it, and the chart
// loads with that file simply absent. Refusing there instead would buy nothing —
// there is no content to refuse.
//
// The consequence is the point of this test, and it is not a loader property at
// all: symlink safety in the archive path is decided at PACKAGING time. A packer
// that stores links as links — GNU tar's default, which is what
// swarmcli-charts/Makefile's `tar -czf` uses — produces the entry this test
// builds, and the link is dropped. A packer that FOLLOWS links resolves them
// while packing, so the target's bytes arrive as an ordinary regular file and
// no loader can tell. packDirToTgz in this package is itself such a packer: it
// calls os.ReadFile and writes tar.TypeReg. So `tar -czhf`, or a Go packer
// written the way that helper is, would put /etc/passwd in a chart and every
// check downstream would see a perfectly ordinary file.
//
// That is why the refusal lives in LoadChartDir, where the link is still a link.
func TestTheArchiveLoaderDropsASymlinkRatherThanRefusingIt(t *testing.T) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	write := func(name, body string) {
		require.NoError(t, tw.WriteHeader(&tar.Header{
			Name: "x/" + name, Mode: 0o644, Size: int64(len(body)), Typeflag: tar.TypeReg,
		}))
		_, err := tw.Write([]byte(body))
		require.NoError(t, err)
	}
	write("Chart.yaml", "name: x\nversion: 1.0.0\n")
	write("templates/stack.yaml", "services: {}\n")
	write("files/real.conf", "kept\n")
	// The link itself: no body, and a Typeflag the loader's regular-file test
	// rejects before it reads anything.
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name: "x/files/link.conf", Linkname: "/etc/passwd", Mode: 0o777, Typeflag: tar.TypeSymlink,
	}))
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())

	ch, err := LoadChartArchive(&buf)
	require.NoError(t, err, "the archive loader drops a symlink; it does not refuse the chart")
	require.Equal(t, map[string][]byte{"files/real.conf": []byte("kept\n")}, ch.Files)
}

// tar carries no entry for an empty directory, so the archive loader cannot see
// an empty files/ where the directory loader can. That asymmetry is real and
// invisible to either loader's own tests, and the answer both must give is nil
// rather than an empty map.
func TestLoadersAgreeOnEmptyFilesDir(t *testing.T) {
	dir := t.TempDir()
	writeMinimalChart(t, dir)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "files"), 0o755))

	fromDir, err := LoadChartDir(dir)
	require.NoError(t, err)
	fromArchive, err := LoadChartArchive(strings.NewReader(packDirToTgz(t, dir, "x")))
	require.NoError(t, err)

	require.Nil(t, fromDir.Files)
	require.Nil(t, fromArchive.Files)
	require.Equal(t, *fromDir, *fromArchive)
}
