// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package charts

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseRequirementsDefaults(t *testing.T) {
	// driver, attachable and autoCreate default when omitted.
	req, err := parseRequirements([]byte("networks:\n  - name: pub\n"))
	require.NoError(t, err)
	require.Len(t, req.Networks, 1)
	require.Equal(t, "overlay", req.Networks[0].Driver)
	require.NotNil(t, req.Networks[0].Attachable)
	require.True(t, *req.Networks[0].Attachable)
	require.NotNil(t, req.Networks[0].AutoCreate)
	require.True(t, *req.Networks[0].AutoCreate)
}

func TestParseRequirementsExplicitFalse(t *testing.T) {
	req, err := parseRequirements([]byte("networks:\n  - name: pub\n    autoCreate: false\n    attachable: false\n"))
	require.NoError(t, err)
	require.False(t, *req.Networks[0].AutoCreate)
	require.False(t, *req.Networks[0].Attachable)
}

func TestParseRequirementsRejectsNamelessEntry(t *testing.T) {
	_, err := parseRequirements([]byte("secrets:\n  - description: no name here\n"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "secrets[0] has no name")
}

func TestExternalResourceNames(t *testing.T) {
	cases := []struct {
		name     string
		manifest string
		key      string
		want     []string
	}{
		{
			name:     "external true uses the map key",
			manifest: "secrets:\n  db:\n    external: true\n",
			key:      "secrets",
			want:     []string{"db"},
		},
		{
			name:     "external long form uses the resolved name",
			manifest: "configs:\n  alias:\n    external:\n      name: real-config\n",
			key:      "configs",
			want:     []string{"real-config"},
		},
		{
			name:     "non-external and external:false are ignored",
			manifest: "secrets:\n  inline:\n    file: ./x\n  off:\n    external: false\n  on:\n    external: true\n",
			key:      "secrets",
			want:     []string{"on"},
		},
		{
			name:     "missing block yields nil",
			manifest: "services:\n  app:\n    image: x\n",
			key:      "secrets",
			want:     nil,
		},
		{
			name:     "sorted across multiple",
			manifest: "configs:\n  b:\n    external: true\n  a:\n    external: true\n",
			key:      "configs",
			want:     []string{"a", "b"},
		},
		{
			name:     "unrelated complex top-level keys do not interfere",
			manifest: "services:\n  app:\n    image: x\n    deploy:\n      replicas: 2\nsecrets:\n  s:\n    external: true\n",
			key:      "secrets",
			want:     []string{"s"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, externalResourceNames(tc.manifest, tc.key))
		})
	}
}

func TestEnsureExternalSecretsConfigs(t *testing.T) {
	ctx := context.Background()

	t.Run("no external secrets or configs is a no-op", func(t *testing.T) {
		e := testEngine(newFakeBackend())
		require.NoError(t, e.ensureExternalSecretsConfigs(ctx, "services:\n  app:\n    image: x\n", nil))
	})

	t.Run("present secret and config pass", func(t *testing.T) {
		fb := newFakeBackend()
		fb.secrets["db"] = struct{}{}
		fb.configs["cfg"] = fakeConfig{}
		e := testEngine(fb)
		m := "secrets:\n  db:\n    external: true\nconfigs:\n  cfg:\n    external: true\n"
		require.NoError(t, e.ensureExternalSecretsConfigs(ctx, m, nil))
	})

	t.Run("missing secret errors with remediation", func(t *testing.T) {
		e := testEngine(newFakeBackend())
		err := e.ensureExternalSecretsConfigs(ctx, "secrets:\n  db:\n    external: true\n", nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), `secret "db" does not exist`)
		require.Contains(t, err.Error(), "docker secret create db <file>")
	})

	t.Run("missing config errors with remediation", func(t *testing.T) {
		e := testEngine(newFakeBackend())
		err := e.ensureExternalSecretsConfigs(ctx, "configs:\n  nginx:\n    external: true\n", nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), `config "nginx" does not exist`)
		require.Contains(t, err.Error(), "docker config create nginx <file>")
	})

	t.Run("long-form external name is resolved", func(t *testing.T) {
		fb := newFakeBackend()
		fb.secrets["real-secret"] = struct{}{}
		e := testEngine(fb)
		// alias maps to real-secret, which exists -> passes
		require.NoError(t, e.ensureExternalSecretsConfigs(ctx,
			"secrets:\n  alias:\n    external:\n      name: real-secret\n", nil))
	})

	t.Run("backend error is surfaced", func(t *testing.T) {
		fb := newFakeBackend()
		fb.secretsErr = errors.New("daemon down")
		e := testEngine(fb)
		err := e.ensureExternalSecretsConfigs(ctx, "secrets:\n  db:\n    external: true\n", nil)
		require.ErrorContains(t, err, "checking external secrets")
	})

	t.Run("undeclared external secret is a contract error when requirements present", func(t *testing.T) {
		fb := newFakeBackend()
		fb.secrets["db"] = struct{}{} // it exists, but is not declared
		e := testEngine(fb)
		req := &Requirements{} // present but declares nothing
		err := e.ensureExternalSecretsConfigs(ctx, "secrets:\n  db:\n    external: true\n", req)
		require.Error(t, err)
		require.Contains(t, err.Error(), "secret(s) the manifest declares external are not declared in requirements.yaml")
		require.Contains(t, err.Error(), "db")
	})

	t.Run("declared description enriches missing-secret remediation", func(t *testing.T) {
		e := testEngine(newFakeBackend())
		req := &Requirements{Secrets: []ResourceRequirement{{Name: "db", Description: "Postgres password"}}}
		err := e.ensureExternalSecretsConfigs(ctx, "secrets:\n  db:\n    external: true\n", req)
		require.Error(t, err)
		require.Contains(t, err.Error(), `secret "db" does not exist (Postgres password)`)
	})
}

// A missing external secret must abort the install before any network is
// auto-created, leaving no swarm state behind.
func TestInstallMissingSecretAbortsBeforeNetworkCreate(t *testing.T) {
	fb := newFakeBackend()
	e := testEngine(fb)
	manifest := "services:\n  app:\n    image: x\n" +
		"networks:\n  pub:\n    external: true\n" +
		"secrets:\n  db:\n    external: true\n"

	rel, err := e.Install(context.Background(), "demo",
		ReleaseChart{Name: "demo", Version: "0.1.0"}, nil, manifest, InstallOptions{})

	require.Error(t, err)
	require.Contains(t, err.Error(), `secret "db" does not exist`)
	require.Equal(t, StatusFailed, rel.Status)
	require.NotContains(t, fb.networkScopes, "pub", "no network should be created when a prerequisite secret is missing")
	require.Empty(t, fb.deployed, "nothing should be deployed")
}

// With the prerequisite secret present, the install proceeds: the external
// network is created and the stack deploys.
func TestInstallProceedsWhenSecretPresent(t *testing.T) {
	fb := newFakeBackend()
	fb.secrets["db"] = struct{}{}
	e := testEngine(fb)
	manifest := "services:\n  app:\n    image: x\n" +
		"networks:\n  pub:\n    external: true\n" +
		"secrets:\n  db:\n    external: true\n"

	_, err := e.Install(context.Background(), "demo",
		ReleaseChart{Name: "demo", Version: "0.1.0"}, nil, manifest, InstallOptions{})

	require.NoError(t, err)
	require.Contains(t, fb.networkScopes, "pub")
	require.Contains(t, fb.deployed, "demo")
}
