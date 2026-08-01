// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package charts

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// writeChartFile writes one file under the chart's files/ directory, creating
// whatever parents the key names.
func writeChartFile(t *testing.T, dir, rel, body string) {
	t.Helper()
	full := filepath.Join(dir, filepath.FromSlash(rel))
	require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
	require.NoError(t, os.WriteFile(full, []byte(body), 0o644))
}

// writeSparse creates a file of exactly n bytes without spending them. The
// limit tests need sizes in the tens of MiB and none of them care what is in
// the file, so the bytes are a hole.
func writeSparse(t *testing.T, dir, rel string, n int64) {
	t.Helper()
	full := filepath.Join(dir, filepath.FromSlash(rel))
	require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
	f, err := os.Create(full)
	require.NoError(t, err)
	require.NoError(t, f.Truncate(n))
	require.NoError(t, f.Close())
}

func TestLoadChartDirCollectsFiles(t *testing.T) {
	dir := t.TempDir()
	writeMinimalChart(t, dir)
	writeChartFile(t, dir, "files/nginx.conf", "server {}\n")
	writeChartFile(t, dir, "files/tls/ca.pem", "-----BEGIN CERTIFICATE-----\n")

	ch, err := LoadChartDir(dir)
	require.NoError(t, err)
	require.Equal(t, map[string][]byte{
		"files/nginx.conf": []byte("server {}\n"),
		"files/tls/ca.pem": []byte("-----BEGIN CERTIFICATE-----\n"),
	}, ch.Files)
}

func TestLoadChartArchiveCollectsFiles(t *testing.T) {
	tgz := packEntries(t, map[string]string{
		"demo/Chart.yaml":           "name: demo\nversion: 1.0.0\n",
		"demo/templates/stack.yaml": "services: {}\n",
		"demo/files/nginx.conf":     "server {}\n",
		"demo/files/tls/ca.pem":     "-----BEGIN CERTIFICATE-----\n",
	})
	ch, err := LoadChartArchive(strings.NewReader(tgz))
	require.NoError(t, err)
	require.Equal(t, map[string][]byte{
		"files/nginx.conf": []byte("server {}\n"),
		"files/tls/ca.pem": []byte("-----BEGIN CERTIFICATE-----\n"),
	}, ch.Files)
}

// Nil, not empty: every chart written before files/ existed must load exactly
// as it did, and a caller that ranges over ch.Files must see the same nothing
// from both loaders.
func TestLoadChartWithoutFilesIsNil(t *testing.T) {
	dir := t.TempDir()
	writeMinimalChart(t, dir)
	ch, err := LoadChartDir(dir)
	require.NoError(t, err)
	require.Nil(t, ch.Files)

	tgz := packEntries(t, map[string]string{
		"demo/Chart.yaml":           "name: demo\nversion: 1.0.0\n",
		"demo/templates/stack.yaml": "services: {}\n",
	})
	ch, err = LoadChartArchive(strings.NewReader(tgz))
	require.NoError(t, err)
	require.Nil(t, ch.Files)
}

func TestLoadChartDirRefusesSymlinkInFiles(t *testing.T) {
	dir := t.TempDir()
	writeMinimalChart(t, dir)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, filesDir), 0o755))
	require.NoError(t, os.Symlink("/etc/passwd", filepath.Join(dir, filesDir, "app.conf")))

	_, err := LoadChartDir(dir)
	require.ErrorContains(t, err, "files/app.conf")
	require.ErrorContains(t, err, "symlink")
}

// The behaviour change: a symlinked template used to be followed, so
// templates/x.yaml -> /etc/passwd made an arbitrary operator-side file a
// template source. LoadChartArchive never had that hole (it skips every entry
// that is not tar.TypeReg), and now neither loader does.
func TestLoadChartDirRefusesSymlinkInTemplates(t *testing.T) {
	dir := t.TempDir()
	writeMinimalChart(t, dir)
	require.NoError(t, os.Symlink("/etc/passwd", filepath.Join(dir, templatesDir, "leak.yaml")))

	_, err := LoadChartDir(dir)
	require.ErrorContains(t, err, "templates/leak.yaml")
	require.ErrorContains(t, err, "symlink")
}

// The directory loader's mirror of TestLoadChartArchiveTooManyFiles: the three
// limits were declared for both loaders and enforced by one.
func TestLoadChartDirTooManyFiles(t *testing.T) {
	dir := t.TempDir()
	writeMinimalChart(t, dir)
	fdir := filepath.Join(dir, filesDir)
	require.NoError(t, os.MkdirAll(fdir, 0o755))
	for i := 0; i <= maxChartFiles; i++ {
		if err := os.WriteFile(filepath.Join(fdir, fmt.Sprintf("f%d", i)), []byte("x"), 0o644); err != nil {
			require.NoError(t, err)
		}
	}
	_, err := LoadChartDir(dir)
	require.ErrorContains(t, err, "too many files")
}

func TestLoadChartDirFileTooLarge(t *testing.T) {
	dir := t.TempDir()
	writeMinimalChart(t, dir)
	writeSparse(t, dir, "files/big.bin", maxChartFileSize+1)

	_, err := LoadChartDir(dir)
	require.ErrorContains(t, err, "files/big.bin")
	require.ErrorContains(t, err, fmt.Sprintf("exceeds %d bytes", maxChartFileSize))
}

func TestLoadChartDirTotalTooLarge(t *testing.T) {
	dir := t.TempDir()
	writeMinimalChart(t, dir)
	for i := 0; i <= maxChartTotalSize/maxChartFileSize; i++ {
		writeSparse(t, dir, fmt.Sprintf("files/big%d.bin", i), maxChartFileSize)
	}

	_, err := LoadChartDir(dir)
	require.ErrorContains(t, err, fmt.Sprintf("more than %d bytes", maxChartTotalSize))
}

// The counters are per-chart, not per-walk. templates/ holds 40 MiB and files/
// holds 40 MiB: each walk is comfortably inside the 64 MiB limit and the chart
// is not, so a loader counting each directory on its own accepts this and this
// test is what says so.
func TestLoadChartDirLimitsAreSharedAcrossDirectories(t *testing.T) {
	const each = 4 // 4 x 10 MiB = 40 MiB per directory, 80 MiB in total
	dir := t.TempDir()
	writeMinimalChart(t, dir)
	for i := 0; i < each; i++ {
		writeSparse(t, dir, fmt.Sprintf("templates/big%d.yaml", i), maxChartFileSize)
		writeSparse(t, dir, fmt.Sprintf("files/big%d.bin", i), maxChartFileSize)
	}
	require.Less(t, int64(each)*maxChartFileSize, int64(maxChartTotalSize), "each directory must be under the limit on its own")

	_, err := LoadChartDir(dir)
	require.ErrorContains(t, err, fmt.Sprintf("more than %d bytes", maxChartTotalSize))
}
