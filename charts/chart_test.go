// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package charts

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// packEntries packs in-memory files into a gzipped tar for archive-loader tests.
func packEntries(t *testing.T, files map[string]string) string {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, body := range files {
		require.NoError(t, tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(body)), Typeflag: tar.TypeReg}))
		_, err := tw.Write([]byte(body))
		require.NoError(t, err)
	}
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
	return buf.String()
}

func TestLoadChartArchiveCorruptGzip(t *testing.T) {
	_, err := LoadChartArchive(strings.NewReader("not a gzip stream"))
	require.ErrorContains(t, err, "open gzip")
}

func TestLoadChartArchiveMissingChartfile(t *testing.T) {
	tgz := packEntries(t, map[string]string{"demo/templates/x.yaml": "services: {}"})
	_, err := LoadChartArchive(strings.NewReader(tgz))
	require.ErrorContains(t, err, "name is required")
}

// Validating apiVersion is what reserves a future value for a format break: an
// older build must refuse a chart it cannot read rather than load half of it.
func TestLoadChartAPIVersion(t *testing.T) {
	for _, tc := range []struct {
		name    string
		line    string
		wantErr string
	}{
		{name: "absent means v1", line: ""},
		{name: "explicit v1", line: "apiVersion: v1\n"},
		{name: "a future format is refused", line: "apiVersion: v2\n", wantErr: `unsupported apiVersion 'v2'`},
		{name: "nonsense is refused", line: "apiVersion: v99\n", wantErr: `unsupported apiVersion 'v99'`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tgz := packEntries(t, map[string]string{
				"demo/Chart.yaml":           tc.line + "name: demo\nversion: 0.1.0\n",
				"demo/templates/stack.yaml": "services: {}",
			})
			ch, err := LoadChartArchive(strings.NewReader(tgz))
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, "demo", ch.Metadata.Name)
		})
	}
}

func TestLoadChartSwarmcliVersion(t *testing.T) {
	tgz := packEntries(t, map[string]string{
		"demo/Chart.yaml":           "name: demo\nversion: 0.1.0\nswarmcliVersion: \">= 1.13.0\"\n",
		"demo/templates/stack.yaml": "services: {}",
	})
	ch, err := LoadChartArchive(strings.NewReader(tgz))
	require.NoError(t, err)
	require.Equal(t, ">= 1.13.0", ch.Metadata.SwarmcliVersion)
}

// The constraint is metadata, not a gate inside the loader: loading must succeed
// so callers can read the requirement and report it. Refusing here would make
// `charts show` unable to answer which version the chart wants.
func TestLoadChartUnsatisfiableSwarmcliVersionStillLoads(t *testing.T) {
	withEngineVersion(t, "1.0.0")
	tgz := packEntries(t, map[string]string{
		"demo/Chart.yaml":           "name: demo\nversion: 0.1.0\nswarmcliVersion: \">= 99.0.0\"\n",
		"demo/templates/stack.yaml": "services: {}",
	})
	ch, err := LoadChartArchive(strings.NewReader(tgz))
	require.NoError(t, err)
	require.Equal(t, CompatIncompatible, CheckCompat(ch.Metadata).Status)
}

func TestLoadChartArchiveTooManyFiles(t *testing.T) {
	files := map[string]string{}
	for i := 0; i <= maxChartFiles; i++ {
		files[fmt.Sprintf("demo/templates/f%d.yaml", i)] = "x"
	}
	_, err := LoadChartArchive(strings.NewReader(packEntries(t, files)))
	require.ErrorContains(t, err, "too many files")
}

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

func writeMinimalChart(t *testing.T, dir string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "templates"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Chart.yaml"), []byte("name: x\nversion: 1.0.0\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "templates", "stack.yaml"), []byte("services: {}\n"), 0o644))
}

func TestLoadChartDirParsesRequirements(t *testing.T) {
	dir := t.TempDir()
	writeMinimalChart(t, dir)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "requirements.yaml"),
		[]byte("networks:\n  - name: traefik-public\n    description: shared\n"), 0o644))

	ch, err := LoadChartDir(dir)
	require.NoError(t, err)
	require.NotNil(t, ch.Requirements)
	require.Len(t, ch.Requirements.Networks, 1)
	require.Equal(t, "traefik-public", ch.Requirements.Networks[0].Name)
	require.True(t, *ch.Requirements.Networks[0].AutoCreate) // defaulted
}

// requirements.yaml is a Go template, so a chart may `range` over a user-supplied
// list (e.g. the extra overlays to attach to). Its raw bytes are then not valid YAML
// on their own — chart load must NOT fail on that, and RenderRequirements must
// resolve it against the release's values.
func TestLoadChartDirRequirementsTemplateWithControlFlow(t *testing.T) {
	dir := t.TempDir()
	writeMinimalChart(t, dir)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "requirements.yaml"), []byte(
		"networks:\n"+
			"{{- range .Values.extraNetworks }}\n"+
			"  - name: \"{{ . }}\"\n"+
			"    autoCreate: false\n"+
			"{{- end }}\n"), 0o644))

	ch, err := LoadChartDir(dir)
	require.NoError(t, err, "a templated requirements.yaml must not fail chart load")
	require.NotNil(t, ch.RequirementsRaw)

	req, err := RenderRequirements(ch, RenderContext{Values: map[string]any{
		"extraNetworks": []any{"ai-internal", "mail-internal"},
	}})
	require.NoError(t, err)
	require.Len(t, req.Networks, 2)
	require.Equal(t, "ai-internal", req.Networks[0].Name)
	require.Equal(t, "mail-internal", req.Networks[1].Name)
	require.False(t, *req.Networks[0].AutoCreate)
	require.Equal(t, "overlay", req.Networks[0].Driver) // defaulted

	// An empty list declares nothing at all.
	empty, err := RenderRequirements(ch, RenderContext{Values: map[string]any{"extraNetworks": []any{}}})
	require.NoError(t, err)
	require.Empty(t, empty.Networks)
}

// Load is best-effort, but the authoritative parse must still reject a genuinely
// malformed requirements.yaml — the error just surfaces at render time instead.
func TestRenderRequirementsRejectsMalformedAfterRender(t *testing.T) {
	dir := t.TempDir()
	writeMinimalChart(t, dir)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "requirements.yaml"),
		[]byte("networks:\n  - name: \"a\"\n   bad-indent: oops\n"), 0o644))

	ch, err := LoadChartDir(dir)
	require.NoError(t, err)

	_, err = RenderRequirements(ch, RenderContext{Values: map[string]any{}})
	require.Error(t, err)
	require.Contains(t, err.Error(), requirementsName)
}

func TestLoadChartDirNoRequirementsIsNil(t *testing.T) {
	dir := t.TempDir()
	writeMinimalChart(t, dir)
	ch, err := LoadChartDir(dir)
	require.NoError(t, err)
	require.Nil(t, ch.Requirements)
}

func TestLoadChartDirRetainsValuesRaw(t *testing.T) {
	dir := t.TempDir()
	writeMinimalChart(t, dir)
	raw := "# a comment\nfoo: bar  # inline\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "values.yaml"), []byte(raw), 0o644))

	ch, err := LoadChartDir(dir)
	require.NoError(t, err)
	require.Equal(t, raw, string(ch.ValuesRaw)) // verbatim: comments + order preserved
	require.Equal(t, "bar", ch.Values["foo"])   // still parsed
}

func TestLoadChartDirNoValuesRawIsNil(t *testing.T) {
	dir := t.TempDir()
	writeMinimalChart(t, dir)
	ch, err := LoadChartDir(dir)
	require.NoError(t, err)
	require.Nil(t, ch.ValuesRaw)
}

func TestLoadChartArchiveRetainsValuesRaw(t *testing.T) {
	raw := "# a comment\nfoo: bar\n"
	tgz := packEntries(t, map[string]string{
		"demo/Chart.yaml":           "name: demo\nversion: 1.0.0\n",
		"demo/templates/stack.yaml": "services: {}\n",
		"demo/values.yaml":          raw,
	})
	ch, err := LoadChartArchive(strings.NewReader(tgz))
	require.NoError(t, err)
	require.Equal(t, raw, string(ch.ValuesRaw))
	require.Equal(t, "bar", ch.Values["foo"])
}

func TestLoadChartArchiveParsesRequirements(t *testing.T) {
	tgz := packEntries(t, map[string]string{
		"demo/Chart.yaml":           "name: demo\nversion: 1.0.0\n",
		"demo/templates/stack.yaml": "services: {}\n",
		"demo/requirements.yaml":    "networks:\n  - name: traefik-public\n",
	})
	ch, err := LoadChartArchive(strings.NewReader(tgz))
	require.NoError(t, err)
	require.NotNil(t, ch.Requirements)
	require.Equal(t, "traefik-public", ch.Requirements.Networks[0].Name)
}

func TestLoadChartArchiveInvalidRequirements(t *testing.T) {
	tgz := packEntries(t, map[string]string{
		"demo/Chart.yaml":           "name: demo\nversion: 1.0.0\n",
		"demo/templates/stack.yaml": "services: {}\n",
		"demo/requirements.yaml":    "networks:\n  - description: nameless\n",
	})
	_, err := LoadChartArchive(strings.NewReader(tgz))
	require.ErrorContains(t, err, "networks[0] has no name")
}
