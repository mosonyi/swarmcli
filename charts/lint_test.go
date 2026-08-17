// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package charts

import (
	"fmt"
	"strings"
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
	findings := Lint(ch, "1.13.0", nil, nil)
	require.Empty(t, findings, "a satisfied chart must lint silently, got:\n%s", messages(findings))
	require.False(t, HasErrors(findings))
}

// The field is optional, so its absence is advice rather than a failure — but a
// chart that names no floor is the problem swarmcliVersion exists to fix.
func TestLintNoSwarmcliVersionWarnsButPasses(t *testing.T) {
	findings := Lint(lintChart(t, nil), "1.13.0", nil, nil)
	require.Len(t, findings, 1)
	require.Equal(t, LintWarning, findings[0].Severity)
	require.Contains(t, findings[0].Message, "declares no swarmcliVersion")
	require.False(t, HasErrors(findings), "a missing floor must not fail a lint")
}

func TestLintUnsatisfiedFloorIsAnError(t *testing.T) {
	ch := lintChart(t, func(c *Chart) { c.Metadata.SwarmcliVersion = ">= 1.13.0" })
	findings := Lint(ch, "1.12.0", nil, nil)
	require.True(t, HasErrors(findings))
	require.Contains(t, messages(findings), "requires swarmcli >= 1.13.0")
	require.Contains(t, messages(findings), "1.12.0 does not satisfy")
}

// Lint must never claim the version it was asked about is the one running: under
// --for-version it is not.
func TestLintDoesNotClaimTheAskedVersionIsThisBuild(t *testing.T) {
	withEngineVersion(t, "1.13.0") // this build
	ch := lintChart(t, func(c *Chart) { c.Metadata.SwarmcliVersion = ">= 2.0.0" })
	require.NotContains(t, messages(Lint(ch, "1.9.0", nil, nil)), "this build provides")
}

func TestLintUnparseableConstraintWarns(t *testing.T) {
	ch := lintChart(t, func(c *Chart) { c.Metadata.SwarmcliVersion = "newer than 1.13 please" })
	findings := Lint(ch, "1.13.0", nil, nil)
	require.False(t, HasErrors(findings), "an unusable constraint is a warning, matching CheckCompat")
	require.Contains(t, messages(findings), "not a valid SemVer constraint")
}

// Without --for-version an unstamped dev build has nothing to compare against.
// That must not manufacture an error.
func TestLintUnstampedEngineWarns(t *testing.T) {
	ch := lintChart(t, func(c *Chart) { c.Metadata.SwarmcliVersion = ">= 1.13.0" })
	findings := Lint(ch, "", nil, nil)
	require.False(t, HasErrors(findings))
	require.Contains(t, messages(findings), "no chart-engine version")
}

func TestLintBrokenTemplateIsAnError(t *testing.T) {
	ch := lintChart(t, func(c *Chart) {
		c.Metadata.SwarmcliVersion = ">= 1.0.0"
		c.Templates = map[string]string{"templates/stack.yaml": "{{ toYamlPretty .Values }}"}
	})
	findings := Lint(ch, "1.13.0", nil, nil)
	require.True(t, HasErrors(findings))
	require.Contains(t, messages(findings), "render with default values failed")
	require.Contains(t, messages(findings), "toYamlPretty")
}

// A chart with a required, undefaulted input (a {{ fail }} / {{ required }}
// guard) cannot render from bare defaults — so lint must accept the same values
// an install would supply, or it flags a working chart as broken. This is the
// renovate case found by linting the real charts.
func TestLintRequiredInputNeedsValues(t *testing.T) {
	newChart := func() *Chart {
		return lintChart(t, func(c *Chart) {
			c.Metadata.SwarmcliVersion = ">= 1.0.0"
			c.Values = map[string]any{}
			c.Schema = nil
			c.Templates = map[string]string{
				"templates/stack.yaml": `{{ if not .Values.repositories }}{{ fail "set repositories" }}{{ end }}` +
					"\nservices:\n  app:\n    image: busybox\n",
			}
		})
	}

	bare := Lint(newChart(), "1.13.0", nil, nil)
	require.True(t, HasErrors(bare), "bare defaults must surface the unmet requirement")
	require.Contains(t, messages(bare), "set repositories")

	withValues := Lint(newChart(), "1.13.0", [][]byte{[]byte("repositories: [owner/repo]\n")}, nil)
	require.False(t, HasErrors(withValues), "supplying the value must clear it, got:\n%s", messages(withValues))
}

// --set reaches the render too.
func TestLintSetOverride(t *testing.T) {
	ch := lintChart(t, func(c *Chart) {
		c.Metadata.SwarmcliVersion = ">= 1.0.0"
		c.Values = map[string]any{}
		c.Schema = nil
		c.Templates = map[string]string{
			"templates/stack.yaml": `{{ if not .Values.enabled }}{{ fail "set enabled=true" }}{{ end }}` +
				"\nservices:\n  app:\n    image: busybox\n",
		}
	})
	require.False(t, HasErrors(Lint(ch, "1.13.0", nil, []string{"enabled=true"})))
}

func TestLintValuesFailingSchemaIsAnError(t *testing.T) {
	ch := lintChart(t, func(c *Chart) {
		c.Metadata.SwarmcliVersion = ">= 1.0.0"
		c.Schema = []byte(`{"type":"object","properties":{"replicas":{"type":"string"}}}`)
	})
	findings := Lint(ch, "1.13.0", nil, nil)
	require.True(t, HasErrors(findings))
	require.Contains(t, messages(findings), "values.schema.json")
}

// A chart author wants the list, not a game of whack-a-mole.
func TestLintReportsEveryProblemNotJustTheFirst(t *testing.T) {
	ch := lintChart(t, func(c *Chart) {
		c.Metadata.SwarmcliVersion = ">= 1.13.0"                                           // problem 1
		c.Schema = []byte(`{"type":"object","properties":{"replicas":{"type":"string"}}}`) // problem 2
	})
	findings := Lint(ch, "1.12.0", nil, nil)
	require.GreaterOrEqual(t, len(findings), 2, "got:\n%s", messages(findings))
}

func TestHasErrors(t *testing.T) {
	require.False(t, HasErrors(nil))
	require.False(t, HasErrors([]LintFinding{{Severity: LintWarning}}))
	require.True(t, HasErrors([]LintFinding{{Severity: LintWarning}, {Severity: LintError}}))
}

func lintMonitorFindings(t *testing.T, manifest string) []LintFinding {
	t.Helper()
	var out []LintFinding
	lintHealthcheckMonitor(manifest, func(s LintSeverity, format string, a ...any) {
		out = append(out, LintFinding{Severity: s, Message: fmt.Sprintf(format, a...)})
	})
	return out
}

// The footgun: monitor elapses before the healthcheck can fail even once, so a
// container that goes unhealthy afterwards leaves the rollout reported as
// completed.
func TestLintHealthcheckMonitorTooShort(t *testing.T) {
	got := lintMonitorFindings(t, `
services:
  api:
    image: nginx
    healthcheck:
      test: ["CMD", "true"]
      interval: 10s
      retries: 3
      start_period: 30s
    deploy:
      update_config:
        monitor: 15s
`)
	require.Len(t, got, 1)
	require.Equal(t, LintWarning, got[0].Severity)
	require.Contains(t, got[0].Message, `service "api"`)
	require.Contains(t, got[0].Message, "1m0s") // 30s + 3x10s
}

func TestLintHealthcheckMonitorSufficient(t *testing.T) {
	require.Empty(t, lintMonitorFindings(t, `
services:
  api:
    healthcheck:
      test: ["CMD", "true"]
      interval: 10s
      retries: 3
      start_period: 30s
    deploy:
      update_config:
        monitor: 90s
`))
}

// Docker's defaults (interval 30s, retries 3) apply when the chart omits them,
// so the implied window is 90s and a 30s monitor is still too short.
func TestLintHealthcheckMonitorUsesDockerDefaults(t *testing.T) {
	got := lintMonitorFindings(t, `
services:
  api:
    healthcheck:
      test: ["CMD", "true"]
    deploy:
      update_config:
        monitor: 30s
`)
	require.Len(t, got, 1)
	require.Contains(t, got[0].Message, "1m30s") // 3 x 30s
}

// An unset monitor is the common way to hit this, not an exemption from it:
// swarm silently applies a 5s default, and a healthcheck needing 30s to fail
// therefore cannot fail the rollout. The rule originally skipped this case,
// which is why it fired on nothing at all.
func TestLintHealthcheckMonitorFlagsTheInheritedDefault(t *testing.T) {
	got := lintMonitorFindings(t, `
services:
  api:
    healthcheck:
      test: ["CMD", "true"]
      interval: 10s
`)
	require.Len(t, got, 1)
	require.Contains(t, got[0].Message, "no deploy.update_config.monitor")
	require.Contains(t, got[0].Message, "5s default")
	require.Contains(t, got[0].Message, "at least 30s", "the finding names the value to set")
}

// The recommended value is a floor, not a budget: swarm starts the monitor when
// it CREATES the task, so the image pull happens inside the window too. That is
// unknowable from the manifest, so the finding says so rather than inventing an
// allowance — a constant would be wrong for every chart differently, and would
// re-fire the rule on charts whose monitors came from this formula.
func TestLintHealthcheckMonitorNamesPullTimeAsUncounted(t *testing.T) {
	got := lintMonitorFindings(t, `
services:
  api:
    healthcheck:
      test: ["CMD", "true"]
      interval: 10s
`)
	require.Len(t, got, 1)
	require.Contains(t, got[0].Message, "pull runs inside it")
}

// A monitor that already covers the healthcheck is fine whether it was declared
// or inherited: a fast healthcheck under swarm's 5s default is not a finding.
func TestLintHealthcheckMonitorAcceptsAnAdequateInheritedDefault(t *testing.T) {
	require.Empty(t, lintMonitorFindings(t, `
services:
  api:
    healthcheck:
      test: ["CMD", "true"]
      interval: 1s
      retries: 3
      start_period: 1s
`))
}

func TestLintHealthcheckMonitorSkipsDisabled(t *testing.T) {
	require.Empty(t, lintMonitorFindings(t, `
services:
  a:
    healthcheck:
      test: ["NONE"]
    deploy:
      update_config:
        monitor: 1s
  b:
    healthcheck:
      test: ["CMD", "true"]
      disable: true
    deploy:
      update_config:
        monitor: 1s
  c:
    image: nginx
    deploy:
      update_config:
        monitor: 1s
`))
}

// Findings must not shuffle between runs — service names come out of a map.
func TestLintHealthcheckMonitorIsOrdered(t *testing.T) {
	manifest := `
services:
  zebra:
    healthcheck: {test: ["CMD", "true"], interval: 10s, retries: 3}
    deploy: {update_config: {monitor: 1s}}
  alpha:
    healthcheck: {test: ["CMD", "true"], interval: 10s, retries: 3}
    deploy: {update_config: {monitor: 1s}}
`
	for i := 0; i < 20; i++ {
		got := lintMonitorFindings(t, manifest)
		require.Len(t, got, 2)
		require.Contains(t, got[0].Message, `"alpha"`)
		require.Contains(t, got[1].Message, `"zebra"`)
	}
}

// envChart returns the demo chart with one service whose environment: block is
// env, so a finding can only have come from the entry under test.
func envChart(t *testing.T, env string) *Chart {
	t.Helper()
	return lintChart(t, func(c *Chart) {
		c.Metadata.SwarmcliVersion = ">= 1.0.0"
		c.Templates = map[string]string{
			"templates/stack.yaml": "services:\n  web:\n    image: nginx\n    environment:\n" + env,
		}
	})
}

// inlineFindings is every finding the inline-secret rule produced, which is
// every finding naming an environment key.
func inlineFindings(findings []LintFinding) []string {
	var out []string
	for _, f := range findings {
		if strings.Contains(f.Message, ".environment.") {
			out = append(out, f.Message)
		}
	}
	return out
}

// The rule fires on the key, so it must fire on every spelling of a key that
// names a credential, in both shapes compose accepts for the block. Covering
// the mapping shape proves nothing about the sequence shape: they are read by
// different branches of envPairs.
func TestLintWarnsOnInlineSecrets(t *testing.T) {
	for _, tc := range []struct {
		name string
		env  string
		key  string
	}{
		{"password", "      DB_PASSWORD: hunter2\n", "DB_PASSWORD"},
		{"passwd", "      MYSQL_PASSWD: hunter2\n", "MYSQL_PASSWD"},
		{"secret", "      APP_SECRET: s3kr3t\n", "APP_SECRET"},
		{"token", "      GITHUB_TOKEN: ghp_aaaa\n", "GITHUB_TOKEN"},
		{"api key, underscored", "      API_KEY: abc\n", "API_KEY"},
		{"api key, run together", "      APIKEY: abc\n", "APIKEY"},
		{"credential", "      AWS_CREDENTIALS: abc\n", "AWS_CREDENTIALS"},
		{"lowercase key", "      db_password: hunter2\n", "db_password"},
		{"sequence shape", "      - DB_PASSWORD=hunter2\n", "DB_PASSWORD"},
		// An undefaulted value is the shape worth warning about, not an
		// exemption from it: the key is what carries the credential once an
		// operator supplies one, which is the chart its author should hear about.
		{"empty value", "      DB_PASSWORD: \"\"\n", "DB_PASSWORD"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			findings := inlineFindings(Lint(envChart(t, tc.env), "1.13.0", nil, nil))
			require.Len(t, findings, 1)
			require.Contains(t, findings[0], "services.web.environment."+tc.key)
			require.Contains(t, findings[0], "names a credential")
			// The finding must never quote the value it matched on: a warning is
			// printed to a terminal and scraped into CI logs.
			require.NotContains(t, findings[0], "hunter2")
		})
	}
}

// The sanctioned patterns must lint silently, or the rule punishes exactly the
// charts that got it right — all fifteen published charts rely on these.
func TestLintAcceptsTheSecretConventions(t *testing.T) {
	for _, tc := range []struct {
		name string
		env  string
	}{
		{"file variable naming a mounted secret", "      DB_PASSWORD_FILE: /run/secrets/db-password\n"},
		{"file variable naming any path", "      DB_PASSWORD_FILE: /etc/creds/db\n"},
		{"secret path under an ordinary key", "      DB_PASSWORD: /run/secrets/db-password\n"},
		{"braced interpolation", "      DB_PASSWORD: ${DB_PASSWORD}\n"},
		{"bare interpolation", "      DB_PASSWORD: $DB_PASSWORD\n"},
		{"compose-escaped interpolation", "      DB_PASSWORD: $$DB_PASSWORD\n"},
		{"innocuous key and value", "      LOG_LEVEL: debug\n"},
		{"integer value", "      PORT: 8080\n"},
		{"public certificate", "      TLS_CERT: \"-----BEGIN CERTIFICATE-----\\nMIIB\\n\"\n"},
		{"sequence passthrough without a value", "      - DB_PASSWORD\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			findings := Lint(envChart(t, tc.env), "1.13.0", nil, nil)
			// Before the silence assertion: a chart that failed to render
			// produces no environment findings either, and would pass this test
			// without the rule having read a single key.
			require.False(t, HasErrors(findings),
				"the chart must render for the silence below to mean anything, got:\n%s", messages(findings))
			require.Empty(t, inlineFindings(findings),
				"a chart following the convention must lint silently, got:\n%s", messages(findings))
		})
	}
}

// A PEM private key is material wherever it appears, so it is caught on a key
// name the rule would otherwise pass — including a *_FILE name, which claims to
// be a path and is not. That case must not then advise a DB_KEY_FILE_FILE.
func TestLintWarnsOnAnInlinedPrivateKey(t *testing.T) {
	const pem = "-----BEGIN RSA PRIVATE KEY-----\\nMIIEow\\n-----END RSA PRIVATE KEY-----"
	for _, tc := range []struct{ name, env, key string }{
		{"innocuous key", "      SIGNING_MATERIAL: \"" + pem + "\"\n", "SIGNING_MATERIAL"},
		{"file key holding the key itself", "      TLS_KEY_FILE: \"" + pem + "\"\n", "TLS_KEY_FILE"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			findings := inlineFindings(Lint(envChart(t, tc.env), "1.13.0", nil, nil))
			require.Len(t, findings, 1)
			require.Contains(t, findings[0], "services.web.environment."+tc.key)
			require.Contains(t, findings[0], "inlines a PEM private key")
			require.NotContains(t, findings[0], "MIIEow")
			require.NotContains(t, findings[0], "_FILE_FILE")
		})
	}
}

// An integer sibling must not hide a credential: decoding the block into
// map[string]string fails on PORT and would take DB_PASSWORD with it.
func TestLintReadsAnEnvBlockOfMixedTypes(t *testing.T) {
	findings := inlineFindings(Lint(envChart(t,
		"      PORT: 8080\n      DB_PASSWORD: hunter2\n      REPLICAS: 3\n"), "1.13.0", nil, nil))
	require.Len(t, findings, 1)
	require.Contains(t, findings[0], "services.web.environment.DB_PASSWORD")
}

// The rule is a warning, never an error: the engine cannot tell a credential
// from a value that reads like one, so it must not fail a chart over a guess.
func TestLintInlineSecretIsAWarning(t *testing.T) {
	findings := Lint(envChart(t, "      DB_PASSWORD: hunter2\n"), "1.13.0", nil, nil)
	require.NotEmpty(t, inlineFindings(findings))
	require.False(t, HasErrors(findings))
}
