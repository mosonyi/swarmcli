// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package charts

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func loadDemo(t *testing.T) *Chart {
	t.Helper()
	ch, err := LoadChartDir("testdata/demo")
	require.NoError(t, err)
	return ch
}

func TestLoadChartDir(t *testing.T) {
	ch := loadDemo(t)
	require.Equal(t, "demo", ch.Metadata.Name)
	require.Equal(t, "0.1.0", ch.Metadata.Version)
	require.NotEmpty(t, ch.Schema)
	require.Contains(t, ch.Templates, "templates/stack.yaml")
	require.Contains(t, ch.Templates, "templates/extras.yaml")
}

func TestMergeValuesPrecedence(t *testing.T) {
	ch := loadDemo(t)
	file := []byte("replicas: 2\nimage:\n  tag: v3.1.0\n")
	got, err := MergeValues(ch.Values, [][]byte{file}, []string{"replicas=5"})
	require.NoError(t, err)

	require.Equal(t, 5, got["replicas"]) // --set wins over file and default
	img := got["image"].(map[string]any)
	require.Equal(t, "v3.1.0", img["tag"])   // file overrides default
	require.Equal(t, "traefik", img["repo"]) // default preserved by deep-merge

	// defaults must not be mutated
	require.Equal(t, 1, ch.Values["replicas"])
}

func TestSetTypeInferenceAndNesting(t *testing.T) {
	out := map[string]any{}
	require.NoError(t, applySet(out, "a.b=3,a.c=true,d=hello"))
	a := out["a"].(map[string]any)
	require.Equal(t, 3, a["b"])
	require.Equal(t, true, a["c"])
	require.Equal(t, "hello", out["d"])
}

func TestRenderInterpolatesAndGates(t *testing.T) {
	ch := loadDemo(t)
	vals, err := MergeValues(ch.Values, nil, []string{"image.tag=v9", "replicas=3"})
	require.NoError(t, err)

	out, err := Render(ch, RenderContext{
		Values:  vals,
		Release: ReleaseMeta{Name: "my-demo", Namespace: "my-demo", Revision: 1},
		Chart:   ChartMeta{Name: ch.Metadata.Name, Version: ch.Metadata.Version},
	})
	require.NoError(t, err)

	var doc map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(out), &doc))
	services := doc["services"].(map[string]any)
	app := services["app"].(map[string]any)
	require.Equal(t, "traefik:v9", app["image"])
	// extras.enabled is false by default -> sidecar gated out
	require.NotContains(t, services, "sidecar")
	require.Contains(t, out, "my-demo")
}

func TestRenderConditionalServiceEnabled(t *testing.T) {
	ch := loadDemo(t)
	vals, err := MergeValues(ch.Values, nil, []string{"extras.enabled=true"})
	require.NoError(t, err)
	out, err := Render(ch, RenderContext{Values: vals, Release: ReleaseMeta{Name: "x", Namespace: "x", Revision: 1}})
	require.NoError(t, err)
	require.Contains(t, out, "sidecar")
}

func TestValidateValuesSchema(t *testing.T) {
	ch := loadDemo(t)
	require.NoError(t, ValidateValues(ch.Schema, ch.Values))

	bad, err := MergeValues(ch.Values, nil, []string{"replicas=0"})
	require.NoError(t, err)
	require.Error(t, ValidateValues(ch.Schema, bad)) // violates minimum:1
}

func TestLoadChartArchive(t *testing.T) {
	tgz := packDirToTgz(t, "testdata/demo", "demo")
	ch, err := LoadChartArchive(strings.NewReader(tgz))
	require.NoError(t, err)
	require.Equal(t, "demo", ch.Metadata.Name)
	require.Contains(t, ch.Templates, "templates/stack.yaml")
	require.NotEmpty(t, ch.Schema)
}
