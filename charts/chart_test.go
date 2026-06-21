// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package charts

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// packDirToTgz packs a directory into a gzipped tar whose entries are prefixed
// with prefix/, matching the layout of a packaged chart.
func packDirToTgz(t *testing.T, dir, prefix string) string {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	err := filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return err
		}
		body, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		hdr := &tar.Header{Name: prefix + "/" + filepath.ToSlash(rel), Mode: 0o644, Size: int64(len(body)), Typeflag: tar.TypeReg}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		_, err = tw.Write(body)
		return err
	})
	require.NoError(t, err)
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
	return buf.String()
}

func TestLoadChartDirMissingChartfile(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "templates"), 0o755))
	_, err := LoadChartDir(dir)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Chart.yaml")
}

func TestLoadChartDirNoTemplates(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Chart.yaml"), []byte("name: x\nversion: 1.0.0\n"), 0o644))
	_, err := LoadChartDir(dir)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "templates"))
}
