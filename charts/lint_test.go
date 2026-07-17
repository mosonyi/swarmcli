// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package charts

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// lintChart loads testdata/demo and applies opt, so each test starts from a
// chart that is known to lint clean.
func lintChart(t *testing.T, opt func(*Chart)) *Chart {
	t.Helper()
	ch, err := LoadChartDir("testdata/demo")
	require.NoError(t, err)
	if opt != nil {
		opt(ch)
	}
	return ch
}

func messages(findings []LintFinding) string {
	var s string
	for _, f := range findings {
		s += f.Severity.String() + ": " + f.Message + "\n"
	}
	return s
}

func TestLintCleanChart(t *testing.T) {
	ch := lintChart(t, func(c *Chart) { c.Metadata.SwarmcliVersion = ">= 1.0.0" })
	findings := Lint(ch, "1.13.0")
	require.Empty(t, findings, "a satisfied chart must lint silently, got:\n%s", messages(findings))
	require.False(t, HasErrors(findings))
}

// The field is optional, so its absence is advice rather than a failure — but a
// chart that names no floor is the problem swarmcliVersion exists to fix.
func TestLintNoSwarmcliVersionWarnsButPasses(t *testing.T) {
	findings := Lint(lintChart(t, nil), "1.13.0")
	require.Len(t, findings, 1)
	require.Equal(t, LintWarning, findings[0].Severity)
	require.Contains(t, findings[0].Message, "declares no swarmcliVersion")
	require.False(t, HasErrors(findings), "a missing floor must not fail a lint")
}

func TestLintUnsatisfiedFloorIsAnError(t *testing.T) {
	ch := lintChart(t, func(c *Chart) { c.Metadata.SwarmcliVersion = ">= 1.13.0" })
	findings := Lint(ch, "1.12.0")
	require.True(t, HasErrors(findings))
	require.Contains(t, messages(findings), "requires swarmcli >= 1.13.0")
	require.Contains(t, messages(findings), "1.12.0 does not satisfy")
}

// Lint must never claim the version it was asked about is the one running: under
// --for-version it is not.
func TestLintDoesNotClaimTheAskedVersionIsThisBuild(t *testing.T) {
	withEngineVersion(t, "1.13.0") // this build
	ch := lintChart(t, func(c *Chart) { c.Metadata.SwarmcliVersion = ">= 2.0.0" })
	require.NotContains(t, messages(Lint(ch, "1.9.0")), "this build provides")
}

func TestLintUnparseableConstraintWarns(t *testing.T) {
	ch := lintChart(t, func(c *Chart) { c.Metadata.SwarmcliVersion = "newer than 1.13 please" })
	findings := Lint(ch, "1.13.0")
	require.False(t, HasErrors(findings), "an unusable constraint is a warning, matching CheckCompat")
	require.Contains(t, messages(findings), "not a valid SemVer constraint")
}

// Without --for-version an unstamped dev build has nothing to compare against.
// That must not manufacture an error.
func TestLintUnstampedEngineWarns(t *testing.T) {
	ch := lintChart(t, func(c *Chart) { c.Metadata.SwarmcliVersion = ">= 1.13.0" })
	findings := Lint(ch, "")
	require.False(t, HasErrors(findings))
	require.Contains(t, messages(findings), "no chart-engine version")
}

func TestLintBrokenTemplateIsAnError(t *testing.T) {
	ch := lintChart(t, func(c *Chart) {
		c.Metadata.SwarmcliVersion = ">= 1.0.0"
		c.Templates = map[string]string{"templates/stack.yaml": "{{ toYamlPretty .Values }}"}
	})
	findings := Lint(ch, "1.13.0")
	require.True(t, HasErrors(findings))
	require.Contains(t, messages(findings), "render with default values failed")
	require.Contains(t, messages(findings), "toYamlPretty")
}

func TestLintValuesFailingSchemaIsAnError(t *testing.T) {
	ch := lintChart(t, func(c *Chart) {
		c.Metadata.SwarmcliVersion = ">= 1.0.0"
		c.Schema = []byte(`{"type":"object","properties":{"replicas":{"type":"string"}}}`)
	})
	findings := Lint(ch, "1.13.0")
	require.True(t, HasErrors(findings))
	require.Contains(t, messages(findings), "values.schema.json")
}

// A chart author wants the list, not a game of whack-a-mole.
func TestLintReportsEveryProblemNotJustTheFirst(t *testing.T) {
	ch := lintChart(t, func(c *Chart) {
		c.Metadata.SwarmcliVersion = ">= 1.13.0"                                           // problem 1
		c.Schema = []byte(`{"type":"object","properties":{"replicas":{"type":"string"}}}`) // problem 2
	})
	findings := Lint(ch, "1.12.0")
	require.GreaterOrEqual(t, len(findings), 2, "got:\n%s", messages(findings))
}

func TestHasErrors(t *testing.T) {
	require.False(t, HasErrors(nil))
	require.False(t, HasErrors([]LintFinding{{Severity: LintWarning}}))
	require.True(t, HasErrors([]LintFinding{{Severity: LintWarning}, {Severity: LintError}}))
}
