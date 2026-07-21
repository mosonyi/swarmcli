// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package charts

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOwnerRefRoundTrip(t *testing.T) {
	for _, tc := range []struct {
		name string
		ref  OwnerRef
		want string
	}{
		{"apply", OwnerRef{ID: "apply/prod-swarm", Kind: OwnerKindRelease, Name: "whoami"}, "apply/prod-swarm:release/whoami"},
		{"controller", OwnerRef{ID: "cd/edge", Kind: OwnerKindRelease, Name: "traefik"}, "cd/edge:release/traefik"},
		// Release names admit '.', '-' and '_', none of which are separators.
		{"punctuated name", OwnerRef{ID: "apply/a", Kind: OwnerKindRelease, Name: "a.b-c_d"}, "apply/a:release/a.b-c_d"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, tc.ref.String())
			got, err := ParseOwner(tc.want)
			require.NoError(t, err)
			require.Equal(t, tc.ref, got)
		})
	}
}

// The id carries a '/' by convention ("apply/<name>"), so the split has to cut
// the id at the first ':' rather than the first '/'.
func TestParseOwnerKeepsSlashesInTheID(t *testing.T) {
	got, err := ParseOwner("apply/deeply/nested:release/hello")
	require.NoError(t, err)
	require.Equal(t, OwnerRef{ID: "apply/deeply/nested", Kind: OwnerKindRelease, Name: "hello"}, got)
}

func TestParseOwnerRejectsMalformedStamps(t *testing.T) {
	for _, s := range []string{
		"",
		"apply/prod",            // no resource half
		"apply/prod:release",    // no name
		":release/hello",        // no id
		"apply/prod:/hello",     // no kind
		"apply/prod:release/",   // empty name
		"apply/prod release/hi", // wrong separator
	} {
		_, err := ParseOwner(s)
		require.Error(t, err, "stamp %q must not parse", s)
	}
}

func TestValidateOwnerID(t *testing.T) {
	require.NoError(t, validateOwnerID("apply/prod-swarm"))
	require.Error(t, validateOwnerID(""))
	// ':' separates the id from the resource half; an id carrying one would
	// decode as a different owner than it was written as.
	require.Error(t, validateOwnerID("apply:prod"))
}

// --- stamping ---

// A stamped install records the owner in both places it has to be: the Config
// label, so a prune can find candidates without decoding every payload, and the
// payload itself, so the record stays self-describing.
func TestInstallStampsOwnerOnLabelAndPayload(t *testing.T) {
	fb := newFakeBackend()
	e := NewEngineWith(fb)
	ctx := context.Background()

	rel, err := e.Install(ctx, "hello", ReleaseChart{Name: "demo", Version: "0.1.0"}, nil,
		"services: {}\n", InstallOptions{Owner: "apply/prod-swarm"})
	require.NoError(t, err)
	require.Equal(t, "apply/prod-swarm:release/hello", rel.Owner)

	cfg := fb.configs["swarmcli.release.hello.v1"]
	require.Equal(t, "apply/prod-swarm:release/hello", cfg.labels[LabelOwner])

	stored, err := e.GetRevision(ctx, "hello", 1)
	require.NoError(t, err)
	require.Equal(t, "apply/prod-swarm:release/hello", stored.Owner)
}

// An unowned install is the default, and it must leave the label off entirely
// rather than blank: a prune filtering on the label's presence would otherwise
// match every release ever installed by hand.
func TestInstallWithoutOwnerLeavesNoLabel(t *testing.T) {
	fb := newFakeBackend()
	e := NewEngineWith(fb)

	rel, err := e.Install(context.Background(), "hello", ReleaseChart{Name: "demo", Version: "0.1.0"}, nil,
		"services: {}\n", InstallOptions{})
	require.NoError(t, err)
	require.Empty(t, rel.Owner)
	require.NotContains(t, fb.configs["swarmcli.release.hello.v1"].labels, LabelOwner)
}

// A malformed owner has to fail before anything is deployed: recording it would
// produce a stamp that decodes as a different owner than the caller passed.
func TestInstallRejectsMalformedOwnerBeforeDeploying(t *testing.T) {
	fb := newFakeBackend()
	e := NewEngineWith(fb)

	_, err := e.Install(context.Background(), "hello", ReleaseChart{Name: "demo", Version: "0.1.0"}, nil,
		"services: {}\n", InstallOptions{Owner: "apply:prod"})
	require.ErrorContains(t, err, "must not contain ':'")
	require.Empty(t, fb.deployed)
	require.Empty(t, fb.configs)
}

// The stamp is computed before the dry-run return, so a plan shows the ownership
// it would record without touching the swarm.
func TestDryRunReportsTheOwnerItWouldRecord(t *testing.T) {
	fb := newFakeBackend()
	e := NewEngineWith(fb)

	rel, err := e.Install(context.Background(), "hello", ReleaseChart{Name: "demo", Version: "0.1.0"}, nil,
		"services: {}\n", InstallOptions{Owner: "cd/edge", DryRun: true})
	require.NoError(t, err)
	require.Equal(t, "cd/edge:release/hello", rel.Owner)
	require.Empty(t, fb.configs)
}

// Every revision is stamped with whoever wrote it. A second manifest upgrading a
// release takes it over, and the history says exactly where the handover
// happened rather than silently keeping the first owner's claim alive.
func TestUpgradeByAnotherOwnerTakesTheReleaseOver(t *testing.T) {
	e := NewEngineWith(newFakeBackend())
	ctx := context.Background()
	chart := ReleaseChart{Name: "demo", Version: "0.1.0"}

	_, err := e.Install(ctx, "hello", chart, nil, "services: {}\n", InstallOptions{Owner: "apply/a"})
	require.NoError(t, err)
	_, err = e.Upgrade(ctx, "hello", chart, nil, "services: {b: {}}\n", InstallOptions{Owner: "apply/b"})
	require.NoError(t, err)

	hist, err := e.History(ctx, "hello")
	require.NoError(t, err)
	require.Len(t, hist, 2)
	require.Equal(t, "apply/a:release/hello", hist[0].Owner)
	require.Equal(t, "apply/b:release/hello", hist[1].Owner)
}

// A rollback is a new revision written by whoever ran it, so it carries the
// current owner rather than resurrecting the one the target revision recorded.
func TestRollbackStampsTheCurrentOwnerNotTheTargets(t *testing.T) {
	e := NewEngineWith(newFakeBackend())
	ctx := context.Background()
	chart := ReleaseChart{Name: "demo", Version: "0.1.0"}

	_, err := e.Install(ctx, "hello", chart, nil, "services: {}\n", InstallOptions{Owner: "apply/a"})
	require.NoError(t, err)
	_, err = e.Upgrade(ctx, "hello", chart, nil, "services: {b: {}}\n", InstallOptions{Owner: "apply/b"})
	require.NoError(t, err)

	rel, err := e.Rollback(ctx, "hello", 1, InstallOptions{Owner: "apply/b"})
	require.NoError(t, err)
	require.Equal(t, "apply/b:release/hello", rel.Owner)
	require.Equal(t, "services: {}\n", rel.Manifest, "rollback still restores the target's content")
}
