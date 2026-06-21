// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// capture redirects stdout/stderr to buffers for the duration of fn.
func capture(t *testing.T, fn func()) (string, string) {
	t.Helper()
	var outBuf, errBuf bytes.Buffer
	origOut, origErr := stdout, stderr
	stdout, stderr = &outBuf, &errBuf
	defer func() { stdout, stderr = origOut, origErr }()
	fn()
	return outBuf.String(), outBuf.String() + errBuf.String() // second = combined for convenience
}

func TestDispatchVersion(t *testing.T) {
	var code int
	o, _ := capture(t, func() { code = Dispatch([]string{"version"}, "1.2.3") })
	require.Equal(t, 0, code)
	require.Equal(t, "1.2.3", strings.TrimSpace(o))
}

func TestDispatchUnknownCommand(t *testing.T) {
	var code int
	capture(t, func() { code = Dispatch([]string{"frobnicate"}, "dev") })
	require.Equal(t, 2, code)
}

func TestChartsTemplateLocalDir(t *testing.T) {
	var code int
	o, _ := capture(t, func() {
		code = Dispatch([]string{"charts", "template", "my-demo", "../charts/testdata/demo", "--set", "replicas=3", "--set", "image.tag=v9"}, "dev")
	})
	require.Equal(t, 0, code)
	require.Contains(t, o, "traefik:v9")
	require.Contains(t, o, "replicas: 3")
	require.Contains(t, o, "com.swarmcli.release: my-demo")
}

func TestChartsTemplateSchemaRejection(t *testing.T) {
	var code int
	_, combined := capture(t, func() {
		code = Dispatch([]string{"charts", "template", "x", "../charts/testdata/demo", "--set", "replicas=0"}, "dev")
	})
	require.Equal(t, 1, code)
	require.Contains(t, combined, "schema validation")
}

func TestChartsTemplateUnknownFlag(t *testing.T) {
	var code int
	capture(t, func() {
		code = Dispatch([]string{"charts", "template", "x", "../charts/testdata/demo", "--bogus"}, "dev")
	})
	require.Equal(t, 2, code)
}

func TestChartsShowValues(t *testing.T) {
	var code int
	o, _ := capture(t, func() {
		code = Dispatch([]string{"charts", "show", "values", "../charts/testdata/demo"}, "dev")
	})
	require.Equal(t, 0, code)
	require.Contains(t, o, "replicas: 1")
}

func TestParseArgs(t *testing.T) {
	pos, f, err := parseArgs([]string{"rel", "repo/chart", "-f", "a.yaml", "--values", "b.yaml", "--set", "x=1", "--dry-run", "--timeout", "10m"})
	require.NoError(t, err)
	require.Equal(t, []string{"rel", "repo/chart"}, pos)
	require.Equal(t, []string{"a.yaml", "b.yaml"}, f.values)
	require.Equal(t, []string{"x=1"}, f.sets)
	require.True(t, f.dryRun)
	require.Equal(t, "10m0s", f.timeout.String())
}

func TestParseArgsInlineValue(t *testing.T) {
	_, f, err := parseArgs([]string{"--set=a=1", "--version=2.0.0"})
	require.NoError(t, err)
	require.Equal(t, []string{"a=1"}, f.sets)
	require.Equal(t, "2.0.0", f.version)
}
